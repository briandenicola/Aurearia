package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrWishlistSearchAlertNotFound       = errors.New("wishlist search alert not found")
	ErrWishlistSearchAlertInvalid        = errors.New("invalid wishlist search alert")
	ErrWishlistSearchAlertNoCriteria     = errors.New("at least one search criterion is required")
	ErrWishlistSearchAlertPriceRange     = errors.New("invalid price range")
	ErrWishlistSearchAlertDateRange      = errors.New("invalid date range")
	ErrWishlistSearchAlertCadence        = errors.New("unsupported cadence")
	ErrWishlistSearchAlertSourceFilter   = errors.New("invalid source filter")
	ErrWishlistSearchAlertStringTooLong  = errors.New("wishlist search alert field too long")
	ErrWishlistSearchAlertDisabled       = errors.New("wishlist search alert is disabled")
	ErrWishlistSearchAlertRunLimited     = errors.New("wishlist search alert run rate limited")
	ErrWishlistSearchAlertAgent          = errors.New("wishlist search alert discovery unavailable")
	ErrWishlistSearchAlertCandidateState = errors.New("invalid candidate state")
	ErrWishlistSearchAlertDuplicate      = errors.New("duplicate wishlist item warning requires acknowledgement")
	ErrWishlistSearchAlertConversion     = errors.New("invalid candidate conversion")
)

const (
	AlertResultCapDefault = 20
	AlertResultCapMax     = 50
)

type WishlistAlertCriteriaInput struct {
	RulerOrIssuer    string   `json:"rulerOrIssuer"`
	CoinType         string   `json:"coinType"`
	DateFrom         *int     `json:"dateFrom"`
	DateTo           *int     `json:"dateTo"`
	Mint             string   `json:"mint"`
	Material         string   `json:"material"`
	GradeOrCondition string   `json:"gradeOrCondition"`
	PriceMin         *float64 `json:"priceMin"`
	PriceMax         *float64 `json:"priceMax"`
	Currency         string   `json:"currency"`
	DealerPreference string   `json:"dealerPreference"`
	SourceFilters    []string `json:"sourceFilters"`
	Keywords         string   `json:"keywords"`
	Notes            string   `json:"notes"`
}

type WishlistSearchAlertInput struct {
	Name     string                     `json:"name"`
	Criteria WishlistAlertCriteriaInput `json:"criteria"`
	Cadence  string                     `json:"cadence"`
	IsActive *bool                      `json:"isActive"`
}

type WishlistSearchAlertService struct {
	repo       *repository.WishlistSearchAlertRepository
	agentProxy *AgentProxy
	settings   *SettingsService
	coinSvc    *CoinService
}

func NewWishlistSearchAlertService(repo *repository.WishlistSearchAlertRepository) *WishlistSearchAlertService {
	return &WishlistSearchAlertService{repo: repo}
}

func (s *WishlistSearchAlertService) WithDiscovery(agentProxy *AgentProxy, settings *SettingsService) *WishlistSearchAlertService {
	s.agentProxy = agentProxy
	s.settings = settings
	return s
}

func (s *WishlistSearchAlertService) WithCoinCreation(coinSvc *CoinService) *WishlistSearchAlertService {
	s.coinSvc = coinSvc
	return s
}

type RunAlertInput struct {
	MaxCandidates int `json:"maxCandidates"`
}

type AlertRunResult struct {
	RunID           uint                    `json:"runId"`
	AlertID         uint                    `json:"alertId"`
	Status          models.AlertRunStatus   `json:"status"`
	StartedAt       time.Time               `json:"startedAt"`
	CompletedAt     *time.Time              `json:"completedAt"`
	ResultCount     int                     `json:"resultCount"`
	NewCount        int                     `json:"newCount"`
	DuplicateCount  int                     `json:"duplicateCount"`
	DismissedCount  int                     `json:"dismissedCount"`
	PartialWarnings models.StringList       `json:"partialWarnings"`
	RateLimitStatus string                  `json:"rateLimitStatus"`
	ErrorMessage    string                  `json:"errorMessage,omitempty"`
	Candidates      []models.AlertCandidate `json:"candidates,omitempty"`
}

type CandidateListResult struct {
	Candidates []models.AlertCandidate `json:"candidates"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	Limit      int                     `json:"limit"`
}

type DismissCandidateInput struct {
	Reason string `json:"reason"`
	Notes  string `json:"notes"`
}

type ConvertCandidateInput struct {
	Coin                        models.Coin `json:"coin"`
	AcknowledgeDuplicateWarning bool        `json:"acknowledgeDuplicateWarning"`
}

type ConvertCandidateResult struct {
	Coin      models.Coin           `json:"coin"`
	Candidate models.AlertCandidate `json:"candidate"`
	Warnings  []string              `json:"warnings"`
}

type AdjustCriteriaInput struct {
	CandidateIDs []uint                     `json:"candidateIds"`
	Criteria     WishlistAlertCriteriaInput `json:"criteria"`
}

func (s *WishlistSearchAlertService) CreateAlert(userID uint, input WishlistSearchAlertInput) (*models.WishlistSearchAlert, error) {
	alert, err := buildAlertFromInput(userID, nil, input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateAlert(alert); err != nil {
		return nil, err
	}
	if input.IsActive != nil && !*input.IsActive && alert.IsActive {
		alert.IsActive = false
		if err := s.repo.UpdateAlert(alert); err != nil {
			return nil, err
		}
	}
	return alert, nil
}

func (s *WishlistSearchAlertService) ListAlerts(userID uint, active *bool, page, limit int) ([]models.WishlistSearchAlert, int64, error) {
	return s.repo.ListAlerts(userID, repository.WishlistSearchAlertFilters{Active: active, Page: page, Limit: limit})
}

func (s *WishlistSearchAlertService) GetAlert(id, userID uint) (*models.WishlistSearchAlert, error) {
	alert, err := s.repo.GetAlert(id, userID)
	if repository.IsRecordNotFound(err) {
		return nil, ErrWishlistSearchAlertNotFound
	}
	return alert, err
}

func (s *WishlistSearchAlertService) UpdateAlert(id, userID uint, input WishlistSearchAlertInput) (*models.WishlistSearchAlert, error) {
	existing, err := s.GetAlert(id, userID)
	if err != nil {
		return nil, err
	}
	updated, err := buildAlertFromInput(userID, existing, input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateAlert(updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *WishlistSearchAlertService) SetAlertActive(id, userID uint, active bool) (*models.WishlistSearchAlert, error) {
	alert, err := s.GetAlert(id, userID)
	if err != nil {
		return nil, err
	}
	alert.IsActive = active
	if err := s.repo.UpdateAlert(alert); err != nil {
		return nil, err
	}
	return alert, nil
}

func (s *WishlistSearchAlertService) DeleteAlert(id, userID uint) error {
	if _, err := s.GetAlert(id, userID); err != nil {
		return err
	}
	return s.repo.DeleteAlert(id, userID)
}

func (s *WishlistSearchAlertService) RunNow(alertID, userID uint, input RunAlertInput) (*AlertRunResult, error) {
	alert, err := s.GetAlert(alertID, userID)
	if err != nil {
		return nil, err
	}
	if !alert.IsActive {
		return nil, ErrWishlistSearchAlertDisabled
	}
	maxCandidates := normalizeMaxCandidates(input.MaxCandidates)
	snapshot, err := s.CriteriaSnapshot(alert, maxCandidates)
	if err != nil {
		return nil, fmt.Errorf("build criteria snapshot: %w", err)
	}
	run := &models.AlertRun{
		AlertID:          alert.ID,
		UserID:           userID,
		TriggerType:      models.AlertRunTriggerManual,
		Status:           models.AlertRunStatusRunning,
		StartedAt:        time.Now(),
		CriteriaSnapshot: snapshot,
		RateLimitStatus:  "ok",
	}
	acquired, err := s.repo.CreateManualRunIfAvailable(run, time.Now().Add(-30*time.Second))
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, ErrWishlistSearchAlertRunLimited
	}
	if s.agentProxy == nil || s.settings == nil {
		return s.failRun(run, "Discovery service is unavailable.", ErrWishlistSearchAlertAgent)
	}
	llmConfig, err := s.settings.ResolveLLMConfig()
	if err != nil {
		return s.failRun(run, "Discovery service is not configured.", ErrWishlistSearchAlertAgent)
	}
	proxyReq := AlertDiscoveryProxyRequest{
		LLM: llmConfig,
		Alert: AlertDiscoveryRequestDetail{
			AlertID: alert.ID,
			CriteriaSnapshot: AlertDiscoveryCriteriaSnapshotProxy{
				Name:             alert.Name,
				RulerOrIssuer:    alert.RulerOrIssuer,
				CoinType:         alert.CoinType,
				DateFrom:         alert.DateFrom,
				DateTo:           alert.DateTo,
				Mint:             alert.Mint,
				Material:         alert.Material,
				GradeOrCondition: alert.GradeOrCondition,
				PriceMin:         alert.PriceMin,
				PriceMax:         alert.PriceMax,
				Currency:         alert.Currency,
				DealerPreference: alert.DealerPreference,
				SourceFilters:    []string(alert.SourceFilters),
				Keywords:         alert.Keywords,
				Notes:            alert.Notes,
			},
			MaxCandidates: maxCandidates,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	resp, err := s.agentProxy.DiscoverAlertCandidates(ctx, proxyReq)
	cancel()
	if err != nil {
		return s.failRun(run, "Discovery service is unavailable.", ErrWishlistSearchAlertAgent)
	}
	result, err := s.ingestCandidates(run, alert, resp)
	if err != nil {
		return s.failRun(run, "Unable to save discovered candidates.", err)
	}
	alert.LastRunAt = &run.StartedAt
	if err := s.repo.UpdateAlert(alert); err != nil {
		return s.failRun(run, "Unable to save alert run metadata.", err)
	}
	return result, nil
}

func (s *WishlistSearchAlertService) failRun(run *models.AlertRun, message string, retErr error) (*AlertRunResult, error) {
	now := time.Now()
	run.Status = models.AlertRunStatusFailed
	run.CompletedAt = &now
	run.DurationMs = now.Sub(run.StartedAt).Milliseconds()
	run.ErrorMessage = message
	run.RateLimitStatus = "failed"
	_ = s.repo.UpdateRun(run)
	return runResult(run, nil), retErr
}

func (s *WishlistSearchAlertService) ingestCandidates(run *models.AlertRun, alert *models.WishlistSearchAlert, resp *AlertDiscoveryProxyResponse) (*AlertRunResult, error) {
	now := time.Now()
	if resp == nil {
		resp = &AlertDiscoveryProxyResponse{}
	}
	candidates := resp.Candidates
	if len(candidates) > AlertResultCapMax {
		candidates = candidates[:AlertResultCapMax]
		resp.Warnings = append(resp.Warnings, "Some candidates were omitted because the run result cap was reached.")
		resp.Partial = true
	}
	persisted := make([]models.AlertCandidate, 0, len(candidates))
	for _, proxyCandidate := range candidates {
		if strings.TrimSpace(proxyCandidate.SourceURL) == "" || strings.TrimSpace(proxyCandidate.Title) == "" || strings.TrimSpace(proxyCandidate.ReasonForMatch) == "" {
			run.DuplicateCount++
			continue
		}
		candidate, provenance := buildCandidateFromProxy(run, alert, proxyCandidate)
		matched := s.matchExistingWishlist(alert.UserID, candidate.SourceURL, candidate.CanonicalSourceURL)
		if matched != nil {
			candidate.MatchingWishlistCoinID = &matched.ID
		}
		existing, err := s.repo.FindCandidateByCanonicalURL(alert.UserID, alert.ID, candidate.CanonicalSourceURL)
		if repository.IsRecordNotFound(err) {
			existing, err = s.repo.FindCandidateByDuplicateKey(alert.UserID, candidate.DuplicateKey)
		}
		if err == nil && existing != nil {
			existing.RunID = run.ID
			existing.LastSeenAt = candidate.LastSeenAt
			existing.SourceName = candidate.SourceName
			existing.Title = candidate.Title
			existing.NormalizedTitle = candidate.NormalizedTitle
			existing.ObservedPrice = candidate.ObservedPrice
			existing.ObservedCurrency = candidate.ObservedCurrency
			existing.ReasonForMatch = candidate.ReasonForMatch
			existing.Fields = candidate.Fields
			existing.ProvenanceStatus = candidate.ProvenanceStatus
			if matched != nil {
				existing.MatchingWishlistCoinID = &matched.ID
			}
			if existing.LifecycleState == models.AlertCandidateStateDismissed {
				run.DismissedCount++
			} else {
				run.DuplicateCount++
			}
			if err := s.repo.UpdateCandidate(existing); err != nil {
				return nil, fmt.Errorf("update alert candidate: %w", err)
			}
			if err := s.repo.ReplaceCandidateProvenance(existing.ID, provenance); err != nil {
				return nil, fmt.Errorf("replace alert candidate provenance: %w", err)
			}
			existing.Provenance = provenance
			persisted = append(persisted, *existing)
			continue
		}
		if err := s.repo.CreateCandidate(candidate, provenance); err != nil {
			return nil, fmt.Errorf("create alert candidate: %w", err)
		}
		candidate.Provenance = provenance
		run.NewCount++
		persisted = append(persisted, *candidate)
	}
	run.ResultCount = len(persisted)
	run.PartialWarnings = models.StringList(sanitizedWarnings(resp.Warnings))
	if resp.Partial {
		run.Status = models.AlertRunStatusPartial
	} else {
		run.Status = models.AlertRunStatusCompleted
	}
	run.CompletedAt = &now
	run.DurationMs = now.Sub(run.StartedAt).Milliseconds()
	if err := s.repo.UpdateRun(run); err != nil {
		return nil, fmt.Errorf("update alert run: %w", err)
	}
	return runResult(run, persisted), nil
}

func (s *WishlistSearchAlertService) ListRuns(alertID, userID uint, page, limit int) ([]models.AlertRun, int64, error) {
	if _, err := s.GetAlert(alertID, userID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListRuns(alertID, userID, page, limit)
}

func (s *WishlistSearchAlertService) GetRun(alertID, runID, userID uint) (*models.AlertRun, error) {
	if _, err := s.GetAlert(alertID, userID); err != nil {
		return nil, err
	}
	run, err := s.repo.GetRun(alertID, runID, userID)
	if repository.IsRecordNotFound(err) {
		return nil, ErrWishlistSearchAlertNotFound
	}
	return run, err
}

func (s *WishlistSearchAlertService) ListCandidates(alertID, userID uint, state, provenanceStatus string, page, limit int) (*CandidateListResult, error) {
	if _, err := s.GetAlert(alertID, userID); err != nil {
		return nil, err
	}
	candidates, total, err := s.repo.ListCandidates(alertID, userID, repository.AlertCandidateFilters{
		State: state, ProvenanceStatus: provenanceStatus, Page: page, Limit: limit,
	})
	if err != nil {
		return nil, err
	}
	page, limit = normalizePageLimit(page, limit)
	return &CandidateListResult{Candidates: candidates, Total: total, Page: page, Limit: limit}, nil
}

func (s *WishlistSearchAlertService) DismissCandidate(alertID, candidateID, userID uint, input DismissCandidateInput) (*models.AlertCandidate, error) {
	candidate, err := s.getCandidate(alertID, candidateID, userID)
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(input.Reason)
	if reason != "" && !validDismissalReason(reason) {
		return nil, ErrWishlistSearchAlertInvalid
	}
	candidate.LifecycleState = models.AlertCandidateStateDismissed
	candidate.DismissalReason = reason
	action := reviewAction(candidate.ID, userID, models.CandidateReviewDismissed, reason, strings.TrimSpace(input.Notes))
	if err := s.repo.UpdateCandidateWithReviewAction(candidate, action); err != nil {
		return nil, err
	}
	return candidate, nil
}

func (s *WishlistSearchAlertService) RestoreCandidate(alertID, candidateID, userID uint) (*models.AlertCandidate, error) {
	candidate, err := s.getCandidate(alertID, candidateID, userID)
	if err != nil {
		return nil, err
	}
	if candidate.LifecycleState == models.AlertCandidateStateConverted {
		return nil, ErrWishlistSearchAlertCandidateState
	}
	candidate.LifecycleState = models.AlertCandidateStateActive
	candidate.DismissalReason = ""
	action := reviewAction(candidate.ID, userID, models.CandidateReviewRestored, "", "")
	if err := s.repo.UpdateCandidateWithReviewAction(candidate, action); err != nil {
		return nil, err
	}
	return candidate, nil
}

func (s *WishlistSearchAlertService) ConvertCandidate(alertID, candidateID, userID uint, input ConvertCandidateInput) (*ConvertCandidateResult, error) {
	candidate, err := s.getCandidate(alertID, candidateID, userID)
	if err != nil {
		return nil, err
	}
	if candidate.LifecycleState == models.AlertCandidateStateConverted || candidate.ConvertedCoinID != nil {
		return nil, ErrWishlistSearchAlertCandidateState
	}
	coin := input.Coin
	coin.UserID = userID
	coin.IsWishlist = true
	coin.SourceAlertCandidateID = &candidate.ID
	if strings.TrimSpace(coin.Name) == "" {
		coin.Name = candidate.Title
	}
	if strings.TrimSpace(coin.ReferenceURL) == "" {
		coin.ReferenceURL = candidate.SourceURL
	}
	if coin.PurchasePrice == nil && candidate.ObservedPrice != nil {
		coin.PurchasePrice = candidate.ObservedPrice
	}
	if coin.CurrentValue == nil && candidate.ObservedPrice != nil {
		coin.CurrentValue = candidate.ObservedPrice
	}
	if strings.TrimSpace(coin.PurchaseLocation) == "" {
		coin.PurchaseLocation = candidate.SourceName
	}
	if strings.TrimSpace(coin.ReferenceText) == "" {
		coin.ReferenceText = fmt.Sprintf("Source-backed candidate from wishlist search alert #%d", alertID)
	}
	if strings.TrimSpace(coin.Notes) == "" {
		coin.Notes = fmt.Sprintf("Converted from alert candidate #%d", candidate.ID)
	}
	if err := validateConversionCoin(coin); err != nil {
		return nil, err
	}
	warnings := s.duplicateWarnings(userID, candidate, coin.ReferenceURL)
	if len(warnings) > 0 && !input.AcknowledgeDuplicateWarning {
		return &ConvertCandidateResult{Candidate: *candidate, Warnings: warnings}, ErrWishlistSearchAlertDuplicate
	}
	action := reviewAction(candidate.ID, userID, models.CandidateReviewConverted, "", "")
	if len(warnings) > 0 {
		action.Metadata = `{"duplicateWarningAcknowledged":true}`
	}
	if s.coinSvc == nil {
		return nil, ErrWishlistSearchAlertConversion
	}
	if err := s.coinSvc.prepareCoinForCreate(&coin); err != nil {
		return nil, err
	}
	if err := s.repo.ConvertCandidateToWishlist(candidate, s.coinSvc.PreparedCoinCreator(&coin), action); err != nil {
		return nil, err
	}
	return &ConvertCandidateResult{Coin: coin, Candidate: *candidate, Warnings: warnings}, nil
}

func (s *WishlistSearchAlertService) AdjustCriteria(alertID, userID uint, input AdjustCriteriaInput) (*models.WishlistSearchAlert, error) {
	alert, err := s.GetAlert(alertID, userID)
	if err != nil {
		return nil, err
	}
	update := WishlistSearchAlertInput{
		Name:     alert.Name,
		Criteria: input.Criteria,
		Cadence:  string(alert.Cadence),
		IsActive: &alert.IsActive,
	}
	updated, err := s.UpdateAlert(alertID, userID, update)
	if err != nil {
		return nil, err
	}
	for _, candidateID := range input.CandidateIDs {
		candidate, err := s.getCandidate(alertID, candidateID, userID)
		if err == nil {
			_ = s.repo.CreateReviewAction(reviewAction(candidate.ID, userID, models.CandidateReviewCriteriaAdjusted, "", ""))
		}
	}
	return updated, nil
}

func (s *WishlistSearchAlertService) CriteriaSnapshot(alert *models.WishlistSearchAlert, maxCandidates int) (string, error) {
	if maxCandidates <= 0 {
		maxCandidates = AlertResultCapDefault
	}
	if maxCandidates > AlertResultCapMax {
		maxCandidates = AlertResultCapMax
	}
	snapshot := map[string]interface{}{
		"alertId":          alert.ID,
		"name":             alert.Name,
		"rulerOrIssuer":    alert.RulerOrIssuer,
		"coinType":         alert.CoinType,
		"dateFrom":         alert.DateFrom,
		"dateTo":           alert.DateTo,
		"mint":             alert.Mint,
		"material":         alert.Material,
		"gradeOrCondition": alert.GradeOrCondition,
		"priceMin":         alert.PriceMin,
		"priceMax":         alert.PriceMax,
		"currency":         alert.Currency,
		"dealerPreference": alert.DealerPreference,
		"sourceFilters":    []string(alert.SourceFilters),
		"keywords":         alert.Keywords,
		"notes":            alert.Notes,
		"cadence":          alert.Cadence,
		"isActive":         alert.IsActive,
		"maxCandidates":    maxCandidates,
	}
	b, err := json.Marshal(snapshot)
	return string(b), err
}

func buildAlertFromInput(userID uint, existing *models.WishlistSearchAlert, input WishlistSearchAlertInput) (*models.WishlistSearchAlert, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrWishlistSearchAlertInvalid)
	}
	if len(name) > 200 {
		return nil, ErrWishlistSearchAlertStringTooLong
	}
	cadence := models.WishlistAlertCadence(strings.TrimSpace(input.Cadence))
	if cadence == "" {
		cadence = models.WishlistAlertCadenceManual
	}
	if !validCadence(cadence) {
		return nil, ErrWishlistSearchAlertCadence
	}
	criteria := trimCriteria(input.Criteria)
	if err := validateCriteria(criteria); err != nil {
		return nil, err
	}
	filters, err := NormalizeSourceFilters(criteria.SourceFilters)
	if err != nil {
		return nil, err
	}
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}
	alert := existing
	if alert == nil {
		alert = &models.WishlistSearchAlert{UserID: userID}
	}
	alert.UserID = userID
	alert.Name = name
	alert.RulerOrIssuer = criteria.RulerOrIssuer
	alert.CoinType = criteria.CoinType
	alert.DateFrom = criteria.DateFrom
	alert.DateTo = criteria.DateTo
	alert.Mint = criteria.Mint
	alert.Material = criteria.Material
	alert.GradeOrCondition = criteria.GradeOrCondition
	alert.PriceMin = criteria.PriceMin
	alert.PriceMax = criteria.PriceMax
	alert.Currency = criteria.Currency
	alert.DealerPreference = criteria.DealerPreference
	alert.SourceFilters = models.StringList(filters)
	alert.Keywords = criteria.Keywords
	alert.Notes = criteria.Notes
	alert.Cadence = cadence
	alert.IsActive = active
	return alert, nil
}

func trimCriteria(c WishlistAlertCriteriaInput) WishlistAlertCriteriaInput {
	c.RulerOrIssuer = strings.TrimSpace(c.RulerOrIssuer)
	c.CoinType = strings.TrimSpace(c.CoinType)
	c.Mint = strings.TrimSpace(c.Mint)
	c.Material = strings.TrimSpace(c.Material)
	c.GradeOrCondition = strings.TrimSpace(c.GradeOrCondition)
	c.Currency = strings.ToUpper(strings.TrimSpace(c.Currency))
	if c.Currency == "" {
		c.Currency = "USD"
	}
	c.DealerPreference = strings.TrimSpace(c.DealerPreference)
	c.Keywords = strings.TrimSpace(c.Keywords)
	c.Notes = strings.TrimSpace(c.Notes)
	return c
}

func validateCriteria(c WishlistAlertCriteriaInput) error {
	if c.PriceMin != nil && *c.PriceMin < 0 || c.PriceMax != nil && *c.PriceMax < 0 {
		return ErrWishlistSearchAlertPriceRange
	}
	if c.PriceMin != nil && c.PriceMax != nil && *c.PriceMin > *c.PriceMax {
		return ErrWishlistSearchAlertPriceRange
	}
	if c.DateFrom != nil && c.DateTo != nil && *c.DateFrom > *c.DateTo {
		return ErrWishlistSearchAlertDateRange
	}
	if len(c.RulerOrIssuer) > 200 || len(c.CoinType) > 200 || len(c.Mint) > 200 || len(c.Material) > 100 ||
		len(c.GradeOrCondition) > 200 || len(c.DealerPreference) > 500 || len(c.Keywords) > 500 || len(c.Notes) > 5000 {
		return ErrWishlistSearchAlertStringTooLong
	}
	if len(c.Currency) != 3 {
		return fmt.Errorf("%w: currency must be a three-letter code", ErrWishlistSearchAlertInvalid)
	}
	if c.RulerOrIssuer == "" && c.CoinType == "" && c.DateFrom == nil && c.DateTo == nil && c.Mint == "" &&
		c.Material == "" && c.GradeOrCondition == "" && c.PriceMin == nil && c.PriceMax == nil &&
		c.DealerPreference == "" && len(c.SourceFilters) == 0 && c.Keywords == "" {
		return ErrWishlistSearchAlertNoCriteria
	}
	return nil
}

func validCadence(c models.WishlistAlertCadence) bool {
	switch c {
	case models.WishlistAlertCadenceManual, models.WishlistAlertCadenceDaily, models.WishlistAlertCadenceWeekly, models.WishlistAlertCadenceMonthly:
		return true
	default:
		return false
	}
}

var hostnameLabel = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func NormalizeSourceFilters(filters []string) ([]string, error) {
	out := make([]string, 0, len(filters))
	seen := map[string]struct{}{}
	for _, raw := range filters {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			continue
		}
		if len(value) > 253 {
			return nil, ErrWishlistSearchAlertSourceFilter
		}
		if strings.Contains(value, "://") {
			parsed, err := url.Parse(value)
			if err != nil || parsed.Hostname() == "" {
				return nil, ErrWishlistSearchAlertSourceFilter
			}
			value = parsed.Hostname()
		}
		value = strings.TrimSuffix(value, ".")
		if ip := net.ParseIP(value); ip != nil || !validHostname(value) {
			return nil, ErrWishlistSearchAlertSourceFilter
		}
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	return out, nil
}

func validHostname(host string) bool {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if !hostnameLabel.MatchString(part) {
			return false
		}
	}
	return true
}

func CanonicalSourceURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimSpace(raw)
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	q := parsed.Query()
	for key := range q {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" {
			q.Del(key)
		}
	}
	parsed.RawQuery = q.Encode()
	parsed.Fragment = ""
	return parsed.String()
}

func NormalizeCandidateTitle(title string) string {
	var b strings.Builder
	lastSpace := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteRune(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func DuplicateKey(alertID uint, canonicalURL, normalizedTitle string, observedPrice *float64, currency string) string {
	price := ""
	if observedPrice != nil {
		price = fmt.Sprintf("%.2f %s", *observedPrice, strings.ToUpper(currency))
	}
	source := fmt.Sprintf("%d|%s|%s|%s", alertID, canonicalURL, normalizedTitle, price)
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

func normalizeMaxCandidates(value int) int {
	if value <= 0 {
		return AlertResultCapDefault
	}
	if value > AlertResultCapMax {
		return AlertResultCapMax
	}
	return value
}

func runResult(run *models.AlertRun, candidates []models.AlertCandidate) *AlertRunResult {
	return &AlertRunResult{
		RunID:           run.ID,
		AlertID:         run.AlertID,
		Status:          run.Status,
		StartedAt:       run.StartedAt,
		CompletedAt:     run.CompletedAt,
		ResultCount:     run.ResultCount,
		NewCount:        run.NewCount,
		DuplicateCount:  run.DuplicateCount,
		DismissedCount:  run.DismissedCount,
		PartialWarnings: run.PartialWarnings,
		RateLimitStatus: run.RateLimitStatus,
		ErrorMessage:    run.ErrorMessage,
		Candidates:      candidates,
	}
}

func buildCandidateFromProxy(run *models.AlertRun, alert *models.WishlistSearchAlert, proxy AlertCandidateProxy) (*models.AlertCandidate, []models.CandidateProvenance) {
	lastSeen := parseAgentTime(proxy.LastSeenAt, time.Now())
	canonicalURL := CanonicalSourceURL(proxy.SourceURL)
	normalizedTitle := NormalizeCandidateTitle(proxy.Title)
	currency := strings.ToUpper(strings.TrimSpace(proxy.ObservedCurrency))
	if currency == "" {
		currency = alert.Currency
	}
	state := models.AlertCandidateStateActive
	status := models.CandidateProvenanceStatus(proxy.ProvenanceStatus)
	if status != models.CandidateProvenanceVerified {
		state = models.AlertCandidateStateNeedsReview
	}
	candidate := &models.AlertCandidate{
		UserID:             alert.UserID,
		AlertID:            alert.ID,
		RunID:              run.ID,
		SourceURL:          strings.TrimSpace(proxy.SourceURL),
		CanonicalSourceURL: canonicalURL,
		SourceName:         strings.TrimSpace(proxy.SourceName),
		Title:              strings.TrimSpace(proxy.Title),
		NormalizedTitle:    normalizedTitle,
		ObservedPrice:      proxy.ObservedPrice,
		ObservedCurrency:   currency,
		ReasonForMatch:     strings.TrimSpace(proxy.ReasonForMatch),
		Fields:             normalizedCandidateFields(proxy.Fields),
		LastSeenAt:         lastSeen,
		FirstSeenAt:        lastSeen,
		ProvenanceStatus:   status,
		LifecycleState:     state,
		DuplicateKey:       DuplicateKey(alert.ID, canonicalURL, normalizedTitle, proxy.ObservedPrice, currency),
	}
	provenance := make([]models.CandidateProvenance, 0, len(proxy.Provenance)+3)
	for _, item := range proxy.Provenance {
		observedAt := parseAgentTime(item.ObservedAt, lastSeen)
		provenance = append(provenance, models.CandidateProvenance{
			Field:             strings.TrimSpace(item.Field),
			Value:             strings.TrimSpace(item.Value),
			SourceURL:         strings.TrimSpace(item.SourceURL),
			ObservedAt:        observedAt,
			Confidence:        strings.TrimSpace(item.Confidence),
			VerificationState: models.CandidateProvenanceStatus(item.VerificationState),
			Notes:             strings.TrimSpace(item.Notes),
		})
	}
	if len(provenance) == 0 {
		provenance = append(provenance, models.CandidateProvenance{
			Field:             "source_url",
			Value:             candidate.SourceURL,
			SourceURL:         candidate.SourceURL,
			ObservedAt:        lastSeen,
			Confidence:        "low",
			VerificationState: candidate.ProvenanceStatus,
			Notes:             "Agent returned candidate without detailed provenance.",
		})
	}
	return candidate, provenance
}

func normalizedCandidateFields(fields map[string]string) models.StringMap {
	result := models.StringMap{}
	for key, value := range fields {
		normalizedKey := strings.TrimSpace(key)
		normalizedValue := strings.TrimSpace(value)
		if normalizedKey == "" || normalizedValue == "" {
			continue
		}
		result[normalizedKey] = normalizedValue
	}
	return result
}

func parseAgentTime(value string, fallback time.Time) time.Time {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	return fallback
}

func sanitizedWarnings(warnings []string) []string {
	out := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		clean := strings.TrimSpace(warning)
		if clean == "" {
			continue
		}
		if len(clean) > 300 {
			clean = clean[:300]
		}
		out = append(out, clean)
	}
	return out
}

func (s *WishlistSearchAlertService) matchExistingWishlist(userID uint, sourceURL, canonicalURL string) *models.Coin {
	for _, candidateURL := range []string{sourceURL, canonicalURL} {
		if strings.TrimSpace(candidateURL) == "" {
			continue
		}
		coin, err := s.repo.FindWishlistByReferenceURL(userID, candidateURL)
		if err == nil {
			return coin
		}
	}
	return nil
}

func (s *WishlistSearchAlertService) getCandidate(alertID, candidateID, userID uint) (*models.AlertCandidate, error) {
	if _, err := s.GetAlert(alertID, userID); err != nil {
		return nil, err
	}
	candidate, err := s.repo.GetCandidate(alertID, candidateID, userID)
	if repository.IsRecordNotFound(err) {
		return nil, ErrWishlistSearchAlertNotFound
	}
	return candidate, err
}

func validDismissalReason(reason string) bool {
	switch reason {
	case "irrelevant", "duplicate", "price_too_high", "poor_provenance", "other":
		return true
	default:
		return false
	}
}

func reviewAction(candidateID, userID uint, actionType models.CandidateReviewActionType, reason, metadata string) *models.CandidateReviewAction {
	return &models.CandidateReviewAction{
		CandidateID: candidateID,
		UserID:      userID,
		Action:      actionType,
		Reason:      strings.TrimSpace(reason),
		Metadata:    strings.TrimSpace(metadata),
	}
}

func validateConversionCoin(coin models.Coin) error {
	if strings.TrimSpace(coin.Name) == "" || strings.TrimSpace(coin.ReferenceURL) == "" {
		return ErrWishlistSearchAlertConversion
	}
	if coin.Category == "" {
		coin.Category = models.CategoryOther
	}
	switch coin.Category {
	case models.CategoryRoman, models.CategoryGreek, models.CategoryByzantine, models.CategoryModern, models.CategoryOther:
	default:
		return ErrWishlistSearchAlertConversion
	}
	if coin.Era != "" {
		switch coin.Era {
		case models.EraAncient, models.EraMedieval, models.EraModern:
		default:
			return ErrWishlistSearchAlertConversion
		}
	}
	return nil
}

func (s *WishlistSearchAlertService) duplicateWarnings(userID uint, candidate *models.AlertCandidate, referenceURL string) []string {
	warnings := []string{}
	if coin, err := s.repo.FindWishlistBySourceAlertCandidateID(userID, candidate.ID); err == nil && coin.ID != 0 {
		warnings = append(warnings, "This candidate already has a converted wishlist item.")
	}
	for _, candidateURL := range []string{referenceURL, candidate.SourceURL, candidate.CanonicalSourceURL} {
		if strings.TrimSpace(candidateURL) == "" {
			continue
		}
		if coin, err := s.repo.FindWishlistByReferenceURL(userID, candidateURL); err == nil && coin.ID != 0 {
			warnings = append(warnings, "A wishlist item already uses this source URL.")
			break
		}
	}
	return warnings
}

func normalizePageLimit(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
