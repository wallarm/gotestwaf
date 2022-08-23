package placeholder

import (
	"net/http"
	"net/url"
	"strings"
)

type URLParam struct {
	name string
}

var DefaultURLParam = URLParam{name: "URLParam"}

var _ Placeholder = (*URLParam)(nil)

func (p URLParam) GetName() string {
	return p.name
}

func (p URLParam) CreateRequest(requestURL, payload string) (*http.Request, error) {
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
	return req, err
}
