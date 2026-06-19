package repository

import (
	"errors"

	"gorm.io/gorm"
)

// IsRecordNotFound reports whether err represents a missing database row.
func IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
