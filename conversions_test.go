package iccarus

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProfile_ToCIEXYZ_Errors(t *testing.T) {
	t.Run("Tag Not Found", func(t *testing.T) {
		p := &Profile{}
		_, err := p.ToCIEXYZ()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "A2B0 tag not found")
	})

	t.Run("Tag Not Found (nil)", func(t *testing.T) {
		p := &Profile{
			tagsByHeader: map[TagHeaderName]*Tag{
				TagHeaderAToB0: nil,
			},
		}
		_, err := p.ToCIEXYZ()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "A2B0 tag not found")
	})

	t.Run("Tag fails to decode", func(t *testing.T) {
		p := &Profile{
			tagsByHeader: map[TagHeaderName]*Tag{
				TagHeaderAToB0: {
					error: errors.New("foo"),
				},
			},
		}
		_, err := p.ToCIEXYZ()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode A2B0 tag:")
	})

	t.Run("Tag does not implement interface", func(t *testing.T) {
		p := &Profile{
			tagsByHeader: map[TagHeaderName]*Tag{
				TagHeaderAToB0: {
					value: nil,
				},
			},
		}
		_, err := p.ToCIEXYZ()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "A2B0 tag does not implement interface ToCIEXYZ")
	})

	t.Run("Tag incorrect number of channels", func(t *testing.T) {
		p := &Profile{
			tagsByHeader: map[TagHeaderName]*Tag{
				TagHeaderAToB0: {
					value: &ModularTag{InputChannels: 3},
				},
			},
		}
		_, err := p.ToCIEXYZ()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected 3 input channels, got 0")
	})
}

func TestProfile_FromCIEXYZ_Errors(t *testing.T) {
	t.Run("Tag Not Found", func(t *testing.T) {
		p := &Profile{}
		_, err := p.FromCIEXYZ()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "B2A0 tag not found")
	})

	t.Run("Tag Not Found (nil)", func(t *testing.T) {
		p := &Profile{
			tagsByHeader: map[TagHeaderName]*Tag{
				TagHeaderBToA0: nil,
			},
		}
		_, err := p.FromCIEXYZ()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "B2A0 tag not found")
	})

	t.Run("Tag fails to decode", func(t *testing.T) {
		p := &Profile{
			tagsByHeader: map[TagHeaderName]*Tag{
				TagHeaderBToA0: {
					error: errors.New("foo"),
				},
			},
		}
		_, err := p.FromCIEXYZ()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode B2A0 tag:")
	})

	t.Run("Tag does not implement interface", func(t *testing.T) {
		p := &Profile{
			tagsByHeader: map[TagHeaderName]*Tag{
				TagHeaderBToA0: {
					value: nil,
				},
			},
		}
		_, err := p.FromCIEXYZ()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "B2A0 tag does not implement interface FromCIEXYZ")
	})

	t.Run("Tag incorrect number of channels", func(t *testing.T) {
		p := &Profile{
			tagsByHeader: map[TagHeaderName]*Tag{
				TagHeaderBToA0: {
					value: &ModularTag{InputChannels: 3},
				},
			},
		}
		_, err := p.FromCIEXYZ()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected 3 input channels, got 0")
	})
}
