package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const showcaseTestJWTSecret = "showcase-handler-test-secret"

var showcaseHandlerDBCounter atomic.Uint64

func setupShowcaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:showcase_handler_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), showcaseHandlerDBCounter.Add(1))), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{}, &models.StorageLocation{}, &models.Coin{}, &models.CoinImage{},
		&models.Tag{}, &models.CoinTag{},
		&models.Showcase{}, &models.ShowcaseCoin{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func makeShowcaseTestJWT(userID uint) string {
	claims := jwt.MapClaims{
		"userId":   float64(userID),
		"username": "testuser",
		"role":     "user",
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(showcaseTestJWTSecret))
	return signed
}

func showcaseTestAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		tokenString := authHeader[len("Bearer "):]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(showcaseTestJWTSecret), nil
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

func setupShowcaseRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupShowcaseTestDB(t)
	repo := repository.NewShowcaseRepository(db)
	handler := NewShowcaseHandler(repo)

	r := gin.New()
	protected := r.Group("/api", showcaseTestAuthMiddleware())
	protected.GET("/showcases", handler.ListShowcases)
	protected.GET("/showcases/:id", handler.GetShowcase)
	protected.POST("/showcases", handler.CreateShowcase)
	protected.PUT("/showcases/:id", handler.UpdateShowcase)
	protected.DELETE("/showcases/:id", handler.DeleteShowcase)
	protected.PUT("/showcases/:id/coins", handler.SetShowcaseCoins)
	r.GET("/api/showcase/:slug", handler.GetPublicShowcase)

	return r, db
}

// TestShowcaseEditFlow validates the full editing workflow:
// create a showcase, fetch it for editing, update title/description, and set coins.
func TestShowcaseEditFlow(t *testing.T) {
	r, db := setupShowcaseRouter(t)

	// Create a user and some coins
	user := models.User{Username: "testuser", Email: "test@example.com", PasswordHash: "x"}
	db.Create(&user)

	coins := []models.Coin{
		{Name: "Denarius", Category: "Roman", UserID: user.ID},
		{Name: "Tetradrachm", Category: "Greek", UserID: user.ID},
		{Name: "Solidus", Category: "Byzantine", UserID: user.ID},
	}
	for i := range coins {
		db.Create(&coins[i])
	}

	token := makeShowcaseTestJWT(user.ID)

	// Step 1: Create a showcase
	createBody, _ := json.Marshal(map[string]string{
		"title":       "My Roman Collection",
		"description": "Best coins",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/showcases", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create, got %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	showcaseID := int(createResp["id"].(float64))

	// Step 2: Get the showcase for editing (this is what ShowcaseEditPage does)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/showcases/%d", showcaseID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on get showcase, got %d: %s", w.Code, w.Body.String())
	}

	var getResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &getResp)

	showcase, ok := getResp["showcase"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing 'showcase' object")
	}
	if showcase["title"] != "My Roman Collection" {
		t.Errorf("expected title 'My Roman Collection', got %q", showcase["title"])
	}
	if showcase["description"] != "Best coins" {
		t.Errorf("expected description 'Best coins', got %q", showcase["description"])
	}

	// Step 3: Update the title
	updateBody, _ := json.Marshal(map[string]string{"title": "Updated Title"})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/showcases/%d", showcaseID), bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on update title, got %d: %s", w.Code, w.Body.String())
	}

	// Step 4: Set coins
	coinIDs := []uint{coins[0].ID, coins[2].ID}
	setBody, _ := json.Marshal(map[string]interface{}{"coinIds": coinIDs})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/showcases/%d/coins", showcaseID), bytes.NewBuffer(setBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on set coins, got %d: %s", w.Code, w.Body.String())
	}

	// Step 5: Verify the showcase now has updated data and coins
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/showcases/%d", showcaseID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on final get, got %d: %s", w.Code, w.Body.String())
	}

	json.Unmarshal(w.Body.Bytes(), &getResp)
	showcase = getResp["showcase"].(map[string]interface{})
	if showcase["title"] != "Updated Title" {
		t.Errorf("expected updated title, got %q", showcase["title"])
	}

	respCoins, ok := getResp["coins"].([]interface{})
	if !ok {
		t.Fatal("response missing 'coins' array")
	}
	if len(respCoins) != 2 {
		t.Errorf("expected 2 coins in showcase, got %d", len(respCoins))
	}

	coinIDsResp := showcase["coinIds"].([]interface{})
	if len(coinIDsResp) != 2 {
		t.Errorf("expected 2 coinIds, got %d", len(coinIDsResp))
	}
}

// TestShowcaseGetNotFound verifies that requesting a non-existent showcase returns 404.
func TestShowcaseGetNotFound(t *testing.T) {
	r, db := setupShowcaseRouter(t)

	user := models.User{Username: "testuser", Email: "test@example.com", PasswordHash: "x"}
	db.Create(&user)

	token := makeShowcaseTestJWT(user.ID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/showcases/999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent showcase, got %d", w.Code)
	}
}

func TestPublicShowcaseReturnsOnlyMemberCoinsWithTrayFields(t *testing.T) {
	r, db := setupShowcaseRouter(t)

	user := models.User{Username: "testuser", Email: "test@example.com", PasswordHash: "x"}
	otherUser := models.User{Username: "otheruser", Email: "other@example.com", PasswordHash: "x"}
	db.Create(&user)
	db.Create(&otherUser)

	memberDiameter := 18.5
	nonMemberDiameter := 24.0
	memberCoin := models.Coin{Name: "Included Denarius", Category: "Roman", UserID: user.ID, DiameterMm: &memberDiameter}
	otherMemberCoin := models.Coin{Name: "Included Sestertius", Category: "Roman", UserID: user.ID}
	nonMemberCoin := models.Coin{Name: "Non-member Aureus", Category: "Roman", UserID: user.ID, DiameterMm: &nonMemberDiameter}
	otherShowcaseCoin := models.Coin{Name: "Other Showcase Coin", Category: "Greek", UserID: user.ID}
	otherOwnerCoin := models.Coin{Name: "Other Owner Linked Coin", Category: "Roman", UserID: otherUser.ID}
	db.Create(&memberCoin)
	db.Create(&otherMemberCoin)
	db.Create(&nonMemberCoin)
	db.Create(&otherShowcaseCoin)
	db.Create(&otherOwnerCoin)

	db.Create(&models.CoinImage{CoinID: memberCoin.ID, FilePath: "cards/denarius-card.webp", ImageType: models.ImageTypeDetail, IsPrimary: true})
	db.Create(&models.CoinImage{CoinID: memberCoin.ID, FilePath: "coins/denarius-obverse.webp", ImageType: models.ImageTypeObverse})

	showcase := models.Showcase{
		UserID:      user.ID,
		Slug:        "featured-set",
		Title:       "Featured Set",
		Description: "Public tray",
		IsActive:    true,
	}
	otherShowcase := models.Showcase{
		UserID:   user.ID,
		Slug:     "other-set",
		Title:    "Other Set",
		IsActive: true,
	}
	db.Create(&showcase)
	db.Create(&otherShowcase)
	db.Create(&models.ShowcaseCoin{ShowcaseID: showcase.ID, CoinID: memberCoin.ID, SortOrder: 0})
	db.Create(&models.ShowcaseCoin{ShowcaseID: showcase.ID, CoinID: otherMemberCoin.ID, SortOrder: 1})
	db.Create(&models.ShowcaseCoin{ShowcaseID: showcase.ID, CoinID: otherOwnerCoin.ID, SortOrder: 2})
	db.Create(&models.ShowcaseCoin{ShowcaseID: otherShowcase.ID, CoinID: otherShowcaseCoin.ID, SortOrder: 0})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/showcase/featured-set", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on public showcase, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Coins []struct {
			Name       string   `json:"name"`
			DiameterMm *float64 `json:"diameterMm"`
			Images     []struct {
				FilePath  string `json:"filePath"`
				ImageType string `json:"imageType"`
				IsPrimary bool   `json:"isPrimary"`
			} `json:"images"`
		} `json:"coins"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Coins) != 2 {
		t.Fatalf("expected 2 member coins, got %d: %s", len(resp.Coins), w.Body.String())
	}
	if resp.Coins[0].Name != "Included Denarius" || resp.Coins[1].Name != "Included Sestertius" {
		t.Fatalf("expected only showcase member coins in sort order, got %#v", resp.Coins)
	}
	for _, coin := range resp.Coins {
		if coin.Name == "Non-member Aureus" {
			t.Fatal("public showcase response included a non-member coin")
		}
		if coin.Name == "Other Showcase Coin" {
			t.Fatal("public showcase response included a coin from a different showcase")
		}
		if coin.Name == "Other Owner Linked Coin" {
			t.Fatal("public showcase response included a coin not owned by the showcase owner")
		}
	}
	if resp.Coins[0].DiameterMm == nil || *resp.Coins[0].DiameterMm != memberDiameter {
		t.Fatalf("expected diameterMm %v for tray sizing, got %#v", memberDiameter, resp.Coins[0].DiameterMm)
	}
	if len(resp.Coins[0].Images) != 2 {
		t.Fatalf("expected member images for tray image selection, got %#v", resp.Coins[0].Images)
	}
	hasPrimaryCard := false
	hasObverse := false
	for _, image := range resp.Coins[0].Images {
		if image.FilePath == "cards/denarius-card.webp" && image.IsPrimary {
			hasPrimaryCard = true
		}
		if image.FilePath == "coins/denarius-obverse.webp" && image.ImageType == string(models.ImageTypeObverse) {
			hasObverse = true
		}
	}
	if !hasPrimaryCard || !hasObverse {
		t.Fatalf("expected isPrimary and imageType fields for tray image selection, got %#v", resp.Coins[0].Images)
	}
}
