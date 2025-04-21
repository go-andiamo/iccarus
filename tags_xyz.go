package iccarus

import (
	"encoding/binary"
	"fmt"
)

type XYZNumber struct {
	X, Y, Z float64
}

func xyzDecoder(raw []byte, _ []TagHeader) (any, error) {
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
