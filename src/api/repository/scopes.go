package repository

import "gorm.io/gorm"

// OwnedBy scopes a query to records belonging to the given user.
func OwnedBy(userID uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ?", userID)
	}
}

// OwnedByID scopes a query to a specific record owned by the given user.
func OwnedByID(id uint, userID uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND user_id = ?", id, userID)
	}
}

// ActiveCollection scopes to coins that are not wishlist and not sold.
func ActiveCollection(userID uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ? AND is_wishlist = ? AND is_sold = ?", userID, false, false)
	}
}

// PublicCoins scopes to coins that are visible to other users.
func PublicCoins(userID uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ? AND is_wishlist = false AND is_sold = false AND is_private = false", userID)
	}
}
