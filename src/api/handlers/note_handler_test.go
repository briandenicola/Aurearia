package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupNoteHandlerRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := setupCoinHandlerTestDB(t)
	repo := repository.NewNoteRepository(db)
	handler := NewNoteHandler(services.NewNoteService(repo))

	r := gin.New()
	protected := r.Group("/api")
	protected.Use(coinTestAuthMiddleware())
	protected.GET("/notes", handler.List)
	protected.POST("/notes", handler.Create)
	protected.GET("/notes/:id", handler.Get)
	protected.PUT("/notes/:id", handler.Update)
	protected.DELETE("/notes/:id", handler.Delete)
	return r, db
}

func noteJSON(title, body string) *bytes.Reader {
	b, _ := json.Marshal(map[string]string{"title": title, "body": body})
	return bytes.NewReader(b)
}

func sendNoteRequest(router *gin.Engine, method, path string, userID uint, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", authHeader(userID))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestNoteHandler_UserCanCreateListReadUpdateDeleteOwnNotes(t *testing.T) {
	router, db := setupNoteHandlerRouter(t)
	createTestUser(t, db, 1, "notesuser")

	w := sendNoteRequest(router, http.MethodPost, "/api/notes", 1, noteJSON("Research", "**RIC II** attribution"))
	if w.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created models.Note
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created note: %v", err)
	}
	if created.ID == 0 || created.UserID != 1 || created.Title != "Research" || created.Body != "**RIC II** attribution" {
		t.Fatalf("unexpected created note: %+v", created)
	}

	w = sendNoteRequest(router, http.MethodGet, "/api/notes", 1, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Notes []models.Note `json:"notes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode listed notes: %v", err)
	}
	if len(resp.Notes) != 1 || resp.Notes[0].ID != created.ID {
		t.Fatalf("expected one own note, got %+v", resp.Notes)
	}

	w = sendNoteRequest(router, http.MethodGet, "/api/notes/"+uintString(created.ID), 1, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("read expected 200, got %d: %s", w.Code, w.Body.String())
	}

	w = sendNoteRequest(router, http.MethodPut, "/api/notes/"+uintString(created.ID), 1, noteJSON("Updated research", "- revised"))
	if w.Code != http.StatusOK {
		t.Fatalf("update expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var updated models.Note
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated note: %v", err)
	}
	if updated.Title != "Updated research" || updated.Body != "- revised" {
		t.Fatalf("unexpected updated note: %+v", updated)
	}

	w = sendNoteRequest(router, http.MethodDelete, "/api/notes/"+uintString(created.ID), 1, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete expected 204, got %d: %s", w.Code, w.Body.String())
	}
	w = sendNoteRequest(router, http.MethodGet, "/api/notes/"+uintString(created.ID), 1, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("read after delete expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNoteHandler_OtherUserCannotReadUpdateDeleteNote(t *testing.T) {
	router, db := setupNoteHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	createTestUser(t, db, 2, "intruder")
	note := models.Note{UserID: 1, Title: "Private", Body: "owner only"}
	if err := db.Create(&note).Error; err != nil {
		t.Fatalf("seed note: %v", err)
	}

	for _, tc := range []struct {
		name   string
		method string
		body   io.Reader
	}{
		{name: "read", method: http.MethodGet},
		{name: "update", method: http.MethodPut, body: noteJSON("Stolen", "changed")},
		{name: "delete", method: http.MethodDelete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := sendNoteRequest(router, tc.method, "/api/notes/"+uintString(note.ID), 2, tc.body)
			if w.Code != http.StatusNotFound {
				t.Fatalf("expected cross-user %s to return 404, got %d: %s", tc.name, w.Code, w.Body.String())
			}
		})
	}

	var found models.Note
	if err := db.First(&found, note.ID).Error; err != nil {
		t.Fatalf("owner note should remain: %v", err)
	}
	if found.Title != "Private" || found.Body != "owner only" {
		t.Fatalf("cross-user update changed note: %+v", found)
	}
}

func TestNoteHandler_ListOnlyOwnNotes(t *testing.T) {
	router, db := setupNoteHandlerRouter(t)
	createTestUser(t, db, 1, "owner")
	createTestUser(t, db, 2, "other")
	db.Create(&models.Note{UserID: 1, Title: "Mine", Body: "one"})
	db.Create(&models.Note{UserID: 2, Title: "Theirs", Body: "two"})

	w := sendNoteRequest(router, http.MethodGet, "/api/notes", 1, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Notes []models.Note `json:"notes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode notes: %v", err)
	}
	if len(resp.Notes) != 1 || resp.Notes[0].Title != "Mine" {
		t.Fatalf("expected only owner note, got %+v", resp.Notes)
	}
}

func TestNoteHandler_Validation(t *testing.T) {
	router, db := setupNoteHandlerRouter(t)
	createTestUser(t, db, 1, "validator")

	for _, tc := range []struct {
		name  string
		title string
		body  string
	}{
		{name: "missing title", title: "", body: "body"},
		{name: "blank title", title: "   ", body: "body"},
		{name: "title too long", title: strings.Repeat("a", services.MaxNoteTitleLength+1), body: "body"},
		{name: "body too long", title: "Title", body: strings.Repeat("b", services.MaxNoteBodyLength+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := sendNoteRequest(router, http.MethodPost, "/api/notes", 1, noteJSON(tc.title, tc.body))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}

	w := sendNoteRequest(router, http.MethodPost, "/api/notes", 1, noteJSON(strings.Repeat("t", services.MaxNoteTitleLength), strings.Repeat("b", services.MaxNoteBodyLength)))
	if w.Code != http.StatusCreated {
		t.Fatalf("boundary-valid note expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func uintString(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
