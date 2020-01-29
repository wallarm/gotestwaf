package encoder

import "fmt"
import "regexp"

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
	for _, v := range data {
		matched, _ := regexp.MatchString(`[a-zA-Z0-9]`, string(v))
		if matched {
			ret += string(v)
		} else {
			ret += "\\u00" + fmt.Sprintf("%x", v)
		}
	}
	return ret, nil
}
