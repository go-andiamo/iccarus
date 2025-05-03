package iccarus

import (
	"encoding/binary"
	"errors"
)

type ViewingConditionsTag struct {
	Illuminant     XYZNumber
	Surround       XYZNumber
	IlluminantType uint32
}

func viewDecoder(raw []byte) (any, error) {
	if len(raw) < 36 {
		return nil, errors.New("view tag too short")
	}
	return &ViewingConditionsTag{
		Illuminant: XYZNumber{
			X: readS15Fixed16BE(raw[8:12]),
			Y: readS15Fixed16BE(raw[12:16]),
			Z: readS15Fixed16BE(raw[16:20]),
		},
		Surround: XYZNumber{
			X: readS15Fixed16BE(raw[20:24]),
			Y: readS15Fixed16BE(raw[24:28]),
			Z: readS15Fixed16BE(raw[28:32]),
		},
		IlluminantType: binary.BigEndian.Uint32(raw[32:36]),
	}, nil
}
