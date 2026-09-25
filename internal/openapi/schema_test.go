package openapi

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// OpenAPI 3 does not require the type field, the type must be inferred from
// the schema structure (https://github.com/wallarm/gotestwaf/issues/257).
func TestSchemaToMapUntypedSchema(t *testing.T) {
	spec := `
openapi: 3.0.0
info: {title: t, version: "1"}
paths: {}
components:
  schemas:
    Untyped:
      properties:
        Dates:
          items: {type: string, format: date-time}
        Nested:
          properties:
            Name: {type: string}
        Free:
          description: no type at all
`
	doc, err := openapi3.NewLoader().LoadFromData([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}

	value, strAvailable, _, err := schemaToMap("", doc.Components.Schemas["Untyped"].Value, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strAvailable {
		t.Fatal("expected string parameters to be available")
	}

	obj, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object, got %T", value)
	}
	if _, ok := obj["Dates"].([]interface{}); !ok {
		t.Fatalf("expected Dates to be an array, got %T", obj["Dates"])
	}
	if _, ok := obj["Nested"].(map[string]interface{}); !ok {
		t.Fatalf("expected Nested to be an object, got %T", obj["Nested"])
	}
	if _, ok := obj["Free"].(string); !ok {
		t.Fatalf("expected Free to be a string, got %T", obj["Free"])
	}
}
