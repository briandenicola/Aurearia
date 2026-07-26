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
	return setupMintLocationHandlerRouterForUser(t, authenticated, role, 1)
}

func setupMintLocationHandlerRouterForUser(t *testing.T, authenticated bool, role models.UserRole, userID uint) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.MintLocation{}); err != nil {
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
		c.Set("userId", userID)
		c.Set("userRole", string(role))
		c.Next()
	})
	protected.GET("/mint-locations", handler.List)
	protected.GET("/mint-locations/geocode", handler.Geocode)
	protected.POST("/mint-locations", handler.CreatePrivate)
	protected.PUT("/mint-locations/:id", handler.UpdatePrivate)
	protected.DELETE("/mint-locations/:id", handler.DeletePrivate)

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

func TestMintLocationHandler_ListExcludesOtherUsersPrivateMints(t *testing.T) {
	router, db := setupMintLocationHandlerRouterForUser(t, true, models.RoleUser, 1)
	otherUser := uint(2)
	if err := db.Create(&models.MintLocation{
		UserID:         &otherUser,
		DisplayName:    "Someone Else's Mint",
		NormalizedName: models.NormalizeMintLocationName("Someone Else's Mint"),
		Lat:            1,
		Lng:            1,
		Aliases:        models.StringList{},
	}).Error; err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	w := performMintLocationRequest(router, http.MethodGet, "/api/mint-locations", "")
	var resp struct {
		MintLocations []models.MintLocation `json:"mintLocations"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.MintLocations) != 0 {
		t.Fatalf("expected another user's private mint to be excluded, got %+v", resp.MintLocations)
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

func TestMintLocationHandler_AdminDeleteRejectsWhenCoinInUse(t *testing.T) {
	router, db := setupMintLocationHandlerRouter(t, true, models.RoleAdmin)

	body := `{"displayName":"Rome","lat":41.9028,"lng":12.4964}`
	w := performMintLocationRequest(router, http.MethodPost, "/api/admin/mint-locations", body)
	var created models.MintLocation
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	if err := db.Create(&models.Coin{Name: "Test Coin", UserID: 1, MintLocationID: &created.ID}).Error; err != nil {
		t.Fatalf("seed coin failed: %v", err)
	}

	id := strconv.FormatUint(uint64(created.ID), 10)
	w = performMintLocationRequest(router, http.MethodDelete, "/api/admin/mint-locations/"+id, "")
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 when mint location is in use, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMintLocationHandler_SelfServiceCreateUpdateDelete(t *testing.T) {
	router, _ := setupMintLocationHandlerRouter(t, true, models.RoleUser)

	createBody := `{"displayName":"My Custom Mint","lat":10,"lng":10,"aliases":[]}`
	w := performMintLocationRequest(router, http.MethodPost, "/api/mint-locations", createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created models.MintLocation
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.UserID == nil || *created.UserID != 1 {
		t.Fatalf("expected private mint owned by user 1, got %+v", created)
	}

	updateBody := `{"displayName":"My Renamed Mint","lat":11,"lng":11,"aliases":[]}`
	id := strconv.FormatUint(uint64(created.ID), 10)
	w = performMintLocationRequest(router, http.MethodPut, "/api/mint-locations/"+id, updateBody)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	w = performMintLocationRequest(router, http.MethodDelete, "/api/mint-locations/"+id, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMintLocationHandler_SelfServiceCannotMutateGlobalMint(t *testing.T) {
	router, db := setupMintLocationHandlerRouter(t, true, models.RoleUser)
	if err := db.Create(&models.MintLocation{
		DisplayName:    "Rome",
		NormalizedName: models.NormalizeMintLocationName("Rome"),
		Lat:            41.9,
		Lng:            12.5,
		Aliases:        models.StringList{},
	}).Error; err != nil {
		t.Fatalf("seed global mint failed: %v", err)
	}
	var global models.MintLocation
	if err := db.First(&global).Error; err != nil {
		t.Fatalf("load global mint failed: %v", err)
	}

	id := strconv.FormatUint(uint64(global.ID), 10)
	updateBody := `{"displayName":"Hijacked","lat":1,"lng":1,"aliases":[]}`
	w := performMintLocationRequest(router, http.MethodPut, "/api/mint-locations/"+id, updateBody)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when a user tries to edit a global mint via self-service, got %d: %s", w.Code, w.Body.String())
	}

	w = performMintLocationRequest(router, http.MethodDelete, "/api/mint-locations/"+id, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when a user tries to delete a global mint via self-service, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMintLocationHandler_Geocode_ReturnsCandidatesFromGeocoder(t *testing.T) {
	nominatim := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"display_name":"Rome, Italy","lat":"41.9028","lon":"12.4964"}]`))
	}))
	defer nominatim.Close()

	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.MintLocation{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	svc := services.NewMintLocationService(repository.NewMintLocationRepository(db))
	handler := NewMintLocationHandler(svc).WithGeocoding(services.NewGeocodeServiceForTest(nominatim.URL))
	r := gin.New()
	protected := r.Group("/api")
	protected.Use(func(c *gin.Context) {
		c.Set("userId", uint(1))
		c.Next()
	})
	protected.GET("/mint-locations/geocode", handler.Geocode)

	w := performMintLocationRequest(r, http.MethodGet, "/api/mint-locations/geocode?query=Rome", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Candidates []services.GeocodeCandidate `json:"candidates"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Candidates) != 1 || resp.Candidates[0].DisplayName != "Rome, Italy" {
		t.Fatalf("expected one Rome candidate, got %+v", resp.Candidates)
	}
}

func TestMintLocationHandler_Geocode_NoClientReturnsEmptyCandidates(t *testing.T) {
	router, _ := setupMintLocationHandlerRouter(t, true, models.RoleUser)

	w := performMintLocationRequest(router, http.MethodGet, "/api/mint-locations/geocode?query=Rome", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Candidates []services.GeocodeCandidate `json:"candidates"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Candidates) != 0 {
		t.Fatalf("expected no candidates without a geocoder wired in, got %+v", resp.Candidates)
	}
}

func TestMintLocationHandler_SelfServiceCannotMutateAnotherUsersPrivateMint(t *testing.T) {
	router, db := setupMintLocationHandlerRouterForUser(t, true, models.RoleUser, 1)
	otherUser := uint(2)
	if err := db.Create(&models.MintLocation{
		UserID:         &otherUser,
		DisplayName:    "Someone Else's Mint",
		NormalizedName: models.NormalizeMintLocationName("Someone Else's Mint"),
		Lat:            1,
		Lng:            1,
		Aliases:        models.StringList{},
	}).Error; err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	var other models.MintLocation
	if err := db.First(&other).Error; err != nil {
		t.Fatalf("load failed: %v", err)
	}

	id := strconv.FormatUint(uint64(other.ID), 10)
	w := performMintLocationRequest(router, http.MethodDelete, "/api/mint-locations/"+id, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when a user tries to delete another user's private mint, got %d: %s", w.Code, w.Body.String())
	}
}
