package placeholder

import (
	"net/http"
	"net/url"
)

func URLPath(requestURL, payload string) (*http.Request, error) {
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

	return req, nil
}
