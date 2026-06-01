package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type CoinIntakeHandler struct {
	service *services.CoinIntakeService
}

func NewCoinIntakeHandler(service *services.CoinIntakeService, logger *services.Logger) *CoinIntakeHandler {
	_ = logger
	return &CoinIntakeHandler{service: service}
}

// CreateDraft generates an AI intake draft from uploaded observation images.
//
//	@Summary		Create coin intake draft
//	@Description	Uploads one or more coin images and optionally a coin card image, then returns an editable draft.
//	@Tags			Coins
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			images			formData	file	true	"Coin images (use multiple files)"
//	@Param			coinCardImage	formData	file	false	"Optional coin card image"
//	@Success		200	{object}	IntakeDraftCreateResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/intake/draft [post]
func (h *CoinIntakeHandler) CreateDraft(c *gin.Context) {
	userID := c.GetUint("userId")

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

	var coinCardImage *string
	if cardFiles := form.File["coinCardImage"]; len(cardFiles) > 0 {
		cardURI, err := fileToDataURI(cardFiles[0])
		if err != nil {
			respondError(c, http.StatusBadRequest, "Invalid coin card image", err)
			return
		}
		coinCardImage = &cardURI
	}

	draft, err := h.service.CreateDraft(userID, services.IntakeDraftRequest{
		Images:        images,
		CoinCardImage: coinCardImage,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create intake draft", err)
		return
	}

	c.JSON(http.StatusOK, draft)
}

// CommitDraft confirms a draft and creates a new coin record transactionally.
//
//	@Summary		Commit coin intake draft
//	@Description	Explicitly confirms and commits an intake draft into a persistent coin record.
//	@Tags			Coins
//	@Accept			json
//	@Produce		json
//	@Param			request	body		IntakeDraftCommitRequest	true	"Commit request with explicit confirmation and optional overrides"
//	@Success		200		{object}	IntakeDraftCommitResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/intake/commit [post]
func (h *CoinIntakeHandler) CommitDraft(c *gin.Context) {
	userID := c.GetUint("userId")

	var req services.IntakeCommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	result, err := h.service.CommitDraft(userID, req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrIntakeConfirmMissing):
			c.JSON(http.StatusBadRequest, gin.H{"error": "confirm must be true to commit"})
		case errors.Is(err, services.ErrIntakeDraftNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Draft not found"})
		case errors.Is(err, services.ErrIntakeDraftConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "Draft is no longer confirmable"})
		default:
			respondError(c, http.StatusInternalServerError, "Failed to commit intake draft", err)
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

func fileToDataURI(fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader == nil {
		return "", fmt.Errorf("missing image file")
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		return "", fmt.Errorf("unsupported image type: %s", ext)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("empty image file")
	}
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("unsupported content type: %s", contentType)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded), nil
}
