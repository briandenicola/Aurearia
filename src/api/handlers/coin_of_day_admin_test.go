package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

const testCoinOfDayAdminJWTSecret = "coin-of-day-admin-test-jwt-secret"

func setupCoinOfDayAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{}, &models.RefreshToken{}, &models.Coin{}, &models.FeaturedCoin{},
		&models.CoinOfDayRun{}, &models.AppSetting{}, &models.Notification{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func makeCoinOfDayAdminTestJWT(userID uint, role string) string {
	claims := jwt.MapClaims{
		"userId":   float64(userID),
		"username": "testuser",
		"role":     role,
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(testCoinOfDayAdminJWTSecret))
	return signed
}

func coinOfDayTestAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		tokenString := authHeader[len("Bearer "):]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(testCoinOfDayAdminJWTSecret), nil
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

func coinOfDayTestAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}

func setupCoinOfDayAdminRouter(t *testing.T) (*gin.Engine, *gorm.DB, uint, uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := setupCoinOfDayAdminTestDB(t)

	adminUser := &models.User{Username: "admin", Email: "admin-cotd@test.com", Role: "admin"}
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("adminpass"), bcrypt.DefaultCost)
	adminUser.PasswordHash = string(hashedPassword)
	db.Create(adminUser)

	regularUser := &models.User{Username: "user", Email: "user-cotd@test.com", Role: "user"}
	hashedPassword, _ = bcrypt.GenerateFromPassword([]byte("userpass"), bcrypt.DefaultCost)
	regularUser.PasswordHash = string(hashedPassword)
	db.Create(regularUser)

	featuredRepo := repository.NewFeaturedCoinRepository(db)
	runRepo := repository.NewCoinOfDayRunRepository(db)
	userRepo := repository.NewUserRepository(db)
	coinRepo := repository.NewCoinRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := services.NewSettingsService(settingsRepo)
	logger := services.NewLogger(50)
	scheduler := services.NewCoinOfDayScheduler(featuredRepo, runRepo, userRepo, coinRepo, nil, settingsSvc, logger)
	handler := NewCoinOfDayAdminHandler(scheduler, logger)

	r := gin.New()
	admin := r.Group("/api/admin")
	admin.Use(coinOfDayTestAuthMiddleware())
	admin.Use(coinOfDayTestAdminMiddleware())
	admin.POST("/coin-of-day/run", handler.TriggerRun)
	admin.GET("/coin-of-day-runs", handler.ListRuns)
	admin.GET("/coin-of-day-runs/:id", handler.GetRun)

	return r, db, adminUser.ID, regularUser.ID
}

func TestCoinOfDayAdminHandlerTriggerRunReturnsAccepted(t *testing.T) {
	router, _, adminID, _ := setupCoinOfDayAdminRouter(t)
	token := makeCoinOfDayAdminTestJWT(adminID, "admin")
	req := httptest.NewRequest(http.MethodPost, "/api/admin/coin-of-day/run", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["runId"] == nil {
		t.Fatalf("expected runId in response")
	}
}

func TestCoinOfDayAdminHandlerListRunsAsAdmin(t *testing.T) {
	router, db, adminID, _ := setupCoinOfDayAdminRouter(t)
	token := makeCoinOfDayAdminTestJWT(adminID, "admin")
	if err := db.Create(&models.CoinOfDayRun{
		TriggerType: models.CoinOfDayRunTriggerManual,
		Status:      models.CoinOfDayRunStatusCompleted,
		StartedAt:   time.Now(),
		CompletedAt: ptrTime(time.Now()),
		Picked:      1,
	}).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/coin-of-day-runs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["runs"] == nil || resp["total"] == nil {
		t.Fatalf("expected runs and total in response")
	}
}

func TestCoinOfDayAdminHandlerGetRunAsAdmin(t *testing.T) {
	router, db, adminID, _ := setupCoinOfDayAdminRouter(t)
	token := makeCoinOfDayAdminTestJWT(adminID, "admin")
	run := &models.CoinOfDayRun{
		TriggerType: models.CoinOfDayRunTriggerManual,
		Status:      models.CoinOfDayRunStatusQueued,
		StartedAt:   time.Now(),
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/coin-of-day-runs/"+strconv.FormatUint(uint64(run.ID), 10), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
