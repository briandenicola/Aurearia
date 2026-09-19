package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

const WishlistURLBodyLimit int64 = 4 * 1024

type WishlistURLHandler struct {
	service *services.WishlistURLService
}

type wishlistURLRequest struct {
	URL string `json:"url"`
}

func NewWishlistURLHandler(service *services.WishlistURLService) *WishlistURLHandler {
	return &WishlistURLHandler{service: service}
}

// Analyze retrieves one public listing page and returns a transient wishlist proposal.
//
//	@Summary		Analyze wishlist listing URL
//	@Description	Retrieves one bounded public HTML page and returns an editable transient coin proposal. No coin is created.
//	@Tags			Wishlist
//	@Accept			json
//	@Produce		json
//	@Param			request	body		wishlistURLRequest	true	"Public listing URL"
//	@Success		200		{object}	services.WishlistURLAnalysis
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		413		{object}	ErrorResponse
//	@Failure		422		{object}	ErrorResponse
//	@Failure		502		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/wishlist/url-intake/analyze [post]
func (h *WishlistURLHandler) Analyze(c *gin.Context) {
	userID := c.GetUint("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, WishlistURLBodyLimit)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var request wishlistURLRequest
	if err := decoder.Decode(&request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request payload is too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	result, err := h.service.Analyze(c.Request.Context(), userID, request.URL)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrWishlistURLInvalid), services.IsOutboundTargetBlockedError(err):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "URL must identify a public HTTP(S) listing"})
		case errors.Is(err, services.ErrWishlistURLFetchFailed):
			c.JSON(http.StatusBadGateway, gin.H{"error": "Unable to retrieve the listing page"})
		default:
			respondError(c, http.StatusBadGateway, "Unable to analyze the listing page", err)
		}
		return
	}
	c.JSON(http.StatusOK, result)
}
