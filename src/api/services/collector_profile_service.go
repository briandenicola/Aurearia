package services

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	CollectorProfileMaxBudget = 100000000
	collectorProfileMaxItems  = 20
)

var collectorCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type CollectorProfileValidationError struct {
	Field string
}

func (e *CollectorProfileValidationError) Error() string {
	return fmt.Sprintf("invalid collector profile field %s", e.Field)
}

type CollectorProfileInput struct {
	BudgetMin           *float64
	BudgetMax           *float64
	Currency            *string
	PreferredPeriods    []string
	PreferredCategories []string
	ExcludedCategories  []string
	PreferredDealers    []string
	CollectingGoals     []string
}

type CollectorProfileResponse struct {
	BudgetMin           *float64   `json:"budgetMin"`
	BudgetMax           *float64   `json:"budgetMax"`
	Currency            *string    `json:"currency"`
	PreferredPeriods    []string   `json:"preferredPeriods"`
	PreferredCategories []string   `json:"preferredCategories"`
	ExcludedCategories  []string   `json:"excludedCategories"`
	PreferredDealers    []string   `json:"preferredDealers"`
	CollectingGoals     []string   `json:"collectingGoals"`
	UpdatedAt           *time.Time `json:"updatedAt"`
	IsDefault           bool       `json:"isDefault"`
}

type CollectorContext struct {
	BudgetMin           *float64  `json:"budget_min,omitempty"`
	BudgetMax           *float64  `json:"budget_max,omitempty"`
	Currency            *string   `json:"currency,omitempty"`
	PreferredPeriods    []string  `json:"preferred_periods"`
	PreferredCategories []string  `json:"preferred_categories"`
	ExcludedCategories  []string  `json:"excluded_categories"`
	PreferredDealers    []string  `json:"preferred_dealers"`
	CollectingGoals     []string  `json:"collecting_goals"`
	CapturedAt          time.Time `json:"captured_at"`
}

type CollectorProfileService struct {
	repo *repository.CollectorProfileRepository
}

func NewCollectorProfileService(repo *repository.CollectorProfileRepository) *CollectorProfileService {
	return &CollectorProfileService{repo: repo}
}

func (s *CollectorProfileService) Get(userID uint) (CollectorProfileResponse, error) {
	profile, err := s.repo.Get(userID)
	if err != nil {
		return CollectorProfileResponse{}, err
	}
	if profile == nil {
		return neutralCollectorProfile(), nil
	}
	return collectorProfileResponse(profile), nil
}

func (s *CollectorProfileService) Replace(userID uint, input CollectorProfileInput) (CollectorProfileResponse, error) {
	profile, err := normalizeCollectorProfile(input)
	if err != nil {
		return CollectorProfileResponse{}, err
	}
	if err := s.repo.Replace(userID, profile); err != nil {
		return CollectorProfileResponse{}, err
	}
	return collectorProfileResponse(profile), nil
}

func (s *CollectorProfileService) Context(userID uint) (CollectorContext, error) {
	response, err := s.Get(userID)
	if err != nil {
		return CollectorContext{}, err
	}
	return CollectorContext{
		BudgetMin: response.BudgetMin, BudgetMax: response.BudgetMax, Currency: response.Currency,
		PreferredPeriods: response.PreferredPeriods, PreferredCategories: response.PreferredCategories,
		ExcludedCategories: response.ExcludedCategories, PreferredDealers: response.PreferredDealers,
		CollectingGoals: response.CollectingGoals, CapturedAt: time.Now().UTC(),
	}, nil
}

func neutralCollectorProfile() CollectorProfileResponse {
	return CollectorProfileResponse{
		PreferredPeriods: []string{}, PreferredCategories: []string{},
		ExcludedCategories: []string{}, PreferredDealers: []string{},
		CollectingGoals: []string{}, IsDefault: true,
	}
}

func collectorProfileResponse(profile *models.CollectorProfile) CollectorProfileResponse {
	updatedAt := profile.UpdatedAt.UTC()
	return CollectorProfileResponse{
		BudgetMin: profile.BudgetMin, BudgetMax: profile.BudgetMax, Currency: profile.Currency,
		PreferredPeriods:    copyStrings(profile.PreferredPeriods),
		PreferredCategories: copyStrings(profile.PreferredCategories),
		ExcludedCategories:  copyStrings(profile.ExcludedCategories),
		PreferredDealers:    copyStrings(profile.PreferredDealers),
		CollectingGoals:     copyStrings(profile.CollectingGoals),
		UpdatedAt:           &updatedAt, IsDefault: false,
	}
}

func normalizeCollectorProfile(input CollectorProfileInput) (*models.CollectorProfile, error) {
	if err := validateBudget("budgetMin", input.BudgetMin); err != nil {
		return nil, err
	}
	if err := validateBudget("budgetMax", input.BudgetMax); err != nil {
		return nil, err
	}
	if input.BudgetMin != nil && input.BudgetMax != nil && *input.BudgetMin > *input.BudgetMax {
		return nil, &CollectorProfileValidationError{Field: "budgetMin"}
	}
	currency, err := normalizeCurrency(input.Currency)
	if err != nil {
		return nil, err
	}
	periods, err := normalizeList("preferredPeriods", input.PreferredPeriods, 100)
	if err != nil {
		return nil, err
	}
	categories, err := normalizeList("preferredCategories", input.PreferredCategories, 100)
	if err != nil {
		return nil, err
	}
	excluded, err := normalizeList("excludedCategories", input.ExcludedCategories, 100)
	if err != nil {
		return nil, err
	}
	dealers, err := normalizeList("preferredDealers", input.PreferredDealers, 200)
	if err != nil {
		return nil, err
	}
	goals, err := normalizeList("collectingGoals", input.CollectingGoals, 500)
	if err != nil {
		return nil, err
	}
	return &models.CollectorProfile{
		BudgetMin: input.BudgetMin, BudgetMax: input.BudgetMax, Currency: currency,
		PreferredPeriods: periods, PreferredCategories: categories,
		ExcludedCategories: excluded, PreferredDealers: dealers, CollectingGoals: goals,
	}, nil
}

func validateBudget(field string, value *float64) error {
	if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 || *value > CollectorProfileMaxBudget) {
		return &CollectorProfileValidationError{Field: field}
	}
	return nil
}

func normalizeCurrency(value *string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	normalized := strings.Join(strings.Fields(*value), " ")
	if !collectorCurrencyPattern.MatchString(normalized) {
		return nil, &CollectorProfileValidationError{Field: "currency"}
	}
	return &normalized, nil
}

func normalizeList(field string, values []string, maxLength int) (models.StringList, error) {
	if len(values) > collectorProfileMaxItems {
		return nil, &CollectorProfileValidationError{Field: field}
	}
	normalized := make(models.StringList, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.Join(strings.Fields(value), " ")
		length := utf8.RuneCountInString(value)
		key := strings.ToLower(value)
		if length < 1 || length > maxLength {
			return nil, &CollectorProfileValidationError{Field: field}
		}
		if _, exists := seen[key]; exists {
			return nil, &CollectorProfileValidationError{Field: field}
		}
		seen[key] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized, nil
}

func copyStrings(values models.StringList) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}
