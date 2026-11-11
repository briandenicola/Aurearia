package handlers

import (
	"errors"
	"fmt"
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
	svc    *services.PaperlessInvoiceService
	logger *services.Logger
}

func NewPaperlessInvoiceHandler(svc *services.PaperlessInvoiceService, logger *services.Logger) *PaperlessInvoiceHandler {
	return &PaperlessInvoiceHandler{svc: svc, logger: logger}
}

// fail logs the request's paperless error to the app logger (Admin → Logs)
// and answers with the same description the user sees.
func (h *PaperlessInvoiceHandler) fail(c *gin.Context, status int, msg string, err error) {
	h.logger.Error("paperless", "%s %s: %s (%v)", c.Request.Method, c.Request.URL.Path, msg, err)
	c.JSON(status, gin.H{"error": msg})
}

func (h *PaperlessInvoiceHandler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrPaperlessCoinNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
	case errors.Is(err, services.ErrPaperlessNoDocument):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrPaperlessUnsupportedFile):
		h.fail(c, http.StatusBadGateway, err.Error(), err)
	case errors.Is(err, services.ErrPaperlessNotConfigured):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrPaperlessTagNotFound):
		h.fail(c, http.StatusBadRequest, services.DescribePaperlessError(err), err)
	case errors.Is(err, services.ErrPaperlessTaskFailed):
		h.fail(c, http.StatusBadGateway, err.Error(), err)
	case errors.Is(err, services.ErrOutboundTargetBlocked):
		h.fail(c, http.StatusBadRequest, services.DescribePaperlessError(err), err)
	default:
		h.fail(c, http.StatusBadGateway, services.DescribePaperlessError(err), err)
	}
}

// TestConnection checks the saved paperless-ngx URL and API token.
//
//	@Summary	Test paperless-ngx connection
//	@Tags		Users
//	@Produce	json
//	@Success	200	{object}	map[string]bool
//	@Failure	400	{object}	ErrorResponse
//	@Failure	502	{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/paperless/test [post]
func (h *PaperlessInvoiceHandler) TestConnection(c *gin.Context) {
	if err := h.svc.TestConnection(c.Request.Context(), c.GetUint("userId")); err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
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
		h.fail(c, http.StatusBadRequest, "Could not read invoice file", err)
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

func (h *PaperlessInvoiceHandler) streamDocument(c *gin.Context, kind services.PaperlessFileKind) {
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	file, err := h.svc.OpenDocument(c.Request.Context(), c.GetUint("userId"), coinID, kind)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	defer file.Body.Close()
	extra := map[string]string{
		// The URL is per coin, not per document: replacing or unlinking the
		// invoice must never show a cached copy of the previous file.
		"Cache-Control":           "no-store",
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "default-src 'none'; sandbox",
		"Content-Disposition":     fmt.Sprintf("inline; filename=%q", fmt.Sprintf("invoice-%d-%s", coinID, kind)),
	}
	c.DataFromReader(http.StatusOK, file.ContentLength, file.ContentType, file.Body, extra)
}

// InvoiceDocument streams the viewable rendition of the coin's linked paperless-ngx document.
//
//	@Summary	View coin invoice document from paperless-ngx
//	@Tags		Coins
//	@Produce	application/pdf
//	@Produce	image/png
//	@Produce	image/jpeg
//	@Param		id	path	int	true	"Coin ID"
//	@Success	200
//	@Failure	404	{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/coins/{id}/invoice/paperless/document [get]
func (h *PaperlessInvoiceHandler) InvoiceDocument(c *gin.Context) {
	h.streamDocument(c, services.PaperlessFilePreview)
}

// InvoiceThumbnail streams the page-1 thumbnail of the coin's linked paperless-ngx document.
//
//	@Summary	Thumbnail of coin invoice document from paperless-ngx
//	@Tags		Coins
//	@Produce	image/png
//	@Produce	image/webp
//	@Param		id	path	int	true	"Coin ID"
//	@Success	200
//	@Failure	404	{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/coins/{id}/invoice/paperless/thumbnail [get]
func (h *PaperlessInvoiceHandler) InvoiceThumbnail(c *gin.Context) {
	h.streamDocument(c, services.PaperlessFileThumbnail)
}

// UnlinkInvoice removes the coin's link to its paperless-ngx document. The document is not deleted from paperless-ngx.
//
//	@Summary	Unlink coin invoice document
//	@Tags		Coins
//	@Param		id	path	int	true	"Coin ID"
//	@Success	204
//	@Failure	404	{object}	ErrorResponse
//	@Security	BearerAuth
//	@Router		/coins/{id}/invoice/paperless [delete]
func (h *PaperlessInvoiceHandler) UnlinkInvoice(c *gin.Context) {
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.UnlinkDocument(c.GetUint("userId"), coinID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
