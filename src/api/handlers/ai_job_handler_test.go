package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type fakeAIJobHandlerAgent struct{}

func (fakeAIJobHandlerAgent) AnalyzeCoin(context.Context, services.AnalyzeProxyRequest) (string, error) {
	return "analysis", nil
}

func (fakeAIJobHandlerAgent) GradeCoin(context.Context, services.GradeProxyRequest) (string, error) {
	return "grading report", nil
}

func (fakeAIJobHandlerAgent) CollectPortfolioReview(context.Context, services.PortfolioReviewProxyRequest) (string, error) {
	return "", errors.New("not used")
}

func setupAIJobHandlerTest(t *testing.T, userID uint) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.AppSetting{}, &models.AIJob{}, &models.CoinJournal{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	if err := db.Create(&models.User{ID: userID, Username: "grader", Email: "grader@example.com", PasswordHash: "x"}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	db.Create(&models.AppSetting{Key: services.SettingAIProvider, Value: "ollama"})
	db.Create(&models.AppSetting{Key: services.SettingOllamaModel, Value: "llava"})

	settingsSvc := services.NewSettingsService(repository.NewSettingsRepository(db))
	svc := services.NewAIJobService(
		repository.NewAIJobRepository(db),
		fakeAIJobHandlerAgent{},
		repository.NewUserRepository(db),
		settingsSvc,
		nil,
		services.NewLogger(10),
	)
	handler := NewAIJobHandler(svc, services.NewLogger(10))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", userID)
		c.Next()
	})
	router.POST("/api/coins/:id/grade", handler.Grade)
	return router, db
}

func TestAIJobHandlerGradeRejectsNoImageClearly(t *testing.T) {
	router, db := setupAIJobHandlerTest(t, 1)
	coin := models.Coin{Name: "No image coin", UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/coins/1/grade", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if w.Body.String() != `{"error":"No image available for grading"}` {
		t.Fatalf("unexpected error body: %s", w.Body.String())
	}
}

func TestAIJobHandlerGradeOwnerScoped(t *testing.T) {
	router, db := setupAIJobHandlerTest(t, 2)
	coin := models.Coin{Name: "Other user's coin", UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}
	if err := db.Create(&models.CoinImage{CoinID: coin.ID, FilePath: "x.jpg", ImageType: models.ImageTypeObverse}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/coins/1/grade", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAIJobHandlerGradeEnqueuesJob(t *testing.T) {
	router, db := setupAIJobHandlerTest(t, 1)
	coin := models.Coin{Name: "Image coin", UserID: 1}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}
	if err := db.Create(&models.CoinImage{CoinID: coin.ID, FilePath: "x.jpg", ImageType: models.ImageTypeObverse}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/coins/1/grade", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	var job models.AIJob
	if err := db.Where("coin_id = ? AND user_id = ? AND job_type = ?", coin.ID, uint(1), models.AIJobTypeCoinGrading).First(&job).Error; err != nil {
		t.Fatalf("expected grading job: %v", err)
	}
}
