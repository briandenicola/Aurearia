package services

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExplorationDTOJSONShapesMatchOpenAPI(t *testing.T) {
	schemas := loadExplorationOpenAPISchemas(t)
	assertJSONFieldsMatchSchema(t, reflect.TypeOf(ExplorationDecisionRequest{}), schemas, "DecisionRequest")
	assertJSONFieldsMatchSchema(t, reflect.TypeOf(ExplorationObservation{}), schemas, "Observation")
	assertJSONFieldsMatchSchema(t, reflect.TypeOf(ExplorationDecisionResponse{}), schemas, "DecisionResponse")
	assertJSONFieldsMatchSchema(t, reflect.TypeOf(ExplorationActionTarget{}), schemas, "ActionTarget")
	assertJSONFieldsMatchSchema(t, reflect.TypeOf(ExplorationModelTriage{}), schemas, "ModelTriage")
	assertJSONFieldsMatchSchema(t, reflect.TypeOf(ExplorationModelUsage{}), schemas, "ModelUsage")
}

func TestExplorationContractConstantsMatchOpenAPI(t *testing.T) {
	if ExplorationDecisionSchemaVersion != "aurearia.browser-exploration-decision/v1" {
		t.Fatalf("unexpected schema version %q", ExplorationDecisionSchemaVersion)
	}
	if ExplorationDecisionPath != "/internal/browser-exploration/decide" {
		t.Fatalf("unexpected decision path %q", ExplorationDecisionPath)
	}
}

func loadExplorationOpenAPISchemas(t *testing.T) map[string]any {
	t.Helper()
	path := filepath.Join("..", "..", "..", "specs", "360-ai-driven-browser-testing", "contracts", "exploration-agent.openapi.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse OpenAPI contract: %v", err)
	}
	components := document["components"].(map[string]any)
	return components["schemas"].(map[string]any)
}

func assertJSONFieldsMatchSchema(t *testing.T, dto reflect.Type, schemas map[string]any, schemaName string) {
	t.Helper()
	schema := schemas[schemaName].(map[string]any)
	properties := schema["properties"].(map[string]any)
	actual := make(map[string]bool, dto.NumField())
	for index := 0; index < dto.NumField(); index++ {
		tag := dto.Field(index).Tag.Get("json")
		name := tag
		for offset, char := range name {
			if char == ',' {
				name = name[:offset]
				break
			}
		}
		actual[name] = true
	}
	if len(actual) != len(properties) {
		t.Fatalf("%s field count drift: Go=%v OpenAPI=%v", schemaName, actual, properties)
	}
	for name := range properties {
		if !actual[name] {
			t.Errorf("%s missing Go JSON field %q", schemaName, name)
		}
	}
}
