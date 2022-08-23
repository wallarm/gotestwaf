package placeholder

import (
	"net/http"
	"net/url"
)

type Header struct {
	name string
}

var DefaultHeader = Header{name: "Header"}

var _ Placeholder = (*Header)(nil)

func (p Header) GetName() string {
	return p.name
}

func (p Header) CreateRequest(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	randomName, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	randomHeader := "X-" + randomName
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(randomHeader, payload)
	return req, nil
}
