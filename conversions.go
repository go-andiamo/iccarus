package iccarus

import (
	"errors"
	"fmt"
)

type ToCIEXYZ interface {
	ToCIEXYZ(channels ...float64) ([]float64, error)
}

type FromCIEXYZ interface {
	FromCIEXYZ(channels ...float64) ([]float64, error)
}

type ChannelTransformer interface {
	Transform(input []float64) ([]float64, error)
}

var _ ToCIEXYZ = (*Profile)(nil)
var _ FromCIEXYZ = (*Profile)(nil)

func (p *Profile) ToCIEXYZ(channels ...float64) ([]float64, error) {
	a2bTag, err := p.findA2B0()
	if err != nil {
		return nil, err
	}
	return a2bTag.ToCIEXYZ(channels...)
}

func (p *Profile) FromCIEXYZ(channels ...float64) ([]float64, error) {
	b2aTag, err := p.findB2A0()
	if err != nil {
		return nil, err
	}
	return b2aTag.FromCIEXYZ(channels...)
}

func (p *Profile) findA2B0() (ToCIEXYZ, error) {
	if p.a2b0 != nil {
		return p.a2b0, nil
	}
	tag, ok := p.TagByHeader(TagHeaderAToB0)
	if !ok || tag == nil {
		return nil, errors.New("A2B0 tag not found")
	}
	val, err := tag.Value()
	if err != nil {
		return nil, fmt.Errorf("failed to decode A2B0 tag: %w", err)
	}
	modular, ok := val.(ToCIEXYZ)
	if !ok {
		return nil, fmt.Errorf("A2B0 tag does not implement interface ToCIEXYZ (got %T)", val)
	}
	p.a2b0 = modular
	return modular, nil
}

func (p *Profile) findB2A0() (FromCIEXYZ, error) {
	if p.b2a0 != nil {
		return p.b2a0, nil
	}
	tag, ok := p.TagByHeader(TagHeaderBToA0)
	if !ok || tag == nil {
		return nil, errors.New("B2A0 tag not found")
	}
	val, err := tag.Value()
	if err != nil {
		return nil, fmt.Errorf("failed to decode B2A0 tag: %w", err)
	}
	modular, ok := val.(FromCIEXYZ)
	if !ok {
		return nil, fmt.Errorf("B2A0 tag does not implement interface FromCIEXYZ (got %T)", val)
	}
	p.b2a0 = modular
	return modular, nil
}
