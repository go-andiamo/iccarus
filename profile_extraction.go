package iccarus

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// ExtractFromJPEG extracts ICC profile from a .jpeg image
func ExtractFromJPEG(r io.Reader, options *ParseOptions) (result *Profile, err error) {
	const (
		signature    = "ICC_PROFILE\x00"
		signatureLen = len(signature)
	)
	var marker [2]byte
	if _, err = io.ReadFull(r, marker[:]); err != nil {
		return nil, err
	} else if marker[0] != 0xFF || marker[1] != 0xD8 {
		return nil, errors.New("not a JPEG file")
	}
	chunks := make(map[int][]byte)
	totalLen := 0
	for err == nil {
		if _, err = io.ReadFull(r, marker[:]); err == nil {
			if marker[0] != 0xFF || marker[1] == 0xD9 {
				break
			}
			var length uint16
			if err = binary.Read(r, binary.BigEndian, &length); err == nil {
				length -= 2
				data := make([]byte, length)
				if _, err = io.ReadFull(r, data); err == nil && marker[1] == 0xE2 && bytes.HasPrefix(data, []byte(signature)) {
					seqNo := int(data[signatureLen])
					if len(data) >= signatureLen+2 {
						chunks[seqNo] = data[signatureLen+2:]
						totalLen += int(length)
					} else {
						err = errors.New("invalid profile length")
					}
				}
			}
		} else if err == io.EOF {
			err = nil
			break
		}
	}
	if err == nil {
		if len(chunks) == 0 {
			return nil, errors.New("no ICC profile found")
		}
		combined := make([]byte, 0, totalLen)
		for i := 1; i <= len(chunks); i++ {
			if chunk, ok := chunks[i]; !ok {
				return nil, fmt.Errorf("missing ICC chunk #%d", i)
			} else {
				combined = append(combined, chunk...)
			}
		}
		result, err = ParseProfile(bytes.NewReader(combined), options)
	}
	return result, err
}

// ExtractFromTIFF extracts ICC profile from a .tif image
func ExtractFromTIFF(r io.Reader, options *ParseOptions) (*Profile, error) {
	const (
		tagICCProfile = 34675
	)
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("failed to read TIFF header: %w", err)
	}
	var bo binary.ByteOrder
	switch string(header[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return nil, errors.New("invalid TIFF byte order")
	}
	if bo.Uint16(header[2:4]) != 42 {
		return nil, errors.New("not a valid TIFF file (missing 42)")
	}
	ifdOffset := bo.Uint32(header[4:8])
	if _, err := io.CopyN(io.Discard, r, int64(ifdOffset-8)); err != nil {
		return nil, fmt.Errorf("failed to seek to IFD: %w", err)
	}
	var count uint16
	if err := binary.Read(r, bo, &count); err != nil {
		return nil, fmt.Errorf("failed to read IFD entry count: %w", err)
	}
	var iccOffset, iccLength uint32
	entry := make([]byte, 12)
	for i := 0; i < int(count); i++ {
		if _, err := io.ReadFull(r, entry); err != nil {
			return nil, fmt.Errorf("failed to read IFD entry: %w", err)
		}
		tag := bo.Uint16(entry[0:2])
		if tag == tagICCProfile {
			iccLength = bo.Uint32(entry[4:8])
			iccOffset = bo.Uint32(entry[8:12])
			break
		}
	}
	if iccLength == 0 {
		return nil, errors.New("no ICC profile found")
	}
	if _, err := io.CopyN(io.Discard, r, int64(iccOffset)-(int64(ifdOffset)+2+12*int64(count))); err != nil {
		return nil, fmt.Errorf("failed to seek to ICC profile: %w", err)
	}
	iccData := make([]byte, iccLength)
	if _, err := io.ReadFull(r, iccData); err != nil {
		return nil, fmt.Errorf("failed to read ICC profile: %w", err)
	}
	return ParseProfile(bytes.NewReader(iccData), options)
}

// ExtractFromPNG extracts ICC profile from a .png image
func ExtractFromPNG(r io.Reader, options *ParseOptions) (*Profile, error) {
	const iccpChunk = "iCCP"
	sig := make([]byte, 8)
	if _, err := io.ReadFull(r, sig); err != nil {
		return nil, fmt.Errorf("failed to read PNG signature: %w", err)
	}
	if !bytes.Equal(sig, []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return nil, errors.New("not a valid PNG file")
	}
	for {
		var length uint32
		if err := binary.Read(r, binary.BigEndian, &length); err != nil {
			return nil, fmt.Errorf("failed to read chunk length: %w", err)
		}
		chunkType := make([]byte, 4)
		if _, err := io.ReadFull(r, chunkType); err != nil {
			return nil, fmt.Errorf("failed to read chunk type: %w", err)
		}
		chunkData := make([]byte, length)
		if _, err := io.ReadFull(r, chunkData); err != nil {
			return nil, fmt.Errorf("failed to read chunk data: %w", err)
		}
		if _, err := io.CopyN(io.Discard, r, 4); err != nil {
			return nil, fmt.Errorf("failed to discard CRC: %w", err)
		}
		if string(chunkType) == iccpChunk {
			parts := bytes.SplitN(chunkData, []byte{0}, 2)
			if len(parts) != 2 || len(parts[1]) < 1 {
				return nil, errors.New("invalid iCCP chunk format")
			}
			compressed := parts[1][1:] // skip compression method byte
			iccData, err := decompressZlib(compressed)
			if err != nil {
				return nil, fmt.Errorf("failed to decompress ICC profile: %w", err)
			}
			return ParseProfile(bytes.NewReader(iccData), options)
		}
		if string(chunkType) == "IEND" {
			break
		}
	}
	return nil, errors.New("no ICC profile found")
}

func decompressZlib(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = r.Close()
	}()
	return io.ReadAll(r)
}

// ExtractFromWebP extracts ICC profile from a .webp image
func ExtractFromWebP(r io.Reader, options *ParseOptions) (*Profile, error) {
	const iccpChunk = "ICCP"
	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("failed to read RIFF header: %w", err)
	}
	if !bytes.Equal(header[:4], []byte("RIFF")) || !bytes.Equal(header[8:12], []byte("WEBP")) {
		return nil, errors.New("not a valid WebP (missing RIFF/WEBP headers)")
	}
	for {
		chunkHeader := make([]byte, 8)
		if _, err := io.ReadFull(r, chunkHeader); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read chunk header: %w", err)
		}
		chunkType := string(chunkHeader[:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])
		if chunkType == iccpChunk {
			iccData := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, iccData); err != nil {
				return nil, fmt.Errorf("failed to read ICCP chunk: %w", err)
			}
			if chunkSize%2 == 1 {
				if _, err := io.CopyN(io.Discard, r, 1); err != nil {
					return nil, fmt.Errorf("failed to discard ICCP chunk: %w", err)
				}
			}
			return ParseProfile(bytes.NewReader(iccData), options)
		}
		if _, err := io.CopyN(io.Discard, r, int64(chunkSize+(chunkSize%2))); err != nil {
			return nil, fmt.Errorf("failed to skip chunk %q: %w", chunkType, err)
		}
	}
	return nil, errors.New("no ICC profile found")
}
