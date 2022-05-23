package openapi

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math/rand"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

var (
	jsonContentType      = "application/json"
	xmlContentType       = "application/xml"
	xWwwFormContentType  = "application/x-www-form-urlencoded"
	plainTextContentType = "text/plain"
	anyContentType       = "*/*"
)

// schemaToMap converts openapi3.Schema to value or map[string]interface{}.
func schemaToMap(schema *openapi3.Schema) (value interface{}, strAvailable bool, err error) {
	strAvailable = false

	switch schema.Type {
	case openapi3.TypeInteger:
		value = fmt.Sprintf("%d", rand.Uint64())

	case openapi3.TypeNumber:
		value = fmt.Sprintf("%f", rand.Float64())

	case openapi3.TypeString:
		value = bodyStringPlaceholder
		strAvailable = true

	case openapi3.TypeBoolean:
		value = "false"

	case openapi3.TypeArray:
		inner, innerStrAvailable, err := schemaToMap(schema.Items.Value)
		if err != nil {
			return nil, false, err
		}

		strAvailable = strAvailable || innerStrAvailable
		v := make([]interface{}, 1)
		v[0] = inner

		return v, strAvailable, nil

	case openapi3.TypeObject:
		mapStructure := make(map[string]interface{})

		for name, obj := range schema.Properties {
			inner, innerStrAvailable, err := schemaToMap(obj.Value)
			if err != nil {
				return nil, false, err
			}

			strAvailable = strAvailable || innerStrAvailable
			mapStructure[name] = inner
		}

		return mapStructure, strAvailable, nil
	}

	return value, strAvailable, nil
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
