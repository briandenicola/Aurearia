package services

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var noteServiceDBCounter atomic.Uint64

func setupNoteService(t *testing.T) (*NoteService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:note_service_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), noteServiceDBCounter.Add(1))), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Note{}); err != nil {
		t.Fatalf("migrate note: %v", err)
	}
	return NewNoteService(repository.NewNoteRepository(db)), db
}

func TestNoteService_CRUDScopesToUser(t *testing.T) {
	svc, db := setupNoteService(t)
	if err := db.Create(&models.Note{UserID: 2, Title: "Other", Body: "private"}).Error; err != nil {
		t.Fatalf("seed other note: %v", err)
	}

	created, err := svc.Create(1, NoteInput{Title: "  Research  ", Body: "**markdown**"})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if created.UserID != 1 || created.Title != "Research" || created.Body != "**markdown**" {
		t.Fatalf("unexpected created note: %+v", created)
	}

	listed, err := svc.List(1)
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("expected only user 1 note, got %+v", listed)
	}

	if _, err := svc.Get(created.ID, 2); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("cross-user get error = %v, want ErrNoteNotFound", err)
	}
	if _, err := svc.Update(created.ID, 2, NoteInput{Title: "Stolen", Body: "changed"}); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("cross-user update error = %v, want ErrNoteNotFound", err)
	}
	if err := svc.Delete(created.ID, 2); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("cross-user delete error = %v, want ErrNoteNotFound", err)
	}

	updated, err := svc.Update(created.ID, 1, NoteInput{Title: "Updated", Body: "- revised"})
	if err != nil {
		t.Fatalf("owner update: %v", err)
	}
	if updated.Title != "Updated" || updated.Body != "- revised" {
		t.Fatalf("unexpected updated note: %+v", updated)
	}

	if err := svc.Delete(created.ID, 1); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
	if _, err := svc.Get(created.ID, 1); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("get after delete error = %v, want ErrNoteNotFound", err)
	}
}

func TestNoteService_Validation(t *testing.T) {
	svc, _ := setupNoteService(t)
	for _, tc := range []struct {
		name string
		in   NoteInput
		want error
	}{
		{name: "title required", in: NoteInput{Title: " ", Body: "body"}, want: ErrNoteTitleRequired},
		{name: "title too long", in: NoteInput{Title: strings.Repeat("t", MaxNoteTitleLength+1), Body: "body"}, want: ErrNoteTitleTooLong},
		{name: "body too long", in: NoteInput{Title: "Title", Body: strings.Repeat("b", MaxNoteBodyLength+1)}, want: ErrNoteBodyTooLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Create(1, tc.in); !errors.Is(err, tc.want) {
				t.Fatalf("create error = %v, want %v", err, tc.want)
			}
		})
	}

	if _, err := svc.Create(1, NoteInput{Title: strings.Repeat("t", MaxNoteTitleLength), Body: strings.Repeat("b", MaxNoteBodyLength)}); err != nil {
		t.Fatalf("boundary-valid note rejected: %v", err)
	}
}
