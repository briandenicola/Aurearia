package services

import (
	"sync/atomic"

	"github.com/briandenicola/ancient-coins-api/models"
)

type deepIdentificationRuntimeMetrics struct {
	activeSSEStreams      atomic.Int64
	reconnectCount        atomic.Int64
	truncationCount       atomic.Int64
	hintDeletionSuccess   atomic.Int64
	hintDeletionFailure   atomic.Int64
	janitorRecoverySweeps atomic.Int64
	janitorRetentionSweep atomic.Int64
	janitorFailures       atomic.Int64
}

// BeginSSEStream records a live stream and returns a completion callback.
// The handler must defer the callback after the stream is subscribed.
func (s *DeepIdentificationService) BeginSSEStream(reconnect bool) func() {
	s.metrics.activeSSEStreams.Add(1)
	if reconnect {
		s.metrics.reconnectCount.Add(1)
	}
	return func() {
		s.metrics.activeSSEStreams.Add(-1)
	}
}

// RecordSSETruncation records that a reconnect could not be fully replayed.
func (s *DeepIdentificationService) RecordSSETruncation() {
	s.metrics.truncationCount.Add(1)
}

// GetObservabilitySummary combines durable repository aggregates with
// process-local counters. It contains operational metadata only.
func (s *DeepIdentificationService) GetObservabilitySummary() (*models.DeepIdentificationObservabilitySummary, error) {
	durable, err := s.repo.GetObservabilityMetrics()
	if err != nil {
		return nil, err
	}
	for _, provider := range []models.DeepProviderName{
		models.DeepProviderNomisma,
		models.DeepProviderNumista,
		models.DeepProviderNGC,
		models.DeepProviderOCRE,
		models.DeepProviderRPC,
	} {
		item := durable.Providers[provider]
		if item.StatusCounts == nil {
			item.StatusCounts = make(map[models.DeepProviderRunStatus]int64)
		}
		durable.Providers[provider] = item
	}
	return &models.DeepIdentificationObservabilitySummary{
		JobsByTerminalStatus: durable.JobsByTerminalStatus,
		PartialSuccessRate:   durable.PartialSuccessRate,
		Duration:             durable.Duration,
		Providers:            durable.Providers,
		ActiveSSEStreams:     s.metrics.activeSSEStreams.Load(),
		ReconnectCount:       s.metrics.reconnectCount.Load(),
		TruncationCount:      s.metrics.truncationCount.Load(),
		QueueDepth:           durable.QueueDepth,
		HintDeletion: models.DeepIdentificationHintDeletionMetrics{
			Success: s.metrics.hintDeletionSuccess.Load(),
			Failure: s.metrics.hintDeletionFailure.Load(),
		},
		Janitor: models.DeepIdentificationJanitorMetrics{
			RecoverySweeps:  s.metrics.janitorRecoverySweeps.Load(),
			RetentionSweeps: s.metrics.janitorRetentionSweep.Load(),
			Failures:        s.metrics.janitorFailures.Load(),
		},
	}, nil
}
