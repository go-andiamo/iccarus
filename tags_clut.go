package iccarus

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type CLUTTag struct {
	GridPoints     []uint8 // e.g., [17,17,17] for 3D CLUT
	InputChannels  uint8
	OutputChannels uint8
	Values         []float64 // flattened [in1, in2, ..., out1, out2, ...]
	expectedValues int
}

var _ ChannelTransformer = (*CLUTTag)(nil)

func clutDecoder(raw []byte) (any, error) {
	if len(raw) < 16 {
		return nil, errors.New("clut tag too short")
	}
	inputCh := int(raw[8])
	outputCh := int(raw[9])
	gridPoints := make([]uint8, inputCh)
	copy(gridPoints, raw[10:10+inputCh])
	body := raw[10+inputCh:]
	if len(body)%2 != 0 {
		return nil, errors.New("clut body size must be even")
	}
	// expected size: (product of grid points) * output channels * 2 bytes each
	expected := 1
	for _, gp := range gridPoints {
		expected *= int(gp)
	}
	expected *= outputCh * 2 // 2 bytes per value (uint16)
	if len(body) != expected {
		return nil, fmt.Errorf("CLUT unexpected body length: expected %d, got %d", expected, len(body))
	}
	values := make([]float64, 0, expected/2)
	for i := 0; i < len(body); i += 2 {
		v := binary.BigEndian.Uint16(body[i : i+2])
		values = append(values, float64(v)/65535.0)
	}
	return &CLUTTag{
		GridPoints:     gridPoints,
		InputChannels:  uint8(inputCh),
		OutputChannels: uint8(outputCh),
		Values:         values,
		expectedValues: expectedValues(gridPoints, outputCh),
	}, nil
}

func expectedValues(gridPoints []uint8, outputChannels int) int {
	expectedPoints := 1
	for _, g := range gridPoints {
		expectedPoints *= int(g)
	}
	return expectedPoints * outputChannels
}

func (c *CLUTTag) expected() int {
	if c.expectedValues == 0 {
		c.expectedValues = expectedValues(c.GridPoints, int(c.OutputChannels))
	}
	return c.expectedValues
}

func (c *CLUTTag) Transform(input []float64) ([]float64, error) {
	if len(input) != int(c.InputChannels) {
		return nil, fmt.Errorf("expected %d input channels, got %d", c.InputChannels, len(input))
	}
	if len(c.Values) < c.expected() {
		return nil, fmt.Errorf("not enough CLUT values: expected %d, got %d", c.expected(), len(c.Values))
	}
	return c.Lookup(input)
}

func (c *CLUTTag) Lookup(inputs []float64) ([]float64, error) {
	if len(inputs) != int(c.InputChannels) {
		return nil, fmt.Errorf("expected %d inputs, got %d", c.InputChannels, len(inputs))
	}
	if len(c.GridPoints) != int(c.InputChannels) {
		return nil, fmt.Errorf("grid points mismatch: expected %d, got %d", c.InputChannels, len(c.GridPoints))
	}
	// clamp input values to 0-1...
	clamped := make([]float64, len(inputs))
	for i, v := range inputs {
		clamped[i] = clamp01(v)
	}
	// find the grid positions and interpolation factors...
	gridPos := make([]int, len(clamped))
	gridFrac := make([]float64, len(clamped))
	for i, v := range clamped {
		nPoints := int(c.GridPoints[i])
		if nPoints < 2 {
			return nil, fmt.Errorf("CLUT input channel %d has invalid grid points: %d", i, nPoints)
		}
		pos := v * float64(nPoints-1)
		gridPos[i] = int(pos)
		if gridPos[i] >= nPoints-1 {
			gridPos[i] = nPoints - 2 // clamp
			gridFrac[i] = 1.0
		} else {
			gridFrac[i] = pos - float64(gridPos[i])
		}
	}
	// perform multi-dimensional interpolation (recursive)...
	return c.triLinearInterpolate(gridPos, gridFrac)
}

func (c *CLUTTag) triLinearInterpolate(gridPos []int, gridFrac []float64) ([]float64, error) {
	numInputs := int(c.InputChannels)
	numOutputs := int(c.OutputChannels)
	numCorners := 1 << numInputs // 2^inputs
	out := make([]float64, numOutputs)
	// walk all corners of the hypercube
	for corner := 0; corner < numCorners; corner++ {
		weight := 1.0
		idx := 0
		stride := 1
		for dim := numInputs - 1; dim >= 0; dim-- {
			bit := (corner >> dim) & 1
			pos := gridPos[dim] + bit
			if pos >= int(c.GridPoints[dim]) {
				return nil, fmt.Errorf("CLUT corner position out of bounds at dimension %d", dim)
			}
			idx += pos * stride
			stride *= int(c.GridPoints[dim])
			if bit == 0 {
				weight *= 1 - gridFrac[dim]
			} else {
				weight *= gridFrac[dim]
			}
		}
		base := idx * numOutputs
		if base+numOutputs > len(c.Values) {
			return nil, errors.New("CLUT value index out of bounds")
		}
		for o := 0; o < numOutputs; o++ {
			out[o] += weight * c.Values[base+o]
		}
	}
	return out, nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
