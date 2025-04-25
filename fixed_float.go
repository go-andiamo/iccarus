package iccarus

func readS15Fixed16BE(raw []byte) float64 {
	if len(raw) < 4 {
		panic("readS15Fixed16BE: not enough bytes")
	}
	msb := int16(raw[0])<<8 | int16(raw[1])
	lsb := uint16(raw[2])<<8 | uint16(raw[3])
	return float64(msb) + float64(lsb)/65536.0
}
