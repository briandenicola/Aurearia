package models

import "time"

type CollectorProfile struct {
	ID                  uint       `gorm:"primaryKey" json:"-"`
	UserID              uint       `gorm:"not null;uniqueIndex" json:"-"`
	User                User       `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	BudgetMin           *float64   `json:"budgetMin"`
	BudgetMax           *float64   `json:"budgetMax"`
	Currency            *string    `gorm:"type:varchar(3)" json:"currency"`
	PreferredPeriods    StringList `gorm:"type:text;not null;default:'[]'" json:"preferredPeriods"`
	PreferredCategories StringList `gorm:"type:text;not null;default:'[]'" json:"preferredCategories"`
	ExcludedCategories  StringList `gorm:"type:text;not null;default:'[]'" json:"excludedCategories"`
	PreferredDealers    StringList `gorm:"type:text;not null;default:'[]'" json:"preferredDealers"`
	CollectingGoals     StringList `gorm:"type:text;not null;default:'[]'" json:"collectingGoals"`
	CreatedAt           time.Time  `json:"-"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}
