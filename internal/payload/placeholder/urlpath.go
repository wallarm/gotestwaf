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
	if urlWithPayload[len(urlWithPayload)-1] == '/' {
		urlWithPayload += payload
	} else {
		urlWithPayload += "/" + payload
	}

	req, err := http.NewRequest("GET", urlWithPayload, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}
