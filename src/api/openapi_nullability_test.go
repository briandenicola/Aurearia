package main

import (
	"encoding/json"
	"os"
	"testing"
)

type openAPISchemaDocument struct {
	Swagger     string `json:"swagger"`
	Definitions map[string]struct {
		Type       string `json:"type"`
		Properties map[string]struct {
			Ref   string `json:"$ref"`
			AllOf []struct {
				Ref string `json:"$ref"`
			} `json:"allOf"`
			XNullable bool `json:"x-nullable"`
		} `json:"properties"`
	} `json:"definitions"`
}

func TestOpenAPINumistaObjectNullability(t *testing.T) {
	data, err := os.ReadFile("docs/swagger.json")
	if err != nil {
		t.Fatalf("read generated OpenAPI: %v", err)
	}
	var document openAPISchemaDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse generated OpenAPI: %v", err)
	}
	if document.Swagger != "2.0" {
		t.Fatalf("expected Swagger 2.0 document, got %q", document.Swagger)
	}

	tests := []struct {
		definition string
		property   string
		ref        string
	}{
		{
			definition: "handlers.CoinLookupSwaggerResponse",
			property:   "numistaLookup",
			ref:        "#/definitions/models.NumistaLookupOutcome",
		},
		{
			definition: "models.QuickCaptureDraft",
			property:   "selectedNumistaReference",
			ref:        "#/definitions/models.QuickCaptureDraftReference",
		},
	}
	for _, test := range tests {
		t.Run(test.definition+"."+test.property, func(t *testing.T) {
			definition, ok := document.Definitions[test.definition]
			if !ok {
				t.Fatalf("generated definition %q is missing", test.definition)
			}
			property, ok := definition.Properties[test.property]
			if !ok {
				t.Fatalf("generated property %q is missing", test.property)
			}
			reference := property.Ref
			if reference == "" && len(property.AllOf) == 1 {
				reference = property.AllOf[0].Ref
			}
			if reference != test.ref {
				t.Fatalf("generated object reference = %q, want %q", reference, test.ref)
			}
			targetName := test.ref[len("#/definitions/"):]
			if target, ok := document.Definitions[targetName]; !ok || target.Type != "object" {
				t.Fatalf("referenced definition %q must exist with type object", targetName)
			}
			if !property.XNullable {
				t.Fatal("generated Swagger 2.0 object reference must set x-nullable: true")
			}
		})
	}
}
