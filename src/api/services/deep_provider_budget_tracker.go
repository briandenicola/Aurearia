package services

import (
	"fmt"
	"sync"
)

// DeepProviderBudgetTracker enforces a per-job, per-provider call budget for
// the Go-hosted provider tool endpoints called by the Python deep
// identification pipeline (contracts/agent-internal-contract.md §7:
// numista_search/numista_detail count against the Numista budget,
// nomisma_search against the fixed Nomisma budget). Budgets are tracked
// in-process for the lifetime of a job run - Python holds no state
// (Principle II) and Go is the sole authority for enforcement.
//
// It is safe for concurrent use: the bounded provider fan-out
// (contracts/agent-internal-contract.md §6) may call these endpoints from
// multiple goroutines for the same job at once.
type DeepProviderBudgetTracker struct {
	mu     sync.Mutex
	counts map[string]int
}

// NewDeepProviderBudgetTracker creates an empty tracker.
func NewDeepProviderBudgetTracker() *DeepProviderBudgetTracker {
	return &DeepProviderBudgetTracker{counts: make(map[string]int)}
}

func budgetKey(jobID uint, provider string) string {
	return fmt.Sprintf("%d:%s", jobID, provider)
}

// TryConsume increments the call count for (jobID, provider) and reports
// whether the call is within budget. A non-positive budget is treated as
// unlimited (never expected in practice - all callers pass a validated
// positive budget). When the budget is exceeded the count is NOT
// incremented further and callCount reflects the count at rejection time,
// so callers can log/telemetry the over-budget attempt without it silently
// counting toward future retries.
func (t *DeepProviderBudgetTracker) TryConsume(jobID uint, provider string, budget int) (allowed bool, callCount int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := budgetKey(jobID, provider)
	current := t.counts[key]
	if budget > 0 && current >= budget {
		return false, current
	}
	current++
	t.counts[key] = current
	return true, current
}

// Reset clears all tracked budget usage for a job. Called once a job
// reaches a terminal state (or is retried into a new job with a fresh
// jobID) so no unbounded memory growth accrues across the process
// lifetime for long-running deployments.
func (t *DeepProviderBudgetTracker) Reset(jobID uint) {
	t.mu.Lock()
	defer t.mu.Unlock()
	prefix := fmt.Sprintf("%d:", jobID)
	for key := range t.counts {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(t.counts, key)
		}
	}
}
