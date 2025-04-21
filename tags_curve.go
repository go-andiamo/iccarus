package iccarus

import (
	"encoding/binary"
	"fmt"
)

type CurveType uint

const (
	CurveTypeIdentity CurveType = iota
	CurveTypeGamma
	CurveTypePoints
)

type CurveTag struct {
	Type   CurveType
	Gamma  float64  // Type == CurveTypeGamma
	Points []uint16 // Type == CurveTypePoints
}

type ParametricCurveTag struct {
	FunctionType uint16
	Parameters   []float64
}

func curveDecoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 12 {
		return nil, fmt.Errorf("curv tag too short")
	}
	count := int(binary.BigEndian.Uint32(raw[8:12]))
	if count == 0 {
		return &CurveTag{Type: CurveTypeIdentity}, nil
	}
	if count == 1 {
		if len(raw) < 14 {
			return nil, fmt.Errorf("curv tag missing gamma value")
		}
		// 8.8 fixed-point
		gammaRaw := binary.BigEndian.Uint16(raw[12:14])
		return &CurveTag{Type: CurveTypeGamma, Gamma: float64(gammaRaw) / 256.0}, nil
	}
	if len(raw) < 12+(count*2) {
		return nil, fmt.Errorf("curv tag truncated")
	}
	points := make([]uint16, count)
	for i := 0; i < count; i++ {
		points[i] = binary.BigEndian.Uint16(raw[12+i*2 : 14+i*2])
	}
	return &CurveTag{Type: CurveTypePoints, Points: points}, nil
}

func parametricCurveDecoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 12 {
		return nil, fmt.Errorf("para tag too short")
	}
	funcType := binary.BigEndian.Uint16(raw[8:10])
	var expected int
	switch funcType {
	case 0:
		expected = 1
	case 1:
		expected = 3
	case 2:
		expected = 4
	case 3:
		expected = 5
	case 4:
		expected = 7
	default:
		return nil, fmt.Errorf("unknown parametric function type: %d", funcType)
	}
	if len(raw) < 12+(expected*4) {
		return nil, fmt.Errorf("para tag truncated for function %d", funcType)
	}
	params := make([]float64, expected)
	for i := 0; i < expected; i++ {
		f := Fixed1616(binary.BigEndian.Uint32(raw[12+(i*4) : 16+(i*4)]))
		params[i] = f.Float64()
	}
	return &ParametricCurveTag{
		FunctionType: funcType,
		Parameters:   params,
	}, nil
}
