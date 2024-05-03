package placeholder

import (
	"net/http"
	"net/url"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ Placeholder = (*URLPath)(nil)

var DefaultURLPath = &URLPath{name: "URLPath"}

type URLPath struct {
	name string
}

func (p *URLPath) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *URLPath) GetName() string {
	return p.name
}

func (p *URLPath) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
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

	return &types.GoHTTPRequest{Req: req}, nil
}
