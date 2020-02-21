package placeholder

import (
	"net/http"
	"reflect"
)

type Placeholder struct{}

func (p Placeholder) RequestBody(url string, data string) (*http.Request, error) {
	return RequestBody(url, data)
}

func (p Placeholder) JsonBody(url string, data string) (*http.Request, error) {
	return JsonBody(url, data)
}

func (p Placeholder) UrlParam(url string, data string) (*http.Request, error) {
	return UrlParam(url, data)
}

func (p Placeholder) UrlPath(url string, data string) (*http.Request, error) {
	return UrlPath(url, data)
}

func Apply(host string, placeholder_name string, data string) *http.Request {
	var p Placeholder
	inputs := make([]reflect.Value, 2)
	inputs[0] = reflect.ValueOf(host)
	inputs[1] = reflect.ValueOf(data)
	req := reflect.ValueOf(&p).MethodByName(placeholder_name).Call(inputs)[0].Interface().(*http.Request)

	return req
}
