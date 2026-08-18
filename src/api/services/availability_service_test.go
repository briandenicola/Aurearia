package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupAvailServiceDB creates an in-memory SQLite DB with required tables.
func setupAvailServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.CoinImage{},
		&models.AvailabilityRun{},
		&models.AvailabilityResult{},
		&models.AppSetting{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// TestCheckURL_200KeywordDetection verifies that the keyword detector works for exact patterns.
func TestCheckURL_200KeywordDetection(t *testing.T) {
	tests := []struct {
		name           string
		htmlBody       string
		expectedStatus string
	}{
		{
			name:           "Exact >sold< pattern",
			htmlBody:       `<div><button>sold</button></div>`,
			expectedStatus: "unavailable",
		},
		{
			name:           "Status: sold text",
			htmlBody:       `<p>Status: Sold</p>`,
			expectedStatus: "unavailable",
		},
		{
			name:           "Add to cart available",
			htmlBody:       `<button id="addToCart">Add to Cart</button>`,
			expectedStatus: "available",
		},
		{
			name:           "Generic purchase text remains ambiguous",
			htmlBody:       `<div>See purchase history for this coin.</div>`,
			expectedStatus: "unknown",
		},
		{
			name:           "Ambiguous page",
			htmlBody:       `<div>Contact us for details</div>`,
			expectedStatus: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.htmlBody))
			}))
			defer server.Close()

			svc := &AvailabilityService{logger: NewLogger(100)}
			result, err := svc.CheckURL(server.URL)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s (reason: %s)", tt.expectedStatus, result.Status, result.Reason)
			}
		})
	}
}

// TestCheckURL_VCoinsSoldBannerBug is the regression test for the specific bug:
// VCoins sold pages have standalone "Sold" text that doesn't always match
// the >sold< keyword pattern and falls through to "unknown" status.
func TestCheckURL_VCoinsSoldBannerBug(t *testing.T) {
	tests := []struct {
		name           string
		htmlBody       string
		expectedStatus string
		description    string
	}{
		{
			name: "VCoins exact HTML structure",
			htmlBody: `<!DOCTYPE html>
<html><head><title>Coin</title></head>
<body>
<div class="item-status">Sold</div>
<div class="follow-buttons">
  <button>Follow Store</button>
  <button>Add to Watch List</button>
</div>
</body></html>`,
			expectedStatus: "unavailable",
			description:    "VCoins page with standalone 'Sold' in styled div",
		},
		{
			name: "Sold with surrounding whitespace",
			htmlBody: `<html><body>
<div class="status">
  Sold
</div>
</body></html>`,
			expectedStatus: "unavailable",
			description:    "Sold text with newlines/whitespace",
		},
		{
			name: "Sold in span",
			htmlBody: `<html><body>
<span class="badge">Sold</span>
</body></html>`,
			expectedStatus: "unavailable",
			description:    "Sold in span element",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.htmlBody))
			}))
			defer server.Close()

			svc := &AvailabilityService{logger: NewLogger(100)}
			result, err := svc.CheckURL(server.URL)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// REGRESSION: These pages should be classified as unavailable, not unknown
			if result.Status != tt.expectedStatus {
				t.Errorf("REGRESSION: %s - expected status %s, got %s (reason: %s)",
					tt.description, tt.expectedStatus, result.Status, result.Reason)
			}
		})
	}
}

// TestCheckURL_404ReturnsUnavailable verifies that 404 responses are immediately classified as unavailable.
func TestCheckURL_404ReturnsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc := &AvailabilityService{logger: NewLogger(100)}
	result, err := svc.CheckURL(server.URL)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("expected status unavailable for 404, got %s", result.Status)
	}
	if result.HttpStatus == nil || *result.HttpStatus != 404 {
		t.Errorf("expected HttpStatus 404, got %v", result.HttpStatus)
	}
}

// TestCheckURL_410ReturnsUnavailable verifies that 410 Gone responses are immediately classified as unavailable.
func TestCheckURL_410ReturnsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer server.Close()

	svc := &AvailabilityService{logger: NewLogger(100)}
	result, err := svc.CheckURL(server.URL)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Status != "unavailable" {
		t.Errorf("expected status unavailable for 410, got %s", result.Status)
	}
	if result.HttpStatus == nil || *result.HttpStatus != 410 {
		t.Errorf("expected HttpStatus 410, got %v", result.HttpStatus)
	}
}

// TestCheckURL_500ReturnsUnknown verifies that server errors are classified as unknown.
func TestCheckURL_500ReturnsUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := &AvailabilityService{logger: NewLogger(100)}
	result, err := svc.CheckURL(server.URL)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Status != "unknown" {
		t.Errorf("expected status unknown for 500, got %s", result.Status)
	}
	if result.HttpStatus == nil || *result.HttpStatus != 500 {
		t.Errorf("expected HttpStatus 500, got %v", result.HttpStatus)
	}
}

// TestCheckWishlistForUser_ClassifiesSoldSignalsWithoutAgent is the regression
// test for the VCoins sold page bug. It verifies scheduled/manual summary counts
// reflect obvious sold and purchase signals even when agent escalation is absent.
func TestCheckWishlistForUser_ClassifiesSoldSignalsWithoutAgent(t *testing.T) {
	db := setupAvailServiceDB(t)

	// Create test user
	user := models.User{Username: "testuser", Email: "test@example.com"}
	db.Create(&user)

	// Mock servers for two VCoins sold pages
	soldServer1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>
			<div class="sold-banner">Sold</div>
			Valentinian I coin
		</body></html>`))
	}))
	defer soldServer1.Close()

	soldServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>
			<div class="status-box green">Sold</div>
			Constantine II coin
		</body></html>`))
	}))
	defer soldServer2.Close()

	// Mock server for an available coin
	availServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Available for purchase</body></html>`))
	}))
	defer availServer.Close()

	// Create three wishlist coins with URLs
	coin1 := models.Coin{
		UserID:       user.ID,
		Name:         "Valentinian I GLORIA ROMANORVM",
		ReferenceURL: soldServer1.URL,
		IsWishlist:   true,
	}
	coin2 := models.Coin{
		UserID:       user.ID,
		Name:         "Constantine II",
		ReferenceURL: soldServer2.URL,
		IsWishlist:   true,
	}
	coin3 := models.Coin{
		UserID:       user.ID,
		Name:         "Julius Caesar Denarius",
		ReferenceURL: availServer.URL,
		IsWishlist:   true,
	}
	db.Create(&coin1)
	db.Create(&coin2)
	db.Create(&coin3)

	// Set up service with repositories
	coinRepo := repository.NewCoinRepository(db)
	availRepo := repository.NewAvailabilityRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)

	svc := &AvailabilityService{
		coinRepo:    coinRepo,
		availRepo:   availRepo,
		agentProxy:  nil,
		settingsSvc: settingsSvc,
		logger:      NewLogger(100),
	}

	// Run the check
	run, err := svc.CheckWishlistForUser(user.ID, "manual", nil, nil)

	if err != nil {
		t.Fatalf("CheckWishlistForUser failed: %v", err)
	}

	// Verify the run summary counts
	if run.CoinsChecked != 3 {
		t.Errorf("expected 3 coins checked, got %d", run.CoinsChecked)
	}

	// REGRESSION: This is the bug — unavailable count should be 2, not 0
	if run.Unavailable != 2 {
		t.Errorf("REGRESSION: expected 2 unavailable (sold) coins, got %d", run.Unavailable)
	}

	if run.Available != 1 {
		t.Errorf("expected 1 available coin, got %d", run.Available)
	}

	// REGRESSION: Unknown should be 0 after page-content checks, not 3.
	if run.Unknown != 0 {
		t.Errorf("REGRESSION: expected 0 unknown after page-content checks, got %d", run.Unknown)
	}

	// Verify individual results in the database
	var results []models.AvailabilityResult
	db.Where("run_id = ?", run.ID).Find(&results)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify both sold coins are marked unavailable
	soldCount := 0
	availCount := 0
	agentUsedCount := 0
	for _, r := range results {
		if r.AgentUsed {
			agentUsedCount++
		}
		if r.Status == "unavailable" {
			soldCount++
			// REGRESSION: Sold coins should be unavailable from page-content signals.
			if r.CoinName == "Valentinian I GLORIA ROMANORVM" || r.CoinName == "Constantine II" {
				if r.HttpStatus == nil || *r.HttpStatus != 200 {
					t.Errorf("coin %s: expected HTTP 200, got %v", r.CoinName, r.HttpStatus)
				}
				if r.Reason == "Requires AI analysis to determine availability" {
					t.Errorf("REGRESSION: coin %s: sold signal was not detected, reason still shows %s", r.CoinName, r.Reason)
				}
			}
		}
		if r.Status == "available" {
			availCount++
		}
	}

	if soldCount != 2 {
		t.Errorf("expected 2 results with status unavailable, got %d", soldCount)
	}
	if availCount != 1 {
		t.Errorf("expected 1 result with status available, got %d", availCount)
	}
	if agentUsedCount != 0 {
		t.Errorf("expected 0 results using agent analysis for clear page signals, got %d", agentUsedCount)
	}
}

// TestCheckWishlistForUser_SummaryCountsWithoutAgent verifies that summary counts
// are correct even when agent is not available or fails.
func TestCheckWishlistForUser_SummaryCountsWithoutAgent(t *testing.T) {
	db := setupAvailServiceDB(t)

	user := models.User{Username: "testuser2", Email: "test2@example.com"}
	db.Create(&user)

	// Mock servers: one 404, one 200 (no agent to resolve)
	notFoundServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundServer.Close()

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	coin1 := models.Coin{
		UserID:       user.ID,
		Name:         "Not Found Coin",
		ReferenceURL: notFoundServer.URL,
		IsWishlist:   true,
	}
	coin2 := models.Coin{
		UserID:       user.ID,
		Name:         "Ambiguous Coin",
		ReferenceURL: okServer.URL,
		IsWishlist:   true,
	}
	db.Create(&coin1)
	db.Create(&coin2)

	coinRepo := repository.NewCoinRepository(db)
	availRepo := repository.NewAvailabilityRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)

	// No agent proxy configured
	svc := &AvailabilityService{
		coinRepo:    coinRepo,
		availRepo:   availRepo,
		agentProxy:  nil,
		settingsSvc: settingsSvc,
		logger:      NewLogger(100),
	}

	run, err := svc.CheckWishlistForUser(user.ID, "manual", nil, nil)

	if err != nil {
		t.Fatalf("CheckWishlistForUser failed: %v", err)
	}

	if run.CoinsChecked != 2 {
		t.Errorf("expected 2 coins checked, got %d", run.CoinsChecked)
	}

	// Without agent, the 404 is unavailable, the 200 stays unknown
	if run.Unavailable != 1 {
		t.Errorf("expected 1 unavailable (404), got %d", run.Unavailable)
	}
	if run.Unknown != 1 {
		t.Errorf("expected 1 unknown (no agent), got %d", run.Unknown)
	}
	if run.Available != 0 {
		t.Errorf("expected 0 available, got %d", run.Available)
	}
}

// TestCheckWishlistForUser_ListingStatusUpdate verifies that coin listing status
// is updated after availability check.
func TestCheckWishlistForUser_ListingStatusUpdate(t *testing.T) {
	db := setupAvailServiceDB(t)

	user := models.User{Username: "testuser3", Email: "test3@example.com"}
	db.Create(&user)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	coin := models.Coin{
		UserID:       user.ID,
		Name:         "Test Coin",
		ReferenceURL: server.URL,
		IsWishlist:   true,
	}
	db.Create(&coin)

	coinRepo := repository.NewCoinRepository(db)
	availRepo := repository.NewAvailabilityRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)

	svc := &AvailabilityService{
		coinRepo:    coinRepo,
		availRepo:   availRepo,
		agentProxy:  nil,
		settingsSvc: settingsSvc,
		logger:      NewLogger(100),
	}

	_, err := svc.CheckWishlistForUser(user.ID, "manual", nil, nil)
	if err != nil {
		t.Fatalf("CheckWishlistForUser failed: %v", err)
	}

	// Reload coin and verify listing status was updated
	var updatedCoin models.Coin
	db.First(&updatedCoin, coin.ID)

	if updatedCoin.ListingStatus != "unavailable" {
		t.Errorf("expected listing status unavailable, got %s", updatedCoin.ListingStatus)
	}
	if updatedCoin.ListingCheckReason == "" {
		t.Error("expected listing check reason to be set")
	}
	if updatedCoin.ListingCheckedAt == nil {
		t.Error("expected listing checked at to be set")
	}
}

// TestCheckWishlistForUser_RateLimiting verifies that rate limiting is applied
// between URL checks (smoke test — checks that delay exists without mocking time).
func TestCheckWishlistForUser_RateLimiting(t *testing.T) {
	db := setupAvailServiceDB(t)

	user := models.User{Username: "testuser4", Email: "test4@example.com"}
	db.Create(&user)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create multiple coins to trigger rate limiting
	for i := 0; i < 3; i++ {
		coin := models.Coin{
			UserID:       user.ID,
			Name:         fmt.Sprintf("Coin %d", i),
			ReferenceURL: server.URL,
			IsWishlist:   true,
		}
		db.Create(&coin)
	}

	coinRepo := repository.NewCoinRepository(db)
	availRepo := repository.NewAvailabilityRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := NewSettingsService(settingsRepo)

	svc := &AvailabilityService{
		coinRepo:    coinRepo,
		availRepo:   availRepo,
		agentProxy:  nil,
		settingsSvc: settingsSvc,
		logger:      NewLogger(100),
	}

	start := time.Now()
	_, err := svc.CheckWishlistForUser(user.ID, "manual", nil, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("CheckWishlistForUser failed: %v", err)
	}

	// With 3 coins and 750ms delay between requests, expect at least 1.5s total
	// (2 delays between 3 requests). Allow some tolerance for test execution.
	minExpected := 1400 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf("expected at least %v for rate limiting, took %v", minExpected, elapsed)
	}
}

// --- Feature 353: cycle-aware service tests (T011, T017, T024-T027, T040, T049) ---

// setupAvailServiceFullDB creates an in-memory DB with the full Feature 353 schema (cycles,
// notifications, pushover settings) — used by tests that need cross-repo aggregation, admin
// fan-out, and notification assertions.
func setupAvailServiceFullDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	// Pushover sends fire on a background goroutine; pin the pool to one connection so it
	// can't land on a second, unmigrated ":memory:" instance.
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.CoinImage{},
		&models.AvailabilityCycle{},
		&models.AvailabilityRun{},
		&models.AvailabilityResult{},
		&models.AppSetting{},
		&models.Notification{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// setupFullAvailabilityService builds an AvailabilityService wired exactly like main.go
// (availRepo + availCycleRepo + notification/pushover services all cross-linked).
func setupFullAvailabilityService(t *testing.T, db *gorm.DB) (*AvailabilityService, *repository.AvailabilityCycleRepository, *repository.AvailabilityRepository) {
	t.Helper()
	coinRepo := repository.NewCoinRepository(db)
	availRepo := repository.NewAvailabilityRepository(db)
	availCycleRepo := repository.NewAvailabilityCycleRepository(db)
	availRepo.WithCycleRepo(availCycleRepo)
	userRepo := repository.NewUserRepository(db)
	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	logger := NewLogger(100)
	pushoverSvc := NewPushoverService(settingsSvc, logger)
	notifSvc := NewNotificationService(repository.NewNotificationRepository(db), nil, userRepo, pushoverSvc, logger)
	svc := NewAvailabilityService(coinRepo, availRepo, nil, notifSvc, pushoverSvc, userRepo, settingsSvc, logger).WithCycleRepo(availCycleRepo)
	return svc, availCycleRepo, availRepo
}

// T011: CheckWishlistForUser always creates exactly one child run whose UserID is the
// invoking owner's ID — never 0 (US1 AC1 / FR-002).
func TestCheckWishlistForUser_CreatesChildRunWithNonZeroOwnerUserID(t *testing.T) {
	db := setupAvailServiceFullDB(t)
	svc, _, _ := setupFullAvailabilityService(t, db)

	user := models.User{Username: "owner-t011", Email: "owner-t011@test.com"}
	db.Create(&user)

	run, err := svc.CheckWishlistForUser(user.ID, models.AvailabilityRunTriggerOwner, &user.ID, nil)
	if err != nil {
		t.Fatalf("CheckWishlistForUser: %v", err)
	}
	if run.UserID == 0 {
		t.Fatal("expected child run UserID > 0, got 0")
	}
	if run.UserID != user.ID {
		t.Fatalf("expected child run UserID %d, got %d", user.ID, run.UserID)
	}

	var count int64
	db.Model(&models.AvailabilityRun{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 child run for owner, got %d", count)
	}
}

// T011 (zero-URL variant): an owner with wishlist coins but none carrying a reference URL
// still gets a single completed child run (never skipped/queued forever) and a notification.
func TestCheckWishlistForUser_ZeroURLOwnerGetsCompletedChildAndNotification(t *testing.T) {
	db := setupAvailServiceFullDB(t)
	svc, _, _ := setupFullAvailabilityService(t, db)

	user := models.User{Username: "zero-url-owner", Email: "zero-url-owner@test.com"}
	db.Create(&user)
	// Wishlist coin with no reference URL — GetWishlistWithURLs excludes it, so this owner has
	// zero checkable URLs but must still receive a completed run + notification.
	db.Create(&models.Coin{UserID: user.ID, Name: "No URL Coin", IsWishlist: true})

	run, err := svc.CheckWishlistForUser(user.ID, models.AvailabilityRunTriggerOwner, &user.ID, nil)
	if err != nil {
		t.Fatalf("CheckWishlistForUser: %v", err)
	}
	if run.Status != models.AvailabilityRunStatusCompleted {
		t.Fatalf("expected completed status for zero-URL owner, got %q", run.Status)
	}
	if run.CoinsChecked != 0 {
		t.Fatalf("expected 0 coins checked, got %d", run.CoinsChecked)
	}

	var notifCount int64
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, NotificationTypeAvailabilityRun).Count(&notifCount)
	if notifCount != 1 {
		t.Fatalf("expected exactly 1 availability-run notification for zero-URL owner, got %d", notifCount)
	}
}

// T017: RunAdminCycle fans out exactly one child run per distinct wishlist owner (never a
// child with UserID 0) and finalizes the parent cycle to "completed" once every child
// succeeds (FR-006/FR-008).
func TestRunAdminCycle_FansOutOneChildPerOwner_AllSucceed(t *testing.T) {
	db := setupAvailServiceFullDB(t)
	svc, availCycleRepo, _ := setupFullAvailabilityService(t, db)

	available := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><button>Add to Cart</button></body></html>`))
	}))
	defer available.Close()

	owner1 := models.User{Username: "admin-cycle-owner1", Email: "aco1@test.com"}
	owner2 := models.User{Username: "admin-cycle-owner2", Email: "aco2@test.com"}
	db.Create(&owner1)
	db.Create(&owner2)
	db.Create(&models.Coin{UserID: owner1.ID, Name: "Coin 1", ReferenceURL: available.URL, IsWishlist: true})
	db.Create(&models.Coin{UserID: owner2.ID, Name: "Coin 2", ReferenceURL: available.URL, IsWishlist: true})

	adminID := uint(999)
	cycle := &models.AvailabilityCycle{
		TriggerType:   models.AvailabilityRunTriggerAdmin,
		TriggerUserID: &adminID,
		Status:        models.AvailabilityCycleStatusRunning,
		StartedAt:     time.Now(),
	}
	if err := db.Create(cycle).Error; err != nil {
		t.Fatalf("seed cycle: %v", err)
	}

	if err := svc.RunAdminCycle(cycle); err != nil {
		t.Fatalf("RunAdminCycle: %v", err)
	}

	var children []models.AvailabilityRun
	db.Where("cycle_id = ?", cycle.ID).Find(&children)
	if len(children) != 2 {
		t.Fatalf("expected 2 child runs (one per owner), got %d", len(children))
	}
	for _, c := range children {
		if c.UserID == 0 {
			t.Fatal("expected every child run to have UserID > 0")
		}
		if c.Status != models.AvailabilityRunStatusCompleted {
			t.Fatalf("expected child completed, got %q", c.Status)
		}
	}

	reloaded, err := availCycleRepo.GetCycleWithChildren(cycle.ID)
	if err != nil {
		t.Fatalf("GetCycleWithChildren: %v", err)
	}
	if reloaded.Status != models.AvailabilityCycleStatusCompleted {
		t.Fatalf("expected parent cycle completed, got %q", reloaded.Status)
	}
}

// T024/T040: matrix + full integration test — an admin cycle fanning out over 3 owners with
// mixed outcomes (one succeeds with no change, one succeeds with a newly-unavailable coin,
// one fails) aggregates the parent to partial_failure, notifies each owner exactly once, and
// sends exactly one admin child-failure notification (D6/FR-006/FR-008/FR-011).
func TestRunAdminCycle_MixedOutcomes_PartialFailureAndNotifications(t *testing.T) {
	db := setupAvailServiceFullDB(t)
	svc, availCycleRepo, availRepo := setupFullAvailabilityService(t, db)

	available := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><button>Add to Cart</button></body></html>`))
	}))
	defer available.Close()
	sold := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><div>Sold</div></body></html>`))
	}))
	defer sold.Close()

	admin := models.User{Username: "admin-t040", Email: "admin-t040@test.com", Role: models.RoleAdmin}
	ownerNoChange := models.User{Username: "owner-nochange", Email: "onc@test.com"}
	ownerChanged := models.User{Username: "owner-changed", Email: "ochg@test.com"}
	ownerFailing := models.User{Username: "owner-failing", Email: "ofail@test.com"}
	db.Create(&admin)
	db.Create(&ownerNoChange)
	db.Create(&ownerChanged)
	db.Create(&ownerFailing)
	db.Create(&models.Coin{UserID: ownerNoChange.ID, Name: "Steady Coin", ReferenceURL: available.URL, IsWishlist: true})
	db.Create(&models.Coin{UserID: ownerChanged.ID, Name: "Newly Sold Coin", ReferenceURL: sold.URL, IsWishlist: true})
	// ownerFailing has no wishlist coin at all — its "failure" is injected directly below to
	// deterministically exercise the FailChildRun -> aggregate -> admin-notify path without
	// relying on a flaky real I/O failure.

	cycle := &models.AvailabilityCycle{
		TriggerType:   models.AvailabilityRunTriggerAdmin,
		TriggerUserID: &admin.ID,
		Status:        models.AvailabilityCycleStatusRunning,
		StartedAt:     time.Now(),
	}
	if err := db.Create(cycle).Error; err != nil {
		t.Fatalf("seed cycle: %v", err)
	}

	// Two owners succeed through the real CheckWishlistForUser path.
	if _, err := svc.CheckWishlistForUser(ownerNoChange.ID, cycle.TriggerType, cycle.TriggerUserID, &cycle.ID); err != nil {
		t.Fatalf("CheckWishlistForUser (no change): %v", err)
	}
	if _, err := svc.CheckWishlistForUser(ownerChanged.ID, cycle.TriggerType, cycle.TriggerUserID, &cycle.ID); err != nil {
		t.Fatalf("CheckWishlistForUser (changed): %v", err)
	}

	// The third owner's check fails — reproduce exactly the production failure path used by
	// CheckWishlistForUser's own GetWishlistWithURLs-error branch (create running child, fail
	// it with the shared generic message, then fire the same terminal-notification hook).
	failingRun := &models.AvailabilityRun{
		UserID:        ownerFailing.ID,
		CycleID:       &cycle.ID,
		TriggerType:   cycle.TriggerType,
		TriggerUserID: cycle.TriggerUserID,
		Status:        models.AvailabilityRunStatusRunning,
		StartedAt:     time.Now(),
	}
	if err := availRepo.CreateChildRun(failingRun); err != nil {
		t.Fatalf("create failing child run: %v", err)
	}
	if _, err := availRepo.FailChildRun(failingRun, models.GenericAvailabilityFailureMessage); err != nil {
		t.Fatalf("fail child run: %v", err)
	}
	svc.notifyRunTerminal(failingRun, nil)

	reloadedCycle, err := availCycleRepo.GetCycleWithChildren(cycle.ID)
	if err != nil {
		t.Fatalf("GetCycleWithChildren: %v", err)
	}
	if reloadedCycle.Status != models.AvailabilityCycleStatusPartialFailure {
		t.Fatalf("expected parent partial_failure with 2 completed + 1 failed child, got %q", reloadedCycle.Status)
	}
	if reloadedCycle.CompletedChildren != 2 || reloadedCycle.FailedChildren != 1 {
		t.Fatalf("expected completed=2 failed=1, got completed=%d failed=%d",
			reloadedCycle.CompletedChildren, reloadedCycle.FailedChildren)
	}

	// Exactly one owner notification per owner (T024 matrix: one per outcome, no dupes).
	for _, owner := range []models.User{ownerNoChange, ownerChanged, ownerFailing} {
		var count int64
		db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", owner.ID, NotificationTypeAvailabilityRun).Count(&count)
		if count != 1 {
			t.Errorf("expected exactly 1 availability-run notification for owner %d, got %d", owner.ID, count)
		}
	}

	// Exactly one admin notification about the failing child (T026).
	var adminNotifCount int64
	db.Model(&models.Notification{}).Where("user_id = ?", admin.ID).Count(&adminNotifCount)
	if adminNotifCount != 1 {
		t.Fatalf("expected exactly 1 admin notification for the failed child, got %d", adminNotifCount)
	}
}

// T025: a Pushover send failure (deterministic — no app token configured) must never change
// the child run's terminal status; the DB status transition and the Pushover attempt are
// fully decoupled (FR-011).
func TestCheckWishlistForUser_PushoverFailureDoesNotAffectRunStatus(t *testing.T) {
	db := setupAvailServiceFullDB(t)
	svc, _, _ := setupFullAvailabilityService(t, db)
	// Pushover app token is deliberately left unset — every send will fail deterministically
	// with ErrPushoverNotConfigured without any network call.

	user := models.User{Username: "pushover-fail-owner", Email: "pfo@test.com"}
	db.Create(&user)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><button>Add to Cart</button></body></html>`))
	}))
	defer server.Close()
	db.Create(&models.Coin{UserID: user.ID, Name: "Coin", ReferenceURL: server.URL, IsWishlist: true})

	run, err := svc.CheckWishlistForUser(user.ID, models.AvailabilityRunTriggerOwner, &user.ID, nil)
	if err != nil {
		t.Fatalf("CheckWishlistForUser: %v", err)
	}
	if run.Status != models.AvailabilityRunStatusCompleted {
		t.Fatalf("expected completed status despite Pushover being unconfigured, got %q", run.Status)
	}

	// The in-app notification must still have been persisted even though Pushover failed.
	var notifCount int64
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, NotificationTypeAvailabilityRun).Count(&notifCount)
	if notifCount != 1 {
		t.Fatalf("expected the in-app notification to persist despite Pushover failure, got %d", notifCount)
	}
}

// T027: the existing per-coin "wishlist_unavailable" notification and the new per-run
// "wishlist_availability_run" notification both fire for the same run — neither suppresses
// the other (D6/FR-014 regression guard against silently dropping the legacy per-coin alert).
func TestCheckWishlistForUser_PerCoinAndPerRunNotificationsCoexist(t *testing.T) {
	db := setupAvailServiceFullDB(t)
	svc, _, _ := setupFullAvailabilityService(t, db)

	user := models.User{Username: "coexist-owner", Email: "coexist@test.com"}
	db.Create(&user)

	sold := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><div>Sold</div></body></html>`))
	}))
	defer sold.Close()
	db.Create(&models.Coin{UserID: user.ID, Name: "Soon Sold Coin", ReferenceURL: sold.URL, IsWishlist: true, ListingStatus: "available"})

	if _, err := svc.CheckWishlistForUser(user.ID, models.AvailabilityRunTriggerOwner, &user.ID, nil); err != nil {
		t.Fatalf("CheckWishlistForUser: %v", err)
	}

	var perCoinCount, perRunCount int64
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, "wishlist_unavailable").Count(&perCoinCount)
	db.Model(&models.Notification{}).Where("user_id = ? AND type = ?", user.ID, NotificationTypeAvailabilityRun).Count(&perRunCount)
	if perCoinCount != 1 {
		t.Errorf("expected 1 per-coin wishlist_unavailable notification, got %d", perCoinCount)
	}
	if perRunCount != 1 {
		t.Errorf("expected 1 per-run wishlist_availability_run notification, got %d", perRunCount)
	}
}

// T049: sweeps every notification and every run/cycle FailMessage generated across a mix of
// success/failure/zero-URL/admin-failure scenarios for leaked implementation detail — no
// http(s):// URL, and no raw error/panic-style text, may ever surface in user-facing content.
func TestAvailabilityNotifications_NeverLeakURLsOrInternalDetails(t *testing.T) {
	db := setupAvailServiceFullDB(t)
	svc, _, availRepo := setupFullAvailabilityService(t, db)

	sensitiveURL := "https://dealer.example.test/secret-listing-path?token=abc123"

	if strings.Contains(models.GenericAvailabilityFailureMessage, "http://") || strings.Contains(models.GenericAvailabilityFailureMessage, "https://") {
		t.Fatalf("GenericAvailabilityFailureMessage itself must never contain a URL: %q", models.GenericAvailabilityFailureMessage)
	}

	owner := models.User{Username: "leak-check-owner", Email: "leak@test.com"}
	db.Create(&owner)

	sold := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><div>Sold</div></body></html>`))
	}))
	defer sold.Close()
	db.Create(&models.Coin{UserID: owner.ID, Name: "Leak Check Coin", ReferenceURL: sold.URL, IsWishlist: true})

	if _, err := svc.CheckWishlistForUser(owner.ID, models.AvailabilityRunTriggerOwner, &owner.ID, nil); err != nil {
		t.Fatalf("CheckWishlistForUser (success path): %v", err)
	}

	// Directly exercise the failure path (mirrors CheckWishlistForUser's own error branch,
	// which always passes the shared generic message — never a raw error/URL string) to make
	// sure no failure-path notification or stored field ever leaks a URL.
	admin := models.User{Username: "leak-check-admin", Email: "leak-admin@test.com", Role: models.RoleAdmin}
	db.Create(&admin)
	failingRun := &models.AvailabilityRun{
		UserID:        owner.ID,
		TriggerType:   models.AvailabilityRunTriggerAdmin,
		TriggerUserID: &admin.ID,
		Status:        models.AvailabilityRunStatusRunning,
		StartedAt:     time.Now(),
	}
	if err := availRepo.CreateChildRun(failingRun); err != nil {
		t.Fatalf("create failing run: %v", err)
	}
	if _, err := availRepo.FailChildRun(failingRun, models.GenericAvailabilityFailureMessage); err != nil {
		t.Fatalf("fail child run: %v", err)
	}
	svc.notifyRunTerminal(failingRun, nil)
	_ = sensitiveURL // documents the kind of value that must never leak; asserted via the sold-URL test server above

	var notifications []models.Notification
	if err := db.Find(&notifications).Error; err != nil {
		t.Fatalf("load notifications: %v", err)
	}
	if len(notifications) == 0 {
		t.Fatal("expected at least one notification to inspect")
	}
	for _, n := range notifications {
		for _, field := range []string{n.Title, n.Message} {
			if strings.Contains(field, "http://") || strings.Contains(field, "https://") {
				t.Errorf("notification %d leaked a URL: %q", n.ID, field)
			}
			if strings.Contains(strings.ToLower(field), "panic") || strings.Contains(strings.ToLower(field), "sql") {
				t.Errorf("notification %d leaked internal detail: %q", n.ID, field)
			}
		}
	}

	var runsAndCycles []models.AvailabilityRun
	db.Find(&runsAndCycles)
	for _, r := range runsAndCycles {
		if strings.Contains(r.FailMessage, "http://") || strings.Contains(r.FailMessage, "https://") {
			t.Errorf("run %d FailMessage leaked a URL: %q", r.ID, r.FailMessage)
		}
	}
	var cycles []models.AvailabilityCycle
	db.Find(&cycles)
	for _, c := range cycles {
		if strings.Contains(c.FailMessage, "http://") || strings.Contains(c.FailMessage, "https://") {
			t.Errorf("cycle %d FailMessage leaked a URL: %q", c.ID, c.FailMessage)
		}
	}
}
