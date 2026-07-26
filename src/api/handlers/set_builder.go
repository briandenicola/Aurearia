package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// SetBuilderHandler handles Agentic set builder proposal workflow requests.
type SetBuilderHandler struct {
	service *services.SetBuilderService
}

// NewSetBuilderHandler creates a new SetBuilderHandler.
func NewSetBuilderHandler(service *services.SetBuilderService) *SetBuilderHandler {
	return &SetBuilderHandler{service: service}
}

type createSetBuilderRunRequest struct {
	Prompt string `json:"prompt"`
}

type rejectSetProposalRequest struct {
	Reason string `json:"reason"`
}

type updateSetProposalSlotRequest struct {
	Label              string                 `json:"label"`
	Criteria           map[string]interface{} `json:"criteria"`
	Group              string                 `json:"group"`
	SortOrder          int                    `json:"sortOrder"`
	VerificationStatus string                 `json:"verificationStatus"`
	SourceNote         string                 `json:"sourceNote"`
	ValidationNote     string                 `json:"validationNote"`
}

type updateSetProposalRequest struct {
	ProposedName  string                         `json:"proposedName"`
	Description   string                         `json:"description"`
	Color         string                         `json:"color"`
	SelectedScope string                         `json:"selectedScope"`
	Slots         []updateSetProposalSlotRequest `json:"slots"`
}

type regenerateSetProposalRequest struct {
	Feedback string `json:"feedback"`
}

// CreateRun queues an Agentic set proposal request.
//
//	@Summary		Queue Agentic set proposal
//	@Description	Submits a natural-language Agentic set prompt for asynchronous proposal generation. No set is created by this endpoint.
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		createSetBuilderRunRequest	true	"Agentic set prompt"
//	@Success		202		{object}	object{run=object}
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Router			/set-builder/runs [post]
func (h *SetBuilderHandler) CreateRun(c *gin.Context) {
	var body createSetBuilderRunRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	run, err := h.service.CreateRun(c.GetUint("userId"), services.SetBuilderRunRequest{Prompt: body.Prompt})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSetBuilderPromptRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Set builder prompt is required"})
		case errors.Is(err, services.ErrSetBuilderPromptTooLong):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Set builder prompt is too long"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit set proposal request"})
		}
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"run": run})
}

// ListProposals returns Agentic set proposals for the current user.
//
//	@Summary		List Agentic set proposals
//	@Description	Returns human-reviewable Agentic set proposals for the current user.
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	object{proposals=[]models.SetProposal}
//	@Failure		401	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Router			/set-builder/proposals [get]
func (h *SetBuilderHandler) ListProposals(c *gin.Context) {
	proposals, err := h.service.ListProposals(c.GetUint("userId"), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load set proposals"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"proposals": proposals})
}

// GetProposal returns one Agentic set proposal for review.
//
//	@Summary		Get Agentic set proposal
//	@Description	Returns one human-reviewable Agentic set proposal with slots and run summary.
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Proposal ID"
//	@Success		200	{object}	models.SetProposal
//	@Failure		400	{object}	object{error=string}
//	@Failure		401	{object}	object{error=string}
//	@Failure		404	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Router			/set-builder/proposals/{id} [get]
func (h *SetBuilderHandler) GetProposal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	proposal, err := h.service.GetProposal(c.GetUint("userId"), id)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Set proposal not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load set proposal"})
		return
	}
	c.JSON(http.StatusOK, proposal)
}

// UpdateProposal updates one pending Agentic set proposal before approval.
//
//	@Summary		Update Agentic set proposal
//	@Description	Edits proposal metadata and roster slots while the proposal is pending.
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	int							true	"Proposal ID"
//	@Param			body	body	updateSetProposalRequest	true	"Proposal edits"
//	@Success		200		{object}	models.SetProposal
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		404		{object}	object{error=string}
//	@Failure		409		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Router			/set-builder/proposals/{id} [put]
func (h *SetBuilderHandler) UpdateProposal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var body updateSetProposalRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	proposal, err := h.service.UpdateProposal(c.GetUint("userId"), id, services.SetProposalUpdateRequest{
		ProposedName:  body.ProposedName,
		Description:   body.Description,
		Color:         body.Color,
		SelectedScope: body.SelectedScope,
		Slots:         proposalSlotDraftsFromRequest(body.Slots),
	})
	if err != nil {
		writeSetProposalError(c, err, "Failed to update set proposal")
		return
	}
	c.JSON(http.StatusOK, proposal)
}

// RejectProposal rejects one pending Agentic set proposal without creating a set.
//
//	@Summary		Reject Agentic set proposal
//	@Description	Rejects a pending Agentic set proposal. No set is created.
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	int							true	"Proposal ID"
//	@Param			body	body	rejectSetProposalRequest	false	"Rejection reason"
//	@Success		200		{object}	object{status=string}
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		404		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Router			/set-builder/proposals/{id}/reject [post]
func (h *SetBuilderHandler) RejectProposal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var body rejectSetProposalRequest
	_ = c.ShouldBindJSON(&body)
	if err := h.service.RejectProposal(c.GetUint("userId"), id, body.Reason); err != nil {
		if repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pending set proposal not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject set proposal"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}

// RegenerateProposal starts a new Agentic set proposal workflow with human feedback.
//
//	@Summary		Regenerate Agentic set proposal
//	@Description	Rejects the current pending proposal as superseded and queues a new Python workflow with review feedback.
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	int								true	"Proposal ID"
//	@Param			body	body	regenerateSetProposalRequest	true	"Regeneration feedback"
//	@Success		202		{object}	object{run=models.SetBuilderRun}
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		404		{object}	object{error=string}
//	@Failure		409		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Router			/set-builder/proposals/{id}/regenerate [post]
func (h *SetBuilderHandler) RegenerateProposal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var body regenerateSetProposalRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	run, err := h.service.RegenerateProposal(c.GetUint("userId"), id, body.Feedback)
	if err != nil {
		writeSetProposalError(c, err, "Failed to regenerate set proposal")
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"run": run})
}

// ApproveProposal approves one pending Agentic set proposal and creates the set roster.
//
//	@Summary		Approve Agentic set proposal
//	@Description	Approves a pending Agentic set proposal and transactionally creates the Agentic set with target slots.
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Proposal ID"
//	@Success		200	{object}	object{set=object}
//	@Failure		400	{object}	object{error=string}
//	@Failure		401	{object}	object{error=string}
//	@Failure		404	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Router			/set-builder/proposals/{id}/approve [post]
func (h *SetBuilderHandler) ApproveProposal(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	set, err := h.service.ApproveProposal(c.GetUint("userId"), id)
	if err != nil {
		writeSetProposalError(c, err, "Failed to approve set proposal")
		return
	}
	c.JSON(http.StatusOK, gin.H{"set": set})
}

func proposalSlotDraftsFromRequest(slots []updateSetProposalSlotRequest) []services.SetProposalSlotDraft {
	drafts := make([]services.SetProposalSlotDraft, 0, len(slots))
	for _, slot := range slots {
		var criteria *models.JSONObject
		if len(slot.Criteria) > 0 {
			jsonCriteria := models.JSONObject{}
			for key, value := range slot.Criteria {
				jsonCriteria[key] = value
			}
			criteria = &jsonCriteria
		}
		drafts = append(drafts, services.SetProposalSlotDraft{
			Label:              slot.Label,
			Criteria:           criteria,
			GroupName:          slot.Group,
			SortOrder:          slot.SortOrder,
			VerificationStatus: models.ProposalSlotVerificationStatus(slot.VerificationStatus),
			SourceNote:         slot.SourceNote,
			ValidationNote:     slot.ValidationNote,
		})
	}
	return drafts
}

func writeSetProposalError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, services.ErrSetProposalNameRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Set proposal name is required"})
	case errors.Is(err, services.ErrSetProposalNameTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Set proposal name is too long"})
	case errors.Is(err, services.ErrSetProposalSlotsRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Set proposal must include at least one slot"})
	case errors.Is(err, services.ErrSetProposalTooManySlots):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Set proposal includes too many slots"})
	case errors.Is(err, services.ErrSetProposalSlotLabelRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Set proposal slot label is required"})
	case errors.Is(err, services.ErrSetProposalVerificationStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Set proposal slot verification status is invalid"})
	case errors.Is(err, services.ErrSetProposalFeedbackRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Regeneration feedback is required"})
	case errors.Is(err, services.ErrSetProposalExpired):
		c.JSON(http.StatusConflict, gin.H{"error": "Set proposal has expired. Regenerate it before approving."})
	case repository.IsRecordNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "Pending set proposal not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
	}
}
