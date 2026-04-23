package models

// Tag represents a user-defined label for organizing coins.
type Tag struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"not null;index;uniqueIndex:idx_user_tag_name" json:"userId"`
	Name   string `gorm:"not null;type:varchar(50);uniqueIndex:idx_user_tag_name" json:"name"`
	Color  string `gorm:"type:varchar(7);default:'#6b7280'" json:"color"`
}

// CoinTag is the join table for the many-to-many relationship between coins and tags.
type CoinTag struct {
	CoinID uint `gorm:"primaryKey" json:"coinId"`
	TagID  uint `gorm:"primaryKey" json:"tagId"`
}
