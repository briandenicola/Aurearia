# Skill: Adding a Scheduled Background Job to the Go API

**Category:** Backend Development  
**Applies to:** Ancient Coins Go API  
**Last Updated:** 2026-05-21  

## When to Use This Skill

Use this recipe when you need to add a new recurring background task that runs on a schedule (e.g., daily cleanup, periodic checks, scheduled notifications).

## Prerequisites

- Existing scheduler pattern in the codebase (e.g., `availability_scheduler.go`, `valuation_scheduler.go`)
- Database repository for any data the scheduler needs to query or update
- Service layer for any business logic (e.g., sending notifications)
- Settings service for configuration

## Step-by-Step Recipe

### 1. Create the Scheduler Service

**File:** `src/api/services/{feature}_scheduler.go`

**Pattern:**
```go
package services

import (
	"sync"
	"time"
	"github.com/briandenicola/ancient-coins-api/repository"
)

type FeatureScheduler struct {
	repo        *repository.FeatureRepository
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
}

func NewFeatureScheduler(
	repo *repository.FeatureRepository,
	settingsSvc *SettingsService,
	logger *Logger,
) *FeatureScheduler {
	return &FeatureScheduler{
		repo:        repo,
		settingsSvc: settingsSvc,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

func (s *FeatureScheduler) Start() {
	s.logger.Info("scheduler", "Feature scheduler started")
	
	// Initial startup delay
	select {
	case <-time.After(30 * time.Second):
	case <-s.stopCh:
		return
	}
	
	for {
		wait := s.timeUntilNextRun()
		s.logger.Info("scheduler", "Next feature check in %s", wait)
		
		select {
		case <-time.After(wait):
		case <-s.stopCh:
			s.logger.Info("scheduler", "Feature scheduler stopped")
			return
		}
		
		s.runCycle()
	}
}

func (s *FeatureScheduler) Stop() {
	s.once.Do(func() { close(s.stopCh) })
}

func (s *FeatureScheduler) timeUntilNextRun() time.Duration {
	now := time.Now()
	startHour, startMin := s.getStartTime()
	interval := s.getInterval()
	
	anchor := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMin, 0, 0, now.Location())
	if anchor.After(now) {
		return anchor.Sub(now)
	}
	
	elapsed := now.Sub(anchor)
	periods := int(elapsed/interval) + 1
	next := anchor.Add(time.Duration(periods) * interval)
	return next.Sub(now)
}

func (s *FeatureScheduler) getStartTime() (int, int) {
	raw := s.settingsSvc.GetSetting(SettingFeatureCheckStartTime)
	var h, m int
	if _, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 8, 0 // default
	}
	return h, m
}

func (s *FeatureScheduler) getInterval() time.Duration {
	minStr := s.settingsSvc.GetSetting(SettingFeatureCheckInterval)
	mins, err := strconv.Atoi(minStr)
	if err != nil || mins < 5 {
		mins = 1440 // default: 24 hours
	}
	return time.Duration(mins) * time.Minute
}

func (s *FeatureScheduler) runCycle() {
	enabled := s.settingsSvc.GetSetting(SettingFeatureCheckEnabled)
	if enabled != "true" {
		s.logger.Debug("scheduler", "Feature check disabled, skipping cycle")
		return
	}
	
	s.logger.Info("scheduler", "Starting scheduled feature check cycle")
	
	// Your business logic here
	// Query data, process, send notifications, etc.
	
	s.logger.Info("scheduler", "Feature check cycle complete")
}
```

**Key Points:**
- Use `sync.Once` in `Stop()` to prevent double-close panics
- Always check the `Enabled` setting in `runCycle()`
- Log at Info level for start/stop/cycle, Debug for skipped cycles
- Use `time.After` with `select` to allow clean shutdown

### 2. Add Settings Constants

**File:** `src/api/services/settings_service.go`

**Add three constants:**
```go
const (
	// ... existing constants
	SettingFeatureCheckEnabled  = "FeatureCheckEnabled"
	SettingFeatureCheckInterval = "FeatureCheckInterval"
	SettingFeatureCheckStartTime = "FeatureCheckStartTime"
)
```

**Add defaults to the map:**
```go
var settingDefaults = map[string]string{
	// ... existing defaults
	SettingFeatureCheckEnabled: "false",
	SettingFeatureCheckInterval: "1440", // minutes
	SettingFeatureCheckStartTime: "08:00",
}
```

**Naming Convention:**
- `{Feature}CheckEnabled` — Boolean string (`"true"` / `"false"`)
- `{Feature}CheckInterval` — Integer string (minutes for sub-daily, or use `IntervalDays` suffix for daily+)
- `{Feature}CheckStartTime` — HH:MM format (24-hour)

### 3. Add Repository Methods (if needed)

If your scheduler needs to query data, add methods to the appropriate repository:

```go
// GetItemsRequiringAction returns items that need processing today.
func (r *FeatureRepository) GetItemsRequiringAction() ([]models.Feature, error) {
	var items []models.Feature
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	err := r.db.Where("status = ? AND action_date >= ? AND action_date < ?",
		"pending", startOfDay, endOfDay).
		Order("user_id ASC").
		Find(&items).Error
	return items, err
}
```

**Testing:** Always add a unit test for new repository methods using the in-memory SQLite pattern.

### 4. Wire the Scheduler in main.go

**Location:** After Ollama check, before "Application ready" log.

**Steps:**
1. Ensure all dependencies (repos, services) are already created before this point
2. Construct the scheduler
3. Start it in a goroutine

```go
// Check Ollama connectivity at startup
func() {
	// ... ollama check
}()

// Start wishlist availability scheduler
scheduler := services.NewAvailabilityScheduler(availSvc, coinRepo, settingsSvc, logger)
go scheduler.Start()

// Start YOUR NEW SCHEDULER HERE
featureScheduler := services.NewFeatureScheduler(featureRepo, settingsSvc, logger)
go featureScheduler.Start()

logger.Info("startup", "Application ready")
```

**Note:** If the scheduler needs a repository that's defined inside the `protected` route group, move the repository creation up before the schedulers (see auction ending scheduler example).

### 5. Update README.md

**File:** `src/api/README.md`

Add your scheduler to the "Background Schedulers" section:

```markdown
## Background Schedulers

The API runs [N] background schedulers that start automatically on server startup:

...existing schedulers...

N. **Feature Scheduler** — Brief description. Configured via `FeatureCheckEnabled`, `FeatureCheckStartTime`, and `FeatureCheckInterval` settings. Sends notifications when X happens.
```

### 6. Test & Verify

**Commands:**
```bash
cd src/api
go vet ./...       # Must pass
go test -v ./...   # All tests must pass
```

**Manual Testing:**
1. Start the server
2. Check logs for "Feature scheduler started"
3. Set the enabled flag via Admin Settings
4. Verify cycle runs at the configured time
5. Check notification delivery

## Common Patterns

### Daily Cadence
Default interval: `1440` minutes (24 hours)  
Default start time: `08:00` (8 AM local)

### Idempotency (Daily Jobs)
If your scheduler runs more than once per day but should only act once per day per user:

**In-memory map:**
```go
type FeatureScheduler struct {
	// ...
	lastNotified map[uint]string // userID -> date string (YYYY-MM-DD)
	mu          sync.RWMutex
}

func (s *FeatureScheduler) runCycle() {
	today := time.Now().Format("2006-01-02")
	
	for userID, items := range userItems {
		s.mu.RLock()
		lastDate := s.lastNotified[userID]
		s.mu.RUnlock()
		
		if lastDate == today {
			continue // already notified today
		}
		
		// Do the work
		s.notifyUser(userID, items)
		
		s.mu.Lock()
		s.lastNotified[userID] = today
		s.mu.Unlock()
	}
}
```

**When to use in-memory vs. DB column:**
- In-memory: Simple, daily cadence, acceptable to lose state on restart
- DB column: Need persistent tracking, multiple services, or auditing

### Grouped Notifications
If notifying multiple users, group by user and send one consolidated message per user (not one per item):

```go
// Group by user
userItems := make(map[uint][]models.Feature)
for _, item := range items {
	userItems[item.UserID] = append(userItems[item.UserID], item)
}

// Send one notification per user
for userID, items := range userItems {
	s.notifyUser(userID, items)
}
```

### Pushover Integration
```go
func (s *FeatureScheduler) notifyUser(userID uint, items []models.Feature) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil || !user.PushoverEnabled || user.PushoverUserKey == "" {
		return
	}
	
	title := "Feature Alert"
	message := fmt.Sprintf("%d items require attention:\n\n", len(items))
	for _, item := range items {
		message += fmt.Sprintf("• %s\n", item.Name)
	}
	
	go func() {
		s.pushoverSvc.SendNotification(user.PushoverUserKey, title, message, "")
	}()
}
```

## Architecture Checklist

- [ ] Scheduler uses constructor injection
- [ ] Settings follow naming convention (`{Feature}CheckEnabled`, etc.)
- [ ] Repository owns all GORM queries
- [ ] `sync.Once` used in `Stop()` to prevent double-close
- [ ] Logs at appropriate levels (Info for lifecycle, Debug for skipped cycles)
- [ ] No new HTTP endpoints (uses existing settings API)
- [ ] Tests added for new repository methods
- [ ] `go vet ./...` passes
- [ ] `go test -v ./...` passes
- [ ] README.md updated

## Examples in Codebase

1. **`services/availability_scheduler.go`** — Wishlist availability checking (2-hour interval)
2. **`services/valuation_scheduler.go`** — Collection valuation (7-day interval)
3. **`services/auction_ending_scheduler.go`** — Auction ending alerts (24-hour interval, in-memory idempotency)

## Adding a Manual Trigger + Run Log to an Existing Scheduler

If you already have a scheduler that runs on a schedule, and you want to add:
1. A manual "run now" admin endpoint
2. A run log table that records every run (scheduled or manual)

Follow this pattern (used by Valuation, Availability, and Auction Ending schedulers):

### 1. Create Run Log Model

**File:** `src/api/models/{feature}_run.go`

**Pattern:**
```go
package models

import "time"

type FeatureRun struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TriggerType   string    `gorm:"type:varchar(20);not null" json:"triggerType"`
	TriggerUserID *uint     `json:"triggerUserId"`
	Status        string    `gorm:"type:varchar(20);not null;default:'running'" json:"status"`
	ItemsChecked  int       `json:"itemsChecked"`
	ActionsCount  int       `json:"actionsCount"`
	DurationMs    int64     `json:"durationMs"`
	StartedAt     time.Time `gorm:"not null" json:"startedAt"`
	CompletedAt   *time.Time `json:"completedAt"`
	ErrorMessage  string    `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}
```

**Field naming:**
- `TriggerType` — "scheduled" or "manual"
- `TriggerUserID` — nullable, set only for manual runs
- `Status` — "running", "success", or "error"
- `ItemsChecked`, `ActionsCount` — domain-specific counters (e.g., `LotsChecked`, `AlertsSent`)
- `DurationMs` — run duration in milliseconds

### 2. Add to AutoMigrate

**File:** `src/api/database/database.go`

Add your new model to the AutoMigrate list:
```go
err = DB.AutoMigrate(&models.User{}, ..., &models.FeatureRun{})
```

### 3. Create Run Log Repository

**File:** `src/api/repository/{feature}_repository.go`

**Methods:**
- `CreateRun(run *models.FeatureRun) error`
- `CompleteRun(run *models.FeatureRun) error`
- `ListRuns(page, limit int) ([]models.FeatureRun, int64, error)`
- `GetRunByID(runID uint) (*models.FeatureRun, error)` (optional)
- `PruneOldRuns(keep int)` — prunes to keep N most recent runs

**Example:**
```go
func (r *FeatureRepository) CreateRun(run *models.FeatureRun) error {
	return r.db.Create(run).Error
}

func (r *FeatureRepository) CompleteRun(run *models.FeatureRun) error {
	err := r.db.Model(run).Updates(map[string]interface{}{
		"status":        run.Status,
		"items_checked": run.ItemsChecked,
		"actions_count": run.ActionsCount,
		"duration_ms":   run.DurationMs,
		"completed_at":  run.CompletedAt,
		"error_message": run.ErrorMessage,
	}).Error
	if err == nil {
		r.PruneOldRuns(100) // Keep 100 most recent
	}
	return err
}

func (r *FeatureRepository) ListRuns(page, limit int) ([]models.FeatureRun, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	var total int64
	if err := r.db.Model(&models.FeatureRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var runs []models.FeatureRun
	offset := (page - 1) * limit
	err := r.db.Order("started_at DESC").Offset(offset).Limit(limit).Find(&runs).Error
	return runs, total, err
}
```

### 4. Refactor Scheduler to Log Runs

**File:** `src/api/services/{feature}_scheduler.go`

**Changes:**
1. Add the run repository to the scheduler struct
2. Extract the existing `runCycle()` logic into `runCycleWithTrigger(triggerType, triggerUserID)`
3. Update `runCycle()` to call `runCycleWithTrigger("scheduled", nil)`
4. Add `RunNow(triggerUserID)` method that calls `runCycleWithTrigger("manual", triggerUserID)`
5. Wrap all logic in: create run → execute → finalize run

**Pattern:**
```go
type FeatureScheduler struct {
	repo        *repository.FeatureRepository
	runLogRepo  *repository.FeatureRunRepository
	settingsSvc *SettingsService
	logger      *Logger
	stopCh      chan struct{}
	once        sync.Once
}

func (s *FeatureScheduler) runCycle() {
	enabled := s.settingsSvc.GetSetting(SettingFeatureCheckEnabled)
	if enabled != "true" {
		s.logger.Debug("scheduler", "Feature check disabled, skipping cycle")
		return
	}
	s.runCycleWithTrigger("scheduled", nil)
}

func (s *FeatureScheduler) RunNow(triggerUserID *uint) (*models.FeatureRun, error) {
	return s.runCycleWithTrigger("manual", triggerUserID)
}

func (s *FeatureScheduler) runCycleWithTrigger(triggerType string, triggerUserID *uint) (*models.FeatureRun, error) {
	s.logger.Info("scheduler", "Starting %s feature check cycle", triggerType)
	startedAt := time.Now()

	// Create run record
	run := &models.FeatureRun{
		TriggerType:   triggerType,
		TriggerUserID: triggerUserID,
		Status:        "running",
		StartedAt:     startedAt,
	}
	if err := s.runLogRepo.CreateRun(run); err != nil {
		s.logger.Error("scheduler", "Failed to create run record: %s", err)
		return nil, err
	}

	// Execute business logic, count results
	items, err := s.repo.GetItemsToCheck()
	if err != nil {
		now := time.Now()
		run.Status = "error"
		run.ErrorMessage = fmt.Sprintf("Failed to fetch items: %v", err)
		run.CompletedAt = &now
		run.DurationMs = time.Since(startedAt).Milliseconds()
		s.runLogRepo.CompleteRun(run)
		return run, err
	}

	run.ItemsChecked = len(items)
	// ... process items, count actions ...
	run.ActionsCount = actionsCount

	// Finalize
	now := time.Now()
	run.Status = "success"
	run.CompletedAt = &now
	run.DurationMs = time.Since(startedAt).Milliseconds()
	s.runLogRepo.CompleteRun(run)

	return run, nil
}
```

### 5. Create Admin Handler

**File:** `src/api/handlers/{feature}_admin.go`

**Endpoints:**
1. `ListRuns(c *gin.Context)` — GET with pagination
2. `TriggerRun(c *gin.Context)` — POST that calls `scheduler.RunNow()`

**Pattern:**
```go
package handlers

import (
	"net/http"
	"strconv"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type FeatureAdminHandler struct {
	runRepo   *repository.FeatureRunRepository
	scheduler *services.FeatureScheduler
	logger    *services.Logger
}

func NewFeatureAdminHandler(
	runRepo *repository.FeatureRunRepository,
	scheduler *services.FeatureScheduler,
	logger *services.Logger,
) *FeatureAdminHandler {
	return &FeatureAdminHandler{runRepo: runRepo, scheduler: scheduler, logger: logger}
}

// @Summary List feature runs
// @Tags Admin
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /admin/feature-runs [get]
func (h *FeatureAdminHandler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	runs, total, err := h.runRepo.ListRuns(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list runs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runs":  runs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// @Summary Trigger manual feature check
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /admin/feature/run [post]
func (h *FeatureAdminHandler) TriggerRun(c *gin.Context) {
	triggerUserID := c.GetUint("userId")

	run, err := h.scheduler.RunNow(&triggerUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run check"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runId":        run.ID,
		"itemsChecked": run.ItemsChecked,
		"actionsCount": run.ActionsCount,
		"status":       run.Status,
		"durationMs":   run.DurationMs,
	})
}
```

### 6. Wire in main.go

**Steps:**
1. Create the run repository early (around line 158 where other repos are created)
2. Create the scheduler **before** the admin route group (around line 165)
3. Register admin routes inside the `admin := api.Group("/admin")` block

**Example:**
```go
// Around line 158 — Create repositories
auctionEndingRepo := repository.NewAuctionEndingRepository(database.DB)

// Around line 165 — Create schedulers before routes
auctionEndingScheduler := services.NewAuctionEndingScheduler(auctionLotRepo, auctionEndingRepo, userRepoForVal, pushoverSvc, settingsSvc, logger)

// Inside admin route group (around line 383)
auctionEndingAdminHandler := handlers.NewAuctionEndingAdminHandler(auctionEndingRepo, auctionEndingScheduler, logger)
admin.GET("/auction-ending-runs", auctionEndingAdminHandler.ListRuns)
admin.POST("/auction-ending/run", auctionEndingAdminHandler.TriggerRun)

// Around line 406 — Start schedulers
go auctionEndingScheduler.Start()
```

### 7. Update README.md

Add a note to the Background Schedulers section:
```markdown
3. **Feature Scheduler** — Brief description. Each run is logged in the `feature_runs` table. Runs can be manually triggered via `/admin/feature/run`.

All schedulers honor the enabled flag. Run history is available via admin endpoints:
- `GET /admin/availability-runs`
- `GET /admin/valuation-runs`
- `GET /admin/auction-ending-runs`
```

### Checklist

- [ ] Created run log model (`models/{feature}_run.go`)
- [ ] Added to AutoMigrate in `database/database.go`
- [ ] Created run log repository with CreateRun, CompleteRun, ListRuns, PruneOldRuns
- [ ] Refactored scheduler to wrap cycles in run logging
- [ ] Added `RunNow(triggerUserID)` method to scheduler
- [ ] Created admin handler with ListRuns and TriggerRun endpoints
- [ ] Wired scheduler before admin routes in `main.go`
- [ ] Registered admin routes under `admin` group
- [ ] Updated README.md Background Schedulers section
- [ ] `go vet ./...` passes
- [ ] `go test -v ./...` passes

## Common Pitfalls

1. **Forgetting `sync.Once` in Stop()** — Causes double-close panic on shutdown
2. **Not checking the Enabled setting** — Scheduler runs even when disabled
3. **Defining repo inside protected group** — Move it before schedulers if needed
4. **Hardcoding interval/start time** — Always read from settings
5. **Not handling missing Pushover config** — Check `PushoverEnabled` and key presence
6. **Sending one notification per item** — Batch per user instead

## Related Skills

- Adding a new repository method with tests
- Adding app settings (see `settings_service.go`)
- Integrating Pushover notifications (see `pushover_service.go`)
