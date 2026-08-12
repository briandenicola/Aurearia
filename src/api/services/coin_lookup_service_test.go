package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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
			name:     "compact cert with 10 digits",
			input:    "2412821034",
			expected: "2412821-034",
		},
		{
			name:     "compact cert with 9 digits",
			input:    "823160093",
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
			name:     "raw text with compact cert number",
			input:    "This is an NGC Ancients slab with cert number 2412821034 clearly visible.",
			wantCert: true,
			wantNum:  "2412821-034",
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
				expectedURL := "https://www.ngccoin.com/certlookup/" + strings.ReplaceAll(tt.wantNum, "-", "") + "/NGCAncients/"
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

func TestNGCLookupURLUsesAncientsPath(t *testing.T) {
	got := ngcLookupURL("823160-093")
	want := "https://www.ngccoin.com/certlookup/823160093/NGCAncients/"
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
				"name": "Trajan Denarius",
				"ruler": "Trajan",
				"mint": "Rome",
				"era": "ancient",
				"denomination": "Denarius",
				"material": "Silver",
				"category": "Roman",
				"obverseInscription": "IMP TRAIANO AVG",
				"reverseInscription": "SPQR OPTIMO PRINCIPI",
				"obverseDescription": "Laureate bust right",
				"reverseDescription": "Victory standing left",
				"weightGrams": 3.2,
				"diameterMm": 19,
				"rarityRating": "Common",
				"grade": "VF"
			}`,
			wantFields: map[string]string{
				"name":               "Trajan Denarius",
				"ruler":              "Trajan",
				"mint":               "Rome",
				"era":                "ancient",
				"denomination":       "Denarius",
				"material":           "Silver",
				"category":           "Roman",
				"obverseInscription": "IMP TRAIANO AVG",
				"reverseInscription": "SPQR OPTIMO PRINCIPI",
				"obverseDescription": "Laureate bust right",
				"reverseDescription": "Victory standing left",
				"rarityRating":       "Common",
				"grade":              "VF",
			},
			wantFieldsLen: 15,
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
			name: "nested intake-style coin fields",
			input: `{
				"ngcCert": null,
				"coin": {
					"name": "Hadrian Sestertius",
					"ruler": "Hadrian",
					"era": "ancient",
					"denomination": "Sestertius",
					"material": "Bronze",
					"category": "Roman",
					"obverse_description": "Laureate head right",
					"reverse_description": "Salus standing left"
				}
			}`,
			wantFields: map[string]string{
				"name":               "Hadrian Sestertius",
				"ruler":              "Hadrian",
				"era":                "ancient",
				"denomination":       "Sestertius",
				"material":           "Bronze",
				"category":           "Roman",
				"obverseDescription": "Laureate head right",
				"reverseDescription": "Salus standing left",
			},
			wantFieldsLen: 8,
		},
		{
			name:          "invalid JSON",
			input:         "not json",
			wantFieldsLen: 0,
		},
		{
			name: "note-like fields from raw analysis",
			input: `Name: Julia Domna Denarius
Ruler: Julia Domna
Denomination: Denarius
Category: Roman`,
			wantFields: map[string]string{
				"name":         "Julia Domna Denarius",
				"ruler":        "Julia Domna",
				"denomination": "Denarius",
				"category":     "Roman",
			},
			wantFieldsLen: 4,
		},
		{
			name:  "NGC label text extracts visible attribution",
			input: `ROMAN EMPIRE / Constantine I, AD 307-337 / BI Reduced Nummus / LONDON MINT`,
			wantFields: map[string]string{
				"name":         "Constantine I Reduced Nummus",
				"ruler":        "Constantine I",
				"denomination": "Reduced Nummus",
				"material":     "Billon",
				"category":     "Roman",
				"era":          "ancient",
				"mint":         "London",
			},
			wantFieldsLen: 7,
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

func TestPhotoNumistaProposalUsesSharedBuilderAndPreservesRichEvidence(t *testing.T) {
	data := &LookupExtractedData{
		LabelText: "NGC Ancients Honorius AE3 RIC IX 46",
		CoinFields: map[string]any{
			"name":               "Honorius AE3 GLORIA ROMANORVM RIC IX 46 LRBC 2424",
			"ruler":              "Honorius",
			"denomination":       "AE3",
			"mint":               "SMNT",
			"dateRange":          "393-423 CE",
			"material":           "Bronze",
			"obverseInscription": "DN HONORIVS PF AVG",
			"reverseInscription": "GLORIA ROMANORVM",
		},
	}

	evidence := buildPhotoNumistaEvidence(data)
	builder := NewNumistaQueryBuilder()
	if got := builder.Build(evidence).Primary; got != "Honorius GLORIA ROMANORVM Nicomedia" {
		t.Fatalf("shared provider proposal = %q", got)
	}
	if evidence.Title != "Honorius AE3 GLORIA ROMANORVM RIC IX 46 LRBC 2424" ||
		evidence.DateText != "393-423 CE" || evidence.Material != "Bronze" ||
		evidence.ObverseInscription != "DN HONORIVS PF AVG" ||
		evidence.VisibleText != "NGC Ancients Honorius AE3 RIC IX 46" {
		t.Fatalf("rich local evidence was changed: %#v", evidence)
	}
}

func TestBuildPrefilledDraftFallsBackToStandardCoinAnalysisFields(t *testing.T) {
	svc := &CoinLookupService{}
	data := &LookupExtractedData{
		RawAnalysis: "Standard image analysis narrative",
		CoinFields: map[string]any{
			"ruler":              "Hadrian",
			"denomination":       "Sestertius",
			"material":           "Bronze",
			"category":           "Roman",
			"obverseDescription": "Laureate head right",
			"reverseDescription": "Salus standing left",
		},
	}

	draft := svc.buildPrefilledDraft(data, nil)

	if draft["name"] != "Hadrian Sestertius" {
		t.Fatalf("draft name = %q, want fallback Hadrian Sestertius", draft["name"])
	}
	if draft["material"] != "Bronze" {
		t.Errorf("draft material = %q, want Bronze", draft["material"])
	}
	if draft["obverseDescription"] != "Laureate head right" {
		t.Errorf("draft obverseDescription = %q, want Laureate head right", draft["obverseDescription"])
	}
	if draft["aiAnalysis"] != "Standard image analysis narrative" {
		t.Errorf("draft aiAnalysis = %q, want standard image analysis narrative", draft["aiAnalysis"])
	}
	if _, ok := draft["notes"]; ok {
		t.Errorf("draft notes should not include NGC notes for a raw coin analysis: %q", draft["notes"])
	}
}

func TestBuildPrefilledDraftKeepsExplicitNameFromAnalysis(t *testing.T) {
	svc := &CoinLookupService{}
	data := &LookupExtractedData{
		CoinFields: map[string]any{
			"name":         "Augustus Denarius",
			"ruler":        "Augustus",
			"denomination": "Denarius",
		},
	}

	draft := svc.buildPrefilledDraft(data, nil)

	if draft["name"] != "Augustus Denarius" {
		t.Fatalf("draft name = %q, want explicit analysis name", draft["name"])
	}
}

func TestBuildPrefilledDraftUsesBackfilledNoteLikeName(t *testing.T) {
	svc := &CoinLookupService{}
	fields := svc.extractCoinFields(`Name: Julia Domna Denarius
Ruler: Julia Domna
Denomination: Denarius
Category: Roman`)

	draft := svc.buildPrefilledDraft(&LookupExtractedData{CoinFields: fields}, nil)

	if draft["name"] != "Julia Domna Denarius" {
		t.Fatalf("draft name = %q, want Julia Domna Denarius", draft["name"])
	}
}

func TestBuildPrefilledDraftUsesNGCLabelTextAttribution(t *testing.T) {
	svc := &CoinLookupService{}
	fields := svc.extractCoinFields("ROMAN EMPIRE / Constantine I, AD 307-337 / BI Reduced Nummus / LONDON MINT")

	draft := svc.buildPrefilledDraft(&LookupExtractedData{
		NGC: &NGCData{
			NormalizedCert: "2068676-077",
			LookupURL:      "https://www.ngccoin.com/certlookup/2068676077/NGCAncients/",
		},
		CoinFields: fields,
	}, nil)

	if draft["name"] == "Unidentified Coin" {
		t.Fatalf("draft name should not be Unidentified Coin when visible NGC label attribution is present")
	}
	if draft["name"] != "Constantine I Reduced Nummus" {
		t.Fatalf("draft name = %q, want Constantine I Reduced Nummus", draft["name"])
	}
	if draft["mint"] != "London" {
		t.Errorf("draft mint = %q, want London", draft["mint"])
	}
	if draft["material"] != "Billon" {
		t.Errorf("draft material = %q, want Billon", draft["material"])
	}
	notes, ok := draft["notes"].(string)
	if !ok {
		t.Fatalf("draft notes = %T, want string", draft["notes"])
	}
	if !strings.Contains(notes, "NGC Cert: 2068676-077") {
		t.Errorf("draft notes = %q, want NGC cert", notes)
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

	if len(refs) != 1 {
		t.Errorf("buildCandidateReferences() returned %d refs, want 1", len(refs))
	}

	// Check NGC reference
	if refs[0].Catalog != "NGC" {
		t.Errorf("refs[0].Catalog = %q, want NGC", refs[0].Catalog)
	}
	if refs[0].Number != "823160-093" {
		t.Errorf("refs[0].Number = %q, want 823160-093", refs[0].Number)
	}

}

func TestSlabCertDetectionReturnsCertData(t *testing.T) {
	svc := &CoinLookupService{}
	data := &LookupExtractedData{
		NGC: svc.extractNGCCert(`{"ngcCert":"2412821034","ngcGrade":"Ch VF","ngcDescription":"Roman Empire"}`),
	}

	if data.NGC == nil {
		t.Fatal("expected NGC cert data")
	}
	if data.NGC.NormalizedCert != "2412821-034" {
		t.Fatalf("normalized cert = %q, want 2412821-034", data.NGC.NormalizedCert)
	}

	refs := svc.buildCandidateReferences(data, nil)
	if len(refs) != 1 {
		t.Fatalf("refs len = %d, want 1", len(refs))
	}
	if refs[0].Catalog != "NGC" || refs[0].Number != "2412821-034" {
		t.Fatalf("ref = %+v, want NGC 2412821-034", refs[0])
	}
}

func TestBuildPrefilledDraft_NonSlabAnalysisDoesNotInferTopCandidate(t *testing.T) {
	svc := &CoinLookupService{}

	data := &LookupExtractedData{
		CoinFields: map[string]any{
			"ruler":        "Trajan",
			"era":          "ancient",
			"denomination": "Denarius",
			"material":     "Silver",
			"category":     "Roman",
		},
	}
	candidates := []NumistaCandidate{
		{
			ID:    "12345",
			Title: "Denarius - Trajan (98-117)",
			URL:   "https://en.numista.com/catalogue/pieces12345.html",
		},
	}

	draft := svc.buildPrefilledDraft(data, candidates)

	expected := map[string]any{
		"name":         "Trajan Denarius",
		"ruler":        "Trajan",
		"era":          "ancient",
		"denomination": "Denarius",
		"material":     "Silver",
		"category":     "Roman",
	}
	for key, want := range expected {
		if got := draft[key]; got != want {
			t.Errorf("draft[%q] = %v, want %v", key, got, want)
		}
	}
	if _, ok := draft["notes"]; ok {
		t.Errorf("draft[notes] was set for non-slab lookup, want absent")
	}
	if _, ok := draft["numista_id"]; ok {
		t.Fatal("unselected Numista candidate leaked into prefilled draft")
	}
	if _, ok := draft["numista_url"]; ok {
		t.Fatal("unselected Numista URL leaked into prefilled draft")
	}
}

func newCoinLookupServiceForResponse(t *testing.T, analysis string, handler func(*http.Request)) *CoinLookupService {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler != nil {
			handler(r)
		}
		_ = json.NewEncoder(w).Encode(AnalyzeProxyResponse{Analysis: analysis})
	}))
	t.Cleanup(server.Close)
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:coin_lookup_phase4_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatal(err)
	}
	for _, setting := range []models.AppSetting{
		{Key: SettingAIProvider, Value: "anthropic"},
		{Key: SettingAnthropicAPIKey, Value: "test-key"},
		{Key: SettingAnthropicModel, Value: "test-model"},
	} {
		if err := db.Create(&setting).Error; err != nil {
			t.Fatal(err)
		}
	}
	return NewCoinLookupService(
		NewAgentProxy(server.URL, "internal-token", NewLogger(10)),
		NewSettingsService(repository.NewSettingsRepository(db)),
		NewLogger(10),
	)
}

func TestCoinLookupPhotoProposalWithoutEagerNumistaAndNGCFirstAliases(t *testing.T) {
	nonNGC := `{"ngcCert":null,"labelText":"IMP TRAIANO","name":"Trajan Denarius","ruler":"Trajan","mint":"Rome","denomination":"Denarius","material":"Silver","obverseInscription":"IMP TRAIANO","reverseInscription":"PAX"}`
	svc := newCoinLookupServiceForResponse(t, nonNGC, func(r *http.Request) {
		if r.URL.Path != "/api/analyze" {
			t.Fatalf("unexpected eager provider request: %s", r.URL.Path)
		}
	})
	result, err := svc.Lookup(context.Background(), 1, CoinLookupRequest{Images: []string{"data:image/png;base64,AA=="}})
	if err != nil {
		t.Fatal(err)
	}
	if result.ProposedNumistaQuery != "Trajan PAX Rome" {
		t.Fatalf("canonical editable proposal = %q, want %q", result.ProposedNumistaQuery, "Trajan PAX Rome")
	}
	if result.NumistaEvidence.Title != "Trajan Denarius" || result.NumistaEvidence.Mint != "Rome" {
		t.Fatalf("typed evidence mismatch: %#v", result.NumistaEvidence)
	}
	if result.NumistaLookup != nil || len(result.NumistaCandidates) != 0 || len(result.CandidateReferences) != 0 {
		t.Fatalf("unselected Numista results leaked through aliases: %#v", result)
	}

	ngc := `{"ngcCert":"823160-093","ngcGrade":"Ch AU","labelText":"NGC","name":"Trajan Denarius","ruler":"Trajan"}`
	svc = newCoinLookupServiceForResponse(t, ngc, nil)
	result, err = svc.Lookup(context.Background(), 1, CoinLookupRequest{Images: []string{"data:image/png;base64,AA=="}})
	if err != nil {
		t.Fatal(err)
	}
	if result.ProposedNumistaQuery != "" || result.NumistaEvidence != (models.NumistaEvidence{}) {
		t.Fatalf("NGC-first result should suppress Numista proposal: %#v", result)
	}
	if len(result.NumistaCandidates) != 0 || len(result.CandidateReferences) != 1 ||
		result.CandidateReferences[0].Catalog != "NGC" {
		t.Fatalf("deprecated aliases should contain NGC only: %#v", result)
	}
}

func TestCoinLookupResponseNumistaLookupJSONNullability(t *testing.T) {
	for _, test := range []struct {
		name   string
		lookup *models.NumistaLookupOutcome
		want   any
	}{
		{name: "null", lookup: nil, want: nil},
		{
			name: "object",
			lookup: &models.NumistaLookupOutcome{
				Status:         models.NumistaStatusEmpty,
				EffectiveQuery: "Trajan denarius",
				Candidates:     []models.NumistaCandidate{},
				Stage:          "broad",
			},
			want: map[string]any{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(CoinLookupResponse{NumistaLookup: test.lookup})
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]any
			if err := json.Unmarshal(data, &wire); err != nil {
				t.Fatal(err)
			}
			got, ok := wire["numistaLookup"]
			if !ok {
				t.Fatal("numistaLookup must remain present in runtime JSON")
			}
			if test.want == nil {
				if got != nil {
					t.Fatalf("numistaLookup = %#v, want null", got)
				}
				return
			}
			object, ok := got.(map[string]any)
			if !ok {
				t.Fatalf("numistaLookup = %T, want object", got)
			}
			if object["status"] != string(models.NumistaStatusEmpty) ||
				object["effectiveQuery"] != "Trajan denarius" {
				t.Fatalf("unexpected numistaLookup object: %#v", object)
			}
		})
	}
}

func TestCoinLookupPhotoEvidenceBoundsNoiseNoImageAndCancellation(t *testing.T) {
	long := strings.Repeat("é", 700)
	evidence := buildPhotoNumistaEvidence(&LookupExtractedData{
		LabelText: "not visible",
		CoinFields: map[string]any{
			"name": long, "ruler": "unknown", "denomination": "Denarius",
			"obverseInscription": long,
		},
	})
	if len([]rune(evidence.Title)) != 200 || len([]rune(evidence.ObverseInscription)) != 500 {
		t.Fatalf("evidence was not bounded: title=%d obverse=%d", len([]rune(evidence.Title)), len([]rune(evidence.ObverseInscription)))
	}
	if evidence.Issuer != "" || evidence.VisibleText != "" {
		t.Fatalf("noisy non-specific evidence should be omitted: %#v", evidence)
	}
	if len([]rune(buildPhotoNumistaQuery(evidence))) > models.NumistaMaxQueryLength {
		t.Fatal("proposed query exceeded contract bound")
	}

	svc := newCoinLookupServiceForResponse(t, `{}`, nil)
	emptyResult, err := svc.Lookup(context.Background(), 1, CoinLookupRequest{
		Images: []string{"data:image/png;base64,AA=="},
	})
	if err != nil {
		t.Fatal(err)
	}
	if emptyResult.ProposedNumistaQuery != "" || emptyResult.NumistaEvidence != (models.NumistaEvidence{}) ||
		emptyResult.NumistaLookup != nil || len(emptyResult.NumistaCandidates) != 0 {
		t.Fatalf("empty photo evidence should remain an editable manual-search state: %#v", emptyResult)
	}

	if _, err := svc.Lookup(context.Background(), 1, CoinLookupRequest{}); err == nil {
		t.Fatal("no-image lookup should fail")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.AppSetting{}); err != nil {
		t.Fatal(err)
	}
	for _, setting := range []models.AppSetting{
		{Key: SettingAIProvider, Value: "anthropic"},
		{Key: SettingAnthropicAPIKey, Value: "test-key"},
	} {
		if err := db.Create(&setting).Error; err != nil {
			t.Fatal(err)
		}
	}
	cancelSvc := NewCoinLookupService(
		NewAgentProxy(server.URL, "token", NewLogger(10)),
		NewSettingsService(repository.NewSettingsRepository(db)),
		NewLogger(10),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cancelSvc.Lookup(ctx, 1, CoinLookupRequest{Images: []string{"data:image/png;base64,AA=="}}); err == nil {
		t.Fatal("cancelled lookup should fail")
	}
}
