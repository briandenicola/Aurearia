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
// appear there. It runs two minutes after startup and then daily at
// ImageVariantCheckStartTime (03:17 by default) in the schedule time zone,
// one photo at a time; a photo that has both variants costs two file checks.
// ImageVariantCheckEnabled turns it off, RunNow included. A pass stops between
// photos when the scheduler stops, and passes never overlap; the next pass
// picks up where an interrupted one left off, since made variants are kept.
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

// RunCycle checks every coin photo once and makes the variants it is missing.
// Passes never overlap: a RunNow during a scheduled pass waits for it.
func (s *ImageVariantScheduler) RunCycle() ImageVariantCycle {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	var out ImageVariantCycle
	if !s.enabled() {
		s.logger.Debug("scheduler", "Image variant check disabled, skipping cycle")
		out.Skipped = true
		return out
	}
	s.setRunning(true)
	defer s.setRunning(false)
	started := time.Now()

	var after uint
walk:
	for {
		batch, err := s.photos.ListPhotosAfter(after, imageVariantBatchSize)
		if err != nil {
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
	s.logger.Info("scheduler", "Image variants: checked %d photos in %s, made variants for %d, %d could not be resized%s",
		out.Checked, time.Since(started).Round(time.Millisecond), out.Made, out.Failed, stopped)
	return out
}

func (s *ImageVariantScheduler) setRunning(running bool) {
	s.statusMu.Lock()
	s.isRunning = running
	s.statusMu.Unlock()
}
