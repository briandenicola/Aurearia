package services

import (
	"context"
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
	if err := db.AutoMigrate(&models.AppSetting{}, &models.Coin{}, &models.SetBuilderRun{}, &models.SetProposal{}, &models.ProposalSlot{}, &models.Notification{}, &models.CoinSet{}, &models.CoinSetTarget{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := NewSetBuilderService(repository.NewSetBuilderRepository(db), repository.NewNotificationRepository(db))
	fixedNow := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixedNow }
	return svc, db
}

type fakeSetBuilderAgent struct {
	response *SetBuilderProxyResponse
	err      error
	request  SetBuilderProxyRequest
	calls    int
}

func (f *fakeSetBuilderAgent) RunSetBuilder(ctx context.Context, req SetBuilderProxyRequest) (*SetBuilderProxyResponse, error) {
	f.calls++
	f.request = req
	return f.response, f.err
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

func TestSetBuilderServiceUpdateProposalEditsPendingRoster(t *testing.T) {
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
	criteria := models.JSONObject{"year": "1999", "denomination": "Quarter"}
	updated, err := svc.UpdateProposal(1, proposal.ID, SetProposalUpdateRequest{
		ProposedName:  "Edited State Quarters",
		Description:   "Human-reviewed roster",
		Color:         "#123456",
		SelectedScope: "Edited scope",
		Slots: []SetProposalSlotDraft{{
			Label:              "1999 Delaware Quarter",
			Criteria:           &criteria,
			GroupName:          "1999",
			VerificationStatus: models.ProposalSlotVerificationVerified,
		}},
	})
	if err != nil {
		t.Fatalf("update proposal: %v", err)
	}
	if updated.ProposedName != "Edited State Quarters" || updated.SelectedScope != "Edited scope" || len(updated.Slots) != 1 {
		t.Fatalf("unexpected updated proposal: %#v", updated)
	}
	if updated.Slots[0].Label != "1999 Delaware Quarter" || (*updated.Slots[0].Criteria)["year"] != "1999" {
		t.Fatalf("unexpected updated slot: %#v", updated.Slots[0])
	}
}

func TestSetBuilderServiceRegenerateProposalQueuesFeedbackRunAndSupersedesOldProposal(t *testing.T) {
	svc, db := setupSetBuilderServiceTest(t)
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

	regeneratedRun, err := svc.RegenerateProposal(1, proposal.ID, "Group by mint mark instead")
	if err != nil {
		t.Fatalf("regenerate proposal: %v", err)
	}
	if regeneratedRun.Prompt != proposal.OriginalPrompt || regeneratedRun.Feedback != "Group by mint mark instead" {
		t.Fatalf("unexpected regenerated run: %#v", regeneratedRun)
	}
	var superseded models.SetProposal
	if err := db.First(&superseded, proposal.ID).Error; err != nil {
		t.Fatalf("load superseded proposal: %v", err)
	}
	if superseded.Status != models.SetProposalStatusRejected || superseded.RejectionReason != "Superseded by regeneration request" {
		t.Fatalf("old proposal should be superseded, got %#v", superseded)
	}
}

func TestSetBuilderServiceApproveExpiredProposalFailsWithoutCreatingSet(t *testing.T) {
	svc, db := setupSetBuilderServiceTest(t)
	svc.WithSetRepository(repository.NewSetRepository(db))
	run := &models.SetBuilderRun{UserID: 1, Prompt: "All US state quarters", Status: models.SetBuilderRunStatusCompleted}
	if err := svc.repo.CreateRun(run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	expiredAt := svc.now().Add(-time.Hour)
	proposal, err := svc.CreateProposalFromWorkflow(1, run.ID, SetProposalDraft{
		OriginalPrompt: "All US state quarters",
		ProposedName:   "US State Quarters",
		ExpiresAt:      &expiredAt,
		Slots:          []SetProposalSlotDraft{{Label: "Delaware Quarter"}},
	})
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	if _, err := svc.ApproveProposal(1, proposal.ID); !errors.Is(err, ErrSetProposalExpired) {
		t.Fatalf("expected expired proposal error, got %v", err)
	}
	var setCount int64
	if err := db.Model(&models.CoinSet{}).Count(&setCount).Error; err != nil {
		t.Fatalf("count sets: %v", err)
	}
	if setCount != 0 {
		t.Fatalf("expired approval must not create set, got %d", setCount)
	}
}

func TestSetBuilderServiceProcessRunCallsPythonAndPersistsProposal(t *testing.T) {
	svc, db := setupSetBuilderServiceTest(t)
	if err := db.Create(&models.AppSetting{Key: SettingAIProvider, Value: "ollama"}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	if err := db.Create(&models.Coin{UserID: 1, Name: "1940 Washington Quarter", Category: "Modern", Denomination: "Quarter", Material: "Silver"}).Error; err != nil {
		t.Fatalf("seed coin: %v", err)
	}
	agent := &fakeSetBuilderAgent{response: &SetBuilderProxyResponse{
		Status:            "completed",
		TranscriptSummary: "Intent Analyst, Roster Researcher, Collection Matcher, and Validator agreed on a date set.",
		TurnsUsed:         4,
		Proposal: &SetBuilderProposalProxy{
			Name:          "US Silver Quarters 1940-1964",
			SlugHint:      "us-silver-quarters-1940-1964",
			Description:   "A date run of US silver Washington quarters.",
			SelectedScope: "Date set",
			ScopeOptions:  []SetBuilderScopeOptionProxy{{Label: "1940-1964", EstimatedSlotCount: 25, Recommended: true}},
			Slots: []SetBuilderSlotProxy{
				{
					Label:              "1940 US Silver Quarter",
					Criteria:           map[string]string{"year": "1940", "denomination": "Quarter"},
					Group:              "1940s",
					SortOrder:          1,
					VerificationStatus: "verified",
					SourceNote:         "Washington quarter date in requested range",
				},
			},
			PrematchSummary: SetBuilderPrematchSummaryProxy{EstimatedFilled: 1, EstimatedTotal: 25, Notes: "One likely collection match."},
		},
	}}
	svc.WithWorkflow(agent, NewSettingsService(repository.NewSettingsRepository(db)), repository.NewAgentRepository(db), NewLogger(100))
	run, err := svc.CreateRun(1, SetBuilderRunRequest{Prompt: "All US silver quarters from 1940s to 1960s"})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	svc.processRun(run.ID)

	if agent.calls != 1 {
		t.Fatalf("expected one Python workflow call, got %d", agent.calls)
	}
	if agent.request.Prompt != run.Prompt || agent.request.RunID != run.ID || agent.request.Collection == nil || agent.request.Collection.TotalCoins != 1 {
		t.Fatalf("unexpected Python request: %#v", agent.request)
	}
	if agent.request.MaxSlots != maxSetBuilderAgentSlots {
		t.Fatalf("set builder max_slots must stay within Python contract, got %d want %d", agent.request.MaxSlots, maxSetBuilderAgentSlots)
	}
	var persistedRun models.SetBuilderRun
	if err := db.First(&persistedRun, run.ID).Error; err != nil {
		t.Fatalf("load run: %v", err)
	}
	if persistedRun.Status != models.SetBuilderRunStatusCompleted || persistedRun.UsedTurns == nil || *persistedRun.UsedTurns != 4 {
		t.Fatalf("run was not completed with workflow metadata: %#v", persistedRun)
	}
	var proposal models.SetProposal
	if err := db.Preload("Slots").Where("builder_run_id = ?", run.ID).First(&proposal).Error; err != nil {
		t.Fatalf("proposal missing: %v", err)
	}
	if proposal.Status != models.SetProposalStatusPending || len(proposal.Slots) != 1 {
		t.Fatalf("unexpected proposal: %#v", proposal)
	}
	if proposal.Slots[0].Criteria == nil || (*proposal.Slots[0].Criteria)["year"] != "1940" {
		t.Fatalf("slot criteria not persisted from Python response: %#v", proposal.Slots[0])
	}
	var setCount int64
	if err := db.Model(&models.CoinSet{}).Count(&setCount).Error; err != nil {
		t.Fatalf("count sets: %v", err)
	}
	if setCount != 0 {
		t.Fatalf("worker must not create a set before human approval, got %d", setCount)
	}
	var notification models.Notification
	if err := db.Where("type = ?", NotificationTypeAgenticSetProposalReady).First(&notification).Error; err != nil {
		t.Fatalf("proposal-ready notification missing: %v", err)
	}
}

func TestSetBuilderServiceProcessRunFailsVisibleWhenPythonNeedsClarification(t *testing.T) {
	svc, db := setupSetBuilderServiceTest(t)
	if err := db.Create(&models.AppSetting{Key: SettingAIProvider, Value: "ollama"}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	agent := &fakeSetBuilderAgent{response: &SetBuilderProxyResponse{
		Status:                "clarification_needed",
		ClarificationQuestion: "Which country or denomination should this cover?",
		TranscriptSummary:     "Intent Analyst found the prompt too broad.",
		TurnsUsed:             1,
	}}
	svc.WithWorkflow(agent, NewSettingsService(repository.NewSettingsRepository(db)), repository.NewAgentRepository(db), NewLogger(100))
	run, err := svc.CreateRun(1, SetBuilderRunRequest{Prompt: "All coins"})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	svc.processRun(run.ID)

	var persistedRun models.SetBuilderRun
	if err := db.First(&persistedRun, run.ID).Error; err != nil {
		t.Fatalf("load run: %v", err)
	}
	if persistedRun.Status != models.SetBuilderRunStatusFailed || persistedRun.ErrorMessage == "" {
		t.Fatalf("run should fail visibly, got %#v", persistedRun)
	}
	var proposalCount int64
	if err := db.Model(&models.SetProposal{}).Count(&proposalCount).Error; err != nil {
		t.Fatalf("count proposals: %v", err)
	}
	if proposalCount != 0 {
		t.Fatalf("clarification response must not create proposal, got %d", proposalCount)
	}
	var notification models.Notification
	if err := db.Where("type = ?", "agentic_set_proposal_failed").First(&notification).Error; err != nil {
		t.Fatalf("failure notification missing: %v", err)
	}
}
