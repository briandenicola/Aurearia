package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestQuickCaptureDraftEnums(t *testing.T) {
	if !IsValidQuickCaptureDraftStatus(QuickCaptureDraftStatusActive) {
		t.Fatal("active status should be valid")
	}
	if IsValidQuickCaptureDraftStatus(QuickCaptureDraftStatus("archived")) {
		t.Fatal("unexpected status should be invalid")
	}
	if !IsValidDraftLifecycleEventType(DraftLifecycleEventImageAdded) {
		t.Fatal("image_added lifecycle event should be valid")
	}
	if IsValidDraftLifecycleEventType(DraftLifecycleEventType("raw filesystem path")) {
		t.Fatal("unexpected lifecycle event should be invalid")
	}
}

func TestSelectedNumistaReferenceValidationAndCanonicalization(t *testing.T) {
	valid, err := ParseSelectedNumistaReference("0012345", "https://en.numista.com/catalogue/pieces12345.html")
	if err != nil {
		t.Fatalf("parse valid selection: %v", err)
	}
	if valid.Catalog != "Numista" || valid.Number != "12345" ||
		valid.URI != "https://en.numista.com/catalogue/pieces12345.html" {
		t.Fatalf("unexpected canonical selection: %#v", valid)
	}

	for _, test := range []struct {
		name   string
		number string
		uri    string
	}{
		{"zero", "0", "https://en.numista.com/catalogue/pieces0.html"},
		{"negative", "-1", "https://en.numista.com/catalogue/pieces-1.html"},
		{"non numeric", "abc", "https://en.numista.com/catalogue/pieces1.html"},
		{"missing uri", "1", ""},
		{"wrong id", "1", "https://en.numista.com/catalogue/pieces2.html"},
		{"wrong host", "1", "https://example.com/catalogue/pieces1.html"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseSelectedNumistaReference(test.number, test.uri); err == nil {
				t.Fatal("expected invalid selection")
			}
		})
	}

	invalidCatalog := valid
	invalidCatalog.Catalog = "Other"
	if err := invalidCatalog.Validate(); err == nil {
		t.Fatal("expected fixed Numista catalog validation")
	}
}

func TestQuickCaptureDraftOptionalSelectedReferenceJSON(t *testing.T) {
	without, err := json.Marshal(QuickCaptureDraft{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(without), `"selectedNumistaReference":null`) {
		t.Fatalf("missing additive null relation: %s", without)
	}

	with, err := json.Marshal(QuickCaptureDraft{
		SelectedNumistaReference: &QuickCaptureDraftReference{
			ID: 9, DraftID: 7, UserID: 3, Catalog: "Numista", Number: "12345",
			URI: "https://en.numista.com/catalogue/pieces12345.html",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(with)
	if !strings.Contains(jsonText, `"selectedNumistaReference":{"catalog":"Numista","number":"12345","uri":"https://en.numista.com/catalogue/pieces12345.html"}`) {
		t.Fatalf("unexpected selected relation JSON: %s", jsonText)
	}
	for _, privateField := range []string{`"draftId"`, `"userId":3`, `"id":9`} {
		if strings.Contains(jsonText, privateField) {
			t.Fatalf("internal selected-reference field leaked in %s", jsonText)
		}
	}
}
