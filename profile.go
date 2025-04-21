package iccarus

import (
	"fmt"
	"io"
)

type ParseMode uint8

const (
	ParseFull ParseMode = iota
	ParseHeaderAndTagHeaderTable
	ParseHeaderOnly
)

// ParseOptions represents the parsing options passed to ParseProfile
type ParseOptions struct {
	// Mode determines how much of the profile to parse
	//
	// the default is ParseFull - parses everything
	//
	// the minimal is ParseHeaderOnly - useful for just listing profile header metadata
	//
	// ParseHeaderAndTagHeaderTable allows parsing the profile header and header tag table
	// without actually parsing the tags
	Mode ParseMode
	// LazyTagDecode determines whether tag values are decoded at profile parse time
	//
	// defaults to false - actual tags are decoded at profile parse time
	//
	// setting LazyTagDecode to true means that tag decoding is deferred until
	// the Tag.Value is called
	LazyTagDecode bool
	// ErrorOnUnknownTags determines whether unknown tags will cause an error on profile parse
	ErrorOnUnknownTags bool
	// ErrorOnTagDecode determines whether tag decoding errors should cause profile parse errors
	//
	// only used when LazyTagDecode is false
	ErrorOnTagDecode bool
	// TagDecoders allows you to provide custom tag decoders (or override default tag decoders)
	TagDecoders map[string]func(raw []byte, hdrs []TagHeader) (any, error)
}

// Profile represents the contents of an ICC Profile file
type Profile struct {
	// Header represents the ICC profile header (metadata)
	Header Header
	// TagHeaderTable is the tag headers
	TagHeaderTable TagHeaderTable
	// TagBlocks is the actual tag blocks (values)
	TagBlocks    []*Tag
	tagsByHeader map[TagHeaderName]*Tag
	tagsByName   map[TagName][]*Tag
}

// TagByHeader retrieves the Tag associated with a given TagHeaderName
func (p *Profile) TagByHeader(name TagHeaderName) (result *Tag, ok bool) {
	result, ok = p.tagsByHeader[name]
	return result, ok
}

// TagsByName retrieves all the actual tags of a specified TagName
//
// added for utility - often better to use Profile.TagByHeader
func (p *Profile) TagsByName(name TagName) (result []*Tag, ok bool) {
	result, ok = p.tagsByName[name]
	return result, ok
}

// TagValue retrieves an actual tag value by TagHeaderName
//
// Even if the ParseOptions.LazyTagDecode was set to true, the actual tag value
// will be decoded (once) on calling this
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

// ParseProfile parses an ICC colour profile from the supplied reader with the supplied ParseOptions
//
// if the ParseOptions supplied is nil, default (full) options are used
func ParseProfile(r io.Reader, options *ParseOptions) (result *Profile, err error) {
	if options == nil {
		options = &ParseOptions{
			Mode: ParseFull,
		}
	}
	result = &Profile{}
	if result.Header, err = parseHeader(r); err == nil && options.Mode < ParseHeaderOnly {
		if result.TagHeaderTable, err = parseTagHeaders(r); err == nil && options.Mode < ParseHeaderAndTagHeaderTable {
			result.TagBlocks, err = parseTags(r, result.TagHeaderTable, options)
			result.mapTags()
		}
	}
	return result, err
}
