package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

const maxTagsPerUser = 50

// TagHandler handles tag-related HTTP requests.
type TagHandler struct {
	repo *repository.TagRepository
}

// NewTagHandler creates a new TagHandler.
func NewTagHandler(repo *repository.TagRepository) *TagHandler {
	return &TagHandler{repo: repo}
}

// List returns all tags for the authenticated user.
func (h *TagHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")
	tags, err := h.repo.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tags"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// Create adds a new tag for the authenticated user.
func (h *TagHandler) Create(c *gin.Context) {
	userID := c.GetUint("userId")

	var body struct {
		Name  string `json:"name" binding:"required"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" || len(name) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tag name must be 1-50 characters"})
		return
	}

	// Check max tags limit
	count, err := h.repo.CountByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check tag count"})
		return
	}
	if count >= maxTagsPerUser {
		c.JSON(http.StatusConflict, gin.H{"error": "Maximum of 50 tags allowed"})
		return
	}

	// Check case-insensitive uniqueness
	exists, err := h.repo.ExistsByName(userID, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check tag name"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "A tag with this name already exists"})
		return
	}

	color := "#6b7280"
	if body.Color != "" {
		color = body.Color
	}

	tag := models.Tag{
		UserID: userID,
		Name:   name,
		Color:  color,
	}
	if err := h.repo.Create(&tag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
		return
	}
	c.JSON(http.StatusCreated, tag)
}

// Update modifies a tag's name and/or color.
func (h *TagHandler) Update(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}
	tagID := uint(id)

	tag, err := h.repo.GetByID(tagID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
		return
	}

	var body struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updates := make(map[string]interface{})

	if body.Name != nil {
		name := strings.TrimSpace(*body.Name)
		if name == "" || len(name) > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tag name must be 1-50 characters"})
			return
		}
		// Check uniqueness if name is changing
		if !strings.EqualFold(name, tag.Name) {
			exists, err := h.repo.ExistsByName(userID, name)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check tag name"})
				return
			}
			if exists {
				c.JSON(http.StatusConflict, gin.H{"error": "A tag with this name already exists"})
				return
			}
		}
		updates["name"] = name
	}
	if body.Color != nil {
		updates["color"] = *body.Color
	}

	if len(updates) == 0 {
		c.JSON(http.StatusOK, tag)
		return
	}

	if err := h.repo.Update(tag, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tag"})
		return
	}

	// Re-fetch to return updated state
	tag, err = h.repo.GetByID(tagID, userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to reload tag", err)
		return
	}
	c.JSON(http.StatusOK, tag)
}

// Delete removes a tag and all its coin associations.
func (h *TagHandler) Delete(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	if err := h.repo.Delete(uint(id), userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Tag deleted"})
}

// AttachToCoin adds a tag to a coin.
func (h *TagHandler) AttachToCoin(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	var body struct {
		TagID uint `json:"tagId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tagId is required"})
		return
	}

	if err := h.repo.AttachToCoin(uint(coinID), body.TagID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin or tag not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Tag added"})
}

// DetachFromCoin removes a tag from a coin.
func (h *TagHandler) DetachFromCoin(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}
	tagID, err := strconv.ParseUint(c.Param("tagId"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	if err := h.repo.DetachFromCoin(uint(coinID), uint(tagID), userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Tag removed"})
}
