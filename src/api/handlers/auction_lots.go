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
	"gorm.io/gorm"
)

// AuctionLotHandler handles HTTP requests for auction lot operations.
type AuctionLotHandler struct {
	repo     *repository.AuctionLotRepository
	svc      *services.AuctionLotService
	userRepo *repository.UserRepository
	nbSvc    *services.NumisBidsService
	logger   *services.Logger
}

// NewAuctionLotHandler creates a new AuctionLotHandler.
func NewAuctionLotHandler(repo *repository.AuctionLotRepository, svc *services.AuctionLotService, userRepo *repository.UserRepository, nbSvc *services.NumisBidsService, logger *services.Logger) *AuctionLotHandler {
	return &AuctionLotHandler{repo: repo, svc: svc, userRepo: userRepo, nbSvc: nbSvc, logger: logger}
}

// List returns a paginated list of auction lots for the authenticated user.
//
//	@Summary		List auction lots
//	@Description	Returns a paginated list of auction lots belonging to the authenticated user.
//	@Tags			Auctions
//	@Produce		json
//	@Param			status	query		string	false	"Filter by status (watching, bidding, won, lost, passed)"
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
//	@Description	Returns a map of status → count for the authenticated user.
//	@Tags			Auctions
//	@Produce		json
//	@Success		200	{object}	map[string]int64
//	@Security		BearerAuth
//	@Router			/auctions/counts [get]
func (h *AuctionLotHandler) Counts(c *gin.Context) {
	userID := c.GetUint("userId")
	counts, err := h.repo.CountByStatus(userID)
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
	Currency       *string    `json:"currency"`
}

func (h *AuctionLotHandler) Update(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	existing, err := h.repo.GetByID(uint(id), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
	Status string   `json:"status" binding:"required"`
	MaxBid *float64 `json:"maxBid,omitempty"`
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

	// Save max bid if provided (typically when transitioning to "bidding")
	if req.MaxBid != nil {
		lot, _ := h.repo.GetByID(uint(id), userID)
		if lot != nil {
			h.repo.UpdateFields(lot, map[string]interface{}{"max_bid": req.MaxBid})
		}
	}

	lot, _ := h.repo.GetByID(uint(id), userID)
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

	// Store the URL and basic info — the frontend will call the agent
	// to scrape details separately, or we can enhance this later
	// to call the agent service directly.
	lot := models.AuctionLot{
		NumisBidsURL: req.URL,
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
	if err := h.repo.Upsert(&lot); err != nil {
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

// SyncWatchlist syncs auction lots from the user's NumisBids watchlist.
//
//	@Summary		Sync NumisBids watchlist
//	@Description	Logs into NumisBids with the user's stored credentials, fetches their watchlist, and upserts each lot.
//	@Tags			Auctions
//	@Produce		json
//	@Success		200	{object}	SyncWatchlistResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/auctions/sync [post]
func (h *AuctionLotHandler) SyncWatchlist(c *gin.Context) {
	userID := c.GetUint("userId")
	h.debug("NumisBids sync started for user %d", userID)

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		h.warn("NumisBids sync failed to load user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user"})
		return
	}

	if user.NumisBidsUsername == "" || user.NumisBidsPassword == "" {
		h.warn("NumisBids sync blocked for user %d: credentials not configured", userID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "NumisBids credentials not configured. Go to Settings to add them."})
		return
	}

	h.debug("NumisBids sync logging in for user %d", userID)
	client, err := h.nbSvc.Login(user.NumisBidsUsername, user.NumisBidsPassword)
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

		if err := h.repo.Upsert(&lot); err != nil {
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

// ValidateNumisBids tests the given credentials against NumisBids.
//
//	@Summary		Validate NumisBids credentials
//	@Description	Attempts to log in to NumisBids with the provided credentials to verify they are correct.
//	@Tags			Auctions
//	@Accept			json
//	@Produce		json
//	@Param			body	body		ValidateNumisBidsRequest	true	"NumisBids credentials"
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

	_, err := h.nbSvc.Login(req.Username, req.Password)
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
}

// ImportLotRequest holds the data for importing a lot from NumisBids.
type ImportLotRequest struct {
	URL          string   `json:"url" binding:"required"`
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
