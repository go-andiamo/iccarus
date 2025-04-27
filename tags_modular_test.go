package iccarus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestModularDecoder(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("mAB ")       // Signature (4 bytes)
		buf.Write([]byte{0, 0, 0, 0}) // Reserved (4 bytes)

		// Input channels = 2, Output channels = 3
		_ = binary.Write(&buf, binary.BigEndian, uint16(2)) // input
		_ = binary.Write(&buf, binary.BigEndian, uint16(3)) // output

		// Fake offset table (one element starting at byte 20)
		_ = binary.Write(&buf, binary.BigEndian, uint32(20))

		// Fill with dummy bytes to reach offset 20
		for buf.Len() < 20 {
			buf.WriteByte(0)
		}

		// Embedded tag block: "curv" + reserved + data
		buf.WriteString("curv")
		buf.Write([]byte{0, 0, 0, 0}) // Reserved
		buf.Write([]byte{0, 0, 0, 1}) // One curve point (1)

		val, err := modularDecoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &ModularTag{}, val)

		tag := val.(*ModularTag)
		assert.Equal(t, "mAB", tag.Signature)
		assert.Equal(t, uint8(2), tag.InputChannels)
		assert.Equal(t, uint8(3), tag.OutputChannels)
		assert.Len(t, tag.Elements, 1)
	})

	t.Run("HappyPathSingleElementNoOffsets", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("mBA ")       // Signature (4 bytes)
		buf.Write([]byte{0, 0, 0, 0}) // Reserved

		_ = binary.Write(&buf, binary.BigEndian, uint16(1)) // input
		_ = binary.Write(&buf, binary.BigEndian, uint16(1)) // output

		// No offset table, straight into embedded tag
		buf.WriteString("curv")
		buf.Write([]byte{0, 0, 0, 0})
		buf.Write([]byte{0, 0, 0, 1})

		val, err := modularDecoder(buf.Bytes(), nil)
		require.NoError(t, err)
		require.IsType(t, &ModularTag{}, val)

		tag := val.(*ModularTag)
		assert.Equal(t, "mBA", tag.Signature)
		assert.Equal(t, uint8(1), tag.InputChannels)
		assert.Equal(t, uint8(1), tag.OutputChannels)
		assert.Len(t, tag.Elements, 1)
	})

	t.Run("TooShort", func(t *testing.T) {
		data := []byte{1, 2, 3}
		_, err := modularDecoder(data, nil)
		assert.ErrorContains(t, err, "modular (mAB/mBA) tag too short")
	})

	t.Run("ElementTooShort", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("mAB ")
		buf.Write([]byte{0, 0, 0, 0})
		_ = binary.Write(&buf, binary.BigEndian, uint16(1))
		_ = binary.Write(&buf, binary.BigEndian, uint16(1))

		// No offset table, but embedded tag is less than 8 bytes
		buf.Write([]byte("bad"))

		_, err := modularDecoder(buf.Bytes(), nil)
		assert.ErrorContains(t, err, "modular (mAB/mBA): element 0 too short to contain header")
	})
}

func TestIsASCII(t *testing.T) {
	assert.True(t, isASCII([]byte("foob`")))
	assert.False(t, isASCII([]byte{0, 0, 0, 0}))
}

func TestModularTag_transformChannels(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mod := &ModularTag{
			InputChannels:  3,
			OutputChannels: 3,
			Elements: []*Tag{
				{
					Signature: "curv",
					value:     &mockTransformer{offset: 1.0},
				},
				{
					Signature: "curv",
					value:     &mockTransformer{offset: 2.0},
				},
			},
		}

		result, err := mod.transformChannels([]float64{0.1, 0.2, 0.3})
		require.NoError(t, err)
		require.Len(t, result, 3)

		assert.InDelta(t, 0.1+1.0+2.0, result[0], 0.001)
		assert.InDelta(t, 0.2+1.0+2.0, result[1], 0.001)
		assert.InDelta(t, 0.3+1.0+2.0, result[2], 0.001)
	})

	t.Run("WrongChannelCount", func(t *testing.T) {
		mod := &ModularTag{InputChannels: 3}
		_, err := mod.transformChannels([]float64{0.1, 0.2}) // only 2 channels
		assert.ErrorContains(t, err, "expected 3 input channels")
	})

	t.Run("NoTransformers", func(t *testing.T) {
		mod := &ModularTag{
			InputChannels: 3,
			Elements: []*Tag{
				{
					Signature: "noop", // value is nil
				},
			},
		}
		_, err := mod.transformChannels([]float64{0.1, 0.2, 0.3})
		assert.ErrorContains(t, err, "no transformable elements")
	})

	t.Run("DecodeFails", func(t *testing.T) {
		mod := &ModularTag{
			InputChannels: 3,
			Elements: []*Tag{
				{
					Signature: TagCurve,
					error:     errors.New("decode failed"),
				},
			},
		}
		_, err := mod.transformChannels([]float64{0.1, 0.2, 0.3})
		assert.ErrorContains(t, err, "failed decoding modular element")
	})

	t.Run("TransformerFails", func(t *testing.T) {
		mod := &ModularTag{
			InputChannels: 3,
			Elements: []*Tag{
				{
					Signature: TagCurve,
					value:     &mockFailTransformer{},
				},
			},
		}
		_, err := mod.transformChannels([]float64{0.1, 0.2, 0.3})
		assert.ErrorContains(t, err, "failed processing modular element")
	})
}

type mockTransformer struct {
	offset float64
}

func (m *mockTransformer) Transform(channels []float64) ([]float64, error) {
	result := make([]float64, len(channels))
	for i, v := range channels {
		result[i] = v + m.offset
	}
	return result, nil
}

type mockFailTransformer struct{}

func (m *mockFailTransformer) Transform(_ []float64) ([]float64, error) {
	return nil, errors.New("mock transform failure")
}
