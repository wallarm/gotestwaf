package placeholder

import (
	"net/http"
	"net/url"
)

type URLParam struct {
	name string
}

var DefaultURLParam = URLParam{name: "URLParam"}

func (p URLParam) GetName() string {
	return p.name
}

// Warning: this placeholder encodes URL anyways
func (p URLParam) CreateRequest(requestURL, payload string) (*http.Request, error) {
	param, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	reqURL.RawQuery = param + "=" + payload
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	return req, err
}
