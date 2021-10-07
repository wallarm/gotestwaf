package placeholder

import (
	"net/http"
	"net/url"
	"strings"
)

func XMLBody(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "text/xml")

	return req, nil
}
