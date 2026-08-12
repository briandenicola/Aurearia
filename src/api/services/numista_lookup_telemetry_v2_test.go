package services

import (
	"context"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func TestNumistaLookupTelemetryAttributesSafeSourceAndAttemptEnums(t *testing.T) {
	client := &sequenceNumistaClient{results: map[string][]models.NumistaCandidate{
		"Honorius GLORIA ROMANORVM Nicomedia": {},
		"Honorius Nicomedia":                  {},
	}}
	settings := &fakeNumistaSettings{key: "configured", config: NumistaSettings{
		SearchTTL: time.Hour, SearchResultLimit: 20, Valid: true,
	}}
	telemetry := NewNumistaTelemetry(10)
	service := NewNumistaLookupService(
		client, NewNumistaCache(nil, 10, 10), NewNumistaV1Scorer(),
		telemetry, settings, nil, NewNumistaQueryBuilder(),
	)
	_, err := service.Lookup(context.Background(), models.NumistaLookupRequest{
		Query: "Honorius GLORIA ROMANORVM Nicomedia",
		Path:  models.NumistaLookupPathDirect,
		Evidence: models.NumistaEvidence{
			Issuer: "Honorius", Mint: "SMNT", ReverseInscription: "GLORIA ROMANORVM",
		},
		QuerySource:       models.NumistaQuerySourceGenerated,
		GenerationVersion: models.NumistaQueryGenerationVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	summary := telemetry.Health(true, true)
	if summary.GeneratedQueryCount != 2 || summary.RelaxedAttemptCount != 1 ||
		summary.UserEditedQueryCount != 0 || summary.ManualQueryCount != 0 {
		t.Fatalf("safe query attribution summary=%+v", summary)
	}
	telemetry.mu.RLock()
	events := append([]NumistaTelemetryEvent(nil), telemetry.events...)
	telemetry.mu.RUnlock()
	if len(events) != 2 ||
		events[0].Source != models.NumistaQuerySourceGenerated ||
		events[0].Attempt != models.NumistaSearchAttemptPrimary ||
		events[1].Source != models.NumistaQuerySourceGenerated ||
		events[1].Attempt != models.NumistaSearchAttemptRelaxed {
		t.Fatalf("safe operation attribution events=%+v", events)
	}
}
