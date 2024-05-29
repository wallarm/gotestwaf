package encoder

import (
	"net/url"
)

var _ Encoder = (*URLEncoder)(nil)

var DefaultURLEncoder = &URLEncoder{name: "URL"}

type URLEncoder struct {
	name string
}

func (enc *URLEncoder) GetName() string {
	return enc.name
}

func (enc *URLEncoder) Encode(data string) (string, error) {
	return url.PathEscape(data), nil
}
