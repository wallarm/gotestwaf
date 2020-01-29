package encoder

import "net/url"

type UrlEncoder struct {
	name string
}

var DefaultUrlEncoder = UrlEncoder{name: "Url"}

func (enc UrlEncoder) GetName() *string {
	return &enc.name
}

func (enc UrlEncoder) Encode(data string) (string, error) {
	return url.QueryEscape(data), nil
}
