package iccarus

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestViewDecoder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("view")       // 4 bytes
		buf.Write([]byte{0, 0, 0, 0}) // reserved
		// Illuminant XYZ: X ≈ 0.9642, Y = 1.0, Z ≈ 0.8249
		buf.Write(encodeS15Fixed16BE(0.9642)) // X
		buf.Write(encodeS15Fixed16BE(1.0))    // Y
		buf.Write(encodeS15Fixed16BE(0.8249)) // Z
		// Surround XYZ: X = 0.2 (0x00003333), Y = 0.3 (0x00004CCC), Z = 0.4 (0x00006666)
		buf.Write(encodeS15Fixed16BE(0.2)) // X
		buf.Write(encodeS15Fixed16BE(0.3)) // Y
		buf.Write(encodeS15Fixed16BE(0.4)) // Z
		// Illuminant Type = 3
		buf.Write([]byte{0x00, 0x00, 0x00, 0x03})
		val, err := viewDecoder(buf.Bytes())
		require.NoError(t, err)
		require.IsType(t, &ViewingConditionsTag{}, val)
		view := val.(*ViewingConditionsTag)
		assert.InDelta(t, 0.9642, view.Illuminant.X, 0.001)
		assert.InDelta(t, 1.0, view.Illuminant.Y, 0.001)
		assert.InDelta(t, 0.8249, view.Illuminant.Z, 0.001)
		assert.InDelta(t, 0.2, view.Surround.X, 0.001)
		assert.InDelta(t, 0.3, view.Surround.Y, 0.001)
		assert.InDelta(t, 0.4, view.Surround.Z, 0.001)
		assert.Equal(t, uint32(3), view.IlluminantType)
	})
	t.Run("TooShort", func(t *testing.T) {
		data := make([]byte, 35) // one byte too short
		_, err := viewDecoder(data)
		assert.ErrorContains(t, err, "view tag too short")
	})
}
