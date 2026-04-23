package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/gin-gonic/gin"
)

type ConversationHandler struct {
	repo *repository.ConversationRepository
}

func NewConversationHandler(repo *repository.ConversationRepository) *ConversationHandler {
	return &ConversationHandler{repo: repo}
}

// ConversationSummary is a lightweight view for listing conversations.
type ConversationSummary struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// List returns all saved conversations for the current user.
//
//	@Summary		List saved conversations
//	@Description	Returns all saved agent conversations for the current user, newest first.
//	@Tags			Agent
//	@Produce		json
//	@Success		200	{array}		ConversationSummary
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/conversations [get]
func (h *ConversationHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")

	var conversations []models.AgentConversation
	conversations, err := h.repo.List(userID)
	if err != nil {
		log.Printf("[handler] List: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list conversations"})
		return
	}

	summaries := make([]ConversationSummary, len(conversations))
	for i, conv := range conversations {
		summaries[i] = ConversationSummary{
			ID:        conv.ID,
			Title:     conv.Title,
			CreatedAt: conv.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: conv.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	c.JSON(http.StatusOK, summaries)
}

// Get returns a single conversation with full messages.
//
//	@Summary		Get conversation
//	@Description	Returns a saved conversation with its full message history.
//	@Tags			Agent
//	@Produce		json
//	@Param			id	path		int	true	"Conversation ID"
//	@Success		200	{object}	models.AgentConversation
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/conversations/{id} [get]
func (h *ConversationHandler) Get(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var conv models.AgentConversation
	found, err := h.repo.FindByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}
	conv = *found

	c.JSON(http.StatusOK, conv)
}

// Save creates or updates a conversation.
//
//	@Summary		Save conversation
//	@Description	Creates a new conversation or updates an existing one.
//	@Tags			Agent
//	@Accept			json
//	@Produce		json
//	@Param			body	body		object{id uint, title string, messages string}	true	"Conversation data"
//	@Success		200		{object}	models.AgentConversation
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/conversations [post]
func (h *ConversationHandler) Save(c *gin.Context) {
	userID := c.GetUint("userId")

	var body struct {
		ID       uint   `json:"id"`
		Title    string `json:"title" binding:"required"`
		Messages string `json:"messages" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title and messages are required"})
		return
	}

	// Update existing
	if body.ID > 0 {
		conv, err := h.repo.FindByID(body.ID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
			return
		}
		conv.Title = body.Title
		conv.Messages = body.Messages
		h.repo.Save(conv)
		c.JSON(http.StatusOK, conv)
		return
	}

	// Create new
	conv := models.AgentConversation{
		UserID:   userID,
		Title:    body.Title,
		Messages: body.Messages,
	}
	if err := h.repo.Create(&conv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save conversation"})
		return
	}
	c.JSON(http.StatusCreated, conv)
}

// Delete removes a saved conversation.
//
//	@Summary		Delete conversation
//	@Description	Deletes a saved agent conversation.
//	@Tags			Agent
//	@Produce		json
//	@Param			id	path		int	true	"Conversation ID"
//	@Success		200	{object}	object{message string}
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/agent/conversations/{id} [delete]
func (h *ConversationHandler) Delete(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	rowsAffected, _ := h.repo.Delete(uint(id), userID)
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation deleted"})
}
