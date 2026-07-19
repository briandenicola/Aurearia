package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const testAuctionEndingAdminJWTSecret = "auction-ending-admin-test-jwt-secret"

func setupAuctionEndingAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{}, &models.RefreshToken{}, &models.AuctionLot{},
		&models.AuctionEndingRun{}, &models.AppSetting{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func makeAuctionEndingAdminTestJWT(userID uint, role string) string {
	claims := jwt.MapClaims{
		"userId":   float64(userID),
		"username": "testuser",
		"role":     role,
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(testAuctionEndingAdminJWTSecret))
	return signed
}

// auctionEndingTestAuthMiddleware is a simplified version for testing that extracts userId and role from JWT
func auctionEndingTestAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		tokenString := authHeader[len("Bearer "):]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(testAuctionEndingAdminJWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		c.Set("userId", uint(claims["userId"].(float64)))
		c.Set("userRole", claims["role"])
		c.Next()
	}
}

// auctionEndingTestAdminMiddleware checks if the user has admin role
func auctionEndingTestAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}

func setupAuctionEndingAdminRouter(t *testing.T) (*gin.Engine, *gorm.DB, uint, uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupAuctionEndingAdminTestDB(t)

	// Create admin user
	adminUser := &models.User{Username: "admin", Email: "admin@test.com", Role: "admin"}
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("adminpass"), bcrypt.DefaultCost)
	adminUser.PasswordHash = string(hashedPassword)
	db.Create(adminUser)

	// Create regular user
	regularUser := &models.User{Username: "user", Email: "user@test.com", Role: "user"}
	hashedPassword, _ = bcrypt.GenerateFromPassword([]byte("userpass"), bcrypt.DefaultCost)
	regularUser.PasswordHash = string(hashedPassword)
	db.Create(regularUser)

	// Wire up handler
	auctionEndingRepo := repository.NewAuctionEndingRepository(db)
	auctionLotRepo := repository.NewAuctionLotRepository(db)
	userRepo := repository.NewUserRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := services.NewSettingsService(settingsRepo)
	logger := services.NewLogger(100)
	pushoverSvc := services.NewPushoverService(settingsSvc, logger)
	scheduler := services.NewAuctionEndingScheduler(auctionLotRepo, auctionEndingRepo, userRepo, pushoverSvc, nil, settingsSvc, logger)
	handler := NewAuctionEndingAdminHandler(auctionEndingRepo, scheduler, logger)
	debugHandler := NewAuctionEndingDebugHandler(auctionLotRepo)

	r := gin.New()
	admin := r.Group("/api/admin")
	admin.Use(auctionEndingTestAuthMiddleware())
	admin.Use(auctionEndingTestAdminMiddleware())
	admin.POST("/auction-ending/run", handler.TriggerRun)
	admin.GET("/auction-ending/debug", debugHandler.DebugGetAuctionEndingInfo)
	admin.GET("/auction-ending-runs", handler.ListRuns)
	admin.GET("/auction-ending-runs/:id", handler.GetRun)

	return r, db, adminUser.ID, regularUser.ID
}

// TestAuctionEndingAdminHandler_TriggerRun_AsAdmin verifies admin can trigger a manual run.
func TestAuctionEndingAdminHandler_TriggerRun_AsAdmin(t *testing.T) {
	router, db, adminID, _ := setupAuctionEndingAdminRouter(t)
	adminToken := makeAuctionEndingAdminTestJWT(adminID, "admin")

	// Enable the scheduler
	db.Create(&models.AppSetting{Key: services.SettingAuctionEndingCheckEnabled, Value: "true"})

	req := httptest.NewRequest(http.MethodPost, "/api/admin/auction-ending/run", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["runId"] == nil {
		t.Error("expected 'runId' in response")
	}
	if resp["status"] == nil {
		t.Error("expected 'status' in response")
	}
	// Async: status must be queued (not success/error yet)
	if got := resp["status"].(string); got != "queued" && got != "running" {
		t.Errorf("expected status queued or running, got %q", got)
	}
}

// TestAuctionEndingAdminHandler_TriggerRun_AsRegularUser verifies non-admin users get 403.
func TestAuctionEndingAdminHandler_TriggerRun_AsRegularUser(t *testing.T) {
	router, _, _, regularID := setupAuctionEndingAdminRouter(t)
	regularToken := makeAuctionEndingAdminTestJWT(regularID, "user")

	req := httptest.NewRequest(http.MethodPost, "/api/admin/auction-ending/run", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// TestAuctionEndingAdminHandler_TriggerRun_NoAuth verifies unauthenticated requests get 401.
func TestAuctionEndingAdminHandler_TriggerRun_NoAuth(t *testing.T) {
	router, _, _, _ := setupAuctionEndingAdminRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/auction-ending/run", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// TestAuctionEndingAdminHandler_ListRuns_AsAdmin verifies admin can list run history.
func TestAuctionEndingAdminHandler_ListRuns_AsAdmin(t *testing.T) {
	router, db, adminID, _ := setupAuctionEndingAdminRouter(t)
	adminToken := makeAuctionEndingAdminTestJWT(adminID, "admin")

	// Create 2 runs manually in DB
	repo := repository.NewAuctionEndingRepository(db)
	for i := 0; i < 2; i++ {
		run := &models.AuctionEndingRun{
			TriggerType: "manual",
			Status:      "success",
			StartedAt:   time.Now(),
		}
		repo.CreateRun(run)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/auction-ending-runs", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["runs"] == nil {
		t.Error("expected 'runs' key in response")
	}
	if resp["total"] == nil {
		t.Error("expected 'total' key in response")
	}
}

// TestAuctionEndingAdminHandler_ListRuns_Pagination verifies pagination query params are respected.
func TestAuctionEndingAdminHandler_ListRuns_Pagination(t *testing.T) {
	router, db, adminID, _ := setupAuctionEndingAdminRouter(t)
	adminToken := makeAuctionEndingAdminTestJWT(adminID, "admin")

	// Create 5 runs
	repo := repository.NewAuctionEndingRepository(db)
	for i := 0; i < 5; i++ {
		run := &models.AuctionEndingRun{
			TriggerType: "scheduled",
			Status:      "success",
			StartedAt:   time.Now().Add(time.Duration(-i) * time.Minute),
		}
		repo.CreateRun(run)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/auction-ending-runs?limit=3", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	runs := resp["runs"].([]interface{})
	if len(runs) != 3 {
		t.Errorf("expected 3 runs, got %d", len(runs))
	}
	if resp["total"].(float64) != 5 {
		t.Errorf("expected total=5, got %v", resp["total"])
	}
}

// TestAuctionEndingAdminHandler_ListRuns_NoAuth verifies unauthenticated requests get 401.
func TestAuctionEndingAdminHandler_ListRuns_NoAuth(t *testing.T) {
	router, _, _, _ := setupAuctionEndingAdminRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/auction-ending-runs", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// TestAuctionEndingDebugHandler_Counts verifies the debug endpoint reports global lot counts via the repository.
func TestAuctionEndingDebugHandler_Counts(t *testing.T) {
	router, db, adminID, regularID := setupAuctionEndingAdminRouter(t)
	adminToken := makeAuctionEndingAdminTestJWT(adminID, "admin")

	lots := []models.AuctionLot{
		{
			NumisBidsURL: "https://example.com/admin-bidding-1",
			Title:        "Admin Bidding 1",
			Status:       models.AuctionStatusBidding,
			LotNumber:    1,
			UserID:       adminID,
		},
		{
			NumisBidsURL: "https://example.com/admin-watching-1",
			Title:        "Admin Watching 1",
			Status:       models.AuctionStatusWatching,
			LotNumber:    2,
			UserID:       adminID,
		},
		{
			NumisBidsURL: "https://example.com/user-bidding-1",
			Title:        "User Bidding 1",
			Status:       models.AuctionStatusBidding,
			LotNumber:    3,
			UserID:       regularID,
		},
		{
			NumisBidsURL: "https://example.com/user-won-1",
			Title:        "User Won 1",
			Status:       models.AuctionStatusWon,
			LotNumber:    4,
			UserID:       regularID,
		},
	}
	for i := range lots {
		if err := db.Create(&lots[i]).Error; err != nil {
			t.Fatalf("failed to create auction lot %d: %v", i, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/auction-ending/debug", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if got := resp["total_lots_in_db"].(float64); got != 4 {
		t.Fatalf("expected total_lots_in_db=4, got %v", got)
	}

	lotsByStatus, ok := resp["lots_by_status"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected lots_by_status map, got %T", resp["lots_by_status"])
	}
	expectedCounts := map[string]float64{
		string(models.AuctionStatusBidding):  2,
		string(models.AuctionStatusWatching): 1,
		string(models.AuctionStatusWon):      1,
	}
	for status, expected := range expectedCounts {
		if got := lotsByStatus[status].(float64); got != expected {
			t.Errorf("expected lots_by_status[%q]=%v, got %v", status, expected, got)
		}
	}
}

// TestAuctionEndingAdminHandler_TriggerRun_Returns202 verifies TriggerRun returns 202 Accepted.
func TestAuctionEndingAdminHandler_TriggerRun_Returns202(t *testing.T) {
	router, db, adminID, _ := setupAuctionEndingAdminRouter(t)
	adminToken := makeAuctionEndingAdminTestJWT(adminID, "admin")
	db.Create(&models.AppSetting{Key: services.SettingAuctionEndingCheckEnabled, Value: "true"})

	req := httptest.NewRequest(http.MethodPost, "/api/admin/auction-ending/run", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["runId"] == nil {
		t.Error("expected 'runId' in 202 response")
	}
	status, _ := resp["status"].(string)
	if status != "queued" && status != "running" {
		t.Errorf("expected status queued or running, got %q", status)
	}
}

// TestAuctionEndingAdminHandler_TriggerRun_DedupActiveRun verifies a second trigger reuses the active run.
func TestAuctionEndingAdminHandler_TriggerRun_DedupActiveRun(t *testing.T) {
	router, db, adminID, _ := setupAuctionEndingAdminRouter(t)
	adminToken := makeAuctionEndingAdminTestJWT(adminID, "admin")

	// Seed an active queued run in the database to simulate an in-flight run.
	repo := repository.NewAuctionEndingRepository(db)
	active := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "queued",
		StartedAt:   time.Now(),
	}
	if err := repo.CreateRun(active); err != nil {
		t.Fatalf("failed to seed active run: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/auction-ending/run", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Should return the existing active run ID, not a new one.
	if got := uint(resp["runId"].(float64)); got != active.ID {
		t.Errorf("expected dedup to return active run ID=%d, got %d", active.ID, got)
	}

	// Verify no new run was created.
	_, total, _ := repo.ListRuns(1, 100)
	if total != 1 {
		t.Errorf("expected exactly 1 run in DB after dedup, got %d", total)
	}
}

// TestAuctionEndingAdminHandler_GetRun_Found verifies GetRun returns a run by ID.
func TestAuctionEndingAdminHandler_GetRun_Found(t *testing.T) {
	router, db, adminID, _ := setupAuctionEndingAdminRouter(t)
	adminToken := makeAuctionEndingAdminTestJWT(adminID, "admin")

	repo := repository.NewAuctionEndingRepository(db)
	now := time.Now()
	run := &models.AuctionEndingRun{
		TriggerType: "manual",
		Status:      "success",
		StartedAt:   now,
		CompletedAt: &now,
		LotsChecked: 7,
		AlertsSent:  2,
	}
	repo.CreateRun(run)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/admin/auction-ending-runs/%d", run.ID), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if got := uint(resp["id"].(float64)); got != run.ID {
		t.Errorf("expected id=%d, got %d", run.ID, got)
	}
	if resp["status"] != "success" {
		t.Errorf("expected status success, got %v", resp["status"])
	}
}

// TestAuctionEndingAdminHandler_GetRun_NotFound verifies GetRun returns 404 for missing runs.
func TestAuctionEndingAdminHandler_GetRun_NotFound(t *testing.T) {
	router, _, adminID, _ := setupAuctionEndingAdminRouter(t)
	adminToken := makeAuctionEndingAdminTestJWT(adminID, "admin")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/auction-ending-runs/99999", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
