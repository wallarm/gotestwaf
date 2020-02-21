package placeholder

import (
	"net/http"
	"net/url"
	"strings"
)

func RequestBody(requestUrl string, payload string) (*http.Request, error) {
	if reqUrl, err := url.Parse(requestUrl); err != nil {
		return nil, err
	} else {
		//check if we need to set Content-Lenght manually here
		if req, err := http.NewRequest("POST", reqUrl.String(), strings.NewReader(payload)); err != nil {
			return nil, err
		} else {
			return req, nil
		}
	}
}
