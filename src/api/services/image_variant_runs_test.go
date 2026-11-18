package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

func newImageVariantRunRepo(t *testing.T) *repository.ImageVariantRunRepository {
	t.Helper()
	db := newImageVariantDB(t)
	if err := db.AutoMigrate(&models.ImageVariantRun{}); err != nil {
		t.Fatal(err)
	}
	return repository.NewImageVariantRunRepository(db)
}

func onlyRun(t *testing.T, runs *repository.ImageVariantRunRepository) models.ImageVariantRun {
	t.Helper()
	list, total, err := runs.ListRuns(1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("runs=%+v total=%d, want one", list, total)
	}
	return list[0]
}

// waitForManualRun waits for a background pass to finish and release the lock.
func waitForManualRun(t *testing.T, s *ImageVariantScheduler) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if s.runMu.TryLock() {
			s.runMu.Unlock()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the manual pass did not finish")
}

func TestImageVariantScheduledPassIsRecorded(t *testing.T) {
	uploadDir := t.TempDir()
	writeUpload(t, uploadDir, "coin-1/1.png", makeLargeTestPNG(t))
	writeUpload(t, uploadDir, "coin-1/2.png", []byte("corrupt"))
	lister := &fakePhotoLister{photos: []models.CoinImage{
		{ID: 1, CoinID: 1, FilePath: "coin-1/1.png"},
		{ID: 2, CoinID: 1, FilePath: "coin-1/2.png"},
	}}
	runs := newImageVariantRunRepo(t)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(newImageVariantDB(t)))
	scheduler := NewImageVariantScheduler(lister, runs, uploadDir, settingsSvc, NewLogger(100))

	scheduler.RunCycle()
	run := onlyRun(t, runs)
	if run.TriggerType != "scheduled" || run.Status != "success" || run.Checked != 2 || run.Made != 1 || run.Failed != 1 ||
		run.CompletedAt == nil || !strings.Contains(run.Failures, "coin-1/2.png") || run.ErrorMessage != "" {
		t.Fatalf("run=%+v, want a successful scheduled pass naming the corrupt photo", run)
	}
}

func TestImageVariantPassThatCannotListPhotosIsRecordedAsAnError(t *testing.T) {
	runs := newImageVariantRunRepo(t)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(newImageVariantDB(t)))
	scheduler := NewImageVariantScheduler(failingPhotoLister{}, runs, t.TempDir(), settingsSvc, NewLogger(100))
	scheduler.RunCycle()
	run := onlyRun(t, runs)
	if run.Status != "error" || !strings.Contains(run.ErrorMessage, "database is locked") {
		t.Fatalf("run=%+v, want an error naming the cause", run)
	}
}

func TestImageVariantTurnedOffSchedulesNoRunButAManualRunGoesAhead(t *testing.T) {
	uploadDir := t.TempDir()
	writeUpload(t, uploadDir, "coin-1/1.png", makeLargeTestPNG(t))
	lister := &fakePhotoLister{photos: []models.CoinImage{{ID: 1, CoinID: 1, FilePath: "coin-1/1.png"}}}
	runs := newImageVariantRunRepo(t)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(newImageVariantDB(t)))
	if err := settingsSvc.SetSetting(SettingImageVariantCheckEnabled, "false"); err != nil {
		t.Fatal(err)
	}
	scheduler := NewImageVariantScheduler(lister, runs, uploadDir, settingsSvc, NewLogger(100))

	scheduler.RunCycle()
	if list, total, _ := runs.ListRuns(1, 20); total != 0 {
		t.Fatalf("a turned-off scheduled pass was recorded: %+v", list)
	}

	if !scheduler.StartManualRun() {
		t.Fatal("the manual run did not start")
	}
	waitForManualRun(t, scheduler)
	run := onlyRun(t, runs)
	if run.TriggerType != "manual" || run.Status != "success" || run.Made != 1 {
		t.Fatalf("run=%+v, want a manual pass that made the variants", run)
	}
}

func TestImageVariantManualRunIsRefusedWhileAPassRuns(t *testing.T) {
	lister := &blockingPhotoLister{entered: make(chan struct{}), release: make(chan struct{})}
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(newImageVariantDB(t)))
	scheduler := NewImageVariantScheduler(lister, nil, t.TempDir(), settingsSvc, NewLogger(100))

	done := make(chan struct{})
	go func() {
		scheduler.RunCycle()
		close(done)
	}()
	<-lister.entered
	if !scheduler.GetStatus().IsRunning {
		t.Fatal("status does not say running during a pass")
	}
	if scheduler.StartManualRun() {
		t.Fatal("a manual run started while a pass was running")
	}
	close(lister.release)
	<-done
	if !scheduler.StartManualRun() {
		t.Fatal("a manual run was refused after the pass finished")
	}
	waitForManualRun(t, scheduler)
}

func TestImageVariantRunsLeftRunningAreMarkedStoppedOnStart(t *testing.T) {
	runs := newImageVariantRunRepo(t)
	if err := runs.CreateRun(&models.ImageVariantRun{TriggerType: "scheduled", Status: "running", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(newImageVariantDB(t)))
	scheduler := NewImageVariantScheduler(&fakePhotoLister{}, runs, t.TempDir(), settingsSvc, NewLogger(100))
	scheduler.Stop()
	scheduler.Start()
	run := onlyRun(t, runs)
	if run.Status != "stopped" || run.ErrorMessage == "" {
		t.Fatalf("run=%+v, want it marked stopped with a reason", run)
	}
}

func TestImageVariantRunHistoryKeepsTheNewestHundred(t *testing.T) {
	runs := newImageVariantRunRepo(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 105; i++ {
		run := &models.ImageVariantRun{TriggerType: "scheduled", Status: "running", StartedAt: start.Add(time.Duration(i) * time.Hour)}
		if err := runs.CreateRun(run); err != nil {
			t.Fatal(err)
		}
		run.Status = "success"
		run.Checked = i
		if err := runs.CompleteRun(run); err != nil {
			t.Fatal(err)
		}
	}
	list, total, err := runs.ListRuns(1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if total != 100 || len(list) != 5 || list[0].Checked != 104 {
		t.Fatalf("total=%d first=%+v, want the newest 100 kept, newest first", total, list[0])
	}
}

type failingPhotoLister struct{}

func (failingPhotoLister) ListPhotosAfter(uint, int) ([]models.CoinImage, error) {
	return nil, errors.New("database is locked")
}

// blockingPhotoLister holds a pass in its first listing until released.
type blockingPhotoLister struct {
	entered chan struct{}
	release chan struct{}
	once    bool
}

func (b *blockingPhotoLister) ListPhotosAfter(uint, int) ([]models.CoinImage, error) {
	if !b.once {
		b.once = true
		close(b.entered)
		<-b.release
	}
	return nil, nil
}
