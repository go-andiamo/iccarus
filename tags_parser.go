package iccarus

import (
	"fmt"
	"io"
	"slices"
)

type Tag struct {
	Headers   []TagHeader
	Signature string // e.g. "desc", "XYZ"
	Raw       []byte
	value     any
	lazy      bool
	error     error
	decoder   func(raw []byte, hdrs []TagHeader) (any, error)
}

func (t *Tag) Value() (any, error) {
	if t.error != nil {
		return nil, t.error
	}
	if t.lazy {
		t.value, t.error = t.decoder(t.Raw, t.Headers)
		t.lazy = false
	}
	return t.value, t.error
}

func parseTags(r io.Reader, table TagHeaderTable, options *ParseOptions) ([]*Tag, error) {
	// sort headers by offset...
	headers := make([]TagHeader, len(table.Entries))
	copy(headers, table.Entries)
	slices.SortFunc(headers, func(a, b TagHeader) int {
		return int(a.Offset) - int(b.Offset)
	})
	// current offset is past header (128 bytes), tag header count (4 bytes) and tag headers (12 bytes each)...
	currentOffset := 128 + 4 + (12 * len(headers))
	// a cache of offsets - for tag block sharing...
	offsetCache := make(map[uint32]*Tag)
	result := make([]*Tag, 0, len(table.Entries))
	for _, hdr := range headers {
		// use cached block if available...
		if block, ok := offsetCache[hdr.Offset]; ok {
			block.Headers = append(block.Headers, hdr)
			result = append(result, block)
			continue
		}
		// skip ahead if needed...
		if hdr.Offset > uint32(currentOffset) {
			skip := int64(hdr.Offset - uint32(currentOffset))
			if _, err := io.CopyN(io.Discard, r, skip); err != nil {
				return nil, fmt.Errorf("failed to skip to tag %q at 0x%X: %w", hdr.Signature, hdr.Offset, err)
			}
			currentOffset = int(hdr.Offset)
		}
		if hdr.Offset < uint32(currentOffset) {
			return nil, fmt.Errorf("tag %q has offset 0x%X before current stream position 0x%X", hdr.Signature, hdr.Offset, currentOffset)
		}
		// read the tag data...
		raw := make([]byte, hdr.Size)
		if _, err := io.ReadFull(r, raw); err != nil {
			return nil, fmt.Errorf("failed to read tag %q at 0x%X: %w", hdr.Signature, hdr.Offset, err)
		}
		currentOffset += int(hdr.Size)
		signature := stringed(raw[0:4]) // first 4 bytes of tag block are the tag type
		block := &Tag{
			Headers:   []TagHeader{hdr},
			Signature: signature,
			Raw:       raw,
			lazy:      options.LazyTagDecode,
			decoder:   defaultDecoders[signature],
		}
		if decoder, ok := options.TagDecoders[signature]; ok {
			block.decoder = decoder
		}
		if block.decoder == nil {
			if options.ErrorOnUnknownTags {
				return nil, fmt.Errorf("unknown tag %q at 0x%X", signature, hdr.Offset)
			}
			block.error = fmt.Errorf("unknown tag %q", signature)
			offsetCache[hdr.Offset] = block
			result = append(result, block)
			continue
		}
		if !options.LazyTagDecode && block.error == nil {
			block.value, block.error = block.decoder(block.Raw, block.Headers)
		}
		if options.ErrorOnTagDecode && block.error != nil {
			return nil, fmt.Errorf("failed to decode tag %q at 0x%X: %w", signature, hdr.Offset, block.error)
		}
		offsetCache[hdr.Offset] = block
		result = append(result, block)
	}
	return result, nil
}

var defaultDecoders map[string]func(raw []byte, hdrs []TagHeader) (any, error)

func init() {
	defaultDecoders = map[string]func(raw []byte, hdrs []TagHeader) (any, error){
		TagColorLookupTable:           clutDecoder,
		TagCurve:                      curveDecoder,
		TagDescription:                descDecoder,
		TagDictionary:                 dictDecoder,
		TagGamutBoundaryDescription:   gbdDecoder,
		TagMatrix:                     mtxDecoder,
		TagModularAB:                  modularDecoder,
		TagModularBA:                  modularDecoder,
		TagMeasurement:                measurementDecoder,
		TagMultiFunctionTable1:        mft1Decoder,
		TagMultiFunctionTable2:        mft2Decoder,
		TagMultiLocalizedUnicode:      mlucDecoder,
		TagParametricCurve:            parametricCurveDecoder,
		TagProfileSequenceDescription: pseqDecoder,
		TagProfileSequenceIdentifier:  psidDecoder,
		TagS15Fixed16ArrayType:        sf32Decoder,
		TagSignatureType:              sigDecoder,
		TagText:                       textDecoder,
		TagView:                       viewDecoder,
		TagXYZ:                        xyzDecoder,
		"MSBN":                        msbnDecoder,
		TagZXML:                       zxmlDecoder,
	}
}
