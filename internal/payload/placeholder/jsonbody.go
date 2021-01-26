package placeholder

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/wallarm/gotestwaf/internal/payload/encoder"
)

func JSONBody(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	param, err := RandomHex(seed)
	if err != nil {
		return nil, err
	}
	encodedPayload, err := encoder.Apply("JSUnicode", payload)
	if err != nil {
		return nil, err
	}
	jsonPayload := fmt.Sprintf("{\"test\":true, \"%s\": \"%s\"}", param, encodedPayload)
	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(jsonPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}
