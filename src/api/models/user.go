package models

import "time"

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type User struct {
	ID                             uint     `gorm:"primaryKey" json:"id"`
	Username                       string   `gorm:"uniqueIndex;not null" json:"username"`
	Email                          string   `gorm:"uniqueIndex" json:"email"`
	PasswordHash                   string   `gorm:"not null" json:"-"`
	Role                           UserRole `gorm:"type:varchar(10);default:'user'" json:"role"`
	AvatarPath                     string   `json:"avatarPath"`
	IsPublic                       bool     `gorm:"default:false" json:"isPublic"`
	Bio                            string   `gorm:"type:text" json:"bio"`
	ZipCode                        string   `gorm:"type:varchar(10)" json:"zipCode"`
	NumisBidsUsername              string   `gorm:"type:varchar(100)" json:"-"`
	NumisBidsPassword              string   `gorm:"type:text" json:"-"`
	CNGUsername                    string   `gorm:"type:varchar(100)" json:"-"`
	CNGPassword                    string   `gorm:"type:text" json:"-"`
	ParcelAppAPIKey                string   `gorm:"type:text" json:"-"`
	PushoverUserKey                string   `gorm:"type:varchar(100)" json:"-"`
	PushoverEnabled                bool     `gorm:"default:false" json:"pushoverEnabled"`
	CoinOfDayEnabled               bool     `gorm:"default:true" json:"coinOfDayEnabled"`
	CoinOfDayIncludeWishlist       bool     `gorm:"not null;default:true" json:"coinOfDayIncludeWishlist"`
	EmperorTrackerEnabled          bool     `gorm:"default:false" json:"emperorTrackerEnabled"`
	EmperorTrackerShowUsurpers     bool     `gorm:"default:false" json:"emperorTrackerShowUsurpers"`
	EmperorTrackerShowEmpresses    bool     `gorm:"default:false" json:"emperorTrackerShowEmpresses"`
	EmperorTrackerShowOtherFigures bool     `gorm:"default:false" json:"emperorTrackerShowOtherFigures"`
	PWASwipeNavEnabled             bool     `gorm:"column:pwa_swipe_nav_enabled;not null;default:false" json:"pwaSwipeNavEnabled"`
	// Display preferences synced across the user's devices. Empty means "not
	// set" so each device falls back to its own local default.
	PrefTheme         string     `gorm:"column:pref_theme;type:varchar(32);not null;default:''" json:"-"`
	PrefTimezone      string     `gorm:"column:pref_timezone;type:varchar(64);not null;default:''" json:"-"`
	PrefDefaultView   string     `gorm:"column:pref_default_view;type:varchar(16);not null;default:''" json:"-"`
	PrefDefaultSort   string     `gorm:"column:pref_default_sort;type:varchar(32);not null;default:''" json:"-"`
	PrefTrayFeltColor string     `gorm:"column:pref_tray_felt_color;type:varchar(16);not null;default:''" json:"-"`
	LockedUntil       *time.Time `gorm:"index" json:"lockedUntil,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
}
