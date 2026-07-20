package models

import "time"

// RomanImperialFigureHighlight stores the user's chosen display coin when
// multiple active Roman coins are matched to the same imperial figure.
type RomanImperialFigureHighlight struct {
	ID                    uint                `gorm:"primaryKey" json:"id"`
	UserID                uint                `gorm:"not null;uniqueIndex:idx_figure_highlight_user_figure" json:"userId"`
	RomanImperialFigureID uint                `gorm:"not null;uniqueIndex:idx_figure_highlight_user_figure" json:"romanImperialFigureId"`
	CoinID                uint                `gorm:"not null" json:"coinId"`
	User                  User                `gorm:"foreignKey:UserID" json:"-"`
	RomanImperialFigure   RomanImperialFigure `gorm:"foreignKey:RomanImperialFigureID" json:"-"`
	Coin                  Coin                `gorm:"foreignKey:CoinID" json:"-"`
	CreatedAt             time.Time           `json:"createdAt"`
	UpdatedAt             time.Time           `json:"updatedAt"`
}
