package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

// imageVariantPhotoLister lists coin photos in id order, a batch at a time.
type imageVariantPhotoLister interface {
	ListPhotosAfter(afterID uint, limit int) ([]models.CoinImage, error)
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
// appear there. It runs shortly after startup and nightly, one photo at a
// time; a photo that has both variants costs two file checks.
type ImageVariantScheduler struct {
	photos      imageVariantPhotoLister
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
	Checked int
	Made    int
	Failed  int
	// Failures names up to imageVariantFailuresLogged photos and why.
	Failures []string
}

// NewImageVariantScheduler creates the scheduler.
func NewImageVariantScheduler(photos imageVariantPhotoLister, uploadDir string, settingsSvc *SettingsService, logger *Logger) *ImageVariantScheduler {
	return &ImageVariantScheduler{
		photos:      photos,
		uploadDir:   uploadDir,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

// Start runs a pass two minutes after startup, then nightly. Call from a goroutine.
func (s *ImageVariantScheduler) Start() {
	s.logger.Info("scheduler", "Image variant scheduler started")
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

// RunNow runs one pass now.
func (s *ImageVariantScheduler) RunNow() error {
	s.RunCycle()
	return nil
}

// GetStatus returns the scheduler runtime status.
func (s *ImageVariantScheduler) GetStatus() SchedulerStatus {
	s.statusMu.RLock()
	running := s.isRunning
	s.statusMu.RUnlock()
	return SchedulerStatus{Name: "Image Variants", Enabled: true, IsRunning: running, NextRunIn: s.timeUntilNextRun()}
}

func (s *ImageVariantScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	return s.nextRun(now).Sub(now)
}

// nextRun is 03:17 in the schedule time zone: a quiet hour, off the half hour
// the other nightly jobs favour.
func (s *ImageVariantScheduler) nextRun(now time.Time) time.Time {
	return dailySchedule{Hour: 3, Minute: 17, Location: scheduleLocation(s.settingsSvc, s.logger)}.slotAfter(now)
}

// RunCycle checks every coin photo once and makes the variants it is missing.
// Passes never overlap: a RunNow during a scheduled pass waits for it.
func (s *ImageVariantScheduler) RunCycle() ImageVariantCycle {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	s.setRunning(true)
	defer s.setRunning(false)

	var out ImageVariantCycle
	var after uint
	for {
		batch, err := s.photos.ListPhotosAfter(after, imageVariantBatchSize)
		if err != nil {
			s.logger.Error("scheduler", "Image variants: could not list photos: %v", err)
			break
		}
		for _, photo := range batch {
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
	s.logger.Info("scheduler", "Image variants: checked %d photos, made variants for %d, %d could not be resized", out.Checked, out.Made, out.Failed)
	return out
}

func (s *ImageVariantScheduler) setRunning(running bool) {
	s.statusMu.Lock()
	s.isRunning = running
	s.statusMu.Unlock()
}
