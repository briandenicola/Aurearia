package services

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

func stopAIJobTestWorkers(t *testing.T, svc *AIJobService) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := svc.StopWorkers(ctx); err != nil {
		t.Error(err)
	}
}

func TestAIJobWorkersReconcileFreshInterruptedJob(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { stopAIJobTestWorkers(t, svc); sqlDB.Close() })
	coin := createAIJobTestCoin(t, db, 1)
	job, _, err := svc.repo.EnqueueOrFindActive(1, coin.ID, models.AIJobTypeValueEstimate, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, claimed, err := svc.repo.ClaimQueued(job.ID); err != nil || !claimed {
		t.Fatalf("claim: %t %v", claimed, err)
	}
	svc.StartWorkers(1)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, err := svc.GetJob(1, job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Status == models.AIJobStatusFailed {
			if stored.ErrorMessage == "" {
				t.Fatal("interruption must be explained")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("fresh interrupted job remains Running after restart")
}

type blockingAIJobAgent struct {
	AIJobAgent
	started chan struct{}
	calls   atomic.Int32
}

func (a *blockingAIJobAgent) CollectPortfolioReview(ctx context.Context, _ PortfolioReviewProxyRequest) (string, error) {
	if a.calls.Add(1) == 1 {
		close(a.started)
	}
	<-ctx.Done()
	return "", ctx.Err()
}

func TestAIJobWorkersBoundWakeAndCancelDuplicates(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { stopAIJobTestWorkers(t, svc); sqlDB.Close() })
	coin := createAIJobTestCoin(t, db, 1)
	agent := &blockingAIJobAgent{started: make(chan struct{})}
	svc.agentProxy = agent
	job, _, err := svc.EnqueueValueEstimate(1, coin.ID)
	if err != nil {
		t.Fatal(err)
	}
	before := runtime.NumGoroutine()
	for i := 0; i < 10000; i++ {
		svc.enqueueID(job.ID)
	}
	if runtime.NumGoroutine() > before+5 {
		t.Fatal("overflow submissions spawned goroutines")
	}
	<-svc.wake
	svc.StartWorkers(4)
	select {
	case <-agent.started:
	case <-time.After(2 * time.Second):
		t.Fatal("missed wake stranded queued job")
	}
	svc.StartWorkers(4)
	for i := 0; i < 10; i++ {
		duplicate, created, err := svc.EnqueueValueEstimate(1, coin.ID)
		if err != nil || created || duplicate.ID != job.ID {
			t.Fatalf("duplicate: %v %t %v", duplicate, created, err)
		}
	}
	stopAIJobTestWorkers(t, svc)
	if agent.calls.Load() != 1 {
		t.Fatalf("executed %d times", agent.calls.Load())
	}
	stored, err := svc.GetJob(1, job.ID)
	if err != nil || stored.Status != models.AIJobStatusFailed {
		t.Fatalf("cancelled state: %v %v", stored, err)
	}
	if _, _, err := svc.EnqueueValueEstimate(1, coin.ID); !errors.Is(err, ErrAIJobStopped) {
		t.Fatalf("accepted after stop: %v", err)
	}
}

func TestAIJobWorkersDrainPersistedPagesAndIgnoreTerminalJobs(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { stopAIJobTestWorkers(t, svc); sqlDB.Close() })
	coin := createAIJobTestCoin(t, db, 1)
	now := time.Now().Add(-2 * time.Hour)
	stale := models.AIJob{UserID: 1, CoinID: coin.ID, JobType: models.AIJobTypeValueEstimate, Status: models.AIJobStatusRunning, StartedAt: &now}
	completed := models.AIJob{UserID: 1, CoinID: coin.ID, JobType: models.AIJobTypeValueEstimate, Status: models.AIJobStatusCompleted, Result: "original"}
	for _, job := range []*models.AIJob{&stale, &completed} {
		if err := db.Create(job).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 105; i++ {
		if err := db.Create(&models.AIJob{UserID: 1, CoinID: coin.ID, JobType: models.AIJobTypeValueEstimate, Status: models.AIJobStatusQueued}).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc.StartWorkers(1)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var count int64
		if err := db.Model(&models.AIJob{}).Where("status IN ?", []models.AIJobStatus{models.AIJobStatusQueued, models.AIJobStatusRunning}).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count == 0 {
			failed, err := svc.GetJob(1, stale.ID)
			if err != nil {
				t.Fatal(err)
			}
			done, err := svc.GetJob(1, completed.ID)
			if err != nil {
				t.Fatal(err)
			}
			if failed.Status != models.AIJobStatusFailed || done.Result != "original" || done.Attempts != 0 {
				t.Fatal("replayed interrupted or terminal work")
			}
			var complete int64
			if err := db.Model(&models.AIJob{}).Where("status = ?", models.AIJobStatusCompleted).Count(&complete).Error; err != nil {
				t.Fatal(err)
			}
			if complete != 106 {
				t.Fatalf("completed %d, want 106", complete)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("persisted queue did not drain beyond page 1")
}
