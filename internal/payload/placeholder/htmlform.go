package placeholder

import (
	"net/http"
	"net/url"
	"strings"
)

func HTMLForm(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	randomName, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	bodyPayload := randomName + "=" + payload
	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(bodyPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}
