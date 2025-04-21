package iccarus

import (
	"encoding/binary"
	"fmt"
)

type MeasurementTag struct {
	Observer   uint32
	Backing    XYZNumber
	Geometry   uint32
	Flare      float64
	Illuminant uint32
}

func measurementDecoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 36 {
		return nil, fmt.Errorf("meas tag too short")
	}
	return &MeasurementTag{
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
