package placeholder

import (
	"net/http"
	"net/url"
)

func URLParam(requestURL, payload string) (*http.Request, error) {
	param, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	urlWithPayload := reqURL.String()
	if reqURL.RawQuery == "" {
		urlWithPayload += "?"
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
