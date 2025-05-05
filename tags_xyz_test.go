package iccarus

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestXYZDecoder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("XYZ ")       // 4-byte type signature
		buf.Write([]byte{0, 0, 0, 0}) // reserved
		// One XYZ triplet:
		// X = 0.9642, Y = 1.0, Z = 0.8249
		buf.Write(encodeS15Fixed16BE(0.9642)) // X
		buf.Write(encodeS15Fixed16BE(1.0))    // Y
		buf.Write(encodeS15Fixed16BE(0.8249)) // Z
		val, err := xyzDecoder(buf.Bytes())
		require.NoError(t, err)
		require.IsType(t, []XYZNumber{}, val)
		xyz := val.([]XYZNumber)
		require.Len(t, xyz, 1)
		assert.InDelta(t, 0.9642, xyz[0].X, 0.001)
		assert.InDelta(t, 1.0, xyz[0].Y, 0.001)
		assert.InDelta(t, 0.8249, xyz[0].Z, 0.001)
	})
	t.Run("TooShort", func(t *testing.T) {
		raw := make([]byte, 19) // less than required 20
		_, err := xyzDecoder(raw)
		assert.ErrorContains(t, err, "XYZ tag too short")
	})
	t.Run("InvalidLength", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("XYZ ")
		buf.Write([]byte{0, 0, 0, 0}) // reserved
		buf.Write(make([]byte, 13))   // not a multiple of 12
		_, err := xyzDecoder(buf.Bytes())
		assert.ErrorContains(t, err, "invalid length")
	})
}
