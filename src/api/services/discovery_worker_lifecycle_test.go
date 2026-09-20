package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

func awaitWorkerState(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("worker did not reach expected state")
}

func stopDiscoveryTestWorker(t *testing.T, stop func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := stop(ctx); err != nil {
		t.Error(err)
	}
}

type blockingSetBuilderAgent struct {
	started chan struct{}
	calls   atomic.Int32
}

func (a *blockingSetBuilderAgent) RunSetBuilder(ctx context.Context, _ SetBuilderProxyRequest) (*SetBuilderProxyResponse, error) {
	if a.calls.Add(1) == 1 {
		close(a.started)
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestSetBuilderWorkersRecoveryWakeAndCancellation(t *testing.T) {
	svc, db := setupSetBuilderServiceTest(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { stopDiscoveryTestWorker(t, svc.StopWorkers); sqlDB.Close() })
	if err := db.Create(&models.AppSetting{Key: SettingAIProvider, Value: "ollama"}).Error; err != nil {
		t.Fatal(err)
	}
	agent := &blockingSetBuilderAgent{started: make(chan struct{})}
	svc.WithWorkflow(agent, NewSettingsService(repository.NewSettingsRepository(db)), nil, NewLogger(10))
	now, stale := time.Now(), time.Now().Add(-2*time.Hour)
	for _, started := range []*time.Time{&now, &stale, nil} {
		if err := svc.repo.CreateRun(&models.SetBuilderRun{UserID: 1, Prompt: "interrupted", Status: models.SetBuilderRunStatusRunning, StartedAt: started}); err != nil {
			t.Fatal(err)
		}
	}
	queued, err := svc.CreateRun(1, SetBuilderRunRequest{Prompt: "queued"})
	if err != nil {
		t.Fatal(err)
	}
	before := runtime.NumGoroutine()
	for i := 0; i < 10000; i++ {
		svc.enqueueRunID(queued.ID)
	}
	if runtime.NumGoroutine() > before+5 {
		t.Fatal("set builder overflow spawned goroutines")
	}
	<-svc.wake
	svc.StartWorkers(2)
	select {
	case <-agent.started:
	case <-time.After(3 * time.Second):
		t.Fatal("queued run stranded")
	}
	svc.StartWorkers(2)
	var failed int64
	if err := db.Model(&models.SetBuilderRun{}).Where("status = ?", models.SetBuilderRunStatusFailed).Count(&failed).Error; err != nil {
		t.Fatal(err)
	}
	if failed != 3 {
		t.Fatalf("recovered %d interrupted runs", failed)
	}
	stopDiscoveryTestWorker(t, svc.StopWorkers)
	stored, err := svc.repo.GetRunForUser(queued.ID, 1)
	if err != nil || stored.Status != models.SetBuilderRunStatusFailed {
		t.Fatalf("shutdown state: %v %v", stored, err)
	}
	if agent.calls.Load() != 1 {
		t.Fatal("run executed twice")
	}
	if _, err := svc.CreateRun(1, SetBuilderRunRequest{Prompt: "after stop"}); !errors.Is(err, ErrSetBuilderStopped) {
		t.Fatalf("accepted after stop: %v", err)
	}
}

func TestWishlistWorkersRecoveryWakeAndCancellation(t *testing.T) {
	started := make(chan struct{})
	var calls atomic.Int32
	svc, db, closeServer := setupWishlistSearchAlertDiscoveryService(t, func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		if calls.Add(1) == 1 {
			close(started)
		}
		<-r.Context().Done()
	})
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stopDiscoveryTestWorker(t, svc.StopWorkers); closeServer(); sqlDB.Close() })
	alert, err := svc.CreateAlert(1, validAlertInput())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, timestamp := range []time.Time{now, now.Add(-2 * time.Hour)} {
		if err := db.Create(&models.AlertRun{UserID: 1, AlertID: alert.ID, Status: models.AlertRunStatusRunning, StartedAt: timestamp}).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Even old active runs must prevent concurrent submissions before reconciliation.
	if _, err := svc.RunNow(alert.ID, 1, RunAlertInput{}); !errors.Is(err, ErrWishlistSearchAlertRunLimited) {
		t.Fatalf("active run not protected: %v", err)
	}
	svc.StartWorkers(2)
	awaitWorkerState(t, func() bool {
		var count int64
		if err := db.Model(&models.AlertRun{}).Where("status = ?", models.AlertRunStatusFailed).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		return count == 2
	})
	snapshot, err := svc.CriteriaSnapshot(alert, 20)
	if err != nil {
		t.Fatal(err)
	}
	queued := models.AlertRun{UserID: 1, AlertID: alert.ID, Status: models.AlertRunStatusQueued, StartedAt: now, CriteriaSnapshot: snapshot}
	// No wake: the periodic scan must discover this committed job.
	if err := db.Create(&queued).Error; err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("missed wake stranded alert")
	}
	svc.StartWorkers(2)
	before := runtime.NumGoroutine()
	for i := 0; i < 10000; i++ {
		svc.enqueueRunID(queued.ID)
	}
	if runtime.NumGoroutine() > before+5 {
		t.Fatal("wishlist overflow spawned goroutines")
	}
	stopDiscoveryTestWorker(t, svc.StopWorkers)
	stored, err := svc.repo.GetRun(alert.ID, queued.ID, 1)
	if err != nil || stored.Status != models.AlertRunStatusFailed {
		t.Fatalf("shutdown state: %v %v", stored, err)
	}
	if calls.Load() != 1 {
		t.Fatal("discovery executed twice")
	}
	if _, err := svc.RunNow(alert.ID, 1, RunAlertInput{}); !errors.Is(err, ErrWishlistSearchAlertStopped) {
		t.Fatalf("accepted after stop: %v", err)
	}
}

func TestAIJobWorkersRetryReconciliationBeforeClaiming(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { stopDiscoveryTestWorker(t, svc.StopWorkers); sqlDB.Close() })
	coin := createAIJobTestCoin(t, db, 1)
	interrupted := models.AIJob{UserID: 1, CoinID: coin.ID, JobType: models.AIJobTypeValueEstimate, Status: models.AIJobStatusRunning}
	if err := db.Create(&interrupted).Error; err != nil {
		t.Fatal(err)
	}
	queued, _, err := svc.EnqueueValueEstimate(1, coin.ID)
	if err != nil {
		t.Fatal(err)
	}
	var attempts atomic.Int32
	if err := db.Callback().Update().Before("gorm:update").Register("fail_first_recovery", func(tx *gorm.DB) {
		if tx.Statement.Table == "ai_jobs" && attempts.Add(1) == 1 {
			tx.AddError(errors.New("injected recovery failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	svc.StartWorkers(1)
	awaitWorkerState(t, func() bool {
		job, err := svc.GetJob(1, interrupted.ID)
		if err != nil {
			t.Fatal(err)
		}
		return job.Status == models.AIJobStatusFailed
	})
	if attempts.Load() < 2 {
		t.Fatal("recovery error was not retried")
	}
	if queued.ID != interrupted.ID {
		t.Fatal("fixture should deduplicate interrupted job")
	}
}
