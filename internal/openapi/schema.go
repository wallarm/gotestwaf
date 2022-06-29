package openapi

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/clbanning/mxj"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pkg/errors"
)

const (
	jsonContentType      = "application/json"
	xmlContentType       = "application/xml"
	xWwwFormContentType  = "application/x-www-form-urlencoded"
	plainTextContentType = "text/plain"

	xmlAttributePrefix = "-"
	xmlHeader          = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"
)

// schemaToMap converts openapi3.Schema to value or map[string]interface{}.
func schemaToMap(name string, schema *openapi3.Schema, isXML bool) (
	value interface{},
	strAvailable bool,
	paramSpec map[string]*parameterSpec,
	err error,
) {
	var allOf openapi3.SchemaRefs
	if schema.AnyOf != nil {
		allOf = schema.AnyOf
	}
	if schema.AllOf != nil {
		allOf = schema.AllOf
	}
	if allOf != nil {
		allOfValue := make(map[string]interface{})
		allOfParamSpec := make(map[string]*parameterSpec)
		allOfStrAvailable := false

		for _, schemaRef := range allOf {
			if schemaRef != nil && schemaRef.Value != nil {
				innerSchema := schemaRef.Value

				innerValue, innerStrAvailable, innerParamSpec, innerErr := schemaToMap(name, innerSchema, isXML)
				if innerErr != nil {
					return nil, false, nil, errors.Wrap(innerErr, "couldn't parse allOf/anyOf")
				}

				innerMap, ok := innerValue.(map[string]interface{})
				if !ok {
					return nil, false, nil, errors.New("unsupported object in allOf/anyOf")
				}

				for k, v := range innerMap {
					allOfValue[k] = v
				}

				for k, v := range innerParamSpec {
					allOfParamSpec[k] = v
				}

				allOfStrAvailable = allOfStrAvailable || innerStrAvailable
			}
		}

		return allOfValue, allOfStrAvailable, allOfParamSpec, nil
	}

	if schema.OneOf != nil {
		for _, schemaRef := range schema.OneOf {
			if schemaRef != nil && schemaRef.Value != nil {
				innerSchema := schemaRef.Value

				innerValue, innerStrAvailable, innerParamSpec, innerErr := schemaToMap(name, innerSchema, isXML)
				if innerErr != nil {
					return nil, false, nil, errors.Wrap(innerErr, "couldn't parse oneOf")
				}

				innerMap, ok := innerValue.(map[string]interface{})
				if !ok {
					return nil, false, nil, errors.New("unsupported object in oneOf")
				}

				if innerStrAvailable && len(innerParamSpec) > len(paramSpec) {
					value = innerMap
					paramSpec = innerParamSpec
					strAvailable = innerStrAvailable
				}
			}
		}

		return
	}

	strAvailable = false

	if isXML && schema.XML != nil {
		if schema.XML.Name != "" {
			name = schema.XML.Name
		}

		if name != "" {
			if schema.XML.Attribute {
				name = xmlAttributePrefix + name
			} else if schema.XML.Prefix != "" {
				name = schema.XML.Prefix + ":" + name
			}
		}
	}

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
		inner, innerStrAvailable, innerParamSpec, err := schemaToMap(name, schema.Items.Value, isXML)
		if err != nil {
			return nil, false, nil, errors.Wrap(err, "couldn't parse array")
		}

		minArrayLength := int(schema.MinLength)
		if minArrayLength == 0 {
			minArrayLength = 1
		}

		v := make([]interface{}, minArrayLength)
		for i := 0; i < minArrayLength; i++ {
			v[i] = inner
		}

		value = v
		strAvailable = innerStrAvailable
		paramSpec = innerParamSpec

	case openapi3.TypeObject:
		paramSpec = make(map[string]*parameterSpec)
		mapStructure := make(map[string]interface{})

		for name, obj := range schema.Properties {
			inner, innerStrAvailable, innerParamSpec, err := schemaToMap(name, obj.Value, isXML)
			if err != nil {
				return nil, false, nil, errors.Wrap(err, "couldn't parse object")
			}

			strAvailable = strAvailable || innerStrAvailable

			innerMap, isInnerMap := inner.(map[string]interface{})
			if isXML && isInnerMap {
				for k, v := range innerMap {
					mapStructure[k] = v
				}
			} else {
				mapStructure[name] = inner
			}

			for k, v := range innerParamSpec {
				paramSpec[k] = v
			}
		}

		value = mapStructure

	default:
		return nil, false, nil, fmt.Errorf("unknown schema type: %s", schema.Type)
	}

	if isXML {
		var wrappedValue map[string]interface{}

		if mapValue, ok := value.(map[string]interface{}); ok {
			wrappedValue = mapValue
		} else {
			wrappedValue = make(map[string]interface{})
			wrappedValue["#text"] = value
			wrappedValue["#seq"] = 0
		}

		if schema.XML != nil && schema.XML.Namespace != "" {
			xmlns := "xmlns"
			if schema.XML.Prefix != "" {
				xmlns = xmlns + ":" + schema.XML.Prefix
			}

			wrappedValue["#attr"] = map[string]interface{}{
				xmlns: map[string]interface{}{
					"#text": schema.XML.Namespace,
					"#seq":  0,
				},
			}
		}

		value = wrappedValue

		if name != "" {
			value = map[string]interface{}{
				name: value,
			}
		}
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
	object, ok := schemaStructure.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("input value must be map[string]interface{}")
	}

	m := mxj.Map(object)

	byteString, err := m.XmlSeq()
	if err != nil {
		return "", errors.Wrap(err, "couldn't marshall object to XML")
	}

	return xmlHeader + string(byteString), nil
}

// htmlFormMarshal dumps structure as HTML Form.
func htmlFormMarshal(schemaStructure interface{}) (string, error) {
	object, ok := schemaStructure.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("input value must be map[string]interface{}")
	}

	var parts []string

	var str string
	var err error

	for k, v := range object {
		if strValue, isStr := v.(string); isStr {
			str = strValue
		} else {
			str, err = jsonMarshal(v)
			if err != nil {
				return "", errors.Wrap(err, "couldn't marshall object to JSON in HTML form field")
			}
			str = url.QueryEscape(str)
		}

		parts = append(parts, k+"="+str)
	}

	return strings.Join(parts, "&"), nil
}
