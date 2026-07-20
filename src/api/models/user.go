package models

import "time"

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type User struct {
	ID                             uint       `gorm:"primaryKey" json:"id"`
	Username                       string     `gorm:"uniqueIndex;not null" json:"username"`
	Email                          string     `gorm:"uniqueIndex" json:"email"`
	PasswordHash                   string     `gorm:"not null" json:"-"`
	Role                           UserRole   `gorm:"type:varchar(10);default:'user'" json:"role"`
	AvatarPath                     string     `json:"avatarPath"`
	IsPublic                       bool       `gorm:"default:false" json:"isPublic"`
	Bio                            string     `gorm:"type:text" json:"bio"`
	ZipCode                        string     `gorm:"type:varchar(10)" json:"zipCode"`
	NumisBidsUsername              string     `gorm:"type:varchar(100)" json:"-"`
	NumisBidsPassword              string     `gorm:"type:text" json:"-"`
	CNGUsername                    string     `gorm:"type:varchar(100)" json:"-"`
	CNGPassword                    string     `gorm:"type:text" json:"-"`
	PushoverUserKey                string     `gorm:"type:varchar(100)" json:"-"`
	PushoverEnabled                bool       `gorm:"default:false" json:"pushoverEnabled"`
	CoinOfDayEnabled               bool       `gorm:"default:true" json:"coinOfDayEnabled"`
	EmperorTrackerEnabled          bool       `gorm:"default:false" json:"emperorTrackerEnabled"`
	EmperorTrackerShowUsurpers     bool       `gorm:"default:false" json:"emperorTrackerShowUsurpers"`
	EmperorTrackerShowEmpresses    bool       `gorm:"default:false" json:"emperorTrackerShowEmpresses"`
	EmperorTrackerShowOtherFigures bool       `gorm:"default:false" json:"emperorTrackerShowOtherFigures"`
	LockedUntil                    *time.Time `gorm:"index" json:"lockedUntil,omitempty"`
	CreatedAt                      time.Time  `json:"createdAt"`
}
