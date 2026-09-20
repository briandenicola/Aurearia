package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

func TestIntakeCommitReportsValidationAndAllowsCorrection(t *testing.T) {
	db := setupCoinHandlerTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.CoinIntakeDraft{}); err != nil {
		t.Fatal(err)
	}
	coinRepo := repository.NewCoinRepository(db)
	service := services.NewCoinIntakeService(repository.NewCoinIntakeDraftRepository(db), coinRepo, nil, nil).
		WithCoinService(services.NewCoinService(coinRepo, nil))
	handler := NewCoinIntakeHandler(service, services.NewLogger(20))
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userId", uint(1)) })
	router.POST("/commit", handler.CommitDraft)
	draft := models.CoinIntakeDraft{
		UserID: 1, DraftPayload: `{"name":"Denarius","category":"Roman","era":"Roman Imperial"}`,
		Status: models.CoinIntakeDraftStatusDrafted, ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		overrides string
		status    int
		message   string
	}{
		{`{}`, http.StatusBadRequest, "era is not supported"},
		{`{"era":""}`, http.StatusOK, `"coinId":`},
		{`{"era":""}`, http.StatusConflict, "no longer confirmable"},
	} {
		body := fmt.Sprintf(`{"draftId":%d,"confirm":true,"overrides":%s}`, draft.ID, test.overrides)
		req := httptest.NewRequest(http.MethodPost, "/commit", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != test.status || !strings.Contains(response.Body.String(), test.message) {
			t.Fatalf("response = %d %s; want %d %s", response.Code, response.Body.String(), test.status, test.message)
		}
	}
	var count int64
	if err := db.Model(&models.Coin{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected exactly one coin: count=%d err=%v", count, err)
	}
}
