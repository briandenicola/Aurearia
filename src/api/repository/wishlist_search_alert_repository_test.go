package repository

import (
	"testing"
	"time"

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

// TestWishlistSearchAlertRepository_GetDueAlerts covers the scheduling gap from
// issue #483: a per-alert Cadence was stored but nothing ever checked it against
// LastRunAt to decide whether an alert is due for an automatic run.
func TestWishlistSearchAlertRepository_GetDueAlerts(t *testing.T) {
	repo, _ := setupWishlistSearchAlertRepository(t)
	now := time.Now()

	eightDaysAgo := now.Add(-8 * 24 * time.Hour)
	twoDaysAgo := now.Add(-2 * 24 * time.Hour)
	twentyFiveHoursAgo := now.Add(-25 * time.Hour)
	tenDaysAgo := now.Add(-10 * 24 * time.Hour)

	alerts := []*models.WishlistSearchAlert{
		{UserID: 1, Name: "manual, never runs automatically", Cadence: models.WishlistAlertCadenceManual, IsActive: true, LastRunAt: &tenDaysAgo},
		{UserID: 1, Name: "weekly, overdue", Cadence: models.WishlistAlertCadenceWeekly, IsActive: true, LastRunAt: &eightDaysAgo},
		{UserID: 1, Name: "weekly, not yet due", Cadence: models.WishlistAlertCadenceWeekly, IsActive: true, LastRunAt: &twoDaysAgo},
		{UserID: 1, Name: "weekly, never run", Cadence: models.WishlistAlertCadenceWeekly, IsActive: true, LastRunAt: nil},
		{UserID: 1, Name: "daily, overdue", Cadence: models.WishlistAlertCadenceDaily, IsActive: true, LastRunAt: &twentyFiveHoursAgo},
		{UserID: 1, Name: "monthly, not yet due", Cadence: models.WishlistAlertCadenceMonthly, IsActive: true, LastRunAt: &tenDaysAgo},
		{UserID: 1, Name: "weekly but inactive", Cadence: models.WishlistAlertCadenceWeekly, IsActive: true, LastRunAt: &eightDaysAgo},
	}
	for _, a := range alerts {
		if err := repo.CreateAlert(a); err != nil {
			t.Fatalf("create alert %q: %v", a.Name, err)
		}
	}
	// Deactivate via UpdateAlert (Save), matching how production code deactivates
	// alerts — CreateAlert with IsActive:false at insert time hits GORM's
	// default-tag zero-value skip and would silently persist as active.
	inactive := alerts[len(alerts)-1]
	inactive.IsActive = false
	if err := repo.UpdateAlert(inactive); err != nil {
		t.Fatalf("deactivate alert: %v", err)
	}

	due, err := repo.GetDueAlerts(now)
	if err != nil {
		t.Fatalf("GetDueAlerts: %v", err)
	}

	got := make(map[string]bool, len(due))
	for _, a := range due {
		got[a.Name] = true
	}

	if len(due) != 3 {
		t.Fatalf("expected 3 due alerts, got %d: %+v", len(due), got)
	}
	for _, name := range []string{"weekly, overdue", "weekly, never run", "daily, overdue"} {
		if !got[name] {
			t.Errorf("expected %q to be due, but it was not returned", name)
		}
	}
	for _, name := range []string{"manual, never runs automatically", "weekly, not yet due", "monthly, not yet due", "weekly but inactive"} {
		if got[name] {
			t.Errorf("expected %q to NOT be due, but it was returned", name)
		}
	}
}
