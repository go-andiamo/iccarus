package iccarus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// MFT2Tag represents a multi function table 2 tag (TagMultiFunctionTable2)
type MFT2Tag struct {
	InputChannels  uint8
	OutputChannels uint8
	GridPoints     uint8
	Matrix         [9]float64
	InputCurves    [][]uint16
	CLUT           []float64 // flattened: len = grid^n * outputChannels
	OutputCurves   [][]uint16
}

var _ ChannelTransformer = (*MFT2Tag)(nil)

// MFT1Tag represents a multi function table 1 tag (TagMultiFunctionTable1)
type MFT1Tag struct {
	InputChannels  uint8
	OutputChannels uint8
	GridPoints     uint8
	Matrix         [9]float64
	InputCurves    [][]uint8 // Each input channel has a 256-entry curve
	CLUT           []float64 // Flat CLUT: len = grid^n * outputChannels
	OutputCurves   [][]uint8 // Each output channel has a 256-entry curve
}

var _ ChannelTransformer = (*MFT1Tag)(nil)

func mft2Decoder(raw []byte) (any, error) {
	if len(raw) < 52 {
		return nil, errors.New("mft2 tag too short")
	}
	inCh := int(raw[8])
	outCh := int(raw[9])
	gridPoints := int(raw[10])
	// Parse 9 matrix entries (S15Fixed16)
	matrix := [9]float64{}
	for i := 0; i < 9; i++ {
		off := 12 + i*4
		if off+4 > len(raw) {
			return nil, errors.New("mft2: matrix out of bounds")
		}
		matrix[i] = readS15Fixed16BE(raw[off : off+4])
	}
	// Input/output curve counts
	if len(raw) < 52 {
		return nil, errors.New("mft2: missing curve counts")
	}
	inputTableEntries := int(binary.BigEndian.Uint16(raw[48:50]))
	outputTableEntries := int(binary.BigEndian.Uint16(raw[50:52]))
	offset := 52
	// Input curves
	inputCurves := make([][]uint16, inCh)
	for i := 0; i < inCh; i++ {
		end := offset + inputTableEntries*2
		if end > len(raw) {
			return nil, fmt.Errorf("mft2: input curve %d out of bounds", i)
		}
		curve := make([]uint16, inputTableEntries)
		for j := 0; j < inputTableEntries; j++ {
			curve[j] = binary.BigEndian.Uint16(raw[offset+(j*2) : offset+(j*2)+2])
		}
		inputCurves[i] = curve
		offset = end
	}
	// CLUT block
	clutEntries := int(math.Pow(float64(gridPoints), float64(inCh))) * outCh
	clutBytes := clutEntries * 2
	if offset+clutBytes > len(raw) {
		return nil, errors.New("mft2: clut out of bounds")
	}
	clut := make([]float64, clutEntries)
	for i := 0; i < clutEntries; i++ {
		clut[i] = float64(binary.BigEndian.Uint16(raw[offset+(i*2):offset+(i*2)+2])) / 65535.0
	}
	offset += clutBytes
	// Output curves
	outputCurves := make([][]uint16, outCh)
	for i := 0; i < outCh; i++ {
		end := offset + outputTableEntries*2
		if end > len(raw) {
			return nil, fmt.Errorf("mft2: output curve %d out of bounds", i)
		}
		curve := make([]uint16, outputTableEntries)
		for j := 0; j < outputTableEntries; j++ {
			curve[j] = binary.BigEndian.Uint16(raw[offset+(j*2) : offset+(j*2)+2])
		}
		outputCurves[i] = curve
		offset = end
	}
	return &MFT2Tag{
		InputChannels:  uint8(inCh),
		OutputChannels: uint8(outCh),
		GridPoints:     uint8(gridPoints),
		Matrix:         matrix,
		InputCurves:    inputCurves,
		CLUT:           clut,
		OutputCurves:   outputCurves,
	}, nil
}

func (tag *MFT2Tag) Transform(input []float64) ([]float64, error) {
	if len(input) != int(tag.InputChannels) {
		return nil, fmt.Errorf("expected %d input channels, got %d", tag.InputChannels, len(input))
	}
	// Apply input curves with linear interpolation
	mapped := make([]float64, tag.InputChannels)
	for i := 0; i < int(tag.InputChannels); i++ {
		val := clamp01(input[i])
		curve := tag.InputCurves[i]
		pos := val * float64(len(curve)-1)
		lo := int(math.Floor(pos))
		hi := int(math.Ceil(pos))
		if lo < 0 || hi >= len(curve) {
			return nil, fmt.Errorf("input curve index out of bounds for channel %d", i)
		}
		if lo == hi {
			mapped[i] = float64(curve[lo]) / 65535.0
		} else {
			frac := pos - float64(lo)
			vlo := float64(curve[lo]) / 65535.0
			vhi := float64(curve[hi]) / 65535.0
			mapped[i] = vlo + frac*(vhi-vlo)
		}
	}
	// CLUT interpolation
	numInputs := int(tag.InputChannels)
	numOutputs := int(tag.OutputChannels)
	grid := int(tag.GridPoints)
	clut := tag.CLUT
	sizePerChannel := int(math.Pow(float64(grid), float64(numInputs)))
	if len(clut) < sizePerChannel*numOutputs {
		return nil, errors.New("CLUT index out of bounds")
	}
	// Compute interpolation weights
	gridPos := make([]int, numInputs)
	gridFrac := make([]float64, numInputs)
	for i := 0; i < numInputs; i++ {
		pos := mapped[i] * float64(grid-1)
		gridPos[i] = int(pos)
		if gridPos[i] >= grid-1 {
			gridPos[i] = grid - 2
			gridFrac[i] = 1.0
		} else {
			gridFrac[i] = pos - float64(gridPos[i])
		}
	}
	// Multi-linear interpolation
	out := make([]float64, numOutputs)
	numCorners := 1 << numInputs
	for corner := 0; corner < numCorners; corner++ {
		weight := 1.0
		idx := 0
		stride := 1
		for dim := numInputs - 1; dim >= 0; dim-- {
			bit := (corner >> dim) & 1
			pos := gridPos[dim] + bit
			idx += pos * stride
			stride *= grid
			if bit == 0 {
				weight *= 1 - gridFrac[dim]
			} else {
				weight *= gridFrac[dim]
			}
		}
		base := idx * numOutputs
		for o := 0; o < numOutputs; o++ {
			out[o] += clut[base+o] * weight
		}
	}
	// Apply output curves with interpolation
	for i := 0; i < int(tag.OutputChannels); i++ {
		val := clamp01(out[i])
		curve := tag.OutputCurves[i]
		pos := val * float64(len(curve)-1)
		lo := int(math.Floor(pos))
		hi := int(math.Ceil(pos))
		if lo < 0 || hi >= len(curve) {
			return nil, fmt.Errorf("output curve index out of bounds for channel %d", i)
		}
		if lo == hi {
			out[i] = float64(curve[lo]) / 65535.0
		} else {
			frac := pos - float64(lo)
			vlo := float64(curve[lo]) / 65535.0
			vhi := float64(curve[hi]) / 65535.0
			out[i] = vlo + frac*(vhi-vlo)
		}
	}
	return out, nil
}

func mft1Decoder(raw []byte) (any, error) {
	if len(raw) < 48 {
		return nil, errors.New("mft1 tag too short")
	}
	inCh := int(raw[8])
	outCh := int(raw[9])
	gridPoints := int(raw[10])
	matrix := [9]float64{}
	for i := 0; i < 9; i++ {
		offset := 12 + i*4
		if offset+4 > len(raw) {
			return nil, errors.New("mft1: matrix out of bounds")
		}
		matrix[i] = readS15Fixed16BE(raw[offset : offset+4])
	}
	offset := 48
	inputCurves := make([][]uint8, inCh)
	for i := 0; i < inCh; i++ {
		if offset+256 > len(raw) {
			return nil, fmt.Errorf("mft1: input curve %d out of bounds", i)
		}
		inputCurves[i] = raw[offset : offset+256]
		offset += 256
	}
	sizePerChannel := int(math.Pow(float64(gridPoints), float64(inCh)))
	clutValues := make([]float64, sizePerChannel*outCh)
	clutBytes := sizePerChannel * outCh
	if offset+clutBytes > len(raw) {
		return nil, errors.New("mft1: clut out of bounds")
	}
	for i := 0; i < sizePerChannel*outCh; i++ {
		clutValues[i] = float64(raw[offset+i]) / 255.0
	}
	offset += clutBytes
	outputCurves := make([][]uint8, outCh)
	for i := 0; i < outCh; i++ {
		if offset+256 > len(raw) {
			return nil, fmt.Errorf("mft1: output curve %d out of bounds", i)
		}
		outputCurves[i] = raw[offset : offset+256]
		offset += 256
	}
	return &MFT1Tag{
		InputChannels:  uint8(inCh),
		OutputChannels: uint8(outCh),
		GridPoints:     uint8(gridPoints),
		Matrix:         matrix,
		InputCurves:    inputCurves,
		CLUT:           clutValues,
		OutputCurves:   outputCurves,
	}, nil
}

func (m *MFT1Tag) Transform(input []float64) ([]float64, error) {
	if len(input) != int(m.InputChannels) {
		return nil, fmt.Errorf("mft1: expected %d input channels, got %d", m.InputChannels, len(input))
	}
	// 1. Apply input curves
	curved := make([]float64, len(input))
	for i, v := range input {
		v = clamp01(v)
		idx := int(v * 255.0)
		if idx >= len(m.InputCurves[i]) {
			return nil, fmt.Errorf("mft1: input curve %d out of bounds", idx)
		}
		curved[i] = float64(m.InputCurves[i][idx]) / 255.0
	}
	// 2. Compute grid coordinates
	numInputs := int(m.InputChannels)
	numOutputs := int(m.OutputChannels)
	grid := int(m.GridPoints)
	positions := make([]int, numInputs)
	fracs := make([]float64, numInputs)
	for i, v := range curved {
		p := v * float64(grid-1)
		pos := int(math.Floor(p))
		if pos >= grid-1 {
			pos = grid - 2
			fracs[i] = 1.0
		} else {
			fracs[i] = p - float64(pos)
		}
		positions[i] = pos
	}
	// 3. Interpolate from CLUT
	result := make([]float64, numOutputs)
	numCorners := 1 << numInputs
	for corner := 0; corner < numCorners; corner++ {
		weight := 1.0
		index := 0
		stride := 1
		for dim := numInputs - 1; dim >= 0; dim-- {
			bit := (corner >> dim) & 1
			pos := positions[dim] + bit
			if pos >= grid {
				return nil, fmt.Errorf("mft1: grid index out of bounds at dim %d", dim)
			}
			index += pos * stride
			stride *= grid
			if bit == 0 {
				weight *= 1 - fracs[dim]
			} else {
				weight *= fracs[dim]
			}
		}
		base := index * numOutputs
		if base+numOutputs > len(m.CLUT) {
			return nil, errors.New("mft1: CLUT index out of bounds")
		}
		for i := 0; i < numOutputs; i++ {
			result[i] += weight * m.CLUT[base+i]
		}
	}
	// 4. Apply output curves
	final := make([]float64, numOutputs)
	for i, v := range result {
		v = clamp01(v)
		idx := int(v * 255.0)
		if idx >= len(m.OutputCurves[i]) {
			return nil, fmt.Errorf("mft1: output curve %d out of bounds", idx)
		}
		final[i] = float64(m.OutputCurves[i][idx]) / 255.0
	}
	return final, nil
}
