package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAvailabilityHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
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

// availHandlerFixture bundles everything needed to exercise the availability handler routes
// (owner, admin run/cycle) against a fully wired cycle-aware stack (Feature 353).
type availHandlerFixture struct {
	router         *gin.Engine
	db             *gorm.DB
	scheduler      *services.AvailabilityScheduler
	availRepo      *repository.AvailabilityRepository
	availCycleRepo *repository.AvailabilityCycleRepository
	adminID        uint
	userID         uint
	otherUserID    uint
}

func setupAvailabilityRouter(t *testing.T, listingURL string) *availHandlerFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupAvailabilityHandlerTestDB(t)
	adminUser := models.User{Username: "admin", Email: "admin@test.com", Role: models.RoleAdmin}
	regularUser := models.User{Username: "user", Email: "user@test.com", Role: models.RoleUser}
	otherUser := models.User{Username: "other", Email: "other@test.com", Role: models.RoleUser}
	db.Create(&adminUser)
	db.Create(&regularUser)
	db.Create(&otherUser)
	db.Create(&models.Coin{
		UserID:       regularUser.ID,
		Name:         "Wishlist Coin",
		ReferenceURL: listingURL,
		IsWishlist:   true,
	})

	coinRepo := repository.NewCoinRepository(db)
	availRepo := repository.NewAvailabilityRepository(db)
	availCycleRepo := repository.NewAvailabilityCycleRepository(db)
	availRepo.WithCycleRepo(availCycleRepo)
	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	logger := services.NewLogger(100)
	availSvc := services.NewAvailabilityService(coinRepo, availRepo, nil, nil, nil, nil, settingsSvc, logger).WithCycleRepo(availCycleRepo)
	scheduler := services.NewAvailabilityScheduler(availSvc, coinRepo, availRepo, settingsSvc, logger).WithCycleRepo(availCycleRepo)
	handler := NewAvailabilityHandler(availSvc, scheduler, availRepo, coinRepo).WithCycleRepo(availCycleRepo)

	router := gin.New()
	auth := func(c *gin.Context) {
		switch c.GetHeader("Authorization") {
		case "admin-token":
			c.Set("userId", adminUser.ID)
			c.Set("userRole", string(models.RoleAdmin))
		case "user-token":
			c.Set("userId", regularUser.ID)
			c.Set("userRole", string(models.RoleUser))
		case "other-token":
			c.Set("userId", otherUser.ID)
			c.Set("userRole", string(models.RoleUser))
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}

	protected := router.Group("/api")
	protected.Use(auth)
	protected.POST("/wishlist/check-availability", handler.CheckAvailability)
	protected.GET("/wishlist/availability-runs", handler.ListOwnerRuns)
	protected.GET("/wishlist/availability-runs/:id", handler.GetOwnerRunDetail)

	admin := router.Group("/api/admin")
	admin.Use(auth)
	admin.Use(AdminRequired())
	admin.POST("/availability/run", handler.TriggerRun)
	admin.GET("/availability-cycles", handler.ListCycles)
	admin.GET("/availability-cycles/:id", handler.GetCycleDetail)

	return &availHandlerFixture{
		router:         router,
		db:             db,
		scheduler:      scheduler,
		availRepo:      availRepo,
		availCycleRepo: availCycleRepo,
		adminID:        adminUser.ID,
		userID:         regularUser.ID,
		otherUserID:    otherUser.ID,
	}
}

// T018: TriggerRun returns 202 with {cycleId, status} per FR-026/AC1.
func TestAvailabilityHandler_TriggerRun_AsAdmin(t *testing.T) {
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><div>Sold</div></body></html>`))
	}))
	defer listing.Close()

	f := setupAvailabilityRouter(t, listing.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req.Header.Set("Authorization", "admin-token")
	w := httptest.NewRecorder()

	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	cycleIDFloat, ok := resp["cycleId"].(float64)
	if !ok || cycleIDFloat <= 0 {
		t.Fatalf("expected cycleId in response, got %+v", resp)
	}
	if resp["status"] != models.AvailabilityCycleStatusQueued {
		t.Fatalf("expected status=queued, got %q", resp["status"])
	}

	cycleID := uint(cycleIDFloat)

	// Verify parent cycle is in DB as queued before worker processes it
	var queuedCycle models.AvailabilityCycle
	if err := f.db.First(&queuedCycle, cycleID).Error; err != nil {
		t.Fatalf("expected availability cycle to be created: %v", err)
	}
	if queuedCycle.TriggerType != models.AvailabilityRunTriggerAdmin {
		t.Fatalf("expected admin trigger, got %q", queuedCycle.TriggerType)
	}
	if queuedCycle.TriggerUserID == nil || *queuedCycle.TriggerUserID != f.adminID {
		t.Fatalf("expected trigger user ID %d, got %v", f.adminID, queuedCycle.TriggerUserID)
	}
	if queuedCycle.Status != models.AvailabilityCycleStatusQueued {
		t.Fatalf("expected queued status before processing, got %q", queuedCycle.Status)
	}

	// Process the cycle synchronously (simulates what the worker does)
	if err := f.scheduler.ProcessCycle(cycleID); err != nil {
		t.Fatalf("process cycle: %v", err)
	}

	// Verify the fanned-out child run for the wishlist owner completed with correct counts
	var childRuns []models.AvailabilityRun
	f.db.Where("cycle_id = ?", cycleID).Find(&childRuns)
	if len(childRuns) != 1 {
		t.Fatalf("expected 1 child run, got %d", len(childRuns))
	}
	child := childRuns[0]
	if child.Status != models.AvailabilityRunStatusCompleted {
		t.Fatalf("expected status=completed, got %q", child.Status)
	}
	if child.Unavailable != 1 || child.Unknown != 0 {
		t.Fatalf("expected sold listing to count unavailable=1 unknown=0, got unavailable=%d unknown=%d",
			child.Unavailable, child.Unknown)
	}
}

// T016/T018: a second TriggerRun while the first cycle is queued/running is rejected with 409.
func TestAvailabilityHandler_TriggerRun_DuplicateBlocked(t *testing.T) {
	f := setupAvailabilityRouter(t, "https://example.test/coin")

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req.Header.Set("Authorization", "admin-token")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("first request: expected 202, got %d: %s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req2.Header.Set("Authorization", "admin-token")
	w2 := httptest.NewRecorder()
	f.router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate request: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestAvailabilityHandler_TriggerRun_AsRegularUser(t *testing.T) {
	f := setupAvailabilityRouter(t, "https://example.test/coin")

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req.Header.Set("Authorization", "user-token")
	w := httptest.NewRecorder()

	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAvailabilityHandler_TriggerRun_NoAuth(t *testing.T) {
	f := setupAvailabilityRouter(t, "https://example.test/coin")

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	w := httptest.NewRecorder()

	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// T012: GET /wishlist/availability-runs is owner-scoped — a caller only ever sees their own
// runs, never another user's, even when both have runs in the same table.
func TestAvailabilityHandler_ListOwnerRuns_OwnerScoped(t *testing.T) {
	f := setupAvailabilityRouter(t, "https://example.test/coin")

	triggerID := f.userID
	if err := f.availRepo.CreateChildRun(&models.AvailabilityRun{
		UserID:        f.userID,
		TriggerType:   models.AvailabilityRunTriggerOwner,
		TriggerUserID: &triggerID,
		Status:        models.AvailabilityRunStatusCompleted,
	}); err != nil {
		t.Fatalf("seed owner run: %v", err)
	}
	otherTriggerID := f.otherUserID
	if err := f.availRepo.CreateChildRun(&models.AvailabilityRun{
		UserID:        f.otherUserID,
		TriggerType:   models.AvailabilityRunTriggerOwner,
		TriggerUserID: &otherTriggerID,
		Status:        models.AvailabilityRunStatusCompleted,
	}); err != nil {
		t.Fatalf("seed other user run: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/wishlist/availability-runs", nil)
	req.Header.Set("Authorization", "user-token")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Runs  []models.AvailabilityRun `json:"runs"`
		Total int64                    `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp.Total != 1 || len(resp.Runs) != 1 {
		t.Fatalf("expected exactly 1 run for the caller, got total=%d len=%d", resp.Total, len(resp.Runs))
	}
	if resp.Runs[0].UserID != f.userID {
		t.Fatalf("expected run owned by %d, got %d", f.userID, resp.Runs[0].UserID)
	}
}

// T012: GET /wishlist/availability-runs/:id returns 404 for a run owned by a different user
// (cross-scope access must never leak another owner's data).
func TestAvailabilityHandler_GetOwnerRunDetail_CrossOwner404(t *testing.T) {
	f := setupAvailabilityRouter(t, "https://example.test/coin")

	triggerID := f.otherUserID
	otherRun := &models.AvailabilityRun{
		UserID:        f.otherUserID,
		TriggerType:   models.AvailabilityRunTriggerOwner,
		TriggerUserID: &triggerID,
		Status:        models.AvailabilityRunStatusCompleted,
	}
	if err := f.availRepo.CreateChildRun(otherRun); err != nil {
		t.Fatalf("seed other user run: %v", err)
	}

	// The caller (regularUser) attempts to fetch a run owned by otherUser.
	req := httptest.NewRequest(http.MethodGet, "/api/wishlist/availability-runs/"+strconv.Itoa(int(otherRun.ID)), nil)
	req.Header.Set("Authorization", "user-token")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-owner run access, got %d: %s", w.Code, w.Body.String())
	}

	// The actual owner can still fetch it successfully.
	req2 := httptest.NewRequest(http.MethodGet, "/api/wishlist/availability-runs/"+strconv.Itoa(int(otherRun.ID)), nil)
	req2.Header.Set("Authorization", "other-token")
	w2 := httptest.NewRecorder()
	f.router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for the actual owner, got %d: %s", w2.Code, w2.Body.String())
	}
}

// T018: GET /admin/availability-cycles and .../:id return the aggregated parent cycle with
// child run summaries (no per-coin results).
func TestAvailabilityHandler_AdminCycleListAndDetail(t *testing.T) {
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><p>Add to Cart</p></body></html>`))
	}))
	defer listing.Close()

	f := setupAvailabilityRouter(t, listing.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/availability/run", nil)
	req.Header.Set("Authorization", "admin-token")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("trigger: expected 202, got %d: %s", w.Code, w.Body.String())
	}
	var triggerResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &triggerResp)
	cycleID := uint(triggerResp["cycleId"].(float64))

	if err := f.scheduler.ProcessCycle(cycleID); err != nil {
		t.Fatalf("process cycle: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/availability-cycles", nil)
	listReq.Header.Set("Authorization", "admin-token")
	listW := httptest.NewRecorder()
	f.router.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list cycles: expected 200, got %d: %s", listW.Code, listW.Body.String())
	}
	var listResp struct {
		Cycles []models.AvailabilityCycle `json:"cycles"`
		Total  int64                      `json:"total"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("parse list response: %v", err)
	}
	if listResp.Total != 1 || len(listResp.Cycles) != 1 {
		t.Fatalf("expected exactly 1 cycle, got total=%d len=%d", listResp.Total, len(listResp.Cycles))
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/admin/availability-cycles/"+strconv.Itoa(int(cycleID)), nil)
	detailReq.Header.Set("Authorization", "admin-token")
	detailW := httptest.NewRecorder()
	f.router.ServeHTTP(detailW, detailReq)
	if detailW.Code != http.StatusOK {
		t.Fatalf("cycle detail: expected 200, got %d: %s", detailW.Code, detailW.Body.String())
	}
	var detail models.AvailabilityCycle
	if err := json.Unmarshal(detailW.Body.Bytes(), &detail); err != nil {
		t.Fatalf("parse cycle detail: %v", err)
	}
	if detail.Status != models.AvailabilityCycleStatusCompleted {
		t.Fatalf("expected completed cycle, got %q", detail.Status)
	}
	if len(detail.Children) != 1 {
		t.Fatalf("expected 1 child run in cycle detail, got %d", len(detail.Children))
	}
}

// T018: admin cycle endpoints reject non-admin callers.
func TestAvailabilityHandler_AdminCycles_RequiresAdmin(t *testing.T) {
	f := setupAvailabilityRouter(t, "https://example.test/coin")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/availability-cycles", nil)
	req.Header.Set("Authorization", "user-token")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
