package placeholder

import (
	"net/http"
)

// Warning: this placeholder encodes URL anyways
func URLParam(requestURL, payload string) (*http.Request, error) {
	param, err := RandomHex(seed)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", requestURL+"/?"+param+"="+payload, nil)
	if err != nil {
		return nil, err
	}
	return req, err
}
