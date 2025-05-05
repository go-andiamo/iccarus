package iccarus

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ModularTag represents a modular tag (TagModularAB / TagModularBA)
type ModularTag struct {
	Signature      string
	InputChannels  uint8
	OutputChannels uint8
	Elements       []*Tag
}

var _ ToCIEXYZ = (*ModularTag)(nil)
var _ FromCIEXYZ = (*ModularTag)(nil)

func modularDecoder(raw []byte) (any, error) {
	if len(raw) < 12 {
		return nil, errors.New("modular (mAB/mBA) tag too short")
	}
	inputCh := int(binary.BigEndian.Uint16(raw[8:10]))
	outputCh := int(binary.BigEndian.Uint16(raw[10:12]))
	offset := 12
	offsets := []int{}
	// decide if there's an offset table or not
	if offset+4 <= len(raw) {
		first := binary.BigEndian.Uint32(raw[offset : offset+4])
		// check if the "first offset" is actually an offset? Or just start of element?
		if first < uint32(len(raw)) && !isASCII(raw[offset:offset+4]) {
			// it's probably a real offset table
			for offset+4 <= len(raw) {
				elemOffset := int(binary.BigEndian.Uint32(raw[offset : offset+4]))
				if elemOffset == 0 || (len(offsets) > 0 && elemOffset <= offsets[len(offsets)-1]) {
					break
				}
				if elemOffset >= len(raw) {
					break
				}
				offsets = append(offsets, elemOffset)
				offset += 4
			}
		}
	}
	if len(offsets) == 0 {
		// fallback - assume single element begins directly after header
		offsets = append(offsets, offset)
	}
	elements := make([]*Tag, 0, len(offsets))
	for i, elemStart := range offsets {
		elemEnd := len(raw)
		if i+1 < len(offsets) {
			elemEnd = offsets[i+1]
		}
		if elemEnd > len(raw) {
			return nil, fmt.Errorf("modular (mAB/mBA): element %d end offset 0x%X out of bounds", i, elemEnd)
		}
		elemBlock := raw[elemStart:elemEnd]
		if len(elemBlock) < 8 {
			return nil, fmt.Errorf("modular (mAB/mBA): element %d too short to contain header", i)
		}
		tagSig := string(elemBlock[0:4])
		elements = append(elements, decodeEmbeddedTag(tagSig, elemBlock))
	}
	return &ModularTag{
		Signature:      stringed(raw[:4]),
		InputChannels:  uint8(inputCh),
		OutputChannels: uint8(outputCh),
		Elements:       elements,
	}, nil
}

func isASCII(b []byte) bool {
	for _, ch := range b[:4] {
		if ch < 32 || ch > 126 {
			return false
		}
	}
	return true
}

func decodeEmbeddedTag(tagSig string, raw []byte) *Tag {
	result := &Tag{
		Name:    tagSig,
		Raw:     raw,
		decoder: defaultDecoders[tagSig],
	}
	if result.decoder == nil {
		result.Name = string(raw[:4])
		result.error = fmt.Errorf("unknown embedded tag type: %q", tagSig)
	} else {
		result.value, result.error = result.decoder(raw)
	}
	return result
}

func (m *ModularTag) ToCIEXYZ(channels ...float64) ([]float64, error) {
	return m.transformChannels(channels)
}

func (m *ModularTag) FromCIEXYZ(channels ...float64) ([]float64, error) {
	return m.transformChannels(channels)
}

func (m *ModularTag) transformChannels(channels []float64) ([]float64, error) {
	if len(channels) != int(m.InputChannels) {
		return nil, fmt.Errorf("expected %d input channels, got %d", m.InputChannels, len(channels))
	}
	result := channels
	applied := false
	for _, element := range m.Elements {
		switch element.Name {
		case TagCurve, TagParametricCurve, TagMatrix, TagColorLookupTable:
			val, err := element.Value()
			if err != nil {
				return nil, fmt.Errorf("failed decoding modular element %q: %w", element.Name, err)
			}
			if proc, ok := val.(ChannelTransformer); ok {
				result, err = proc.Transform(result...)
				if err != nil {
					return nil, fmt.Errorf("failed processing modular element %q: %w", element.Name, err)
				}
				applied = true
			}
		}
	}
	if !applied {
		return nil, errors.New("modular tag has no transformable elements")
	}
	return result, nil
}
