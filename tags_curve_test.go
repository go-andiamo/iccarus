package iccarus

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCurveDecoder(t *testing.T) {
	t.Run("IdentityCurve", func(t *testing.T) {
		raw := []byte("curv\x00\x00\x00\x00" + // sig + reserved
			"\x00\x00\x00\x00") // count = 0
		val, err := curveDecoder([]byte(raw), nil)
		require.NoError(t, err)
		require.IsType(t, &CurveTag{}, val)
		c := val.(*CurveTag)
		assert.Equal(t, CurveTypeIdentity, c.Type)
	})

	t.Run("GammaCurve", func(t *testing.T) {
		raw := []byte("curv\x00\x00\x00\x00" + // sig + reserved
			"\x00\x00\x00\x01" + // count = 1
			"\x01\x00") // gamma = 1.0
		val, err := curveDecoder([]byte(raw), nil)
		require.NoError(t, err)
		require.IsType(t, &CurveTag{}, val)
		c := val.(*CurveTag)
		assert.Equal(t, CurveTypeGamma, c.Type)
		assert.InDelta(t, 1.0, c.Gamma, 0.001)
	})

	t.Run("PointsCurve", func(t *testing.T) {
		raw := []byte("curv\x00\x00\x00\x00" + // sig + reserved
			"\x00\x00\x00\x03" + // count = 3
			"\x00\x10\x00\x20\x00\x30") // 3 x uint16
		val, err := curveDecoder([]byte(raw), nil)
		require.NoError(t, err)
		require.IsType(t, &CurveTag{}, val)
		c := val.(*CurveTag)
		assert.Equal(t, CurveTypePoints, c.Type)
		assert.Equal(t, []uint16{0x10, 0x20, 0x30}, c.Points)
	})

	t.Run("TooShort", func(t *testing.T) {
		_, err := curveDecoder(make([]byte, 11), nil)
		assert.ErrorContains(t, err, "curv tag too short")
	})

	t.Run("MissingGamma", func(t *testing.T) {
		raw := []byte("curv\x00\x00\x00\x00" +
			"\x00\x00\x00\x01") // count = 1 (but no gamma value)
		_, err := curveDecoder(raw, nil)
		assert.ErrorContains(t, err, "curv tag missing gamma value")
	})

	t.Run("TruncatedPoints", func(t *testing.T) {
		raw := []byte("curv\x00\x00\x00\x00" +
			"\x00\x00\x00\x02" + // count = 2
			"\x00\x10") // missing second uint16
		_, err := curveDecoder(raw, nil)
		assert.ErrorContains(t, err, "curv tag truncated")
	})
}

func TestParametricCurveDecoder(t *testing.T) {
	t.Run("Function0", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("para")                             // 4 bytes
		buf.Write([]byte{0, 0, 0, 0})                       // reserved
		_ = binary.Write(&buf, binary.BigEndian, uint16(0)) // function type 0
		buf.Write(encodeS15Fixed16BE(1.0))                  // 1.0

		val, err := parametricCurveDecoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &ParametricCurveTag{}, val)
		p := val.(*ParametricCurveTag)
		assert.Equal(t, uint16(0), p.FunctionType)
		assert.Len(t, p.Parameters, 1)
		assert.InDelta(t, 1.0, p.Parameters[0], 0.0001)
	})

	t.Run("Function1", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("para")
		buf.Write([]byte{0, 0, 0, 0})
		_ = binary.Write(&buf, binary.BigEndian, uint16(1))
		for i := 0; i < 3; i++ {
			buf.Write(encodeS15Fixed16BE(float64(i)))
		}
		val, err := parametricCurveDecoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &ParametricCurveTag{}, val)
		p := val.(*ParametricCurveTag)
		assert.Equal(t, uint16(1), p.FunctionType)
		assert.Len(t, p.Parameters, 3)
		assert.InDelta(t, 0.0, p.Parameters[0], 0.0001)
		assert.InDelta(t, 1.0, p.Parameters[1], 0.0001)
		assert.InDelta(t, 2.0, p.Parameters[2], 0.0001)
	})

	t.Run("Function2", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("para")
		buf.Write([]byte{0, 0, 0, 0})
		_ = binary.Write(&buf, binary.BigEndian, uint16(2))
		for i := 0; i < 4; i++ {
			buf.Write(encodeS15Fixed16BE(float64(i)))
		}
		val, err := parametricCurveDecoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &ParametricCurveTag{}, val)
		p := val.(*ParametricCurveTag)
		assert.Equal(t, uint16(2), p.FunctionType)
		assert.Len(t, p.Parameters, 4)
		assert.InDelta(t, 0.0, p.Parameters[0], 0.0001)
		assert.InDelta(t, 1.0, p.Parameters[1], 0.0001)
		assert.InDelta(t, 2.0, p.Parameters[2], 0.0001)
		assert.InDelta(t, 3.0, p.Parameters[3], 0.0001)
	})

	t.Run("Function3", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("para")
		buf.Write([]byte{0, 0, 0, 0})
		_ = binary.Write(&buf, binary.BigEndian, uint16(3))
		for i := 0; i < 5; i++ {
			buf.Write(encodeS15Fixed16BE(float64(i)))
		}
		val, err := parametricCurveDecoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &ParametricCurveTag{}, val)
		p := val.(*ParametricCurveTag)
		assert.Equal(t, uint16(3), p.FunctionType)
		assert.Len(t, p.Parameters, 5)
		assert.InDelta(t, 0.0, p.Parameters[0], 0.0001)
		assert.InDelta(t, 1.0, p.Parameters[1], 0.0001)
		assert.InDelta(t, 2.0, p.Parameters[2], 0.0001)
		assert.InDelta(t, 3.0, p.Parameters[3], 0.0001)
		assert.InDelta(t, 4.0, p.Parameters[4], 0.0001)
	})

	t.Run("Function4", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("para")
		buf.Write([]byte{0, 0, 0, 0})
		_ = binary.Write(&buf, binary.BigEndian, uint16(4))
		for i := 0; i < 7; i++ {
			buf.Write(encodeS15Fixed16BE(float64(i)))
		}
		val, err := parametricCurveDecoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &ParametricCurveTag{}, val)
		p := val.(*ParametricCurveTag)
		assert.Equal(t, uint16(4), p.FunctionType)
		assert.Len(t, p.Parameters, 7)
		assert.InDelta(t, 0.0, p.Parameters[0], 0.0001)
		assert.InDelta(t, 1.0, p.Parameters[1], 0.0001)
		assert.InDelta(t, 2.0, p.Parameters[2], 0.0001)
		assert.InDelta(t, 3.0, p.Parameters[3], 0.0001)
		assert.InDelta(t, 4.0, p.Parameters[4], 0.0001)
		assert.InDelta(t, 5.0, p.Parameters[5], 0.0001)
		assert.InDelta(t, 6.0, p.Parameters[6], 0.0001)
	})

	t.Run("TooShort", func(t *testing.T) {
		_, err := parametricCurveDecoder(make([]byte, 11), nil)
		assert.ErrorContains(t, err, "para tag too short")
	})

	t.Run("UnknownFunction", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("para")
		buf.Write([]byte{0, 0, 0, 0})
		_ = binary.Write(&buf, binary.BigEndian, uint16(5))
		for i := 0; i < 7; i++ {
			_ = binary.Write(&buf, binary.BigEndian, uint32(0x00010000)) // 1.0
		}
		_, err := parametricCurveDecoder(buf.Bytes(), nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "unknown parametric function type: 5")
	})

	t.Run("TruncatedParameters", func(t *testing.T) {
		raw := []byte("para\x00\x00\x00\x00" +
			"\x00\x02" + // function type 2 (needs 4 params)
			"\x00\x01\x00\x00\x00\x01\x00\x00\x00\x01\x00\x00") // only 3 params
		_, err := parametricCurveDecoder(raw, nil)
		assert.ErrorContains(t, err, "para tag truncated for function 2")
	})
}
