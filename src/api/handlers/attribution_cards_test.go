package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
)

func TestAttributionCardsOwnerOnlyAndBytePreserving(t *testing.T) {
	router, db, uploadDir := setupImageMediaRouter(t)
	photoPath := createStoredCoinImage(t, db, uploadDir, 1, false)
	var coin models.Coin
	if err := db.Where("user_id = ?", 1).First(&coin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.User{}).Where("id = ?", 1).Update("is_public", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Follow{FollowerID: 2, FollowingID: 1, Status: "accepted"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.CoinImage{}).Where("file_path = ?", photoPath).Update("is_primary", true).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewImageRepository(db)
	handler := NewImageHandler(uploadDir, repo, services.NewImageService(repo, uploadDir), services.NewLogger(100))
	router.POST("/coins/:id/images", coinTestAuthMiddleware(), handler.Upload)
	router.POST("/coins/:id/images/base64", coinTestAuthMiddleware(), handler.UploadBase64)
	router.GET("/coins/:id/attribution-cards", coinTestAuthMiddleware(), handler.ListAttributionCards)
	router.GET("/showcase/:slug/uploads/*filepath", handler.ServePublicShowcaseUpload)

	var original bytes.Buffer
	if err := png.Encode(&original, image.NewRGBA(image.Rect(0, 0, 12, 20))); err != nil {
		t.Fatal(err)
	}
	upload := func(t *testing.T, format string, userID uint, primary bool, data []byte) *httptest.ResponseRecorder {
		t.Helper()
		if data == nil {
			data = original.Bytes()
		}
		var body bytes.Buffer
		url := fmt.Sprintf("/coins/%d/images", coin.ID)
		contentType := "application/json"
		if format == "multipart" {
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("image", "dealer-card.png")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write(data); err != nil {
				t.Fatal(err)
			}
			for name, value := range map[string]string{"imageType": "card", "isPrimary": fmt.Sprint(primary), "circleClip": "true"} {
				if err := writer.WriteField(name, value); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			contentType = writer.FormDataContentType()
		} else {
			url += "/base64"
			if err := json.NewEncoder(&body).Encode(base64ImageRequest{
				Image: base64.StdEncoding.EncodeToString(data), FileExtension: ".png",
				ImageType: "card", IsPrimary: primary, CircleClip: true,
			}); err != nil {
				t.Fatal(err)
			}
		}
		req := httptest.NewRequest(http.MethodPost, url, &body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", authHeader(userID))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	for _, format := range []string{"multipart", "base64"} {
		t.Run(format, func(t *testing.T) {
			for _, test := range []struct {
				viewer  uint
				primary bool
				status  int
				data    []byte
			}{{2, false, http.StatusNotFound, nil}, {1, true, http.StatusBadRequest, nil}, {1, false, http.StatusBadRequest, []byte("not an image")}, {1, false, http.StatusCreated, nil}} {
				w := upload(t, format, test.viewer, test.primary, test.data)
				if w.Code != test.status {
					t.Fatalf("upload viewer=%d primary=%t: %d %s", test.viewer, test.primary, w.Code, w.Body.String())
				}
				if w.Code != http.StatusCreated {
					continue
				}
				var card models.CoinImage
				if err := json.Unmarshal(w.Body.Bytes(), &card); err != nil {
					t.Fatal(err)
				}
				if card.ImageType != models.ImageTypeCard || card.IsPrimary {
					t.Fatalf("unexpected card metadata: %+v", card)
				}
				stored, err := os.ReadFile(filepath.Join(uploadDir, filepath.FromSlash(card.FilePath)))
				if err != nil || !bytes.Equal(stored, original.Bytes()) {
					t.Fatalf("original bytes changed despite card type and circleClip=true: %v", err)
				}
			}
		})
	}
	cards, err := repo.ListAttributionCards(coin.ID)
	if err != nil || len(cards) != 2 {
		t.Fatalf("expected exactly two successful uploads: %v %+v", err, cards)
	}
	showcase := models.Showcase{UserID: 1, Slug: "cards-private", Title: "Public coins", IsActive: true}
	if err := db.Create(&showcase).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ShowcaseCoin{ShowcaseID: showcase.ID, CoinID: coin.ID}).Error; err != nil {
		t.Fatal(err)
	}
	empty := models.Coin{Name: "No originals", UserID: 1}
	if err := db.Create(&empty).Error; err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name   string
		url    string
		viewer uint
		status int
	}{
		{"owner list", fmt.Sprintf("/coins/%d/attribution-cards", coin.ID), 1, 200},
		{"empty list", fmt.Sprintf("/coins/%d/attribution-cards", empty.ID), 1, 200},
		{"follower list", fmt.Sprintf("/coins/%d/attribution-cards", coin.ID), 2, 404},
		{"anonymous list", fmt.Sprintf("/coins/%d/attribution-cards", coin.ID), 0, 401},
		{"missing coin", "/coins/999999/attribution-cards", 1, 404},
		{"invalid coin", "/coins/zero/attribution-cards", 1, 400},
		{"zero coin", "/coins/0/attribution-cards", 1, 400},
		{"owner original", "/uploads/" + cards[0].FilePath, 1, 200},
		{"follower original", "/uploads/" + cards[0].FilePath, 2, 404},
		{"follower variant", "/uploads/" + cards[0].FilePath + "?size=thumb", 2, 404},
		{"anonymous original", "/uploads/" + cards[0].FilePath, 0, 401},
		{"public original", "/showcase/cards-private/uploads/" + cards[0].FilePath, 0, 404},
		{"follower photo unchanged", "/uploads/" + photoPath, 2, 200},
		{"public photo unchanged", "/showcase/cards-private/uploads/" + photoPath, 0, 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.url, nil)
			if test.viewer != 0 {
				req.Header.Set("Authorization", authHeader(test.viewer))
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != test.status {
				t.Fatalf("expected %d, got %d: %s", test.status, w.Code, w.Body.String())
			}
			if test.name == "owner original" {
				if !bytes.Equal(w.Body.Bytes(), original.Bytes()) || w.Header().Get("Cache-Control") != "private, no-store" {
					t.Fatal("original response must preserve bytes and disallow caching")
				}
			}
			if test.name == "owner list" {
				var listed []models.CoinImage
				if err := json.Unmarshal(w.Body.Bytes(), &listed); err != nil || len(listed) != 2 {
					t.Fatalf("owner list must contain both cards: %v %s", err, w.Body.String())
				}
			}
			if test.name == "empty list" && strings.TrimSpace(w.Body.String()) != "[]" {
				t.Fatalf("empty lists must be arrays, not null: %s", w.Body.String())
			}
		})
	}
	entries, err := os.ReadDir(filepath.Join(uploadDir, fmt.Sprintf("coin-%d", coin.ID)))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "_thumb") || strings.Contains(entry.Name(), "_medium") {
			t.Fatal("private originals do not need generated variants")
		}
	}
	var primary models.CoinImage
	if err := db.Where("coin_id = ? AND is_primary = ?", coin.ID, true).First(&primary).Error; err != nil || primary.FilePath != photoPath {
		t.Fatalf("card uploads changed the primary coin photo: %v %+v", err, primary)
	}
}
