package openapi

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/pkg/errors"
)

// Templates contains all templates generated from OpenAPI file. Templates are
// sorted by placeholders that can be used to substitute a malicious vector.
type Templates map[string][]*Template

// Template contains all information about template.
type Template struct {
	Method string
	Path   string
	URL    string

	PathParameters        map[string]*parameterSpec
	QueryParameters       map[string]*parameterSpec
	Headers               map[string]*parameterSpec
	RequestBodyParameters map[string]*parameterSpec

	RequestBody map[string]string

	Doc *openapi3.T

	Placeholders map[string]interface{}
}

// NewTemplates parses OpenAPI document and returns all possible templates.
func NewTemplates(openapiDoc *openapi3.T, basePath string) (Templates, error) {
	var unsortedTemplates []*Template
	for path, info := range openapiDoc.Paths {
		pathTemplates, err := pathTemplates(openapiDoc, basePath, path, info)
		if err != nil {
			return nil, err
		}
		unsortedTemplates = append(unsortedTemplates, pathTemplates...)
	}

	templates := make(Templates)

	for _, template := range unsortedTemplates {
		for placeholder, _ := range template.Placeholders {
			if templates[placeholder] == nil {
				templates[placeholder] = make([]*Template, 0)
			}
			templates[placeholder] = append(templates[placeholder], template)
		}
	}

	return templates, nil
}

// pathTemplates parses every path in OpenAPI document.
func pathTemplates(openapiDoc *openapi3.T, basePath string, path string, pathInfo *openapi3.PathItem) ([]*Template, error) {
	var templates []*Template

	if pathInfo.Connect != nil {
		operationTemplate, err := operationTemplates(openapiDoc, basePath, path, http.MethodConnect, pathInfo.Connect)
		if err != nil {
			return nil, err
		}
		templates = append(templates, operationTemplate)
	}
	if pathInfo.Delete != nil {
		operationTemplate, err := operationTemplates(openapiDoc, basePath, path, http.MethodDelete, pathInfo.Delete)
		if err != nil {
			return nil, err
		}
		templates = append(templates, operationTemplate)
	}
	if pathInfo.Get != nil {
		operationTemplate, err := operationTemplates(openapiDoc, basePath, path, http.MethodGet, pathInfo.Get)
		if err != nil {
			return nil, err
		}
		templates = append(templates, operationTemplate)
	}
	if pathInfo.Head != nil {
		operationTemplate, err := operationTemplates(openapiDoc, basePath, path, http.MethodGet, pathInfo.Get)
		if err != nil {
			return nil, err
		}
		templates = append(templates, operationTemplate)
	}
	if pathInfo.Options != nil {
		operationTemplate, err := operationTemplates(openapiDoc, basePath, path, http.MethodOptions, pathInfo.Options)
		if err != nil {
			return nil, err
		}
		templates = append(templates, operationTemplate)
	}
	if pathInfo.Patch != nil {
		operationTemplate, err := operationTemplates(openapiDoc, basePath, path, http.MethodPatch, pathInfo.Patch)
		if err != nil {
			return nil, err
		}
		templates = append(templates, operationTemplate)
	}
	if pathInfo.Post != nil {
		operationTemplate, err := operationTemplates(openapiDoc, basePath, path, http.MethodPost, pathInfo.Post)
		if err != nil {
			return nil, err
		}
		templates = append(templates, operationTemplate)
	}
	if pathInfo.Put != nil {
		operationTemplate, err := operationTemplates(openapiDoc, basePath, path, http.MethodPut, pathInfo.Put)
		if err != nil {
			return nil, err
		}
		templates = append(templates, operationTemplate)
	}
	if pathInfo.Trace != nil {
		operationTemplate, err := operationTemplates(openapiDoc, basePath, path, http.MethodTrace, pathInfo.Trace)
		if err != nil {
			return nil, err
		}
		templates = append(templates, operationTemplate)
	}

	return templates, nil
}

// operationTemplates parses every operation in paths.
func operationTemplates(openapiDoc *openapi3.T, basePath string, path string, operationName string, operationInfo *openapi3.Operation) (*Template, error) {
	template := &Template{
		Method: operationName,
		Path:   path,
		Doc:    openapiDoc,
	}

	params, err := parseParameters(operationInfo.Parameters)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse request parameters")
	}

	template.PathParameters = params.pathParameters
	template.QueryParameters = params.queryParameters
	template.Headers = params.headers
	template.URL = strings.TrimSuffix(basePath, "/") + path

	placeholders := params.supportedPlaceholders

	requestBody := make(map[string]string)
	requestBodyParameters := make(map[string]*parameterSpec)

	if operationInfo.RequestBody != nil {
		for contentType, mediaType := range operationInfo.RequestBody.Value.Content {
			rawBodyStruct, strAvailable, paramSpec, err := schemaToMap("", mediaType.Schema.Value, false)
			if err != nil {
				return nil, errors.Wrap(err, "couldn't parse request body schema")
			}

			switch contentType {
			case jsonContentType:
				if strAvailable {
					placeholders[jsonRequestPlaceholder] = nil
					placeholders[jsonBodyPlaceholder] = nil
				}

				body, err := jsonMarshal(rawBodyStruct)
				if err != nil {
					return nil, err
				}

				requestBody[contentType] = body

			case xmlContentType:
				rawBodyStruct, strAvailable, paramSpec, err = schemaToMap("", mediaType.Schema.Value, true)
				if err != nil {
					return nil, errors.Wrap(err, "couldn't parse request body schema")
				}

				if strAvailable {
					placeholders[xmlBodyPlaceholder] = nil
				}

				body, err := xmlMarshal(rawBodyStruct)
				if err != nil {
					return nil, err
				}

				requestBody[contentType] = body

			case xWwwFormContentType:
				if strAvailable {
					placeholders[htmlFormPlaceholder] = nil
				}

				body, err := htmlFormMarshal(rawBodyStruct)
				if err != nil {
					return nil, err
				}

				requestBody[contentType] = body

			default:
				if strAvailable {
					placeholders[requestBodyPlaceholder] = nil
				}
				for k := range paramSpec {
					requestBody[plainTextContentType] = k
					break
				}
			}

			for k, v := range paramSpec {
				requestBodyParameters[k] = v
			}
		}
	}

	template.RequestBodyParameters = requestBodyParameters
	template.RequestBody = requestBody
	template.Placeholders = placeholders

	return template, nil
}

// CreateRequest generates a new request with the payload substituted as
// the placeholder value.
func (t *Template) CreateRequest(ctx context.Context, placeholder string, payload string) (*http.Request, error) {
	if _, ok := t.Placeholders[placeholder]; !ok {
		return nil, nil
	}

	var body string
	var contentType string
	queryParams := make(map[string]string)
	headers := make(map[string]string)
	path := t.URL

	switch placeholder {
	case headerPlaceholder:
		for header, spec := range t.Headers {
			if spec.paramType == openapi3.TypeString {
				payloadLen := uint64(len(payload))
				if spec.minLength <= payloadLen && payloadLen <= spec.maxLength {
					headers[header] = payload
					continue
				}
			}

			headers[header] = spec.value
		}

	case urlPathPlaceholder:
		for param, spec := range t.PathParameters {
			if spec.paramType == openapi3.TypeString {
				payloadLen := uint64(len(payload))
				if spec.minLength <= payloadLen && payloadLen <= spec.maxLength {
					path = strings.ReplaceAll(path, param, payload)
					continue
				}
			}

			path = strings.ReplaceAll(path, param, spec.value)
		}

	case urlParamPlaceholder:
		for param, spec := range t.QueryParameters {
			if spec.paramType == openapi3.TypeString {
				payloadLen := uint64(len(payload))
				if spec.minLength <= payloadLen && payloadLen <= spec.maxLength {
					queryParams[param] = payload
					continue
				}
			}

			queryParams[param] = spec.value
		}

	case htmlFormPlaceholder:
		body = t.RequestBody[xWwwFormContentType]
		contentType = xWwwFormContentType

		for paramDefaultValue, spec := range t.RequestBodyParameters {
			if spec.paramType == openapi3.TypeString {
				payloadLen := uint64(len(payload))
				if spec.minLength <= payloadLen && payloadLen <= spec.maxLength {
					body = strings.ReplaceAll(body, paramDefaultValue, payload)
					continue
				}
			}
		}

	case jsonBodyPlaceholder:
		fallthrough
	case jsonRequestPlaceholder:
		body = t.RequestBody[jsonContentType]
		contentType = jsonContentType

		for paramDefaultValue, spec := range t.RequestBodyParameters {
			if spec.paramType == openapi3.TypeString {
				payloadLen := uint64(len(payload))
				if spec.minLength <= payloadLen && payloadLen <= spec.maxLength {
					body = strings.ReplaceAll(body, paramDefaultValue, payload)
					continue
				}
			}
		}

	case xmlBodyPlaceholder:
		body = t.RequestBody[xmlContentType]
		contentType = xmlContentType

		for paramDefaultValue, spec := range t.RequestBodyParameters {
			if spec.paramType == openapi3.TypeString {
				payloadLen := uint64(len(payload))
				if spec.minLength <= payloadLen && payloadLen <= spec.maxLength {
					body = strings.ReplaceAll(body, paramDefaultValue, payload)
					continue
				}
			}
		}

	case requestBodyPlaceholder:
		body = t.RequestBody[plainTextContentType]
		contentType = plainTextContentType

		for paramDefaultValue, spec := range t.RequestBodyParameters {
			if spec.paramType == openapi3.TypeString {
				payloadLen := uint64(len(payload))
				if spec.minLength <= payloadLen && payloadLen <= spec.maxLength {
					body = strings.ReplaceAll(body, paramDefaultValue, payload)
					continue
				}
			}
		}

	default:
		return nil, nil
	}

	if placeholder != urlPathPlaceholder {
		for k, v := range t.PathParameters {
			path = strings.ReplaceAll(path, k, v.value)
		}
	}

	req, err := http.NewRequestWithContext(ctx, t.Method, path, bytes.NewReader([]byte(body)))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create request")
	}

	var params []string
	for param, value := range queryParams {
		params = append(params, param+"="+value)
	}
	req.URL.RawQuery = strings.Join(params, "&")

	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}
	for header, value := range headers {
		req.Header.Add(header, value)
	}

	return req, nil
}
