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
	switch httpClientType {
	case types.GoHTTPClient:
		return p.prepareGoHTTPClientRequest(requestURL, payload, config)
	case types.ChromeHTTPClient:
		return p.prepareChromeHTTPClientRequest(requestURL, payload, config)
	default:
		return nil, types.NewUnknownHTTPClientError(httpClientType)
	}
}

func (p *URLPath) prepareGoHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.GoHTTPRequest, error) {
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

func (p *URLPath) prepareChromeHTTPClientRequest(requestURL, payload string, config PlaceholderConfig) (*types.ChromeDPTasks, error) {
	return nil, nil
}
