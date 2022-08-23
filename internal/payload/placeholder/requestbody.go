package placeholder

import (
	"net/http"
	"net/url"
	"strings"
)

type RequestBody struct {
	name string
}

var DefaultRequestBody = RequestBody{name: "RequestBody"}

var _ Placeholder = (*RequestBody)(nil)

func (p RequestBody) GetName() string {
	return p.name
}

func (p RequestBody) CreateRequest(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}
	// check if we need to set Content-Length manually here
	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return req, nil
}
