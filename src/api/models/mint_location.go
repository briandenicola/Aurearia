package models

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"
	"unicode"
)

// StringList stores a JSON array of strings in SQLite.
type StringList []string

// StringMap stores a JSON object of strings in SQLite.
type StringMap map[string]string

// Value serializes the string list for database storage.
func (s StringList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	data, err := json.Marshal([]string(s))
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// Scan deserializes a string list from database storage.
func (s *StringList) Scan(value interface{}) error {
	if value == nil {
		*s = StringList{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		*s = StringList{}
		return nil
	}

	if len(data) == 0 {
		*s = StringList{}
		return nil
	}

	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	*s = StringList(values)
	return nil
}

// Value serializes the string map for database storage.
func (s StringMap) Value() (driver.Value, error) {
	if s == nil {
		return "{}", nil
	}
	data, err := json.Marshal(map[string]string(s))
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// Scan deserializes a string map from database storage.
func (s *StringMap) Scan(value interface{}) error {
	if value == nil {
		*s = StringMap{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		*s = StringMap{}
		return nil
	}

	if len(data) == 0 {
		*s = StringMap{}
		return nil
	}

	values := map[string]string{}
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	*s = StringMap(values)
	return nil
}

// NormalizeMintLocationName canonicalizes mint names for case/punctuation-insensitive matching.
func NormalizeMintLocationName(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// MintLocation represents a global admin-managed mint coordinate reference.
type MintLocation struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	DisplayName    string     `gorm:"type:varchar(128);not null" json:"displayName"`
	NormalizedName string     `gorm:"type:varchar(128);not null;uniqueIndex" json:"-"`
	Lat            float64    `gorm:"not null" json:"lat"`
	Lng            float64    `gorm:"not null" json:"lng"`
	Region         string     `gorm:"type:varchar(128)" json:"region,omitempty"`
	Aliases        StringList `gorm:"type:text;not null" json:"aliases"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}
