package iccarus

import "io"

type ParseMode uint8

const (
	ParseFull ParseMode = iota
	ParseHeaderAndTagTable
	ParseHeaderOnly
)

type ParseOptions struct {
	Mode               ParseMode
	LazyTagDecode      bool
	ErrorOnUnknownTags bool
	ErrorOnTagDecode   bool // only if LazyTagDecode is false
	TagFilter          func(hdrSignature string, signature string) bool
	TagDecoders        map[string]func(raw []byte, hdrs []TagHeader) (any, error)
}

type Profile struct {
	Header         Header
	TagHeaderTable TagHeaderTable
	TagBlocks      []*Tag
}

func ParseProfile(r io.Reader, options *ParseOptions) (result Profile, err error) {
	if options == nil {
		options = &ParseOptions{}
	}
	if result.Header, err = parseHeader(r); err == nil && options.Mode < ParseHeaderOnly {
		if result.TagHeaderTable, err = parseTagHeaders(r); err == nil && options.Mode < ParseHeaderAndTagTable {
			result.TagBlocks, err = parseTags(r, result.TagHeaderTable, options)
		}
	}
	return result, err
}
