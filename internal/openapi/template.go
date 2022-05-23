package openapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	goPath "path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Templates contains all templates generated from OpenAPI file. Templates are
// sorted by placeholders that can be used to substitute a malicious vector.
type Templates map[string][]*Template

// Template contains all information about template.
type Template struct {
	Method          string
	URL             string
	QueryParameters map[string]string
	Headers         map[string]string
	RequestBody     map[string]string

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
	params, err := parseParameters(operationInfo.Parameters)
	if err != nil {
		return nil, err
	}

	for paramName, value := range params.pathParameters {
		path = strings.ReplaceAll(path, "{"+paramName+"}", value)
	}

	templateURL, err := url.Parse(basePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse base URL: %s", err.Error())
	}
	templateURL.Path = goPath.Join(templateURL.Path, path)

	placeholders := params.supportedPlaceholders

	requestBody := make(map[string]string)

	if operationInfo.RequestBody != nil {
		for contentType, mediaType := range operationInfo.RequestBody.Value.Content {
			rawBodyStruct, strAvailable, err := schemaToMap(mediaType.Schema.Value)
			if err != nil {
				return nil, err
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

			case plainTextContentType:
				fallthrough
			case anyContentType:
				if strAvailable {
					placeholders[requestBodyPlaceholder] = nil
				}
				requestBody[contentType] = bodyStringPlaceholder

			default:
				return nil, fmt.Errorf("unsupported Content-Type %s", contentType)
			}
		}
	}

	template := &Template{
		Method:          operationName,
		URL:             templateURL.String(),
		QueryParameters: params.queryParameters,
		Headers:         params.headers,
		RequestBody:     requestBody,
		Doc:             openapiDoc,
		Placeholders:    placeholders,
	}

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
		for header, value := range t.Headers {
			if value == headerStringPlaceholder {
				value = payload
			}
			headers[header] = value
		}

	case urlPathPlaceholder:
		path = strings.ReplaceAll(path, pathStringPlaceholder, payload)

	case urlParamPlaceholder:
		for param, value := range t.QueryParameters {
			if value == parameterStringPlaceholder {
				value = payload
			}
			queryParams[param] = value
		}

	case htmlFormPlaceholder:
		body = t.RequestBody[xWwwFormContentType]
		body = strings.ReplaceAll(body, bodyStringPlaceholder, payload)
		contentType = xWwwFormContentType

	case jsonBodyPlaceholder:
		fallthrough
	case jsonRequestPlaceholder:
		body = t.RequestBody[jsonContentType]
		body = strings.ReplaceAll(body, bodyStringPlaceholder, payload)
		contentType = jsonContentType

	case xmlBodyPlaceholder:
		body = t.RequestBody[xmlContentType]
		body = strings.ReplaceAll(body, bodyStringPlaceholder, payload)
		contentType = xmlContentType

	case requestBodyPlaceholder:
		body = t.RequestBody[plainTextContentType]
		body = strings.ReplaceAll(body, bodyStringPlaceholder, payload)
		contentType = plainTextContentType

	default:
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, t.Method, path, bytes.NewReader([]byte(body)))
	if err != nil {
		return nil, err
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
