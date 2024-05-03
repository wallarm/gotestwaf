package placeholder

import (
	"net/http"
	"net/url"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ Placeholder = (*Header)(nil)

var DefaultHeader = &Header{name: "Header"}

type Header struct {
	name string
}

func (p *Header) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *Header) GetName() string {
	return p.name
}

func (p *Header) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
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

	return &types.GoHTTPRequest{Req: req}, nil
}
