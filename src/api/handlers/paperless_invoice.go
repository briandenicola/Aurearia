package handlers

import (
	"errors"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

const maxPaperlessInvoiceBytes = 25 << 20

var (
	paperlessTaskIDPattern      = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
	allowedPaperlessInvoiceExts = map[string]bool{
		".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".heic": true, ".webp": true, ".tif": true, ".tiff": true,
	}
)

type PaperlessInvoiceHandler struct {
	svc *services.PaperlessInvoiceService
}

func NewPaperlessInvoiceHandler(svc *services.PaperlessInvoiceService) *PaperlessInvoiceHandler {
	return &PaperlessInvoiceHandler{svc: svc}
}

func (h *PaperlessInvoiceHandler) respondServiceError(c *gin.Context, err error) {
	var paperlessErr *services.PaperlessError
	switch {
	case errors.Is(err, services.ErrPaperlessCoinNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
	case errors.Is(err, services.ErrPaperlessNotConfigured):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrPaperlessTaskFailed):
		respondError(c, http.StatusBadGateway, err.Error(), err)
	case errors.Is(err, services.ErrOutboundTargetBlocked):
		respondError(c, http.StatusBadRequest, "paperless-ngx URL points at a blocked address (loopback or link-local)", err)
	case errors.As(err, &paperlessErr) && (paperlessErr.StatusCode == http.StatusUnauthorized || paperlessErr.StatusCode == http.StatusForbidden):
		respondError(c, http.StatusBadGateway, "paperless-ngx rejected the API token", err)
	default:
		respondError(c, http.StatusBadGateway, "Could not reach paperless-ngx", err)
	}
}

// UploadInvoice uploads an invoice file to the user's paperless-ngx and links it to the coin.
//
//	@Summary	Upload coin invoice to paperless-ngx
//	@Tags		Coins
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		id		path		int		true	"Coin ID"
//	@Param		file	formData	file	true	"Invoice (PDF or image)"
//	@Success	200		{object}	services.PaperlessInvoiceResult
//	@Success	202		{object}	services.PaperlessInvoiceResult
//	@Security	BearerAuth
//	@Router		/coins/{id}/invoice/paperless [post]
func (h *PaperlessInvoiceHandler) UploadInvoice(c *gin.Context) {
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPaperlessInvoiceBytes+(1<<20))
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "An invoice file is required (max 25 MB)"})
		return
	}
	if fileHeader.Size > maxPaperlessInvoiceBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Invoice file is larger than 25 MB"})
		return
	}
	filename := filepath.Base(strings.TrimSpace(fileHeader.Filename))
	if !allowedPaperlessInvoiceExts[strings.ToLower(filepath.Ext(filename))] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invoice must be a PDF or image"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		respondError(c, http.StatusBadRequest, "Could not read invoice file", err)
		return
	}
	defer file.Close()

	result, err := h.svc.UploadInvoice(c.Request.Context(), c.GetUint("userId"), coinID, filename, fileHeader.Header.Get("Content-Type"), file)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	status := http.StatusOK
	if result.Status != "linked" {
		status = http.StatusAccepted
	}
	c.JSON(status, result)
}

// ResolveInvoiceTask re-checks a pending paperless-ngx consumption task.
//
//	@Summary	Resolve pending paperless-ngx invoice upload
//	@Tags		Coins
//	@Produce	json
//	@Param		id		path		int		true	"Coin ID"
//	@Param		taskId	path		string	true	"paperless-ngx task id"
//	@Success	200		{object}	services.PaperlessInvoiceResult
//	@Success	202		{object}	services.PaperlessInvoiceResult
//	@Security	BearerAuth
//	@Router		/coins/{id}/invoice/paperless/tasks/{taskId} [post]
func (h *PaperlessInvoiceHandler) ResolveInvoiceTask(c *gin.Context) {
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	taskID := c.Param("taskId")
	if !paperlessTaskIDPattern.MatchString(taskID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task id"})
		return
	}
	result, err := h.svc.ResolveTask(c.Request.Context(), c.GetUint("userId"), coinID, taskID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	status := http.StatusOK
	if result.Status != "linked" {
		status = http.StatusAccepted
	}
	c.JSON(status, result)
}
