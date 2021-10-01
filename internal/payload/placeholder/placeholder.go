package placeholder

import (
	"net/http"
	"reflect"
)

const Seed = 5

type Placeholder struct{}

func (p Placeholder) Header(url, data string) (*http.Request, error) {
	return Header(url, data)
}

func (p Placeholder) RequestBody(url, data string) (*http.Request, error) {
	return RequestBody(url, data)
}

func (p Placeholder) SOAPBody(url, data string) (*http.Request, error) {
	return SOAPBody(url, data)
}

func (p Placeholder) JSONBody(url, data string) (*http.Request, error) {
	return JSONBody(url, data)
}

func (p Placeholder) URLParam(url, data string) (*http.Request, error) {
	return URLParam(url, data)
}

func (p Placeholder) URLPath(url, data string) (*http.Request, error) {
	return URLPath(url, data)
}

func Apply(host, placeholder, data string) *http.Request {
	var p Placeholder
	inputs := make([]reflect.Value, 2)
	inputs[0] = reflect.ValueOf(host)
	inputs[1] = reflect.ValueOf(data)
	req := reflect.ValueOf(&p).MethodByName(placeholder).Call(inputs)[0].Interface().(*http.Request)

	return req
}
