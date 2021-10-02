package encoder

import (
	"net/url"
)

type URLEncoder struct {
	name string
}

var DefaultURLEncoder = URLEncoder{name: "URL"}

func (enc URLEncoder) GetName() string {
	return enc.name
}

func (enc URLEncoder) Encode(data string) (string, error) {
	return url.QueryEscape(data), nil
}
