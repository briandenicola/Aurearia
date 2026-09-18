package services

// Feature 360's Go-to-Python DTOs are intentionally isolated from handlers and
// persistence. They mirror exploration-agent.openapi.yaml exactly.

const (
	ExplorationDecisionSchemaVersion = "aurearia.browser-exploration-decision/v1"
	ExplorationDecisionPath          = "/internal/browser-exploration/decide"
)

type ExplorationDecisionRequest struct {
	SchemaVersion  string                   `json:"schemaVersion"`
	RunID          string                   `json:"runId"`
	StepID         string                   `json:"stepId"`
	Provider       string                   `json:"provider"`
	Model          string                   `json:"model"`
	Workflow       string                   `json:"workflow"`
	Goal           string                   `json:"goal"`
	AllowedActions []string                 `json:"allowedActions"`
	AllowedRoutes  []string                 `json:"allowedRoutes"`
	Observations   []ExplorationObservation `json:"observations"`
}

type ExplorationObservation struct {
	EvidenceID string `json:"evidenceId"`
	Kind       string `json:"kind"`
	Route      string `json:"route"`
	Summary    string `json:"summary"`
}

type ExplorationDecisionResponse struct {
	SchemaVersion     string                   `json:"schemaVersion"`
	Action            string                   `json:"action"`
	Target            *ExplorationActionTarget `json:"target"`
	Rationale         string                   `json:"rationale"`
	SuspectedFindings []ExplorationModelTriage `json:"suspectedFindings"`
	Usage             ExplorationModelUsage    `json:"usage"`
}

type ExplorationActionTarget struct {
	Route          string `json:"route,omitempty"`
	Role           string `json:"role,omitempty"`
	AccessibleName string `json:"accessibleName,omitempty"`
	Label          string `json:"label,omitempty"`
	Value          string `json:"value,omitempty"`
	FixtureID      string `json:"fixtureId,omitempty"`
	Width          int    `json:"width,omitempty"`
	Height         int    `json:"height,omitempty"`
	Milliseconds   int    `json:"milliseconds,omitempty"`
}

type ExplorationModelTriage struct {
	GeneratedByModel  bool     `json:"generatedByModel"`
	Summary           string   `json:"summary"`
	SuggestedCategory string   `json:"suggestedCategory"`
	SuggestedSeverity string   `json:"suggestedSeverity"`
	Confidence        float64  `json:"confidence"`
	EvidenceIDs       []string `json:"evidenceIds"`
}

type ExplorationModelUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}
