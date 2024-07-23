package placeholder

import (
	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/helpers"
	"github.com/wallarm/gotestwaf/internal/scanner/types"
)

const (
	Seed = 5

	payloadPlaceholder = "{{payload}}"
)

type Placeholder interface {
	NewPlaceholderConfig(conf map[any]any) (PlaceholderConfig, error)

	GetName() string
	CreateRequest(
		requestURL, payload string,
		config PlaceholderConfig,
		httpClientType types.HTTPClientType,
	) (types.Request, error)
}

type PlaceholderConfig interface {
	helpers.Hash
}

var Placeholders map[string]Placeholder

var placeholders = []Placeholder{
	DefaultGraphQL,
	DefaultGRPC,
	DefaultHeader,
	DefaultHTMLForm,
	DefaultHTMLMultipartForm,
	DefaultJSONBody,
	DefaultJSONRequest,
	DefaultRawRequest,
	DefaultRequestBody,
	DefaultSOAPBody,
	DefaultURLParam,
	DefaultURLPath,
	DefaultUserAgent,
	DefaultXMLBody,
}

func init() {
	Placeholders = make(map[string]Placeholder)
	for _, placeholder := range placeholders {
		Placeholders[placeholder.GetName()] = placeholder
	}
}

func GetPlaceholderConfig(name string, conf any) (PlaceholderConfig, error) {
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

	phConf, err := ph.NewPlaceholderConfig(phConfMap)
	if err != nil {
		return nil, &BadPlaceholderConfigError{
			name: name,
			err:  err,
		}
	}

	return phConf, err
}

func Apply(
	url, data, placeholder string,
	config PlaceholderConfig,
	httpClientType types.HTTPClientType,
) (types.Request, error) {
	ph, ok := Placeholders[placeholder]
	if !ok {
		return nil, &UnknownPlaceholderError{name: placeholder}
	}

	req, err := ph.CreateRequest(url, data, config, httpClientType)
	if err != nil {
		return nil, err
	}

	return req, nil
}
