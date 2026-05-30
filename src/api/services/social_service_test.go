package services

import (
	"strings"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSocialTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Follow{}, &models.Notification{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func TestFollowUser_CreatesFollowRequestNotification(t *testing.T) {
	db := setupSocialTestDB(t)

	socialRepo := repository.NewSocialRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	userRepo := repository.NewUserRepository(db)
	notifSvc := NewNotificationService(notifRepo, socialRepo, userRepo, nil, NewLogger(100))
	svc := NewSocialService(socialRepo, notifSvc)

	follower := models.User{Username: "alice", Email: "alice@example.com", PasswordHash: "hash", IsPublic: true}
	target := models.User{Username: "bob", Email: "bob@example.com", PasswordHash: "hash", IsPublic: true}
	if err := db.Create(&follower).Error; err != nil {
		t.Fatalf("failed to create follower: %v", err)
	}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("failed to create target user: %v", err)
	}

	status, err := svc.FollowUser(follower.ID, target.ID)
	if err != nil {
		t.Fatalf("FollowUser returned error: %v", err)
	}
	if status != "pending" {
		t.Fatalf("expected pending status, got %q", status)
	}

	var follow models.Follow
	if err := db.Where("follower_id = ? AND following_id = ?", follower.ID, target.ID).First(&follow).Error; err != nil {
		t.Fatalf("expected follow row to be created: %v", err)
	}
	if follow.Status != "pending" {
		t.Fatalf("expected follow status pending, got %q", follow.Status)
	}

	var notifications []models.Notification
	if err := db.Where("user_id = ? AND type = ?", target.ID, NotificationTypeFollowRequest).Find(&notifications).Error; err != nil {
		t.Fatalf("failed to fetch notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 follow request notification, got %d", len(notifications))
	}

	n := notifications[0]
	if n.Title != "New follower request" {
		t.Fatalf("expected title %q, got %q", "New follower request", n.Title)
	}
	if !strings.Contains(n.Message, follower.Username) {
		t.Fatalf("expected notification message to include %q, got %q", follower.Username, n.Message)
	}
	if n.ReferenceID != follower.ID {
		t.Fatalf("expected reference ID %d, got %d", follower.ID, n.ReferenceID)
	}
	if n.ReferenceURL != "/followers" {
		t.Fatalf("expected reference URL /followers, got %q", n.ReferenceURL)
	}
}
