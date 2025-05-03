package iccarus

import (
	"errors"
	"github.com/go-andiamo/iccarus/_test_data/profiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestParseTags(t *testing.T) {
	f, err := profiles.Open("default/ISOcoated_v2_300_eci.icc")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()
	_, err = parseHeader(f)
	require.NoError(t, err)
	tagHeaders, err := parseTagHeaders(f)
	require.NoError(t, err)
	tags, err := parseTags(f, tagHeaders, &ParseOptions{})
	require.NoError(t, err)
	assert.Equal(t, 13, len(tags))
}

func TestParseTags_Errors_RecordUnknownTag(t *testing.T) {
	f, err := profiles.Open("default/ISOcoated_v2_300_eci.icc")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()
	_, err = parseHeader(f)
	require.NoError(t, err)
	tagHeaders, err := parseTagHeaders(f)
	require.NoError(t, err)
	tags, err := parseTags(f, tagHeaders, &ParseOptions{
		// simulate unknown tag...
		TagDecoders: map[string]func(raw []byte) (any, error){
			"text": nil,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 13, len(tags))
	require.Error(t, tags[0].error)
	assert.ErrorContains(t, tags[0].error, "unknown tag")
}

func TestParseTags_Errors_UnknownTag(t *testing.T) {
	f, err := profiles.Open("default/ISOcoated_v2_300_eci.icc")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()
	_, err = parseHeader(f)
	require.NoError(t, err)
	tagHeaders, err := parseTagHeaders(f)
	require.NoError(t, err)
	_, err = parseTags(f, tagHeaders, &ParseOptions{
		// simulate unknown tag...
		ErrorOnUnknownTags: true,
		TagDecoders: map[string]func(raw []byte) (any, error){
			"text": nil,
		},
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown tag")
}

func TestParseTags_Errors_DecodeTag(t *testing.T) {
	f, err := profiles.Open("default/ISOcoated_v2_300_eci.icc")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()
	_, err = parseHeader(f)
	require.NoError(t, err)
	tagHeaders, err := parseTagHeaders(f)
	require.NoError(t, err)
	_, err = parseTags(f, tagHeaders, &ParseOptions{
		ErrorOnTagDecode: true,
		TagDecoders: map[string]func(raw []byte) (any, error){
			"text": func(raw []byte) (any, error) {
				return nil, errors.New("failed to decode tag")
			},
		},
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to decode tag")
}

func TestParseTags_Errors_RecordDecodeTag(t *testing.T) {
	f, err := profiles.Open("default/ISOcoated_v2_300_eci.icc")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()
	_, err = parseHeader(f)
	require.NoError(t, err)
	tagHeaders, err := parseTagHeaders(f)
	require.NoError(t, err)
	tags, err := parseTags(f, tagHeaders, &ParseOptions{
		TagDecoders: map[string]func(raw []byte) (any, error){
			"text": func(raw []byte) (any, error) {
				return nil, errors.New("failed to decode tag")
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 13, len(tags))
	require.Error(t, tags[0].error)
	assert.ErrorContains(t, tags[0].error, "failed to decode tag")
}

func TestParseTags_Errors_ReadError(t *testing.T) {
	f, err := profiles.Open("default/ISOcoated_v2_300_eci.icc")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()
	_, err = parseHeader(f)
	require.NoError(t, err)
	tagHeaders, err := parseTagHeaders(f)
	require.NoError(t, err)
	_, err = parseTags(f, tagHeaders, &ParseOptions{
		// simulate read error by reading from file...
		TagDecoders: map[string]func(raw []byte) (any, error){
			"text": func(raw []byte) (any, error) {
				_, _ = io.CopyN(io.Discard, f, 1024)
				return nil, nil
			},
		},
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read tag")
}

func TestParseTags_Errors_BackwardSeek(t *testing.T) {
	f, err := profiles.Open("default/ISOcoated_v2_300_eci.icc")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()
	_, err = parseHeader(f)
	require.NoError(t, err)
	tagHeaders, err := parseTagHeaders(f)
	require.NoError(t, err)
	// simulate backwards...
	tagHeaders.Entries[0].Offset = 2048
	_, err = parseTags(f, tagHeaders, &ParseOptions{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "before current stream position")
}

func TestParseTags_Errors_SkipError(t *testing.T) {
	f, err := profiles.Open("default/ISOcoated_v2_300_eci.icc")
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()
	_, err = parseHeader(f)
	require.NoError(t, err)
	tagHeaders, err := parseTagHeaders(f)
	require.NoError(t, err)
	// simulate skip too far...
	tagHeaders.Entries[12].Offset = tagHeaders.Entries[12].Offset * 2
	_, err = parseTags(f, tagHeaders, &ParseOptions{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to skip")
}

func TestTag_Value(t *testing.T) {
	tag := &Tag{
		lazy: true,
		decoder: func(raw []byte) (any, error) {
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
