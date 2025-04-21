package iccarus

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDictDecoder(t *testing.T) {
	result, err := dictDecoder([]byte("foo"), nil)
	require.NoError(t, err)
	require.IsType(t, []byte{}, result)
	require.Len(t, result, 3)
}

func TestPsidDecoder(t *testing.T) {
	result, err := psidDecoder([]byte("foo"), nil)
	require.NoError(t, err)
	require.IsType(t, []byte{}, result)
	require.Len(t, result, 3)
}

func TestPseqDecoder(t *testing.T) {
	result, err := pseqDecoder([]byte("foo"), nil)
	require.NoError(t, err)
	require.IsType(t, []byte{}, result)
	require.Len(t, result, 3)
}

func TestGbdDecoder(t *testing.T) {
	result, err := gbdDecoder([]byte("foo"), nil)
	require.NoError(t, err)
	require.IsType(t, []byte{}, result)
	require.Len(t, result, 3)
}

func TestZxmlDecoder(t *testing.T) {
	result, err := zxmlDecoder([]byte("foo"), nil)
	require.NoError(t, err)
	require.IsType(t, []byte{}, result)
	require.Len(t, result, 3)
}

func TestMsbnDecoder(t *testing.T) {
	result, err := msbnDecoder([]byte("foo"), nil)
	require.NoError(t, err)
	require.IsType(t, []byte{}, result)
	require.Len(t, result, 3)
}
