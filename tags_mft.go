package iccarus

import (
	"encoding/binary"
	"fmt"
	"math"
)

type MFT2Tag struct {
	InputChannels  uint8
	OutputChannels uint8
	GridPoints     uint8 // same for all input dims
	Matrix         [9]float64
	InputCurves    [][]uint16
	CLUT           []float64 // flattened: len = grid^n * outputChannels
	OutputCurves   [][]uint16
}

type MFT1Tag struct {
	InputChannels  uint8
	OutputChannels uint8
	GridPoints     uint8
	Matrix         [9]float64
	InputCurves    [][]uint8 // Each input channel has a 256-entry curve
	CLUT           []float64 // Flat CLUT: len = grid^n * outputChannels
	OutputCurves   [][]uint8 // Each output channel has a 256-entry curve
}

func mft2Decoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 52 {
		return nil, fmt.Errorf("mft2 tag too short")
	}
	inCh := int(raw[8])
	outCh := int(raw[9])
	gridPoints := int(raw[10])
	matrix := [9]float64{}
	for i := 0; i < 9; i++ {
		offset := 12 + i*4
		if offset+4 > len(raw) {
			return nil, fmt.Errorf("mft2: matrix out of bounds")
		}
		matrix[i] = Fixed1616(binary.BigEndian.Uint32(raw[offset:])).Float64()
	}
	inputTableEntries := int(binary.BigEndian.Uint16(raw[48:50]))
	outputTableEntries := int(binary.BigEndian.Uint16(raw[50:52]))
	offset := 52
	inputCurves := make([][]uint16, inCh)
	for i := 0; i < inCh; i++ {
		end := offset + inputTableEntries*2
		if end > len(raw) {
			return nil, fmt.Errorf("mft2: input curve %d out of bounds", i)
		}
		curve := make([]uint16, inputTableEntries)
		for j := 0; j < inputTableEntries; j++ {
			curve[j] = binary.BigEndian.Uint16(raw[offset+(j*2):])
		}
		inputCurves[i] = curve
		offset = end
	}
	sizePerChannel := int(math.Pow(float64(gridPoints), float64(inCh)))
	clutValues := make([]float64, sizePerChannel*outCh)
	clutBytes := sizePerChannel * outCh * 2
	if offset+clutBytes > len(raw) {
		return nil, fmt.Errorf("mft2: clut out of bounds")
	}
	for i := 0; i < sizePerChannel*outCh; i++ {
		clutValues[i] = float64(binary.BigEndian.Uint16(raw[offset+i*2:])) / 65535.0
	}
	offset += clutBytes
	outputCurves := make([][]uint16, outCh)
	for i := 0; i < outCh; i++ {
		end := offset + outputTableEntries*2
		if end > len(raw) {
			return nil, fmt.Errorf("mft2: output curve %d out of bounds", i)
		}
		curve := make([]uint16, outputTableEntries)
		for j := 0; j < outputTableEntries; j++ {
			curve[j] = binary.BigEndian.Uint16(raw[offset+(j*2):])
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
		CLUT:           clutValues,
		OutputCurves:   outputCurves,
	}, nil
}

func mft1Decoder(raw []byte, _ []TagHeader) (any, error) {
	if len(raw) < 48 {
		return nil, fmt.Errorf("mft1 tag too short")
	}
	inCh := int(raw[8])
	outCh := int(raw[9])
	gridPoints := int(raw[10])
	matrix := [9]float64{}
	for i := 0; i < 9; i++ {
		offset := 12 + i*4
		if offset+4 > len(raw) {
			return nil, fmt.Errorf("mft1: matrix out of bounds")
		}
		matrix[i] = Fixed1616(binary.BigEndian.Uint32(raw[offset:])).Float64()
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
		return nil, fmt.Errorf("mft1: clut out of bounds")
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
