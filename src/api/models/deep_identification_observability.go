package models

// DeepIdentificationLatencySummary is a bounded latency percentile view in
// milliseconds. Raw timings and per-job identifiers are intentionally omitted.
type DeepIdentificationLatencySummary struct {
	P50MS int64 `json:"p50Ms"`
	P95MS int64 `json:"p95Ms"`
}

// DeepIdentificationProviderMetrics summarizes durable provider attempts
// without exposing claims, citations, queries, or other job content.
type DeepIdentificationProviderMetrics struct {
	StatusCounts map[DeepProviderRunStatus]int64  `json:"statusCounts"`
	Latency      DeepIdentificationLatencySummary `json:"latency"`
}

// DeepIdentificationHintDeletionMetrics reports runtime hint-cleanup outcomes.
type DeepIdentificationHintDeletionMetrics struct {
	Success int64 `json:"success"`
	Failure int64 `json:"failure"`
}

// DeepIdentificationJanitorMetrics reports runtime sweep invocations and
// failures since the current API process started.
type DeepIdentificationJanitorMetrics struct {
	RecoverySweeps  int64 `json:"recoverySweeps"`
	RetentionSweeps int64 `json:"retentionSweeps"`
	Failures        int64 `json:"failures"`
}

// DeepIdentificationObservabilitySummary is the redacted admin-only
// operational view for Deep Identification.
type DeepIdentificationObservabilitySummary struct {
	JobsByTerminalStatus map[DeepJobStatus]int64                                `json:"jobsByTerminalStatus"`
	PartialSuccessRate   float64                                                `json:"partialSuccessRate"`
	Duration             DeepIdentificationLatencySummary                       `json:"duration"`
	Providers            map[DeepProviderName]DeepIdentificationProviderMetrics `json:"providers"`
	ActiveSSEStreams     int64                                                  `json:"activeSseStreams"`
	ReconnectCount       int64                                                  `json:"reconnectCount"`
	TruncationCount      int64                                                  `json:"truncationCount"`
	QueueDepth           int64                                                  `json:"queueDepth"`
	HintDeletion         DeepIdentificationHintDeletionMetrics                  `json:"hintDeletion"`
	Janitor              DeepIdentificationJanitorMetrics                       `json:"janitor"`
}
