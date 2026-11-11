package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrPaperlessNotConfigured = errors.New("paperless-ngx URL and API token are required in Settings → Connections")
	ErrPaperlessTaskFailed    = errors.New("paperless-ngx could not consume the document")
	ErrPaperlessCoinNotFound  = errors.New("coin not found")
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
}

func NewPaperlessInvoiceService(coinRepo *repository.CoinRepository, userRepo *repository.UserRepository, credentials *CredentialEncryptionService, client PaperlessClient) *PaperlessInvoiceService {
	if credentials == nil {
		credentials = NewDisabledCredentialEncryptionService()
	}
	if client == nil {
		client = NewHTTPPaperlessClient()
	}
	return &PaperlessInvoiceService{
		coinRepo:     coinRepo,
		userRepo:     userRepo,
		credentials:  credentials,
		client:       client,
		pollInterval: 1500 * time.Millisecond,
		pollTimeout:  15 * time.Second,
	}
}

// WithPolling overrides the wait used after upload (tests use tiny values).
func (s *PaperlessInvoiceService) WithPolling(interval, timeout time.Duration) *PaperlessInvoiceService {
	s.pollInterval = interval
	s.pollTimeout = timeout
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

// UploadInvoice sends the file to paperless-ngx, waits briefly for
// consumption, and links the resulting document to the coin when it finishes.
func (s *PaperlessInvoiceService) UploadInvoice(ctx context.Context, userID, coinID uint, filename, contentType string, body io.Reader) (*PaperlessInvoiceResult, error) {
	coin, err := s.findCoin(userID, coinID)
	if err != nil {
		return nil, err
	}
	baseURL, token, err := s.connection(userID)
	if err != nil {
		return nil, err
	}
	taskID, err := s.client.PostDocument(ctx, baseURL, token, invoiceDocumentTitle(coin), filename, contentType, body)
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(s.pollTimeout)
	for {
		result, err := s.checkTask(ctx, coin, baseURL, token, taskID)
		if err != nil || result.Status == "linked" || time.Now().After(deadline) {
			return result, err
		}
		select {
		case <-ctx.Done():
			return &PaperlessInvoiceResult{Status: "pending", TaskID: taskID}, nil
		case <-time.After(s.pollInterval):
		}
	}
}

// ResolveTask re-checks a pending consumption task and links the coin on success.
func (s *PaperlessInvoiceService) ResolveTask(ctx context.Context, userID, coinID uint, taskID string) (*PaperlessInvoiceResult, error) {
	coin, err := s.findCoin(userID, coinID)
	if err != nil {
		return nil, err
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
		if err := s.coinRepo.UpdateField(coin, "paperless_document_id", *task.DocumentID); err != nil {
			return nil, err
		}
		return &PaperlessInvoiceResult{Status: "linked", TaskID: taskID, DocumentID: task.DocumentID}, nil
	case PaperlessTaskFailure:
		msg := strings.TrimSpace(task.Result)
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return nil, fmt.Errorf("%w: %s", ErrPaperlessTaskFailed, msg)
	default:
		return &PaperlessInvoiceResult{Status: "pending", TaskID: taskID}, nil
	}
}
