package iccarus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

type DescTag struct {
	ASCII   string
	Unicode string
	Script  string
}

func descDecoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 12 {
		return nil, errors.New("desc tag too short")
	}
	asciiLen := int(binary.BigEndian.Uint32(raw[8:12]))
	if asciiLen < 1 || 12+asciiLen > len(raw) {
		return nil, errors.New("invalid ASCII length in desc tag")
	}
	ascii := raw[12 : 12+asciiLen]
	if i := bytes.IndexByte(ascii, 0); i >= 0 {
		ascii = ascii[:i]
	}

	offset := 12 + asciiLen
	if len(raw) < offset+4 {
		return &DescTag{ASCII: string(ascii)}, nil // ASCII-only, no Unicode
	}

	unicodeCount := int(binary.BigEndian.Uint32(raw[offset : offset+4]))
	offset += 4
	if len(raw) < offset+(unicodeCount*2) {
		return nil, errors.New("desc tag truncated: missing UTF-16 data")
	}
	unicodeData := raw[offset : offset+(unicodeCount*2)]
	offset += unicodeCount * 2
	unicode := decodeUTF16BE(unicodeData)

	if len(raw) <= offset {
		return &DescTag{
			ASCII:   string(ascii),
			Unicode: unicode,
		}, nil
	}

	scriptCount := int(raw[offset])
	offset++
	if len(raw) < offset+scriptCount {
		return nil, errors.New("desc tag truncated: missing ScriptCode data")
	}
	script := string(raw[offset : offset+scriptCount])

	return &DescTag{
		ASCII:   string(ascii),
		Unicode: unicode,
		Script:  script,
	}, nil
}

func textDecoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 8 {
		return nil, errors.New("text tag too short")
	}
	text := raw[8:]
	text = bytes.TrimRight(text, "\x00")
	return string(text), nil
}

func sigDecoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 8 {
		return nil, errors.New("sig tag too short")
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

func mlucDecoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 16 {
		return nil, errors.New("mluc tag too short")
	}
	count := int(binary.BigEndian.Uint32(raw[8:12]))
	recordSize := int(binary.BigEndian.Uint32(raw[12:16]))
	if recordSize != 12 {
		return nil, fmt.Errorf("unexpected mluc record size: %d", recordSize)
	}
	if len(raw) < 16+(count*recordSize) {
		return nil, fmt.Errorf("mluc tag too small for %d records", count)
	}
	tag := &MultiLocalizedTag{Strings: make([]LocalizedString, 0, count)}
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
		decoded := decodeUTF16BE(strData)
		tag.Strings = append(tag.Strings, LocalizedString{
			Language: langCode,
			Country:  countryCode,
			Value:    decoded,
		})
	}
	return tag, nil
}

func decodeUTF16BE(data []byte) string {
	codeUnits := make([]uint16, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		codeUnits[i/2] = binary.BigEndian.Uint16(data[i : i+2])
	}
	return string(utf16.Decode(codeUnits))
}
