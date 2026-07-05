package repository

import (
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AuctionLotRepository encapsulates all auction-lot-related database operations.
type AuctionLotRepository struct {
	db *gorm.DB
}

// NewAuctionLotRepository creates a new AuctionLotRepository.
func NewAuctionLotRepository(db *gorm.DB) *AuctionLotRepository {
	return &AuctionLotRepository{db: db}
}

// WithTx returns a copy of the repository using the given transaction.
func (r *AuctionLotRepository) WithTx(tx *gorm.DB) *AuctionLotRepository {
	return &AuctionLotRepository{db: tx}
}

// Transaction executes fn inside a database transaction.
func (r *AuctionLotRepository) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

// AuctionLotListFilters holds filtering/sorting options for listing auction lots.
type AuctionLotListFilters struct {
	Status    string
	Search    string
	Source    string
	SortField string
	SortOrder string
	Page      int
	Limit     int
}

// AuctionLotUpsertResult describes whether an upsert inserted a new lot and calendar event.
type AuctionLotUpsertResult struct {
	Created      bool
	EventCreated bool
	EventID      *uint
}

// List returns a paginated list of auction lots for the given user.
func (r *AuctionLotRepository) List(userID uint, filters AuctionLotListFilters) ([]models.AuctionLot, int64, error) {
	var lots []models.AuctionLot
	var total int64

	query := r.db.Model(&models.AuctionLot{}).Scopes(OwnedBy(userID))

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.Source != "" {
		query = query.Where("source = ?", filters.Source)
	}
	if filters.Search != "" {
		like := "%" + filters.Search + "%"
		query = query.Where("title LIKE ? OR description LIKE ? OR auction_house LIKE ?", like, like, like)
	}

	query.Count(&total)

	sortField := "updated_at"
	if filters.SortField != "" {
		allowed := map[string]bool{
			"created_at": true, "updated_at": true, "sale_date": true,
			"estimate": true, "current_bid": true, "lot_number": true,
		}
		if allowed[filters.SortField] {
			sortField = filters.SortField
		}
	}
	sortOrder := "desc"
	if filters.SortOrder == "asc" {
		sortOrder = "asc"
	}

	limit := 50
	if filters.Limit > 0 && filters.Limit <= 100 {
		limit = filters.Limit
	}
	page := 1
	if filters.Page > 0 {
		page = filters.Page
	}
	offset := (page - 1) * limit

	err := query.
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: sortField},
			Desc:   sortOrder != "asc",
		}).
		Limit(limit).
		Offset(offset).
		Find(&lots).Error
	return lots, total, err
}

// GetByID returns a single auction lot owned by the given user.
func (r *AuctionLotRepository) GetByID(id, userID uint) (*models.AuctionLot, error) {
	var lot models.AuctionLot
	err := r.db.Scopes(OwnedByID(id, userID)).First(&lot).Error
	if err != nil {
		return nil, err
	}
	return &lot, nil
}

// GetByURL finds an auction lot by its NumisBids URL for the given user.
func (r *AuctionLotRepository) GetByURL(url string, userID uint) (*models.AuctionLot, error) {
	return r.GetBySourceURL(models.AuctionSourceNumisBids, url, userID)
}

// GetBySourceURL finds an auction lot by source URL for the given user.
func (r *AuctionLotRepository) GetBySourceURL(source models.AuctionSource, sourceURL string, userID uint) (*models.AuctionLot, error) {
	var lot models.AuctionLot
	source, sourceURL = normalizeAuctionSourceURL(source, sourceURL)
	err := r.db.Where("source = ? AND source_url = ? AND user_id = ?", source, sourceURL, userID).First(&lot).Error
	if err != nil {
		return nil, err
	}
	return &lot, nil
}

// Create inserts a new auction lot.
func (r *AuctionLotRepository) Create(lot *models.AuctionLot) error {
	normalizeAuctionLotSource(lot)
	return r.db.Create(lot).Error
}

// Update saves changes to an existing auction lot.
func (r *AuctionLotRepository) Update(lot *models.AuctionLot, updates *models.AuctionLot) error {
	return r.db.Model(lot).Updates(updates).Error
}

// UpdateFields updates specific fields on an auction lot.
func (r *AuctionLotRepository) UpdateFields(lot *models.AuctionLot, fields map[string]interface{}) error {
	return r.db.Model(lot).Updates(fields).Error
}

// Delete removes an auction lot.
func (r *AuctionLotRepository) Delete(id, userID uint) (int64, error) {
	result := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.AuctionLot{})
	return result.RowsAffected, result.Error
}

// StatusCount holds a status label and its count.
type StatusCount struct {
	Status string
	Count  int64
}

// CountByStatus returns per-status counts for the given user.
func (r *AuctionLotRepository) CountByStatus(userID uint) (map[string]int64, error) {
	return r.countByStatus(r.db.Where("user_id = ?", userID))
}

// CountByStatusForSource returns per-status counts for the given user's source-specific auction lots.
func (r *AuctionLotRepository) CountByStatusForSource(userID uint, source models.AuctionSource) (map[string]int64, error) {
	return r.countByStatus(r.db.Where("user_id = ? AND source = ?", userID, source))
}

// CountAll returns the total number of auction lots across all users.
func (r *AuctionLotRepository) CountAll() (int64, error) {
	var total int64
	err := r.db.Model(&models.AuctionLot{}).Count(&total).Error
	return total, err
}

// CountAllByStatus returns per-status counts for auction lots across all users.
func (r *AuctionLotRepository) CountAllByStatus() (map[string]int64, error) {
	return r.countByStatus(r.db)
}

func (r *AuctionLotRepository) countByStatus(db *gorm.DB) (map[string]int64, error) {
	var rows []StatusCount
	err := db.Model(&models.AuctionLot{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64)
	for _, r := range rows {
		counts[r.Status] = r.Count
	}
	return counts, nil
}

// Upsert creates or updates an auction lot by its source URL for the given user.
func (r *AuctionLotRepository) Upsert(lot *models.AuctionLot) (AuctionLotUpsertResult, error) {
	return r.upsert(lot, false)
}

// UpsertWithCalendarEvent creates or updates an auction lot and auto-links a calendar
// event only when the lot is newly tracked with a watchable status.
func (r *AuctionLotRepository) UpsertWithCalendarEvent(lot *models.AuctionLot) (AuctionLotUpsertResult, error) {
	return r.upsert(lot, true)
}

func (r *AuctionLotRepository) upsert(lot *models.AuctionLot, autoCreateEvent bool) (AuctionLotUpsertResult, error) {
	normalizeAuctionLotSource(lot)
	result := AuctionLotUpsertResult{}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &AuctionLotRepository{db: tx}
		existing, err := txRepo.GetBySourceURL(lot.Source, lot.SourceURL, lot.UserID)
		if err != nil {
			if !IsRecordNotFound(err) {
				return err
			}
			if err := tx.Create(lot).Error; err != nil {
				return err
			}
			result.Created = true
			if autoCreateEvent && shouldAutoCreateCalendarEvent(lot) {
				event, created, err := txRepo.findOrCreateCalendarEventForLot(lot)
				if err != nil {
					return err
				}
				if err := tx.Model(lot).Update("event_id", event.ID).Error; err != nil {
					return err
				}
				lot.EventID = &event.ID
				result.EventCreated = created
				result.EventID = &event.ID
			}
			return nil
		}
		// Update fields that may have changed
		updates := map[string]interface{}{
			"current_bid":      lot.CurrentBid,
			"estimate":         lot.Estimate,
			"title":            lot.Title,
			"description":      lot.Description,
			"image_url":        lot.ImageURL,
			"auction_house":    lot.AuctionHouse,
			"sale_name":        lot.SaleName,
			"sale_date":        lot.SaleDate,
			"currency":         lot.Currency,
			"lot_number":       lot.LotNumber,
			"auction_end_time": lot.AuctionEndTime,
			"source":           lot.Source,
			"source_url":       lot.SourceURL,
			"source_lot_id":    lot.SourceLotID,
			"source_sale_id":   lot.SourceSaleID,
			"numis_bids_url":   lot.NumisBidsURL,
		}
		// Only update status if the lot is being marked as passed (don't overwrite bidding/won/lost)
		if lot.Status == models.AuctionStatusPassed && existing.Status == models.AuctionStatusWatching {
			updates["status"] = string(models.AuctionStatusPassed)
		}
		return txRepo.UpdateFields(existing, updates)
	})
	return result, err
}

func shouldAutoCreateCalendarEvent(lot *models.AuctionLot) bool {
	return lot.Status == models.AuctionStatusWatching || lot.Status == models.AuctionStatusBidding
}

func (r *AuctionLotRepository) findOrCreateCalendarEventForLot(lot *models.AuctionLot) (*models.AuctionEvent, bool, error) {
	sourceSaleID := strings.TrimSpace(lot.SourceSaleID)
	if sourceSaleID != "" {
		var linkedLot models.AuctionLot
		err := r.db.
			Where("user_id = ? AND source = ? AND source_sale_id = ? AND event_id IS NOT NULL", lot.UserID, lot.Source, sourceSaleID).
			First(&linkedLot).Error
		if err == nil {
			var existing models.AuctionEvent
			if err := r.db.First(&existing, *linkedLot.EventID).Error; err != nil {
				return nil, false, err
			}
			return &existing, false, nil
		}
		if !IsRecordNotFound(err) {
			return nil, false, err
		}
	}

	event := auctionEventFromLot(lot)
	if err := r.db.Create(&event).Error; err != nil {
		return nil, false, err
	}
	return &event, true, nil
}

func auctionEventFromLot(lot *models.AuctionLot) models.AuctionEvent {
	eventDate := lot.AuctionEndTime
	if eventDate == nil {
		eventDate = lot.SaleDate
	}
	startDate := cloneTime(eventDate)
	endDate := cloneTime(eventDate)
	return models.AuctionEvent{
		UserID:       lot.UserID,
		Title:        auctionEventTitle(lot),
		AuctionHouse: lot.AuctionHouse,
		StartDate:    startDate,
		EndDate:      endDate,
		URL:          firstNonBlank(lot.SourceURL, lot.NumisBidsURL),
		Notes:        auctionEventNotes(lot),
	}
}

func auctionEventTitle(lot *models.AuctionLot) string {
	if saleName := strings.TrimSpace(lot.SaleName); saleName != "" {
		return saleName
	}

	title := strings.TrimSpace(lot.Title)
	if title == "" {
		title = "Auction lot"
	}
	if lot.LotNumber > 0 && !strings.Contains(strings.ToLower(title), "lot ") {
		return "Lot " + strconv.Itoa(lot.LotNumber) + " - " + title
	}
	return title
}

func auctionEventNotes(lot *models.AuctionLot) string {
	parts := []string{"Auto-created from " + string(lot.Source) + " watchlist sync."}
	if strings.TrimSpace(lot.SaleName) != "" {
		parts = append(parts, "Sale: "+strings.TrimSpace(lot.SaleName))
	}
	if lot.LotNumber > 0 {
		parts = append(parts, "Lot: "+strconv.Itoa(lot.LotNumber))
	}
	if strings.TrimSpace(lot.SourceSaleID) != "" {
		parts = append(parts, "Source sale ID: "+strings.TrimSpace(lot.SourceSaleID))
	}
	if strings.TrimSpace(lot.SourceLotID) != "" {
		parts = append(parts, "Source lot ID: "+strings.TrimSpace(lot.SourceLotID))
	}
	return strings.Join(parts, "\n")
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeAuctionLotSource(lot *models.AuctionLot) {
	source, sourceURL := normalizeAuctionSourceURL(lot.Source, lot.SourceURL)
	lot.Source = source
	lot.SourceURL = sourceURL
	if lot.NumisBidsURL == "" {
		lot.NumisBidsURL = sourceURL
	}
	if lot.SourceURL == "" {
		lot.SourceURL = lot.NumisBidsURL
	}
}

func normalizeAuctionSourceURL(source models.AuctionSource, sourceURL string) (models.AuctionSource, string) {
	if source == "" {
		source = models.AuctionSourceNumisBids
	}
	return source, sourceURL
}

// MarkPastAuctionsAsPassed updates all "watching" lots for a user where sale_date is before now.
func (r *AuctionLotRepository) MarkPastAuctionsAsPassed(userID uint, now time.Time) {
	r.db.Model(&models.AuctionLot{}).
		Where("user_id = ? AND status = ? AND sale_date IS NOT NULL AND sale_date < ?",
			userID, models.AuctionStatusWatching, now).
		Update("status", string(models.AuctionStatusPassed))
}

// ListByEventID returns all auction lots linked to a specific calendar event.
func (r *AuctionLotRepository) ListByEventID(eventID, userID uint) ([]models.AuctionLot, error) {
	var lots []models.AuctionLot
	err := r.db.Where("event_id = ? AND user_id = ?", eventID, userID).
		Order("lot_number ASC").Find(&lots).Error
	return lots, err
}

// GetEndingSoon returns all auction lots with BIDDING status that end within the next 24 hours.
// Checks both sale_date and auction_end_time fields to handle various auction sources.
// Uses a rolling 24-hour window from now to avoid timezone-related edge cases where lots
// ending at midnight UTC are excluded for users in negative-offset timezones.
func (r *AuctionLotRepository) GetEndingSoon() ([]models.AuctionLot, error) {
	var lots []models.AuctionLot
	now := time.Now()
	next24h := now.Add(24 * time.Hour)

	// Match if sale_date OR auction_end_time is in the next 24 hours
	// Use LOWER() for case-insensitive status comparison
	err := r.db.Where("LOWER(status) = ? AND ("+
		"(sale_date IS NOT NULL AND sale_date > ? AND sale_date <= ?) OR "+
		"(auction_end_time IS NOT NULL AND auction_end_time > ? AND auction_end_time <= ?)"+
		")",
		string(models.AuctionStatusBidding),
		now, next24h, // sale_date range: (now, now+24h]
		now, next24h). // auction_end_time range: (now, now+24h]
		Order("user_id ASC").
		Find(&lots).Error
	return lots, err
}

// GetActiveWatchBidDigestLots returns watched or bidding lots that have not reached their known auction end date.
func (r *AuctionLotRepository) GetActiveWatchBidDigestLots() ([]models.AuctionLot, error) {
	var lots []models.AuctionLot
	now := time.Now()
	statuses := []string{
		string(models.AuctionStatusWatching),
		string(models.AuctionStatusBidding),
	}

	err := r.db.Where("LOWER(status) IN ? AND ("+
		"(auction_end_time IS NOT NULL AND auction_end_time > ?) OR "+
		"(auction_end_time IS NULL AND sale_date IS NOT NULL AND sale_date > ?)"+
		")",
		statuses,
		now, now).
		Order("user_id ASC").
		Order("COALESCE(auction_end_time, sale_date) ASC").
		Order("lot_number ASC").
		Find(&lots).Error
	return lots, err
}

// AuctionLotDebugInfo holds enriched lot data for debugging date/status issues.
type AuctionLotDebugInfo struct {
	ID             uint       `json:"id"`
	LotNumber      int        `json:"lotNumber"`
	Status         string     `json:"status"`
	SaleDate       *time.Time `json:"saleDate"`
	AuctionEndTime *time.Time `json:"auctionEndTime"`
	EventID        *uint      `json:"eventId"`
	EventEndDate   *time.Time `json:"eventEndDate,omitempty"`
	AuctionHouse   string     `json:"auctionHouse"`
	SaleName       string     `json:"saleName"`
	UserID         uint       `json:"userId"`
}

// GetAllBiddingLotsWithEventDates returns all bidding lots with enriched date info (including event dates).
// Joins with AuctionEvent to show event end dates for lots linked to calendar events.
func (r *AuctionLotRepository) GetAllBiddingLotsWithEventDates() ([]AuctionLotDebugInfo, error) {
	var lots []AuctionLotDebugInfo
	query := `
		SELECT 
			al.id, 
			al.lot_number, 
			al.status, 
			al.sale_date, 
			al.auction_end_time, 
			al.event_id,
			ae.end_date as event_end_date,
			al.auction_house,
			al.sale_name,
			al.user_id
		FROM auction_lots al
		LEFT JOIN auction_events ae ON al.event_id = ae.id
		WHERE al.status = ?
		ORDER BY al.user_id ASC, al.created_at DESC
	`
	err := r.db.Raw(query, models.AuctionStatusBidding).Scan(&lots).Error
	return lots, err
}
