package iccarus

func dictDecoder(raw []byte) (any, error) {
	// TODO: dictionary tag (ICC.1:2010-12), parse only if needed
	return raw, nil
}

func psidDecoder(raw []byte) (any, error) {
	//TODO rarely used and spec/usages don't match!
	return raw, nil
}

func pseqDecoder(raw []byte) (any, error) {
	//TODO rarely useful, descriptive only
	return raw, nil
}

func gbdDecoder(raw []byte) (any, error) {
	//TODO vendor-specific, not safely parseable without spec
	return raw, nil
}

func zxmlDecoder(raw []byte) (any, error) {
	// TODO: Vendor-specific, unknown encoding – stubbed
	return raw, nil
}

func msbnDecoder(raw []byte) (any, error) {
	// TODO: Unknown vendor-specific tag (MSBN) – stubbed
	return raw, nil
}
