package openapi

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
)

// LoadOpenAPISpec loads an openAPI file, parses it and validates data.
func LoadOpenAPISpec(ctx context.Context, location string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(location)
	if err != nil {
		return nil, err
	}

	err = doc.Validate(ctx)
	if err != nil {
		return nil, err
	}

	return doc, nil
}
