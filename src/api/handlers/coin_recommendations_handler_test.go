package handlers

import (
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

func setupRecommendationHandlerRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.Tag{},
		&models.CoinTag{},
		&models.CoinSet{},
		&models.CoinSetMembership{},
		&models.CoinRecommendation{},
		&models.RecommendationFeedback{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := repository.NewCoinRecommendationRepository(db)
	service := services.NewCoinRecommendationService(repo, repository.NewTagRepository(db), repository.NewSetRepository(db))
	handler := NewCoinRecommendationHandler(service)

	r := gin.New()
	protected := r.Group("/api")
	protected.Use(coinTestAuthMiddleware())
	protected.GET("/coins/:id/recommendations", handler.List)
	protected.POST("/coins/:id/recommendations/:recommendationId/reject", handler.Reject)
	return r, db
}

func TestCoinRecommendationHandler_ListAndReject(t *testing.T) {
	router, db := setupRecommendationHandlerRouter(t)
	user := models.User{ID: 1, Username: "owner", PasswordHash: "hash", Email: "owner@example.com"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	target := models.Coin{Name: "Julia Domna denarius", Ruler: "Julia Domna", Category: models.CategoryRoman, Era: models.EraAncient, UserID: 1}
	peer1 := models.Coin{Name: "Peer 1", Ruler: "Julia Domna", Category: models.CategoryRoman, Era: models.EraAncient, UserID: 1}
	peer2 := models.Coin{Name: "Peer 2", Ruler: "Julia Domna", Category: models.CategoryRoman, Era: models.EraAncient, UserID: 1}
	if err := db.Create(&[]models.Coin{target, peer1, peer2}).Error; err != nil {
		t.Fatalf("failed to create coins: %v", err)
	}
	var coins []models.Coin
	if err := db.Order("id ASC").Find(&coins).Error; err != nil {
		t.Fatalf("failed to reload coins: %v", err)
	}
	target = coins[0]
	peer1 = coins[1]
	peer2 = coins[2]

	set := models.CoinSet{UserID: 1, Name: "Severan Women", SetType: models.CoinSetTypeGoal}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("failed to create set: %v", err)
	}
	if err := db.Create(&[]models.CoinSetMembership{
		{SetID: set.ID, CoinID: peer1.ID},
		{SetID: set.ID, CoinID: peer2.ID},
	}).Error; err != nil {
		t.Fatalf("failed to create memberships: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/coins/"+strconvUint(target.ID)+"/recommendations", nil)
	listReq.Header.Set("Authorization", authHeader(1))
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d body=%s", listResp.Code, listResp.Body.String())
	}

	var rec models.CoinRecommendation
	if err := db.Where("coin_id = ? AND user_id = ?", target.ID, 1).First(&rec).Error; err != nil {
		t.Fatalf("expected persisted recommendation: %v", err)
	}
	rejectReq := httptest.NewRequest(http.MethodPost, "/api/coins/"+strconvUint(target.ID)+"/recommendations/"+strconvUint(rec.ID)+"/reject", nil)
	rejectReq.Header.Set("Authorization", authHeader(1))
	rejectResp := httptest.NewRecorder()
	router.ServeHTTP(rejectResp, rejectReq)
	if rejectResp.Code != http.StatusOK {
		t.Fatalf("expected reject status 200, got %d body=%s", rejectResp.Code, rejectResp.Body.String())
	}
}
