package placeholder

import (
	"net/http"
	"net/url"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

const UAHeader = "User-Agent"

var _ Placeholder = (*UserAgent)(nil)

var DefaultUserAgent = &UserAgent{name: "UserAgent"}

type UserAgent struct {
	name string
}

func (p *UserAgent) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *UserAgent) GetName() string {
	return p.name
}

func (p *UserAgent) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(UAHeader, payload)

	return &types.GoHTTPRequest{Req: req}, nil
}
