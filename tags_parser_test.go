package iccarus

import (
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTag_Value(t *testing.T) {
	tag := &Tag{
		lazy: true,
		decoder: func(raw []byte, hdrs []TagHeader) (any, error) {
			return raw, nil
		},
	}
	v, err := tag.Value()
	require.NoError(t, err)
	require.Nil(t, v)

	tag = &Tag{
		value: "foo",
	}
	v, err = tag.Value()
	require.NoError(t, err)
	require.Equal(t, "foo", v)

	tag = &Tag{
		error: errors.New("foo"),
	}
	_, err = tag.Value()
	require.Error(t, err)
}
