package repository

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupWishlistSearchAlertRepository(t *testing.T) (*WishlistSearchAlertRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.WishlistSearchAlert{}, &models.AlertRun{}, &models.AlertCandidate{}, &models.CandidateProvenance{}, &models.CandidateReviewAction{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewWishlistSearchAlertRepository(db), db
}

func TestWishlistSearchAlertRepository_OwnerScopedAlertCRUD(t *testing.T) {
	repo, db := setupWishlistSearchAlertRepository(t)
	alert := &models.WishlistSearchAlert{UserID: 1, Name: "Owner alert", RulerOrIssuer: "Domitian", Cadence: models.WishlistAlertCadenceManual, IsActive: true}
	if err := repo.CreateAlert(alert); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := repo.GetAlert(alert.ID, 2); !IsRecordNotFound(err) {
		t.Fatalf("non-owner get error = %v", err)
	}
	list, total, err := repo.ListAlerts(1, WishlistSearchAlertFilters{Page: 1, Limit: 20})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("owner list len=%d total=%d err=%v", len(list), total, err)
	}
	active := false
	if _, total, err := repo.ListAlerts(1, WishlistSearchAlertFilters{Active: &active, Page: 1, Limit: 20}); err != nil || total != 0 {
		t.Fatalf("inactive filter total=%d err=%v", total, err)
	}
	alert.Name = "Updated"
	if err := repo.UpdateAlert(alert); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := repo.DeleteAlert(alert.ID, 2); err != nil {
		t.Fatalf("non-owner delete should be generic/no-op: %v", err)
	}
	if _, err := repo.GetAlert(alert.ID, 1); err != nil {
		t.Fatalf("non-owner delete removed alert: %v", err)
	}
	if err := repo.DeleteAlert(alert.ID, 1); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
	if _, err := repo.GetAlert(alert.ID, 1); !IsRecordNotFound(err) {
		t.Fatalf("deleted get error = %v", err)
	}
	var deleted models.WishlistSearchAlert
	if err := db.First(&deleted, alert.ID).Error; err != nil {
		t.Fatalf("soft-deleted alert was not preserved: %v", err)
	}
	if deleted.DeletedAt == nil {
		t.Fatalf("deleted alert missing soft-delete timestamp: %+v", deleted)
	}
}
