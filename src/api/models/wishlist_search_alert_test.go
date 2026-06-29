package models

import "testing"

func TestWishlistSearchAlertEnums(t *testing.T) {
	if WishlistAlertCadenceManual != "manual" || WishlistAlertCadenceWeekly != "weekly" {
		t.Fatalf("unexpected cadence enum values")
	}
	if AlertCandidateStateActive != "active" || CandidateProvenanceVerified != "verified" {
		t.Fatalf("unexpected candidate enum values")
	}
	var _ = Coin{SourceAlertCandidateID: nil}
}
