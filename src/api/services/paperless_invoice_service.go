package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrPaperlessNotConfigured   = errors.New("paperless-ngx URL and API token are required in Settings → Connections")
	ErrPaperlessTaskFailed      = errors.New("paperless-ngx could not consume the document")
	ErrPaperlessCoinNotFound    = errors.New("coin not found")
	ErrPaperlessNoDocument      = errors.New("no paperless-ngx document is linked to this coin")
	ErrPaperlessUnsupportedFile = errors.New("paperless-ngx returned a file type that cannot be displayed")
)

// Size caps for documents streamed through Aurearia.
const (
	MaxPaperlessPreviewBytes   = 50 << 20
	MaxPaperlessThumbnailBytes = 2 << 20
)

// PaperlessInvoiceResult reports where an invoice upload ended up.
type PaperlessInvoiceResult struct {
	Status     string `json:"status"` // "linked" or "pending"
	TaskID     string `json:"taskId"`
	DocumentID *int   `json:"documentId,omitempty"`
}

type PaperlessInvoiceService struct {
	coinRepo     *repository.CoinRepository
	userRepo     *repository.UserRepository
	credentials  *CredentialEncryptionService
	client       PaperlessClient
	pollInterval time.Duration
	pollTimeout  time.Duration
	// backgroundTimeout bounds how long a slow consumption keeps being
	// watched server-side after the upload request has returned.
	backgroundTimeout  time.Duration
	backgroundInterval time.Duration
	logger             *Logger
}

func (s *PaperlessInvoiceService) logInfo(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Info("paperless", format, args...)
		return
	}
	log.Printf("[paperless] "+format, args...)
}

func (s *PaperlessInvoiceService) logWarn(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Warn("paperless", format, args...)
		return
	}
	log.Printf("[paperless] "+format, args...)
}

func NewPaperlessInvoiceService(coinRepo *repository.CoinRepository, userRepo *repository.UserRepository, credentials *CredentialEncryptionService, client PaperlessClient, logger *Logger) *PaperlessInvoiceService {
	if credentials == nil {
		credentials = NewDisabledCredentialEncryptionService()
	}
	if client == nil {
		client = NewHTTPPaperlessClient()
	}
	return &PaperlessInvoiceService{
		coinRepo:           coinRepo,
		userRepo:           userRepo,
		credentials:        credentials,
		client:             client,
		pollInterval:       1500 * time.Millisecond,
		pollTimeout:        15 * time.Second,
		backgroundTimeout:  10 * time.Minute,
		backgroundInterval: 3 * time.Second,
		logger:             logger,
	}
}

// WithPolling overrides the wait used after upload (tests use tiny values).
func (s *PaperlessInvoiceService) WithPolling(interval, timeout time.Duration) *PaperlessInvoiceService {
	s.pollInterval = interval
	s.pollTimeout = timeout
	return s
}

// WithBackgroundPolling overrides the server-side follow-up after a pending upload.
func (s *PaperlessInvoiceService) WithBackgroundPolling(interval, timeout time.Duration) *PaperlessInvoiceService {
	s.backgroundInterval = interval
	s.backgroundTimeout = timeout
	return s
}

func (s *PaperlessInvoiceService) findCoin(userID, coinID uint) (*models.Coin, error) {
	coin, err := s.coinRepo.FindByID(coinID, userID)
	if repository.IsRecordNotFound(err) {
		return nil, ErrPaperlessCoinNotFound
	}
	return coin, err
}

func (s *PaperlessInvoiceService) connection(userID uint) (baseURL, token string, err error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", "", err
	}
	baseURL = strings.TrimSpace(user.PaperlessURL)
	if baseURL == "" || strings.TrimSpace(user.PaperlessAPIToken) == "" {
		return "", "", ErrPaperlessNotConfigured
	}
	token, _, err = s.credentials.DecryptStringWithAAD(user.PaperlessAPIToken, UserCredentialAAD(userID, "paperless_api_token"))
	if err != nil {
		return "", "", err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", ErrPaperlessNotConfigured
	}
	return baseURL, token, nil
}

// TestConnection checks the saved paperless-ngx URL and token with one
// authenticated request.
func (s *PaperlessInvoiceService) TestConnection(ctx context.Context, userID uint) error {
	baseURL, token, err := s.connection(userID)
	if err != nil {
		return err
	}
	return s.client.Ping(ctx, baseURL, token)
}

// SplitPaperlessTags splits the comma-separated tag setting into trimmed,
// non-empty, de-duplicated names.
func SplitPaperlessTags(raw string) []string {
	var names []string
	seen := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" || seen[strings.ToLower(name)] {
			continue
		}
		seen[strings.ToLower(name)] = true
		names = append(names, name)
	}
	return names
}

func invoiceDocumentTitle(coin *models.Coin) string {
	title := strings.TrimSpace(coin.Name)
	if invoice := strings.TrimSpace(coin.VendorInvoice); invoice != "" {
		title = strings.TrimSpace(title + " – Invoice " + invoice)
	} else {
		title = strings.TrimSpace(title + " – Invoice")
	}
	if len(title) > 128 {
		title = title[:128]
	}
	return title
}

// UploadInvoice sends the file to paperless-ngx and waits briefly for
// consumption. If paperless is still working when the wait ends, the task id
// is stored on the coin and the server keeps watching it in the background,
// so the coin is linked even if the user navigates away.
func (s *PaperlessInvoiceService) UploadInvoice(ctx context.Context, userID, coinID uint, filename, contentType string, body io.Reader) (*PaperlessInvoiceResult, error) {
	coin, err := s.findCoin(userID, coinID)
	if err != nil {
		return nil, err
	}
	baseURL, token, err := s.connection(userID)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	var tagIDs []int
	if names := SplitPaperlessTags(user.PaperlessTags); len(names) > 0 {
		tagIDs, err = s.client.TagIDs(ctx, baseURL, token, names)
		if err != nil {
			s.logWarn("coin %d: tag lookup failed, upload not sent: %s", coinID, DescribePaperlessError(err))
			return nil, err
		}
	}
	taskID, err := s.client.PostDocument(ctx, baseURL, token, invoiceDocumentTitle(coin), filename, contentType, tagIDs, body)
	if err != nil {
		return nil, err
	}
	if err := s.coinRepo.UpdateField(coin, "paperless_task_id", taskID); err != nil {
		return nil, err
	}
	s.logInfo("coin %d: uploaded %q, waiting for task %s", coinID, filename, taskID)
	// Paperless has the file. From here the server links the coin on its own,
	// whether or not this request, or any page, is still open.
	s.watchInBackground(userID, coinID, taskID)

	pending := &PaperlessInvoiceResult{Status: "pending", TaskID: taskID}
	deadline := time.Now().Add(s.pollTimeout)
	for {
		result, err := s.checkTask(ctx, coin, baseURL, token, taskID)
		switch {
		case errors.Is(err, ErrPaperlessTaskFailed):
			return nil, err
		case err != nil:
			// The upload succeeded; a failed status check (including a dropped
			// request) must not look like a failed upload, or the user re-uploads
			// and creates a duplicate. The background watcher keeps checking.
			s.logWarn("coin %d: task %s: status check during upload failed, background watcher continues: %s", coinID, taskID, DescribePaperlessError(err))
			return pending, nil
		case result.Status == "linked":
			return result, nil
		}
		if time.Now().After(deadline) {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return pending, nil
		case <-time.After(s.pollInterval):
		}
	}
}

func (s *PaperlessInvoiceService) watchInBackground(userID, coinID uint, taskID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), s.backgroundTimeout)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				s.logWarn("coin %d: task %s still pending after %s; stopped watching (opening the coin re-checks it)", coinID, taskID, s.backgroundTimeout)
				return
			case <-time.After(s.backgroundInterval):
			}
			result, err := s.ResolveTask(ctx, userID, coinID, taskID)
			if err != nil {
				s.logWarn("coin %d: task %s: %s", coinID, taskID, DescribePaperlessError(err))
				return
			}
			if result.Status != "pending" {
				return
			}
		}
	}()
}

// ResolveTask re-checks a consumption task and links the coin on success.
// A task that no longer matches the coin's pending task is reported as-is
// without touching the coin.
func (s *PaperlessInvoiceService) ResolveTask(ctx context.Context, userID, coinID uint, taskID string) (*PaperlessInvoiceResult, error) {
	coin, err := s.findCoin(userID, coinID)
	if err != nil {
		return nil, err
	}
	if coin.PaperlessTaskID != taskID {
		if coin.PaperlessDocumentID != nil {
			return &PaperlessInvoiceResult{Status: "linked", TaskID: taskID, DocumentID: coin.PaperlessDocumentID}, nil
		}
		return nil, ErrPaperlessTaskFailed
	}
	baseURL, token, err := s.connection(userID)
	if err != nil {
		return nil, err
	}
	return s.checkTask(ctx, coin, baseURL, token, taskID)
}

func (s *PaperlessInvoiceService) checkTask(ctx context.Context, coin *models.Coin, baseURL, token, taskID string) (*PaperlessInvoiceResult, error) {
	task, err := s.client.GetTask(ctx, baseURL, token, taskID)
	if err != nil {
		return nil, err
	}
	switch task.Status {
	case PaperlessTaskSuccess:
		if task.DocumentID == nil {
			return &PaperlessInvoiceResult{Status: "pending", TaskID: taskID}, nil
		}
		if err := s.coinRepo.UpdateFields(coin, map[string]interface{}{
			"paperless_document_id": *task.DocumentID,
			"paperless_task_id":     "",
		}); err != nil {
			return nil, err
		}
		s.logInfo("coin %d: linked to document %d (task %s)", coin.ID, *task.DocumentID, taskID)
		return &PaperlessInvoiceResult{Status: "linked", TaskID: taskID, DocumentID: task.DocumentID}, nil
	case PaperlessTaskFailure:
		if err := s.coinRepo.UpdateField(coin, "paperless_task_id", ""); err != nil {
			return nil, err
		}
		msg := strings.TrimSpace(task.Result)
		if len(msg) > 300 {
			msg = msg[:300]
		}
		s.logWarn("coin %d: task %s failed in paperless: %s", coin.ID, taskID, msg)
		return nil, fmt.Errorf("%w: %s", ErrPaperlessTaskFailed, msg)
	default:
		return &PaperlessInvoiceResult{Status: "pending", TaskID: taskID}, nil
	}
}

// OpenDocument streams the rendition of the document linked to the coin. Only
// the coin's own linked document can be fetched; there is no way to request
// an arbitrary paperless document id.
func (s *PaperlessInvoiceService) OpenDocument(ctx context.Context, userID, coinID uint, kind PaperlessFileKind) (*PaperlessFile, error) {
	coin, err := s.findCoin(userID, coinID)
	if err != nil {
		return nil, err
	}
	if coin.PaperlessDocumentID == nil {
		return nil, ErrPaperlessNoDocument
	}
	baseURL, token, err := s.connection(userID)
	if err != nil {
		return nil, err
	}
	file, err := s.client.GetDocumentFile(ctx, baseURL, token, *coin.PaperlessDocumentID, kind)
	if err != nil {
		return nil, err
	}
	contentType, ok := displayablePaperlessContentType(file.ContentType)
	if !ok {
		file.Body.Close()
		return nil, ErrPaperlessUnsupportedFile
	}
	limit := int64(MaxPaperlessPreviewBytes)
	if kind == PaperlessFileThumbnail {
		limit = MaxPaperlessThumbnailBytes
	}
	if file.ContentLength > limit {
		file.Body.Close()
		return nil, fmt.Errorf("%w: document is larger than %d MB", ErrPaperlessUnsupportedFile, limit>>20)
	}
	file.ContentType = contentType
	file.Body = limitedReadCloser{Reader: io.LimitReader(file.Body, limit), Closer: file.Body}
	return file, nil
}

// displayablePaperlessContentType allow-lists what may be served from
// Aurearia's origin: PDFs and raster images only (never HTML or SVG).
func displayablePaperlessContentType(raw string) (string, bool) {
	mediaType, _, err := mime.ParseMediaType(raw)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(mediaType) {
	case "application/pdf", "image/png", "image/jpeg", "image/webp", "image/gif", "image/tiff":
		return strings.ToLower(mediaType), true
	}
	return "", false
}

type limitedReadCloser struct {
	io.Reader
	io.Closer
}

// UnlinkDocument clears the coin's link to paperless-ngx. Nothing is deleted
// in paperless-ngx.
func (s *PaperlessInvoiceService) UnlinkDocument(userID, coinID uint) error {
	coin, err := s.findCoin(userID, coinID)
	if err != nil {
		return err
	}
	return s.coinRepo.UpdateFields(coin, map[string]interface{}{
		"paperless_document_id": nil,
		"paperless_task_id":     "",
	})
}
