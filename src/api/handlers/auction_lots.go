package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// AuctionLotHandler handles HTTP requests for auction lot operations.
type AuctionLotHandler struct {
	repo        *repository.AuctionLotRepository
	svc         *services.AuctionLotService
	userRepo    *repository.UserRepository
	nbSvc       *services.NumisBidsService
	cngSvc      *services.CNGAuctionService
	logger      *services.Logger
	credentials *services.CredentialEncryptionService
}

// NewAuctionLotHandler creates a new AuctionLotHandler.
func NewAuctionLotHandler(repo *repository.AuctionLotRepository, svc *services.AuctionLotService, userRepo *repository.UserRepository, nbSvc *services.NumisBidsService, cngSvc *services.CNGAuctionService, logger *services.Logger, credentialSvc ...*services.CredentialEncryptionService) *AuctionLotHandler {
	credentials := services.NewDisabledCredentialEncryptionService()
	if len(credentialSvc) > 0 && credentialSvc[0] != nil {
		credentials = credentialSvc[0]
	}
	return &AuctionLotHandler{repo: repo, svc: svc, userRepo: userRepo, nbSvc: nbSvc, cngSvc: cngSvc, logger: logger, credentials: credentials}
}

// List returns a paginated list of auction lots for the authenticated user.
//
//	@Summary		List auction lots
//	@Description	Returns a paginated list of auction lots belonging to the authenticated user.
//	@Tags			Auctions
//	@Produce		json
//	@Param			status	query		string	false	"Filter by status (watching, bidding, won, lost, passed)"
//	@Param			source	query		string	false	"Filter by auction source (numisbids, cng)"
//	@Param			search	query		string	false	"Search across title, description, auction house"
//	@Param			page	query		int		false	"Page number"	default(1)
//	@Param			limit	query		int		false	"Items per page"	default(50)
//	@Param			sort	query		string	false	"Sort field"	default(updated_at)
//	@Param			order	query		string	false	"Sort order"	default(desc)
//	@Success		200		{object}	AuctionLotListResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions [get]
func (h *AuctionLotHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	filters := repository.AuctionLotListFilters{
		Status:    c.Query("status"),
		Search:    c.Query("search"),
		Source:    c.Query("source"),
		SortField: c.DefaultQuery("sort", "updated_at"),
		SortOrder: c.DefaultQuery("order", "desc"),
		Page:      page,
		Limit:     limit,
	}

	lots, total, err := h.repo.List(userID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list auction lots"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"lots": lots, "total": total})
}

// Counts returns per-status counts for the authenticated user's auction lots.
//
//	@Summary		Get auction lot counts by status
//	@Description	Returns a map of status → count for the authenticated user, optionally scoped to one auction source.
//	@Tags			Auctions
//	@Produce		json
//	@Param			source	query	string	false	"Filter by auction source (numisbids, cng)"
//	@Success		200	{object}	map[string]int64
//	@Security		BearerAuth
//	@Router			/auctions/counts [get]
func (h *AuctionLotHandler) Counts(c *gin.Context) {
	userID := c.GetUint("userId")
	source := models.AuctionSource(c.Query("source"))
	var counts map[string]int64
	var err error
	if source != "" {
		if source != models.AuctionSourceNumisBids && source != models.AuctionSourceCNG {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported auction source"})
			return
		}
		counts, err = h.repo.CountByStatusForSource(userID, source)
	} else {
		counts, err = h.repo.CountByStatus(userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count auction lots"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"counts": counts})
}

// Get returns a single auction lot by ID.
//
//	@Summary		Get auction lot
//	@Description	Returns a single auction lot by ID.
//	@Tags			Auctions
//	@Produce		json
//	@Param			id	path		int	true	"Auction lot ID"
//	@Success		200	{object}	models.AuctionLot
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/{id} [get]
func (h *AuctionLotHandler) Get(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	lot, err := h.repo.GetByID(uint(id), userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Auction lot not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get auction lot"})
		return
	}

	c.JSON(http.StatusOK, lot)
}

// Create adds a new auction lot for tracking.
//
//	@Summary		Create auction lot
//	@Description	Creates a new auction lot for tracking.
//	@Tags			Auctions
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.AuctionLot	true	"Auction lot data"
//	@Success		201		{object}	models.AuctionLot
//	@Failure		400		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions [post]
func (h *AuctionLotHandler) Create(c *gin.Context) {
	userID := c.GetUint("userId")
	var lot models.AuctionLot
	if err := c.ShouldBindJSON(&lot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	lot.UserID = userID
	if lot.Status == "" {
		lot.Status = models.AuctionStatusWatching
	}

	if err := h.repo.Create(&lot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create auction lot"})
		return
	}

	c.JSON(http.StatusCreated, lot)
}

// Update modifies an existing auction lot.
//
//	@Summary		Update auction lot
//	@Description	Updates an existing auction lot.
//	@Tags			Auctions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Auction lot ID"
//	@Param			body	body		models.AuctionLot	true	"Updated lot data"
//	@Success		200		{object}	models.AuctionLot
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/{id} [put]
//
// UpdateLotRequest is the narrow set of fields a user may edit on an auction lot.
// Fields like UserID, CoinID, EventID, Status, and computed fields are intentionally
// excluded — those have dedicated endpoints with their own authorization rules.
type UpdateLotRequest struct {
	Title          *string    `json:"title"`
	NumisBidsURL   *string    `json:"numisBidsUrl"`
	AuctionHouse   *string    `json:"auctionHouse"`
	SaleName       *string    `json:"saleName"`
	LotNumber      *int       `json:"lotNumber"`
	SaleDate       *time.Time `json:"saleDate"`
	AuctionEndTime *time.Time `json:"auctionEndTime"`
	Description    *string    `json:"description"`
	Notes          *string    `json:"notes"`
	Category       *string    `json:"category"`
	Estimate       *float64   `json:"estimate"`
	InitialBid     *float64   `json:"initialBid"`
	CurrentBid     *float64   `json:"currentBid"`
	MaxBid         *float64   `json:"maxBid"`
	WinningBid     *float64   `json:"winningBid"`
	Currency       *string    `json:"currency"`
}

// Update applies editable auction lot fields for the authenticated user.
//
//	@Summary		Update auction lot
//	@Description	Updates editable fields on an auction lot owned by the authenticated user.
//	@Tags			Auctions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Auction lot ID"
//	@Param			body	body		UpdateLotRequest	true	"Updated lot data"
//	@Success		200		{object}	models.AuctionLot
//	@Failure		400		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/{id} [put]
func (h *AuctionLotHandler) Update(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	existing, err := h.repo.GetByID(uint(id), userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Auction lot not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get auction lot"})
		return
	}

	var req UpdateLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	fields := map[string]interface{}{}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title cannot be empty"})
			return
		}
		fields["title"] = title
	}
	if req.NumisBidsURL != nil {
		url := strings.TrimSpace(*req.NumisBidsURL)
		if url == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "URL cannot be empty"})
			return
		}
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "URL must start with http:// or https://"})
			return
		}
		fields["numis_bids_url"] = url
		fields["source_url"] = url
		if existing.Source == "" {
			fields["source"] = auctionSourceFromRequest("", url)
		}
	}
	if req.AuctionHouse != nil {
		fields["auction_house"] = strings.TrimSpace(*req.AuctionHouse)
	}
	if req.SaleName != nil {
		fields["sale_name"] = strings.TrimSpace(*req.SaleName)
	}
	if req.LotNumber != nil {
		fields["lot_number"] = *req.LotNumber
	}
	if req.SaleDate != nil {
		fields["sale_date"] = *req.SaleDate
	}
	if req.AuctionEndTime != nil {
		fields["auction_end_time"] = *req.AuctionEndTime
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Notes != nil {
		fields["notes"] = *req.Notes
	}
	if req.Category != nil {
		fields["category"] = models.Category(*req.Category)
	}
	if req.Estimate != nil {
		fields["estimate"] = *req.Estimate
	}
	if req.InitialBid != nil {
		fields["initial_bid"] = *req.InitialBid
	}
	if req.CurrentBid != nil {
		fields["current_bid"] = *req.CurrentBid
	}
	if req.MaxBid != nil {
		fields["max_bid"] = *req.MaxBid
	}
	if req.WinningBid != nil {
		fields["winning_bid"] = *req.WinningBid
	}
	if req.Currency != nil {
		fields["currency"] = strings.TrimSpace(*req.Currency)
	}

	if len(fields) == 0 {
		c.JSON(http.StatusOK, existing)
		return
	}

	if err := h.repo.UpdateFields(existing, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update auction lot"})
		return
	}

	updated, _ := h.repo.GetByID(uint(id), userID)
	c.JSON(http.StatusOK, updated)
}

// UpdateStatusRequest holds the new status for an auction lot.
type UpdateStatusRequest struct {
	Status     string   `json:"status" binding:"required"`
	MaxBid     *float64 `json:"maxBid,omitempty"`
	WinningBid *float64 `json:"winningBid,omitempty"`
}

// UpdateStatus transitions an auction lot to a new status.
//
//	@Summary		Update auction lot status
//	@Description	Transitions an auction lot to a new status.
//	@Tags			Auctions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Auction lot ID"
//	@Param			body	body		UpdateStatusRequest		true	"New status"
//	@Success		200		{object}	models.AuctionLot
//	@Failure		400		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/{id}/status [put]
func (h *AuctionLotHandler) UpdateStatus(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
		return
	}

	newStatus := models.AuctionLotStatus(req.Status)
	lot, err := h.repo.GetByID(uint(id), userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Auction lot not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	if lot.Status != newStatus {
		if err := h.svc.UpdateStatus(uint(id), userID, newStatus); err != nil {
			if errors.Is(err, services.ErrAuctionLotNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Auction lot not found"})
				return
			}
			if errors.Is(err, services.ErrInvalidStatus) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status transition"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
			return
		}
	}

	if req.MaxBid != nil {
		lot, err = h.repo.GetByID(uint(id), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
			return
		}
		if err := h.repo.UpdateFields(lot, map[string]interface{}{"max_bid": *req.MaxBid}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update max bid"})
			return
		}
	}

	if req.WinningBid != nil && newStatus == models.AuctionStatusWon {
		lot, err = h.repo.GetByID(uint(id), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
			return
		}
		if err := h.repo.UpdateFields(lot, map[string]interface{}{"winning_bid": *req.WinningBid}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update winning bid"})
			return
		}
	}

	lot, _ = h.repo.GetByID(uint(id), userID)
	c.JSON(http.StatusOK, lot)
}

// LinkEventRequest holds the event ID to associate with a lot.
type LinkEventRequest struct {
	EventID *uint `json:"eventId"`
}

// LinkEvent associates or disassociates an auction lot with a calendar event.
//
//	@Summary		Link lot to calendar event
//	@Description	Sets or clears the calendar event association for an auction lot.
//	@Tags			Auctions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Auction lot ID"
//	@Param			body	body		LinkEventRequest	true	"Event ID (null to unlink)"
//	@Success		200		{object}	models.AuctionLot
//	@Failure		400		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/{id}/event [put]
func (h *AuctionLotHandler) LinkEvent(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	lot, err := h.repo.GetByID(uint(id), userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Auction lot not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get auction lot"})
		return
	}

	var req LinkEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	h.repo.UpdateFields(lot, map[string]interface{}{"event_id": req.EventID})
	updated, _ := h.repo.GetByID(uint(id), userID)
	c.JSON(http.StatusOK, updated)
}

// BulkLinkEventRequest holds lot IDs and the target event ID.
type BulkLinkEventRequest struct {
	LotIDs  []uint `json:"lotIds" binding:"required"`
	EventID *uint  `json:"eventId"`
}

// BulkLinkEvent associates or disassociates multiple auction lots with a calendar event.
//
//	@Summary		Bulk link lots to calendar event
//	@Description	Sets or clears the calendar event for multiple auction lots at once.
//	@Tags			Auctions
//	@Accept			json
//	@Produce		json
//	@Param			body	body		BulkLinkEventRequest	true	"Lot IDs and event ID"
//	@Success		200		{object}	map[string]int
//	@Failure		400		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/bulk-link-event [put]
func (h *AuctionLotHandler) BulkLinkEvent(c *gin.Context) {
	userID := c.GetUint("userId")

	var req BulkLinkEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updated := 0
	for _, lotID := range req.LotIDs {
		lot, err := h.repo.GetByID(lotID, userID)
		if err != nil {
			continue
		}
		h.repo.UpdateFields(lot, map[string]interface{}{"event_id": req.EventID})
		updated++
	}

	c.JSON(http.StatusOK, gin.H{"updated": updated})
}

// ConvertToCoin creates an owned Coin from a won auction lot.
//
//	@Summary		Convert won lot to coin
//	@Description	Creates an owned Coin from a won auction lot.
//	@Tags			Auctions
//	@Produce		json
//	@Param			id	path		int	true	"Auction lot ID"
//	@Success		201	{object}	models.Coin
//	@Failure		400	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/{id}/convert [post]
func (h *AuctionLotHandler) ConvertToCoin(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	coin, err := h.svc.ConvertToCoin(uint(id), userID)
	if err != nil {
		if errors.Is(err, services.ErrAuctionLotNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Auction lot not found"})
			return
		}
		if errors.Is(err, services.ErrInvalidStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only won lots can be converted to coins"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert lot to coin"})
		return
	}

	c.JSON(http.StatusCreated, coin)
}

// Delete removes an auction lot.
//
//	@Summary		Delete auction lot
//	@Description	Removes an auction lot from tracking.
//	@Tags			Auctions
//	@Param			id	path	int	true	"Auction lot ID"
//	@Success		204
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/{id} [delete]
func (h *AuctionLotHandler) Delete(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	rows, err := h.repo.Delete(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete auction lot"})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Auction lot not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ImportFromURL imports a lot from a NumisBids URL via the agent scraper.
//
//	@Summary		Import lot from NumisBids URL
//	@Description	Scrapes a NumisBids lot page and creates an auction lot entry.
//	@Tags			Auctions
//	@Accept			json
//	@Produce		json
//	@Param			body	body		ImportLotRequest	true	"NumisBids lot URL"
//	@Success		201		{object}	models.AuctionLot
//	@Failure		400		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/import [post]
func (h *AuctionLotHandler) ImportFromURL(c *gin.Context) {
	userID := c.GetUint("userId")

	var req ImportLotRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	source := auctionSourceFromRequest(req.Source, req.URL)
	if source == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported auction URL"})
		return
	}

	if source == models.AuctionSourceCNG {
		wl, err := h.cngSvc.ScrapeLot(req.URL)
		if err != nil {
			h.warn("CNG lot import scrape failed for user %d url=%s: %v", userID, req.URL, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to scrape CNG lot"})
			return
		}
		lot := auctionLotFromWatchlist(source, userID, wl, models.AuctionStatusWatching, services.ParseCNGDate(wl.SaleDate))
		if req.Category != "" {
			lot.Category = models.Category(req.Category)
		}
		if _, err := h.repo.Upsert(&lot); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import auction lot"})
			return
		}
		imported, err := h.repo.GetBySourceURL(source, lot.SourceURL, userID)
		if err != nil {
			c.JSON(http.StatusCreated, lot)
			return
		}
		c.JSON(http.StatusCreated, imported)
		return
	}

	// Store the URL and basic info — the frontend will call the agent
	// to scrape details separately, or we can enhance this later
	// to call the agent service directly.
	lot := models.AuctionLot{
		NumisBidsURL: req.URL,
		Source:       source,
		SourceURL:    req.URL,
		Title:        req.Title,
		Description:  req.Description,
		AuctionHouse: req.AuctionHouse,
		SaleName:     req.SaleName,
		Category:     models.Category(req.Category),
		ImageURL:     req.ImageURL,
		Estimate:     req.Estimate,
		CurrentBid:   req.CurrentBid,
		Currency:     req.Currency,
		Status:       models.AuctionStatusWatching,
		UserID:       userID,
	}

	if lot.Currency == "" {
		lot.Currency = "USD"
	}

	// Upsert: if already tracking this lot, update it
	if _, err := h.repo.Upsert(&lot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import auction lot"})
		return
	}

	// Return the current state
	imported, err := h.repo.GetByURL(req.URL, userID)
	if err != nil {
		c.JSON(http.StatusCreated, lot)
		return
	}
	c.JSON(http.StatusCreated, imported)
}

// SyncWatchlist syncs auction lots from the user's configured auction provider watchlist.
//
//	@Summary		Sync auction watchlist
//	@Description	Logs into NumisBids or CNG with the user's stored credentials, fetches the selected watchlist, and upserts each lot. Defaults to NumisBids when source is omitted.
//	@Tags			Auctions
//	@Produce		json
//	@Param			source	query	string	false	"Auction source (numisbids, cng)"
//	@Param			body	body	SyncWatchlistRequest	false	"Auction source request"
//	@Success		200	{object}	SyncWatchlistResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/sync [post]
func (h *AuctionLotHandler) SyncWatchlist(c *gin.Context) {
	userID := c.GetUint("userId")
	source := models.AuctionSource(c.DefaultQuery("source", string(models.AuctionSourceNumisBids)))
	if c.Request.ContentLength > 0 {
		var req SyncWatchlistRequest
		if err := c.ShouldBindJSON(&req); err == nil && req.Source != "" {
			source = models.AuctionSource(req.Source)
		}
	}
	if source != models.AuctionSourceNumisBids && source != models.AuctionSourceCNG {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported auction source"})
		return
	}
	h.debug("%s sync started for user %d", source, userID)

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		h.warn("%s sync failed to load user %d: %v", source, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user"})
		return
	}

	if source == models.AuctionSourceCNG {
		h.syncCNGWatchlist(c, userID, user)
		return
	}

	if user.NumisBidsUsername == "" || user.NumisBidsPassword == "" {
		h.warn("NumisBids sync blocked for user %d: credentials not configured", userID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "NumisBids credentials not configured. Go to Settings to add them."})
		return
	}

	password, shouldMigrate, err := h.decryptStoredCredential(user.ID, "numis_bids_password", user.NumisBidsPassword)
	if err != nil {
		h.warn("NumisBids credential decrypt failed for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read stored NumisBids credentials"})
		return
	}

	if shouldMigrate {
		h.migrateStoredCredential(user, "numis_bids_password", password)
	}
	h.debug("NumisBids sync logging in for user %d", userID)
	client, err := h.nbSvc.Login(user.NumisBidsUsername, password)
	if err != nil {
		h.warn("NumisBids login failed for user %d: %v", userID, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "NumisBids login failed. Check your credentials in Settings."})
		return
	}
	h.debug("NumisBids sync login succeeded for user %d", userID)

	rawHTML, err := h.nbSvc.FetchWatchlist(client)
	if err != nil {
		if errors.Is(err, services.ErrNumisBidsAuthenticationRequired) {
			h.warn("NumisBids watchlist returned login page for user %d after login", userID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "NumisBids login succeeded but watchlist access was not authenticated. Check your credentials in Settings."})
			return
		}
		h.warn("NumisBids watchlist fetch failed for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watchlist from NumisBids"})
		return
	}
	diagnostics := h.nbSvc.WatchlistDiagnostics(rawHTML)
	h.debug("NumisBids watchlist fetched for user %d: bytes=%d candidateLinks=%d hasWatchlistText=%t hasLoginPrompt=%t",
		userID, diagnostics.HTMLBytes, diagnostics.CandidateLinkCount, diagnostics.HasWatchlistText, diagnostics.HasLoginPrompt)

	parsed := h.nbSvc.ParseWatchlist(rawHTML)
	h.debug("NumisBids watchlist parsed for user %d: lots=%d", userID, len(parsed))
	if len(parsed) == 0 {
		h.warn("NumisBids sync found zero parseable lots for user %d: bytes=%d candidateLinks=%d hasWatchlistText=%t",
			userID, diagnostics.HTMLBytes, diagnostics.CandidateLinkCount, diagnostics.HasWatchlistText)
	}

	var synced []models.AuctionLot
	now := time.Now()

	for _, wl := range parsed {
		h.debug("NumisBids sync processing lot for user %d: saleID=%s lot=%d url=%s", userID, wl.SaleID, wl.LotNumber, wl.URL)
		// Scrape the lot page for image, auction house, sale name, current bid, lot number, description, sale date
		if details, err := h.nbSvc.ScrapeLotPage(wl.URL); err == nil {
			if details.ImageURL != "" {
				wl.ImageURL = details.ImageURL
			}
			wl.AuctionHouse = details.AuctionHouse
			wl.SaleName = details.SaleName
			wl.SaleDate = details.SaleDate
			wl.Description = details.Description
			wl.CurrentBid = details.CurrentBid
			if details.Currency != "" {
				wl.Currency = details.Currency
			}
			if details.LotNumber > 0 {
				wl.LotNumber = details.LotNumber
			}
		} else {
			h.warn("Could not scrape NumisBids lot page for user %d url=%s: %v", userID, wl.URL, err)
		}

		// Determine status: mark as passed if sale date is in the past
		status := models.AuctionStatusWatching
		var saleDate *time.Time
		if wl.SaleDate != "" {
			saleDate = services.ParseSaleDate(wl.SaleDate)
			if saleDate != nil && saleDate.Before(now) {
				status = models.AuctionStatusPassed
			}
		}

		lot := models.AuctionLot{
			NumisBidsURL: wl.URL,
			Source:       models.AuctionSourceNumisBids,
			SourceURL:    wl.URL,
			SourceSaleID: wl.SourceSaleID,
			SaleID:       wl.SaleID,
			LotNumber:    wl.LotNumber,
			Title:        wl.Title,
			Description:  wl.Description,
			ImageURL:     wl.ImageURL,
			Estimate:     wl.Estimate,
			CurrentBid:   wl.CurrentBid,
			Currency:     wl.Currency,
			AuctionHouse: wl.AuctionHouse,
			SaleName:     wl.SaleName,
			SaleDate:     saleDate,
			Status:       status,
			UserID:       userID,
		}
		if lot.Currency == "" {
			lot.Currency = "USD"
		}

		if _, err := h.repo.UpsertWithCalendarEvent(&lot); err != nil {
			h.warn("Failed to upsert NumisBids lot for user %d url=%s: %v", userID, wl.URL, err)
			continue
		}

		if upserted, err := h.repo.GetByURL(wl.URL, userID); err == nil {
			synced = append(synced, *upserted)
		} else {
			h.warn("NumisBids lot upserted but reload failed for user %d url=%s: %v", userID, wl.URL, err)
		}
	}

	// Also mark any existing watching lots whose sale date has passed
	h.repo.MarkPastAuctionsAsPassed(userID, now)
	h.info("NumisBids sync completed for user %d: parsed=%d synced=%d", userID, len(parsed), len(synced))

	c.JSON(http.StatusOK, gin.H{"synced": len(synced), "lots": synced})
}

func (h *AuctionLotHandler) syncCNGWatchlist(c *gin.Context, userID uint, user *models.User) {
	if user.CNGUsername == "" || user.CNGPassword == "" {
		h.warn("CNG sync blocked for user %d: credentials not configured", userID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "CNG credentials not configured. Go to Settings to add them."})
		return
	}

	password, shouldMigrate, err := h.decryptStoredCredential(user.ID, "cng_password", user.CNGPassword)
	if err != nil {
		h.warn("CNG credential decrypt failed for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read stored CNG credentials"})
		return
	}

	if shouldMigrate {
		h.migrateStoredCredential(user, "cng_password", password)
	}
	client, err := h.cngSvc.Login(user.CNGUsername, password)
	if err != nil {
		h.warn("CNG login failed for user %d: %v", userID, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "CNG login failed. Check your credentials in Settings."})
		return
	}

	parsed, err := h.cngSvc.FetchWatchlistLots(client)
	if err != nil {
		if errors.Is(err, services.ErrCNGAuthenticationRequired) {
			h.warn("CNG watchlist returned login page for user %d after login", userID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "CNG login succeeded but watched lots were not authenticated. Check your credentials in Settings."})
			return
		}
		h.warn("CNG watchlist fetch failed for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watched lots from CNG"})
		return
	}
	h.debug("CNG watchlist parsed for user %d: lots=%d", userID, len(parsed))

	var synced []models.AuctionLot
	now := time.Now()
	for _, wl := range parsed {
		// Scrape the individual lot page with the authenticated client to obtain bid_amount
		// (the current winning bid). The watched-lots list page does not include bid_amount,
		// only starting_price, so without this step currentBid always shows the opening price.
		if details, err := h.cngSvc.ScrapeLotWithClient(client, wl.URL); err == nil {
			if details.CurrentBid != nil {
				wl.CurrentBid = details.CurrentBid
			}
		} else {
			h.warn("Could not refresh CNG lot page for user %d url=%s: %v", userID, wl.URL, err)
		}

		// Presence of an autobid means the user has placed a bid on this lot.
		status := models.AuctionStatusWatching
		if wl.MaxBid != nil {
			status = models.AuctionStatusBidding
		}
		auctionEndTime := services.ParseCNGDate(wl.SaleDate)
		if auctionEndTime != nil && auctionEndTime.Before(now) {
			status = models.AuctionStatusPassed
		}

		lot := auctionLotFromWatchlist(models.AuctionSourceCNG, userID, wl, status, auctionEndTime)
		if _, err := h.repo.UpsertWithCalendarEvent(&lot); err != nil {
			h.warn("Failed to upsert CNG lot for user %d url=%s: %v", userID, wl.URL, err)
			continue
		}

		if upserted, err := h.repo.GetBySourceURL(models.AuctionSourceCNG, lot.SourceURL, userID); err == nil {
			synced = append(synced, *upserted)
		} else {
			h.warn("CNG lot upserted but reload failed for user %d url=%s: %v", userID, wl.URL, err)
		}
	}

	h.repo.MarkPastAuctionsAsPassed(userID, now)
	h.info("CNG sync completed for user %d: parsed=%d synced=%d", userID, len(parsed), len(synced))
	c.JSON(http.StatusOK, gin.H{"synced": len(synced), "lots": synced})
}

func auctionLotFromWatchlist(source models.AuctionSource, userID uint, wl services.WatchlistLot, status models.AuctionLotStatus, auctionEndTime *time.Time) models.AuctionLot {
	sourceURL := strings.TrimSpace(wl.URL)
	return models.AuctionLot{
		NumisBidsURL:   sourceURL,
		Source:         source,
		SourceURL:      sourceURL,
		SourceLotID:    wl.SourceLotID,
		SourceSaleID:   firstNonEmpty(wl.SourceSaleID, wl.SaleID),
		SaleID:         wl.SaleID,
		LotNumber:      wl.LotNumber,
		Title:          wl.Title,
		Description:    wl.Description,
		ImageURL:       wl.ImageURL,
		Estimate:       wl.Estimate,
		CurrentBid:     wl.CurrentBid,
		MaxBid:         wl.MaxBid,
		Currency:       firstNonEmpty(wl.Currency, "USD"),
		AuctionHouse:   wl.AuctionHouse,
		SaleName:       wl.SaleName,
		AuctionEndTime: auctionEndTime,
		Status:         status,
		UserID:         userID,
	}
}

func auctionSourceFromRequest(rawSource, rawURL string) models.AuctionSource {
	source := models.AuctionSource(strings.TrimSpace(strings.ToLower(rawSource)))
	if source == "" {
		normalizedURL := strings.ToLower(rawURL)
		switch {
		case strings.Contains(normalizedURL, "auctions.cngcoins.com"):
			source = models.AuctionSourceCNG
		case strings.Contains(normalizedURL, "numisbids.com"):
			source = models.AuctionSourceNumisBids
		default:
			return ""
		}
	}
	if source != models.AuctionSourceNumisBids && source != models.AuctionSourceCNG {
		return ""
	}
	return source
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (h *AuctionLotHandler) decryptStoredCredential(userID uint, field string, stored string) (plain string, shouldMigrate bool, err error) {
	plain, wasEncrypted, err := h.credentials.DecryptStringWithAAD(stored, services.AuctionCredentialAAD(userID, field))
	if err != nil {
		return "", false, err
	}
	return plain, h.credentials.Enabled() && !wasEncrypted && stored != "", nil
}

func (h *AuctionLotHandler) migrateStoredCredential(user *models.User, field string, plain string) {
	encrypted, err := h.credentials.EncryptStringWithAAD(plain, services.AuctionCredentialAAD(user.ID, field))
	if err != nil {
		h.warn("Failed to encrypt legacy auction credential for user %d: %v", user.ID, err)
		return
	}
	if encrypted == plain {
		return
	}
	if err := h.userRepo.UpdateField(user, field, encrypted); err != nil {
		h.warn("Failed to save encrypted legacy auction credential for user %d: %v", user.ID, err)
	}
}

func (h *AuctionLotHandler) debug(format string, args ...interface{}) {
	if h.logger != nil {
		h.logger.Debug("auctions", format, args...)
	}
}

func (h *AuctionLotHandler) info(format string, args ...interface{}) {
	if h.logger != nil {
		h.logger.Info("auctions", format, args...)
	}
}

func (h *AuctionLotHandler) warn(format string, args ...interface{}) {
	if h.logger != nil {
		h.logger.Warn("auctions", format, args...)
	}
}

// SyncWatchlistResponse is the response for the sync watchlist endpoint.
type SyncWatchlistResponse struct {
	Synced int                 `json:"synced"`
	Lots   []models.AuctionLot `json:"lots"`
}

type SyncWatchlistRequest struct {
	Source string `json:"source"`
}

// ValidateNumisBids tests the given credentials against an auction provider.
//
//	@Summary		Validate auction provider credentials
//	@Description	Attempts to log in to NumisBids or CNG with the provided credentials. Defaults to NumisBids when source is omitted.
//	@Tags			Auctions
//	@Accept			json
//	@Produce		json
//	@Param			body	body		ValidateNumisBidsRequest	true	"Auction provider credentials"
//	@Success		200		{object}	map[string]bool
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/validate-credentials [post]
func (h *AuctionLotHandler) ValidateNumisBids(c *gin.Context) {
	var req ValidateNumisBidsRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	source := models.AuctionSource(req.Source)
	if source == "" {
		source = models.AuctionSourceNumisBids
	}
	var err error
	switch source {
	case models.AuctionSourceNumisBids:
		_, err = h.nbSvc.Login(req.Username, req.Password)
	case models.AuctionSourceCNG:
		_, err = h.cngSvc.Login(req.Username, req.Password)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported auction source"})
		return
	}
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false, "error": "Login failed. Check your credentials."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// ValidateNumisBidsRequest holds credentials for validation.
type ValidateNumisBidsRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Source   string `json:"source"`
}

// ImportLotRequest holds the data for importing a lot from NumisBids.
type ImportLotRequest struct {
	URL          string   `json:"url" binding:"required"`
	Source       string   `json:"source"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	AuctionHouse string   `json:"auctionHouse"`
	SaleName     string   `json:"saleName"`
	Category     string   `json:"category"`
	ImageURL     string   `json:"imageUrl"`
	Estimate     *float64 `json:"estimate"`
	CurrentBid   *float64 `json:"currentBid"`
	Currency     string   `json:"currency"`
}

// AuctionLotListResponse is the response structure for list endpoints.
type AuctionLotListResponse struct {
	Lots  []models.AuctionLot `json:"lots"`
	Total int64               `json:"total"`
}
