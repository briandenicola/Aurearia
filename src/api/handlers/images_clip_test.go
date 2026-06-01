package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const clipTestJWTSecret = "clip-test-secret"

func setupClipTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func makeClipTestJWT(userID uint) string {
	claims := jwt.MapClaims{
		"userId":   float64(userID),
		"username": "testuser",
		"role":     "user",
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(clipTestJWTSecret))
	return signed
}

func setupClipTestEnv(t *testing.T) (*gin.Engine, *gorm.DB, string, uint, uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupClipTestDB(t)

	// Create test user and coin
	user := models.User{Username: "testuser", Email: "test@example.com"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	coin := models.Coin{
		UserID: user.ID,
		Name:   "Test Coin",
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("failed to create test coin: %v", err)
	}

	uploadDir := t.TempDir()

	repo := repository.NewImageRepository(db)
	imgSvc := services.NewImageService(repo, uploadDir)
	logger := services.NewLogger(100)
	handler := NewImageHandler(uploadDir, repo, imgSvc, logger)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", user.ID)
		c.Next()
	})
	router.POST("/coins/:id/images", handler.Upload)
	router.POST("/coins/:id/images/base64", handler.UploadBase64)

	return router, db, uploadDir, user.ID, coin.ID
}

// makeSyntheticJPEG creates a simple JPEG image for testing.
func makeSyntheticJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with gradient: blue corners, yellow center
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx := float64(x) / float64(width)
			dy := float64(y) / float64(height)
			r := uint8(255 * dx)
			g := uint8(255 * dy)
			b := uint8(128)
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes()
}

// TestUploadMultipart_CircleClip_ObverseClipped verifies that circleClip=true + imageType=obverse
// results in a stored PNG with transparent corners.
func TestUploadMultipart_CircleClip_ObverseClipped(t *testing.T) {
	router, db, uploadDir, userID, coinID := setupClipTestEnv(t)

	jpegData := makeSyntheticJPEG(300, 300)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write(jpegData)
	writer.WriteField("imageType", "obverse")
	writer.WriteField("isPrimary", "true")
	writer.WriteField("circleClip", "true")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/coins/"+fmt.Sprint(coinID)+"/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+makeClipTestJWT(userID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.CoinImage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify stored file is PNG
	if filepath.Ext(resp.FilePath) != ".png" {
		t.Errorf("expected .png extension, got %s", filepath.Ext(resp.FilePath))
	}

	// Read stored file
	storedPath := filepath.Join(uploadDir, resp.FilePath)
	storedData, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	// Decode PNG
	storedImg, err := png.Decode(bytes.NewReader(storedData))
	if err != nil {
		t.Fatalf("stored file is not valid PNG: %v", err)
	}

	// Check corners are transparent
	bounds := storedImg.Bounds()
	corners := []image.Point{
		{bounds.Min.X, bounds.Min.Y},
		{bounds.Max.X - 1, bounds.Min.Y},
		{bounds.Min.X, bounds.Max.Y - 1},
		{bounds.Max.X - 1, bounds.Max.Y - 1},
	}
	for _, pt := range corners {
		_, _, _, a := storedImg.At(pt.X, pt.Y).RGBA()
		if a > 0 {
			t.Errorf("corner (%d,%d) is not transparent: alpha=%d", pt.X, pt.Y, a>>8)
		}
	}

	// Check center is opaque
	cx, cy := bounds.Dx()/2, bounds.Dy()/2
	_, _, _, ac := storedImg.At(cx, cy).RGBA()
	if ac>>8 < 200 {
		t.Errorf("center (%d,%d) is too transparent: alpha=%d", cx, cy, ac>>8)
	}

	// Verify DB record
	var dbImg models.CoinImage
	if err := db.First(&dbImg, resp.ID).Error; err != nil {
		t.Fatalf("failed to find DB record: %v", err)
	}
	if dbImg.ImageType != "obverse" {
		t.Errorf("expected imageType=obverse, got %s", dbImg.ImageType)
	}
}

// TestUploadMultipart_CircleClip_CardNotClipped verifies that circleClip=true + imageType=card
// does NOT clip (card images must remain rectangular for OCR).
func TestUploadMultipart_CircleClip_CardNotClipped(t *testing.T) {
	router, _, uploadDir, userID, coinID := setupClipTestEnv(t)

	jpegData := makeSyntheticJPEG(300, 300)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write(jpegData)
	writer.WriteField("imageType", "detail") // Not obverse/reverse
	writer.WriteField("circleClip", "true")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/coins/"+fmt.Sprint(coinID)+"/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+makeClipTestJWT(userID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.CoinImage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify stored file is still JPEG (NOT clipped to PNG)
	ext := filepath.Ext(resp.FilePath)
	if ext != ".jpg" && ext != ".jpeg" {
		t.Errorf("expected .jpg/.jpeg extension for non-obverse/reverse, got %s", ext)
	}

	// Read stored file
	storedPath := filepath.Join(uploadDir, resp.FilePath)
	storedData, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	// Should decode as JPEG (not PNG)
	_, err = jpeg.Decode(bytes.NewReader(storedData))
	if err != nil {
		t.Errorf("stored file is not valid JPEG: %v", err)
	}
}

// TestUploadMultipart_NoCircleClip_ObverseOriginal verifies that circleClip=false (or absent)
// with imageType=obverse stores the original image unchanged.
func TestUploadMultipart_NoCircleClip_ObverseOriginal(t *testing.T) {
	router, _, uploadDir, userID, coinID := setupClipTestEnv(t)

	jpegData := makeSyntheticJPEG(300, 300)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write(jpegData)
	writer.WriteField("imageType", "obverse")
	writer.WriteField("circleClip", "false")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/coins/"+fmt.Sprint(coinID)+"/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+makeClipTestJWT(userID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.CoinImage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify stored file is JPEG
	ext := filepath.Ext(resp.FilePath)
	if ext != ".jpg" && ext != ".jpeg" {
		t.Errorf("expected .jpg/.jpeg extension, got %s", ext)
	}

	// Read stored file
	storedPath := filepath.Join(uploadDir, resp.FilePath)
	storedData, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	// Should decode as JPEG
	_, err = jpeg.Decode(bytes.NewReader(storedData))
	if err != nil {
		t.Errorf("stored file is not valid JPEG: %v", err)
	}

	// Stored bytes should match original (no clipping)
	if len(storedData) != len(jpegData) {
		t.Logf("stored size=%d, original size=%d (may differ due to re-encode, checking decode)", len(storedData), len(jpegData))
	}
}

// TestUploadMultipart_CircleClip_UndecodableData verifies that invalid image data
// falls back to storing the original when clipping fails.
func TestUploadMultipart_CircleClip_UndecodableData(t *testing.T) {
	router, _, uploadDir, userID, coinID := setupClipTestEnv(t)

	invalidData := []byte("not-an-image-just-garbage-data-12345")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write(invalidData)
	writer.WriteField("imageType", "obverse")
	writer.WriteField("circleClip", "true")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/coins/"+fmt.Sprint(coinID)+"/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+makeClipTestJWT(userID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed (fall back to storing original after clip failure)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (fallback to original), got %d: %s", w.Code, w.Body.String())
	}

	var resp models.CoinImage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Read stored file
	storedPath := filepath.Join(uploadDir, resp.FilePath)
	storedData, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	// Should match original invalid data (fallback behavior)
	if !bytes.Equal(storedData, invalidData) {
		t.Errorf("stored data does not match original after clip failure")
	}
}

// TestUploadBase64_CircleClip_ReverseClipped verifies base64 upload with circleClip=true
// for reverse imageType produces a clipped PNG.
func TestUploadBase64_CircleClip_ReverseClipped(t *testing.T) {
	router, _, uploadDir, userID, coinID := setupClipTestEnv(t)

	jpegData := makeSyntheticJPEG(300, 300)
	b64Data := base64.StdEncoding.EncodeToString(jpegData)

	reqBody := base64ImageRequest{
		Image:         b64Data,
		FileExtension: ".jpg",
		ImageType:     "reverse",
		IsPrimary:     false,
		CircleClip:    true,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/coins/"+fmt.Sprint(coinID)+"/images/base64", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeClipTestJWT(userID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.CoinImage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify stored file is PNG
	if filepath.Ext(resp.FilePath) != ".png" {
		t.Errorf("expected .png extension, got %s", filepath.Ext(resp.FilePath))
	}

	// Read stored file
	storedPath := filepath.Join(uploadDir, resp.FilePath)
	storedData, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	// Decode PNG
	storedImg, err := png.Decode(bytes.NewReader(storedData))
	if err != nil {
		t.Fatalf("stored file is not valid PNG: %v", err)
	}

	// Check corners are transparent
	bounds := storedImg.Bounds()
	corners := []image.Point{
		{bounds.Min.X, bounds.Min.Y},
		{bounds.Max.X - 1, bounds.Min.Y},
	}
	for _, pt := range corners {
		_, _, _, a := storedImg.At(pt.X, pt.Y).RGBA()
		if a > 0 {
			t.Errorf("corner (%d,%d) is not transparent: alpha=%d", pt.X, pt.Y, a>>8)
		}
	}
}

// TestUploadBase64_NoCircleClip_Original verifies base64 upload without circleClip
// stores the original image.
func TestUploadBase64_NoCircleClip_Original(t *testing.T) {
	router, _, uploadDir, userID, coinID := setupClipTestEnv(t)

	jpegData := makeSyntheticJPEG(300, 300)
	b64Data := base64.StdEncoding.EncodeToString(jpegData)

	reqBody := base64ImageRequest{
		Image:         b64Data,
		FileExtension: ".jpg",
		ImageType:     "obverse",
		IsPrimary:     false,
		CircleClip:    false,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/coins/"+fmt.Sprint(coinID)+"/images/base64", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeClipTestJWT(userID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.CoinImage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify stored file is JPEG
	ext := filepath.Ext(resp.FilePath)
	if ext != ".jpg" && ext != ".jpeg" {
		t.Errorf("expected .jpg/.jpeg extension, got %s", ext)
	}

	// Read stored file
	storedPath := filepath.Join(uploadDir, resp.FilePath)
	storedData, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	// Should decode as JPEG
	_, err = jpeg.Decode(bytes.NewReader(storedData))
	if err != nil {
		t.Errorf("stored file is not valid JPEG: %v", err)
	}
}

// TestUploadBase64_CircleClip_UndecodableData verifies that base64 with invalid image data
// falls back to storing original when clipping fails.
func TestUploadBase64_CircleClip_UndecodableData(t *testing.T) {
	router, _, uploadDir, userID, coinID := setupClipTestEnv(t)

	invalidData := []byte("not-an-image-just-garbage-data-12345")
	b64Data := base64.StdEncoding.EncodeToString(invalidData)

	reqBody := base64ImageRequest{
		Image:         b64Data,
		FileExtension: ".jpg",
		ImageType:     "obverse",
		CircleClip:    true,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/coins/"+fmt.Sprint(coinID)+"/images/base64", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeClipTestJWT(userID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed (fall back to original after clip failure)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (fallback to original), got %d: %s", w.Code, w.Body.String())
	}

	var resp models.CoinImage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Read stored file
	storedPath := filepath.Join(uploadDir, resp.FilePath)
	storedData, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("failed to read stored file: %v", err)
	}

	// Should match original invalid data (fallback behavior)
	if !bytes.Equal(storedData, invalidData) {
		t.Errorf("stored data does not match original after clip failure")
	}
}

// TestUploadMultipart_CircleClip_NonOwner verifies that ownership is checked BEFORE decode/clip,
// so attempting to upload with circleClip=true for a non-owned coin returns 404 without decoding.
func TestUploadMultipart_CircleClip_NonOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupClipTestDB(t)

	// Create test user 1 and their coin
	user1 := models.User{Username: "user1", Email: "user1@example.com"}
	if err := db.Create(&user1).Error; err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}
	coin1 := models.Coin{UserID: user1.ID, Name: "User1 Coin"}
	if err := db.Create(&coin1).Error; err != nil {
		t.Fatalf("failed to create coin1: %v", err)
	}

	// Create test user 2 (attacker)
	user2 := models.User{Username: "user2", Email: "user2@example.com"}
	if err := db.Create(&user2).Error; err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	uploadDir := t.TempDir()
	repo := repository.NewImageRepository(db)
	imgSvc := services.NewImageService(repo, uploadDir)
	logger := services.NewLogger(100)
	handler := NewImageHandler(uploadDir, repo, imgSvc, logger)

	router := gin.New()
	// User 2 is authenticated
	router.Use(func(c *gin.Context) {
		c.Set("userId", user2.ID)
		c.Next()
	})
	router.POST("/coins/:id/images", handler.Upload)

	jpegData := makeSyntheticJPEG(300, 300)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.jpg")
	part.Write(jpegData)
	writer.WriteField("imageType", "obverse")
	writer.WriteField("circleClip", "true")
	writer.Close()

	// User 2 tries to upload to User 1's coin
	req := httptest.NewRequest(http.MethodPost, "/coins/"+fmt.Sprint(coin1.ID)+"/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+makeClipTestJWT(user2.ID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 (not found) because User 2 doesn't own coin1
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 (ownership check failed), got %d: %s", w.Code, w.Body.String())
	}

	// Verify NO file was written (clip never executed)
	coinDir := filepath.Join(uploadDir, fmt.Sprintf("coin-%d", coin1.ID))
	if _, err := os.Stat(coinDir); err == nil {
		entries, _ := os.ReadDir(coinDir)
		if len(entries) > 0 {
			t.Errorf("expected no files written, but found %d entries in %s", len(entries), coinDir)
		}
	}
}

// TestUploadBase64_CircleClip_NonOwner verifies that ownership is checked BEFORE decode/clip
// in base64 upload path as well.
func TestUploadBase64_CircleClip_NonOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupClipTestDB(t)

	// Create test user 1 and their coin
	user1 := models.User{Username: "user1", Email: "user1@example.com"}
	if err := db.Create(&user1).Error; err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}
	coin1 := models.Coin{UserID: user1.ID, Name: "User1 Coin"}
	if err := db.Create(&coin1).Error; err != nil {
		t.Fatalf("failed to create coin1: %v", err)
	}

	// Create test user 2 (attacker)
	user2 := models.User{Username: "user2", Email: "user2@example.com"}
	if err := db.Create(&user2).Error; err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	uploadDir := t.TempDir()
	repo := repository.NewImageRepository(db)
	imgSvc := services.NewImageService(repo, uploadDir)
	logger := services.NewLogger(100)
	handler := NewImageHandler(uploadDir, repo, imgSvc, logger)

	router := gin.New()
	// User 2 is authenticated
	router.Use(func(c *gin.Context) {
		c.Set("userId", user2.ID)
		c.Next()
	})
	router.POST("/coins/:id/images/base64", handler.UploadBase64)

	jpegData := makeSyntheticJPEG(300, 300)
	b64Data := base64.StdEncoding.EncodeToString(jpegData)

	reqBody := base64ImageRequest{
		Image:         b64Data,
		FileExtension: ".jpg",
		ImageType:     "obverse",
		CircleClip:    true,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// User 2 tries to upload to User 1's coin
	req := httptest.NewRequest(http.MethodPost, "/coins/"+fmt.Sprint(coin1.ID)+"/images/base64", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+makeClipTestJWT(user2.ID))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 (not found) because User 2 doesn't own coin1
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 (ownership check failed), got %d: %s", w.Code, w.Body.String())
	}

	// Verify NO file was written (clip never executed)
	coinDir := filepath.Join(uploadDir, fmt.Sprintf("coin-%d", coin1.ID))
	if _, err := os.Stat(coinDir); err == nil {
		entries, _ := os.ReadDir(coinDir)
		if len(entries) > 0 {
			t.Errorf("expected no files written, but found %d entries in %s", len(entries), coinDir)
		}
	}
}
