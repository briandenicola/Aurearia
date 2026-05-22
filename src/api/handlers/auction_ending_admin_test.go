package handlers

import (
	"encoding/json"
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
	scheduler := services.NewAuctionEndingScheduler(auctionLotRepo, auctionEndingRepo, userRepo, pushoverSvc, settingsSvc, logger)
	handler := NewAuctionEndingAdminHandler(auctionEndingRepo, scheduler, logger)

	r := gin.New()
	admin := r.Group("/api/admin")
	admin.Use(auctionEndingTestAuthMiddleware())
	admin.Use(auctionEndingTestAdminMiddleware())
	admin.POST("/auction-ending/run", handler.TriggerRun)
	admin.GET("/auction-ending-runs", handler.ListRuns)

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

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
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
