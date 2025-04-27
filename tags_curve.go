package iccarus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type CurveType uint

const (
	CurveTypeIdentity CurveType = 0
	CurveTypeGamma    CurveType = 1
	CurveTypePoints   CurveType = 2
)

type CurveTag struct {
	Type   CurveType
	Gamma  float64  // Type == CurveTypeGamma
	Points []uint16 // Type == CurveTypePoints
}

var _ ChannelTransformer = (*CurveTag)(nil)

type ParametricCurveFunction uint16

const (
	SimpleGammaFunction     ParametricCurveFunction = 0 // Y = X^g
	ConditionalZeroFunction ParametricCurveFunction = 1 // Y = (aX+b)^g for X >= d, else 0
	ConditionalCFunction    ParametricCurveFunction = 2 // Y = (aX+b)^g for X >= d, else c
	SplitFunction           ParametricCurveFunction = 3 // Two different functions split at d
	ComplexFunction         ParametricCurveFunction = 4 // More complex piecewise function
)

type ParametricCurveTag struct {
	FunctionType ParametricCurveFunction
	Parameters   []float64
}

var _ ChannelTransformer = (*ParametricCurveTag)(nil)

func curveDecoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 12 {
		return nil, errors.New("curv tag too short")
	}
	count := int(binary.BigEndian.Uint32(raw[8:12]))
	if count == 0 {
		return &CurveTag{Type: CurveTypeIdentity}, nil
	}
	if count == 1 {
		if len(raw) < 14 {
			return nil, errors.New("curv tag missing gamma value")
		}
		// 8.8 fixed-point
		gammaRaw := binary.BigEndian.Uint16(raw[12:14])
		return &CurveTag{Type: CurveTypeGamma, Gamma: float64(gammaRaw) / 256.0}, nil
	}
	if len(raw) < 12+(count*2) {
		return nil, errors.New("curv tag truncated")
	}
	points := make([]uint16, count)
	for i := 0; i < count; i++ {
		points[i] = binary.BigEndian.Uint16(raw[12+i*2 : 14+i*2])
	}
	return &CurveTag{Type: CurveTypePoints, Points: points}, nil
}

func parametricCurveDecoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 12 {
		return nil, errors.New("para tag too short")
	}
	funcType := ParametricCurveFunction(binary.BigEndian.Uint16(raw[8:10]))
	var expected int
	switch funcType {
	case SimpleGammaFunction:
		expected = 1
	case ConditionalZeroFunction:
		expected = 3
	case ConditionalCFunction:
		expected = 4
	case SplitFunction:
		expected = 5
	case ComplexFunction:
		expected = 7
	default:
		return nil, fmt.Errorf("unknown parametric function type: %d", funcType)
	}
	offset := 10
	if len(raw) < offset+(expected*4) {
		return nil, fmt.Errorf("para tag truncated for function %d", funcType)
	}
	params := make([]float64, expected)
	for i := 0; i < expected; i++ {
		params[i] = readS15Fixed16BE(raw[offset : offset+4])
		offset += 4
	}
	return &ParametricCurveTag{
		FunctionType: funcType,
		Parameters:   params,
	}, nil
}

func (c *CurveTag) Transform(input []float64) ([]float64, error) {
	if len(input) != 1 {
		return nil, fmt.Errorf("curve expects 1 input, got %d", len(input))
	}
	v := input[0]
	switch c.Type {
	case CurveTypeIdentity:
		return []float64{v}, nil
	case CurveTypeGamma:
		return []float64{math.Pow(v, c.Gamma)}, nil
	case CurveTypePoints:
		if len(c.Points) == 0 {
			return nil, fmt.Errorf("curve has no points")
		}
		idx := v * float64(len(c.Points)-1)
		lo := int(math.Floor(idx))
		hi := int(math.Ceil(idx))
		if lo == hi {
			return []float64{float64(c.Points[lo]) / 65535.0}, nil
		}
		p := idx - float64(lo)
		vlo := float64(c.Points[lo]) / 65535.0
		vhi := float64(c.Points[hi]) / 65535.0
		return []float64{vlo + p*(vhi-vlo)}, nil
	default:
		return nil, fmt.Errorf("unknown curve type")
	}
}

func (p *ParametricCurveTag) Transform(input []float64) ([]float64, error) {
	if len(input) != 1 {
		return nil, fmt.Errorf("parametric curve expects 1 input, got %d", len(input))
	}
	x := input[0]
	var result float64
	switch p.FunctionType {
	case SimpleGammaFunction:
		// Y = X^g
		if len(p.Parameters) != 1 {
			return nil, fmt.Errorf("function 0 expects 1 parameter")
		}
		result = math.Pow(x, p.Parameters[0])
	case ConditionalZeroFunction:
		// Y = (aX+b)^g if X ≥ -b/a else 0
		if len(p.Parameters) != 3 {
			return nil, fmt.Errorf("function 1 expects 3 parameters")
		}
		a, b, g := p.Parameters[0], p.Parameters[1], p.Parameters[2]
		if x >= -b/a {
			result = math.Pow(a*x+b, g)
		} else {
			result = 0
		}
	case ConditionalCFunction:
		// Y = (aX+b)^g + c if X ≥ -b/a else c
		if len(p.Parameters) != 4 {
			return nil, fmt.Errorf("function 2 expects 4 parameters")
		}
		a, b, g, c := p.Parameters[0], p.Parameters[1], p.Parameters[2], p.Parameters[3]
		if x >= -b/a {
			result = math.Pow(a*x+b, g) + c
		} else {
			result = c
		}
	case SplitFunction:
		// Y = (aX+b)^g if X ≥ d else cX
		if len(p.Parameters) != 5 {
			return nil, fmt.Errorf("function 3 expects 5 parameters")
		}
		a, b, g, c, d := p.Parameters[0], p.Parameters[1], p.Parameters[2], p.Parameters[3], p.Parameters[4]
		if x >= d {
			result = math.Pow(a*x+b, g)
		} else {
			result = c * x
		}
	case ComplexFunction:
		// Y = (aX+b)^g + e if X ≥ d else cX+f
		if len(p.Parameters) != 7 {
			return nil, fmt.Errorf("function 4 expects 7 parameters")
		}
		a, b, g, c, d, e, f := p.Parameters[0], p.Parameters[1], p.Parameters[2], p.Parameters[3], p.Parameters[4], p.Parameters[5], p.Parameters[6]
		if x >= d {
			result = math.Pow(a*x+b, g) + e
		} else {
			result = c*x + f
		}
	default:
		return nil, fmt.Errorf("unknown parametric function type: %d", p.FunctionType)
	}
	return []float64{result}, nil
}
