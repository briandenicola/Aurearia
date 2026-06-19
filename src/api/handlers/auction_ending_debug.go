package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

// AuctionEndingDebugHandler exposes debug/diagnostic endpoints for the auction ending scheduler.
type AuctionEndingDebugHandler struct {
	auctionRepo *repository.AuctionLotRepository
}

// NewAuctionEndingDebugHandler constructs a new debug handler.
func NewAuctionEndingDebugHandler(auctionRepo *repository.AuctionLotRepository) *AuctionEndingDebugHandler {
	return &AuctionEndingDebugHandler{
		auctionRepo: auctionRepo,
	}
}

// DebugGetAuctionEndingInfo returns comprehensive diagnostic data for the auction ending scheduler.
//
// @Summary Debug auction ending scheduler
// @Description Returns diagnostic info: current time, next 24h window, total lots, lots by status, lots matching the scheduler query, and all BIDDING lots with all their date fields populated
// @Tags admin
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/auction-ending/debug [get]
func (h *AuctionEndingDebugHandler) DebugGetAuctionEndingInfo(c *gin.Context) {
	now := time.Now()
	next24h := now.Add(24 * time.Hour)

	totalLots, err := h.auctionRepo.CountAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count total lots"})
		return
	}

	lotsByStatus, err := h.auctionRepo.CountAllByStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count by status"})
		return
	}

	lotsMatchingQuery, err := h.auctionRepo.GetEndingSoon()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matching lots"})
		return
	}

	allBiddingLots, err := h.auctionRepo.GetAllBiddingLotsWithEventDates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch all bidding lots"})
		return
	}

	querySummary := fmt.Sprintf(
		"WHERE LOWER(status) = 'bidding' AND ((sale_date > %s AND sale_date <= %s) OR (auction_end_time > %s AND auction_end_time <= %s))",
		now.Format("2006-01-02 15:04:05"),
		next24h.Format("2006-01-02 15:04:05"),
		now.Format("2006-01-02 15:04:05"),
		next24h.Format("2006-01-02 15:04:05"),
	)

	c.JSON(http.StatusOK, gin.H{
		"now":                 now.Format(time.RFC3339),
		"next_24h":            next24h.Format(time.RFC3339),
		"query_summary":       querySummary,
		"total_lots_in_db":    totalLots,
		"lots_by_status":      lotsByStatus,
		"lots_matching_query": lotsMatchingQuery,
		"all_bidding_lots":    allBiddingLots,
		"explanation": map[string]string{
			"lots_matching_query": "These are the lots the current scheduler query would find (LOWER(status)='bidding' AND (sale_date within next 24h OR auction_end_time within next 24h))",
			"all_bidding_lots":    "All lots with status=bidding, showing ALL date fields including event dates — helps identify which field actually holds the end date",
		},
	})
}
