package iccarus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Header represents the parsed ICC profile header (128 bytes)
type Header struct {
	ProfileSize     uint32
	CMMType         string
	VersionRaw      uint32
	Version         Version
	DeviceClass     string
	ColorSpace      string
	PCS             string
	Created         time.Time
	Signature       string
	Platform        string
	Flags           uint32
	Manufacturer    string
	Model           string
	Attributes      [8]byte
	RenderingIntent uint32
	Illuminant      [3]float64
	Creator         string
	ProfileID       [16]byte
}

func parseHeader(r io.Reader) (Header, error) {
	var buf [128]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return Header{}, err
	}
	signature := stringed(buf[36:40])
	if signature != "acsp" {
		return Header{}, errors.New("invalid ICC profile: missing 'acsp' signature")
	}
	var profileId [16]byte
	copy(profileId[:], buf[84:100])
	versionRaw := binary.BigEndian.Uint32(buf[8:12])
	return Header{
		ProfileSize: binary.BigEndian.Uint32(buf[0:4]),
		CMMType:     stringed(buf[4:8]),
		VersionRaw:  versionRaw,
		Version:     versionFromRaw(versionRaw),
		DeviceClass: stringed(buf[12:16]),
		ColorSpace:  stringed(buf[16:20]),
		PCS:         stringed(buf[20:24]),
		Created: time.Date(
			int(binary.BigEndian.Uint16(buf[24:26])),
			time.Month(binary.BigEndian.Uint16(buf[26:28])),
			int(binary.BigEndian.Uint16(buf[28:30])),
			int(binary.BigEndian.Uint16(buf[30:32])),
			int(binary.BigEndian.Uint16(buf[32:34])),
			int(binary.BigEndian.Uint16(buf[34:36])),
			0, time.UTC),
		Signature:       signature,
		Platform:        stringed(buf[40:44]),
		Flags:           binary.BigEndian.Uint32(buf[44:48]),
		Manufacturer:    stringed(buf[48:52]),
		Model:           stringed(buf[52:56]),
		Attributes:      [8]byte(buf[56:64]),
		RenderingIntent: binary.BigEndian.Uint32(buf[64:68]),
		Illuminant: [3]float64{
			readS15Fixed16BE(buf[68:72]),
			readS15Fixed16BE(buf[72:76]),
			readS15Fixed16BE(buf[76:80]),
		},
		Creator:   stringed(buf[80:84]),
		ProfileID: profileId,
	}, nil
}

type Version struct {
	Major    int
	Minor    int
	Revision int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Revision)
}

func versionFromRaw(v uint32) Version {
	return Version{
		Major:    int((v >> 24) & 0xFF),
		Minor:    int((v >> 20) & 0x0F),
		Revision: int((v >> 16) & 0x0F),
	}
}

func stringed(data []byte) string {
	if bytes.Equal(data, []byte{0, 0, 0, 0}) {
		return ""
	}
	s := strings.TrimRight(string(data), "\x00 ")
	for _, b := range s {
		if b < 32 || b > 126 {
			return fmt.Sprintf("0x%02X%02X%02X%02X", data[0], data[1], data[2], data[3])
		}
	}
	return s
}
