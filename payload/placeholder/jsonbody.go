package placeholder

import (
	"fmt"
	"gotestwaf/payload/encoder"
	"net/http"
	"net/url"
	"strings"
)

func JsonBody(requestUrl string, payload string) (*http.Request, error) {
	if reqUrl, err := url.Parse(requestUrl); err != nil {
		return nil, err
	} else {
		param, _ := RandomHex(5)
		encodedPayload, _ := encoder.Apply("JSUnicode", payload)
		jsonPayload := fmt.Sprintf("{\"test\":true, \"%s\": \"%s\"}", param, encodedPayload)
		//reqUrl.Path = fmt.Sprintf("%s/%s/", reqUrl.Path, payload)
		if req, err := http.NewRequest("POST", reqUrl.String(), strings.NewReader(jsonPayload)); err != nil {
			return nil, err
		} else {
			req.Header.Add("Content-Type", "application/json")
			return req, nil
		}
	}
}
