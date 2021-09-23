package placeholder

import (
	"fmt"
	"net/http"
	"net/url"
)

type URLPath struct {
	name string
}

var DefaultURLPath = URLPath{name: "URLPath"}

func (p URLPath) GetName() string {
	return p.name
}

func (p URLPath) CreateRequest(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	reqURL.Path = fmt.Sprintf("%s/%s/", reqURL.Path, payload)
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}
