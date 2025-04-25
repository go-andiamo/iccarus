package iccarus

import (
	"bytes"
	"github.com/go-andiamo/iccarus/_test_data/profiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestParseHeader(t *testing.T) {
	names := profiles.List()
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			f, err := profiles.Open(name)
			require.NoError(t, err)
			defer func() {
				_ = f.Close()
			}()
			hdr, err := parseHeader(f)
			require.NoError(t, err)
			assert.NotNil(t, hdr)
			if name == "default/ISOcoated_v2_300_eci.icc" {
				assert.Equal(t, uint32(0x1be8e5), hdr.ProfileSize)
				assert.Equal(t, "acsp", hdr.Signature)
				assert.Equal(t, "HDM", hdr.CMMType)
				assert.Equal(t, uint32(0x2400000), hdr.VersionRaw)
				assert.Equal(t, 2, hdr.Version.Major)
				assert.Equal(t, 4, hdr.Version.Minor)
				assert.Equal(t, 0, hdr.Version.Revision)
				assert.Equal(t, "prtr", hdr.DeviceClass)
				assert.Equal(t, "CMYK", hdr.ColorSpace)
				assert.Equal(t, "Lab", hdr.PCS)
				assert.Equal(t, "2007-02-28T08:00:00Z", hdr.Created.Format(time.RFC3339))
				assert.Equal(t, "", hdr.Platform)
				assert.Equal(t, uint32(0), hdr.Flags)
				assert.Equal(t, "", hdr.Manufacturer)
				assert.Equal(t, "", hdr.Model)
				assert.Equal(t, [8]byte{}, hdr.Attributes)
				assert.Equal(t, uint32(0), hdr.RenderingIntent)
				assert.InDelta(t, 0.964202880859375, hdr.Illuminant[0], 0.001)
				assert.InDelta(t, 1.0, hdr.Illuminant[1], 0.001)
				assert.InDelta(t, 0.8249053955078125, hdr.Illuminant[2], 0.001)
				assert.Equal(t, "HDM", hdr.Creator)
			}
		})
	}
}

func TestParseICCHeader_Errors(t *testing.T) {
	r := strings.NewReader("not an ICC header")
	_, err := parseHeader(r)
	assert.Error(t, err)

	r2 := bytes.NewReader(make([]byte, 128))
	_, err = parseHeader(r2)
	assert.Error(t, err)
}

func TestVersion_String(t *testing.T) {
	v := Version{
		Major:    1,
		Minor:    2,
		Revision: 3,
	}
	assert.Equal(t, "1.2.3", v.String())
}

func TestStringed(t *testing.T) {
	s := stringed([]byte{0, 0, 0, 0})
	assert.Equal(t, "", s)
	s = stringed([]byte{'a', ' ', 0, 0})
	assert.Equal(t, "a", s)
	s = stringed([]byte{'a', 255, 0, 0})
	assert.Equal(t, "0x61FF0000", s)
}
