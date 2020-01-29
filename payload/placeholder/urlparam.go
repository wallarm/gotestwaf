package placeholder

import (
	"log"
	"net/http"
)

// Warning: this placeholder encodes URL anyways
func UrlParam(requestUrl string, payload string) *http.Request {
	param, _ := RandomHex(5)
	req, err := http.NewRequest("GET", requestUrl+"/?"+param+"="+payload, nil)
	if err != nil {
		log.Fatal(err)
	}
	return req
}
