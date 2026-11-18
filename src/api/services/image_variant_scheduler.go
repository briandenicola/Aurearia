package services

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

// imageVariantPhotoLister lists coin photos in id order, a batch at a time.
type imageVariantPhotoLister interface {
	ListPhotosAfter(afterID uint, limit int) ([]models.CoinImage, error)
}

// imageVariantRunStore keeps the run history. The scheduler works without one.
type imageVariantRunStore interface {
	CreateRun(run *models.ImageVariantRun) error
	CompleteRun(run *models.ImageVariantRun) error
	MarkInterrupted() error
	ListRuns(page, limit int) ([]models.ImageVariantRun, int64, error)
}

const (
	imageVariantBatchSize = 200
	// imageVariantFailuresLogged caps how many failed photos a cycle names in
	// the log; the count covers them all.
	imageVariantFailuresLogged = 10
)

// ImageVariantScheduler makes the thumb and medium variants that a coin photo
// is missing: photos uploaded before variants existed, and any a failed resize
// left without them. The MCP coin viewer and other inline transports only
// show originals up to 512 KB, so without variants larger photos never
// appear there. It runs two minutes after startup and then daily at
// ImageVariantCheckStartTime (03:17 by default) in the schedule time zone,
// one photo at a time; a photo that has both variants costs two file checks.
// ImageVariantCheckEnabled turns the schedule off, RunNow included; a pass an
// admin starts with StartManualRun runs either way. A pass stops between
// photos when the scheduler stops, and passes never overlap; the next pass
// picks up where an interrupted one left off, since made variants are kept.
// Each pass that runs is recorded in the run history.
type ImageVariantScheduler struct {
	photos      imageVariantPhotoLister
	runs        imageVariantRunStore
	uploadDir   string
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
	runMu       sync.Mutex
	statusMu    sync.RWMutex
	isRunning   bool
}

// ImageVariantCycle is what one pass over the photos did.
type ImageVariantCycle struct {
	// Skipped is true when the check is turned off; Stopped when the
	// scheduler stopped part way.
	Skipped bool
	Stopped bool
	Checked int
	Made    int
	Failed  int
	// Failures names up to imageVariantFailuresLogged photos and why.
	Failures []string
}

// NewImageVariantScheduler creates the scheduler. runs may be nil, which keeps no history.
func NewImageVariantScheduler(photos imageVariantPhotoLister, runs imageVariantRunStore, uploadDir string, settingsSvc *SettingsService, logger *Logger) *ImageVariantScheduler {
	if runs == nil {
		runs = noImageVariantRuns{}
	}
	return &ImageVariantScheduler{
		photos:      photos,
		runs:        runs,
		uploadDir:   uploadDir,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

// Start runs a pass two minutes after startup, then nightly. Call from a goroutine.
func (s *ImageVariantScheduler) Start() {
	s.logger.Info("scheduler", "Image variant scheduler started")
	if err := s.runs.MarkInterrupted(); err != nil {
		s.logger.Warn("scheduler", "Image variants: could not mark interrupted runs as stopped: %v", err)
	}
	scheduleLoop{
		category:     "scheduler",
		name:         "image variant check",
		logger:       s.logger,
		stopCh:       s.stopCh,
		initialDelay: 2 * time.Minute,
		next: func(now time.Time, catchUp bool) time.Time {
			if catchUp {
				return now
			}
			return s.nextRun(now)
		},
		run: func() { s.RunCycle() },
	}.Run()
}

// Stop signals the scheduler to shut down. Safe to call multiple times.
func (s *ImageVariantScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

// RunNow runs one pass now, unless the check is turned off.
func (s *ImageVariantScheduler) RunNow() error {
	s.RunCycle()
	return nil
}

// StartManualRun starts a pass in the background for an admin, whether or not
// the schedule is turned on. It returns false, starting nothing, when a pass
// is already running.
func (s *ImageVariantScheduler) StartManualRun() bool {
	if !s.runMu.TryLock() {
		return false
	}
	go func() {
		defer s.runMu.Unlock()
		s.walk("manual")
	}()
	return true
}

// ListRuns returns the run history, newest first.
func (s *ImageVariantScheduler) ListRuns(page, limit int) ([]models.ImageVariantRun, int64, error) {
	return s.runs.ListRuns(page, limit)
}

// GetStatus returns the scheduler runtime status.
func (s *ImageVariantScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()
	return SchedulerStatus{Name: "Image Variants", Enabled: s.enabled(), IsRunning: running, NextRunIn: s.timeUntilNextRun()}
}

func (s *ImageVariantScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	return s.nextRun(now).Sub(now)
}

func (s *ImageVariantScheduler) enabled() bool {
	return s.settingsSvc.GetSetting(SettingImageVariantCheckEnabled) != "false"
}

// nextRun is the next daily start time in the schedule time zone.
func (s *ImageVariantScheduler) nextRun(now time.Time) time.Time {
	h, m := parseStartTime(s.settingsSvc.GetSetting(SettingImageVariantCheckStartTime), 3, 17)
	return dailySchedule{Hour: h, Minute: m, Location: scheduleLocation(s.settingsSvc, s.logger)}.slotAfter(now)
}

// RunCycle is the scheduled pass: it checks every coin photo once and makes
// the variants it is missing, unless the check is turned off. Passes never
// overlap: a RunCycle during another pass waits for it.
func (s *ImageVariantScheduler) RunCycle() ImageVariantCycle {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if !s.enabled() {
		s.logger.Debug("scheduler", "Image variant check disabled, skipping cycle")
		return ImageVariantCycle{Skipped: true}
	}
	return s.walk("scheduled")
}

// walk is one pass; the caller holds runMu.
func (s *ImageVariantScheduler) walk(trigger string) ImageVariantCycle {
	var out ImageVariantCycle
	s.setRunning(true)
	defer s.setRunning(false)
	started := time.Now()
	run := &models.ImageVariantRun{TriggerType: trigger, Status: "running", StartedAt: started}
	if err := s.runs.CreateRun(run); err != nil {
		s.logger.Warn("scheduler", "Image variants: could not record the run: %v", err)
	}

	var after uint
	var listErr error
walk:
	for {
		batch, err := s.photos.ListPhotosAfter(after, imageVariantBatchSize)
		if err != nil {
			listErr = err
			s.logger.Error("scheduler", "Image variants: could not list photos: %v", err)
			break
		}
		for _, photo := range batch {
			select {
			case <-s.stopCh:
				out.Stopped = true
				break walk
			default:
			}
			after = photo.ID
			out.Checked++
			made, err := ensureImageVariants(s.uploadDir, photo.FilePath)
			switch {
			case err != nil:
				out.Failed++
				if len(out.Failures) < imageVariantFailuresLogged {
					out.Failures = append(out.Failures, fmt.Sprintf("image %d (coin %d, %s): %v", photo.ID, photo.CoinID, photo.FilePath, err))
				}
			case made:
				out.Made++
			}
		}
		if len(batch) < imageVariantBatchSize {
			break
		}
	}

	for _, f := range out.Failures {
		s.logger.Warn("scheduler", "Image variants: could not resize %s", f)
	}
	stopped := ""
	if out.Stopped {
		stopped = " (stopped part way; the next pass continues)"
	}
	took := time.Since(started)
	s.logger.Info("scheduler", "Image variants: %s pass checked %d photos in %s, made variants for %d, %d could not be resized%s",
		trigger, out.Checked, took.Round(time.Millisecond), out.Made, out.Failed, stopped)

	completed := time.Now()
	run.Checked, run.Made, run.Failed = out.Checked, out.Made, out.Failed
	run.Failures = strings.Join(out.Failures, "\n")
	run.DurationMs = took.Milliseconds()
	run.CompletedAt = &completed
	switch {
	case listErr != nil:
		run.Status = "error"
		run.ErrorMessage = "Could not list coin photos: " + listErr.Error()
	case out.Stopped:
		run.Status = "stopped"
		run.ErrorMessage = "The API stopped during this pass. The next pass carries on."
	default:
		run.Status = "success"
	}
	if run.ID != 0 {
		if err := s.runs.CompleteRun(run); err != nil {
			s.logger.Warn("scheduler", "Image variants: could not record the run's outcome: %v", err)
		}
	}
	return out
}

func (s *ImageVariantScheduler) setRunning(running bool) {
	s.statusMu.Lock()
	s.isRunning = running
	s.statusMu.Unlock()
}

// noImageVariantRuns keeps no history.
type noImageVariantRuns struct{}

func (noImageVariantRuns) CreateRun(*models.ImageVariantRun) error   { return nil }
func (noImageVariantRuns) CompleteRun(*models.ImageVariantRun) error { return nil }
func (noImageVariantRuns) MarkInterrupted() error                    { return nil }
func (noImageVariantRuns) ListRuns(int, int) ([]models.ImageVariantRun, int64, error) {
	return []models.ImageVariantRun{}, 0, nil
}
