package iccarus

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf16"
)

type DescTag struct {
	ASCII   string
	Unicode string
	Script  string
}

func descDecoder(raw []byte, hdrs []TagHeader) (any, error) {
	if len(raw) < 12 {
		return nil, fmt.Errorf("desc tag too short")
	}

	asciiLen := int(binary.BigEndian.Uint32(raw[8:12]))
	if asciiLen < 1 || 12+asciiLen > len(raw) {
		return nil, fmt.Errorf("invalid ASCII length in desc tag")
	}
	ascii := raw[12 : 12+asciiLen]
	if i := bytes.IndexByte(ascii, 0); i >= 0 {
		ascii = ascii[:i]
	}

	offset := 12 + asciiLen
	if len(raw) < offset+4 {
		return DescTag{ASCII: string(ascii)}, nil // ASCII-only, no Unicode
	}

	unicodeCount := int(binary.BigEndian.Uint32(raw[offset : offset+4]))
	offset += 4
	if len(raw) < offset+(unicodeCount*2) {
		return nil, fmt.Errorf("desc tag truncated: missing UTF-16 data")
	}
	unicodeData := raw[offset : offset+(unicodeCount*2)]
	offset += unicodeCount * 2

	unicode, err := decodeUTF16BE(unicodeData)
	if err != nil {
		return nil, fmt.Errorf("invalid UTF-16 in desc tag: %w", err)
	}

	if len(raw) <= offset {
		return DescTag{
			ASCII:   string(ascii),
			Unicode: unicode,
		}, nil
	}

	scriptCount := int(raw[offset])
	offset++
	if len(raw) < offset+scriptCount {
		return nil, fmt.Errorf("desc tag truncated: missing ScriptCode data")
	}
	script := string(raw[offset : offset+scriptCount])

	return DescTag{
		ASCII:   string(ascii),
		Unicode: unicode,
		Script:  script,
	}, nil
}

func textDecoder(raw []byte, hdrs []TagHeader) (any, error) {
	if len(raw) < 8 {
		return nil, fmt.Errorf("text tag too short")
	}
	text := raw[8:]
	text = bytes.TrimRight(text, "\x00 ")
	return string(text), nil
}

func sigDecoder(raw []byte, hdrs []TagHeader) (any, error) {
	if len(raw) < 8 {
		return nil, fmt.Errorf("sig tag too short")
	}
	return stringed(raw[8:12]), nil
}

type MultiLocalizedTag struct {
	Strings []LocalizedString
}

type LocalizedString struct {
	Language string // e.g. "en"
	Country  string // e.g. "US"
	Value    string
}

func mlucDecoder(raw []byte, hdrs []TagHeader) (any, error) {
	if len(raw) < 16 {
		return nil, fmt.Errorf("mluc tag too short")
	}

	count := int(binary.BigEndian.Uint32(raw[8:12]))
	recordSize := int(binary.BigEndian.Uint32(raw[12:16]))
	if recordSize != 12 {
		return nil, fmt.Errorf("unexpected mluc record size: %d", recordSize)
	}
	if len(raw) < 16+(count*recordSize) {
		return nil, fmt.Errorf("mluc tag too small for %d records", count)
	}

	tag := MultiLocalizedTag{Strings: make([]LocalizedString, 0, count)}
	for i := 0; i < count; i++ {
		base := 16 + i*recordSize

		langCode := string(raw[base : base+2])
		countryCode := string(raw[base+2 : base+4])
		strLen := int(binary.BigEndian.Uint32(raw[base+4 : base+8]))
		strOffset := int(binary.BigEndian.Uint32(raw[base+8 : base+12]))

		if strOffset+strLen > len(raw) || strLen%2 != 0 {
			return nil, fmt.Errorf("invalid string offset/length in mluc record %d", i)
		}

		strData := raw[strOffset : strOffset+strLen]
		decoded, err := decodeUTF16BE(strData)
		if err != nil {
			return nil, fmt.Errorf("invalid UTF-16 string in mluc record %d: %w", i, err)
		}

		tag.Strings = append(tag.Strings, LocalizedString{
			Language: langCode,
			Country:  countryCode,
			Value:    decoded,
		})
	}

	return tag, nil
}

type XYZNumber struct {
	X, Y, Z float64
}

func xyzDecoder(raw []byte, hdrs []TagHeader) (any, error) {
	if len(raw) < 20 {
		return nil, fmt.Errorf("XYZ tag too short")
	}
	body := raw[8:]
	if len(body)%12 != 0 {
		return nil, fmt.Errorf("XYZ tag has invalid length (not a multiple of 12)")
	}
	count := len(body) / 12
	result := make([]XYZNumber, 0, count)
	for i := 0; i < count; i++ {
		base := i * 12
		x := Fixed1616(binary.BigEndian.Uint32(body[base : base+4]))
		y := Fixed1616(binary.BigEndian.Uint32(body[base+4 : base+8]))
		z := Fixed1616(binary.BigEndian.Uint32(body[base+8 : base+12]))
		result = append(result, XYZNumber{
			X: x.Float64(),
			Y: y.Float64(),
			Z: z.Float64(),
		})
	}
	return result, nil
}

type CurveType uint

const (
	CurveTypeIdentity CurveType = iota
	CurveTypeGamma
	CurveTypePoints
)

type CurveTag struct {
	Type   CurveType
	Gamma  float64  // Type == CurveTypeGamma
	Points []uint16 // Type == CurveTypePoints
}

func curveDecoder(raw []byte, hdrs []TagHeader) (any, error) {
	if len(raw) < 12 {
		return nil, fmt.Errorf("curv tag too short")
	}
	count := int(binary.BigEndian.Uint32(raw[8:12]))
	if count == 0 {
		return CurveTag{Type: CurveTypeIdentity}, nil
	}
	if count == 1 {
		if len(raw) < 14 {
			return nil, fmt.Errorf("curv tag missing gamma value")
		}
		// 8.8 fixed-point
		gammaRaw := binary.BigEndian.Uint16(raw[12:14])
		return CurveTag{Type: CurveTypeGamma, Gamma: float64(gammaRaw) / 256.0}, nil
	}
	if len(raw) < 12+(count*2) {
		return nil, fmt.Errorf("curv tag truncated")
	}
	points := make([]uint16, count)
	for i := 0; i < count; i++ {
		points[i] = binary.BigEndian.Uint16(raw[12+i*2 : 14+i*2])
	}
	return CurveTag{Type: CurveTypePoints, Points: points}, nil
}

type ParametricCurveTag struct {
	FunctionType uint16
	Parameters   []float64
}

func parametricCurveDecoder(raw []byte, hdrs []TagHeader) (any, error) {
	if len(raw) < 12 {
		return nil, fmt.Errorf("para tag too short")
	}
	funcType := binary.BigEndian.Uint16(raw[8:10])
	var expected int
	switch funcType {
	case 0:
		expected = 1
	case 1:
		expected = 3
	case 2:
		expected = 4
	case 3:
		expected = 5
	case 4:
		expected = 7
	default:
		return nil, fmt.Errorf("unknown parametric function type: %d", funcType)
	}
	if len(raw) < 12+(expected*4) {
		return nil, fmt.Errorf("para tag truncated for function %d", funcType)
	}
	params := make([]float64, expected)
	for i := 0; i < expected; i++ {
		f := Fixed1616(binary.BigEndian.Uint32(raw[12+(i*4) : 16+(i*4)]))
		params[i] = f.Float64()
	}
	return ParametricCurveTag{
		FunctionType: funcType,
		Parameters:   params,
	}, nil
}

type MeasurementTag struct {
	Observer   uint32
	Backing    XYZNumber
	Geometry   uint32
	Flare      float64
	Illuminant uint32
}

func measurementDecoder(raw []byte, hdrs []TagHeader) (any, error) {
	if len(raw) < 36 {
		return nil, fmt.Errorf("meas tag too short")
	}
	return MeasurementTag{
		Observer: binary.BigEndian.Uint32(raw[8:12]),
		Backing: XYZNumber{
			X: Fixed1616(binary.BigEndian.Uint32(raw[12:16])).Float64(),
			Y: Fixed1616(binary.BigEndian.Uint32(raw[16:20])).Float64(),
			Z: Fixed1616(binary.BigEndian.Uint32(raw[20:24])).Float64(),
		},
		Geometry:   binary.BigEndian.Uint32(raw[24:28]),
		Flare:      Fixed1616(binary.BigEndian.Uint32(raw[28:32])).Float64(),
		Illuminant: binary.BigEndian.Uint32(raw[32:36]),
	}, nil
}

type ViewingConditionsTag struct {
	Illuminant     XYZNumber
	Surround       XYZNumber
	IlluminantType uint32
}

func viewDecoder(raw []byte, hdrs []TagHeader) (any, error) {
	if len(raw) < 36 {
		return nil, fmt.Errorf("view tag too short")
	}
	return ViewingConditionsTag{
		Illuminant: XYZNumber{
			X: Fixed1616(binary.BigEndian.Uint32(raw[8:12])).Float64(),
			Y: Fixed1616(binary.BigEndian.Uint32(raw[12:16])).Float64(),
			Z: Fixed1616(binary.BigEndian.Uint32(raw[16:20])).Float64(),
		},
		Surround: XYZNumber{
			X: Fixed1616(binary.BigEndian.Uint32(raw[20:24])).Float64(),
			Y: Fixed1616(binary.BigEndian.Uint32(raw[24:28])).Float64(),
			Z: Fixed1616(binary.BigEndian.Uint32(raw[28:32])).Float64(),
		},
		IlluminantType: binary.BigEndian.Uint32(raw[32:36]),
	}, nil
}

func psidDecoder(raw []byte, _ []TagHeader) (any, error) {
	//TODO rarely used and spec/usages don't match!
	return raw, nil
}

func pseqDecoder(raw []byte, _ []TagHeader) (any, error) {
	//TODO rarely useful, descriptive only
	return raw, nil
}

func gbdDecoder(raw []byte, _ []TagHeader) (any, error) {
	//TODO vendor-specific, not safely parseable without spec
	return raw, nil
}

func mft2Decoder(raw []byte, _ []TagHeader) (any, error) {
	// TODO: complex color lookup table, defer until needed
	return raw, nil
}

func mft1Decoder(raw []byte, _ []TagHeader) (any, error) {
	// TODO: complex color lookup table, defer until needed
	return raw, nil
}

func mABDecoder(raw []byte, _ []TagHeader) (any, error) {
	// TODO: multiProcessElementsType — full decoding deferred
	return raw, nil
}

func mBADecoder(raw []byte, _ []TagHeader) (any, error) {
	// TODO: multiProcessElementsType — full decoding deferred
	return raw, nil
}

func dictDecoder(raw []byte, _ []TagHeader) (any, error) {
	// TODO: dictionary tag (ICC.1:2010-12), parse only if needed
	return raw, nil
}

func zxmlDecoder(raw []byte, _ []TagHeader) (any, error) {
	// TODO: Vendor-specific, unknown encoding – stubbed
	return raw, nil
}

func msbnDecoder(raw []byte, _ []TagHeader) (any, error) {
	// TODO: Unknown vendor-specific tag (MSBN) – stubbed
	return raw, nil
}

func sf32Decoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 8 {
		return nil, fmt.Errorf("sf32 tag too short")
	}
	data := raw[8:] // skip type sig and reserved
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("sf32 float32 data not aligned")
	}
	count := len(data) / 4
	values := make([]float32, count)
	for i := 0; i < count; i++ {
		bits := binary.BigEndian.Uint32(data[i*4 : i*4+4])
		values[i] = math.Float32frombits(bits)
	}
	return values, nil
}

func decodeUTF16BE(data []byte) (string, error) {
	if len(data)%2 != 0 {
		return "", fmt.Errorf("odd length UTF-16BE string")
	}
	codeUnits := make([]uint16, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		codeUnits[i/2] = binary.BigEndian.Uint16(data[i : i+2])
	}
	return string(utf16.Decode(codeUnits)), nil
}
