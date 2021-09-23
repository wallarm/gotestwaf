package encoder

import (
	"bytes"
	"encoding/xml"
)

type XMLEntityEncoder struct {
	name string
}

var DefaultXMLEntityEncoder = XMLEntityEncoder{name: "XMLEntity"}

func (enc XMLEntityEncoder) GetName() string {
	return enc.name
}

func (enc XMLEntityEncoder) Encode(data string) (string, error) {
	b := bytes.NewBufferString("")
	if err := xml.NewEncoder(b).Encode(data); err != nil {
		return "", err
	}
	return b.String(), nil
}
