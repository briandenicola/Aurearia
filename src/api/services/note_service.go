package services

import (
	"errors"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

const (
	MaxNoteTitleLength = 200
	MaxNoteBodyLength  = 20000
)

var (
	ErrNoteNotFound      = errors.New("note not found")
	ErrNoteTitleRequired = errors.New("note title is required")
	ErrNoteTitleTooLong  = errors.New("note title is too long")
	ErrNoteBodyTooLong   = errors.New("note body is too long")
)

type NoteInput struct {
	Title string
	Body  string
}

type NoteService struct {
	repo *repository.NoteRepository
}

func NewNoteService(repo *repository.NoteRepository) *NoteService {
	return &NoteService{repo: repo}
}

func (s *NoteService) List(userID uint) ([]models.Note, error) {
	return s.repo.List(userID)
}

func (s *NoteService) Get(id, userID uint) (models.Note, error) {
	note, err := s.repo.Get(id, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Note{}, ErrNoteNotFound
	}
	return note, err
}

func (s *NoteService) Create(userID uint, input NoteInput) (models.Note, error) {
	if err := validateNoteInput(input); err != nil {
		return models.Note{}, err
	}
	note := models.Note{
		UserID: userID,
		Title:  strings.TrimSpace(input.Title),
		Body:   input.Body,
	}
	return note, s.repo.Create(&note)
}

func (s *NoteService) Update(id, userID uint, input NoteInput) (models.Note, error) {
	if err := validateNoteInput(input); err != nil {
		return models.Note{}, err
	}
	note, err := s.Get(id, userID)
	if err != nil {
		return models.Note{}, err
	}
	note.Title = strings.TrimSpace(input.Title)
	note.Body = input.Body
	return note, s.repo.Update(&note)
}

func (s *NoteService) Delete(id, userID uint) error {
	rows, err := s.repo.Delete(id, userID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNoteNotFound
	}
	return nil
}

func validateNoteInput(input NoteInput) error {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return ErrNoteTitleRequired
	}
	if len(title) > MaxNoteTitleLength {
		return ErrNoteTitleTooLong
	}
	if len(input.Body) > MaxNoteBodyLength {
		return ErrNoteBodyTooLong
	}
	return nil
}
