package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// WishlistSearchAlertHandler handles collector-owned acquisition discovery alerts.
type WishlistSearchAlertHandler struct {
	service *services.WishlistSearchAlertService
}

func NewWishlistSearchAlertHandler(service *services.WishlistSearchAlertService) *WishlistSearchAlertHandler {
	return &WishlistSearchAlertHandler{service: service}
}

// List returns search alerts owned by the authenticated collector.
//
//	@Summary		List wishlist search alerts
//	@Description	Lists acquisition discovery search alerts. This is separate from wishlist availability checking.
//	@Tags			Wishlist Search Alerts
//	@Produce		json
//	@Param			active	query	bool	false	"Filter by active state"
//	@Param			page	query	int		false	"Page number"	default(1)
//	@Param			limit	query	int		false	"Items per page"	default(20)
//	@Success		200	{object}	WishlistSearchAlertListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts [get]
func (h *WishlistSearchAlertHandler) List(c *gin.Context) {
	var active *bool
	if raw := c.Query("active"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid active filter"})
			return
		}
		active = &parsed
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	alerts, total, err := h.service.ListAlerts(c.GetUint("userId"), active, page, limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list search alerts", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts, "total": total, "page": page, "limit": limit})
}

// Create creates an acquisition discovery search alert.
//
//	@Summary		Create wishlist search alert
//	@Description	Creates a saved discovery alert without creating wishlist items or availability rows.
//	@Tags			Wishlist Search Alerts
//	@Accept			json
//	@Produce		json
//	@Param			body	body		WishlistSearchAlertRequest	true	"Alert"
//	@Success		201		{object}	models.WishlistSearchAlert
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts [post]
func (h *WishlistSearchAlertHandler) Create(c *gin.Context) {
	var req services.WishlistSearchAlertInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search alert payload"})
		return
	}
	alert, err := h.service.CreateAlert(c.GetUint("userId"), req)
	if err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusCreated, alert)
}

// Get returns one owned search alert.
//
//	@Summary		Get wishlist search alert
//	@Description	Gets one acquisition discovery alert owned by the authenticated collector.
//	@Tags			Wishlist Search Alerts
//	@Produce		json
//	@Param			alertId	path	int	true	"Alert ID"
//	@Success		200	{object}	models.WishlistSearchAlert
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId} [get]
func (h *WishlistSearchAlertHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "alertId")
	if !ok {
		return
	}
	alert, err := h.service.GetAlert(id, c.GetUint("userId"))
	if err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusOK, alert)
}

// Update updates one owned search alert.
//
//	@Summary		Update wishlist search alert
//	@Description	Updates saved discovery criteria/cadence/active metadata without changing availability state.
//	@Tags			Wishlist Search Alerts
//	@Accept			json
//	@Produce		json
//	@Param			alertId	path	int							true	"Alert ID"
//	@Param			body	body	WishlistSearchAlertRequest	true	"Alert"
//	@Success		200		{object}	models.WishlistSearchAlert
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId} [put]
func (h *WishlistSearchAlertHandler) Update(c *gin.Context) {
	id, ok := parseID(c, "alertId")
	if !ok {
		return
	}
	var req services.WishlistSearchAlertInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search alert payload"})
		return
	}
	alert, err := h.service.UpdateAlert(id, c.GetUint("userId"), req)
	if err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusOK, alert)
}

// Delete soft-deletes an owned search alert.
//
//	@Summary		Delete wishlist search alert
//	@Description	Soft-deletes a saved discovery alert while preserving history.
//	@Tags			Wishlist Search Alerts
//	@Produce		json
//	@Param			alertId	path	int	true	"Alert ID"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId} [delete]
func (h *WishlistSearchAlertHandler) Delete(c *gin.Context) {
	id, ok := parseID(c, "alertId")
	if !ok {
		return
	}
	if err := h.service.DeleteAlert(id, c.GetUint("userId")); err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// RunNow runs an active alert immediately and persists auditable results.
//
//	@Summary		Run wishlist search alert
//	@Description	Manually runs a saved discovery alert and stores candidates separately from availability checks.
//	@Tags			Wishlist Search Alerts
//	@Accept			json
//	@Produce		json
//	@Param			alertId	path	int						true	"Alert ID"
//	@Param			body	body	WishlistSearchAlertRunRequest	false	"Run request"
//	@Success		200		{object}	WishlistSearchAlertRunResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		503		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId}/run [post]
func (h *WishlistSearchAlertHandler) RunNow(c *gin.Context) {
	alertID, ok := parseID(c, "alertId")
	if !ok {
		return
	}
	var req services.RunAlertInput
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&req)
	}
	result, err := h.service.RunNow(alertID, c.GetUint("userId"), req)
	if err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListRuns lists run history for one alert.
//
//	@Summary		List wishlist search alert runs
//	@Tags			Wishlist Search Alerts
//	@Produce		json
//	@Param			alertId	path	int	true	"Alert ID"
//	@Param			page	query	int	false	"Page number"	default(1)
//	@Param			limit	query	int	false	"Items per page"	default(20)
//	@Success		200	{object}	WishlistSearchAlertRunListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId}/runs [get]
func (h *WishlistSearchAlertHandler) ListRuns(c *gin.Context) {
	alertID, ok := parseID(c, "alertId")
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	runs, total, err := h.service.ListRuns(alertID, c.GetUint("userId"), page, limit)
	if err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"runs": runs, "total": total, "page": page, "limit": limit})
}

// GetRun returns one alert run with candidates and provenance.
//
//	@Summary		Get wishlist search alert run
//	@Tags			Wishlist Search Alerts
//	@Produce		json
//	@Param			alertId	path	int	true	"Alert ID"
//	@Param			runId	path	int	true	"Run ID"
//	@Success		200	{object}	models.AlertRun
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId}/runs/{runId} [get]
func (h *WishlistSearchAlertHandler) GetRun(c *gin.Context) {
	alertID, ok := parseID(c, "alertId")
	if !ok {
		return
	}
	runID, ok := parseID(c, "runId")
	if !ok {
		return
	}
	run, err := h.service.GetRun(alertID, runID, c.GetUint("userId"))
	if err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusOK, run)
}

// ListCandidates lists alert candidates for review.
//
//	@Summary		List wishlist search alert candidates
//	@Tags			Wishlist Search Alerts
//	@Produce		json
//	@Param			alertId				path	int		true	"Alert ID"
//	@Param			state				query	string	false	"Candidate state"
//	@Param			provenanceStatus	query	string	false	"Provenance status"
//	@Param			page				query	int		false	"Page number"	default(1)
//	@Param			limit				query	int		false	"Items per page"	default(20)
//	@Success		200					{object}	WishlistSearchAlertCandidateListResponse
//	@Failure		401					{object}	ErrorResponse
//	@Failure		404					{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId}/candidates [get]
func (h *WishlistSearchAlertHandler) ListCandidates(c *gin.Context) {
	alertID, ok := parseID(c, "alertId")
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	result, err := h.service.ListCandidates(alertID, c.GetUint("userId"), c.Query("state"), c.Query("provenanceStatus"), page, limit)
	if err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// DismissCandidate dismisses an unwanted candidate.
//
//	@Summary		Dismiss wishlist search alert candidate
//	@Tags			Wishlist Search Alerts
//	@Accept			json
//	@Produce		json
//	@Param			alertId		path	int							true	"Alert ID"
//	@Param			candidateId	path	int							true	"Candidate ID"
//	@Param			body		body	WishlistSearchAlertDismissRequest	true	"Dismiss request"
//	@Success		200			{object}	models.AlertCandidate
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId}/candidates/{candidateId}/dismiss [post]
func (h *WishlistSearchAlertHandler) DismissCandidate(c *gin.Context) {
	alertID, candidateID, ok := parseAlertCandidateIDs(c)
	if !ok {
		return
	}
	var req services.DismissCandidateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid candidate review payload"})
		return
	}
	candidate, err := h.service.DismissCandidate(alertID, candidateID, c.GetUint("userId"), req)
	if err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusOK, candidate)
}

// RestoreCandidate restores a dismissed candidate.
//
//	@Summary		Restore wishlist search alert candidate
//	@Tags			Wishlist Search Alerts
//	@Produce		json
//	@Param			alertId		path	int	true	"Alert ID"
//	@Param			candidateId	path	int	true	"Candidate ID"
//	@Success		200			{object}	models.AlertCandidate
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId}/candidates/{candidateId}/restore [post]
func (h *WishlistSearchAlertHandler) RestoreCandidate(c *gin.Context) {
	alertID, candidateID, ok := parseAlertCandidateIDs(c)
	if !ok {
		return
	}
	candidate, err := h.service.RestoreCandidate(alertID, candidateID, c.GetUint("userId"))
	if err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusOK, candidate)
}

// ConvertCandidate explicitly converts a candidate to one wishlist coin.
//
//	@Summary		Convert wishlist search alert candidate
//	@Tags			Wishlist Search Alerts
//	@Accept			json
//	@Produce		json
//	@Param			alertId		path	int									true	"Alert ID"
//	@Param			candidateId	path	int									true	"Candidate ID"
//	@Param			body		body	WishlistSearchAlertConvertRequest	true	"Convert request"
//	@Success		201			{object}	WishlistSearchAlertConvertResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId}/candidates/{candidateId}/convert [post]
func (h *WishlistSearchAlertHandler) ConvertCandidate(c *gin.Context) {
	alertID, candidateID, ok := parseAlertCandidateIDs(c)
	if !ok {
		return
	}
	var req services.ConvertCandidateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid candidate conversion payload"})
		return
	}
	result, err := h.service.ConvertCandidate(alertID, candidateID, c.GetUint("userId"), req)
	if err != nil {
		if errors.Is(err, services.ErrWishlistSearchAlertDuplicate) && result != nil {
			c.JSON(http.StatusConflict, result)
			return
		}
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// AdjustCriteria updates alert criteria from review context.
//
//	@Summary		Adjust wishlist search alert criteria
//	@Tags			Wishlist Search Alerts
//	@Accept			json
//	@Produce		json
//	@Param			alertId	path	int									true	"Alert ID"
//	@Param			body	body	WishlistSearchAlertCriteriaAdjustRequest	true	"Criteria adjustment"
//	@Success		200		{object}	models.WishlistSearchAlert
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/search-alerts/{alertId}/criteria-adjustments [post]
func (h *WishlistSearchAlertHandler) AdjustCriteria(c *gin.Context) {
	alertID, ok := parseID(c, "alertId")
	if !ok {
		return
	}
	var req services.AdjustCriteriaInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid criteria adjustment payload"})
		return
	}
	alert, err := h.service.AdjustCriteria(alertID, c.GetUint("userId"), req)
	if err != nil {
		respondWishlistSearchAlertError(c, err)
		return
	}
	c.JSON(http.StatusOK, alert)
}

func respondWishlistSearchAlertError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrWishlistSearchAlertNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Search alert not found"})
	case errors.Is(err, services.ErrWishlistSearchAlertNoCriteria):
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one meaningful search criterion is required"})
	case errors.Is(err, services.ErrWishlistSearchAlertPriceRange):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price range"})
	case errors.Is(err, services.ErrWishlistSearchAlertDateRange):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date range"})
	case errors.Is(err, services.ErrWishlistSearchAlertCadence):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported cadence"})
	case errors.Is(err, services.ErrWishlistSearchAlertSourceFilter):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source or domain filter"})
	case errors.Is(err, services.ErrWishlistSearchAlertDisabled), errors.Is(err, services.ErrWishlistSearchAlertCandidateState), errors.Is(err, services.ErrWishlistSearchAlertConversion):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search alert action"})
	case errors.Is(err, services.ErrWishlistSearchAlertRunLimited):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Search alert run limit exceeded"})
	case errors.Is(err, services.ErrWishlistSearchAlertAgent):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Search alert discovery is unavailable"})
	case errors.Is(err, services.ErrWishlistSearchAlertDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "Duplicate wishlist warning must be acknowledged"})
	case errors.Is(err, services.ErrWishlistSearchAlertStringTooLong), errors.Is(err, services.ErrWishlistSearchAlertInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		respondError(c, http.StatusInternalServerError, "Failed to process search alert", err)
	}
}

func parseAlertCandidateIDs(c *gin.Context) (uint, uint, bool) {
	alertID, ok := parseID(c, "alertId")
	if !ok {
		return 0, 0, false
	}
	candidateID, ok := parseID(c, "candidateId")
	if !ok {
		return 0, 0, false
	}
	return alertID, candidateID, true
}
