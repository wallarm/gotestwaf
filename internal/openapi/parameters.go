package openapi

import (
	"fmt"
	"math/rand"

	"github.com/getkin/kin-openapi/openapi3"
)

// allParameters contains names of parameters and their value placeholders.
type allParameters struct {
	pathParameters  map[string]string
	queryParameters map[string]string
	headers         map[string]string

	supportedPlaceholders map[string]interface{}
}

// parseParameters returns information about parameters in path, query and headers.
func parseParameters(parameters openapi3.Parameters) (*allParameters, error) {
	pathParams := make(map[string]string)
	queryParams := make(map[string]string)
	headers := make(map[string]string)
	supportedPlaceholders := make(map[string]interface{})

	if parameters != nil {
		for _, p := range parameters {
			switch p.Value.In {
			case openapi3.ParameterInPath:
				param, value, paramType, err := parsePathParameter(p.Value)
				if err != nil {
					return nil, err
				}

				pathParams[param] = value

				if paramType == openapi3.TypeString {
					supportedPlaceholders[urlPathPlaceholder] = nil
				}

			case openapi3.ParameterInQuery:
				param, value, paramType, err := parseQueryParameter(p.Value)
				if err != nil {
					return nil, err
				}

				queryParams[param] = value

				if paramType == openapi3.TypeString {
					supportedPlaceholders[urlParamPlaceholder] = nil
				}

			case openapi3.ParameterInHeader:
				header, value, paramType, err := parseHeaderParameter(p.Value)
				if err != nil {
					return nil, err
				}

				headers[header] = value

				if paramType == openapi3.TypeString {
					supportedPlaceholders[headerPlaceholder] = nil
				}

			default:
				return nil, fmt.Errorf("unsupported parameter place: %s", openapi3.ParameterInCookie)
			}
		}
	}

	params := &allParameters{
		pathParameters:  pathParams,
		queryParameters: queryParams,
		headers:         headers,

		supportedPlaceholders: supportedPlaceholders,
	}

	return params, nil
}

// parseHeaderParameter returns the path parameter name, path parameter value
// placeholder and value type.
func parsePathParameter(parameter *openapi3.Parameter) (paramName, value, paramType string, err error) {
	paramName = parameter.Name

	style := parameter.Style
	if style != "" && style != openapi3.SerializationSimple {
		return "", "", "", fmt.Errorf("unsupported path parameter style: %s", style)
	}

	schema := parameter.Schema.Value
	paramType = schema.Type
	switch schema.Type {
	case openapi3.TypeInteger:
		value = fmt.Sprintf("%d", rand.Uint64())
	case openapi3.TypeString:
		value = parameterStringPlaceholder
	default:
		return "", "", "", fmt.Errorf("unsupported path parameter type: %s", schema.Type)
	}

	return
}

// parseHeaderParameter returns the query parameter name, query parameter value
// placeholder and value type.
func parseQueryParameter(parameter *openapi3.Parameter) (paramName, value, paramType string, err error) {
	paramName = parameter.Name

	style := parameter.Style
	if style != "" &&
		style != openapi3.SerializationForm {
		return "", "", "", fmt.Errorf("unsupported query parameter style: %s", style)
	}

	schema := parameter.Schema.Value
	paramType = schema.Type
	switch schema.Type {
	case openapi3.TypeInteger:
		value = fmt.Sprintf("%d", rand.Uint64())
	case openapi3.TypeString:
		value = parameterStringPlaceholder
	case openapi3.TypeArray:
		items := schema.Items.Value
		paramType = items.Type
		switch items.Type {
		case openapi3.TypeInteger:
			value = fmt.Sprintf("%d", rand.Uint64())
		case openapi3.TypeString:
			value = parameterStringPlaceholder
		default:
			return "", "", "", fmt.Errorf("unsupported type of items in query parameter array: %s", items.Type)
		}
	default:
		return "", "", "", fmt.Errorf("unsupported query parameter type: %s", schema.Type)
	}

	return
}

// parseHeaderParameter returns the header name, header value placeholder and
// value type.
func parseHeaderParameter(parameter *openapi3.Parameter) (paramName, value, paramType string, err error) {
	paramName = parameter.Name

	style := parameter.Style
	if style != "" &&
		style != openapi3.SerializationSimple {
		return "", "", "", fmt.Errorf("unsupported header parameter style: %s", style)
	}

	schema := parameter.Schema.Value
	paramType = schema.Type
	switch schema.Type {
	case openapi3.TypeInteger:
		value = fmt.Sprintf("%d", rand.Uint64())
	case openapi3.TypeString:
		value = headerStringPlaceholder
	default:
		return "", "", "", fmt.Errorf("unsupported header parameter type: %s", schema.Type)
	}

	return
}
