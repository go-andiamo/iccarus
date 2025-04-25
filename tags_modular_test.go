package iccarus

import (
	"bytes"
	"encoding/binary"
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
