package services

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// ValueEstimate holds the parsed AI valuation response.
type ValueEstimate struct {
	EstimatedValue float64               `json:"estimatedValue"`
	Confidence     string                `json:"confidence"`
	Reasoning      string                `json:"reasoning"`
	Comparables    []ValueEstimateComp   `json:"comparables"`
}

// ValueEstimateComp represents a comparable sale found by the AI.
type ValueEstimateComp struct {
	Source string `json:"source"`
	Price  string `json:"price"`
	URL    string `json:"url"`
}

// ParseValueEstimate extracts structured fields from the AI response.
// First tries to parse a ```json block (the prompt requests one), then
// falls back to regex extraction from free text.
func ParseValueEstimate(text string) ValueEstimate {
	if result := tryParseJSONEstimate(text); result != nil {
		return *result
	}

	result := ValueEstimate{
		Confidence:  "medium",
		Reasoning:   summarizeReasoning(text),
		Comparables: []ValueEstimateComp{},
	}

	// Extract dollar amount: patterns like $150, $150-200, $1,500
	priceRe := regexp.MustCompile(`\$[\d,]+(?:\.\d{2})?(?:\s*[-–]\s*\$?[\d,]+(?:\.\d{2})?)?`)
	if match := priceRe.FindString(text); match != "" {
		numRe := regexp.MustCompile(`[\d,]+(?:\.\d{2})?`)
		nums := numRe.FindAllString(match, -1)
		if len(nums) > 0 {
			first := parsePrice(nums[0])
			if len(nums) > 1 {
				second := parsePrice(nums[len(nums)-1])
				result.EstimatedValue = (first + second) / 2
			} else {
				result.EstimatedValue = first
			}
		}
	}

	// Extract confidence
	lower := strings.ToLower(text)
	if strings.Contains(lower, "high confidence") || strings.Contains(lower, "confidence: high") || strings.Contains(lower, "confidence level: high") {
		result.Confidence = "high"
	} else if strings.Contains(lower, "low confidence") || strings.Contains(lower, "confidence: low") || strings.Contains(lower, "confidence level: low") {
		result.Confidence = "low"
	}

	return result
}

func tryParseJSONEstimate(text string) *ValueEstimate {
	start := strings.Index(text, "```json")
	if start == -1 {
		return nil
	}
	start += len("```json")
	end := strings.Index(text[start:], "```")
	if end == -1 {
		return nil
	}
	jsonStr := strings.TrimSpace(text[start : start+end])

	var parsed struct {
		EstimatedValue float64 `json:"estimatedValue"`
		Confidence     string  `json:"confidence"`
		Reasoning      string  `json:"reasoning"`
		Comparables    []struct {
			Source string `json:"source"`
			Price  string `json:"price"`
			URL    string `json:"url"`
		} `json:"comparables"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil
	}

	if parsed.EstimatedValue <= 0 {
		return nil
	}

	comps := make([]ValueEstimateComp, 0, len(parsed.Comparables))
	for _, c := range parsed.Comparables {
		comps = append(comps, ValueEstimateComp{Source: c.Source, Price: c.Price, URL: c.URL})
	}

	confidence := parsed.Confidence
	if confidence == "" {
		confidence = "medium"
	}

	return &ValueEstimate{
		EstimatedValue: parsed.EstimatedValue,
		Confidence:     confidence,
		Reasoning:      parsed.Reasoning,
		Comparables:    comps,
	}
}

func summarizeReasoning(text string) string {
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "##", "")
	text = strings.ReplaceAll(text, "# ", "")

	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return text
	}

	keywords := []string{"estimat", "value", "price", "worth", "$", "market", "condition", "grade", "comparable", "range", "auction"}
	var relevant []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) < 15 {
			continue
		}
		lower := strings.ToLower(s)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				relevant = append(relevant, s)
				break
			}
		}
	}

	if len(relevant) == 0 {
		limit := 3
		if len(sentences) < limit {
			limit = len(sentences)
		}
		return strings.Join(sentences[:limit], " ")
	}

	limit := 3
	if len(relevant) < limit {
		limit = len(relevant)
	}
	return strings.Join(relevant[:limit], " ")
}

func splitSentences(text string) []string {
	re := regexp.MustCompile(`[.!?]\s+`)
	parts := re.Split(text, -1)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p+".")
		}
	}
	return result
}

func parsePrice(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	val, _ := strconv.ParseFloat(s, 64)
	return val
}
