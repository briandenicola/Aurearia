package repository

import (
	"errors"

	"gorm.io/gorm"
)

// ErrRecordNotFound is the repository-layer sentinel for missing rows.
var ErrRecordNotFound = gorm.ErrRecordNotFound

// IsRecordNotFound reports whether err represents a missing database row.
func IsRecordNotFound(err error) bool {
	return errors.Is(err, ErrRecordNotFound)
}
