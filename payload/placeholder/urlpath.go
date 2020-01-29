package placeholder

import (
	"fmt"
	"net/http"
	"net/url"
)

func UrlPath(requestUrl string, payload string) (*http.Request, error) {
	if reqUrl, err := url.Parse(requestUrl); err != nil {
		return nil, err
	} else {
		reqUrl.Path = fmt.Sprintf("%s/%s/", reqUrl.Path, url.QueryEscape(payload))
		if req, err := http.NewRequest("GET", reqUrl.String(), nil); err != nil {
			return nil, err
		} else {
			return req, nil
		}
	}
}
