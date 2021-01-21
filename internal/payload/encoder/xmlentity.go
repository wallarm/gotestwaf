package encoder

import "fmt"

type XMLEntityEncoder struct {
	name string
}

var DefaultXMLEntityEncoder = XMLEntityEncoder{name: "XMLEntity"}

func (enc XMLEntityEncoder) GetName() *string {
	return &enc.name
}

func (enc XMLEntityEncoder) Encode(data string) (string, error) {
	ret := ""
	for _, v := range data {
		ret += "&#x" + fmt.Sprintf("%x", v) + ";"
	}
	return ret, nil
}
