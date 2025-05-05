package iccarus

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMeasurementDecoder(t *testing.T) {
	t.Run("ValidMeasurement", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("meas")                             // tag sig
		buf.Write([]byte{0, 0, 0, 0})                       // reserved
		_ = binary.Write(&buf, binary.BigEndian, uint32(1)) // Observer
		buf.Write(encodeS15Fixed16BE(1.0))                  // Backing.X = 1.0
		buf.Write(encodeS15Fixed16BE(0.5))                  // Backing.Y = 0.5
		buf.Write(encodeS15Fixed16BE(0.25))                 // Backing.Z = 0.25
		_ = binary.Write(&buf, binary.BigEndian, uint32(2)) // Geometry
		buf.Write(encodeS15Fixed16BE(0.125))                // Flare = 0.125
		_ = binary.Write(&buf, binary.BigEndian, uint32(3)) // Illuminant
		val, err := measurementDecoder(buf.Bytes())
		assert.NoError(t, err)
		tag := val.(*MeasurementTag)
		assert.Equal(t, uint32(1), tag.Observer)
		assert.InDelta(t, 1.0, tag.Backing.X, 0.00001)
		assert.InDelta(t, 0.5, tag.Backing.Y, 0.00001)
		assert.InDelta(t, 0.25, tag.Backing.Z, 0.00001)
		assert.Equal(t, uint32(2), tag.Geometry)
		assert.InDelta(t, 0.125, tag.Flare, 0.00001)
		assert.Equal(t, uint32(3), tag.Illuminant)
	})
	t.Run("TooShort", func(t *testing.T) {
		data := make([]byte, 35) // < 36
		_, err := measurementDecoder(data)
		assert.ErrorContains(t, err, "meas tag too short")
	})
}
