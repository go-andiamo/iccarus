package iccarus

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

func TestSF32Decoder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("sf32")       // tag signature
		buf.Write([]byte{0, 0, 0, 0}) // reserved

		// Write 3 float32 values: 1.0, 0.5, -2.0
		for _, f := range []float32{1.0, 0.5, -2.0} {
			bits := math.Float32bits(f)
			_ = binary.Write(&buf, binary.BigEndian, bits)
		}

		val, err := sf32Decoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, []float32{}, val)

		result := val.([]float32)
		require.Len(t, result, 3)
		assert.InDelta(t, 1.0, result[0], 0.0001)
		assert.InDelta(t, 0.5, result[1], 0.0001)
		assert.InDelta(t, -2.0, result[2], 0.0001)
	})

	t.Run("TooShort", func(t *testing.T) {
		// Only 6 bytes (less than required 8)
		data := []byte{0, 1, 2, 3, 4, 5}
		_, err := sf32Decoder(data, nil)
		assert.ErrorContains(t, err, "sf32 tag too short")
	})

	t.Run("MisalignedFloatData", func(t *testing.T) {
		// 8 bytes header + 5 byte float data = 13 total
		buf := append([]byte("sf32\x00\x00\x00\x00"), []byte{1, 2, 3, 4, 5}...)
		_, err := sf32Decoder(buf, nil)
		assert.ErrorContains(t, err, "sf32 float32 data not aligned")
	})
}
