package placeholder

import (
	"net/http"
	"net/url"
	"strings"
)

func FormBody(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}
