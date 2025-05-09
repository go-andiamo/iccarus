package iccarus

import (
	"errors"
	"fmt"
)

// MatrixTag represents a matrix tag (TagMatrix)
type MatrixTag struct {
	Matrix [3][3]float64
	Offset *[3]float64 // offset is not always present
}

var _ ChannelTransformer = (*MatrixTag)(nil)

func mtxDecoder(raw []byte) (any, error) {
	const (
		minLength     = 8 + (9 * 4) // 8 bytes for type/reserved + 9 * 4-byte matrix numbers
		offsetsLength = 12 * 4      // 4 * matrix numbers (4-byte) + 3 * offset numbers
	)
	if len(raw) < minLength {
		return nil, errors.New("mtx tag too short")
	}
	result := &MatrixTag{}
	body := raw[8:]
	for i := 0; i < 9; i++ {
		result.Matrix[i/3][i%3] = readS15Fixed16BE(body[i*4 : (i+1)*4])
	}
	if len(body) >= offsetsLength {
		offset := [3]float64{}
		for i := 0; i < 3; i++ {
			offset[i] = readS15Fixed16BE(body[36+i*4 : 36+(i+1)*4])
		}
		result.Offset = &offset
	}
	return result, nil
}

func (m *MatrixTag) Transform(inputs ...float64) ([]float64, error) {
	if len(inputs) != 3 {
		return nil, fmt.Errorf("matrix transform expects 3 inputs, got %d", len(inputs))
	}
	out := make([]float64, 3)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			out[i] += m.Matrix[i][j] * inputs[j]
		}
	}
	if m.Offset != nil {
		for i := 0; i < 3; i++ {
			out[i] += m.Offset[i]
		}
	}
	return out, nil
}
