package models

import "testing"

func uintPtr(value uint) *uint { return &value }

func TestDeepIdentificationJobSourceBindingInvariant(t *testing.T) {
	tests := []struct {
		name  string
		job   DeepIdentificationJob
		valid bool
	}{
		{"intake unbound", DeepIdentificationJob{Source: DeepJobSourceIntake}, true},
		{"intake rejects coin", DeepIdentificationJob{Source: DeepJobSourceIntake, CoinID: uintPtr(1)}, false},
		{"intake rejects draft", DeepIdentificationJob{Source: DeepJobSourceIntake, SourceDraftID: uintPtr(1)}, false},
		{"saved coin bound", DeepIdentificationJob{Source: DeepJobSourceSavedCoin, CoinID: uintPtr(1)}, true},
		{"saved coin requires coin", DeepIdentificationJob{Source: DeepJobSourceSavedCoin}, false},
		{"saved coin rejects draft", DeepIdentificationJob{Source: DeepJobSourceSavedCoin, CoinID: uintPtr(1), SourceDraftID: uintPtr(2)}, false},
		{"Copilot draft bound", DeepIdentificationJob{Source: DeepJobSourceCopilotDraft, SourceDraftID: uintPtr(2)}, true},
		{"Copilot draft requires draft", DeepIdentificationJob{Source: DeepJobSourceCopilotDraft}, false},
		{"Copilot draft rejects coin", DeepIdentificationJob{Source: DeepJobSourceCopilotDraft, CoinID: uintPtr(1), SourceDraftID: uintPtr(2)}, false},
		{"unknown source", DeepIdentificationJob{Source: DeepJobSource("future")}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsValidDeepJobSourceBinding(&test.job); got != test.valid {
				t.Fatalf("IsValidDeepJobSourceBinding()=%v want %v", got, test.valid)
			}
		})
	}
}

func TestSupportedDeepJobSourcesIncludesOnlyClosedVocabulary(t *testing.T) {
	got := SupportedDeepJobSources()
	want := []DeepJobSource{DeepJobSourceIntake, DeepJobSourceSavedCoin, DeepJobSourceCopilotDraft}
	if len(got) != len(want) {
		t.Fatalf("sources=%v want=%v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("sources=%v want=%v", got, want)
		}
		if !IsSupportedDeepJobSource(got[index]) {
			t.Fatalf("listed source %q is not supported", got[index])
		}
	}
	for _, source := range []DeepJobSource{"", "future", "wishlist"} {
		if IsSupportedDeepJobSource(source) {
			t.Fatalf("unsupported source %q was accepted", source)
		}
	}
}
