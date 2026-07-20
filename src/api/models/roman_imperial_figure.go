package models

import "time"

// ImperialFigureRole classifies a RomanImperialFigure. Only figures tagged
// ImperialFigureRoleEmperor count toward F028's default completion goal —
// the other roles were, by definition, never accepted as legitimate emperors
// (or, for Julius Caesar, predate the imperial title entirely) and are
// tracked only if a user opts into that category in Settings.
type ImperialFigureRole string

const (
	ImperialFigureRoleEmperor ImperialFigureRole = "emperor"
	ImperialFigureRoleEmpress ImperialFigureRole = "empress"
	ImperialFigureRoleCaesar  ImperialFigureRole = "caesar"
	ImperialFigureRoleUsurper ImperialFigureRole = "usurper"
	ImperialFigureRoleOther   ImperialFigureRole = "other"
)

// ImperialFigureRegion is west for every figure through the unified empire
// (Augustus through Theodosius I) and for the Western line after the 395
// split; east is used only for the definitively separate Eastern line from
// 395 onward (Arcadius and after).
type ImperialFigureRegion string

const (
	ImperialFigureRegionWest ImperialFigureRegion = "west"
	ImperialFigureRegionEast ImperialFigureRegion = "east"
)

// RarityTier is a hand-curated, first-guess relative availability rating
// used to sort the V1 "what to pursue next" suggestions list. Not sourced
// against real auction-frequency data yet (see F028's open questions).
type RarityTier string

const (
	RarityTierCommon   RarityTier = "common"
	RarityTierScarce   RarityTier = "scarce"
	RarityTierRare     RarityTier = "rare"
	RarityTierVeryRare RarityTier = "very_rare"
)

// RomanImperialFigure is a curated, seeded (not user-editable) reference
// entry for a Roman imperial-era figure used by F028's collection tracker.
// ReignStart/ReignEnd use plain calendar years with BC years encoded as
// negative integers (e.g. 27 BC = -27); there is no year-zero adjustment.
type RomanImperialFigure struct {
	ID             uint                 `gorm:"primaryKey" json:"id"`
	Name           string               `gorm:"type:varchar(128);not null" json:"name"`
	NormalizedName string               `gorm:"type:varchar(128);not null;uniqueIndex" json:"-"`
	Aliases        StringList           `gorm:"type:text;not null" json:"aliases"`
	Role           ImperialFigureRole   `gorm:"type:varchar(20);not null;index" json:"role"`
	Region         ImperialFigureRegion `gorm:"type:varchar(10);not null" json:"region"`
	Dynasty        string               `gorm:"type:varchar(128);not null;index" json:"dynasty"`
	ReignStart     int                  `gorm:"not null" json:"reignStart"`
	ReignEnd       int                  `gorm:"not null" json:"reignEnd"`
	SortOrder      int                  `gorm:"not null;index" json:"sortOrder"`
	RarityTier     RarityTier           `gorm:"type:varchar(20);not null" json:"rarityTier"`
	Notes          string               `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}
