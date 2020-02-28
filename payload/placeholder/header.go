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
			return nil, err
		} else {
			req.Header.Add(randomHeader, payload)
			return req, nil
		}
	}
}
