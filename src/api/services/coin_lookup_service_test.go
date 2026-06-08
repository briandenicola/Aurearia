package services

import (
	"testing"
)

func TestNormalizeCertNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid cert with 7 digits",
			input:    "823160-093",
			expected: "823160-093",
		},
		{
			name:     "valid cert with 6 digits",
			input:    "123456-001",
			expected: "123456-001",
		},
		{
			name:     "cert with spaces (normalized)",
			input:    "823160 - 093",
			expected: "823160-093",
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "cert with leading/trailing spaces",
			input:    "  823160-093  ",
			expected: "823160-093",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCertNumber(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeCertNumber(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractNGCCert(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCert bool
		wantNum  string
	}{
		{
			name: "valid JSON with NGC cert",
			input: `{
				"ngcCert": "823160-093",
				"ngcGrade": "Ch AU",
				"ngcDescription": "Roman Empire"
			}`,
			wantCert: true,
			wantNum:  "823160-093",
		},
		{
			name: "JSON with null cert",
			input: `{
				"ngcCert": "null",
				"ruler": "Trajan"
			}`,
			wantCert: false,
		},
		{
			name:     "raw text with cert number",
			input:    "This is an NGC slab with cert number 1234567-001 clearly visible.",
			wantCert: true,
			wantNum:  "1234567-001",
		},
		{
			name:     "no cert number",
			input:    "This coin has no NGC certification.",
			wantCert: false,
		},
	}

	svc := &CoinLookupService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.extractNGCCert(tt.input)
			if tt.wantCert {
				if result == nil {
					t.Errorf("extractNGCCert() = nil, want cert data")
					return
				}
				if result.NormalizedCert != tt.wantNum {
					t.Errorf("extractNGCCert().NormalizedCert = %q, want %q", result.NormalizedCert, tt.wantNum)
				}
				expectedURL := "https://www.ngccoin.com/certlookup/?CertNumber=" + tt.wantNum
				if result.LookupURL != expectedURL {
					t.Errorf("extractNGCCert().LookupURL = %q, want %q", result.LookupURL, expectedURL)
				}
			} else {
				if result != nil {
					t.Errorf("extractNGCCert() = %+v, want nil", result)
				}
			}
		})
	}
}

func TestExtractLabelText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "valid JSON with label text",
			input: `{
				"labelText": "NGC Ch AU 5/5 4/5",
				"ruler": "Trajan"
			}`,
			expected: "NGC Ch AU 5/5 4/5",
		},
		{
			name: "JSON with null label text",
			input: `{
				"labelText": "null",
				"ruler": "Trajan"
			}`,
			expected: "",
		},
		{
			name: "JSON with empty label text",
			input: `{
				"labelText": "",
				"ruler": "Trajan"
			}`,
			expected: "",
		},
		{
			name:     "invalid JSON",
			input:    "not json",
			expected: "",
		},
	}

	svc := &CoinLookupService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.extractLabelText(tt.input)
			if result != tt.expected {
				t.Errorf("extractLabelText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNGCLookupURLUsesCertNumberQuery(t *testing.T) {
	got := ngcLookupURL("823160-093")
	want := "https://www.ngccoin.com/certlookup/?CertNumber=823160-093"
	if got != want {
		t.Errorf("ngcLookupURL() = %q, want %q", got, want)
	}
}

func TestExtractCoinFields(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantFields    map[string]string
		wantFieldsLen int
	}{
		{
			name: "all fields present",
			input: `{
				"ruler": "Trajan",
				"era": "ancient",
				"denomination": "Denarius",
				"material": "Silver",
				"category": "Roman"
			}`,
			wantFields: map[string]string{
				"ruler":        "Trajan",
				"era":          "ancient",
				"denomination": "Denarius",
				"material":     "Silver",
				"category":     "Roman",
			},
			wantFieldsLen: 5,
		},
		{
			name: "some fields with null",
			input: `{
				"ruler": "Trajan",
				"era": "ancient",
				"denomination": "null",
				"material": "Silver",
				"category": "null"
			}`,
			wantFields: map[string]string{
				"ruler":    "Trajan",
				"era":      "ancient",
				"material": "Silver",
			},
			wantFieldsLen: 3,
		},
		{
			name:          "invalid JSON",
			input:         "not json",
			wantFieldsLen: 0,
		},
	}

	svc := &CoinLookupService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.extractCoinFields(tt.input)
			if len(result) != tt.wantFieldsLen {
				t.Errorf("extractCoinFields() returned %d fields, want %d", len(result), tt.wantFieldsLen)
			}
			for k, v := range tt.wantFields {
				if result[k] != v {
					t.Errorf("extractCoinFields()[%q] = %q, want %q", k, result[k], v)
				}
			}
		})
	}
}

func TestDetermineConfidence(t *testing.T) {
	tests := []struct {
		name     string
		data     *LookupExtractedData
		expected string
	}{
		{
			name: "high confidence with NGC and coin fields",
			data: &LookupExtractedData{
				NGC: &NGCData{
					CertNumber: "823160-093",
				},
				LabelText: "NGC Ch AU",
				CoinFields: map[string]any{
					"ruler":        "Trajan",
					"era":          "ancient",
					"denomination": "Denarius",
				},
			},
			expected: "high",
		},
		{
			name: "medium confidence with NGC only",
			data: &LookupExtractedData{
				NGC: &NGCData{
					CertNumber: "823160-093",
				},
			},
			expected: "medium",
		},
		{
			name: "medium confidence with label and few fields",
			data: &LookupExtractedData{
				LabelText: "Some text",
				CoinFields: map[string]any{
					"ruler": "Trajan",
				},
			},
			expected: "medium",
		},
		{
			name: "low confidence with one field only",
			data: &LookupExtractedData{
				CoinFields: map[string]any{
					"ruler": "Trajan",
				},
			},
			expected: "low",
		},
		{
			name:     "low confidence with no data",
			data:     &LookupExtractedData{},
			expected: "low",
		},
	}

	svc := &CoinLookupService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.determineConfidence(tt.data)
			if result != tt.expected {
				t.Errorf("determineConfidence() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildNumistaQuery(t *testing.T) {
	tests := []struct {
		name       string
		coinFields map[string]any
		expected   string
	}{
		{
			name: "all fields present",
			coinFields: map[string]any{
				"ruler":        "Trajan",
				"denomination": "Denarius",
				"era":          "ancient",
			},
			expected: "Trajan Denarius ancient",
		},
		{
			name: "only ruler and denomination",
			coinFields: map[string]any{
				"ruler":        "Trajan",
				"denomination": "Denarius",
			},
			expected: "Trajan Denarius",
		},
		{
			name: "only ruler",
			coinFields: map[string]any{
				"ruler": "Trajan",
			},
			expected: "Trajan",
		},
		{
			name:       "no fields",
			coinFields: map[string]any{},
			expected:   "",
		},
	}

	svc := &CoinLookupService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.buildNumistaQuery(tt.coinFields)
			if result != tt.expected {
				t.Errorf("buildNumistaQuery() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildCandidateReferences(t *testing.T) {
	svc := &CoinLookupService{}

	data := &LookupExtractedData{
		NGC: &NGCData{
			NormalizedCert: "823160-093",
			LookupURL:      "https://www.ngccoin.com/certlookup/823160-093/",
		},
	}

	candidates := []NumistaCandidate{
		{
			ID:  "12345",
			URL: "https://en.numista.com/catalogue/pieces12345.html",
		},
	}

	refs := svc.buildCandidateReferences(data, candidates)

	if len(refs) != 2 {
		t.Errorf("buildCandidateReferences() returned %d refs, want 2", len(refs))
	}

	// Check NGC reference
	if refs[0].Catalog != "NGC" {
		t.Errorf("refs[0].Catalog = %q, want NGC", refs[0].Catalog)
	}
	if refs[0].Number != "823160-093" {
		t.Errorf("refs[0].Number = %q, want 823160-093", refs[0].Number)
	}

	// Check Numista reference
	if refs[1].Catalog != "Numista" {
		t.Errorf("refs[1].Catalog = %q, want Numista", refs[1].Catalog)
	}
	if refs[1].Number != "12345" {
		t.Errorf("refs[1].Number = %q, want 12345", refs[1].Number)
	}
}
