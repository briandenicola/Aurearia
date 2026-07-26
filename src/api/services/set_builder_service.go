package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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
)

const (
	NotificationTypeAgenticSetProposalReady   = "agentic_set_proposal_ready"
	NotificationTypeAgenticSetCreated         = "agentic_set_created"
	NotificationTypeAgenticSetCreationFailed  = "agentic_set_creation_failed"
	agenticSetProposalReviewURLFormat         = "/sets/proposals/%d"
	agenticSetProposalDefaultVerificationNote = "Pending human review"
)

var (
	ErrSetBuilderPromptRequired      = errors.New("set builder prompt is required")
	ErrSetBuilderPromptTooLong       = errors.New("set builder prompt is too long")
	ErrSetProposalNameRequired       = errors.New("set proposal name is required")
	ErrSetProposalNameTooLong        = errors.New("set proposal name is too long")
	ErrSetProposalSlotsRequired      = errors.New("set proposal must include at least one slot")
	ErrSetProposalTooManySlots       = errors.New("set proposal includes too many slots")
	ErrSetProposalSlotLabelRequired  = errors.New("set proposal slot label is required")
	ErrSetProposalRunNotComplete     = errors.New("set builder run must be completed before creating a proposal")
	ErrSetProposalVerificationStatus = errors.New("set proposal slot verification status is invalid")
)

// SetBuilderService coordinates Agentic set proposal lifecycle outside of AI inference.
type SetBuilderService struct {
	repo      *repository.SetBuilderRepository
	notifRepo *repository.NotificationRepository
	now       func() time.Time
}

// NewSetBuilderService creates a SetBuilderService.
func NewSetBuilderService(repo *repository.SetBuilderRepository, notifRepo *repository.NotificationRepository) *SetBuilderService {
	return &SetBuilderService{
		repo:      repo,
		notifRepo: notifRepo,
		now:       time.Now,
	}
}

type SetBuilderRunRequest struct {
	Prompt      string
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

// CreateRun validates and persists a queued Agentic set builder run.
func (s *SetBuilderService) CreateRun(userID uint, request SetBuilderRunRequest) (*models.SetBuilderRun, error) {
	prompt, err := normalizeSetBuilderPrompt(request.Prompt)
	if err != nil {
		return nil, err
	}
	run := &models.SetBuilderRun{
		UserID:      userID,
		Prompt:      prompt,
		Status:      models.SetBuilderRunStatusQueued,
		Provider:    strings.TrimSpace(request.Provider),
		Model:       strings.TrimSpace(request.Model),
		MaxTurns:    request.MaxTurns,
		TokenBudget: request.TokenBudget,
	}
	if err := s.repo.CreateRun(run); err != nil {
		return nil, err
	}
	return run, nil
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

func (s *SetBuilderService) RejectProposal(userID, proposalID uint, reason string) error {
	return s.repo.MarkRejected(proposalID, userID, s.now(), strings.TrimSpace(reason))
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
