package openapi

import (
	"fmt"
	"math"

	"github.com/getkin/kin-openapi/openapi3"
)

const defaultPlaceholderSize = 16
const defaultMaxInt = 10000
const defaultStringSize = 16

// allParameters contains names of parameters and their value placeholders.
type allParameters struct {
	pathParameters  map[string]*parameterSpec
	queryParameters map[string]*parameterSpec
	headers         map[string]*parameterSpec

	supportedPlaceholders map[string]interface{}
}

// parameterSpec contains a specific value for any parameter and
// length limits for string parameters.
type parameterSpec struct {
	paramType string
	value     string
	minLength uint64
	maxLength uint64
}

// parseParameters returns information about parameters in path, query and headers.
func parseParameters(parameters openapi3.Parameters) (*allParameters, error) {
	pathParams := make(map[string]*parameterSpec)
	queryParams := make(map[string]*parameterSpec)
	headers := make(map[string]*parameterSpec)
	supportedPlaceholders := make(map[string]interface{})

	if parameters != nil {
		for _, p := range parameters {
			switch p.Value.In {
			case openapi3.ParameterInPath:
				param, spec, err := parsePathParameter(p.Value)
				if err != nil {
					return nil, err
				}

				pathParams[param] = spec

				if spec.paramType == openapi3.TypeString {
					supportedPlaceholders[urlPathPlaceholder] = nil
				}

			case openapi3.ParameterInQuery:
				param, spec, err := parseQueryParameter(p.Value)
				if err != nil {
					return nil, err
				}

				queryParams[param] = spec

				if spec.paramType == openapi3.TypeString {
					supportedPlaceholders[urlParamPlaceholder] = nil
				}

			case openapi3.ParameterInHeader:
				header, spec, err := parseHeaderParameter(p.Value)
				if err != nil {
					return nil, err
				}

				headers[header] = spec

				if spec.paramType == openapi3.TypeString {
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
func parsePathParameter(parameter *openapi3.Parameter) (paramName string, spec *parameterSpec, err error) {
	paramName = "{" + parameter.Name + "}"
	spec = &parameterSpec{}

	style := parameter.Style
	if style != "" && style != openapi3.SerializationSimple {
		return "", nil, fmt.Errorf("unsupported path parameter style: %s", style)
	}

	schema := parameter.Schema.Value
	spec.paramType = schema.Type
	switch schema.Type {
	case openapi3.TypeInteger:
		randInt := genRandomInt(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		spec.value = fmt.Sprintf("%d", randInt)

	case openapi3.TypeString:
		spec.minLength = schema.MinLength
		if schema.MaxLength == nil {
			spec.maxLength = math.MaxUint64
			spec.value = genRandomString(spec.minLength, spec.minLength+defaultStringSize)
		} else {
			spec.maxLength = *schema.MaxLength
			spec.value = genRandomString(spec.minLength, spec.maxLength)
		}

	default:
		return "", nil, fmt.Errorf("unsupported path parameter type: %s", schema.Type)
	}

	return
}

// parseHeaderParameter returns the query parameter name, query parameter value
// placeholder and value type.
func parseQueryParameter(parameter *openapi3.Parameter) (paramName string, spec *parameterSpec, err error) {
	paramName = parameter.Name
	spec = &parameterSpec{}

	style := parameter.Style
	if style != "" &&
		style != openapi3.SerializationForm {
		return "", nil, fmt.Errorf("unsupported query parameter style: %s", style)
	}

	schema := parameter.Schema.Value
	spec.paramType = schema.Type
	switch schema.Type {
	case openapi3.TypeInteger:
		randInt := genRandomInt(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		spec.value = fmt.Sprintf("%d", randInt)

	case openapi3.TypeString:
		spec.minLength = schema.MinLength
		if schema.MaxLength == nil {
			spec.maxLength = math.MaxUint64
			spec.value = genRandomString(spec.minLength, spec.minLength+defaultStringSize)
		} else {
			spec.maxLength = *schema.MaxLength
			spec.value = genRandomString(spec.minLength, spec.maxLength)
		}

	case openapi3.TypeArray:
		items := schema.Items.Value
		spec.paramType = items.Type

		switch items.Type {
		case openapi3.TypeInteger:
			randInt := genRandomInt(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
			spec.value = fmt.Sprintf("%d", randInt)

		case openapi3.TypeString:
			spec.minLength = schema.MinLength
			if schema.MaxLength == nil {
				spec.maxLength = math.MaxUint64
				spec.value = genRandomString(spec.minLength, spec.minLength+defaultStringSize)
			} else {
				spec.maxLength = *schema.MaxLength
				spec.value = genRandomString(spec.minLength, spec.maxLength)
			}

		default:
			return "", nil, fmt.Errorf("unsupported type of items in query parameter array: %s", items.Type)
		}

	default:
		return "", nil, fmt.Errorf("unsupported query parameter type: %s", schema.Type)
	}

	return
}

// parseHeaderParameter returns the header name, header value placeholder and
// value type.
func parseHeaderParameter(parameter *openapi3.Parameter) (paramName string, spec *parameterSpec, err error) {
	paramName = parameter.Name
	spec = &parameterSpec{}

	style := parameter.Style
	if style != "" &&
		style != openapi3.SerializationSimple {
		return "", nil, fmt.Errorf("unsupported header parameter style: %s", style)
	}

	schema := parameter.Schema.Value
	spec.paramType = schema.Type
	switch schema.Type {
	case openapi3.TypeInteger:
		randInt := genRandomInt(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		spec.value = fmt.Sprintf("%d", randInt)

	case openapi3.TypeString:
		spec.minLength = schema.MinLength
		if schema.MaxLength == nil {
			spec.maxLength = math.MaxUint64
			spec.value = genRandomString(spec.minLength, spec.minLength+defaultStringSize)
		} else {
			spec.maxLength = *schema.MaxLength
			spec.value = genRandomString(spec.minLength, spec.maxLength)
		}

	default:
		return "", nil, fmt.Errorf("unsupported header parameter type: %s", schema.Type)
	}

	return
}
