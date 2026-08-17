package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// DeepIdentificationHandler exposes the REST surface for the deep agentic
// coin identification job domain (344-deep-agentic-coin-identification,
// Phase 5). It is a sibling to AIJobHandler; models.AIJob and its endpoints
// are untouched.
type DeepIdentificationHandler struct {
	service     *services.DeepIdentificationService
	settingsSvc *services.SettingsService
	logger      *services.Logger
	proposalSvc *services.DeepIdentificationProposalService
}

// NewDeepIdentificationHandler constructs the handler, following the
// repo -> service -> handler DI pattern used elsewhere (main.go:246-249).
func NewDeepIdentificationHandler(service *services.DeepIdentificationService, settingsSvc *services.SettingsService, logger *services.Logger) *DeepIdentificationHandler {
	return &DeepIdentificationHandler{service: service, settingsSvc: settingsSvc, logger: logger}
}

// WithProposalSupport wires in the confirm-gated report->write bridge
// (US4). Proposal/apply endpoints 500 if this was never called - every
// production wiring in main.go calls it immediately after construction.
func (h *DeepIdentificationHandler) WithProposalSupport(proposalSvc *services.DeepIdentificationProposalService) *DeepIdentificationHandler {
	h.proposalSvc = proposalSvc
	return h
}

// deepProviderIDs is the closed vocabulary accepted for a provider
// override (contract ProviderId enum). Unknown values are rejected with
// 400 rather than silently ignored.
var deepProviderIDs = map[string]bool{
	"nomisma": true,
	"numista": true,
	"ngc":     true,
	"ocre":    true,
	"rpc":     true,
}

func validateDeepProviderIDs(providers []string) bool {
	for _, p := range providers {
		if !deepProviderIDs[p] {
			return false
		}
	}
	return true
}

// deepJobDTO is the wire shape for a DeepIdentificationJob
// (contracts/deep-identification.openapi.yaml `DeepJob`). It exists
// because the stored model keeps ReportJSON/ProposalJSON/RequestedProviders
// as flat text columns that must be reshaped into structured JSON on the
// way out.
type deepJobDTO struct {
	ID                 uint       `json:"id"`
	CoinID             *uint      `json:"coinId,omitempty"`
	Source             string     `json:"source"`
	Status             string     `json:"status"`
	PartialSuccess     bool       `json:"partialSuccess"`
	SelectedProviders  []string   `json:"selectedProviders,omitempty"`
	RequestedProviders []string   `json:"requestedProviders,omitempty"`
	RouterRationale    string     `json:"routerRationale,omitempty"`
	RetryOfJobID       *uint      `json:"retryOfJobId,omitempty"`
	CancelRequested    bool       `json:"cancelRequested"`
	LastSeq            int64      `json:"lastSeq"`
	EventsAvailable    bool       `json:"eventsAvailable"`
	FailureCode        string     `json:"failureCode,omitempty"`
	FailureMessage     string     `json:"failureMessage,omitempty"`
	AppliedCoinID      *uint      `json:"appliedCoinId,omitempty"`
	AppliedDraftID     *uint      `json:"appliedDraftId,omitempty"`
	AppliedAt          *time.Time `json:"appliedAt,omitempty"`
	StartedAt          *time.Time `json:"startedAt,omitempty"`
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
	ExpiresAt          time.Time  `json:"expiresAt"`
	CreatedAt          time.Time  `json:"createdAt"`
}

// deepJobEnvelope is the wire shape for `DeepJobEnvelope`.
type deepJobEnvelope struct {
	Job      deepJobDTO      `json:"job"`
	Reused   bool            `json:"reused,omitempty"`
	Report   json.RawMessage `json:"report,omitempty"`
	Proposal json.RawMessage `json:"proposal,omitempty"`
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func toDeepJobDTO(job *models.DeepIdentificationJob) deepJobDTO {
	return deepJobDTO{
		ID:                 job.ID,
		CoinID:             job.CoinID,
		Source:             string(job.Source),
		Status:             string(job.Status),
		PartialSuccess:     job.PartialSuccess,
		SelectedProviders:  splitCSV(job.SelectedProviders),
		RequestedProviders: splitCSV(job.RequestedProviders),
		RouterRationale:    job.RouterRationale,
		RetryOfJobID:       job.RetryOfJobID,
		CancelRequested:    job.CancelRequestedAt != nil,
		LastSeq:            job.LastSeq,
		EventsAvailable:    job.EventsPrunedAt == nil,
		FailureCode:        job.FailureCode,
		FailureMessage:     job.FailureMessage,
		AppliedCoinID:      job.AppliedCoinID,
		AppliedDraftID:     job.AppliedDraftID,
		AppliedAt:          job.AppliedAt,
		StartedAt:          job.StartedAt,
		CompletedAt:        job.CompletedAt,
		ExpiresAt:          job.ExpiresAt,
		CreatedAt:          job.CreatedAt,
	}
}

func toDeepJobEnvelope(job *models.DeepIdentificationJob, reused bool) deepJobEnvelope {
	env := deepJobEnvelope{Job: toDeepJobDTO(job), Reused: reused}
	if job.ReportJSON != "" {
		env.Report = json.RawMessage(job.ReportJSON)
	}
	if job.ProposalJSON != "" {
		env.Proposal = json.RawMessage(job.ProposalJSON)
	}
	return env
}

// deepIdentificationEnabled reports the live feature flag. Handlers gate
// job-creation and cancel/retry actions on it (FR-008); read-only GET of an
// already-running job is intentionally NOT gated so in-flight work is never
// stranded by an admin disabling the flag mid-run.
func (h *DeepIdentificationHandler) deepIdentificationEnabled() bool {
	return h.settingsSvc.GetDeepIdentificationSettings().Enabled
}

// deepCapabilityResponse is the wire shape for the Deep Analysis capability
// probe (contracts/deep-identification.openapi.yaml `Capability`).
type deepCapabilityResponse struct {
	Enabled   bool     `json:"enabled"`
	Providers []string `json:"providers"`
}

// Capability reports whether Deep Analysis is currently available to the
// authenticated user (FR-008 feature flag). It exposes only the boolean
// gate derived from the admin feature flag - never the underlying admin
// settings - so normal-user UI can hide/disable the Deep Analysis entry
// point while the backend stays authoritative (job creation still 403s
// when disabled). Read-only and safe for any authenticated user.
//
//	@Summary		Report whether Deep Analysis is enabled for the current user
//	@Description	Returns the Deep Analysis feature-flag state so the client can hide or disable the entry point. The backend remains authoritative; job creation is independently gated.
//	@Tags			Deep Identification
//	@Produce		json
//	@Success		200	{object}	deepCapabilityResponse
//	@Router			/deep-identification/capability [get]
func (h *DeepIdentificationHandler) Capability(c *gin.Context) {
	settings := h.settingsSvc.GetDeepIdentificationSettings()
	providers := []string{"nomisma", "numista"}
	if settings.OCREEnabled {
		providers = append(providers, "ocre")
	}
	c.JSON(http.StatusOK, deepCapabilityResponse{Enabled: settings.Enabled, Providers: providers})
}

// CreateJob starts a Deep Analysis job from multipart intake.
//
//	@Summary		Start a Deep Analysis job
//	@Description	Accepts obverse/reverse coin images (uploaded or reused from a saved coin), optional notes, optional ephemeral hint images, and an optional provider override. Idempotent duplicate submissions (same owner, coin, image content hashes, notes, provider override) return the existing in-flight job with reused=true. A distinct submission that would exceed the per-user concurrency limit is refused with 409 job_at_capacity rather than being matched to an unrelated in-flight job.
//	@Tags			Deep Identification
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			coinId		formData	int		false	"Saved coin to analyse; omit for new-coin intake"
//	@Param			obverse		formData	file	false	"Obverse image; required unless the saved coin already has one"
//	@Param			reverse		formData	file	false	"Reverse image; required unless the saved coin already has one"
//	@Param			hints		formData	file	false	"Ephemeral hint images (max 3)"
//	@Param			notes		formData	string	false	"Free-text notes (max 2000 chars)"
//	@Param			providers	formData	string	false	"Comma-separated provider override"
//	@Success		202	{object}	deepJobEnvelope
//	@Success		200	{object}	deepJobEnvelope
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		413	{object}	ErrorResponse
//	@Failure		415	{object}	ErrorResponse
//	@Failure		422	{object}	ErrorResponse
//	@Failure		429	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		503	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/deep-identification/jobs [post]
func (h *DeepIdentificationHandler) CreateJob(c *gin.Context) {
	userID := c.GetUint("userId")
	if !h.deepIdentificationEnabled() {
		respondError(c, http.StatusForbidden, "Deep Analysis is not currently enabled", nil)
		return
	}

	in := services.CreateJobInput{UserID: userID}

	if raw := c.PostForm("coinId"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			respondError(c, http.StatusBadRequest, "Invalid coinId", err)
			return
		}
		coinID := uint(id)
		in.CoinID = &coinID
	}
	in.Notes = c.PostForm("notes")
	if len(in.Notes) > 2000 {
		respondError(c, http.StatusBadRequest, "notes must be at most 2000 characters", nil)
		return
	}
	if raw := c.PostForm("providers"); raw != "" {
		providers := splitCSV(raw)
		if !validateDeepProviderIDs(providers) {
			respondError(c, http.StatusBadRequest, "Unknown provider id in providers", nil)
			return
		}
		in.RequestedProviders = providers
	}

	obverseBytes, obverseFilename, status, err := readOptionalDeepFile(c, "obverse")
	if err != nil {
		respondError(c, status, err.Error(), err)
		return
	}
	in.ObverseBytes, in.ObverseFilename = obverseBytes, obverseFilename

	reverseBytes, reverseFilename, status, err := readOptionalDeepFile(c, "reverse")
	if err != nil {
		respondError(c, status, err.Error(), err)
		return
	}
	in.ReverseBytes, in.ReverseFilename = reverseBytes, reverseFilename

	hints, status, err := readDeepHintFiles(c)
	if err != nil {
		respondError(c, status, err.Error(), err)
		return
	}
	in.Hints = hints

	// hint_image_in_coin_role (spec.md open question, data-model.md §5):
	// reject a hint image that is byte-identical to the submitted obverse
	// or reverse upload - it indicates the same file was mistakenly
	// attached under both a coin-face role and the ephemeral hints array,
	// rather than genuinely being two distinct images.
	if err := validateHintsDistinctFromCoinFaces(in); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error(), "code": "hint_image_in_coin_role"})
		return
	}

	job, reused, err := h.service.CreateJobFromIntake(in)
	if err != nil {
		h.respondDeepJobError(c, err)
		return
	}
	httpStatus := http.StatusAccepted
	if reused {
		httpStatus = http.StatusOK
	}
	c.JSON(httpStatus, toDeepJobEnvelope(job, reused))
}

// validateHintsDistinctFromCoinFaces rejects a hint upload that is
// byte-identical to the obverse or reverse upload in the same request
// (422 hint_image_in_coin_role).
func validateHintsDistinctFromCoinFaces(in services.CreateJobInput) error {
	for _, h := range in.Hints {
		if len(in.ObverseBytes) > 0 && bytes.Equal(h.Bytes, in.ObverseBytes) {
			return errors.New("a hint image must not duplicate the obverse image")
		}
		if len(in.ReverseBytes) > 0 && bytes.Equal(h.Bytes, in.ReverseBytes) {
			return errors.New("a hint image must not duplicate the reverse image")
		}
	}
	return nil
}

// readOptionalDeepFile reads a single optional multipart file field,
// returning (nil, "", 200, nil) when the field was not supplied at all
// (the caller falls back to a saved coin's existing image, if any).
func readOptionalDeepFile(c *gin.Context, field string) ([]byte, string, int, error) {
	fileHeader, err := c.FormFile(field)
	if err != nil {
		return nil, "", http.StatusOK, nil
	}
	if fileHeader.Size > services.MaxImageUploadBytes {
		return nil, "", http.StatusRequestEntityTooLarge, services.ErrImageTooLarge
	}
	f, err := fileHeader.Open()
	if err != nil {
		return nil, "", http.StatusBadRequest, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, "", http.StatusBadRequest, err
	}
	if err := services.ValidateImageData(data); err != nil {
		if errors.Is(err, services.ErrImageTooLarge) {
			return nil, "", http.StatusRequestEntityTooLarge, err
		}
		return nil, "", http.StatusUnsupportedMediaType, err
	}
	return data, fileHeader.Filename, http.StatusOK, nil
}

func readDeepHintFiles(c *gin.Context) ([]services.CreateJobHintInput, int, error) {
	form, err := c.MultipartForm()
	if err != nil || form == nil {
		return nil, http.StatusOK, nil
	}
	files := form.File["hints"]
	if len(files) == 0 {
		return nil, http.StatusOK, nil
	}
	if len(files) > services.MaxDeepIdentificationHintArtifacts {
		return nil, http.StatusBadRequest, errors.New("at most 3 hint images are allowed")
	}
	hints := make([]services.CreateJobHintInput, 0, len(files))
	for _, fileHeader := range files {
		if fileHeader.Size > services.MaxImageUploadBytes {
			return nil, http.StatusRequestEntityTooLarge, services.ErrImageTooLarge
		}
		f, err := fileHeader.Open()
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		if err := services.ValidateImageData(data); err != nil {
			if errors.Is(err, services.ErrImageTooLarge) {
				return nil, http.StatusRequestEntityTooLarge, err
			}
			return nil, http.StatusUnsupportedMediaType, err
		}
		hints = append(hints, services.CreateJobHintInput{Filename: fileHeader.Filename, Bytes: data})
	}
	return hints, http.StatusOK, nil
}

// ListJobs returns the authenticated owner's Deep Analysis jobs.
//
//	@Summary		List the owner's Deep Analysis jobs
//	@Tags			Deep Identification
//	@Produce		json
//	@Param			coinId		query	int		false	"Filter by coin id"
//	@Param			activeOnly	query	bool	false	"Only queued/running jobs"
//	@Param			status		query	string	false	"Filter by exact job status"
//	@Param			limit		query	int		false	"Page size (default 20, max 100)"
//	@Param			cursor		query	string	false	"Opaque pagination cursor from a previous response"
//	@Success		200	{object}	deepJobListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/deep-identification/jobs [get]
func (h *DeepIdentificationHandler) ListJobs(c *gin.Context) {
	userID := c.GetUint("userId")
	filters := repository.DeepJobListFilters{}
	if raw := c.Query("coinId"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			respondError(c, http.StatusBadRequest, "Invalid coinId", err)
			return
		}
		coinID := uint(id)
		filters.CoinID = &coinID
	}
	if c.Query("activeOnly") == "true" {
		filters.ActiveOnly = true
	}
	if raw := c.Query("status"); raw != "" {
		switch models.DeepJobStatus(raw) {
		case models.DeepJobStatusQueued, models.DeepJobStatusRunning, models.DeepJobStatusCompleted,
			models.DeepJobStatusPartial, models.DeepJobStatusFailed, models.DeepJobStatusCancelled:
			filters.Status = models.DeepJobStatus(raw)
		default:
			respondError(c, http.StatusBadRequest, "Invalid status", nil)
			return
		}
	}
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			respondError(c, http.StatusBadRequest, "Invalid limit", nil)
			return
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}
	if raw := c.Query("cursor"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			respondError(c, http.StatusBadRequest, "Invalid cursor", err)
			return
		}
		beforeID := uint(id)
		filters.BeforeID = &beforeID
	}
	filters.Limit = limit + 1 // fetch one extra to know whether a next page exists

	jobs, err := h.service.ListJobs(userID, filters)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list Deep Analysis jobs", err)
		return
	}
	var nextCursor *string
	if len(jobs) > limit {
		jobs = jobs[:limit]
		cursor := strconv.FormatUint(uint64(jobs[len(jobs)-1].ID), 10)
		nextCursor = &cursor
	}
	dtos := make([]deepJobDTO, len(jobs))
	for i := range jobs {
		dtos[i] = toDeepJobDTO(&jobs[i])
	}
	c.JSON(http.StatusOK, deepJobListResponse{Jobs: dtos, NextCursor: nextCursor})
}

type deepJobListResponse struct {
	Jobs       []deepJobDTO `json:"jobs"`
	NextCursor *string      `json:"nextCursor,omitempty"`
}

// GetJob returns a single owner-scoped Deep Analysis job with its report
// and proposal, when terminal.
//
//	@Summary		Get a Deep Analysis job
//	@Tags			Deep Identification
//	@Produce		json
//	@Param			id	path	int	true	"Job ID"
//	@Success		200	{object}	deepJobEnvelope
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/deep-identification/jobs/{id} [get]
func (h *DeepIdentificationHandler) GetJob(c *gin.Context) {
	userID := c.GetUint("userId")
	jobID, ok := parseID(c, "id")
	if !ok {
		return
	}
	job, err := h.service.GetJob(jobID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			respondError(c, http.StatusNotFound, "Deep Analysis job not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to get Deep Analysis job", err)
		return
	}
	c.JSON(http.StatusOK, toDeepJobEnvelope(job, false))
}

// Cancel requests cancellation of a running (or still-queued) job.
//
//	@Summary		Cancel a Deep Analysis job
//	@Tags			Deep Identification
//	@Produce		json
//	@Param			id	path	int	true	"Job ID"
//	@Success		202	{object}	deepJobEnvelope
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	deepJobEnvelope
//	@Security		BearerAuth
//	@Router			/deep-identification/jobs/{id}/cancel [post]
func (h *DeepIdentificationHandler) Cancel(c *gin.Context) {
	userID := c.GetUint("userId")
	jobID, ok := parseID(c, "id")
	if !ok {
		return
	}
	err := h.service.RequestCancel(jobID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDeepJobNotFound):
			respondError(c, http.StatusNotFound, "Deep Analysis job not found", nil)
		case errors.Is(err, services.ErrDeepJobNotCancellable):
			// Already terminal: report the settled state exactly as every
			// other observer would see it (FR-019).
			job, getErr := h.service.GetJob(jobID, userID)
			if getErr != nil {
				respondError(c, http.StatusNotFound, "Deep Analysis job not found", nil)
				return
			}
			c.JSON(http.StatusConflict, toDeepJobEnvelope(job, false))
		default:
			respondError(c, http.StatusInternalServerError, "Failed to cancel Deep Analysis job", err)
		}
		return
	}
	job, err := h.service.GetJob(jobID, userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to reload cancelled job", err)
		return
	}
	c.JSON(http.StatusAccepted, toDeepJobEnvelope(job, false))
}

// deepRetryRequest is the optional JSON body for a retry request.
type deepRetryRequest struct {
	Notes     *string  `json:"notes,omitempty"`
	Providers []string `json:"providers,omitempty"`
}

// Retry creates a new job linked to a terminal job via retryOfJobId.
//
//	@Summary		Retry a Deep Analysis job
//	@Tags			Deep Identification
//	@Accept			json
//	@Produce		json
//	@Param			id		path	int					true	"Job ID"
//	@Param			request	body	deepRetryRequest	false	"Optional notes/provider override"
//	@Success		202	{object}	deepJobEnvelope
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/deep-identification/jobs/{id}/retry [post]
func (h *DeepIdentificationHandler) Retry(c *gin.Context) {
	userID := c.GetUint("userId")
	jobID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if !h.deepIdentificationEnabled() {
		respondError(c, http.StatusForbidden, "Deep Analysis is not currently enabled", nil)
		return
	}

	var req deepRetryRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "Invalid request body", err)
			return
		}
	}
	if req.Providers != nil && !validateDeepProviderIDs(req.Providers) {
		respondError(c, http.StatusBadRequest, "Unknown provider id in providers", nil)
		return
	}

	job, reused, err := h.service.RetryJob(jobID, userID, req.Notes, req.Providers)
	if err != nil {
		h.respondDeepJobError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, toDeepJobEnvelope(job, reused))
}

// respondDeepJobError maps DeepIdentificationService job-orchestration
// errors to the HTTP status/code vocabulary in
// contracts/deep-identification.openapi.yaml.
func (h *DeepIdentificationHandler) respondDeepJobError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrDeepJobDisabled):
		c.JSON(http.StatusForbidden, gin.H{"error": "Deep Analysis is not currently enabled"})
	case errors.Is(err, services.ErrDeepJobMissingObverse):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "An obverse image is required", "code": "missing_obverse"})
	case errors.Is(err, services.ErrDeepJobMissingReverse):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "A reverse image is required", "code": "missing_reverse"})
	case errors.Is(err, services.ErrDeepArtifactMissingCoin):
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
	case errors.Is(err, services.ErrDeepArtifactRoleExists):
		c.JSON(http.StatusBadRequest, gin.H{"error": "A role for this job already has an image"})
	case errors.Is(err, services.ErrDeepArtifactHintLimit):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hint image limit reached"})
	case errors.Is(err, services.ErrDeepJobNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Deep Analysis job not found"})
	case errors.Is(err, services.ErrDeepJobNotTerminal):
		c.JSON(http.StatusConflict, gin.H{"error": "Job is not yet terminal"})
	case errors.Is(err, services.ErrDeepJobRetryDepth):
		c.JSON(http.StatusConflict, gin.H{"error": "Retry depth limit reached"})
	case errors.Is(err, services.ErrDeepJobQueueFull):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Deep Analysis queue is full, try again shortly"})
	case errors.Is(err, services.ErrDeepJobAtCapacity):
		c.JSON(http.StatusConflict, gin.H{"error": "An analysis is already running. Wait for it to finish or cancel it.", "code": "job_at_capacity"})
	case repository.IsRecordNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	default:
		h.logger.Error("deep-identification", "Deep Analysis job request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process Deep Analysis job request"})
	}
}

// deepProposalPatchRequest is the PATCH .../proposal request body.
type deepProposalPatchRequest struct {
	Fields map[string]struct {
		OwnerValue json.RawMessage `json:"ownerValue"`
		Accepted   *bool           `json:"accepted"`
	} `json:"fields"`
}

// UpdateProposal saves owner edits/accept-reject decisions on a job's
// proposal (T110). It never writes coin/draft data - only the job's own
// ProposalJSON column (FR-031/FR-032).
//
//	@Summary		Save owner edits and per-field decisions on a Deep Analysis proposal
//	@Tags			Deep Identification
//	@Accept			json
//	@Produce		json
//	@Param			id		path	int							true	"Job ID"
//	@Param			request	body	deepProposalPatchRequest	true	"Field edits"
//	@Success		200	{object}	json.RawMessage
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/deep-identification/jobs/{id}/proposal [patch]
func (h *DeepIdentificationHandler) UpdateProposal(c *gin.Context) {
	userID := c.GetUint("userId")
	jobID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req deepProposalPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	edits := make(map[string]services.DeepProposalFieldEdit, len(req.Fields))
	for name, raw := range req.Fields {
		edit := services.DeepProposalFieldEdit{Accepted: raw.Accepted}
		if len(raw.OwnerValue) > 0 && string(raw.OwnerValue) != "null" {
			var v any
			if err := json.Unmarshal(raw.OwnerValue, &v); err != nil {
				respondError(c, http.StatusBadRequest, "Invalid ownerValue for field "+name, err)
				return
			}
			edit.OwnerValue = v
			edit.OwnerValueSet = true
		} else if len(raw.OwnerValue) > 0 {
			edit.OwnerValueSet = true
			edit.OwnerValue = nil
		}
		edits[name] = edit
	}
	doc, err := h.proposalSvc.UpdateProposal(jobID, userID, edits)
	if err != nil {
		h.respondDeepProposalError(c, err)
		return
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to encode proposal", err)
		return
	}
	c.Data(http.StatusOK, "application/json", encoded)
}

// deepApplyRequest is the POST .../apply request body.
type deepApplyRequest struct {
	Target string   `json:"target" binding:"required,oneof=draft coin wishlist"`
	Fields []string `json:"fields,omitempty"`
}

// deepApplyResponse mirrors the `apply` 200 response.
type deepApplyResponse struct {
	JobID         uint      `json:"jobId"`
	DraftID       *uint     `json:"draftId,omitempty"`
	CoinID        *uint     `json:"coinId,omitempty"`
	AppliedFields []string  `json:"appliedFields"`
	AppliedAt     time.Time `json:"appliedAt"`
}

// normalizeDeepApplyTarget validates the requested apply destination
// through a closed switch, rejecting anything not explicitly known (T073).
// The `binding:"oneof=draft coin wishlist"` tag on deepApplyRequest already
// rejects unrecognized targets during ShouldBindJSON; this is the second,
// explicit gate the service's own closed switch in Apply expects its caller
// to have already normalized through, so a future target added to one
// switch without the other fails loudly instead of silently.
func normalizeDeepApplyTarget(target string) (string, bool) {
	switch target {
	case "draft", "coin", "wishlist":
		return target, true
	default:
		return "", false
	}
}

// ApplyProposal confirms the proposal through an existing Go-owned write
// path (T111, FR-031/FR-033): "draft" seeds a QuickCaptureDraft (existing
// promote flow finishes the job); "coin" patches the saved coin via
// CoinService.UpdateCoinWithFields(source="deep_identification"); "wishlist"
// (T072/T073, FR-027) creates a new wishlist models.Coin via
// CoinService.CreateCoin. This handler performs no write of its own.
//
//	@Summary		Confirm a Deep Analysis proposal through the existing write path
//	@Tags			Deep Identification
//	@Accept			json
//	@Produce		json
//	@Param			id		path	int					true	"Job ID"
//	@Param			request	body	deepApplyRequest	true	"Apply target (draft, coin, or wishlist) and optional field subset"
//	@Success		200	{object}	deepApplyResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/deep-identification/jobs/{id}/apply [post]
func (h *DeepIdentificationHandler) ApplyProposal(c *gin.Context) {
	userID := c.GetUint("userId")
	jobID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req deepApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	target, ok := normalizeDeepApplyTarget(req.Target)
	if !ok {
		respondError(c, http.StatusBadRequest, "Invalid apply target", nil)
		return
	}
	result, err := h.proposalSvc.Apply(jobID, userID, target, req.Fields)
	if err != nil {
		h.respondDeepProposalError(c, err)
		return
	}
	c.JSON(http.StatusOK, deepApplyResponse{
		JobID:         result.JobID,
		DraftID:       result.DraftID,
		CoinID:        result.CoinID,
		AppliedFields: result.AppliedFields,
		AppliedAt:     result.AppliedAt,
	})
}

// respondDeepProposalError maps DeepIdentificationProposalService errors to
// the HTTP status/code vocabulary in
// contracts/deep-identification.openapi.yaml (proposal/apply endpoints).
func (h *DeepIdentificationHandler) respondDeepProposalError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrDeepProposalNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Deep Analysis job not found"})
	case errors.Is(err, services.ErrDeepProposalAlreadyApplied):
		c.JSON(http.StatusConflict, gin.H{"error": "Proposal has already been applied", "code": "already_applied"})
	case errors.Is(err, services.ErrDeepProposalSourceMissing):
		c.JSON(http.StatusConflict, gin.H{"error": "Source coin no longer exists", "code": "source_coin_missing"})
	case errors.Is(err, services.ErrDeepProposalNotReady):
		c.JSON(http.StatusConflict, gin.H{"error": "Job has no proposal to edit or apply yet"})
	case errors.Is(err, services.ErrDeepProposalTargetMismatch):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Apply target does not match this job's source"})
	case errors.Is(err, services.ErrDeepProposalFieldNotAllowed):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrDeepProposalNoAcceptedFields):
		c.JSON(http.StatusBadRequest, gin.H{"error": "No accepted fields to apply"})
	case errors.Is(err, services.ErrDeepProposalInvalidCatalogReferences):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "invalid_catalog_references"})
	case repository.IsRecordNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	default:
		h.logger.Error("deep-identification", "Deep Analysis proposal request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process Deep Analysis proposal request"})
	}
}
