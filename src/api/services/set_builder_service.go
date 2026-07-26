package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	maxSetBuilderPromptLength = 1000
	maxSetProposalNameLength  = 80
	maxSetProposalSlots       = 500
	defaultProposalExpiry     = 7 * 24 * time.Hour
	setBuilderQueueSize       = 100
	setBuilderRunTimeout      = 10 * time.Minute
	setBuilderStaleTimeout    = 30 * time.Minute
)

const (
	NotificationTypeAgenticSetProposalReady   = "agentic_set_proposal_ready"
	NotificationTypeAgenticSetCreated         = "agentic_set_created"
	NotificationTypeAgenticSetCreationFailed  = "agentic_set_creation_failed"
	agenticSetProposalReviewURLFormat         = "/sets/proposals/%d"
	agenticSetProposalDefaultVerificationNote = "Pending human review"
)

var (
	ErrSetBuilderPromptRequired       = errors.New("set builder prompt is required")
	ErrSetBuilderPromptTooLong        = errors.New("set builder prompt is too long")
	ErrSetProposalNameRequired        = errors.New("set proposal name is required")
	ErrSetProposalNameTooLong         = errors.New("set proposal name is too long")
	ErrSetProposalSlotsRequired       = errors.New("set proposal must include at least one slot")
	ErrSetProposalTooManySlots        = errors.New("set proposal includes too many slots")
	ErrSetProposalSlotLabelRequired   = errors.New("set proposal slot label is required")
	ErrSetProposalRunNotComplete      = errors.New("set builder run must be completed before creating a proposal")
	ErrSetProposalVerificationStatus  = errors.New("set proposal slot verification status is invalid")
	ErrSetProposalApprovalUnavailable = errors.New("set proposal approval is not configured")
	ErrSetProposalExpired             = errors.New("set proposal has expired")
	ErrSetProposalFeedbackRequired    = errors.New("set proposal regeneration feedback is required")
)

type SetBuilderAgent interface {
	RunSetBuilder(ctx context.Context, req SetBuilderProxyRequest) (*SetBuilderProxyResponse, error)
}

// SetBuilderService coordinates Agentic set proposal lifecycle outside of AI inference.
type SetBuilderService struct {
	repo        *repository.SetBuilderRepository
	notifRepo   *repository.NotificationRepository
	agent       SetBuilderAgent
	settingsSvc *SettingsService
	agentRepo   *repository.AgentRepository
	setRepo     *repository.SetRepository
	logger      *Logger
	queue       chan uint
	now         func() time.Time
}

// NewSetBuilderService creates a SetBuilderService.
func NewSetBuilderService(repo *repository.SetBuilderRepository, notifRepo *repository.NotificationRepository) *SetBuilderService {
	return &SetBuilderService{
		repo:      repo,
		notifRepo: notifRepo,
		queue:     make(chan uint, setBuilderQueueSize),
		now:       time.Now,
	}
}

// WithWorkflow enables asynchronous Python workflow execution for queued runs.
func (s *SetBuilderService) WithWorkflow(agent SetBuilderAgent, settingsSvc *SettingsService, agentRepo *repository.AgentRepository, logger *Logger) *SetBuilderService {
	s.agent = agent
	s.settingsSvc = settingsSvc
	s.agentRepo = agentRepo
	s.logger = logger
	return s
}

// WithSetRepository enables approval-time Agentic set creation.
func (s *SetBuilderService) WithSetRepository(setRepo *repository.SetRepository) *SetBuilderService {
	s.setRepo = setRepo
	return s
}

type SetBuilderRunRequest struct {
	Prompt      string
	Feedback    string
	Provider    string
	Model       string
	MaxTurns    *int
	TokenBudget *int
}

type SetProposalDraft struct {
	OriginalPrompt  string
	ProposedName    string
	ProposedSlug    string
	Description     string
	Color           string
	SelectedScope   string
	ScopeOptions    *models.JSONObject
	RosterPayload   *models.JSONObject
	PreMatchSummary *models.JSONObject
	ExpiresAt       *time.Time
	Slots           []SetProposalSlotDraft
}

type SetProposalSlotDraft struct {
	Label              string
	Criteria           *models.JSONObject
	GroupName          string
	SortOrder          int
	VerificationStatus models.ProposalSlotVerificationStatus
	SourceNote         string
	ValidationNote     string
}

type SetProposalUpdateRequest struct {
	ProposedName  string
	Description   string
	Color         string
	SelectedScope string
	Slots         []SetProposalSlotDraft
}

// CreateRun validates and persists a queued Agentic set builder run.
func (s *SetBuilderService) CreateRun(userID uint, request SetBuilderRunRequest) (*models.SetBuilderRun, error) {
	prompt, err := normalizeSetBuilderPrompt(request.Prompt)
	if err != nil {
		return nil, err
	}
	run := &models.SetBuilderRun{
		UserID:      userID,
		Prompt:      prompt,
		Feedback:    strings.TrimSpace(request.Feedback),
		Status:      models.SetBuilderRunStatusQueued,
		Provider:    strings.TrimSpace(request.Provider),
		Model:       strings.TrimSpace(request.Model),
		MaxTurns:    request.MaxTurns,
		TokenBudget: request.TokenBudget,
	}
	if err := s.repo.CreateRun(run); err != nil {
		return nil, err
	}
	s.enqueueRunID(run.ID)
	return run, nil
}

func (s *SetBuilderService) StartWorkers(workerCount int) {
	if workerCount < 1 {
		workerCount = 1
	}
	if ids, err := s.repo.RecoverStaleRuns(setBuilderStaleTimeout); err == nil {
		for _, id := range ids {
			s.enqueueRunID(id)
		}
	} else if s.logger != nil {
		s.logger.Warn("set-builder", "Failed to recover stale set builder runs: %v", err)
	}
	for i := 0; i < workerCount; i++ {
		go s.worker()
	}
}

func (s *SetBuilderService) enqueueRunID(runID uint) {
	if s.agent == nil || s.settingsSvc == nil {
		return
	}
	select {
	case s.queue <- runID:
	default:
		go func() { s.queue <- runID }()
	}
}

func (s *SetBuilderService) worker() {
	for runID := range s.queue {
		s.processRun(runID)
	}
}

func (s *SetBuilderService) processRun(runID uint) {
	run, claimed, err := s.repo.ClaimQueuedRun(runID, s.now())
	if err != nil {
		s.logError("Failed to claim set builder run %d: %v", runID, err)
		return
	}
	if !claimed {
		return
	}

	if err := s.processClaimedRun(run); err != nil {
		s.logError("Set builder run %d failed: %v", run.ID, err)
		if failErr := s.repo.FailRun(run.ID, run.UserID, s.now(), err.Error(), "go_worker"); failErr != nil {
			s.logError("Failed to persist set builder run %d failure: %v", run.ID, failErr)
		}
		s.notifyRunFailed(run, err.Error())
	}
}

func (s *SetBuilderService) processClaimedRun(run *models.SetBuilderRun) error {
	if s.agent == nil {
		return errors.New("set builder agent workflow is not configured")
	}
	if s.settingsSvc == nil {
		return errors.New("set builder settings service is not configured")
	}
	llmCfg, err := s.settingsSvc.ResolveLLMConfig()
	if err != nil {
		return err
	}
	collection, err := s.collectionSummary(run.UserID)
	if err != nil {
		return err
	}
	maxTurns := 4
	if run.MaxTurns != nil && *run.MaxTurns > 0 {
		maxTurns = *run.MaxTurns
	}
	ctx, cancel := context.WithTimeout(context.Background(), setBuilderRunTimeout)
	defer cancel()
	result, err := s.agent.RunSetBuilder(ctx, SetBuilderProxyRequest{
		LLM:                  llmCfg,
		User:                 UserContextProxy{UserID: run.UserID},
		Prompt:               run.Prompt,
		Collection:           collection,
		MaxTurns:             maxTurns,
		MaxSlots:             maxSetProposalSlots,
		EnableExternalLookup: true,
		Feedback:             run.Feedback,
	})
	if err != nil {
		return err
	}
	if result == nil {
		return errors.New("set builder returned no response")
	}
	if result.Status != "completed" {
		return fmt.Errorf("set builder workflow %s: %s", result.Status, setBuilderFailureMessage(result))
	}
	if result.Proposal == nil || len(result.Proposal.Slots) == 0 {
		return errors.New("set builder completed without proposal slots")
	}
	usedTurns := result.TurnsUsed
	if err := s.repo.CompleteRun(run.ID, run.UserID, s.now(), strings.TrimSpace(result.TranscriptSummary), &usedTurns, nil); err != nil {
		return err
	}
	_, err = s.CreateProposalFromWorkflow(run.UserID, run.ID, s.workflowDraft(run.UserID, run.Prompt, result))
	return err
}

func (s *SetBuilderService) collectionSummary(userID uint) (*PortfolioData, error) {
	if s.agentRepo == nil {
		return nil, nil
	}
	summary, err := s.agentRepo.GetPortfolioSummary(userID)
	if err != nil {
		return nil, err
	}
	return portfolioDataFromSummary(summary), nil
}

func (s *SetBuilderService) workflowDraft(userID uint, prompt string, result *SetBuilderProxyResponse) SetProposalDraft {
	proposal := result.Proposal
	scopeOptions := models.JSONObject{
		"scopeSummary": proposal.ScopeSummary,
		"groupBy":      proposal.GroupBy,
		"options":      proposal.ScopeOptions,
	}
	prematch := models.JSONObject{
		"estimatedFilled": proposal.PrematchSummary.EstimatedFilled,
		"estimatedTotal":  proposal.PrematchSummary.EstimatedTotal,
		"notes":           proposal.PrematchSummary.Notes,
	}
	roster := models.JSONObject{
		"status":            result.Status,
		"transcriptSummary": result.TranscriptSummary,
		"turnsUsed":         result.TurnsUsed,
	}
	slots := make([]SetProposalSlotDraft, 0, len(proposal.Slots))
	for _, slot := range proposal.Slots {
		criteria := stringMapToJSONObject(slot.Criteria)
		slots = append(slots, SetProposalSlotDraft{
			Label:              slot.Label,
			Criteria:           criteria,
			GroupName:          slot.Group,
			SortOrder:          slot.SortOrder,
			VerificationStatus: models.ProposalSlotVerificationStatus(slot.VerificationStatus),
			SourceNote:         slot.SourceNote,
			ValidationNote:     slot.ValidationNotes,
		})
	}
	return SetProposalDraft{
		OriginalPrompt:  prompt,
		ProposedName:    proposal.Name,
		ProposedSlug:    proposal.SlugHint,
		Description:     proposal.Description,
		Color:           DefaultSetColor(userID, proposal.Name, int64(len(proposal.Slots))),
		SelectedScope:   proposal.SelectedScope,
		ScopeOptions:    &scopeOptions,
		RosterPayload:   &roster,
		PreMatchSummary: &prematch,
		Slots:           slots,
	}
}

// FindPendingProposalByPrompt locates an existing unexpired pending proposal for duplicate prompt submissions.
func (s *SetBuilderService) FindPendingProposalByPrompt(userID uint, prompt string) (*models.SetProposal, error) {
	normalized, err := normalizeSetBuilderPrompt(prompt)
	if err != nil {
		return nil, err
	}
	return s.repo.FindPendingProposalByIdempotencyKey(userID, setBuilderPromptKey(normalized), s.now())
}

// CreateProposalFromWorkflow persists a completed workflow proposal and notifies the user for review.
func (s *SetBuilderService) CreateProposalFromWorkflow(userID, runID uint, draft SetProposalDraft) (*models.SetProposal, error) {
	run, err := s.repo.GetRunForUser(runID, userID)
	if err != nil {
		return nil, err
	}
	if run.Status != models.SetBuilderRunStatusCompleted {
		return nil, ErrSetProposalRunNotComplete
	}
	proposal, slots, err := s.buildProposal(userID, runID, draft)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateProposalWithSlots(proposal, slots); err != nil {
		return nil, err
	}
	if err := s.notifyProposalReady(proposal); err != nil {
		return nil, err
	}
	return proposal, nil
}

func (s *SetBuilderService) ListProposals(userID uint, limit int) ([]models.SetProposal, error) {
	return s.repo.ListProposals(userID, limit)
}

func (s *SetBuilderService) GetProposal(userID, proposalID uint) (*models.SetProposal, error) {
	return s.repo.GetProposalForUser(proposalID, userID)
}

func (s *SetBuilderService) UpdateProposal(userID, proposalID uint, request SetProposalUpdateRequest) (*models.SetProposal, error) {
	proposal, err := s.repo.GetProposalForUser(proposalID, userID)
	if err != nil {
		return nil, err
	}
	if err := s.ensurePendingProposal(proposal); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(request.ProposedName)
	if name == "" {
		return nil, ErrSetProposalNameRequired
	}
	if len(name) > maxSetProposalNameLength {
		return nil, ErrSetProposalNameTooLong
	}
	slotDrafts := request.Slots
	if len(slotDrafts) == 0 {
		slotDrafts = proposalSlotsToDrafts(proposal.Slots)
	}
	if len(slotDrafts) == 0 {
		return nil, ErrSetProposalSlotsRequired
	}
	if len(slotDrafts) > maxSetProposalSlots {
		return nil, ErrSetProposalTooManySlots
	}
	slots := make([]models.ProposalSlot, 0, len(slotDrafts))
	for i, slotDraft := range slotDrafts {
		slot, err := buildProposalSlot(slotDraft, i)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	updates := map[string]interface{}{
		"proposed_name":  name,
		"description":    strings.TrimSpace(request.Description),
		"color":          strings.TrimSpace(request.Color),
		"selected_scope": strings.TrimSpace(request.SelectedScope),
		"updated_at":     s.now(),
	}
	if updates["color"] == "" {
		updates["color"] = DefaultSetColor(userID, name, int64(len(slots)))
	}
	return s.repo.UpdatePendingProposalWithSlots(proposalID, userID, updates, slots)
}

func (s *SetBuilderService) RejectProposal(userID, proposalID uint, reason string) error {
	return s.repo.MarkRejected(proposalID, userID, s.now(), strings.TrimSpace(reason))
}

func (s *SetBuilderService) RegenerateProposal(userID, proposalID uint, feedback string) (*models.SetBuilderRun, error) {
	feedback = strings.TrimSpace(feedback)
	if feedback == "" {
		return nil, ErrSetProposalFeedbackRequired
	}
	proposal, err := s.repo.GetProposalForUser(proposalID, userID)
	if err != nil {
		return nil, err
	}
	if err := s.ensurePendingProposal(proposal); err != nil {
		return nil, err
	}
	run := &models.SetBuilderRun{
		UserID:   userID,
		Prompt:   proposal.OriginalPrompt,
		Feedback: feedback,
		Status:   models.SetBuilderRunStatusQueued,
	}
	if err := s.repo.CreateRun(run); err != nil {
		return nil, err
	}
	if err := s.repo.MarkRejected(proposalID, userID, s.now(), "Superseded by regeneration request"); err != nil {
		return nil, err
	}
	s.enqueueRunID(run.ID)
	return run, nil
}

func (s *SetBuilderService) ApproveProposal(userID, proposalID uint) (*models.CoinSet, error) {
	if s.setRepo == nil {
		return nil, ErrSetProposalApprovalUnavailable
	}
	proposal, err := s.repo.GetProposalForUser(proposalID, userID)
	if err != nil {
		return nil, err
	}
	if proposal.Status == models.SetProposalStatusApproved && proposal.ApprovalSetID != nil {
		return s.setRepo.GetByID(*proposal.ApprovalSetID, userID)
	}
	if err := s.ensurePendingProposal(proposal); err != nil {
		return nil, err
	}
	set := &models.CoinSet{
		UserID:        userID,
		Name:          strings.TrimSpace(proposal.ProposedName),
		Description:   strings.TrimSpace(proposal.Description),
		Color:         proposal.Color,
		SetType:       models.CoinSetTypeAgentic,
		CreationMode:  models.CoinSetCreationModeDynamic,
		AgenticPrompt: proposal.OriginalPrompt,
		AgenticStatus: "ready",
	}
	if set.Color == "" {
		set.Color = DefaultSetColor(userID, set.Name, int64(len(proposal.Slots)))
	}
	targets := make([]models.CoinSetTarget, 0, len(proposal.Slots))
	for _, slot := range proposal.Slots {
		targets = append(targets, proposalSlotToTarget(slot))
	}
	created, err := s.repo.ApproveProposalWithSet(proposalID, userID, set, targets, s.now())
	if err != nil {
		return nil, err
	}
	s.notifySetCreated(userID, proposalID, created)
	return created, nil
}

func (s *SetBuilderService) ensurePendingProposal(proposal *models.SetProposal) error {
	if proposal == nil || proposal.Status != models.SetProposalStatusPending {
		return repository.ErrRecordNotFound
	}
	if !proposal.ExpiresAt.IsZero() && !proposal.ExpiresAt.After(s.now()) {
		_ = s.repo.MarkExpired(proposal.ID, proposal.UserID, s.now())
		return ErrSetProposalExpired
	}
	return nil
}

func (s *SetBuilderService) buildProposal(userID, runID uint, draft SetProposalDraft) (*models.SetProposal, []models.ProposalSlot, error) {
	prompt, err := normalizeSetBuilderPrompt(draft.OriginalPrompt)
	if err != nil {
		return nil, nil, err
	}
	name := strings.TrimSpace(draft.ProposedName)
	if name == "" {
		return nil, nil, ErrSetProposalNameRequired
	}
	if len(name) > maxSetProposalNameLength {
		return nil, nil, ErrSetProposalNameTooLong
	}
	if len(draft.Slots) == 0 {
		return nil, nil, ErrSetProposalSlotsRequired
	}
	if len(draft.Slots) > maxSetProposalSlots {
		return nil, nil, ErrSetProposalTooManySlots
	}
	expiresAt := s.now().Add(defaultProposalExpiry)
	if draft.ExpiresAt != nil {
		expiresAt = *draft.ExpiresAt
	}
	proposal := &models.SetProposal{
		UserID:          userID,
		BuilderRunID:    runID,
		OriginalPrompt:  prompt,
		Status:          models.SetProposalStatusPending,
		ProposedName:    name,
		ProposedSlug:    strings.TrimSpace(draft.ProposedSlug),
		Description:     strings.TrimSpace(draft.Description),
		Color:           strings.TrimSpace(draft.Color),
		SelectedScope:   strings.TrimSpace(draft.SelectedScope),
		ScopeOptions:    draft.ScopeOptions,
		RosterPayload:   draft.RosterPayload,
		PreMatchSummary: draft.PreMatchSummary,
		IdempotencyKey:  setBuilderPromptKey(prompt),
		ExpiresAt:       expiresAt,
	}
	slots := make([]models.ProposalSlot, 0, len(draft.Slots))
	for i, slotDraft := range draft.Slots {
		slot, err := buildProposalSlot(slotDraft, i)
		if err != nil {
			return nil, nil, err
		}
		slots = append(slots, slot)
	}
	return proposal, slots, nil
}

func buildProposalSlot(draft SetProposalSlotDraft, defaultSortOrder int) (models.ProposalSlot, error) {
	label := strings.TrimSpace(draft.Label)
	if label == "" {
		return models.ProposalSlot{}, ErrSetProposalSlotLabelRequired
	}
	status := draft.VerificationStatus
	if status == "" {
		status = models.ProposalSlotVerificationUnverified
	}
	if status != models.ProposalSlotVerificationVerified && status != models.ProposalSlotVerificationUnverified {
		return models.ProposalSlot{}, ErrSetProposalVerificationStatus
	}
	validationNote := strings.TrimSpace(draft.ValidationNote)
	if status == models.ProposalSlotVerificationUnverified && validationNote == "" {
		validationNote = agenticSetProposalDefaultVerificationNote
	}
	sortOrder := draft.SortOrder
	if sortOrder == 0 {
		sortOrder = defaultSortOrder
	}
	return models.ProposalSlot{
		Label:              label,
		Criteria:           draft.Criteria,
		GroupName:          strings.TrimSpace(draft.GroupName),
		SortOrder:          sortOrder,
		VerificationStatus: status,
		SourceNote:         strings.TrimSpace(draft.SourceNote),
		ValidationNote:     validationNote,
	}, nil
}

func proposalSlotsToDrafts(slots []models.ProposalSlot) []SetProposalSlotDraft {
	drafts := make([]SetProposalSlotDraft, 0, len(slots))
	for _, slot := range slots {
		drafts = append(drafts, SetProposalSlotDraft{
			Label:              slot.Label,
			Criteria:           slot.Criteria,
			GroupName:          slot.GroupName,
			SortOrder:          slot.SortOrder,
			VerificationStatus: slot.VerificationStatus,
			SourceNote:         slot.SourceNote,
			ValidationNote:     slot.ValidationNote,
		})
	}
	return drafts
}

func (s *SetBuilderService) notifyProposalReady(proposal *models.SetProposal) error {
	if s.notifRepo == nil {
		return nil
	}
	return s.notifRepo.Create(&models.Notification{
		UserID:       proposal.UserID,
		Type:         NotificationTypeAgenticSetProposalReady,
		Title:        "Agentic set ready for review",
		Message:      fmt.Sprintf("%s is ready for review.", proposal.ProposedName),
		ReferenceID:  proposal.ID,
		ReferenceURL: fmt.Sprintf(agenticSetProposalReviewURLFormat, proposal.ID),
	})
}

func (s *SetBuilderService) notifyRunFailed(run *models.SetBuilderRun, reason string) {
	if s.notifRepo == nil {
		return
	}
	message := "Agentic set proposal generation failed. Please check AI provider configuration and try again."
	if reason != "" {
		message = "Agentic set proposal generation failed. Please review the prompt or AI provider configuration and try again."
	}
	_ = s.notifRepo.Create(&models.Notification{
		UserID:       run.UserID,
		Type:         "agentic_set_proposal_failed",
		Title:        "Agentic set proposal failed",
		Message:      message,
		ReferenceID:  run.ID,
		ReferenceURL: "/sets",
	})
}

func (s *SetBuilderService) notifySetCreated(userID, proposalID uint, set *models.CoinSet) {
	if s.notifRepo == nil || set == nil {
		return
	}
	_ = s.notifRepo.Create(&models.Notification{
		UserID:       userID,
		Type:         NotificationTypeAgenticSetCreated,
		Title:        "Agentic set created",
		Message:      fmt.Sprintf("%s has been added to Sets.", set.Name),
		ReferenceID:  proposalID,
		ReferenceURL: fmt.Sprintf("/sets/%d", set.ID),
	})
}

func (s *SetBuilderService) logError(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Error("set-builder", format, args...)
	}
}

func normalizeSetBuilderPrompt(prompt string) (string, error) {
	normalized := strings.TrimSpace(prompt)
	if normalized == "" {
		return "", ErrSetBuilderPromptRequired
	}
	if len(normalized) > maxSetBuilderPromptLength {
		return "", ErrSetBuilderPromptTooLong
	}
	return normalized, nil
}

func setBuilderPromptKey(prompt string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(prompt), " "))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func setBuilderFailureMessage(result *SetBuilderProxyResponse) string {
	if result.FailureReason != "" {
		return result.FailureReason
	}
	if result.ClarificationQuestion != "" {
		return result.ClarificationQuestion
	}
	return "no completed proposal was produced"
}

func stringMapToJSONObject(values map[string]string) *models.JSONObject {
	if len(values) == 0 {
		return nil
	}
	out := models.JSONObject{}
	for key, value := range values {
		out[key] = value
	}
	return &out
}

func portfolioDataFromSummary(s *repository.PortfolioSummary) *PortfolioData {
	if s == nil {
		return nil
	}
	cats := make(map[string]int, len(s.Categories))
	for _, c := range s.Categories {
		cats[c.Category] = c.Count
	}
	mats := make(map[string]int, len(s.Materials))
	for _, m := range s.Materials {
		mats[m.Material] = m.Count
	}
	eras := make([]map[string]any, 0, len(s.Eras))
	for _, e := range s.Eras {
		eras = append(eras, map[string]any{"name": e.Era, "count": e.Count})
	}
	rulers := make([]map[string]any, 0, len(s.Rulers))
	for _, r := range s.Rulers {
		rulers = append(rulers, map[string]any{"name": r.Ruler, "count": r.Count})
	}
	coins := make([]PortfolioCoinProxy, 0, len(s.TopCoins))
	for _, tc := range s.TopCoins {
		var cv float64
		if tc.CurrentValue != nil {
			cv = *tc.CurrentValue
		}
		coins = append(coins, PortfolioCoinProxy{
			Name:         tc.Name,
			Category:     tc.Category,
			Era:          string(tc.Era),
			Ruler:        tc.Ruler,
			Grade:        tc.Grade,
			CurrentValue: cv,
		})
	}
	return &PortfolioData{
		TotalCoins:    int(s.TotalCoins),
		TotalValue:    s.TotalValue,
		TotalInvested: s.TotalInvested,
		Categories:    cats,
		Materials:     mats,
		Eras:          eras,
		Rulers:        rulers,
		TopCoins:      coins,
		MissingFields: s.MissingFields,
	}
}

func proposalSlotToTarget(slot models.ProposalSlot) models.CoinSetTarget {
	target := models.CoinSetTarget{
		Label:     strings.TrimSpace(slot.Label),
		SortOrder: slot.SortOrder,
	}
	if slot.Criteria != nil {
		rules := models.JSONObject{}
		for key, value := range *slot.Criteria {
			rules[key] = value
			switch strings.ToLower(key) {
			case "year", "date":
				if year, ok := parseTargetYear(value); ok {
					target.Year = &year
				}
			case "mintmark", "mint_mark", "mint":
				if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
					target.MintMark = &text
				}
			case "denomination", "coin_type", "type":
				if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
					target.Denomination = &text
				}
			case "country", "issuer":
				if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
					target.Country = &text
				}
			case "material", "metal":
				if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
					target.Material = &text
				}
			}
		}
		target.MatchRules = &rules
	}
	return target
}

func parseTargetYear(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if len(trimmed) >= 4 {
			trimmed = trimmed[:4]
		}
		year, err := strconv.Atoi(trimmed)
		return year, err == nil
	default:
		return 0, false
	}
}
