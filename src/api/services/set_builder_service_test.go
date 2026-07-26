package services

import (
	"errors"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSetBuilderServiceTest(t *testing.T) (*SetBuilderService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.SetBuilderRun{}, &models.SetProposal{}, &models.ProposalSlot{}, &models.Notification{}, &models.CoinSet{}, &models.CoinSetTarget{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := NewSetBuilderService(repository.NewSetBuilderRepository(db), repository.NewNotificationRepository(db))
	fixedNow := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }
	return svc, db
}

func TestSetBuilderServiceCreateRunValidatesPrompt(t *testing.T) {
	svc, _ := setupSetBuilderServiceTest(t)
	if _, err := svc.CreateRun(1, SetBuilderRunRequest{Prompt: "   "}); !errors.Is(err, ErrSetBuilderPromptRequired) {
		t.Fatalf("expected prompt required, got %v", err)
	}

	run, err := svc.CreateRun(1, SetBuilderRunRequest{
		Prompt:   "  All US state quarters  ",
		Provider: "anthropic",
		Model:    "claude",
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if run.ID == 0 || run.Prompt != "All US state quarters" || run.Status != models.SetBuilderRunStatusQueued {
		t.Fatalf("unexpected run: %#v", run)
	}
}

func TestSetBuilderServiceCreateProposalFromWorkflowPersistsReviewAndNotifies(t *testing.T) {
	svc, db := setupSetBuilderServiceTest(t)
	run := &models.SetBuilderRun{
		UserID: 1,
		Prompt: "All US silver quarters from 1940s to 1960s",
		Status: models.SetBuilderRunStatusCompleted,
	}
	if err := svc.repo.CreateRun(run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	criteria := models.JSONObject{"year": 1940, "denomination": "Quarter"}

	proposal, err := svc.CreateProposalFromWorkflow(1, run.ID, SetProposalDraft{
		OriginalPrompt: " All US silver quarters from 1940s to 1960s ",
		ProposedName:   "US Silver Quarters 1940-1964",
		Color:          "#c9a84c",
		SelectedScope:  "Date set",
		Slots: []SetProposalSlotDraft{
			{Label: "1940 US Silver Quarter", Criteria: &criteria, GroupName: "1940s", VerificationStatus: models.ProposalSlotVerificationVerified},
			{Label: "1941 US Silver Quarter", Criteria: &criteria, GroupName: "1940s"},
		},
	})
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	if proposal.ID == 0 || len(proposal.Slots) != 2 {
		t.Fatalf("expected proposal with slots, got %#v", proposal)
	}
	if proposal.Slots[1].VerificationStatus != models.ProposalSlotVerificationUnverified || proposal.Slots[1].ValidationNote == "" {
		t.Fatalf("expected default unverified review note, got %#v", proposal.Slots[1])
	}

	var notification models.Notification
	if err := db.Where("user_id = ? AND type = ?", 1, NotificationTypeAgenticSetProposalReady).First(&notification).Error; err != nil {
		t.Fatalf("proposal-ready notification missing: %v", err)
	}
	if notification.ReferenceID != proposal.ID || notification.ReferenceURL != "/sets/proposals/1" {
		t.Fatalf("unexpected notification link: %#v", notification)
	}

	var setCount, targetCount int64
	if err := db.Model(&models.CoinSet{}).Count(&setCount).Error; err != nil {
		t.Fatalf("count sets: %v", err)
	}
	if err := db.Model(&models.CoinSetTarget{}).Count(&targetCount).Error; err != nil {
		t.Fatalf("count targets: %v", err)
	}
	if setCount != 0 || targetCount != 0 {
		t.Fatalf("proposal creation must not create sets or targets, got sets=%d targets=%d", setCount, targetCount)
	}
}

func TestSetBuilderServiceCreateProposalRequiresCompletedOwnedRun(t *testing.T) {
	svc, _ := setupSetBuilderServiceTest(t)
	run := &models.SetBuilderRun{UserID: 1, Prompt: "All US state quarters", Status: models.SetBuilderRunStatusRunning}
	if err := svc.repo.CreateRun(run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	draft := SetProposalDraft{
		OriginalPrompt: "All US state quarters",
		ProposedName:   "US State Quarters",
		Slots:          []SetProposalSlotDraft{{Label: "Delaware Quarter"}},
	}
	if _, err := svc.CreateProposalFromWorkflow(2, run.ID, draft); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("non-owner run should not create proposal, got %v", err)
	}
	if _, err := svc.CreateProposalFromWorkflow(1, run.ID, draft); !errors.Is(err, ErrSetProposalRunNotComplete) {
		t.Fatalf("running run should not create proposal, got %v", err)
	}
}

func TestSetBuilderServiceFindPendingProposalByPromptUsesCanonicalPromptKey(t *testing.T) {
	svc, _ := setupSetBuilderServiceTest(t)
	run := &models.SetBuilderRun{UserID: 1, Prompt: "All US State Quarters", Status: models.SetBuilderRunStatusCompleted}
	if err := svc.repo.CreateRun(run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	created, err := svc.CreateProposalFromWorkflow(1, run.ID, SetProposalDraft{
		OriginalPrompt: "All   US STATE Quarters",
		ProposedName:   "US State Quarters",
		Slots:          []SetProposalSlotDraft{{Label: "Delaware Quarter"}},
	})
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}

	found, err := svc.FindPendingProposalByPrompt(1, " all us state quarters ")
	if err != nil {
		t.Fatalf("find pending: %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("expected duplicate prompt to find proposal %d, got %d", created.ID, found.ID)
	}
}

func TestSetBuilderServiceRejectProposalIsOwnerScoped(t *testing.T) {
	svc, _ := setupSetBuilderServiceTest(t)
	run := &models.SetBuilderRun{UserID: 1, Prompt: "All US state quarters", Status: models.SetBuilderRunStatusCompleted}
	if err := svc.repo.CreateRun(run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	proposal, err := svc.CreateProposalFromWorkflow(1, run.ID, SetProposalDraft{
		OriginalPrompt: "All US state quarters",
		ProposedName:   "US State Quarters",
		Slots:          []SetProposalSlotDraft{{Label: "Delaware Quarter"}},
	})
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	if err := svc.RejectProposal(2, proposal.ID, "no"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("non-owner reject should fail, got %v", err)
	}
	if err := svc.RejectProposal(1, proposal.ID, "Too broad"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	found, err := svc.GetProposal(1, proposal.ID)
	if err != nil {
		t.Fatalf("get proposal: %v", err)
	}
	if found.Status != models.SetProposalStatusRejected || found.RejectionReason != "Too broad" {
		t.Fatalf("unexpected rejected proposal: %#v", found)
	}
}
