package iccarus

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMFT2Decoder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("mft2")       // tag signature
		buf.Write([]byte{0, 0, 0, 0}) // reserved
		buf.Write([]byte{1, 1, 2, 0}) // inputCh=1, outputCh=1, gridPoints=2, reserved

		// Identity matrix (9 * 4 bytes)
		for i := 0; i < 9; i++ {
			if i%4 == 0 {
				_ = binary.Write(&buf, binary.BigEndian, uint32(0x00010000)) // 1.0
			} else {
				_ = binary.Write(&buf, binary.BigEndian, uint32(0x00000000)) // 0.0
			}
		}

		// Input + output curve sizes (2 bytes each)
		_ = binary.Write(&buf, binary.BigEndian, uint16(4)) // input curve length
		_ = binary.Write(&buf, binary.BigEndian, uint16(4)) // output curve length

		// Input curve: 4 entries (0, 21845, 43690, 65535)
		for _, val := range []uint16{0, 21845, 43690, 65535} {
			_ = binary.Write(&buf, binary.BigEndian, val)
		}

		// CLUT: 2 grid points ^ 1 input * 1 output = 2 values
		for _, val := range []uint16{0x8000, 0xFFFF} {
			_ = binary.Write(&buf, binary.BigEndian, val)
		}

		// Output curve: 4 entries (65535, 43690, 21845, 0)
		for _, val := range []uint16{65535, 43690, 21845, 0} {
			_ = binary.Write(&buf, binary.BigEndian, val)
		}

		val, err := mft2Decoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &MFT2Tag{}, val)

		tag := val.(*MFT2Tag)
		assert.Equal(t, uint8(1), tag.InputChannels)
		assert.Equal(t, uint8(1), tag.OutputChannels)
		assert.Equal(t, uint8(2), tag.GridPoints)

		assert.Len(t, tag.InputCurves, 1)
		assert.Equal(t, []uint16{0, 21845, 43690, 65535}, tag.InputCurves[0])

		assert.Len(t, tag.CLUT, 2)
		assert.InDelta(t, 0.5, tag.CLUT[0], 0.01)
		assert.InDelta(t, 1.0, tag.CLUT[1], 0.01)

		assert.Len(t, tag.OutputCurves, 1)
		assert.Equal(t, []uint16{65535, 43690, 21845, 0}, tag.OutputCurves[0])
	})

	t.Run("TooShort", func(t *testing.T) {
		data := make([]byte, 51) // less than required 52
		_, err := mft2Decoder(data, nil)
		assert.ErrorContains(t, err, "mft2 tag too short")
	})

	t.Run("MatrixOutOfBounds", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("mft2")       // Tag sig (4 bytes)
		buf.Write([]byte{0, 0, 0, 0}) // Reserved (4 bytes)
		buf.Write([]byte{3, 3, 2, 0}) // inCh, outCh, gridPoints, padding (4 bytes)

		// Write only *partial* matrix (say, 8 bytes instead of 36)
		buf.Write(make([]byte, 8))

		// DO NOT pad to 52 bytes!
		// We want len(buf) < 52 so it fails early

		_, err := mft2Decoder(buf.Bytes(), nil)
		assert.ErrorContains(t, err, "mft2 tag too short")
	})

	t.Run("InputCurveOutOfBounds", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("mft2")
		buf.Write([]byte{0, 0, 0, 0, 1, 1, 2, 0})             // 1 in/out, 2 grid
		buf.Write(make([]byte, 36))                           // dummy matrix
		_ = binary.Write(&buf, binary.BigEndian, uint16(100)) // input curve size too big
		_ = binary.Write(&buf, binary.BigEndian, uint16(4))   // output curve size
		_, err := mft2Decoder(buf.Bytes(), nil)
		assert.ErrorContains(t, err, "mft2: input curve 0 out of bounds")
	})

	t.Run("CLUTOutOfBounds", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("mft2")
		buf.Write([]byte{0, 0, 0, 0, 1, 1, 2, 0}) // 1 in/out, 2 grid
		// matrix
		for i := 0; i < 9; i++ {
			_ = binary.Write(&buf, binary.BigEndian, uint32(0))
		}
		_ = binary.Write(&buf, binary.BigEndian, uint16(1))     // input curve = 1
		_ = binary.Write(&buf, binary.BigEndian, uint16(1))     // output curve = 1
		_ = binary.Write(&buf, binary.BigEndian, uint16(12345)) // junk input curve data

		// too little for CLUT
		_, err := mft2Decoder(buf.Bytes(), nil)
		assert.ErrorContains(t, err, "mft2: clut out of bounds")
	})
}

func TestMFT1Decoder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("mft1")       // tag signature
		buf.Write([]byte{0, 0, 0, 0}) // reserved
		buf.Write([]byte{1, 1, 2, 0}) // inputCh=1, outputCh=1, gridPoints=2, reserved

		// Identity matrix (9 * 4 bytes)
		for i := 0; i < 9; i++ {
			if i%4 == 0 {
				_ = binary.Write(&buf, binary.BigEndian, uint32(0x00010000)) // 1.0
			} else {
				_ = binary.Write(&buf, binary.BigEndian, uint32(0x00000000)) // 0.0
			}
		}

		// Input curve: 256 bytes (linear ramp 0..255)
		for i := 0; i < 256; i++ {
			buf.WriteByte(byte(i))
		}

		// CLUT: 2 bytes (grid^inputCh * outputCh = 2)
		buf.WriteByte(0x80)
		buf.WriteByte(0xFF)

		// Output curve: 256 bytes (reverse ramp 255..0)
		for i := 255; i >= 0; i-- {
			buf.WriteByte(byte(i))
		}

		val, err := mft1Decoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &MFT1Tag{}, val)

		tag := val.(*MFT1Tag)
		assert.Equal(t, uint8(1), tag.InputChannels)
		assert.Equal(t, uint8(1), tag.OutputChannels)
		assert.Equal(t, uint8(2), tag.GridPoints)

		assert.Len(t, tag.InputCurves, 1)
		assert.Len(t, tag.InputCurves[0], 256)
		assert.Equal(t, byte(0), tag.InputCurves[0][0])
		assert.Equal(t, byte(255), tag.InputCurves[0][255])

		assert.Len(t, tag.CLUT, 2)
		assert.InDelta(t, 0.5, tag.CLUT[0], 0.01)
		assert.InDelta(t, 1.0, tag.CLUT[1], 0.01)

		assert.Len(t, tag.OutputCurves, 1)
		assert.Len(t, tag.OutputCurves[0], 256)
		assert.Equal(t, byte(255), tag.OutputCurves[0][0])
		assert.Equal(t, byte(0), tag.OutputCurves[0][255])
	})

	t.Run("TooShort", func(t *testing.T) {
		data := make([]byte, 47) // less than required 48 bytes
		_, err := mft1Decoder(data, nil)
		assert.ErrorContains(t, err, "mft1 tag too short")
	})
}
