package models

import "time"

// OCREHealthSummary is the bounded, admin-only operational view of the OCRE
// Deep Analysis provider (Feature 345 US4, FR-034). It exposes only the
// enablement/configuration state and the last recorded provider-run outcome
// class — never any per-job user content (notes, legends, claims, citations).
type OCREHealthSummary struct {
	// Enabled reflects the current DeepIdentificationOCREEnabled flag.
	Enabled bool `json:"enabled"`
	// CallBudget is the current per-job OCRE call budget.
	CallBudget int `json:"callBudget"`
	// GateValidated reports whether the deep-identification settings snapshot
	// parsed to a valid enablement gate (bounded flag + budget). A false value
	// means an operator must correct configuration before OCRE can be trusted.
	GateValidated bool `json:"gateValidated"`
	// LastOutcome is the status of the most recent OCRE provider-run row, or
	// nil when OCRE has never run.
	LastOutcome *DeepProviderRunStatus `json:"lastOutcome"`
	// LastCheckedAt is the created_at of that most recent OCRE provider-run
	// row, or nil when OCRE has never run.
	LastCheckedAt *time.Time `json:"lastCheckedAt"`
}
