package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

type CalendarHandler struct {
	eventRepo   *repository.AuctionEventRepository
	auctionRepo *repository.AuctionLotRepository
}

func NewCalendarHandler(eventRepo *repository.AuctionEventRepository, auctionRepo *repository.AuctionLotRepository) *CalendarHandler {
	return &CalendarHandler{eventRepo: eventRepo, auctionRepo: auctionRepo}
}

// GetCalendar returns auction lots and events in a date range.
func (h *CalendarHandler) GetCalendar(c *gin.Context) {
	userID := c.GetUint("userId")

	startStr := c.Query("start")
	endStr := c.Query("end")

	var start, end time.Time
	var err error

	if startStr != "" {
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date (use YYYY-MM-DD)"})
			return
		}
	} else {
		// Default to start of current month
		now := time.Now()
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date (use YYYY-MM-DD)"})
			return
		}
	} else {
		// Default to 3 months from start
		end = start.AddDate(0, 3, 0)
	}

	lots, events, err := h.eventRepo.GetCalendar(userID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load calendar"})
		return
	}

	lotItems := make([]gin.H, 0, len(lots))
	for _, lot := range lots {
		item := gin.H{
			"id":             lot.ID,
			"type":           "lot",
			"title":          lot.Title,
			"auctionHouse":   lot.AuctionHouse,
			"status":         lot.Status,
			"currentBid":     lot.CurrentBid,
			"estimate":       lot.Estimate,
			"numisBidsUrl":   lot.NumisBidsURL,
			"imageUrl":       lot.ImageURL,
			"saleDate":       lot.SaleDate,
			"auctionEndTime": lot.AuctionEndTime,
		}
		lotItems = append(lotItems, item)
	}

	eventItems := make([]gin.H, 0, len(events))
	for _, ev := range events {
		eventItems = append(eventItems, gin.H{
			"id":           ev.ID,
			"type":         "event",
			"title":        ev.Title,
			"auctionHouse": ev.AuctionHouse,
			"startDate":    ev.StartDate,
			"endDate":      ev.EndDate,
			"url":          ev.URL,
			"notes":        ev.Notes,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"lots":   lotItems,
		"events": eventItems,
		"range": gin.H{
			"start": start.Format("2006-01-02"),
			"end":   end.Format("2006-01-02"),
		},
	})
}

// ListEvents returns all calendar events for the authenticated user.
func (h *CalendarHandler) ListEvents(c *gin.Context) {
	userID := c.GetUint("userId")
	events, err := h.eventRepo.ListByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list events"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

// CreateEvent creates a new auction event.
func (h *CalendarHandler) CreateEvent(c *gin.Context) {
	userID := c.GetUint("userId")

	var req struct {
		Title        string     `json:"title" binding:"required"`
		AuctionHouse string     `json:"auctionHouse"`
		StartDate    *time.Time `json:"startDate"`
		EndDate      *time.Time `json:"endDate"`
		URL          string     `json:"url"`
		Notes        string     `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
		return
	}

	event := models.AuctionEvent{
		UserID:       userID,
		Title:        req.Title,
		AuctionHouse: req.AuctionHouse,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		URL:          req.URL,
		Notes:        req.Notes,
	}

	if err := h.eventRepo.Create(&event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": event.ID})
}

// UpdateEvent updates an auction event.
func (h *CalendarHandler) UpdateEvent(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	event, err := h.eventRepo.GetByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	var req struct {
		Title        *string    `json:"title"`
		AuctionHouse *string    `json:"auctionHouse"`
		StartDate    *time.Time `json:"startDate"`
		EndDate      *time.Time `json:"endDate"`
		URL          *string    `json:"url"`
		Notes        *string    `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Title != nil {
		event.Title = *req.Title
	}
	if req.AuctionHouse != nil {
		event.AuctionHouse = *req.AuctionHouse
	}
	if req.StartDate != nil {
		event.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		event.EndDate = req.EndDate
	}
	if req.URL != nil {
		event.URL = *req.URL
	}
	if req.Notes != nil {
		event.Notes = *req.Notes
	}

	if err := h.eventRepo.Update(event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event updated"})
}

// GetEvent returns a single event with its linked auction lots.
func (h *CalendarHandler) GetEvent(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	event, err := h.eventRepo.GetByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	lots, _ := h.auctionRepo.ListByEventID(uint(id), userID)

	c.JSON(http.StatusOK, gin.H{
		"event": event,
		"lots":  lots,
	})
}

// DeleteEvent deletes an auction event.
func (h *CalendarHandler) DeleteEvent(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	if err := h.eventRepo.Delete(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event deleted"})
}
