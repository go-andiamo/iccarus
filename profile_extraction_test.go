package iccarus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/go-andiamo/iccarus/_test_data/images"
	"github.com/stretchr/testify/require"
	"io"
	"strings"
	"testing"
)

func TestExtractProfile(t *testing.T) {
	names := images.List()
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			f, err := images.Open(name)
			require.NoError(t, err)
			defer func() {
				_ = f.Close()
			}()
			var p *Profile
			switch {
			case strings.HasSuffix(name, ".jpeg"):
				p, err = ExtractFromJPEG(f, nil)
			case strings.HasSuffix(name, ".tif"):
				p, err = ExtractFromTIFF(f, nil)
			case strings.HasSuffix(name, ".png"):
				p, err = ExtractFromPNG(f, nil)
			case strings.HasSuffix(name, ".webp"):
				p, err = ExtractFromWebP(f, nil)
			default:
				err = errors.New("unknown image type")
			}
			require.NoError(t, err)
			require.NotNil(t, p)
		})
	}
}

func TestExtractFromJPEG_Errors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "Not a JPEG (wrong SOI)",
			data:    []byte{0x00, 0x00},
			wantErr: "not a JPEG file",
		},
		{
			name:    "Truncated before SOI",
			data:    []byte{},
			wantErr: "EOF",
		},
		{
			name: "No ICC chunks present",
			data: func() []byte {
				buf := &bytes.Buffer{}
				buf.Write([]byte{0xFF, 0xD8}) // SOI
				// APP0 marker, length=4
				buf.Write([]byte{0xFF, 0xE0, 0x00, 0x04})
				buf.Write([]byte("JFIF"))
				buf.Write([]byte{0xFF, 0xD9}) // EOI
				return buf.Bytes()
			}(),
			wantErr: "no ICC profile found",
		},
		{
			name: "Missing chunk in sequence",
			data: func() []byte {
				buf := &bytes.Buffer{}
				buf.Write([]byte{0xFF, 0xD8}) // SOI

				// Write chunk #2 (but not chunk #1)
				profile := append([]byte("ICC_PROFILE\x00"), 0x02, 0x02)
				profile = append(profile, []byte("chunk2")...)

				length := uint16(len(profile) + 2)
				buf.Write([]byte{0xFF, 0xE2})
				binary.Write(buf, binary.BigEndian, length)
				buf.Write(profile)

				buf.Write([]byte{0xFF, 0xD9}) // EOI
				return buf.Bytes()
			}(),
			wantErr: "missing ICC chunk #1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractFromJPEG(bytes.NewReader(tt.data), &ParseOptions{})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestExtractFromTIFF_Errors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "Too short to read header",
			data:    []byte{},
			wantErr: "failed to read TIFF header",
		},
		{
			name:    "Invalid byte order",
			data:    append([]byte("ZZ\x00\x2A\x00\x00\x00\x08"), make([]byte, 100)...),
			wantErr: "invalid TIFF byte order",
		},
		{
			name:    "Missing magic number 42",
			data:    append([]byte("II\x00\x00\x00\x00\x00\x08"), make([]byte, 100)...),
			wantErr: "not a valid TIFF file",
		},
		{
			name:    "Seek to IFD fails",
			data:    append([]byte("II\x2A\x00\xFF\xFF\xFF\xFF"), make([]byte, 10)...), // offset too large
			wantErr: "failed to seek to IFD",
		},
		{
			name: "IFD entry count read fails",
			data: func() []byte {
				buf := &bytes.Buffer{}
				buf.Write([]byte("II\x2A\x00\x08\x00\x00\x00")) // Header, offset 8
				buf.Write([]byte{0x00})                         // Too short to read 2 bytes of count
				return buf.Bytes()
			}(),
			wantErr: "failed to read IFD entry count",
		},
		{
			name: "IFD entry read fails",
			data: func() []byte {
				buf := &bytes.Buffer{}
				buf.Write([]byte("II\x2A\x00\x08\x00\x00\x00")) // header
				buf.Write([]byte{0x01, 0x00})                   // count = 1
				buf.Write([]byte{0x00})                         // too short for full IFD entry (needs 12 bytes)
				return buf.Bytes()
			}(),
			wantErr: "failed to read IFD entry",
		},
		{
			name: "ICC profile not found",
			data: func() []byte {
				buf := &bytes.Buffer{}
				buf.Write([]byte("II\x2A\x00\x08\x00\x00\x00")) // header
				buf.Write([]byte{0x01, 0x00})                   // count = 1
				buf.Write(make([]byte, 12))                     // 1 dummy entry with unrelated tag
				return buf.Bytes()
			}(),
			wantErr: "no ICC profile found",
		},
		{
			name: "Seek to ICC profile fails",
			data: func() []byte {
				buf := &bytes.Buffer{}
				buf.Write([]byte("II\x2A\x00\x08\x00\x00\x00")) // header
				buf.Write([]byte{0x01, 0x00})                   // count = 1
				entry := make([]byte, 12)
				binary.LittleEndian.PutUint16(entry[0:2], 34675) // tagICCProfile
				binary.LittleEndian.PutUint32(entry[4:8], 1000)  // length
				binary.LittleEndian.PutUint32(entry[8:12], 1000) // offset
				buf.Write(entry)
				return buf.Bytes()
			}(),
			wantErr: "failed to seek to ICC profile",
		},
		{
			name: "Failed to read ICC profile data",
			data: func() []byte {
				buf := &bytes.Buffer{}
				buf.Write([]byte("II\x2A\x00\x08\x00\x00\x00")) // header
				buf.Write([]byte{0x01, 0x00})                   // count = 1
				entry := make([]byte, 12)
				binary.LittleEndian.PutUint16(entry[0:2], 34675)
				binary.LittleEndian.PutUint32(entry[4:8], 16)  // length
				binary.LittleEndian.PutUint32(entry[8:12], 32) // offset
				buf.Write(entry)
				padding := make([]byte, int(32)-(8+2+12)) // to get to offset
				buf.Write(padding)
				buf.Write([]byte("too short")) // less than 16 bytes
				return buf.Bytes()
			}(),
			wantErr: "failed to read ICC profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractFromTIFF(bytes.NewReader(tt.data), &ParseOptions{})
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error to contain %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestExtractFromPNG_Errors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "Too short to read signature",
			data:    []byte{},
			wantErr: "failed to read PNG signature",
		},
		{
			name:    "Invalid PNG signature",
			data:    append([]byte("BAD_SIG!"), make([]byte, 8)...),
			wantErr: "not a valid PNG file",
		},
		{
			name: "Fail to read chunk length",
			data: func() []byte {
				return append(validPNGHeader(), []byte{0x00}...) // less than 4 bytes for length
			}(),
			wantErr: "failed to read chunk length",
		},
		{
			name: "Fail to read chunk type",
			data: func() []byte {
				buf := bytes.NewBuffer(validPNGHeader())
				binary.Write(buf, binary.BigEndian, uint32(4)) // valid length
				buf.Write([]byte{0x00})                        // incomplete type
				return buf.Bytes()
			}(),
			wantErr: "failed to read chunk type",
		},
		{
			name: "Fail to read chunk data",
			data: func() []byte {
				buf := bytes.NewBuffer(validPNGHeader())
				binary.Write(buf, binary.BigEndian, uint32(10)) // chunk length
				buf.Write([]byte("iCCP"))
				buf.Write([]byte("short")) // less than 10 bytes
				return buf.Bytes()
			}(),
			wantErr: "failed to read chunk data",
		},
		{
			name: "Fail to discard CRC",
			data: func() []byte {
				buf := bytes.NewBuffer(validPNGHeader())
				binary.Write(buf, binary.BigEndian, uint32(0)) // chunk length
				buf.Write([]byte("iCCP"))
				// zero-length chunk, but no 4-byte CRC
				return buf.Bytes()
			}(),
			wantErr: "failed to discard CRC",
		},
		{
			name: "Malformed iCCP chunk (no null separator)",
			data: func() []byte {
				buf := bytes.NewBuffer(validPNGHeader())
				data := []byte("nonullheaderwithnocompressionbyte")
				writeChunk(buf, "iCCP", data)
				return buf.Bytes()
			}(),
			wantErr: "invalid iCCP chunk format",
		},
		{
			name: "Zlib decompression fails",
			data: func() []byte {
				buf := bytes.NewBuffer(validPNGHeader())
				data := append([]byte("name\x00\x00"), []byte("notzlib")...) // invalid zlib
				writeChunk(buf, "iCCP", data)
				return buf.Bytes()
			}(),
			wantErr: "failed to decompress ICC profile",
		},
		{
			name: "No ICC profile found (no iCCP)",
			data: func() []byte {
				buf := bytes.NewBuffer(validPNGHeader())
				writeChunk(buf, "IDAT", []byte("dummy"))
				writeChunk(buf, "IEND", []byte{})
				return buf.Bytes()
			}(),
			wantErr: "no ICC profile found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractFromPNG(bytes.NewReader(tt.data), &ParseOptions{})
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error to contain %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func validPNGHeader() []byte {
	return []byte{137, 80, 78, 71, 13, 10, 26, 10}
}

func writeChunk(w io.Writer, chunkType string, data []byte) {
	binary.Write(w, binary.BigEndian, uint32(len(data)))
	w.Write([]byte(chunkType))
	w.Write(data)
	w.Write([]byte{0, 0, 0, 0}) // fake CRC
}

func TestExtractFromWebP_Errors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "Too short to read RIFF header",
			data:    []byte{1, 2, 3},
			wantErr: "failed to read RIFF header",
		},
		{
			name:    "Invalid RIFF magic",
			data:    append([]byte("XXXXxxxxWEBP"), make([]byte, 100)...),
			wantErr: "not a valid WebP",
		},
		{
			name:    "Invalid WEBP magic",
			data:    append([]byte("RIFFxxxxXXXX"), make([]byte, 100)...),
			wantErr: "not a valid WebP",
		},
		{
			name: "Chunk header too short",
			data: func() []byte {
				buf := bytes.NewBuffer(webpHeader())
				buf.Write([]byte{0x00}) // too short for 8-byte chunk header
				return buf.Bytes()
			}(),
			wantErr: "failed to read chunk header",
		},
		{
			name: "Truncated ICCP chunk body",
			data: func() []byte {
				buf := bytes.NewBuffer(webpHeader())
				writeChunkHeader(buf, "ICCP", 5)
				buf.Write([]byte{1, 2}) // less than 5
				return buf.Bytes()
			}(),
			wantErr: "failed to read ICCP chunk",
		},
		{
			name: "Padding read fails after odd-sized ICCP chunk",
			data: func() []byte {
				buf := bytes.NewBuffer(webpHeader())
				writeChunkHeader(buf, "ICCP", 3)
				buf.Write([]byte{1, 2, 3}) // 3 bytes
				// omit 1 byte padding
				return buf.Bytes()
			}(),
			wantErr: "failed to discard ICCP chunk",
		},
		{
			name: "Fail to skip non-ICCP chunk",
			data: func() []byte {
				buf := bytes.NewBuffer(webpHeader())
				writeChunkHeader(buf, "XXXX", 10)
				buf.Write([]byte{1, 2, 3}) // not enough to skip 10
				return buf.Bytes()
			}(),
			wantErr: "failed to skip chunk",
		},
		{
			name: "No ICC profile found",
			data: func() []byte {
				buf := bytes.NewBuffer(webpHeader())
				writeChunkHeader(buf, "VP8 ", 6)
				buf.Write([]byte("abc123"))
				return buf.Bytes()
			}(),
			wantErr: "no ICC profile found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractFromWebP(bytes.NewReader(tt.data), &ParseOptions{})
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error to contain %q, got: %q", tt.wantErr, err.Error())
			}
		})
	}
}

func webpHeader() []byte {
	buf := &bytes.Buffer{}
	buf.Write([]byte("RIFF"))                         // RIFF magic
	binary.Write(buf, binary.LittleEndian, uint32(0)) // dummy size
	buf.Write([]byte("WEBP"))                         // WEBP magic
	return buf.Bytes()
}

func writeChunkHeader(w io.Writer, chunkType string, size uint32) {
	w.Write([]byte(chunkType))
	binary.Write(w, binary.LittleEndian, size)
}
