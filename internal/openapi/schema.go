package openapi

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	jsonContentType      = "application/json"
	xmlContentType       = "application/xml"
	xWwwFormContentType  = "application/x-www-form-urlencoded"
	plainTextContentType = "text/plain"
	anyContentType       = "*/*"
)

// schemaToMap converts openapi3.Schema to value or map[string]interface{}.
func schemaToMap(schema *openapi3.Schema) (
	value interface{},
	strAvailable bool,
	paramSpec map[string]*parameterSpec,
	err error,
) {
	strAvailable = false

	switch schema.Type {
	case openapi3.TypeInteger:
		randInt := genRandomInt(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		value = fmt.Sprintf("%d", randInt)

	case openapi3.TypeNumber:
		randFloat := genRandomFloat(schema.Min, schema.Max, schema.ExclusiveMin, schema.ExclusiveMax)
		value = fmt.Sprintf("%f", randFloat)

	case openapi3.TypeString:
		value = genRandomPlaceholder()
		strAvailable = true

		spec := &parameterSpec{}
		spec.paramType = schema.Type
		spec.minLength = schema.MinLength
		if schema.MaxLength == nil {
			spec.maxLength = math.MaxUint64
			spec.value = genRandomString(spec.minLength, spec.minLength+defaultStringSize)
		} else {
			spec.maxLength = *schema.MaxLength
			spec.value = genRandomString(spec.minLength, spec.maxLength)
		}

		paramSpec = make(map[string]*parameterSpec)
		paramSpec[value.(string)] = spec

	case openapi3.TypeBoolean:
		value = "false"

	case openapi3.TypeArray:
		inner, innerStrAvailable, innerParamSpec, err := schemaToMap(schema.Items.Value)
		if err != nil {
			return nil, false, nil, err
		}

		minArrayLength := int(schema.MinLength)

		v := make([]interface{}, minArrayLength)
		for i := 0; i < minArrayLength; i++ {
			v[i] = inner
		}

		return v, innerStrAvailable, innerParamSpec, nil

	case openapi3.TypeObject:
		paramSpec = make(map[string]*parameterSpec)
		mapStructure := make(map[string]interface{})

		for name, obj := range schema.Properties {
			inner, innerStrAvailable, innerParamSpec, err := schemaToMap(obj.Value)
			if err != nil {
				return nil, false, nil, err
			}

			strAvailable = strAvailable || innerStrAvailable
			mapStructure[name] = inner

			for k, v := range innerParamSpec {
				paramSpec[k] = v
			}
		}

		return mapStructure, strAvailable, paramSpec, nil
	}

	return value, strAvailable, paramSpec, nil
}

// jsonMarshal dumps structure as JSON.
func jsonMarshal(schemaStructure interface{}) (string, error) {
	byteString, err := json.Marshal(schemaStructure)
	if err != nil {
		return "", err
	}

	return string(byteString), nil
}

// xmlMarshal dumps structure as XML.
func xmlMarshal(schemaStructure interface{}) (string, error) {
	byteString, err := xml.Marshal(schemaStructure)
	if err != nil {
		return "", err
	}

	return string(byteString), nil
}

// htmlFormMarshal dumps structure as HTML Form.
func htmlFormMarshal(schemaStructure interface{}) (string, error) {
	object, ok := schemaStructure.(map[string]string)
	if !ok {
		return "", fmt.Errorf("input value must be map[string]string")
	}

	var parts []string

	for k, v := range object {
		parts = append(parts, k+"="+v)
	}

	return strings.Join(parts, "&"), nil
}
