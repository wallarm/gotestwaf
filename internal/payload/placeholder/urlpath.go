package placeholder

import (
	"net/http"
	"net/url"
)

type URLPath struct {
	name string
}

var DefaultURLPath = URLPath{name: "URLPath"}

var _ Placeholder = (*URLPath)(nil)

func (p URLPath) GetName() string {
	return p.name
}

func (p URLPath) CreateRequest(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	urlWithPayload := reqURL.String()
	for i := len(urlWithPayload) - 1; i >= 0; i-- {
		if urlWithPayload[i] != '/' {
			urlWithPayload = urlWithPayload[:i+1]
			break
		}
	}
	urlWithPayload += "/" + payload

	req, err := http.NewRequest("GET", urlWithPayload, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}
