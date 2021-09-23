package placeholder

import (
	"net/http"
)

const Seed = 5

type Placeholder interface {
	GetName() string
	CreateRequest(url, data string) (*http.Request, error)
}

var Placeholders map[string]Placeholder

func init() {
	Placeholders = make(map[string]Placeholder)
	Placeholders[DefaultHeader.GetName()] = DefaultHeader
	Placeholders[DefaultRequestBody.GetName()] = DefaultRequestBody
	Placeholders[DefaultSOAPBody.GetName()] = DefaultSOAPBody
	Placeholders[DefaultJSONBody.GetName()] = DefaultJSONBody
	Placeholders[DefaultURLParam.GetName()] = DefaultURLParam
	Placeholders[DefaultURLPath.GetName()] = DefaultURLPath
}

func Apply(host, placeholder, data string) (*http.Request, error) {
	req, err := Placeholders[placeholder].CreateRequest(host, data)
	if err != nil {
		return nil, err
	}

	return req, nil
}
