package services

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
)

func catalogParserTestRegistry() map[string]*models.CatalogRegistry {
	return map[string]*models.CatalogRegistry{
		"RIC":      {Catalog: "RIC", VolumeRequired: true},
		"RPC":      {Catalog: "RPC", VolumeRequired: true},
		"SNG":      {Catalog: "SNG", VolumeRequired: true},
		"SEAR":     {Catalog: "SEAR", VolumeRequired: false},
		"CRAWFORD": {Catalog: "CRAWFORD", VolumeRequired: false},
		"SPINK":    {Catalog: "SPINK", VolumeRequired: false},
		"DUPLESSY": {Catalog: "DUPLESSY", VolumeRequired: false},
	}
}

// TestParseCatalogReferenceText_ConfidenceTable exercises the FR-017
// confidence table: 0.90 clean (volume not required), 0.90 Roman-numeral
// volume, 0.50 inferred (plausible but non-Roman) volume, and 0.30 when
// NeedsVolume is true.
func TestParseCatalogReferenceText_ConfidenceTable(t *testing.T) {
	registry := catalogParserTestRegistry()

	tests := []struct {
		name            string
		input           string
		wantFound       bool
		wantCatalog     string
		wantVolume      string
		wantNumber      string
		wantConfidence  float64
		wantNeedsVolume bool
	}{
		{
			name:           "clean: volume not required",
			input:          "Sear 1625",
			wantFound:      true,
			wantCatalog:    "SEAR",
			wantVolume:     "",
			wantNumber:     "1625",
			wantConfidence: 0.90,
		},
		{
			name:           "clean: Roman numeral volume",
			input:          "RIC II 207",
			wantFound:      true,
			wantCatalog:    "RIC",
			wantVolume:     "II",
			wantNumber:     "207",
			wantConfidence: 0.90,
		},
		{
			name:           "inferred: numeric (non-Roman) volume",
			input:          "RIC 2 207",
			wantFound:      true,
			wantCatalog:    "RIC",
			wantVolume:     "2",
			wantNumber:     "207",
			wantConfidence: 0.50,
		},
		{
			name:           "inferred: alphabetic (non-Roman) volume",
			input:          "SNG Cop 123",
			wantFound:      true,
			wantCatalog:    "SNG",
			wantVolume:     "Cop",
			wantNumber:     "123",
			wantConfidence: 0.50,
		},
		{
			name:            "needs volume: bare catalog, no tokens follow",
			input:           "RIC",
			wantFound:       true,
			wantCatalog:     "RIC",
			wantVolume:      "",
			wantNumber:      "",
			wantConfidence:  0.30,
			wantNeedsVolume: true,
		},
		{
			name:            "needs volume: bare number, no plausible volume token",
			input:           "RIC 207",
			wantFound:       true,
			wantCatalog:     "RIC",
			wantVolume:      "",
			wantNumber:      "",
			wantConfidence:  0.30,
			wantNeedsVolume: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, found := ParseCatalogReferenceText(tt.input, registry)
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v (parsed: %+v)", found, tt.wantFound, parsed)
			}
			if parsed.Catalog != tt.wantCatalog {
				t.Errorf("Catalog = %q, want %q", parsed.Catalog, tt.wantCatalog)
			}
			if parsed.Volume != tt.wantVolume {
				t.Errorf("Volume = %q, want %q", parsed.Volume, tt.wantVolume)
			}
			if parsed.Number != tt.wantNumber {
				t.Errorf("Number = %q, want %q", parsed.Number, tt.wantNumber)
			}
			if parsed.Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %v, want %v", parsed.Confidence, tt.wantConfidence)
			}
			if parsed.NeedsVolume != tt.wantNeedsVolume {
				t.Errorf("NeedsVolume = %v, want %v", parsed.NeedsVolume, tt.wantNeedsVolume)
			}
		})
	}
}

// TestParseCatalogReferenceText_NeverEmitsVolumeZeroSentinel asserts FR-019:
// the shared helper must never emit the "0" placeholder volume itself. That
// substitution is migration-specific policy that lives only in
// ReferenceMigrationService.parseLegacyReference.
func TestParseCatalogReferenceText_NeverEmitsVolumeZeroSentinel(t *testing.T) {
	registry := catalogParserTestRegistry()

	inputs := []string{
		"RIC",
		"RIC 207",
		"RIC 207; Sear 1625",
		"RPC",
		"SNG",
	}

	for _, input := range inputs {
		parsed, found := ParseCatalogReferenceText(input, registry)
		if parsed.Volume == "0" {
			t.Errorf("input %q: Volume = %q, shared helper must never emit the volume=0 sentinel (FR-019)", input, parsed.Volume)
		}
		if found && parsed.NeedsVolume && parsed.Volume != "" {
			t.Errorf("input %q: NeedsVolume=true must pair with Volume=\"\" (empty), got %q", input, parsed.Volume)
		}
	}
}

// TestParseCatalogReferenceText_NotFoundCases covers the "not found" branches
// used by ReferenceMigrationService to reconstruct its own journal messages:
// empty input, unrecognized catalog alias, catalog not in registry, and
// (volume not required) no number found.
func TestParseCatalogReferenceText_NotFoundCases(t *testing.T) {
	registry := catalogParserTestRegistry()

	t.Run("empty string", func(t *testing.T) {
		parsed, found := ParseCatalogReferenceText("", registry)
		if found {
			t.Fatalf("expected found=false, got true (parsed: %+v)", parsed)
		}
		if parsed.Catalog != "" {
			t.Errorf("Catalog = %q, want empty", parsed.Catalog)
		}
		if parsed.Reason != CatalogParseEmpty {
			t.Errorf("Reason = %v, want %v", parsed.Reason, CatalogParseEmpty)
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		parsed, found := ParseCatalogReferenceText("   ", registry)
		if found {
			t.Fatalf("expected found=false, got true (parsed: %+v)", parsed)
		}
		if parsed.Catalog != "" {
			t.Errorf("Catalog = %q, want empty", parsed.Catalog)
		}
		if parsed.Reason != CatalogParseEmpty {
			t.Errorf("Reason = %v, want %v", parsed.Reason, CatalogParseEmpty)
		}
	})

	t.Run("unrecognized catalog alias", func(t *testing.T) {
		parsed, found := ParseCatalogReferenceText("BMCRE 123", registry)
		if found {
			t.Fatalf("expected found=false, got true (parsed: %+v)", parsed)
		}
		if parsed.Reason != CatalogParseUnrecognizedCatalog {
			t.Errorf("Reason = %v, want %v", parsed.Reason, CatalogParseUnrecognizedCatalog)
		}
		if parsed.Catalog != "" {
			t.Errorf("Catalog = %q, want empty for an unrecognized alias", parsed.Catalog)
		}
		if parsed.RawText != "BMCRE 123" {
			t.Errorf("RawText = %q, want %q (caller derives the offending token from it)", parsed.RawText, "BMCRE 123")
		}
	})

	t.Run("recognized alias but not in registry", func(t *testing.T) {
		emptyRegistry := map[string]*models.CatalogRegistry{}
		parsed, found := ParseCatalogReferenceText("RIC II 207", emptyRegistry)
		if found {
			t.Fatalf("expected found=false, got true (parsed: %+v)", parsed)
		}
		if parsed.Reason != CatalogParseNotInRegistry {
			t.Errorf("Reason = %v, want %v", parsed.Reason, CatalogParseNotInRegistry)
		}
		if parsed.Catalog != "RIC" {
			t.Errorf("Catalog = %q, want normalized code %q", parsed.Catalog, "RIC")
		}
	})

	t.Run("volume not required, no number found", func(t *testing.T) {
		parsed, found := ParseCatalogReferenceText("Sear", registry)
		if found {
			t.Fatalf("expected found=false, got true (parsed: %+v)", parsed)
		}
		if parsed.Reason != CatalogParseNoNumber {
			t.Errorf("Reason = %v, want %v", parsed.Reason, CatalogParseNoNumber)
		}
		if parsed.Catalog != "SEAR" {
			t.Errorf("Catalog = %q, want %q", parsed.Catalog, "SEAR")
		}
		if parsed.RawText != "Sear" {
			t.Errorf("RawText = %q, want %q", parsed.RawText, "Sear")
		}
	})
}

// TestParseCatalogReferenceText_ReasonOKOnSuccess asserts that every
// successful (found=true) branch, including the NeedsVolume=true cases,
// reports Reason == CatalogParseOK — Reason only distinguishes *why a parse
// did not succeed*, never a flavor of success.
func TestParseCatalogReferenceText_ReasonOKOnSuccess(t *testing.T) {
	registry := catalogParserTestRegistry()

	inputs := []string{
		"Sear 1625",  // clean, volume not required
		"RIC II 207", // clean, Roman numeral volume
		"RIC 2 207",  // inferred, numeric volume
		"RIC",        // needs volume
		"RIC 207",    // needs volume
	}

	for _, input := range inputs {
		parsed, found := ParseCatalogReferenceText(input, registry)
		if !found {
			t.Fatalf("input %q: expected found=true, got false (parsed: %+v)", input, parsed)
		}
		if parsed.Reason != CatalogParseOK {
			t.Errorf("input %q: Reason = %v, want %v", input, parsed.Reason, CatalogParseOK)
		}
	}
}

// TestParseCatalogReferenceText_MultiReferenceRawText asserts the shared
// helper's RawText only reflects the segment before the first ";", matching
// the origin-text logic ReferenceMigrationService relies on for its journal
// messages.
func TestParseCatalogReferenceText_MultiReferenceRawText(t *testing.T) {
	registry := catalogParserTestRegistry()

	parsed, found := ParseCatalogReferenceText("RIC 207; Sear 1625", registry)
	if !found {
		t.Fatalf("expected found=true, got false (parsed: %+v)", parsed)
	}
	if parsed.RawText != "RIC 207" {
		t.Errorf("RawText = %q, want %q", parsed.RawText, "RIC 207")
	}
	if !parsed.NeedsVolume {
		t.Errorf("expected NeedsVolume=true for bare RIC before the semicolon")
	}
}
