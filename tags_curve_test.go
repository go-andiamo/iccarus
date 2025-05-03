package iccarus

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

func TestCurveDecoder(t *testing.T) {
	t.Run("IdentityCurve", func(t *testing.T) {
		raw := []byte("curv\x00\x00\x00\x00" + // sig + reserved
			"\x00\x00\x00\x00") // count = 0
		val, err := curveDecoder(raw)
		require.NoError(t, err)
		require.IsType(t, &CurveTag{}, val)
		c := val.(*CurveTag)
		assert.Equal(t, CurveTypeIdentity, c.Type)
	})

	t.Run("GammaCurve", func(t *testing.T) {
		raw := []byte("curv\x00\x00\x00\x00" + // sig + reserved
			"\x00\x00\x00\x01" + // count = 1
			"\x01\x00") // gamma = 1.0
		val, err := curveDecoder(raw)
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
		val, err := curveDecoder(raw)
		require.NoError(t, err)
		require.IsType(t, &CurveTag{}, val)
		c := val.(*CurveTag)
		assert.Equal(t, CurveTypePoints, c.Type)
		assert.Equal(t, []uint16{0x10, 0x20, 0x30}, c.Points)
	})

	t.Run("TooShort", func(t *testing.T) {
		_, err := curveDecoder(make([]byte, 11))
		assert.ErrorContains(t, err, "curv tag too short")
	})

	t.Run("MissingGamma", func(t *testing.T) {
		raw := []byte("curv\x00\x00\x00\x00" +
			"\x00\x00\x00\x01") // count = 1 (but no gamma value)
		_, err := curveDecoder(raw)
		assert.ErrorContains(t, err, "curv tag missing gamma value")
	})

	t.Run("TruncatedPoints", func(t *testing.T) {
		raw := []byte("curv\x00\x00\x00\x00" +
			"\x00\x00\x00\x02" + // count = 2
			"\x00\x10") // missing second uint16
		_, err := curveDecoder(raw)
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

		val, err := parametricCurveDecoder(buf.Bytes())
		require.NoError(t, err)
		require.IsType(t, &ParametricCurveTag{}, val)
		p := val.(*ParametricCurveTag)
		assert.Equal(t, ParametricCurveFunction(0), p.FunctionType)
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
		val, err := parametricCurveDecoder(buf.Bytes())
		require.NoError(t, err)
		require.IsType(t, &ParametricCurveTag{}, val)
		p := val.(*ParametricCurveTag)
		assert.Equal(t, ParametricCurveFunction(1), p.FunctionType)
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
		val, err := parametricCurveDecoder(buf.Bytes())
		require.NoError(t, err)
		require.IsType(t, &ParametricCurveTag{}, val)
		p := val.(*ParametricCurveTag)
		assert.Equal(t, ParametricCurveFunction(2), p.FunctionType)
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
		val, err := parametricCurveDecoder(buf.Bytes())
		require.NoError(t, err)
		require.IsType(t, &ParametricCurveTag{}, val)
		p := val.(*ParametricCurveTag)
		assert.Equal(t, ParametricCurveFunction(3), p.FunctionType)
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
		val, err := parametricCurveDecoder(buf.Bytes())
		require.NoError(t, err)
		require.IsType(t, &ParametricCurveTag{}, val)
		p := val.(*ParametricCurveTag)
		assert.Equal(t, ParametricCurveFunction(4), p.FunctionType)
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
		_, err := parametricCurveDecoder(make([]byte, 11))
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
		_, err := parametricCurveDecoder(buf.Bytes())
		require.Error(t, err)
		assert.ErrorContains(t, err, "unknown parametric function type: 5")
	})

	t.Run("TruncatedParameters", func(t *testing.T) {
		raw := []byte("para\x00\x00\x00\x00" +
			"\x00\x02" + // function type 2 (needs 4 params)
			"\x00\x01\x00\x00\x00\x01\x00\x00\x00\x01\x00\x00") // only 3 params
		_, err := parametricCurveDecoder(raw)
		assert.ErrorContains(t, err, "para tag truncated for function 2")
	})
}

func TestCurveTag_Transform(t *testing.T) {
	t.Run("IdentityCurve", func(t *testing.T) {
		c := &CurveTag{Type: CurveTypeIdentity}
		out, err := c.Transform([]float64{0.5})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.InDelta(t, 0.5, out[0], 0.0001)
	})

	t.Run("GammaCurve", func(t *testing.T) {
		c := &CurveTag{Type: CurveTypeGamma, Gamma: 2.0}
		out, err := c.Transform([]float64{0.5})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.InDelta(t, 0.25, out[0], 0.0001) // 0.5^2 = 0.25
	})

	t.Run("PointsCurve_ExactIndex", func(t *testing.T) {
		c := &CurveTag{Type: CurveTypePoints, Points: []uint16{0, 32768, 65535}}
		out, err := c.Transform([]float64{0.5})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.InDelta(t, 0.5, out[0], 0.01) // halfway
	})

	t.Run("PointsCurve_Interpolation", func(t *testing.T) {
		c := &CurveTag{Type: CurveTypePoints, Points: []uint16{0, 32768, 65535}}
		out, err := c.Transform([]float64{0.25})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.InDelta(t, 0.25, out[0], 0.01) // quarter
	})

	t.Run("UnknownCurveType", func(t *testing.T) {
		c := &CurveTag{Type: 999}
		_, err := c.Transform([]float64{0.5})
		assert.ErrorContains(t, err, "unknown curve type")
	})

	t.Run("WrongInputLength", func(t *testing.T) {
		c := &CurveTag{Type: CurveTypeIdentity}
		_, err := c.Transform([]float64{0.1, 0.2})
		assert.ErrorContains(t, err, "curve expects 1 input")
	})

	t.Run("EmptyPoints", func(t *testing.T) {
		c := &CurveTag{Type: CurveTypePoints, Points: []uint16{}}
		_, err := c.Transform([]float64{0.5})
		assert.ErrorContains(t, err, "curve has no points")
	})
}

func TestParametricCurveTag_Transform(t *testing.T) {
	t.Run("SimpleGammaFunction", func(t *testing.T) {
		curve := &ParametricCurveTag{
			FunctionType: SimpleGammaFunction,
			Parameters:   []float64{2.0}, // Y = X^2
		}
		out, err := curve.Transform([]float64{0.5})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.InDelta(t, 0.25, out[0], 0.001)
	})

	t.Run("SimpleGammaFunction_InvalidParamCount", func(t *testing.T) {
		p := &ParametricCurveTag{
			FunctionType: SimpleGammaFunction,
			Parameters:   []float64{}, // Empty! Should trigger error
		}
		_, err := p.Transform([]float64{0.5})
		assert.ErrorContains(t, err, "function 0 expects 1 parameter")
	})

	t.Run("ConditionalZeroFunction", func(t *testing.T) {
		curve := &ParametricCurveTag{
			FunctionType: ConditionalZeroFunction,
			Parameters:   []float64{1.0, 0.0, 2.0}, // Y = (X)^2 if X>=0 else 0
		}
		out, err := curve.Transform([]float64{-0.5})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.InDelta(t, 0.0, out[0], 0.001)
	})

	t.Run("ConditionalZeroFunction_PositiveBranch", func(t *testing.T) {
		p := &ParametricCurveTag{
			FunctionType: ConditionalZeroFunction,
			Parameters:   []float64{1.0, 0.0, 2.0}, // a=1.0, b=0.0, g=2.0
		}
		out, err := p.Transform([]float64{0.5}) // 0.5 >= -b/a → 0.5 >= 0 → true
		require.NoError(t, err)
		assert.InDelta(t, 0.25, out[0], 0.001) // (1*0.5+0)^2 = 0.25
	})

	t.Run("ConditionalZeroFunction_InvalidParamCount", func(t *testing.T) {
		p := &ParametricCurveTag{
			FunctionType: ConditionalZeroFunction,
			Parameters:   []float64{1.0, 2.0}, // Only 2 instead of 3
		}
		_, err := p.Transform([]float64{0.5})
		assert.ErrorContains(t, err, "function 1 expects 3 parameters")
	})

	t.Run("ConditionalCFunction", func(t *testing.T) {
		curve := &ParametricCurveTag{
			FunctionType: ConditionalCFunction,
			Parameters:   []float64{1.0, 0.0, 2.0, 0.1}, // Y = (X)^2+0.1 if X>=0 else 0.1
		}
		out, err := curve.Transform([]float64{-0.5})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.InDelta(t, 0.1, out[0], 0.001)
	})

	t.Run("ConditionalCFunction_PositiveBranch", func(t *testing.T) {
		p := &ParametricCurveTag{
			FunctionType: ConditionalCFunction,
			Parameters:   []float64{1.0, 0.0, 2.0, 0.1}, // a=1.0, b=0.0, g=2.0, c=0.1
		}
		out, err := p.Transform([]float64{0.5}) // 0.5 >= -b/a → 0.5 >= 0 → true
		require.NoError(t, err)
		assert.InDelta(t, 0.35, out[0], 0.001) // (1*0.5+0)^2 + 0.1 = 0.25 + 0.1 = 0.35
	})

	t.Run("ConditionalCFunction_InvalidParamCount", func(t *testing.T) {
		p := &ParametricCurveTag{
			FunctionType: ConditionalCFunction,
			Parameters:   []float64{1.0, 2.0, 3.0}, // Only 3 instead of 4
		}
		_, err := p.Transform([]float64{0.5})
		assert.ErrorContains(t, err, "function 2 expects 4 parameters")
	})

	t.Run("SplitFunction", func(t *testing.T) {
		curve := &ParametricCurveTag{
			FunctionType: SplitFunction,
			Parameters:   []float64{1.0, 0.0, 2.0, 2.0, 0.5}, // switch at 0.5
		}
		out, err := curve.Transform([]float64{0.4})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.InDelta(t, 0.8, out[0], 0.001) // 2 * 0.4
	})

	t.Run("SplitFunction_PositiveBranch", func(t *testing.T) {
		p := &ParametricCurveTag{
			FunctionType: SplitFunction,
			Parameters:   []float64{1.0, 0.0, 2.0, 0.5, 0.4}, // a=1.0, b=0.0, g=2.0, c=0.5, d=0.4
		}
		out, err := p.Transform([]float64{0.5}) // 0.5 >= 0.4 → true
		require.NoError(t, err)
		assert.InDelta(t, 0.25, out[0], 0.001) // (1*0.5+0)^2 = 0.25
	})

	t.Run("SplitFunction_InvalidParamCount", func(t *testing.T) {
		curve := &ParametricCurveTag{
			FunctionType: SplitFunction,
			Parameters:   []float64{1.0, 0.0, 2.0}, // missing params
		}
		_, err := curve.Transform([]float64{0.5})
		assert.ErrorContains(t, err, "function 3 expects 5 parameters")
	})

	t.Run("ComplexFunction", func(t *testing.T) {
		curve := &ParametricCurveTag{
			FunctionType: ComplexFunction,
			Parameters:   []float64{1.0, 0.0, 2.0, 2.0, 0.5, 0.1, 0.2}, // split at 0.5
		}
		out, err := curve.Transform([]float64{0.6})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.InDelta(t, math.Pow(0.6, 2)+0.1, out[0], 0.001)
	})

	t.Run("ComplexFunction_NegativeBranch", func(t *testing.T) {
		p := &ParametricCurveTag{
			FunctionType: ComplexFunction,
			Parameters:   []float64{1.0, 0.0, 2.0, 0.5, 0.6, 0.1, 0.2}, // a=1, b=0, g=2, c=0.5, d=0.6, e=0.1, f=0.2
		}
		out, err := p.Transform([]float64{0.5}) // 0.5 < 0.6 → false branch
		require.NoError(t, err)
		assert.InDelta(t, 0.45, out[0], 0.001) // 0.5*0.5 + 0.2 = 0.45
	})

	t.Run("ComplexFunction_InvalidParamCount", func(t *testing.T) {
		p := &ParametricCurveTag{
			FunctionType: ComplexFunction,
			Parameters:   []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}, // Only 6 instead of 7
		}
		_, err := p.Transform([]float64{0.5})
		assert.ErrorContains(t, err, "function 4 expects 7 parameters")
	})

	t.Run("WrongInputLength", func(t *testing.T) {
		curve := &ParametricCurveTag{FunctionType: 0, Parameters: []float64{2.0}}
		_, err := curve.Transform([]float64{0.5, 0.5})
		assert.ErrorContains(t, err, "expects 1 input")
	})

	t.Run("UnknownFunctionType", func(t *testing.T) {
		curve := &ParametricCurveTag{
			FunctionType: 99,
			Parameters:   []float64{},
		}
		_, err := curve.Transform([]float64{0.5})
		assert.ErrorContains(t, err, "unknown parametric function type")
	})
}
