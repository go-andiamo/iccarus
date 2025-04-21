package iccarus

import (
	"fmt"
	"io"
)

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
	tagsByHeader   map[TagHeaderName]*Tag
	tagsByName     map[TagName][]*Tag
}

func (p *Profile) TagByHeader(name TagHeaderName) (result *Tag, ok bool) {
	result, ok = p.tagsByHeader[name]
	return result, ok
}

func (p *Profile) TagsByName(name TagName) (result []*Tag, ok bool) {
	result, ok = p.tagsByName[name]
	return result, ok
}

func (p *Profile) TagValue(name TagHeaderName) (any, error) {
	if tag, ok := p.tagsByHeader[name]; ok {
		return tag.Value()
	}
	return nil, fmt.Errorf("tag %q not found", name)
}

func (p *Profile) mapTags() {
	p.tagsByHeader = make(map[TagHeaderName]*Tag)
	p.tagsByName = make(map[TagName][]*Tag)
	for _, tag := range p.TagBlocks {
		p.tagsByName[TagName(tag.Signature)] = append(p.tagsByName[TagName(tag.Signature)], tag)
		for _, hdr := range tag.Headers {
			p.tagsByHeader[TagHeaderName(hdr.Signature)] = tag
		}
	}
}

func ParseProfile(r io.Reader, options *ParseOptions) (result *Profile, err error) {
	if options == nil {
		options = &ParseOptions{}
	}
	result = &Profile{}
	if result.Header, err = parseHeader(r); err == nil && options.Mode < ParseHeaderOnly {
		if result.TagHeaderTable, err = parseTagHeaders(r); err == nil && options.Mode < ParseHeaderAndTagTable {
			result.TagBlocks, err = parseTags(r, result.TagHeaderTable, options)
			result.mapTags()
		}
	}
	return result, err
}
