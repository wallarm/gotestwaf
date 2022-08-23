package placeholder

import (
	"net/http"
	"net/url"
	"strings"
)

type XMLBody struct {
	name string
}

var DefaultXMLBody = XMLBody{name: "XMLBody"}

var _ Placeholder = (*XMLBody)(nil)

func (p XMLBody) GetName() string {
	return p.name
}

func (p XMLBody) CreateRequest(requestURL, payload string) (*http.Request, error) {
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
