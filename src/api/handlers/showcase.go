package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

type ShowcaseHandler struct {
	repo *repository.ShowcaseRepository
}

func NewShowcaseHandler(repo *repository.ShowcaseRepository) *ShowcaseHandler {
	return &ShowcaseHandler{repo: repo}
}

// ListShowcases returns all showcases for the current user.
func (h *ShowcaseHandler) ListShowcases(c *gin.Context) {
	userID := c.GetUint("userId")
	showcases, err := h.repo.ListByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list showcases"})
		return
	}
	// Add coin counts
	result := make([]gin.H, 0, len(showcases))
	for _, s := range showcases {
		entries, _ := h.repo.GetShowcaseCoinEntries(s.ID)
		result = append(result, gin.H{
			"id":          s.ID,
			"slug":        s.Slug,
			"title":       s.Title,
			"description": s.Description,
			"isActive":    s.IsActive,
			"coinCount":   len(entries),
			"createdAt":   s.CreatedAt,
			"updatedAt":   s.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"showcases": result})
}

// GetShowcase returns a single showcase with its coins for the current user.
func (h *ShowcaseHandler) GetShowcase(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid showcase ID"})
		return
	}

	showcase, err := h.repo.GetByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Showcase not found"})
		return
	}

	coins, _ := h.repo.GetShowcaseCoins(showcase.ID)
	entries, _ := h.repo.GetShowcaseCoinEntries(showcase.ID)

	coinIDs := make([]uint, 0, len(entries))
	for _, e := range entries {
		coinIDs = append(coinIDs, e.CoinID)
	}

	c.JSON(http.StatusOK, gin.H{
		"showcase": gin.H{
			"id":          showcase.ID,
			"slug":        showcase.Slug,
			"title":       showcase.Title,
			"description": showcase.Description,
			"isActive":    showcase.IsActive,
			"coinIds":     coinIDs,
			"createdAt":   showcase.CreatedAt,
			"updatedAt":   showcase.UpdatedAt,
		},
		"coins": limitCoinDataSlice(coins),
	})
}

// CreateShowcase creates a new showcase.
func (h *ShowcaseHandler) CreateShowcase(c *gin.Context) {
	userID := c.GetUint("userId")

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
		return
	}

	showcase := models.Showcase{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		IsActive:    true,
	}

	if err := h.repo.Create(&showcase); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create showcase"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":   showcase.ID,
		"slug": showcase.Slug,
	})
}

// UpdateShowcase updates a showcase's metadata.
func (h *ShowcaseHandler) UpdateShowcase(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid showcase ID"})
		return
	}

	showcase, err := h.repo.GetByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Showcase not found"})
		return
	}

	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		IsActive    *bool   `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Title != nil {
		showcase.Title = *req.Title
	}
	if req.Description != nil {
		showcase.Description = *req.Description
	}
	if req.IsActive != nil {
		showcase.IsActive = *req.IsActive
	}

	if err := h.repo.Update(showcase); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update showcase"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Showcase updated"})
}

// DeleteShowcase deletes a showcase.
func (h *ShowcaseHandler) DeleteShowcase(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid showcase ID"})
		return
	}

	if err := h.repo.Delete(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete showcase"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Showcase deleted"})
}

// SetShowcaseCoins sets the coins in a showcase (replaces all).
func (h *ShowcaseHandler) SetShowcaseCoins(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid showcase ID"})
		return
	}

	var req struct {
		CoinIDs []uint `json:"coinIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if req.CoinIDs == nil {
		req.CoinIDs = []uint{}
	}

	if err := h.repo.SetCoins(uint(id), userID, req.CoinIDs); err != nil {
		log.Printf("[handler] SetShowcaseCoins: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Showcase coins updated"})
}

// GetPublicShowcase returns a showcase by slug (no auth required).
func (h *ShowcaseHandler) GetPublicShowcase(c *gin.Context) {
	slug := c.Param("slug")
	showcase, err := h.repo.GetBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Showcase not found"})
		return
	}

	coins, _ := h.repo.GetShowcaseCoins(showcase.ID)
	owner, _ := h.repo.GetOwnerUsername(showcase.UserID)

	c.JSON(http.StatusOK, gin.H{
		"showcase": gin.H{
			"title":       showcase.Title,
			"description": showcase.Description,
			"owner":       owner,
			"createdAt":   showcase.CreatedAt,
		},
		"coins": limitCoinDataSlice(coins),
	})
}

// limitCoinDataSlice returns a limited view of coins for display.
func limitCoinDataSlice(coins []models.Coin) []gin.H {
	result := make([]gin.H, 0, len(coins))
	for _, coin := range coins {
		images := make([]gin.H, 0, len(coin.Images))
		for _, img := range coin.Images {
			images = append(images, gin.H{
				"id":        img.ID,
				"filePath":  img.FilePath,
				"imageType": img.ImageType,
			})
		}

		tags := make([]gin.H, 0)
		for _, tag := range coin.Tags {
			tags = append(tags, gin.H{
				"id":    tag.ID,
				"name":  tag.Name,
				"color": tag.Color,
			})
		}

		result = append(result, gin.H{
			"id":           coin.ID,
			"name":         coin.Name,
			"era":          coin.Era,
			"category":     coin.Category,
			"grade":        coin.Grade,
			"material":     coin.Material,
			"ruler":        coin.Ruler,
			"denomination": coin.Denomination,
			"notes":        coin.Notes,
			"images":       images,
			"tags":         tags,
		})
	}
	return result
}
