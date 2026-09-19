package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCollectorProfileHandler(t *testing.T) (*gin.Engine, *services.CollectorProfileService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:collector_profile_handler_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.CollectorProfile{}); err != nil {
		t.Fatal(err)
	}
	service := services.NewCollectorProfileService(repository.NewCollectorProfileRepository(db))
	handler := NewCollectorProfileHandler(service)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if raw := c.GetHeader("X-Test-User"); raw != "" {
			id, _ := strconv.ParseUint(raw, 10, 64)
			c.Set("userId", uint(id))
		}
		c.Next()
	})
	router.GET("/api/collector-profile", handler.Get)
	router.PUT("/api/collector-profile", handler.Put)
	return router, service
}

func performCollectorProfileRequest(router *gin.Engine, method string, userID uint, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/collector-profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if userID != 0 {
		req.Header.Set("X-Test-User", strconv.FormatUint(uint64(userID), 10))
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestCollectorProfileHandlerAuthenticationNeutralAndOwnerPrivacy(t *testing.T) {
	router, service := setupCollectorProfileHandler(t)
	if w := performCollectorProfileRequest(router, http.MethodGet, 0, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated GET=%d body=%s", w.Code, w.Body.String())
	}
	neutral := performCollectorProfileRequest(router, http.MethodGet, 1, "")
	if neutral.Code != http.StatusOK {
		t.Fatalf("neutral GET=%d body=%s", neutral.Code, neutral.Body.String())
	}
	var neutralBody map[string]any
	if err := json.Unmarshal(neutral.Body.Bytes(), &neutralBody); err != nil {
		t.Fatal(err)
	}
	if neutralBody["isDefault"] != true || neutralBody["updatedAt"] != nil ||
		neutralBody["preferredPeriods"] == nil || neutralBody["budgetMin"] != nil {
		t.Fatalf("invalid neutral response: %#v", neutralBody)
	}
	currency := "USD"
	if _, err := service.Replace(1, services.CollectorProfileInput{
		Currency: &currency, PreferredPeriods: []string{"Secret period"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, other := range []uint{2, 3, 4} { // public owner, follower, and administrator sessions
		w := performCollectorProfileRequest(router, http.MethodGet, other, "")
		if w.Code != http.StatusOK || bytes.Contains(w.Body.Bytes(), []byte("Secret period")) {
			t.Fatalf("session %d disclosed owner profile: %d %s", other, w.Code, w.Body.String())
		}
	}
}

func TestCollectorProfileHandlerStrictReplacementAndSanitizedErrors(t *testing.T) {
	router, _ := setupCollectorProfileHandler(t)
	valid := `{"budgetMin":100,"budgetMax":500,"currency":"USD","preferredPeriods":["Flavian"],"preferredCategories":[],"excludedCategories":[],"preferredDealers":[],"collectingGoals":[]}`
	if w := performCollectorProfileRequest(router, http.MethodPut, 1, valid); w.Code != http.StatusOK {
		t.Fatalf("valid PUT=%d body=%s", w.Code, w.Body.String())
	}
	for _, forged := range []string{
		`{"userId":2}`,
		`{"ownerId":2}`,
		`{"currency":"USD","unknown":true}`,
		`{} {}`,
	} {
		w := performCollectorProfileRequest(router, http.MethodPut, 1, forged)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("forged/unknown payload %q=%d body=%s", forged, w.Code, w.Body.String())
		}
	}
	invalid := performCollectorProfileRequest(router, http.MethodPut, 1, `{"currency":"usd"}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid PUT=%d body=%s", invalid.Code, invalid.Body.String())
	}
	var validation map[string]any
	if err := json.Unmarshal(invalid.Body.Bytes(), &validation); err != nil {
		t.Fatal(err)
	}
	if validation["field"] != "currency" || validation["error"] != "Invalid collector profile" {
		t.Fatalf("validation error was not field-specific and sanitized: %#v", validation)
	}
	after := performCollectorProfileRequest(router, http.MethodGet, 1, "")
	if !bytes.Contains(after.Body.Bytes(), []byte(`"currency":"USD"`)) {
		t.Fatalf("invalid replacement changed stored row: %s", after.Body.String())
	}
}

func TestCollectorProfileHandlerBodyLimit(t *testing.T) {
	router, _ := setupCollectorProfileHandler(t)
	for name, oversized := range map[string]string{
		"oversized value":    `{"collectingGoals":["` + strings.Repeat("x", int(CollectorProfileBodyLimit)+1) + `"]}`,
		"oversized trailing": `{}` + strings.Repeat(" ", int(CollectorProfileBodyLimit)+1),
	} {
		t.Run(name, func(t *testing.T) {
			w := performCollectorProfileRequest(router, http.MethodPut, 1, oversized)
			if w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("oversized PUT=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
