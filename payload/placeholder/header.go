package placeholder

import (
	"net/http"
	"net/url"
)

func Header(requestUrl string, payload string) (*http.Request, error) {
	if reqUrl, err := url.Parse(requestUrl); err != nil {
		return nil, err
	} else {
		randomName, _ := RandomHex(5)
		randomHeader := "X-" + randomName
		if req, err := http.NewRequest("GET", reqUrl.String(), nil); err != nil {
			req.Header.Add(randomHeader, payload)
			return nil, err
		} else {
			return req, nil
		}
	}
}
