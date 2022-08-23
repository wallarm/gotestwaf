package encoder

import (
	"fmt"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type JSUnicodeEncoder struct {
	name string
}

var DefaultJSUnicodeEncoder = JSUnicodeEncoder{name: "JSUnicode"}

var _ Encoder = (*JSUnicodeEncoder)(nil)

func (enc JSUnicodeEncoder) GetName() string {
	return enc.name
}

func (enc JSUnicodeEncoder) Encode(data string) (string, error) {
	encoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewEncoder()
	utf16beStr, _, err := transform.Bytes(encoder, []byte(data))
	if err != nil {
		return "", err
	}

	ret := ""
	for i := 0; i < len(utf16beStr); i += 2 {
		ret += fmt.Sprintf("\\u%02x%02x", utf16beStr[i], utf16beStr[i+1])
	}

	return ret, nil
}
