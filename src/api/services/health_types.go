package services

// HealthGrade is the letter grade for a computed health score.
type HealthGrade string

const (
	HealthGradeA HealthGrade = "A"
	HealthGradeB HealthGrade = "B"
	HealthGradeC HealthGrade = "C"
	HealthGradeD HealthGrade = "D"
	HealthGradeF HealthGrade = "F"
)

// HealthDimension identifies a score dimension in the scorecard.
type HealthDimension string

const (
	HealthDimensionMetadata           HealthDimension = "metadata"
	HealthDimensionImageCoverage      HealthDimension = "imageCoverage"
	HealthDimensionValuationFreshness HealthDimension = "valuationFreshness"
	HealthDimensionAICoverage         HealthDimension = "aiCoverage"
)

// ChecklistDimension identifies the checklist bucket for missing data.
type ChecklistDimension string

const (
	ChecklistDimensionMetadata  ChecklistDimension = "metadata"
	ChecklistDimensionImages    ChecklistDimension = "images"
	ChecklistDimensionValuation ChecklistDimension = "valuation"
	ChecklistDimensionAI        ChecklistDimension = "ai"
)

// ChecklistSeverity indicates how urgent a checklist item is.
type ChecklistSeverity string

const (
	ChecklistSeverityHigh   ChecklistSeverity = "high"
	ChecklistSeverityMedium ChecklistSeverity = "medium"
	ChecklistSeverityLow    ChecklistSeverity = "low"
)

// HealthActionHint maps checklist findings to existing user workflows.
type HealthActionHint string

const (
	HealthActionEditMetadata  HealthActionHint = "edit_metadata"
	HealthActionUploadImages  HealthActionHint = "upload_images"
	HealthActionRunValuation  HealthActionHint = "run_valuation"
	HealthActionRunAIAnalysis HealthActionHint = "run_ai_analysis"
)

const (
	HealthWeightMetadata           = 40
	HealthWeightImageCoverage      = 20
	HealthWeightValuationFreshness = 20
	HealthWeightAICoverage         = 20
	HealthLowScoreThreshold        = 60
)

// HealthWeights is the fixed v1 scoring weight map.
type HealthWeights struct {
	Metadata           int `json:"metadata"`
	ImageCoverage      int `json:"imageCoverage"`
	ValuationFreshness int `json:"valuationFreshness"`
	AICoverage         int `json:"aiCoverage"`
}

// HealthDimensionScores is the normalized 0..100 score per dimension.
type HealthDimensionScores struct {
	Metadata           int `json:"metadata"`
	ImageCoverage      int `json:"imageCoverage"`
	ValuationFreshness int `json:"valuationFreshness"`
	AICoverage         int `json:"aiCoverage"`
}

// HealthTrend30D stores trend signal and delta against the baseline snapshot.
type HealthTrend30D struct {
	Status    string `json:"status"`
	Delta     *int   `json:"delta"`
	Direction string `json:"direction"`
}

// MissingChecklistItem is an actionable gap in a coin's record.
type MissingChecklistItem struct {
	Key        string             `json:"key"`
	Dimension  ChecklistDimension `json:"dimension"`
	Label      string             `json:"label"`
	Severity   ChecklistSeverity  `json:"severity"`
	ActionHint HealthActionHint   `json:"actionHint"`
}

// CollectionHealthSummary is the collection-level health response model.
type CollectionHealthSummary struct {
	Score             int                   `json:"score"`
	Grade             HealthGrade           `json:"grade"`
	EligibleCoinCount int64                 `json:"eligibleCoinCount"`
	Weights           HealthWeights         `json:"weights"`
	Dimensions        HealthDimensionScores `json:"dimensions"`
	Trend30D          HealthTrend30D        `json:"trend30d"`
}

// CoinHealthItem is the per-coin health response model.
type CoinHealthItem struct {
	CoinID       uint                   `json:"coinId"`
	Title        string                 `json:"title"`
	Score        int                    `json:"score"`
	Grade        HealthGrade            `json:"grade"`
	Dimensions   HealthDimensionScores  `json:"dimensions"`
	MissingItems []MissingChecklistItem `json:"missingItems"`
	QuickActions []HealthActionHint     `json:"quickActions"`
}

// HealthPagination is the paging metadata for coin health lists.
type HealthPagination struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// CoinHealthListResponse is the list response for /coins/health.
type CoinHealthListResponse struct {
	Coins      []CoinHealthItem `json:"coins"`
	Pagination HealthPagination `json:"pagination"`
}

// MissingFieldStat captures top aggregate missing fields for admin views.
type MissingFieldStat struct {
	Key        string  `json:"key"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// AdminHealthSummary is the admin aggregate response model.
type AdminHealthSummary struct {
	MedianScore        int                `json:"medianScore"`
	LowScorePercentage float64            `json:"lowScorePercentage"`
	LowScoreThreshold  int                `json:"lowScoreThreshold"`
	EligibleCoinCount  int64              `json:"eligibleCoinCount"`
	TopMissingFields   []MissingFieldStat `json:"topMissingFields"`
}

// DefaultHealthWeights returns fixed v1 weights.
func DefaultHealthWeights() HealthWeights {
	return HealthWeights{
		Metadata:           HealthWeightMetadata,
		ImageCoverage:      HealthWeightImageCoverage,
		ValuationFreshness: HealthWeightValuationFreshness,
		AICoverage:         HealthWeightAICoverage,
	}
}

// ClampHealthScore keeps score values in the expected 0..100 range.
func ClampHealthScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// HealthGradeFromScore maps a normalized score to a fixed letter grade.
func HealthGradeFromScore(score int) HealthGrade {
	s := ClampHealthScore(score)
	switch {
	case s >= 90:
		return HealthGradeA
	case s >= 80:
		return HealthGradeB
	case s >= 70:
		return HealthGradeC
	case s >= 60:
		return HealthGradeD
	default:
		return HealthGradeF
	}
}
