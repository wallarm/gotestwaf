package placeholder

import (
	"net/http"
	"net/url"
)

func Header(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	randomName, err := RandomHex(seed)
	if err != nil {
		return nil, err
	}

	randomHeader := "X-" + randomName
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(randomHeader, payload)
	return req, nil
}
