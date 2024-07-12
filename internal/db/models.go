package db

import (
	"crypto/sha256"

	"github.com/wallarm/gotestwaf/internal/payload/placeholder"

	"github.com/wallarm/gotestwaf/internal/helpers"
)

type Info struct {
	Payload            string
	Encoder            string
	Placeholder        string
	Set                string
	Case               string
	ResponseStatusCode int
	AdditionalInfo     []string
	Type               string
}

type yamlConfig struct {
	Payloads     []string `yaml:"payload"`
	Encoders     []string `yaml:"encoder"`
	Placeholders []any    `yaml:"placeholder"` // array of string or map[string]any
	Type         string   `default:"unknown" yaml:"type"`
}

type Case struct {
	Payloads     []string
	Encoders     []string
	Placeholders []*Placeholder
	Type         string

	Set            string
	Name           string
	IsTruePositive bool
}

var _ helpers.Hash = (*Case)(nil)

func (p *Case) Hash() []byte {
	sha256sum := sha256.New()

	for i := range p.Payloads {
		sha256sum.Write([]byte(p.Payloads[i]))
	}

	for i := range p.Encoders {
		sha256sum.Write([]byte(p.Encoders[i]))
	}

	for i := range p.Placeholders {
		if p.Placeholders[i] != nil {
			sha256sum.Write(p.Placeholders[i].Hash())
		}
	}

	sha256sum.Write([]byte(p.Type))
	sha256sum.Write([]byte(p.Set))
	sha256sum.Write([]byte(p.Name))

	if p.IsTruePositive {
		sha256sum.Write([]byte{0x01})
	} else {
		sha256sum.Write([]byte{0x00})
	}

	return sha256sum.Sum(nil)
}

type Placeholder struct {
	Name   string
	Config placeholder.PlaceholderConfig
}

var _ helpers.Hash = (*Placeholder)(nil)

func (p *Placeholder) Hash() []byte {
	sha256sum := sha256.New()
	sha256sum.Write([]byte(p.Name))
	if p.Config != nil {
		sha256sum.Write(p.Config.Hash())
	}
	return sha256sum.Sum(nil)
}
