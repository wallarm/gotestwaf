package placeholder

import (
	"net/http"
	"net/url"
	"strings"
)

const nonCRUDMethod = "CUST"

type NonCrudUrlPath struct {
	name string
}

type NonCrudUrlParam struct {
	name string
}

type NonCRUDHeader struct {
	name string
}

type NonCRUDRequestBody struct {
	name string
}

var DefaultNonCrudUrlPath = NonCrudUrlPath{name: "NonCrudUrlPath"}
var DefaultNonCrudUrlParam = NonCrudUrlParam{name: "NonCrudUrlParam"}
var DefaultNonCRUDHeader = NonCRUDHeader{name: "NonCRUDHeader"}
var DefaultNonCRUDRequestBody = NonCRUDRequestBody{name: "NonCRUDRequestBody"}

var _ Placeholder = (*NonCrudUrlPath)(nil)
var _ Placeholder = (*NonCrudUrlParam)(nil)
var _ Placeholder = (*NonCRUDHeader)(nil)
var _ Placeholder = (*NonCRUDRequestBody)(nil)

func (p NonCrudUrlPath) GetName() string {
	return p.name
}

func (p NonCrudUrlPath) CreateRequest(requestURL, payload string) (*http.Request, error) {
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

	req, err := http.NewRequest(nonCRUDMethod, urlWithPayload, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (p NonCrudUrlParam) GetName() string {
	return p.name
}

func (p NonCrudUrlParam) CreateRequest(requestURL, payload string) (*http.Request, error) {
	param, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	reqURL.Fragment = ""
	urlWithPayload := reqURL.String()
	if reqURL.RawQuery == "" {
		for i := len(urlWithPayload) - 1; i >= 0; i-- {
			if urlWithPayload[i] != '/' {
				if strings.HasSuffix(reqURL.Path, urlWithPayload[i:]) {
					urlWithPayload = urlWithPayload[:i+1] + "?"
				} else {
					urlWithPayload = urlWithPayload[:i+1] + "/?"
				}
				break
			}
		}
	} else {
		urlWithPayload += "&"
	}
	urlWithPayload += param + "=" + payload

	req, err := http.NewRequest(nonCRUDMethod, urlWithPayload, nil)
	if err != nil {
		return nil, err
	}
	return req, err
}

func (p NonCRUDHeader) GetName() string {
	return p.name
}

func (p NonCRUDHeader) CreateRequest(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	randomName, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	randomHeader := "X-" + randomName
	req, err := http.NewRequest(nonCRUDMethod, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(randomHeader, payload)
	return req, nil
}

func (p NonCRUDRequestBody) GetName() string {
	return p.name
}

func (p NonCRUDRequestBody) CreateRequest(requestURL, payload string) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(nonCRUDMethod, reqURL.String(), strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	return req, nil
}
