package services

import (
	"sync"
	"testing"
)

func TestDeepProviderBudgetTracker_EnforcesPerJobPerProviderBudget(t *testing.T) {
	tracker := NewDeepProviderBudgetTracker()

	for i := 1; i <= 3; i++ {
		allowed, count := tracker.TryConsume(1, "numista", 3)
		if !allowed {
			t.Fatalf("call %d expected allowed", i)
		}
		if count != i {
			t.Fatalf("call %d expected count=%d, got %d", i, i, count)
		}
	}

	allowed, count := tracker.TryConsume(1, "numista", 3)
	if allowed {
		t.Fatal("4th call expected to be rejected as over budget")
	}
	if count != 3 {
		t.Fatalf("expected rejected call to report count=3 (unchanged), got %d", count)
	}
}

func TestDeepProviderBudgetTracker_IsolatesJobsAndProviders(t *testing.T) {
	tracker := NewDeepProviderBudgetTracker()

	// Exhaust job 1's numista budget.
	for i := 0; i < 3; i++ {
		if allowed, _ := tracker.TryConsume(1, "numista", 3); !allowed {
			t.Fatalf("unexpected rejection at call %d", i)
		}
	}
	if allowed, _ := tracker.TryConsume(1, "numista", 3); allowed {
		t.Fatal("job 1 numista should be exhausted")
	}

	// A different job (2) with the same provider must have its own budget.
	if allowed, count := tracker.TryConsume(2, "numista", 3); !allowed || count != 1 {
		t.Fatalf("expected job 2 to have a fresh budget, got allowed=%v count=%d", allowed, count)
	}

	// A different provider on the same exhausted job must also have its
	// own independent budget.
	if allowed, count := tracker.TryConsume(1, "nomisma", 3); !allowed || count != 1 {
		t.Fatalf("expected job 1's nomisma budget to be independent, got allowed=%v count=%d", allowed, count)
	}
}

func TestDeepProviderBudgetTracker_ResetClearsOnlyThatJob(t *testing.T) {
	tracker := NewDeepProviderBudgetTracker()
	tracker.TryConsume(1, "numista", 1)
	tracker.TryConsume(2, "numista", 1)

	tracker.Reset(1)

	if allowed, count := tracker.TryConsume(1, "numista", 1); !allowed || count != 1 {
		t.Fatalf("expected job 1 to be reset, got allowed=%v count=%d", allowed, count)
	}
	if allowed, _ := tracker.TryConsume(2, "numista", 1); allowed {
		t.Fatal("job 2's budget should be unaffected by resetting job 1")
	}
}

func TestDeepProviderBudgetTracker_ConcurrentUseIsRace_Free(t *testing.T) {
	tracker := NewDeepProviderBudgetTracker()
	var wg sync.WaitGroup
	var allowedCount int32
	var mu sync.Mutex
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if allowed, _ := tracker.TryConsume(1, "numista", 10); allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowedCount != 10 {
		t.Fatalf("expected exactly 10 of 50 concurrent calls to be allowed (budget=10), got %d", allowedCount)
	}
}
