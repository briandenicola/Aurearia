package services

import "testing"

func TestMatchKnownValue_ExactMatch(t *testing.T) {
	match, ok := MatchKnownValue("roman", []string{"Roman", "Greek", "Byzantine"})
	if !ok || match != "Roman" {
		t.Fatalf("expected exact match Roman, got %q ok=%v", match, ok)
	}
}

func TestMatchKnownValue_NormalizedNearMiss(t *testing.T) {
	match, ok := MatchKnownValue("Roman Republic", []string{"Roman", "Greek"})
	if !ok || match != "Roman" {
		t.Fatalf("expected near-miss match Roman, got %q ok=%v", match, ok)
	}
}

func TestMatchKnownValue_NoConfidentMatch(t *testing.T) {
	_, ok := MatchKnownValue("Sassanian", []string{"Roman", "Greek", "Byzantine"})
	if ok {
		t.Fatalf("expected no confident match for an unrelated value")
	}
}

func TestMatchKnownValue_EmptyCandidate(t *testing.T) {
	_, ok := MatchKnownValue("   ", []string{"Roman"})
	if ok {
		t.Fatalf("expected blank candidate to never match")
	}
}
