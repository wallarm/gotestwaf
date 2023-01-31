package placeholder

import (
	"net/http"
	"net/url"
	"strings"
)

type GraphQlGET struct {
	name string
}

type GraphQlPOST struct {
	name string
}

var DefaultGraphQlGET = GraphQlGET{name: "GraphQlGET"}
var DefaultGraphQlPOST = GraphQlPOST{name: "GraphQlPOST"}

var _ Placeholder = (*GraphQlGET)(nil)
var _ Placeholder = (*GraphQlPOST)(nil)

func (p GraphQlGET) GetName() string {
	return p.name
}

func (p GraphQlGET) CreateRequest(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	queryParams := reqURL.Query()
	queryParams.Set("query", payload)
	reqURL.RawQuery = queryParams.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (p GraphQlPOST) GetName() string {
	return p.name
}

func (p GraphQlPOST) CreateRequest(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, reqURL.String(), strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	return req, nil
}
