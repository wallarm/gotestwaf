package openapi

import (
	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

var (
	headerPlaceholder      string
	urlParamPlaceholder    string
	urlPathPlaceholder     string
	htmlFormPlaceholder    string
	jsonBodyPlaceholder    string
	jsonRequestPlaceholder string
	xmlBodyPlaceholder     string
	requestBodyPlaceholder string
)

func init() {
	headerPlaceholder = placeholder.DefaultHeader.GetName()
	urlParamPlaceholder = placeholder.DefaultURLParam.GetName()
	urlPathPlaceholder = placeholder.DefaultURLPath.GetName()
	jsonBodyPlaceholder = placeholder.DefaultJSONBody.GetName()
	jsonRequestPlaceholder = placeholder.DefaultJSONRequest.GetName()
	htmlFormPlaceholder = placeholder.DefaultHTMLForm.GetName()
	xmlBodyPlaceholder = placeholder.DefaultXMLBody.GetName()
	requestBodyPlaceholder = placeholder.DefaultRequestBody.GetName()
}
