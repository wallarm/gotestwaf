package encoder

import (
	"bytes"
	"encoding/xml"
)

var _ Encoder = (*XMLEntityEncoder)(nil)

var DefaultXMLEntityEncoder = &XMLEntityEncoder{name: "XMLEntity"}

type XMLEntityEncoder struct {
	name string
}

func (enc *XMLEntityEncoder) GetName() string {
	return enc.name
}

func (enc *XMLEntityEncoder) Encode(data string) (string, error) {
	b := bytes.NewBufferString("")
	if err := xml.NewEncoder(b).Encode(data); err != nil {
		return "", err
	}
	return b.String(), nil
}
