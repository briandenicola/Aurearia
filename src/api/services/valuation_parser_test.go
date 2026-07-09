package services

import (
	"testing"
)

func TestParseValueEstimate_JSONBlock(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantValue  float64
		wantConf   string
		wantReason string
		wantComps  int
	}{
		{
			name: "valid JSON with all fields",
			input: "Here is the estimate:\n```json\n" +
				`{"estimatedValue": 250.00, "confidence": "high", "reasoning": "Well-preserved denarius", "comparables": [{"source": "CNG", "price": "$275", "url": "https://cng.com/1"}]}` +
				"\n```\nEnd.",
			wantValue:  250.00,
			wantConf:   "high",
			wantReason: "Well-preserved denarius",
			wantComps:  1,
		},
		{
			name: "valid JSON without confidence defaults to medium",
			input: "```json\n" +
				`{"estimatedValue": 100, "reasoning": "Common type"}` +
				"\n```",
			wantValue:  100,
			wantConf:   "medium",
			wantReason: "Common type",
			wantComps:  0,
		},
		{
			name: "valid JSON with multiple comparables",
			input: "```json\n" +
				`{"estimatedValue": 500, "confidence": "low", "reasoning": "Rare", "comparables": [` +
				`{"source": "A", "price": "$450", "url": "https://a.com"},` +
				`{"source": "B", "price": "$550", "url": "https://b.com"}]}` +
				"\n```",
			wantValue:  500,
			wantConf:   "low",
			wantReason: "Rare",
			wantComps:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseValueEstimate(tt.input)
			if result.EstimatedValue != tt.wantValue {
				t.Errorf("EstimatedValue = %v, want %v", result.EstimatedValue, tt.wantValue)
			}
			if result.Confidence != tt.wantConf {
				t.Errorf("Confidence = %q, want %q", result.Confidence, tt.wantConf)
			}
			if result.Reasoning != tt.wantReason {
				t.Errorf("Reasoning = %q, want %q", result.Reasoning, tt.wantReason)
			}
			if len(result.Comparables) != tt.wantComps {
				t.Errorf("Comparables count = %d, want %d", len(result.Comparables), tt.wantComps)
			}
		})
	}
}

func TestParseValueEstimate_JSONBlockEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantJSON  bool // if true, expect JSON path was taken (exact value match)
		wantValue float64
	}{
		{
			name:      "zero value in JSON falls through to regex",
			input:     "```json\n{\"estimatedValue\": 0, \"confidence\": \"high\"}\n```",
			wantJSON:  false,
			wantValue: 0, // no dollar amount in text either
		},
		{
			name:      "negative value in JSON falls through to regex",
			input:     "```json\n{\"estimatedValue\": -50, \"confidence\": \"high\"}\n```",
			wantJSON:  false,
			wantValue: 0,
		},
		{
			name:      "malformed JSON falls through to regex",
			input:     "```json\n{broken json\n```\nThe coin is worth $300.",
			wantJSON:  false,
			wantValue: 300,
		},
		{
			name:      "unclosed JSON block falls through",
			input:     "```json\n{\"estimatedValue\": 100}",
			wantJSON:  false,
			wantValue: 0,
		},
		{
			name:      "no JSON block marker",
			input:     "{\"estimatedValue\": 100}",
			wantJSON:  false,
			wantValue: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseValueEstimate(tt.input)
			if result.EstimatedValue != tt.wantValue {
				t.Errorf("EstimatedValue = %v, want %v", result.EstimatedValue, tt.wantValue)
			}
		})
	}
}

func TestParseValueEstimate_PlainTextRegex(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue float64
		wantConf  string
	}{
		{
			name:      "single dollar amount",
			input:     "This coin is worth approximately $150 in current market.",
			wantValue: 150,
			wantConf:  "medium",
		},
		{
			name:      "dollar range averaged",
			input:     "The estimated value is $100-$200 based on recent sales.",
			wantValue: 150, // (100+200)/2
			wantConf:  "medium",
		},
		{
			name:      "dollar amount with cents",
			input:     "Fair market value is $1,250.50 at auction.",
			wantValue: 1250.50,
			wantConf:  "medium",
		},
		{
			name:      "high confidence text",
			input:     "The coin is worth $500. High confidence based on multiple sales.",
			wantValue: 500,
			wantConf:  "high",
		},
		{
			name:      "low confidence text",
			input:     "Roughly $75. Low confidence due to limited comparables.",
			wantValue: 75,
			wantConf:  "low",
		},
		{
			name:      "confidence: high pattern",
			input:     "Value $200. Confidence: high.",
			wantValue: 200,
			wantConf:  "high",
		},
		{
			name:      "no dollar amount",
			input:     "Unable to determine a value for this coin.",
			wantValue: 0,
			wantConf:  "medium",
		},
		{
			name:      "empty string",
			input:     "",
			wantValue: 0,
			wantConf:  "medium",
		},
		{
			name:      "comma-separated thousands",
			input:     "This coin sold for $12,500 at the last auction.",
			wantValue: 12500,
			wantConf:  "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseValueEstimate(tt.input)
			if result.EstimatedValue != tt.wantValue {
				t.Errorf("EstimatedValue = %v, want %v", result.EstimatedValue, tt.wantValue)
			}
			if result.Confidence != tt.wantConf {
				t.Errorf("Confidence = %q, want %q", result.Confidence, tt.wantConf)
			}
			if result.Comparables == nil {
				t.Error("Comparables should not be nil (expect empty slice)")
			}
		})
	}
}

func TestParseValueEstimate_ReasoningExtraction(t *testing.T) {
	// Verify reasoning is extracted from plain text (non-empty for value-related text)
	input := "Based on market conditions, the estimated value of this Roman denarius is $300. " +
		"The grade is VF and comparable auction results support this price range. " +
		"The coin shows typical wear for the period."
	result := ParseValueEstimate(input)

	if result.Reasoning == "" {
		t.Error("expected non-empty reasoning for text with value keywords")
	}
	if result.EstimatedValue != 300 {
		t.Errorf("EstimatedValue = %v, want 300", result.EstimatedValue)
	}
}

func TestParseValueEstimate_ChangeExplanation(t *testing.T) {
	input := "```json\n" +
		`{"estimatedValue": 325, "confidence": "high", "reasoning": "Recent sales support the estimate.", "changeExplanation": "The value increased because comparable VF examples are selling above the prior estimate."}` +
		"\n```"

	result := ParseValueEstimate(input)
	if result.ChangeExplanation != "The value increased because comparable VF examples are selling above the prior estimate." {
		t.Errorf("ChangeExplanation = %q", result.ChangeExplanation)
	}
}

func TestParseValueEstimate_ChangeExplanationBackwardCompatible(t *testing.T) {
	input := "```json\n" +
		`{"estimatedValue": 325, "confidence": "high", "reasoning": "Recent sales support the estimate."}` +
		"\n```"

	result := ParseValueEstimate(input)
	if result.ChangeExplanation != "" {
		t.Errorf("ChangeExplanation = %q, want empty", result.ChangeExplanation)
	}
}
