package iccarus

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCLUTDecoder(t *testing.T) {
	t.Run("Success3D", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("clut")       // 4 bytes
		buf.Write([]byte{0, 0, 0, 0}) // reserved (4 bytes)
		buf.WriteByte(3)              // input channels
		buf.WriteByte(3)              // output channels
		buf.Write([]byte{2, 2, 2})    // grid points for each input (2×2×2)
		// 2x2x2 = 8 grid points, 3 outputs per point = 24 outputs
		for i := 0; i < 8*3; i++ {
			val := uint16((i * 65535) / (8*3 - 1)) // Spread nicely 0..65535
			_ = binary.Write(&buf, binary.BigEndian, val)
		}

		val, err := clutDecoder(buf.Bytes())
		require.NoError(t, err)
		require.IsType(t, &CLUTTag{}, val)

		clut := val.(*CLUTTag)
		assert.Equal(t, uint8(3), clut.InputChannels)
		assert.Equal(t, uint8(3), clut.OutputChannels)
		assert.Equal(t, []uint8{2, 2, 2}, clut.GridPoints)
		assert.Len(t, clut.Values, 8*3)

		assert.InDelta(t, 0.0, clut.Values[0], 0.001)
		assert.InDelta(t, 1.0, clut.Values[len(clut.Values)-1], 0.001) // <-- Now will pass!
	})

	t.Run("TooShort", func(t *testing.T) {
		data := make([]byte, 15) // should be at least 16 bytes
		_, err := clutDecoder(data)
		assert.ErrorContains(t, err, "clut tag too short")
	})

	t.Run("BodyOddLength", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("clut")
		buf.Write([]byte{0, 0, 0, 0})
		buf.WriteByte(1)                    // input channels
		buf.WriteByte(1)                    // output channels
		buf.WriteByte(2)                    // grid points for input
		buf.Write([]byte{0x00, 0x00, 0x00}) // 3 bytes body: odd size
		for buf.Len() < 16 {
			buf.WriteByte(0)
		}
		_, err := clutDecoder(buf.Bytes())
		assert.ErrorContains(t, err, "clut body size must be even")
	})

	t.Run("UnexpectedBodyLength", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("clut")
		buf.Write([]byte{0, 0, 0, 0})
		buf.WriteByte(2)        // input channels
		buf.WriteByte(1)        // output channels
		buf.Write([]byte{2, 2}) // grid points 2x2
		// Should have 2x2 = 4 grid points × 1 output × 2 bytes = 8 bytes
		// Only write 4 bytes
		for i := 0; i < 2; i++ {
			_ = binary.Write(&buf, binary.BigEndian, uint16(i*1234))
		}
		_, err := clutDecoder(buf.Bytes())
		assert.ErrorContains(t, err, "CLUT unexpected body length")
	})
}

func TestCLUTTransform(t *testing.T) {
	t.Run("HappyPath_1D", func(t *testing.T) {
		clut := &CLUTTag{
			InputChannels:  1,
			OutputChannels: 1,
			GridPoints:     []uint8{2},
			Values:         []float64{0.0, 1.0}, // 2 values: for 1D input, 1 output channel
		}
		// Test input 0.0 → should return 0.0
		out, err := clut.Transform([]float64{0.0})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.InDelta(t, 0.0, out[0], 1e-6)
		// Test input 1.0 → should return 1.0
		out, err = clut.Transform([]float64{1.0})
		require.NoError(t, err)
		assert.InDelta(t, 1.0, out[0], 1e-6)
		// Test input 0.5 → should return 0.5 via interpolation
		out, err = clut.Transform([]float64{0.5})
		require.NoError(t, err)
		assert.InDelta(t, 0.5, out[0], 1e-6)
	})
	t.Run("HappyPath_3D", func(t *testing.T) {
		clut := &CLUTTag{
			InputChannels:  3,
			OutputChannels: 1,
			GridPoints:     []uint8{2, 2, 2},
			Values: []float64{
				0.0, 0.1, 0.2, 0.3,
				0.4, 0.5, 0.6, 1.0,
			}, // 8 points, 1 output each
		}
		out, err := clut.Transform([]float64{0.0, 0.0, 0.0}) // Should hit [0.0]
		require.NoError(t, err)
		assert.InDelta(t, 0.0, out[0], 1e-6)
		out, err = clut.Transform([]float64{1.0, 1.0, 1.0}) // Should hit [1.0]
		require.NoError(t, err)
		assert.InDelta(t, 1.0, out[0], 1e-6)
	})
	t.Run("WrongInputChannelCount", func(t *testing.T) {
		clut := &CLUTTag{
			InputChannels:  3,
			OutputChannels: 1,
			GridPoints:     []uint8{2, 2, 2},
			Values:         make([]float64, 8), // dummy
		}
		_, err := clut.Transform([]float64{0.5, 0.5}) // only 2 inputs
		assert.ErrorContains(t, err, "expected 3 input channels")
	})
	t.Run("EmptyCLUTValues", func(t *testing.T) {
		clut := &CLUTTag{
			InputChannels:  1,
			OutputChannels: 1,
			GridPoints:     []uint8{2},
			Values:         []float64{}, // empty
		}
		_, err := clut.Transform([]float64{0.0})
		assert.ErrorContains(t, err, "not enough CLUT values")
	})
}

func TestCLUTTag_Lookup_MismatchErrors(t *testing.T) {
	t.Run("InputChannelsMismatch", func(t *testing.T) {
		clut := &CLUTTag{
			InputChannels:  3,
			OutputChannels: 3,
			GridPoints:     []uint8{2, 2, 2},
			Values:         make([]float64, 8*3),
		}
		_, err := clut.Lookup([]float64{0.5, 0.5}) // only 2 inputs instead of 3
		assert.ErrorContains(t, err, "expected 3 inputs")
	})

	t.Run("GridPointsMismatch", func(t *testing.T) {
		clut := &CLUTTag{
			InputChannels:  3,
			OutputChannels: 3,
			GridPoints:     []uint8{2, 2}, // should be 3
			Values:         make([]float64, 8*3),
		}
		_, err := clut.Lookup([]float64{0.5, 0.5, 0.5})
		assert.ErrorContains(t, err, "grid points mismatch")
	})
}

func TestClamp01(t *testing.T) {
	require.Equal(t, float64(1), clamp01(1.0001))
	require.Equal(t, float64(0), clamp01(-1))
	require.Equal(t, float64(0.5), clamp01(0.5))
}
