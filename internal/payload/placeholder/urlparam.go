package placeholder

import (
	"net/http"
	"net/url"
)

// Warning: this placeholder encodes URL anyways
func URLParam(requestURL, payload string) (*http.Request, error) {
	param, err := RandomHex(seed)
	if err != nil {
		return nil, err
	}

	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	reqURL.RawQuery = param + "=" + payload
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	return req, err
}
