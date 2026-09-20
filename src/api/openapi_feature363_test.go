package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestFeature363OpenAPISurface(t *testing.T) {
	data, err := os.ReadFile("docs/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths       map[string]map[string]json.RawMessage `json:"paths"`
		Definitions map[string]json.RawMessage            `json:"definitions"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}

	expected := map[string][]string{
		"/collector-profile":           {"get", "put"},
		"/wishlist/url-intake/analyze": {"post"},
	}
	for path, methods := range expected {
		for _, method := range methods {
			if _, ok := document.Paths[path][method]; !ok {
				t.Errorf("OpenAPI missing %s %s", method, path)
			}
		}
	}

	for _, forbidden := range []string{
		"/wishlist-actions",
		"/wishlist/url-intake/confirm",
		"/wishlist/url-intake/stage",
		"/wishlist/url-intake/fetch",
	} {
		if _, exists := document.Paths[forbidden]; exists {
			t.Errorf("OpenAPI exposes forbidden Feature 363 route %s", forbidden)
		}
	}

	for _, schema := range []string{
		"handlers.wishlistURLRequest",
		"services.WishlistURLAnalysis",
		"services.WishlistURLHypothesis",
		"services.WishlistURLHypothesisField",
	} {
		if _, ok := document.Definitions[schema]; !ok {
			t.Errorf("OpenAPI missing URL-intake schema %s", schema)
		}
	}
}
