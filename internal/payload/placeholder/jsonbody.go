package placeholder

import (
	"net/http"
	"net/url"
	"strings"
)

type JSONBody struct {
	name string
}

var DefaultJSONBody = JSONBody{name: "JSONBody"}

var _ Placeholder = (*JSONBody)(nil)

func (p JSONBody) GetName() string {
	return p.name
}

func (p JSONBody) CreateRequest(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	return req, nil
}
