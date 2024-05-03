package placeholder

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ Placeholder = (*JSONBody)(nil)

var DefaultJSONBody = &JSONBody{name: "JSONBody"}

type JSONBody struct {
	name string
}

func (p *JSONBody) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *JSONBody) GetName() string {
	return p.name
}

func (p *JSONBody) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	return &types.GoHTTPRequest{Req: req}, nil
}
