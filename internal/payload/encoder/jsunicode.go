package encoder

import (
	"fmt"
)

type JSUnicodeEncoder struct {
	name string
}

var DefaultJSUnicodeEncoder = JSUnicodeEncoder{name: "JSUnicode"}

func (enc JSUnicodeEncoder) GetName() *string {
	return &enc.name
}

func (enc JSUnicodeEncoder) Encode(data string) (string, error) {
	ret := ""
	// TODO: check hot it works with unicode multibytes
	for _, r := range data {
		if r < 'a' || r > 'z' &&
			r < 'A' || r > 'Z' &&
			r < '0' || r > '9' {
			ret += string(r)
		} else {
			ret += "\\u00" + fmt.Sprintf("%x", r)
		}
	}
	return ret, nil
}
