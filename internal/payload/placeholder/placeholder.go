package placeholder

import (
	"fmt"
	"net/http"
)

const Seed = 5

type Placeholder interface {
	GetName() string
	CreateRequest(url, data string) (*http.Request, error)
}

var Placeholders map[string]Placeholder

type UnknownPlaceholderError struct {
	name string
}

func (e *UnknownPlaceholderError) Error() string {
	return fmt.Sprintf("unknown placeholder: %s", e.name)
}

var _ error = (*UnknownPlaceholderError)(nil)

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
	ph, ok := Placeholders[placeholder]
	if !ok {
		return nil, &UnknownPlaceholderError{name: placeholder}
	}

	req, err := ph.CreateRequest(host, data)
	if err != nil {
		return nil, err
	}

	return req, nil
}
