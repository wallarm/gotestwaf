package placeholder

import (
	"net/http"

	"github.com/pkg/errors"
)

const (
	Seed = 5

	payloadPlaceholder = "{{payload}}"
)

type Placeholder interface {
	newConfig(conf map[any]any) (any, error)

	GetName() string
	CreateRequest(url, data string, config any) (*http.Request, error)
}

var Placeholders map[string]Placeholder

func init() {
	Placeholders = make(map[string]Placeholder)
	Placeholders[DefaultGRPC.GetName()] = DefaultGRPC
	Placeholders[DefaultHeader.GetName()] = DefaultHeader
	Placeholders[DefaultUserAgent.GetName()] = DefaultUserAgent
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
	Placeholders[DefaultRawRequest.GetName()] = DefaultRawRequest
}

func GetPlaceholderConfig(name string, conf any) (any, error) {
	ph, ok := Placeholders[name]
	if !ok {
		return nil, &UnknownPlaceholderError{name: name}
	}

	phConfMap, ok := conf.(map[any]any)
	if !ok {
		return nil, &BadPlaceholderConfigError{
			name: name,
			err:  errors.Errorf("bad placeholder config, expected: map[any]any, got: %T", conf),
		}
	}

	phConf, err := ph.newConfig(phConfMap)
	if err != nil {
		return nil, &BadPlaceholderConfigError{
			name: name,
			err:  err,
		}
	}

	return phConf, err
}

func Apply(url, data, placeholder string, config any) (*http.Request, error) {
	ph, ok := Placeholders[placeholder]
	if !ok {
		return nil, &UnknownPlaceholderError{name: placeholder}
	}

	req, err := ph.CreateRequest(url, data, config)
	if err != nil {
		return nil, err
	}

	return req, nil
}
