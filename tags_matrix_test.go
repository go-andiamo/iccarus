package iccarus

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMtxDecoder(t *testing.T) {
	t.Run("SuccessWithOffsets", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("mtx ")       // Signature
		buf.Write([]byte{0, 0, 0, 0}) // Reserved

		// Write 9 matrix values
		values := []float64{
			1.0, 0.0, 0.0,
			0.0, 1.0, 0.0,
			0.0, 0.0, 1.0,
		}
		for _, v := range values {
			buf.Write(encodeS15Fixed16BE(v))
		}

		// Write 3 offset values
		offsets := []float64{0.1, 0.2, 0.3}
		for _, v := range offsets {
			buf.Write(encodeS15Fixed16BE(v))
		}

		val, err := mtxDecoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &MatrixTag{}, val)

		mtx := val.(*MatrixTag)

		// Check matrix values
		expectedMatrix := [3][3]float64{
			{1.0, 0.0, 0.0},
			{0.0, 1.0, 0.0},
			{0.0, 0.0, 1.0},
		}
		assert.Equal(t, expectedMatrix, mtx.Matrix)

		// Check offset values
		require.NotNil(t, mtx.Offset)
		assert.InDelta(t, 0.1, (*mtx.Offset)[0], 0.0001)
		assert.InDelta(t, 0.2, (*mtx.Offset)[1], 0.0001)
		assert.InDelta(t, 0.3, (*mtx.Offset)[2], 0.0001)
	})

	t.Run("SuccessWithoutOffsets", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("mtx ")       // Signature
		buf.Write([]byte{0, 0, 0, 0}) // Reserved

		// Write only the 9 matrix values (no offsets)
		values := []float64{
			1.0, 2.0, 3.0,
			4.0, 5.0, 6.0,
			7.0, 8.0, 9.0,
		}
		for _, v := range values {
			buf.Write(encodeS15Fixed16BE(v))
		}

		val, err := mtxDecoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &MatrixTag{}, val)

		mtx := val.(*MatrixTag)

		expectedMatrix := [3][3]float64{
			{1.0, 2.0, 3.0},
			{4.0, 5.0, 6.0},
			{7.0, 8.0, 9.0},
		}
		assert.Equal(t, expectedMatrix, mtx.Matrix)
		assert.Nil(t, mtx.Offset)
	})

	t.Run("TooShort", func(t *testing.T) {
		data := make([]byte, 20)
		_, err := mtxDecoder(data, nil)
		assert.ErrorContains(t, err, "mtx tag too short")
	})
}

func TestMatrixTag_Transform(t *testing.T) {
	t.Run("SuccessWithoutOffset", func(t *testing.T) {
		matrix := &MatrixTag{
			Matrix: [3][3]float64{
				{1, 0, 0},
				{0, 1, 0},
				{0, 0, 1},
			},
			Offset: nil,
		}
		input := []float64{0.5, 0.25, 0.75}
		output, err := matrix.Transform(input)
		require.NoError(t, err)
		assert.InDeltaSlice(t, input, output, 0.0001)
	})

	t.Run("SuccessWithOffset", func(t *testing.T) {
		matrix := &MatrixTag{
			Matrix: [3][3]float64{
				{1, 0, 0},
				{0, 1, 0},
				{0, 0, 1},
			},
			Offset: &[3]float64{0.1, 0.2, 0.3},
		}
		input := []float64{0.5, 0.25, 0.75}
		expected := []float64{0.6, 0.45, 1.05} // input + offset
		output, err := matrix.Transform(input)
		require.NoError(t, err)
		assert.InDeltaSlice(t, expected, output, 0.0001)
	})

	t.Run("MatrixApplied", func(t *testing.T) {
		matrix := &MatrixTag{
			Matrix: [3][3]float64{
				{2, 0, 0},
				{0, 3, 0},
				{0, 0, 4},
			},
			Offset: nil,
		}
		input := []float64{1, 1, 1}
		expected := []float64{2, 3, 4}
		output, err := matrix.Transform(input)
		require.NoError(t, err)
		assert.InDeltaSlice(t, expected, output, 0.0001)
	})

	t.Run("WrongInputLength", func(t *testing.T) {
		matrix := &MatrixTag{
			Matrix: [3][3]float64{},
		}
		_, err := matrix.Transform([]float64{1, 2}) // only 2 inputs
		assert.ErrorContains(t, err, "matrix transform expects 3 inputs")
	})
}
