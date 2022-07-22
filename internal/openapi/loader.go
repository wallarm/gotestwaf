package openapi

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	routers_legacy "github.com/getkin/kin-openapi/routers/legacy"
	"github.com/pkg/errors"
)

// LoadOpenAPISpec loads an openAPI file, parses it and validates data.
func LoadOpenAPISpec(ctx context.Context, location string) (*openapi3.T, routers.Router, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(location)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't load OpenAPI file")
	}

	err = doc.Validate(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't validate OpenAPI spec")
	}

	router, err := routers_legacy.NewRouter(doc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't create router from OpenAPI spec")
	}

	return doc, router, nil
}
