package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// CoinSetType defines the type of coin set.
type CoinSetType string

const (
	CoinSetTypeOpen    CoinSetType = "open"
	CoinSetTypeDefined CoinSetType = "defined"
	CoinSetTypeSmart   CoinSetType = "smart"
	CoinSetTypeGoal    CoinSetType = "goal"
)

// CoinSet represents a user-owned set evolved from tags.
type CoinSet struct {
	ID                   uint        `gorm:"primaryKey" json:"id"`
	UserID               uint        `gorm:"not null;index;uniqueIndex:idx_user_set_name" json:"userId"`
	Name                 string      `gorm:"not null;type:varchar(80);uniqueIndex:idx_user_set_name" json:"name"`
	Description          string      `gorm:"type:text" json:"description"`
	Color                string      `gorm:"type:varchar(7);default:'#6b7280'" json:"color"`
	Icon                 string      `gorm:"type:varchar(50)" json:"icon"`
	SetType              CoinSetType `gorm:"type:varchar(20);not null;default:'open'" json:"setType"`
	ParentSetID          *uint       `gorm:"index" json:"parentSetId"`
	TargetCompletionDate *time.Time  `json:"targetCompletionDate"`
	IsPublic             bool        `gorm:"default:false" json:"isPublic"`
	ShareToken           *string     `gorm:"type:varchar(64);uniqueIndex" json:"shareToken"`
	SmartCriteria        *JSONObject `gorm:"type:text" json:"smartCriteria"`
	CreatedAt            time.Time   `json:"createdAt"`
	UpdatedAt            time.Time   `json:"updatedAt"`
}

// CoinSetMembership represents manual membership for open, defined, and goal sets.
type CoinSetMembership struct {
	SetID     uint      `gorm:"primaryKey" json:"setId"`
	CoinID    uint      `gorm:"primaryKey" json:"coinId"`
	AddedAt   time.Time `gorm:"not null" json:"addedAt"`
	SortOrder int       `gorm:"not null;default:0" json:"sortOrder"`
	Notes     string    `gorm:"type:text" json:"notes"`
}

// CoinSetTarget defines expected coins for completion tracking.
type CoinSetTarget struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	SetID        uint        `gorm:"not null;index" json:"setId"`
	Label        string      `gorm:"not null" json:"label"`
	Year         *int        `json:"year"`
	MintMark     *string     `gorm:"type:varchar(20)" json:"mintMark"`
	Denomination *string     `gorm:"type:varchar(200)" json:"denomination"`
	Country      *string     `gorm:"type:varchar(100)" json:"country"`
	Material     *string     `gorm:"type:varchar(50)" json:"material"`
	MatchRules   *JSONObject `gorm:"type:text" json:"matchRules"`
	SortOrder    int         `gorm:"not null" json:"sortOrder"`
	CreatedAt    time.Time   `json:"createdAt"`
}

// CoinSetValuationSnapshot captures aggregate time-series data for trend tracking.
type CoinSetValuationSnapshot struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	SetID                uint      `gorm:"not null;index;uniqueIndex:idx_set_snapshot_date" json:"setId"`
	UserID               uint      `gorm:"not null;index" json:"userId"`
	SnapshotDate         time.Time `gorm:"not null;uniqueIndex:idx_set_snapshot_date" json:"snapshotDate"`
	TotalValue           float64   `gorm:"not null;default:0" json:"totalValue"`
	TotalInvested        float64   `gorm:"not null;default:0" json:"totalInvested"`
	CoinCount            int       `gorm:"not null" json:"coinCount"`
	CompletionPercentage *float64  `json:"completionPercentage"`
	AvgValuePerCoin      *float64  `json:"avgValuePerCoin"`
	HighestValueCoinID   *uint     `json:"highestValueCoinId"`
	CreatedAt            time.Time `json:"createdAt"`
}

// CoinSetMilestoneAlert tracks alerts for set milestones.
type CoinSetMilestoneAlert struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	SetID           uint       `gorm:"not null;index" json:"setId"`
	UserID          uint       `gorm:"not null;index" json:"userId"`
	Metric          string     `gorm:"type:varchar(50);not null" json:"metric"`
	Threshold       float64    `gorm:"not null" json:"threshold"`
	Direction       string     `gorm:"type:varchar(20);not null" json:"direction"`
	LastTriggeredAt *time.Time `json:"lastTriggeredAt"`
	Enabled         bool       `gorm:"default:true" json:"enabled"`
}

// JSONObject is a custom type for storing JSON in GORM
type JSONObject map[string]interface{}

// Scan implements the sql.Scanner interface for JSONObject
func (j *JSONObject) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	result := make(map[string]interface{})
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

// Value implements the driver.Valuer interface for JSONObject
func (j JSONObject) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}
