package iccarus

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseTagHeaders(t *testing.T) {
	// 2 tags: 'desc' at offset 128, size 48; 'rXYZ' at offset 176, size 20
	buf := append(
		[]byte{0, 0, 0, 2}, // count = 2
		[]byte{
			'd', 'e', 's', 'c', 0, 0, 0, 128, 0, 0, 0, 48,
			'X', 'Y', 'Z', 0, 0, 0, 0, 176, 0, 0, 0, 20,
		}...,
	)
	r := bytes.NewReader(buf)
	tagTable, err := parseTagHeaders(r)
	require.NoError(t, err)
	assert.Len(t, tagTable.Entries, 2)
	assert.Equal(t, "desc", tagTable.Entries[0].Name)
	assert.Equal(t, "XYZ", tagTable.Entries[1].Name)
	assert.Equal(t, uint32(0x80), tagTable.Entries[0].Offset)
	assert.Equal(t, uint32(0xb0), tagTable.Entries[1].Offset)
	assert.Equal(t, uint32(0x30), tagTable.Entries[0].Size)
	assert.Equal(t, uint32(0x14), tagTable.Entries[1].Size)
}

func TestParseTagHeaders_Errors(t *testing.T) {
	r := bytes.NewReader([]byte{})
	_, err := parseTagHeaders(r)
	require.Error(t, err)

	r = bytes.NewReader([]byte{0, 0, 0, 1})
	_, err = parseTagHeaders(r)
	require.Error(t, err)

	r = bytes.NewReader([]byte{0, 1, 0, 1})
	_, err = parseTagHeaders(r)
	require.Error(t, err)
	require.Equal(t, "tag count 65537 exceeds max allowed (1024)", err.Error())
}
