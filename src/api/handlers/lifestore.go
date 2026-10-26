package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// LifestoreHandler serves the read-only feed the life store reads.
type LifestoreHandler struct {
	svc *services.LifestoreRecordsService
}

// NewLifestoreHandler creates a LifestoreHandler.
func NewLifestoreHandler(svc *services.LifestoreRecordsService) *LifestoreHandler {
	return &LifestoreHandler{svc: svc}
}

// ListRecords returns every coin the caller owns, a page at a time.
//
//	@Summary		List records for the life store
//	@Description	Returns every coin the authenticated user owns (collection, wishlist and sold, private ones included), with its tags, sets, catalog references, journal and value history, in a stable order a page at a time. A reader takes the whole list each time and treats a coin missing from it as deleted. Start without page and pass back next until it is empty. A read-only API key is enough. Private attribution card images are not counted.
//	@Tags			Lifestore
//	@Produce		json
//	@Param			page	query		string	false	"Token from the previous page's next; omit for the first page"
//	@Param			limit	query		int		false	"Records per page (max 500)"	default(100)
//	@Success		200		{object}	services.LifestoreRecordsPage
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Security		ApiKeyAuth
//	@Router			/lifestore/records [get]
func (h *LifestoreHandler) ListRecords(c *gin.Context) {
	userID := c.GetUint("userId")
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := h.svc.Page(userID, c.Query("page"), limit)
	if errors.Is(err, services.ErrInvalidLifestorePage) {
		respondError(c, http.StatusBadRequest, "page is not a token this server gave out; start again without it", err)
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list records", err)
		return
	}
	c.JSON(http.StatusOK, page)
}
