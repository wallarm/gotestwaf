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
	Placeholders[DefaultGRPC.GetName()] = DefaultGRPC
	Placeholders[DefaultHeader.GetName()] = DefaultHeader
	Placeholders[DefaultHTMLForm.GetName()] = DefaultHTMLForm
	Placeholders[DefaultHTMLMultipartForm.GetName()] = DefaultHTMLMultipartForm
	Placeholders[DefaultJSONBody.GetName()] = DefaultJSONBody
	Placeholders[DefaultJSONRequest.GetName()] = DefaultJSONRequest
	Placeholders[DefaultRequestBody.GetName()] = DefaultRequestBody
	Placeholders[DefaultSOAPBody.GetName()] = DefaultSOAPBody
	Placeholders[DefaultURLParam.GetName()] = DefaultURLParam
	Placeholders[DefaultURLPath.GetName()] = DefaultURLPath
	Placeholders[DefaultXMLBody.GetName()] = DefaultXMLBody
	Placeholders[DefaultNonCrudUrlPath.GetName()] = DefaultNonCrudUrlPath
	Placeholders[DefaultNonCrudUrlParam.GetName()] = DefaultNonCrudUrlParam
	Placeholders[DefaultNonCRUDHeader.GetName()] = DefaultNonCRUDHeader
	Placeholders[DefaultNonCRUDRequestBody.GetName()] = DefaultNonCRUDRequestBody
}

func Apply(host, placeholder, data string) (*http.Request, error) {
	req, err := Placeholders[placeholder].CreateRequest(host, data)
	if err != nil {
		return nil, err
	}

	return req, nil
}
