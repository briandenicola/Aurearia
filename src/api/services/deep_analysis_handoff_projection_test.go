package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/briandenicola/ancient-coins-api/models"
)

func oversizedDeepAnalysisHandoffResult() DeepAnalysisHandoffResult {
	evidence := make([]DeepAnalysisHandoffEvidence, 0, 160)
	for index := 0; index < 160; index++ {
		evidence = append(evidence, DeepAnalysisHandoffEvidence{
			Provider: "numista",
			Source:   "Numista",
			URL:      fmt.Sprintf("https://en.numista.com/catalogue/pieces%d.html", index+1),
			Summary:  strings.Repeat("whole UTF-8 evidence ευρώ ", 20),
		})
	}
	return DeepAnalysisHandoffResult{
		SchemaVersion: 1,
		Operation:     "status",
		Outcome:       "status",
		Job: &DeepAnalysisHandoffJob{
			ID: 314, Source: "saved_coin", Status: "completed", Reused: true,
			CreatedAt: "2026-09-18T18:00:00Z", CompletedAt: stringPointer("2026-09-18T18:02:00Z"),
		},
		InputDigest:            strings.Repeat("a", 64),
		ReviewURL:              "/deep-analysis/314",
		FreshAnalysisAvailable: true,
		Result: &DeepAnalysisHandoffResultBody{
			State:          "complete",
			Narrative:      "Required lifecycle narrative.",
			PartialSuccess: false,
			ImageOnly:      false,
			Fields: []DeepAnalysisHandoffField{
				{Name: "ruler", Value: "Maximinus I", Confidence: 0.88, Evidence: evidence},
				{Name: "denomination", Value: "Denarius", Confidence: 0.91, Evidence: evidence},
			},
			Disagreements: []DeepAnalysisHandoffDisagreement{
				{Field: "ruler", Summary: "One source differs."},
				{Field: "denomination", Summary: "Weight is ambiguous."},
			},
			UnresolvedQuestions: []string{"Question one?", "Question two?"},
			Coverage: []DeepAnalysisHandoffCoverage{
				{Provider: "numista", Status: "contributed"},
				{Provider: "nomisma", Status: "no_match"},
			},
			Attributions: []DeepAnalysisHandoffAttribution{
				{Provider: "numista", Label: "Numista"},
				{Provider: "nomisma", Label: "Nomisma"},
			},
			Limitations: []string{"Required result limitation."},
		},
		Limitations: []string{"Required top-level limitation."},
	}
}

func stringPointer(value string) *string { return &value }

func handoffSHA256(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func TestDeepAnalysisHandoffProjectionHashesCompleteCanonicalResultBeforeOmission(t *testing.T) {
	result := oversizedDeepAnalysisHandoffResult()
	complete, err := CanonicalDeepAnalysisHandoffResultBytes(result)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := ProjectDeepAnalysisHandoffResult(result, DeepAnalysisHandoffMaxPersistedResultBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Bytes) > DeepAnalysisHandoffMaxPersistedResultBytes {
		t.Fatalf("persisted bytes=%d", len(projection.Bytes))
	}
	if !projection.Metadata.Truncated || projection.Metadata.OriginalBytes != len(complete) ||
		projection.Metadata.PersistedBytes != len(projection.Bytes) {
		t.Fatalf("incorrect truncation metadata: %#v", projection.Metadata)
	}
	if projection.Metadata.Digest != handoffSHA256(complete) {
		t.Fatalf("digest=%q want complete-result digest=%q", projection.Metadata.Digest, handoffSHA256(complete))
	}
	if projection.Metadata.OmittedEvidence == 0 {
		t.Fatal("oversized evidence was not omitted")
	}
}

func TestDeepAnalysisHandoffProjectionIsStableAndKeepsWholeRequiredJSON(t *testing.T) {
	firstInput := oversizedDeepAnalysisHandoffResult()
	secondInput := oversizedDeepAnalysisHandoffResult()
	secondInput.Result.Fields[0], secondInput.Result.Fields[1] =
		secondInput.Result.Fields[1], secondInput.Result.Fields[0]
	secondInput.Result.Coverage[0], secondInput.Result.Coverage[1] =
		secondInput.Result.Coverage[1], secondInput.Result.Coverage[0]
	secondInput.Result.Attributions[0], secondInput.Result.Attributions[1] =
		secondInput.Result.Attributions[1], secondInput.Result.Attributions[0]

	first, err := ProjectDeepAnalysisHandoffResult(firstInput, DeepAnalysisHandoffMaxPersistedResultBytes)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ProjectDeepAnalysisHandoffResult(secondInput, DeepAnalysisHandoffMaxPersistedResultBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes, second.Bytes) || first.Metadata != second.Metadata {
		t.Fatalf("projection changed with input ordering:\nfirst=%#v\nsecond=%#v", first.Metadata, second.Metadata)
	}
	if !utf8.Valid(first.Bytes) || !json.Valid(first.Bytes) {
		t.Fatal("projection sliced UTF-8 or JSON")
	}
	var persisted DeepAnalysisHandoffResult
	if err := json.Unmarshal(first.Bytes, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Outcome != "status" || persisted.Job == nil || persisted.Job.ID != 314 ||
		persisted.ReviewURL != "/deep-analysis/314" || len(persisted.Limitations) == 0 ||
		persisted.Result == nil || len(persisted.Result.Limitations) == 0 {
		t.Fatalf("required lifecycle/link/limitations were omitted: %#v", persisted)
	}
}

func TestDeepAnalysisHandoffProjectionFailsWhenMinimalEnvelopeCannotFit(t *testing.T) {
	result := oversizedDeepAnalysisHandoffResult()
	if _, err := ProjectDeepAnalysisHandoffResult(result, 64); err == nil {
		t.Fatal("non-fitting minimal envelope was accepted")
	}
}

func TestFeature362BuildsOnlyValidatedPersistedDeepResultVariants(t *testing.T) {
	now := time.Now().UTC()
	coinID := uint(41)
	base := models.DeepIdentificationJob{
		ID: 314, UserID: 7, CoinID: &coinID, Source: models.DeepJobSourceSavedCoin,
		Status: models.DeepJobStatusCompleted, InputFingerprint: strings.Repeat("a", 64),
		CompletedAt: &now, CreatedAt: now.Add(-time.Minute),
		ReportJSON: `{
			"narrative":"Persisted report only.",
			"proposed_fields":{
				"ruler":{"value":"Maximinus I","confidence":0.31,"evidence_refs":[
					{"provider":"numista","claim_index":0},{"provider":"image"}
				]}
			},
			"disagreements":[{"field":"ruler","claim_refs":[{"provider":"numista","claim_index":0}],"resolution":"unresolved"}],
			"unresolved_questions":["Reverse legend remains uncertain."],
			"coverage":[
				{"provider":"numista","status":"contributed"},
				{"provider":"rpc","status":"unavailable"}
			],
			"attributions":[{"provider":"numista","text":"Numista attribution"}],
			"face_analyses":[
				{"role":"obverse","status":"completed","narrative":"Radiate portrait.","limitation":""},
				{"role":"reverse","status":"completed","narrative":"Pax reverse.","limitation":""}
			],
			"partial_success":false
		}`,
		ProposalJSON: `{
			"schemaVersion":1,
			"targetCoinId":41,
			"fields":{
				"ruler":{
					"proposed":"Maximinus I",
					"confidence":0.31,
					"evidence":[
						{"field":"ruler","value":"Maximinus I","confidence":0.31,
						 "citation":"https://en.numista.com/catalogue/pieces1.html","excerpt":"Persisted evidence"},
						{"field":"ruler","value":"forged","confidence":0.99,
						 "citation":"https://evil.example/private","excerpt":"must be omitted"}
					],
					"ownerEdited":true,"ownerValue":"private owner edit","accepted":true
				}
			}
		}`,
		Notes: "raw private notes /srv/uploads/secret.jpg Authorization: ******",
	}
	target := &DeepAnalysisHandoffTarget{Type: "coin", ID: 41}
	result, err := BuildDeepAnalysisHandoffResultFromJob("status", target, "Owned denarius", &base)
	if err != nil {
		t.Fatal(err)
	}
	if result.Result == nil || result.Result.State != "complete" ||
		len(result.Result.Fields) != 1 || result.Result.Fields[0].Confidence != 0.31 ||
		len(result.Result.Fields[0].Evidence) != 1 ||
		result.Result.Fields[0].Evidence[0].URL != "https://en.numista.com/catalogue/pieces1.html" ||
		len(result.Result.Disagreements) != 1 || len(result.Result.UnresolvedQuestions) != 1 ||
		len(result.Result.Coverage) != 2 || len(result.Result.Attributions) != 1 {
		t.Fatalf("persisted complete/conflict/provider projection lost validated state: %#v", result.Result)
	}
	encoded, err := CanonicalDeepAnalysisHandoffResultBytes(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"raw private notes", "/srv/uploads", "Authorization", "private owner edit",
		`"accepted"`, `"ownerEdited"`, "evil.example",
	} {
		if bytes.Contains(encoded, []byte(forbidden)) {
			t.Fatalf("projection leaked forbidden persisted/apply data %q: %s", forbidden, encoded)
		}
	}

	t.Run("partial", func(t *testing.T) {
		job := base
		job.Status = models.DeepJobStatusPartial
		job.PartialSuccess = true
		got, err := BuildDeepAnalysisHandoffResultFromJob("status", target, "Owned denarius", &job)
		if err != nil || got.Result == nil || got.Result.State != "partial" || !got.Result.PartialSuccess {
			t.Fatalf("partial projection=%#v err=%v", got.Result, err)
		}
	})
	t.Run("no match", func(t *testing.T) {
		job := base
		job.ReportJSON = `{"narrative":"No reliable match.","proposed_fields":{},"disagreements":[],"unresolved_questions":[],"coverage":[{"provider":"numista","status":"no_match"}],"attributions":[],"partial_success":false}`
		job.ProposalJSON = `{"schemaVersion":1,"fields":{}}`
		got, err := BuildDeepAnalysisHandoffResultFromJob("status", target, "Owned denarius", &job)
		if err != nil || got.Result == nil || got.Result.State != "no_match" {
			t.Fatalf("no-match projection=%#v err=%v", got.Result, err)
		}
	})
	t.Run("image only low confidence", func(t *testing.T) {
		job := base
		job.ReportJSON = `{"narrative":"Image-only hypothesis.","proposed_fields":{"ruler":{"value":"Maximinus I","confidence":0.2,"evidence_refs":[{"provider":"image"}]}},"disagreements":[],"unresolved_questions":[],"coverage":[{"provider":"numista","status":"no_match"}],"attributions":[],"partial_success":false}`
		job.ProposalJSON = `{"schemaVersion":1,"fields":{"ruler":{"proposed":"Maximinus I","confidence":0.2,"ownerEdited":false,"ownerValue":null,"accepted":null}}}`
		got, err := BuildDeepAnalysisHandoffResultFromJob("status", target, "Owned denarius", &job)
		if err != nil || got.Result == nil || !got.Result.ImageOnly ||
			len(got.Result.Fields) != 1 || got.Result.Fields[0].Confidence != 0.2 {
			t.Fatalf("image-only projection=%#v err=%v", got.Result, err)
		}
	})
	t.Run("draft proposal transform does not erase report fields", func(t *testing.T) {
		job := base
		job.Source = models.DeepJobSourceCopilotDraft
		job.CoinID = nil
		draftID := uint(42)
		job.SourceDraftID = &draftID
		job.ReportJSON = `{"narrative":"Image-supported draft attribution.","proposed_fields":{"ruler":{"value":"Maximinus I","confidence":0.4,"evidence_refs":[{"provider":"image"}]}},"disagreements":[],"unresolved_questions":[],"coverage":[],"attributions":[],"partial_success":false}`
		job.ProposalJSON = `{"schemaVersion":1,"fields":{"workingTitle":{"proposed":"Maximinus I coin","confidence":0.4,"ownerEdited":false,"ownerValue":null,"accepted":null}}}`
		draftTarget := &DeepAnalysisHandoffTarget{Type: "draft", ID: draftID}
		if !models.IsValidDeepJobSourceBinding(&job) {
			t.Fatalf("invalid draft source fixture: %#v", job)
		}
		var report persistedDeepHandoffReport
		if err := decodeStrictPersistedJSON(job.ReportJSON, &report); err != nil {
			t.Fatalf("invalid draft report fixture: %v", err)
		}
		var proposal deepProposalDocument
		if err := decodeStrictPersistedJSON(job.ProposalJSON, &proposal); err != nil {
			t.Fatalf("invalid draft proposal fixture: %v", err)
		}
		got, err := BuildDeepAnalysisHandoffResultFromJob("status", draftTarget, "Active draft", &job)
		if err != nil || got.Result == nil || len(got.Result.Fields) != 1 ||
			got.Result.Fields[0].Name != "ruler" || got.Result.Fields[0].Value != "Maximinus I" {
			t.Fatalf("draft projection=%#v err=%v", got.Result, err)
		}
	})
}

func TestFeature362PersistedProjectionRejectsMalformedDeepState(t *testing.T) {
	now := time.Now().UTC()
	target := &DeepAnalysisHandoffTarget{Type: "coin", ID: 41}
	for name, report := range map[string]string{
		"invalid JSON":            `{`,
		"unknown provider":        `{"narrative":"x","proposed_fields":{},"coverage":[{"provider":"forged","status":"contributed"}]}`,
		"out of range confidence": `{"narrative":"x","proposed_fields":{"ruler":{"value":"x","confidence":1.1,"evidence_refs":[{"provider":"image"}]}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			coinID := uint(41)
			job := &models.DeepIdentificationJob{
				ID: 314, UserID: 7, CoinID: &coinID, Source: models.DeepJobSourceSavedCoin,
				Status: models.DeepJobStatusCompleted, CreatedAt: now, CompletedAt: &now,
				InputFingerprint: strings.Repeat("a", 64), ReportJSON: report,
				ProposalJSON: `{"schemaVersion":1,"fields":{}}`,
			}
			if _, err := BuildDeepAnalysisHandoffResultFromJob("status", target, "Owned denarius", job); err == nil {
				t.Fatal("malformed persisted Deep state was projected")
			}
		})
	}
}

func TestFeature362DeliveryUsesIndependentPersistedAndPublicBounds(t *testing.T) {
	service := &CoinCopilotService{}
	payload, err := service.PrepareDeepAnalysisHandoffDelivery(oversizedDeepAnalysisHandoffResult())
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) > DeepAnalysisHandoffMaxPersistedResultBytes {
		t.Fatalf("persisted delivery bytes=%d exceeds %d", len(payload), DeepAnalysisHandoffMaxPersistedResultBytes)
	}
	if err := ValidateDeepAnalysisHandoffPublicEventEnvelope(payload); err != nil {
		t.Fatalf("separately bounded public envelope rejected projected result: %v", err)
	}
	var persisted DeepAnalysisHandoffResult
	if err := json.Unmarshal(payload, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Truncation == nil || !persisted.Truncation.Truncated ||
		persisted.Truncation.PersistedBytes != len(payload) {
		t.Fatalf("missing exact persisted truncation metadata: %#v", persisted.Truncation)
	}
}
