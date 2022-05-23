package openapi

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/wallarm/gotestwaf/internal/payload/placeholder"
)

var (
	pathStringPlaceholder      string
	parameterStringPlaceholder string
	headerStringPlaceholder    string
	bodyStringPlaceholder      string

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
	var randValue [16]byte

	_, _ = rand.Read(randValue[:])
	pathStringPlaceholder = hex.EncodeToString(randValue[:])
	_, _ = rand.Read(randValue[:])
	parameterStringPlaceholder = hex.EncodeToString(randValue[:])
	_, _ = rand.Read(randValue[:])
	headerStringPlaceholder = hex.EncodeToString(randValue[:])
	_, _ = rand.Read(randValue[:])
	bodyStringPlaceholder = hex.EncodeToString(randValue[:])

	headerPlaceholder = placeholder.DefaultHeader.GetName()
	urlParamPlaceholder = placeholder.DefaultURLParam.GetName()
	urlPathPlaceholder = placeholder.DefaultURLPath.GetName()
	jsonBodyPlaceholder = placeholder.DefaultJSONBody.GetName()
	jsonRequestPlaceholder = placeholder.DefaultJSONRequest.GetName()
	htmlFormPlaceholder = placeholder.DefaultHTMLForm.GetName()
	xmlBodyPlaceholder = placeholder.DefaultXMLBody.GetName()
	requestBodyPlaceholder = placeholder.DefaultRequestBody.GetName()
}
