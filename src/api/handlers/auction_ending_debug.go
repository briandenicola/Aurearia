package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuctionEndingDebugHandler exposes debug/diagnostic endpoints for the auction ending scheduler.
type AuctionEndingDebugHandler struct {
	db          *gorm.DB
	auctionRepo *repository.AuctionLotRepository
}

// NewAuctionEndingDebugHandler constructs a new debug handler.
func NewAuctionEndingDebugHandler(db *gorm.DB, auctionRepo *repository.AuctionLotRepository) *AuctionEndingDebugHandler {
	return &AuctionEndingDebugHandler{
		db:          db,
		auctionRepo: auctionRepo,
	}
}

// DebugGetAuctionEndingInfo returns comprehensive diagnostic data for the auction ending scheduler.
//
// @Summary Debug auction ending scheduler
// @Description Returns diagnostic info: today's date, total lots, lots by status, lots matching the scheduler query, and all BIDDING lots with all their date fields populated
// @Tags admin
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/auction-ending/debug [get]
func (h *AuctionEndingDebugHandler) DebugGetAuctionEndingInfo(c *gin.Context) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// 1. Total lots in DB
	var totalLots int64
	if err := h.db.Model(&models.AuctionLot{}).Count(&totalLots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count total lots"})
		return
	}

	// 2. Lots by status
	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusCounts []StatusCount
	if err := h.db.Model(&models.AuctionLot{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count by status"})
		return
	}
	lotsByStatus := make(map[string]int64)
	for _, sc := range statusCounts {
		lotsByStatus[sc.Status] = sc.Count
	}

	// 3. Lots matching the current scheduler query (delegates to repo)
	lotsMatchingQuery, err := h.auctionRepo.GetEndingToday()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matching lots"})
		return
	}

	// 4. ALL bidding lots with enriched date info (including event dates) — delegates to repo
	allBiddingLots, err := h.auctionRepo.GetAllBiddingLotsWithEventDates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch all bidding lots"})
		return
	}

	// 5. Build summary of query logic
	querySummary := fmt.Sprintf(
		"WHERE status = 'bidding' AND ((sale_date >= %s AND sale_date < %s) OR (auction_end_time >= %s AND auction_end_time < %s))",
		startOfDay.Format("2006-01-02 15:04:05"),
		endOfDay.Format("2006-01-02 15:04:05"),
		startOfDay.Format("2006-01-02 15:04:05"),
		endOfDay.Format("2006-01-02 15:04:05"),
	)

	c.JSON(http.StatusOK, gin.H{
		"now":                  now.Format(time.RFC3339),
		"today_start":          startOfDay.Format(time.RFC3339),
		"today_end":            endOfDay.Format(time.RFC3339),
		"query_summary":        querySummary,
		"total_lots_in_db":     totalLots,
		"lots_by_status":       lotsByStatus,
		"lots_matching_query":  lotsMatchingQuery,
		"all_bidding_lots":     allBiddingLots,
		"explanation": map[string]string{
			"lots_matching_query": "These are the lots the current scheduler query would find (status=bidding AND (sale_date today OR auction_end_time today))",
			"all_bidding_lots":    "All lots with status=bidding, showing ALL date fields including event dates — helps identify which field actually holds the end date",
		},
	})
}
