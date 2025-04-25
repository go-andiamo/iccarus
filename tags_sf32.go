package iccarus

import (
	"encoding/binary"
	"errors"
	"math"
)

func sf32Decoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 8 {
		return nil, errors.New("sf32 tag too short")
	}
	data := raw[8:] // skip type sig and reserved
	if len(data)%4 != 0 {
		return nil, errors.New("sf32 float32 data not aligned")
	}
	count := len(data) / 4
	values := make([]float32, count)
	for i := 0; i < count; i++ {
		bits := binary.BigEndian.Uint32(data[i*4 : i*4+4])
		values[i] = math.Float32frombits(bits)
	}
	return values, nil
}
