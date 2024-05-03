package placeholder

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

var _ Placeholder = (*URLParam)(nil)

var DefaultURLParam = &URLParam{name: "URLParam"}

type URLParam struct {
	name string
}

func (p *URLParam) NewPlaceholderConfig(map[any]any) (PlaceholderConfig, error) {
	return nil, nil
}

func (p *URLParam) GetName() string {
	return p.name
}

func (p *URLParam) CreateRequest(requestURL, payload string, config PlaceholderConfig, httpClientType types.HTTPClientType) (types.Request, error) {
	param, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	reqURL.Fragment = ""
	urlWithPayload := reqURL.String()
	if reqURL.RawQuery == "" {
		for i := len(urlWithPayload) - 1; i >= 0; i-- {
			if urlWithPayload[i] != '/' {
				if strings.HasSuffix(reqURL.Path, urlWithPayload[i:]) {
					urlWithPayload = urlWithPayload[:i+1] + "?"
				} else {
					urlWithPayload = urlWithPayload[:i+1] + "/?"
				}
				break
			}
		}
	} else {
		urlWithPayload += "&"
	}
	urlWithPayload += param + "=" + payload

	req, err := http.NewRequest("GET", urlWithPayload, nil)
	if err != nil {
		return nil, err
	}

	return &types.GoHTTPRequest{Req: req}, err
}
