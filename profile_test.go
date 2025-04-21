package iccarus

import (
	"github.com/go-andiamo/iccarus/_test_data/profiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestParseProfile(t *testing.T) {
	names := profiles.List()
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			f, err := profiles.Open(name)
			require.NoError(t, err)
			defer func() {
				_ = f.Close()
			}()
			p, err := ParseProfile(f, nil)
			require.NoError(t, err)
			require.NotNil(t, p)
			if name == "default/ISOcoated_v2_300_eci.icc" {
				hdr := p.Header
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
				assert.Equal(t, [3]Fixed1616{0xf6d6, 0x10000, 0xd32d}, hdr.Illuminant)
				assert.Equal(t, "HDM", hdr.Creator)

				assert.Equal(t, 13, len(p.TagHeaderTable.Entries))
				assert.Equal(t, 13, len(p.TagBlocks))
				tag, ok := p.TagByHeader(TagHeaderCopyright)
				require.True(t, ok)
				assert.Equal(t, string(TagText), tag.Signature)
				val, err := p.TagValue(TagHeaderCopyright)
				require.NoError(t, err)
				assert.IsType(t, "", val)
				_, err = p.TagValue(TagHeaderName("foo"))
				assert.Error(t, err)
				tags, ok := p.TagsByName(TagText)
				require.True(t, ok)
				assert.Equal(t, 3, len(tags))
			}
		})
	}
}
