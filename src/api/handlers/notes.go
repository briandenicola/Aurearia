package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type NoteHandler struct {
	service *services.NoteService
}

type noteRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func NewNoteHandler(service *services.NoteService) *NoteHandler {
	return &NoteHandler{service: service}
}

// List returns the authenticated user's notes.
//
//	@Summary		List notes
//	@Description	Returns all notes owned by the authenticated user.
//	@Tags			Notes
//	@Produce		json
//	@Success		200	{object}	NoteListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/notes [get]
func (h *NoteHandler) List(c *gin.Context) {
	notes, err := h.service.List(c.GetUint("userId"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list notes", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notes": notes})
}

// Get returns one note.
//
//	@Summary		Get note
//	@Description	Returns one note owned by the authenticated user.
//	@Tags			Notes
//	@Produce		json
//	@Param			id	path		int	true	"Note ID"
//	@Success		200	{object}	models.Note
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/notes/{id} [get]
func (h *NoteHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	note, err := h.service.Get(id, c.GetUint("userId"))
	if err != nil {
		respondNoteError(c, err)
		return
	}
	c.JSON(http.StatusOK, note)
}

// Create creates a note.
//
//	@Summary		Create note
//	@Description	Creates a Markdown-capable note for the authenticated user.
//	@Tags			Notes
//	@Accept			json
//	@Produce		json
//	@Param			body	body		noteRequest	true	"Note"
//	@Success		201		{object}	models.Note
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/notes [post]
func (h *NoteHandler) Create(c *gin.Context) {
	var req noteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note payload"})
		return
	}
	note, err := h.service.Create(c.GetUint("userId"), services.NoteInput{Title: req.Title, Body: req.Body})
	if err != nil {
		respondNoteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, note)
}

// Update updates a note.
//
//	@Summary		Update note
//	@Description	Updates one note owned by the authenticated user.
//	@Tags			Notes
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int			true	"Note ID"
//	@Param			body	body		noteRequest	true	"Note"
//	@Success		200		{object}	models.Note
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/notes/{id} [put]
func (h *NoteHandler) Update(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req noteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note payload"})
		return
	}
	note, err := h.service.Update(id, c.GetUint("userId"), services.NoteInput{Title: req.Title, Body: req.Body})
	if err != nil {
		respondNoteError(c, err)
		return
	}
	c.JSON(http.StatusOK, note)
}

// Delete deletes a note.
//
//	@Summary		Delete note
//	@Description	Deletes one note owned by the authenticated user.
//	@Tags			Notes
//	@Produce		json
//	@Param			id	path	int	true	"Note ID"
//	@Success		204
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/notes/{id} [delete]
func (h *NoteHandler) Delete(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(id, c.GetUint("userId")); err != nil {
		respondNoteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func respondNoteError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNoteNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
	case errors.Is(err, services.ErrNoteTitleRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
	case errors.Is(err, services.ErrNoteTitleTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title must be 200 characters or fewer"})
	case errors.Is(err, services.ErrNoteBodyTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Body must be 20000 characters or fewer"})
	default:
		respondError(c, http.StatusInternalServerError, "Failed to process note", err)
	}
}
