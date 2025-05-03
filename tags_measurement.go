package iccarus

import (
	"encoding/binary"
	"errors"
)

type MeasurementTag struct {
	Observer   uint32
	Backing    XYZNumber
	Geometry   uint32
	Flare      float64
	Illuminant uint32
}

func measurementDecoder(raw []byte) (any, error) {
	if len(raw) < 36 {
		return nil, errors.New("meas tag too short")
	}
	return &MeasurementTag{
		Observer: binary.BigEndian.Uint32(raw[8:12]),
		Backing: XYZNumber{
			X: readS15Fixed16BE(raw[12:16]),
			Y: readS15Fixed16BE(raw[16:20]),
			Z: readS15Fixed16BE(raw[20:24]),
		},
		Geometry:   binary.BigEndian.Uint32(raw[24:28]),
		Flare:      readS15Fixed16BE(raw[28:32]),
		Illuminant: binary.BigEndian.Uint32(raw[32:36]),
	}, nil
}
