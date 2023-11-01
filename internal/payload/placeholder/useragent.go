package placeholder

import (
	"net/http"
	"net/url"
)

const UAHeader = "User-Agent"

type UserAgent struct {
	name string
}

var DefaultUserAgent = UserAgent{name: "UserAgent"}

var _ Placeholder = (*UserAgent)(nil)

func (p UserAgent) newConfig(_ map[any]any) (any, error) {
	return nil, nil
}

func (p UserAgent) GetName() string {
	return p.name
}

func (p UserAgent) CreateRequest(requestURL, payload string, _ any) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(UAHeader, payload)
	return req, nil
}
