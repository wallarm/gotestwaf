package placeholder

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/wallarm/gotestwaf/internal/payload/encoder"
)

type SOAPBody struct {
	name string
}

var DefaultSOAPBody = SOAPBody{name: "SOAPBody"}

var _ Placeholder = (*SOAPBody)(nil)

func (p SOAPBody) newConfig(_ map[any]any) (any, error) {
	return nil, nil
}

func (p SOAPBody) GetName() string {
	return p.name
}

func (p SOAPBody) CreateRequest(requestURL, payload string, _ any) (*http.Request, error) {
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}

	param, err := RandomHex(Seed)
	if err != nil {
		return nil, err
	}

	param = "ab" + param

	encodedPayload, err := encoder.Apply("XMLEntity", payload)
	if err != nil {
		return nil, err
	}

	soapPayload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
	xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
	<soapenv:Header>
		<ns1:RequestHeader soapenv:actor="http://schemas.xmlsoap.org/soap/actor/next"
               soapenv:mustUnderstand="0" xmlns:ns1="https://www.google.com/apis/ads/publisher/v202002">
		</ns1:RequestHeader>
	</soapenv:Header>
	<soapenv:Body>
		<getAdUnitsByStatement xmlns="https://www.google.com/apis/ads/publisher/v202002">
			<filterStatement>
				<%s>%s</%s>
			</filterStatement>
		</getAdUnitsByStatement>
	</soapenv:Body>
</soapenv:Envelope>`, param, encodedPayload, param)
	req, err := http.NewRequest("POST", reqURL.String(), strings.NewReader(soapPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Add("SOAPAction", "\"http://schemas.xmlsoap.org/soap/actor/next\"")
	req.Header.Add("Content-Type", "text/xml")
	return req, nil
}
