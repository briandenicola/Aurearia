package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

var (
	ErrIntakeDraftNotFound  = errors.New("intake draft not found")
	ErrIntakeDraftConflict  = errors.New("intake draft is no longer in a confirmable state")
	ErrIntakeConfirmMissing = errors.New("explicit confirmation is required")
)

type IntakeDraftRequest struct {
	Images        []string
	CoinCardImage *string
}

type IntakeConfidenceSummary struct {
	Overall         string   `json:"overall"`
	UncertainFields []string `json:"uncertainFields"`
}

type IntakeEvidence struct {
	Type       string `json:"type"`
	Source     string `json:"source"`
	Field      string `json:"field"`
	Value      string `json:"value"`
	Confidence string `json:"confidence"`
	Notes      string `json:"notes,omitempty"`
}

type IntakeDraftResponse struct {
	DraftID           uint                   `json:"draftId"`
	Status            string                 `json:"status"`
	Coin              map[string]interface{} `json:"coin"`
	ConfidenceSummary IntakeConfidenceSummary `json:"confidenceSummary"`
	Evidence          []IntakeEvidence       `json:"evidence"`
	UnresolvedFields  []string               `json:"unresolvedFields"`
	ExpiresAt         time.Time              `json:"expiresAt"`
}

type IntakeCommitRequest struct {
	DraftID   uint                   `json:"draftId"`
	Confirm   bool                   `json:"confirm"`
	Overrides map[string]interface{} `json:"overrides"`
}

type IntakeCommitResponse struct {
	DraftID uint   `json:"draftId"`
	Status  string `json:"status"`
	CoinID  uint   `json:"coinId"`
}

type IntakeProxyClient interface {
	GenerateIntakeDraft(llmConfig LLMConfig, images []string, coinCardImage *string) (*IntakeProxyDraftResponse, error)
}

type LLMConfigResolver interface {
	ResolveLLMConfig() (LLMConfig, error)
}

type CoinIntakeService struct {
	draftRepo   *repository.CoinIntakeDraftRepository
	coinRepo    *repository.CoinRepository
	proxyClient IntakeProxyClient
	llmResolver LLMConfigResolver
}

func NewCoinIntakeService(
	draftRepo *repository.CoinIntakeDraftRepository,
	coinRepo *repository.CoinRepository,
	proxyClient IntakeProxyClient,
	llmResolver LLMConfigResolver,
) *CoinIntakeService {
	return &CoinIntakeService{
		draftRepo:   draftRepo,
		coinRepo:    coinRepo,
		proxyClient: proxyClient,
		llmResolver: llmResolver,
	}
}

func (s *CoinIntakeService) CreateDraft(userID uint, req IntakeDraftRequest) (*IntakeDraftResponse, error) {
	if len(req.Images) == 0 {
		return nil, fmt.Errorf("at least one image is required")
	}

	llmCfg, err := s.llmResolver.ResolveLLMConfig()
	if err != nil {
		return nil, err
	}

	aiDraft, err := s.proxyClient.GenerateIntakeDraft(llmCfg, req.Images, req.CoinCardImage)
	if err != nil {
		return nil, err
	}

	draftPayloadJSON, err := json.Marshal(aiDraft.Coin)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize draft payload: %w", err)
	}
	confidenceJSON, err := json.Marshal(aiDraft.ConfidenceSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize confidence summary: %w", err)
	}
	evidenceJSON, err := json.Marshal(aiDraft.Evidence)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize evidence: %w", err)
	}
	unresolvedJSON, err := json.Marshal(aiDraft.UnresolvedFields)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize unresolved fields: %w", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	draft := &models.CoinIntakeDraft{
		UserID:            userID,
		DraftPayload:      string(draftPayloadJSON),
		ConfidenceSummary: string(confidenceJSON),
		Evidence:          string(evidenceJSON),
		UnresolvedFields:  string(unresolvedJSON),
		Status:            models.CoinIntakeDraftStatusDrafted,
		ExpiresAt:         expiresAt,
	}
	if err := s.draftRepo.Create(draft); err != nil {
		return nil, err
	}

	return &IntakeDraftResponse{
		DraftID:           draft.ID,
		Status:            draft.Status,
		Coin:              aiDraft.Coin,
		ConfidenceSummary: mapProxyConfidenceSummary(aiDraft.ConfidenceSummary),
		Evidence:          mapProxyEvidence(aiDraft.Evidence),
		UnresolvedFields:  aiDraft.UnresolvedFields,
		ExpiresAt:         draft.ExpiresAt,
	}, nil
}

func mapProxyConfidenceSummary(source IntakeProxyConfidenceSummary) IntakeConfidenceSummary {
	return IntakeConfidenceSummary{
		Overall:         source.Overall,
		UncertainFields: source.UncertainFields,
	}
}

func mapProxyEvidence(source []IntakeProxyEvidence) []IntakeEvidence {
	if len(source) == 0 {
		return nil
	}
	mapped := make([]IntakeEvidence, 0, len(source))
	for _, entry := range source {
		mapped = append(mapped, IntakeEvidence{
			Type:       entry.Type,
			Source:     entry.Source,
			Field:      entry.Field,
			Value:      entry.Value,
			Confidence: entry.Confidence,
			Notes:      entry.Notes,
		})
	}
	return mapped
}

func (s *CoinIntakeService) CommitDraft(userID uint, req IntakeCommitRequest) (*IntakeCommitResponse, error) {
	if !req.Confirm {
		return nil, ErrIntakeConfirmMissing
	}

	draft, err := s.draftRepo.FindByIDForUser(req.DraftID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrIntakeDraftNotFound
		}
		return nil, err
	}
	if draft.Status != models.CoinIntakeDraftStatusDrafted {
		return nil, ErrIntakeDraftConflict
	}
	if time.Now().After(draft.ExpiresAt) {
		if updateErr := s.draftRepo.UpdateStatus(draft.ID, userID, models.CoinIntakeDraftStatusExpired); updateErr != nil {
			return nil, updateErr
		}
		return nil, ErrIntakeDraftConflict
	}

	candidate, err := mergeCandidateAndOverrides(draft.DraftPayload, req.Overrides)
	if err != nil {
		return nil, err
	}
	coin, err := mapToCoin(candidate)
	if err != nil {
		return nil, err
	}
	coin.UserID = userID

	err = s.coinRepo.DB().Transaction(func(tx *gorm.DB) error {
		txDraftRepo := s.draftRepo.WithTx(tx)
		rows, err := txDraftRepo.MarkConfirmedIfDrafted(draft.ID, userID, time.Now())
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrIntakeDraftConflict
		}

		txCoinRepo := s.coinRepo.WithTx(tx)
		if err := txCoinRepo.Create(coin); err != nil {
			return err
		}
		if err := txDraftRepo.AttachConfirmedCoin(draft.ID, userID, coin.ID); err != nil {
			return err
		}
		if err := txCoinRepo.RecordValueSnapshot(userID); err != nil {
			return err
		}
		journalEntry := &models.CoinJournal{
			CoinID: coin.ID,
			UserID: userID,
			Entry:  fmt.Sprintf("[coin_intake] Coin created from AI intake draft #%d", draft.ID),
		}
		if err := txCoinRepo.CreateJournalEntry(journalEntry); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrIntakeDraftConflict) {
			return nil, ErrIntakeDraftConflict
		}
		return nil, err
	}

	return &IntakeCommitResponse{
		DraftID: draft.ID,
		Status:  models.CoinIntakeDraftStatusConfirmed,
		CoinID:  coin.ID,
	}, nil
}

var allowedOverrideFields = map[string]struct{}{
	"name":                 {},
	"issuer":               {},
	"period":               {},
	"date_range":           {},
	"denomination":         {},
	"material":             {},
	"mint":                 {},
	"reference_number":     {},
	"obverse_description":  {},
	"reverse_description":  {},
	"obverse_inscription":  {},
	"reverse_inscription":  {},
	"weight_grams":         {},
	"diameter_mm":          {},
	"rarity":               {},
	"grade":                {},
	"historical_context":   {},
	"provenance":           {},
	"purchase_price":       {},
	"current_value":        {},
	"purchase_date":        {},
	"purchase_location":    {},
	"notes":                {},
	"is_wishlist":          {},
	"is_sold":              {},
	"is_private":           {},
	"sale_price":           {},
	"sale_date":            {},
	"sale_notes":           {},
	"currency":             {},
	"year":                 {},
	"country":              {},
	"composition":          {},
	"coin_condition":       {},
	"category":             {},
	"ruler":                {},
	"era":                  {},
	"estimated_value":      {},
	"market_data":          {},
	"reference_url":        {},
	"dealer":               {},
	"price":                {},
	"listed_date":          {},
	"source":               {},
	"url":                  {},
	"listing_status":       {},
	"listing_checked_at":   {},
	"listing_check_reason": {},
}

func mergeCandidateAndOverrides(draftPayload string, overrides map[string]interface{}) (map[string]interface{}, error) {
	merged := make(map[string]interface{})
	if err := json.Unmarshal([]byte(draftPayload), &merged); err != nil {
		return nil, fmt.Errorf("failed to parse stored draft payload: %w", err)
	}

	for key, value := range overrides {
		snake := toSnakeCase(key)
		if _, ok := allowedOverrideFields[snake]; !ok {
			continue
		}
		merged[snake] = value
	}

	coerceNumericField(merged, "weight_grams")
	coerceNumericField(merged, "diameter_mm")
	coerceNumericField(merged, "purchase_price")
	coerceNumericField(merged, "current_value")
	coerceNumericField(merged, "estimated_value")
	coerceNumericField(merged, "sale_price")
	coerceIntField(merged, "year")
	coerceBoolField(merged, "is_wishlist")
	coerceBoolField(merged, "is_sold")
	coerceBoolField(merged, "is_private")
	coerceDateField(merged, "purchase_date")
	coerceDateField(merged, "sale_date")
	coerceDateField(merged, "listed_date")
	coerceDateField(merged, "listing_checked_at")
	return merged, nil
}

func mapToCoin(payload map[string]interface{}) (*models.Coin, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize candidate payload: %w", err)
	}
	var coin models.Coin
	if err := json.Unmarshal(data, &coin); err != nil {
		return nil, fmt.Errorf("failed to map draft payload to coin model: %w", err)
	}
	coin.Name = strings.TrimSpace(coin.Name)
	if coin.Name == "" {
		coin.Name = "Unidentified Coin"
	}
	return &coin, nil
}

func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func coerceNumericField(m map[string]interface{}, key string) {
	v, ok := m[key]
	if !ok || v == nil {
		return
	}
	switch t := v.(type) {
	case float64:
		return
	case float32:
		m[key] = float64(t)
	case int:
		m[key] = float64(t)
	case int64:
		m[key] = float64(t)
	case json.Number:
		if f, err := t.Float64(); err == nil {
			m[key] = f
		}
	case string:
		if t == "" {
			m[key] = nil
			return
		}
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			m[key] = f
		}
	}
}

func coerceIntField(m map[string]interface{}, key string) {
	v, ok := m[key]
	if !ok || v == nil {
		return
	}
	switch t := v.(type) {
	case int:
		return
	case int64:
		m[key] = int(t)
	case float64:
		m[key] = int(t)
	case json.Number:
		if i, err := t.Int64(); err == nil {
			m[key] = int(i)
		}
	case string:
		if t == "" {
			m[key] = nil
			return
		}
		if i, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			m[key] = i
		}
	}
}

func coerceBoolField(m map[string]interface{}, key string) {
	v, ok := m[key]
	if !ok || v == nil {
		return
	}
	switch t := v.(type) {
	case bool:
		return
	case string:
		if t == "" {
			m[key] = false
			return
		}
		if parsed, err := strconv.ParseBool(strings.TrimSpace(t)); err == nil {
			m[key] = parsed
		}
	}
}

func coerceDateField(m map[string]interface{}, key string) {
	v, ok := m[key]
	if !ok || v == nil {
		return
	}
	s, ok := v.(string)
	if !ok {
		return
	}
	s = strings.TrimSpace(s)
	if s == "" {
		m[key] = nil
		return
	}
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		m[key] = t.Format(time.RFC3339)
	}
}
