package openapi

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pkg/errors"
)

const (
	defaultPlaceholderSize = 16
	defaultMaxInt          = 10000
	defaultStringSize      = 16

	defaultQueryParameterArrayBinder = ","
	spaceQueryParameterArrayBinder   = "%20"
	pipeQueryParameterArrayBinder    = "|"
)

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
	explode   bool
	paramSpec map[string]*parameterSpec
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
	case openapi3.TypeNumber:
		randFloat := genRandomFloat(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		spec.value = fmt.Sprintf("%f", randFloat)

	case openapi3.TypeInteger:
		randInt := genRandomInt(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		spec.value = fmt.Sprintf("%d", randInt)

	case openapi3.TypeString:
		spec.minLength = schema.MinLength
		if schema.MaxLength == nil {
			spec.maxLength = math.MaxUint64
		} else {
			spec.maxLength = *schema.MaxLength
		}
		spec.value = genRandomString(spec.minLength, spec.minLength+defaultStringSize)

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
	if style == "" {
		style = openapi3.SerializationForm
	}
	if style != openapi3.SerializationForm &&
		style != openapi3.SerializationSpaceDelimited &&
		style != openapi3.SerializationPipeDelimited &&
		style != openapi3.SerializationDeepObject {
		return "", nil, fmt.Errorf("unsupported query parameter style: %s", style)
	}

	var schema *openapi3.Schema
	var isJSON bool

	if parameter.Schema != nil {
		schema = parameter.Schema.Value
	} else if parameter.Content != nil {
		if _, ok := parameter.Content[jsonContentType]; !ok {
			return "", nil, fmt.Errorf("unsupported content type in content of query parameter specification")
		}
		schema = parameter.Content[jsonContentType].Schema.Value
		isJSON = true
	} else {
		return "", nil, fmt.Errorf("neither schema nor content not found in query parameter specification")
	}

	spec.paramType = schema.Type
	switch schema.Type {
	case openapi3.TypeNumber:
		randFloat := genRandomFloat(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		spec.value = fmt.Sprintf("%f", randFloat)

	case openapi3.TypeInteger:
		randInt := genRandomInt(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		spec.value = fmt.Sprintf("%d", randInt)

	case openapi3.TypeString:
		spec.minLength = schema.MinLength
		if schema.MaxLength == nil {
			spec.maxLength = math.MaxUint64
		} else {
			spec.maxLength = *schema.MaxLength
		}
		spec.value = genRandomString(spec.minLength, spec.minLength+defaultStringSize)

	case openapi3.TypeArray:
		items := schema.Items.Value
		spec.paramType = items.Type

		switch items.Type {
		case openapi3.TypeNumber:
			randFloat := genRandomFloat(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
			spec.value = fmt.Sprintf("%f", randFloat)

		case openapi3.TypeInteger:
			randInt := genRandomInt(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
			spec.value = fmt.Sprintf("%d", randInt)

		case openapi3.TypeString:
			spec.minLength = schema.MinLength
			if schema.MaxLength == nil {
				spec.maxLength = math.MaxUint64
			} else {
				spec.maxLength = *schema.MaxLength
			}
			spec.value = genRandomString(spec.minLength, spec.minLength+defaultStringSize)

		default:
			return "", nil, fmt.Errorf("unsupported type of items in query parameter array: %s", items.Type)
		}

		if schema.MinItems > 1 {
			if parameter.Explode != nil {
				spec.explode = *parameter.Explode
			}

			items := []string{spec.value}

			for i := schema.MinItems; i > 1; i-- {
				items = append(items, spec.value)
			}

			if spec.explode {
				prefix := paramName + "="
				binder := "&" + paramName + "="

				// spec.value = "paramName=item1&paramName=item2&paramName=item3"
				spec.value = prefix + strings.Join(items, binder)
			} else {
				binder := defaultQueryParameterArrayBinder
				if style == openapi3.SerializationSpaceDelimited {
					binder = spaceQueryParameterArrayBinder
				} else if style == openapi3.SerializationPipeDelimited {
					binder = pipeQueryParameterArrayBinder
				}

				spec.value = strings.Join(items, binder)
			}
		}

	case openapi3.TypeObject:
		value, strAvailable, paramSpec, err := schemaToMap("", schema, false)
		if err != nil {
			return "", nil, errors.Wrap(err, "couldn't parse query parameter object")
		}

		if isJSON {
			jsonValue, err := jsonMarshal(value)
			if err != nil {
				return "", nil, errors.Wrap(err, "couldn't marshal query parameter object to JSON")
			}
			spec.value = url.QueryEscape(jsonValue)
		} else {
			parts := queryParamStructParts(paramName, value)
			spec.value = strings.Join(parts, "&")
			spec.explode = true
		}

		if strAvailable {
			spec.paramSpec = paramSpec
		}

	case openapi3.TypeBoolean:
		spec.value = "false"

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

	var schema *openapi3.Schema
	var isJSON bool

	if parameter.Schema != nil {
		schema = parameter.Schema.Value
	} else if parameter.Content != nil {
		if _, ok := parameter.Content[jsonContentType]; !ok {
			return "", nil, fmt.Errorf("unsupported content type in content of header specification")
		}
		schema = parameter.Content[jsonContentType].Schema.Value
		isJSON = true
	} else {
		return "", nil, fmt.Errorf("neither schema nor content not found in header specification")
	}

	spec.paramType = schema.Type
	switch schema.Type {
	case openapi3.TypeNumber:
		randFloat := genRandomFloat(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		spec.value = fmt.Sprintf("%f", randFloat)

	case openapi3.TypeInteger:
		randInt := genRandomInt(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		spec.value = fmt.Sprintf("%d", randInt)

	case openapi3.TypeString:
		spec.minLength = schema.MinLength
		if schema.MaxLength == nil {
			spec.maxLength = math.MaxUint64
		} else {
			spec.maxLength = *schema.MaxLength
		}
		spec.value = genRandomString(spec.minLength, spec.minLength+defaultStringSize)

	case openapi3.TypeObject:
		value, strAvailable, paramSpec, err := schemaToMap("", schema, false)
		if err != nil {
			return "", nil, errors.Wrap(err, "couldn't parse header object")
		}

		if isJSON {
			jsonValue, err := jsonMarshal(value)
			if err != nil {
				return "", nil, errors.Wrap(err, "couldn't marshal header object to JSON")
			}
			spec.value = url.QueryEscape(jsonValue)
		} else {
			return "", nil, fmt.Errorf("unsupported content type in content of header specification")
		}

		if strAvailable {
			spec.paramSpec = paramSpec
		}

	default:
		return "", nil, fmt.Errorf("unsupported header parameter type: %s", schema.Type)
	}

	return
}

func queryParamStructParts(paramName string, queryParamStruct interface{}) []string {
	var parts []string

	part := paramName

	switch v := queryParamStruct.(type) {
	case string:
		part += "=" + v
		parts = append(parts, part)

	case []interface{}:
		for n, item := range v {
			part := fmt.Sprintf("%s[%d]", part, n)
			parts = append(parts, queryParamStructParts(part, item)...)
		}

	case map[string]interface{}:
		for k, v := range v {
			part := fmt.Sprintf("%s[%s]", part, k)

			parts = append(parts, queryParamStructParts(part, v)...)
		}
	}

	return parts
}
