package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type NoteRepository struct {
	db *gorm.DB
}

func NewNoteRepository(db *gorm.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) List(userID uint) ([]models.Note, error) {
	var notes []models.Note
	err := r.db.Scopes(OwnedBy(userID)).Order("updated_at DESC, id DESC").Find(&notes).Error
	return notes, err
}

func (r *NoteRepository) Get(id, userID uint) (models.Note, error) {
	var note models.Note
	err := r.db.Scopes(OwnedByID(id, userID)).First(&note).Error
	return note, err
}

func (r *NoteRepository) Create(note *models.Note) error {
	return r.db.Create(note).Error
}

func (r *NoteRepository) Update(note *models.Note) error {
	return r.db.Save(note).Error
}

func (r *NoteRepository) Delete(id, userID uint) (int64, error) {
	result := r.db.Scopes(OwnedByID(id, userID)).Delete(&models.Note{})
	return result.RowsAffected, result.Error
}
