package encoder

import "fmt"

type XmlEntityEncoder struct {
	name string
}

var DefaultXmlEntityEncoder = XmlEntityEncoder{name: "XmlEntity"}

func (enc XmlEntityEncoder) GetName() *string {
	return &enc.name
}

func (enc XmlEntityEncoder) Encode(data string) (string, error) {
	ret := ""
	for _, v := range data {
		ret += "&#x" + fmt.Sprintf("%x", v) + ";"
	}
	return ret, nil
}
