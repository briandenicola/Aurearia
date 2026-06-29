package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

type QuickCaptureHandler struct {
	svc    *services.QuickCaptureService
	logger *services.Logger
}

func NewQuickCaptureHandler(svc *services.QuickCaptureService, logger *services.Logger) *QuickCaptureHandler {
	return &QuickCaptureHandler{svc: svc, logger: logger}
}

type quickCaptureDraftListResponse struct {
	Drafts []models.QuickCaptureDraft `json:"drafts"`
	Total  int64                      `json:"total"`
	Page   int                        `json:"page"`
	Limit  int                        `json:"limit"`
}

// CreateDraft creates a sparse Quick Capture draft for the authenticated user.
//
//	@Summary		Create Quick Capture draft
//	@Description	Creates an owner-scoped sparse draft with optional obverse/reverse/detail images.
//	@Tags			Quick Capture
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			workingTitle		formData	string	false	"Working title"
//	@Param			dateRange			formData	string	false	"Freeform date range"
//	@Param			era					formData	string	false	"Era"
//	@Param			acquisitionSource	formData	string	false	"Acquisition source"
//	@Param			purchasePrice		formData	number	false	"Purchase price"
//	@Param			notes				formData	string	false	"Notes"
//	@Param			obverseImage		formData	file	false	"Obverse image"
//	@Param			reverseImage		formData	file	false	"Reverse image"
//	@Param			detailImages		formData	file	false	"Detail images"
//	@Success		201					{object}	models.QuickCaptureDraft
//	@Failure		400					{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/quick-capture/drafts [post]
func (h *QuickCaptureHandler) CreateDraft(c *gin.Context) {
	input := services.CreateQuickCaptureDraftInput{
		UserID:            c.GetUint("userId"),
		WorkingTitle:      c.PostForm("workingTitle"),
		DateRange:         c.PostForm("dateRange"),
		Era:               c.PostForm("era"),
		AcquisitionSource: c.PostForm("acquisitionSource"),
		Notes:             c.PostForm("notes"),
	}
	if rawPrice := c.PostForm("purchasePrice"); rawPrice != "" {
		price, err := strconv.ParseFloat(rawPrice, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "purchasePrice must be a number"})
			return
		}
		input.PurchasePrice = &price
	}

	images, err := readQuickCaptureUploads(c)
	if err != nil {
		if errors.Is(err, services.ErrImageTooLarge) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image exceeds 20MB limit"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read uploaded image"})
		return
	}
	input.Images = images

	draft, err := h.svc.CreateDraft(input)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrQuickCaptureMinimumIdentity):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Add a working title, note, or image before saving a draft"})
		case errors.Is(err, services.ErrQuickCaptureInvalidPrice):
			c.JSON(http.StatusBadRequest, gin.H{"error": "purchasePrice must be zero or greater"})
		case errors.Is(err, services.ErrInvalidImageType):
			c.JSON(http.StatusBadRequest, gin.H{"error": "imageType must be one of: obverse, reverse, detail, other"})
		case errors.Is(err, services.ErrInvalidFileExt):
			c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed. Accepted: .jpg, .jpeg, .png, .gif, .webp"})
		case errors.Is(err, services.ErrImageTooLarge):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image exceeds 20MB limit"})
		default:
			if h.logger != nil {
				h.logger.Error("quick-capture", "Create draft failed: %v", err)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create quick capture draft"})
		}
		return
	}
	c.JSON(http.StatusCreated, draft)
}

// ListDrafts lists Quick Capture drafts for the authenticated user.
//
//	@Summary		List Quick Capture drafts
//	@Description	Lists owner-scoped Quick Capture drafts, defaulting to active drafts.
//	@Tags			Quick Capture
//	@Produce		json
//	@Param			status	query		string	false	"Draft status"	Enums(active, promoted, discarded)
//	@Param			page	query		int		false	"Page number"
//	@Param			limit	query		int		false	"Page size"
//	@Success		200		{object}	quickCaptureDraftListResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/quick-capture/drafts [get]
func (h *QuickCaptureHandler) ListDrafts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	status := models.QuickCaptureDraftStatus(c.DefaultQuery("status", string(models.QuickCaptureDraftStatusActive)))
	drafts, total, err := h.svc.ListDrafts(c.GetUint("userId"), status, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list quick capture drafts"})
		return
	}
	c.JSON(http.StatusOK, quickCaptureDraftListResponse{Drafts: drafts, Total: total, Page: page, Limit: limit})
}

// GetDraft returns a Quick Capture draft for the authenticated owner.
//
//	@Summary		Get Quick Capture draft
//	@Description	Returns one owner-scoped Quick Capture draft with image metadata.
//	@Tags			Quick Capture
//	@Produce		json
//	@Param			id	path		int	true	"Draft ID"
//	@Success		200	{object}	models.QuickCaptureDraft
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/quick-capture/drafts/{id} [get]
func (h *QuickCaptureHandler) GetDraft(c *gin.Context) {
	draftID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid draft ID"})
		return
	}
	draft, err := h.svc.GetDraft(c.GetUint("userId"), uint(draftID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Draft not found"})
		return
	}
	c.JSON(http.StatusOK, draft)
}

// promoteDraftRequest is the JSON body for POST /quick-capture/drafts/:id/promote.
type promoteDraftRequest struct {
	Confirm   bool `json:"confirm"`
	Overrides struct {
		Name             string   `json:"name"`
		Category         string   `json:"category"`
		Material         string   `json:"material"`
		Era              string   `json:"era"`
		PurchasePrice    *float64 `json:"purchasePrice"`
		PurchaseLocation string   `json:"purchaseLocation"`
		Notes            string   `json:"notes"`
	} `json:"overrides"`
}

// UpdateDraft updates an active Quick Capture draft for the authenticated owner.
//
//	@Summary		Update Quick Capture draft
//	@Description	Updates an active owner-scoped draft. Supports field changes and image replacement/removal.
//	@Tags			Quick Capture
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id					path		int		true	"Draft ID"
//	@Param			workingTitle		formData	string	false	"Working title"
//	@Param			notes				formData	string	false	"Notes"
//	@Param			dateRange			formData	string	false	"Date range"
//	@Param			era					formData	string	false	"Era"
//	@Param			acquisitionSource	formData	string	false	"Acquisition source"
//	@Param			purchasePrice		formData	number	false	"Purchase price"
//	@Param			removeImageIds		formData	string	false	"Comma-separated image IDs to remove"
//	@Param			replaceObverse		formData	bool	false	"Replace existing obverse images"
//	@Param			replaceReverse		formData	bool	false	"Replace existing reverse images"
//	@Param			obverseImage		formData	file	false	"New obverse image"
//	@Param			reverseImage		formData	file	false	"New reverse image"
//	@Param			detailImages		formData	file	false	"New detail images"
//	@Success		200					{object}	models.QuickCaptureDraft
//	@Failure		400					{object}	ErrorResponse
//	@Failure		404					{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/quick-capture/drafts/{id} [put]
func (h *QuickCaptureHandler) UpdateDraft(c *gin.Context) {
	draftID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid draft ID"})
		return
	}

	input := services.UpdateQuickCaptureDraftInput{
		UserID:            c.GetUint("userId"),
		WorkingTitle:      c.PostForm("workingTitle"),
		DateRange:         c.PostForm("dateRange"),
		Era:               c.PostForm("era"),
		AcquisitionSource: c.PostForm("acquisitionSource"),
		Notes:             c.PostForm("notes"),
		RemoveImageIDsRaw: c.PostForm("removeImageIds"),
		ReplaceObverse:    c.PostForm("replaceObverse") == "true",
		ReplaceReverse:    c.PostForm("replaceReverse") == "true",
	}
	if rawPrice := c.PostForm("purchasePrice"); rawPrice != "" {
		price, err := strconv.ParseFloat(rawPrice, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "purchasePrice must be a number"})
			return
		}
		input.PurchasePrice = &price
		input.PurchasePriceSet = true
	}

	images, err := readQuickCaptureUploads(c)
	if err != nil {
		if errors.Is(err, services.ErrImageTooLarge) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image exceeds 20MB limit"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read uploaded image"})
		return
	}
	input.NewImages = images

	draft, err := h.svc.UpdateDraft(c.GetUint("userId"), uint(draftID), input)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrQuickCaptureNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Draft not found"})
		case errors.Is(err, services.ErrQuickCaptureMinimumIdentity):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Add a working title, note, or image before saving a draft"})
		case errors.Is(err, services.ErrQuickCaptureInvalidPrice):
			c.JSON(http.StatusBadRequest, gin.H{"error": "purchasePrice must be zero or greater"})
		case errors.Is(err, services.ErrInvalidImageType):
			c.JSON(http.StatusBadRequest, gin.H{"error": "imageType must be one of: obverse, reverse, detail, other"})
		case errors.Is(err, services.ErrInvalidFileExt):
			c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed. Accepted: .jpg, .jpeg, .png, .gif, .webp"})
		case errors.Is(err, services.ErrImageTooLarge):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Image exceeds 20MB limit"})
		default:
			if h.logger != nil {
				h.logger.Error("quick-capture", "Update draft failed: %v", err)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quick capture draft"})
		}
		return
	}
	c.JSON(http.StatusOK, draft)
}

// DiscardDraft marks a Quick Capture draft as discarded for the authenticated owner.
// Idempotent: discarding an already-discarded draft returns the draft unchanged.
//
//	@Summary		Discard Quick Capture draft
//	@Description	Marks an active owner-scoped draft as discarded. Idempotent for already-discarded drafts.
//	@Tags			Quick Capture
//	@Produce		json
//	@Param			id	path		int	true	"Draft ID"
//	@Success		200	{object}	models.QuickCaptureDraft
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/quick-capture/drafts/{id}/discard [post]
func (h *QuickCaptureHandler) DiscardDraft(c *gin.Context) {
	draftID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid draft ID"})
		return
	}
	draft, err := h.svc.DiscardDraft(c.GetUint("userId"), uint(draftID))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrQuickCaptureNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Draft not found"})
		case errors.Is(err, services.ErrQuickCaptureDraftAlreadyPromoted):
			c.JSON(http.StatusConflict, gin.H{"error": "Promoted drafts cannot be discarded"})
		default:
			if h.logger != nil {
				h.logger.Error("quick-capture", "Discard draft failed: %v", err)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to discard quick capture draft"})
		}
		return
	}
	c.JSON(http.StatusOK, draft)
}

// PromoteDraft promotes an active Quick Capture draft into a normal Coin record.
// Idempotent: repeated calls return the existing promoted coin.
//
//	@Summary		Promote Quick Capture draft
//	@Description	Transactionally promotes a valid active draft into a normal Coin. Idempotent on repeat.
//	@Tags			Quick Capture
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Draft ID"
//	@Param			body	body		promoteDraftRequest		true	"Promotion request"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		404		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/quick-capture/drafts/{id}/promote [post]
func (h *QuickCaptureHandler) PromoteDraft(c *gin.Context) {
	draftID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid draft ID"})
		return
	}

	var req promoteDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if !req.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Complete required fields before promotion",
			"fields": map[string]string{"confirm": "confirm must be true to promote"},
		})
		return
	}

	input := services.PromoteDraftInput{
		Confirm: req.Confirm,
		Overrides: services.PromoteOverrides{
			Name:             req.Overrides.Name,
			Category:         req.Overrides.Category,
			Material:         req.Overrides.Material,
			Era:              req.Overrides.Era,
			PurchasePrice:    req.Overrides.PurchasePrice,
			PurchaseLocation: req.Overrides.PurchaseLocation,
			Notes:            req.Overrides.Notes,
		},
	}

	result, err := h.svc.PromoteDraft(c.GetUint("userId"), uint(draftID), input)
	if err != nil {
		var valErr *services.QuickCapturePromotionValidationError
		switch {
		case errors.As(err, &valErr):
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "Complete required fields before promotion",
				"fields": valErr.Fields,
			})
		case errors.Is(err, services.ErrQuickCaptureNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Draft not found"})
		case errors.Is(err, services.ErrQuickCaptureDraftConcurrentAction):
			c.JSON(http.StatusConflict, gin.H{"error": "Draft is discarded or currently being promoted"})
		default:
			if h.logger != nil {
				h.logger.Error("quick-capture", "Promote draft failed: %v", err)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote quick capture draft"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"draftId":         result.DraftID,
		"status":          "promoted",
		"coinId":          result.CoinID,
		"alreadyPromoted": result.AlreadyPromoted,
	})
}

func readQuickCaptureUploads(c *gin.Context) ([]services.QuickCaptureImageUpload, error) {
	images := make([]services.QuickCaptureImageUpload, 0, 4)
	addOne := func(fieldName, imageType string, primary bool) error {
		file, err := c.FormFile(fieldName)
		if err != nil {
			return nil
		}
		if file.Size > services.MaxImageUploadBytes {
			return services.ErrImageTooLarge
		}
		opened, err := file.Open()
		if err != nil {
			return err
		}
		defer opened.Close()
		data, err := io.ReadAll(opened)
		if err != nil {
			return err
		}
		images = append(images, services.QuickCaptureImageUpload{
			Filename:  file.Filename,
			Data:      data,
			ImageType: imageType,
			IsPrimary: primary,
		})
		return nil
	}
	if err := addOne("obverseImage", "obverse", true); err != nil {
		return nil, err
	}
	if err := addOne("reverseImage", "reverse", false); err != nil {
		return nil, err
	}
	form, err := c.MultipartForm()
	if err == nil && form != nil {
		for _, file := range form.File["detailImages"] {
			if file.Size > services.MaxImageUploadBytes {
				return nil, services.ErrImageTooLarge
			}
			opened, err := file.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(opened)
			opened.Close()
			if err != nil {
				return nil, err
			}
			images = append(images, services.QuickCaptureImageUpload{
				Filename:  file.Filename,
				Data:      data,
				ImageType: "detail",
			})
		}
	}
	return images, nil
}
