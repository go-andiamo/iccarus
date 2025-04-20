package iccarus

import (
	"encoding/binary"
	"fmt"
	"io"
)

type TagHeader struct {
	Signature string // Signature (e.g., "desc", "rXYZ", "A2B0")
	Offset    uint32 // Offset from beginning of file
	Size      uint32 // Size of tag data in bytes
}

type TagHeaderTable struct {
	Entries []TagHeader
}

const maxTagCount = 1024

func parseTagHeaders(r io.Reader) (TagHeaderTable, error) {
	var countBuf [4]byte
	if _, err := io.ReadFull(r, countBuf[:]); err != nil {
		return TagHeaderTable{}, err
	}
	count := binary.BigEndian.Uint32(countBuf[:])
	if count > maxTagCount {
		return TagHeaderTable{}, fmt.Errorf("tag count %d exceeds max allowed (%d)", count, maxTagCount)
	}
	tagBytes := make([]byte, count*12)
	if _, err := io.ReadFull(r, tagBytes); err != nil {
		return TagHeaderTable{}, err
	}
	entries := make([]TagHeader, count)
	for i := uint32(0); i < count; i++ {
		base := i * 12
		entries[i].Signature = stringed(tagBytes[base : base+4])
		entries[i].Offset = binary.BigEndian.Uint32(tagBytes[base+4 : base+8])
		entries[i].Size = binary.BigEndian.Uint32(tagBytes[base+8 : base+12])
	}
	return TagHeaderTable{Entries: entries}, nil
}
