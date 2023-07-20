package placeholder

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/wallarm/gotestwaf/internal/payload/encoder"
)

type JSONRequest struct {
	name string
}

var DefaultJSONRequest = JSONRequest{name: "JSONRequest"}

var _ Placeholder = (*JSONRequest)(nil)

func (p JSONRequest) newConfig(_ map[any]any) (any, error) {
	return nil, nil
}

func (p JSONRequest) GetName() string {
	return p.name
}

func (p JSONRequest) CreateRequest(requestURL, payload string, _ any) (*http.Request, error) {
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
	jsonPayload := fmt.Sprintf("{\"test\":true, \"%s\": \"%s\"}", param, encodedPayload)
	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(jsonPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}
