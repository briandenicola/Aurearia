package services

import (
	"context"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
)

// deepIdentificationJanitor runs the retention/recovery sweep loop
// (FR-012/FR-017/FR-034): on boot and every 60s it recovers stale running
// jobs; hourly it prunes events past the retention window and expires
// job/artifact rows past ExpiresAt. It also sweeps hint artifacts left
// un-deleted by a crash (T040 defensive backstop), independent of the
// terminal-hook path.
//
// This is the janitor/retention seam split out of DeepIdentificationService
// (T103). It depends on the artifact store for hint/job artifact deletion,
// but owns no state shared with the worker pool or job-lifecycle seams, so
// it is constructible and testable on its own.
type deepIdentificationJanitor struct {
	repo             *repository.DeepIdentificationRepository
	settingsSvc      *SettingsService
	broker           *DeepIdentificationBroker
	metrics          *deepIdentificationRuntimeMetrics
	providerBudgets  *DeepProviderBudgetTracker
	internalTokenSvc *InternalTokenService
	logger           *Logger
	artifacts        *deepIdentificationArtifactStore
}

// newDeepIdentificationJanitor constructs the janitor, following the
// repo -> service -> handler DI pattern (Principle I).
func newDeepIdentificationJanitor(
	repo *repository.DeepIdentificationRepository,
	settingsSvc *SettingsService,
	broker *DeepIdentificationBroker,
	metrics *deepIdentificationRuntimeMetrics,
	logger *Logger,
	artifacts *deepIdentificationArtifactStore,
) *deepIdentificationJanitor {
	return &deepIdentificationJanitor{
		repo:        repo,
		settingsSvc: settingsSvc,
		broker:      broker,
		metrics:     metrics,
		logger:      logger,
		artifacts:   artifacts,
	}
}

// setProviderBudgetTracker/setInternalTokenService mirror
// DeepIdentificationService.SetProviderBudgetTracker/SetInternalTokenService
// (T078/T081): both are wired in after construction (main.go's object
// graph), and both are nil-guarded at every call site so the janitor works
// without them in tests that don't need budget/token cleanup.
func (j *deepIdentificationJanitor) setProviderBudgetTracker(tracker *DeepProviderBudgetTracker) {
	j.providerBudgets = tracker
}

func (j *deepIdentificationJanitor) setInternalTokenService(tokenSvc *InternalTokenService) {
	j.internalTokenSvc = tokenSvc
}

// StartJanitor runs the retention/recovery sweep loop (FR-012/FR-017/
// FR-034): on boot and every 60s it recovers stale running jobs; hourly it
// prunes events past the retention window and expires job/artifact rows
// past ExpiresAt. It also sweeps hint artifacts left un-deleted by a crash
// (T040 defensive backstop), independent of the terminal-hook path.
func (j *deepIdentificationJanitor) StartJanitor(ctx context.Context) {
	j.recoverStaleAndSweepHints()

	staleTicker := time.NewTicker(60 * time.Second)
	retentionTicker := time.NewTicker(time.Hour)
	go func() {
		defer staleTicker.Stop()
		defer retentionTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-staleTicker.C:
				j.recoverStaleAndSweepHints()
			case <-retentionTicker.C:
				j.runRetentionSweep()
			}
		}
	}()
}

func (j *deepIdentificationJanitor) recoverStaleAndSweepHints() {
	j.metrics.janitorRecoverySweeps.Add(1)
	failed := false
	settings := j.settingsSvc.GetDeepIdentificationSettings()
	staleAfter := settings.HardTimeout + 2*time.Minute
	recoveredIDs, err := j.repo.RecoverStaleJobs(staleAfter)
	if err != nil {
		failed = true
		if j.logger != nil {
			j.logger.Error("deep-identification", "failed to recover stale jobs: %v", err)
		}
	}
	for _, id := range recoveredIDs {
		j.broker.Publish(id)
		// Same terminal-cleanup guarantee as runJob's own terminal path
		// (T078): a job whose worker goroutine never returned (e.g. wedged
		// past a restart) still needs its tracked budget entries released
		// once the janitor forces it to a terminal state.
		if j.providerBudgets != nil {
			j.providerBudgets.Reset(id)
		}
		if j.internalTokenSvc != nil {
			j.internalTokenSvc.RevokeJob(id)
		}
		if err := j.artifacts.DeleteHintArtifacts(id); err != nil {
			failed = true
			if j.logger != nil {
				j.logger.Error("deep-identification", "failed to clean up hints for recovered job %d: %v", id, err)
			}
		}
	}

	// Defensive backstop: sweep hint artifacts for any terminal job that a
	// crash left un-cleaned (independent of the two hooks above).
	jobIDs, err := j.repo.ListJobIDsWithUndeletedHintArtifacts()
	if err != nil {
		failed = true
		if j.logger != nil {
			j.logger.Error("deep-identification", "failed to list jobs with undeleted hints: %v", err)
		}
		if failed {
			j.metrics.janitorFailures.Add(1)
		}
		return
	}
	for _, id := range jobIDs {
		if err := j.artifacts.DeleteHintArtifacts(id); err != nil {
			failed = true
			if j.logger != nil {
				j.logger.Error("deep-identification", "failed to sweep hints for job %d: %v", id, err)
			}
		}
	}
	if failed {
		j.metrics.janitorFailures.Add(1)
	}
	if j.logger != nil {
		j.logger.Debug("deep-identification", "janitor recovery sweep recovered_jobs=%d hint_jobs=%d failed=%t", len(recoveredIDs), len(jobIDs), failed)
	}
}

func (j *deepIdentificationJanitor) runRetentionSweep() {
	j.metrics.janitorRetentionSweep.Add(1)
	failed := false
	settings := j.settingsSvc.GetDeepIdentificationSettings()
	cutoff := time.Now().Add(-settings.EventRetention)
	if err := j.repo.PruneEventsBefore(cutoff); err != nil {
		failed = true
		if j.logger != nil {
			j.logger.Error("deep-identification", "failed to prune events: %v", err)
		}
	}
	expiredIDs, err := j.repo.ListExpiredJobIDs(time.Now())
	if err != nil {
		failed = true
		if j.logger != nil {
			j.logger.Error("deep-identification", "failed to list expired jobs: %v", err)
		}
		j.metrics.janitorFailures.Add(1)
		return
	}
	for _, id := range expiredIDs {
		if err := j.artifacts.DeleteJobArtifacts(id); err != nil {
			failed = true
			if j.logger != nil {
				j.logger.Error("deep-identification", "failed to delete artifacts for expired job %d: %v", id, err)
			}
		}
	}
	if failed {
		j.metrics.janitorFailures.Add(1)
	}
	if j.logger != nil {
		j.logger.Debug("deep-identification", "janitor retention sweep expired_jobs=%d failed=%t", len(expiredIDs), failed)
	}
}
