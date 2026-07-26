package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSetBuilderRepositoryTest(t *testing.T) (*SetBuilderRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CoinSet{}, &models.CoinSetTarget{}, &models.SetBuilderRun{}, &models.SetProposal{}, &models.ProposalSlot{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewSetBuilderRepository(db), db
}

func TestSetBuilderRepositoryCreateProposalWithSlotsPersistsReviewDataOnly(t *testing.T) {
	repo, db := setupSetBuilderRepositoryTest(t)
	run := &models.SetBuilderRun{
		UserID: 1,
		Prompt: "All US silver quarters from 1940s to 1960s",
		Status: models.SetBuilderRunStatusCompleted,
	}
	if err := repo.CreateRun(run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	criteria := models.JSONObject{"year": 1940, "denomination": "Quarter", "material": "Silver"}
	proposal := &models.SetProposal{
		UserID:         1,
		BuilderRunID:   run.ID,
		OriginalPrompt: "All US silver quarters from 1940s to 1960s",
		Status:         models.SetProposalStatusPending,
		ProposedName:   "US Silver Quarters 1940-1964",
		Color:          "#c9a84c",
		IdempotencyKey: "prompt-hash",
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
	}
	slots := []models.ProposalSlot{
		{
			Label:              "1940 US Silver Quarter",
			Criteria:           &criteria,
			GroupName:          "1940s",
			SortOrder:          0,
			VerificationStatus: models.ProposalSlotVerificationVerified,
			SourceNote:         "validated",
		},
		{
			Label:              "1941 US Silver Quarter",
			Criteria:           &criteria,
			GroupName:          "1940s",
			SortOrder:          1,
			VerificationStatus: models.ProposalSlotVerificationUnverified,
			ValidationNote:     "needs review",
		},
	}

	if err := repo.CreateProposalWithSlots(proposal, slots); err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	if proposal.ID == 0 || len(proposal.Slots) != 2 {
		t.Fatalf("expected persisted proposal with slots, got %#v", proposal)
	}
	if proposal.Slots[0].SortOrder != 0 || proposal.Slots[1].VerificationStatus != models.ProposalSlotVerificationUnverified {
		t.Fatalf("slots not loaded in review order: %#v", proposal.Slots)
	}

	var setCount, targetCount int64
	if err := db.Model(&models.CoinSet{}).Count(&setCount).Error; err != nil {
		t.Fatalf("count sets: %v", err)
	}
	if err := db.Model(&models.CoinSetTarget{}).Count(&targetCount).Error; err != nil {
		t.Fatalf("count targets: %v", err)
	}
	if setCount != 0 || targetCount != 0 {
		t.Fatalf("proposal persistence must not create sets or targets, got sets=%d targets=%d", setCount, targetCount)
	}

	if _, err := repo.GetProposalForUser(proposal.ID, 2); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("non-owner should not read proposal, got %v", err)
	}
}

func TestSetBuilderRepositoryFindPendingProposalByIdempotencyKeyIsUserScopedAndUnexpired(t *testing.T) {
	repo, _ := setupSetBuilderRepositoryTest(t)
	now := time.Now()
	for _, proposal := range []models.SetProposal{
		{UserID: 1, BuilderRunID: 1, OriginalPrompt: "same", Status: models.SetProposalStatusPending, ProposedName: "Expired", IdempotencyKey: "same-key", ExpiresAt: now.Add(-time.Hour)},
		{UserID: 2, BuilderRunID: 1, OriginalPrompt: "same", Status: models.SetProposalStatusPending, ProposedName: "Other user", IdempotencyKey: "same-key", ExpiresAt: now.Add(time.Hour)},
		{UserID: 1, BuilderRunID: 1, OriginalPrompt: "same", Status: models.SetProposalStatusRejected, ProposedName: "Rejected", IdempotencyKey: "same-key", ExpiresAt: now.Add(time.Hour)},
		{UserID: 1, BuilderRunID: 1, OriginalPrompt: "same", Status: models.SetProposalStatusPending, ProposedName: "Active", IdempotencyKey: "same-key", ExpiresAt: now.Add(time.Hour)},
	} {
		p := proposal
		if err := repo.db.Create(&p).Error; err != nil {
			t.Fatalf("seed proposal: %v", err)
		}
	}

	found, err := repo.FindPendingProposalByIdempotencyKey(1, "same-key", now)
	if err != nil {
		t.Fatalf("find pending: %v", err)
	}
	if found.ProposedName != "Active" {
		t.Fatalf("expected active owner proposal, got %q", found.ProposedName)
	}
}

func TestSetBuilderRepositoryLifecycleTransitionsRequirePendingOwnerProposal(t *testing.T) {
	repo, _ := setupSetBuilderRepositoryTest(t)
	now := time.Now()
	proposal := &models.SetProposal{
		UserID:         1,
		BuilderRunID:   1,
		OriginalPrompt: "All US state quarters",
		Status:         models.SetProposalStatusPending,
		ProposedName:   "US State Quarters",
		IdempotencyKey: "state-quarters",
		ExpiresAt:      now.Add(time.Hour),
	}
	if err := repo.db.Create(proposal).Error; err != nil {
		t.Fatalf("seed proposal: %v", err)
	}

	if err := repo.MarkApproved(proposal.ID, 2, 99, now); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("non-owner approve should fail, got %v", err)
	}
	if err := repo.MarkApproved(proposal.ID, 1, 99, now); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := repo.MarkRejected(proposal.ID, 1, now, "changed mind"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("approved proposal should not reject, got %v", err)
	}

	found, err := repo.GetProposalForUser(proposal.ID, 1)
	if err != nil {
		t.Fatalf("get proposal: %v", err)
	}
	if found.Status != models.SetProposalStatusApproved || found.ApprovalSetID == nil || *found.ApprovalSetID != 99 {
		t.Fatalf("proposal not approved with set id: %#v", found)
	}
}
