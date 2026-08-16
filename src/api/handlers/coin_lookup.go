package handlers

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type CoinLookupHandler struct {
	service *services.CoinLookupService
	logger  *services.Logger
}

func NewCoinLookupHandler(service *services.CoinLookupService, logger *services.Logger) *CoinLookupHandler {
	return &CoinLookupHandler{
		service: service,
		logger:  logger,
	}
}

// Lookup performs coin lookup from uploaded images.
//
//	@Summary		Coin lookup from images
//	@Description	Analyzes coin/slab images, preserves NGC-first behavior, and proposes bounded Numista evidence/query without running Numista lookup.
//	@Tags			Coins
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			images	formData	file	true	"Coin or slab images (use multiple files)"
//	@Param			imageRoles	formData	[]string	false	"Semantic role for each image: obverse, reverse, or notes"	collectionFormat(multi)
//	@Param			notes	formData	string	false	"Collector-provided identification context (max 2000 characters)"
//	@Success		200	{object}	CoinLookupSwaggerResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/lookup [post]
func (h *CoinLookupHandler) Lookup(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")

	logger.Info("coin-lookup-handler", "Lookup request from user %d", userID)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid multipart form data"})
		return
	}

	var images []string
	for _, fileHeader := range form.File["images"] {
		dataURI, err := fileToDataURI(fileHeader)
		if err != nil {
			respondError(c, http.StatusBadRequest, "Invalid image upload", err)
			return
		}
		images = append(images, dataURI)
	}

	if len(images) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one image is required"})
		return
	}

	imageRoles := form.Value["imageRoles"]
	if len(imageRoles) > 0 {
		if len(imageRoles) != len(images) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Each image must have one image role"})
			return
		}
		for _, role := range imageRoles {
			if role != "obverse" && role != "reverse" && role != "notes" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Image roles must be obverse, reverse, or notes"})
				return
			}
		}
	}

	notes := strings.TrimSpace(c.PostForm("notes"))
	if utf8.RuneCountInString(notes) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Notes must be 2000 characters or fewer"})
		return
	}

	logger.Info("coin-lookup-handler", "Processing %d images for lookup", len(images))

	result, err := h.service.Lookup(c.Request.Context(), userID, services.CoinLookupRequest{
		Images:     images,
		ImageRoles: imageRoles,
		Notes:      notes,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Coin lookup failed", err)
		return
	}

	logger.Info("coin-lookup-handler", "Lookup completed: NGC=%v, Numista proposal=%v",
		result.ExtractedData.NGC != nil, result.ProposedNumistaQuery != "")

	c.JSON(http.StatusOK, result)
}
