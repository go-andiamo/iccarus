package iccarus

import (
	"errors"
)

type XYZNumber struct {
	X, Y, Z float64
}

func xyzDecoder(raw []byte) (any, error) {
	if len(raw) < 20 {
		return nil, errors.New("XYZ tag too short")
	}
	body := raw[8:]
	if len(body)%12 != 0 {
		return nil, errors.New("XYZ tag has invalid length (not a multiple of 12)")
	}
	count := len(body) / 12
	result := make([]XYZNumber, 0, count)
	for i := 0; i < count; i++ {
		base := i * 12
		result = append(result, XYZNumber{
			X: readS15Fixed16BE(body[base : base+4]),
			Y: readS15Fixed16BE(body[base+4 : base+8]),
			Z: readS15Fixed16BE(body[base+8 : base+12]),
		})
	}
	return result, nil
}
