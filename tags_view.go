package iccarus

import (
	"encoding/binary"
	"fmt"
)

type ViewingConditionsTag struct {
	Illuminant     XYZNumber
	Surround       XYZNumber
	IlluminantType uint32
}

func viewDecoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 36 {
		return nil, fmt.Errorf("view tag too short")
	}
	return &ViewingConditionsTag{
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
