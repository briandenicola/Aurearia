package handlers

import (
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

type JournalHandler struct {
	repo *repository.JournalRepository
}

func NewJournalHandler(repo *repository.JournalRepository) *JournalHandler {
	return &JournalHandler{repo: repo}
}

// ListEntries returns all journal entries for a coin.
//
//	@Summary		List journal entries
//	@Description	Returns all activity log entries for a coin, newest first.
//	@Tags			Journal
//	@Produce		json
//	@Param			id	path		int	true	"Coin ID"
//	@Success		200	{array}		models.CoinJournal
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/journal [get]
func (h *JournalHandler) ListEntries(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	// Verify coin ownership
	if !h.repo.CoinExists(uint(coinID), userID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	entries, _ := h.repo.GetEntries(uint(coinID), userID)

	c.JSON(http.StatusOK, entries)
}

// AddEntry creates a new journal entry for a coin.
//
//	@Summary		Add journal entry
//	@Description	Creates a new activity log entry for a coin.
//	@Tags			Journal
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Coin ID"
//	@Param			body	body		object{entry string}	true	"Journal entry"
//	@Success		201		{object}	models.CoinJournal
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/journal [post]
func (h *JournalHandler) AddEntry(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	// Verify coin ownership
	if !h.repo.CoinExists(uint(coinID), userID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	var body struct {
		Entry string `json:"entry" binding:"required,max=10000"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entry text is required"})
		return
	}

	entry := models.CoinJournal{
		CoinID: uint(coinID),
		UserID: userID,
		Entry:  body.Entry,
	}

	if err := h.repo.CreateEntry(&entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create journal entry"})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

// DeleteEntry removes a journal entry.
//
//	@Summary		Delete journal entry
//	@Description	Deletes an activity log entry. Only the entry owner can delete it.
//	@Tags			Journal
//	@Produce		json
//	@Param			id		path		int	true	"Coin ID"
//	@Param			entryId	path		int	true	"Entry ID"
//	@Success		200		{object}	object{message string}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/journal/{entryId} [delete]
func (h *JournalHandler) DeleteEntry(c *gin.Context) {
	userID := c.GetUint("userId")
	entryID, ok := parseID(c, "entryId")
	if !ok {
		return
	}

	rowsAffected, _ := h.repo.DeleteEntry(entryID, userID)
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entry not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Entry deleted"})
}
