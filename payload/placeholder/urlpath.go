package placeholder

import (
	"fmt"
	"net/http"
	"net/url"
)

func URLPath(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	reqURL.Path = fmt.Sprintf("%s/%s/", reqURL.Path, payload)
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}
