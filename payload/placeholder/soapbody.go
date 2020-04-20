package placeholder

import (
	"fmt"
	"gotestwaf/payload/encoder"
	"net/http"
	"net/url"
	"strings"
)

func SoapBody(requestUrl string, payload string) (*http.Request, error) {
	if reqUrl, err := url.Parse(requestUrl); err != nil {
		return nil, err
	} else {
		param, _ := RandomHex(5)
		encodedPayload, _ := encoder.Apply("XmlEntity", payload)
		soapPayload := fmt.Sprintf(`
      <?xml version="1.0" encoding="UTF-8"?>
      <soapenv:Envelope
              xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
              xmlns:xsd="http://www.w3.org/2001/XMLSchema"
              xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
        <soapenv:Header>
          <ns1:RequestHeader
               soapenv:actor="http://schemas.xmlsoap.org/soap/actor/next"
               soapenv:mustUnderstand="0"
               xmlns:ns1="https://www.google.com/apis/ads/publisher/v202002">
          </ns1:RequestHeader>
        </soapenv:Header>
        <soapenv:Body>
          <getAdUnitsByStatement xmlns="https://www.google.com/apis/ads/publisher/v202002">
            <filterStatement>
              <%s>%s</%s>
            </filterStatement>
          </getAdUnitsByStatement>
        </soapenv:Body>
      </soapenv:Envelope>
      `, param, encodedPayload, param)
		//reqUrl.Path = fmt.Sprintf("%s/%s/", reqUrl.Path, payload)
		if req, err := http.NewRequest("POST", reqUrl.String(), strings.NewReader(soapPayload)); err != nil {
			return nil, err
		} else {
			req.Header.Add("SOAPAction", "\"http://schemas.xmlsoap.org/soap/actor/next\"")
			req.Header.Add("Content-Type", "text/xml")
			return req, nil
		}
	}
}
