package encoder

type JSUnicodeEncoder struct {
	name string
}

var DefaultJSUnicodeEncoder = JSUnicodeEncoder{name: "JSUnicode"}

func (enc JSUnicodeEncoder) GetName() *string {
	return &enc.name
}

func (enc JSUnicodeEncoder) Encode(data string) (string, error) {
	ret := ""
	// TODO: check how it works with unicode multibytes
	for _, c := range data {
		if c < 'a' || c > 'z' &&
			c < 'A' || c > 'Z' &&
			c < '0' || c > '9' {
			ret += string(c)
			continue
		}
		ret += "\\u00" + string(c)
	}
	return ret, nil
}
