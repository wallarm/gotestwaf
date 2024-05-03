package placeholder

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/wallarm/gotestwaf/internal/scanner/types"

	"github.com/wallarm/gotestwaf/internal/payload/encoder"
)

const jsonRequestPayloadWrapper = "{\"test\":true, \"%s\": \"%s\"}"

var _ Placeholder = (*JSONRequest)(nil)

var DefaultJSONRequest = &JSONRequest{name: "JSONRequest"}

type JSONRequest struct {
	name string
}

func (p *JSONRequest) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *JSONRequest) GetName() string {
	return p.name
}

func (p *JSONRequest) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	param, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	encodedPayload, err := encoder.Apply("JSUnicode", payload)
	if err != nil {
		return nil, err
	}

	jsonPayload := fmt.Sprintf(jsonRequestPayloadWrapper, param, encodedPayload)

	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	return &types.GoHTTPRequest{Req: req}, nil
}
