package handlers

import (
	"bytes"
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

func setupMintLocationHandlerRouter(t *testing.T, authenticated bool, role models.UserRole) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.MintLocation{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	svc := services.NewMintLocationService(repository.NewMintLocationRepository(db))
	handler := NewMintLocationHandler(svc)
	r := gin.New()

	protected := r.Group("/api")
	protected.Use(func(c *gin.Context) {
		if !authenticated {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.Set("userRole", string(role))
		c.Next()
	})
	protected.GET("/mint-locations", handler.List)

	admin := protected.Group("/admin")
	admin.Use(AdminRequired())
	admin.POST("/mint-locations", handler.Create)
	admin.PUT("/mint-locations/:id", handler.Update)
	admin.DELETE("/mint-locations/:id", handler.Delete)

	return r, db
}

func performMintLocationRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestMintLocationHandler_ListRequiresAuthentication(t *testing.T) {
	router, _ := setupMintLocationHandlerRouter(t, false, models.RoleUser)

	w := performMintLocationRequest(router, http.MethodGet, "/api/mint-locations", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMintLocationHandler_ListReturnsMintLocations(t *testing.T) {
	router, db := setupMintLocationHandlerRouter(t, true, models.RoleUser)
	if err := db.Create(&models.MintLocation{
		DisplayName:    "Rome",
		NormalizedName: models.NormalizeMintLocationName("Rome"),
		Lat:            41.9028,
		Lng:            12.4964,
		Aliases:        models.StringList{"Roma"},
	}).Error; err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	w := performMintLocationRequest(router, http.MethodGet, "/api/mint-locations", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		MintLocations []models.MintLocation `json:"mintLocations"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.MintLocations) != 1 || resp.MintLocations[0].DisplayName != "Rome" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestMintLocationHandler_AdminEndpointsRequireAdmin(t *testing.T) {
	router, _ := setupMintLocationHandlerRouter(t, true, models.RoleUser)

	body := `{"displayName":"Rome","lat":41.9028,"lng":12.4964,"aliases":[]}`
	w := performMintLocationRequest(router, http.MethodPost, "/api/admin/mint-locations", body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMintLocationHandler_AdminCreateUpdateDelete(t *testing.T) {
	router, _ := setupMintLocationHandlerRouter(t, true, models.RoleAdmin)

	createBody := `{"displayName":" Rome ","lat":41.9028,"lng":12.4964,"region":" Italy ","aliases":["Roma","Rome mint"]}`
	w := performMintLocationRequest(router, http.MethodPost, "/api/admin/mint-locations", createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created models.MintLocation
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.DisplayName != "Rome" || created.Region != "Italy" {
		t.Fatalf("expected trimmed fields, got %+v", created)
	}

	updateBody := `{"displayName":"Roma","lat":41.9,"lng":12.5,"aliases":["Rome"]}`
	id := strconv.FormatUint(uint64(created.ID), 10)
	w = performMintLocationRequest(router, http.MethodPut, "/api/admin/mint-locations/"+id, updateBody)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	w = performMintLocationRequest(router, http.MethodDelete, "/api/admin/mint-locations/"+id, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMintLocationHandler_AdminDuplicateReturnsConflict(t *testing.T) {
	router, _ := setupMintLocationHandlerRouter(t, true, models.RoleAdmin)

	body := `{"displayName":"Rome","lat":41.9028,"lng":12.4964}`
	w := performMintLocationRequest(router, http.MethodPost, "/api/admin/mint-locations", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w = performMintLocationRequest(router, http.MethodPost, "/api/admin/mint-locations", body)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}
