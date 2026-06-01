package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

var (
	ErrProposalTokenInvalid    = errors.New("proposal token is invalid")
	ErrProposalStateConflict   = errors.New("proposal is not pending")
	ErrProposalConfirmationReq = errors.New("explicit confirmation is required")
	ErrInvalidFieldChanges     = errors.New("one or more field changes are invalid")
)

type CollectionCoinSummary struct {
	ID           uint     `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category,omitempty"`
	Era          string   `json:"era,omitempty"`
	Ruler        string   `json:"ruler,omitempty"`
	Material     string   `json:"material,omitempty"`
	CurrentValue *float64 `json:"currentValue,omitempty"`
}

type CollectionAggregateSummary struct {
	TotalCoins       int64   `json:"totalCoins"`
	TotalWishlist    int64   `json:"totalWishlist"`
	TotalSold        int64   `json:"totalSold"`
	TotalCurrentUSD  float64 `json:"totalCurrentUsd"`
	TotalPurchaseUSD float64 `json:"totalPurchaseUsd"`
}

type CollectionProposalPreview struct {
	ProposalID    string         `json:"proposalId"`
	ProposalToken string         `json:"proposalToken"`
	CoinID        uint           `json:"coinId"`
	CoinName      string         `json:"coinName"`
	ChangedFields []string       `json:"changedFields"`
	Changes       map[string]any `json:"changes"`
	ExpiresAt     time.Time      `json:"expiresAt"`
}

type CommitCollectionProposalResult struct {
	ProposalID    string   `json:"proposalId"`
	Status        string   `json:"status"`
	CoinID        uint     `json:"coinId"`
	ChangedFields []string `json:"changedFields"`
	JournalSource string   `json:"journalSource"`
	Message       string   `json:"message"`
}

type CancelCollectionProposalResult struct {
	ProposalID string `json:"proposalId"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

// CollectionToolsService handles owner-scoped collection reads and confirm-gated writes.
type CollectionToolsService struct {
	coinRepo     *repository.CoinRepository
	proposalRepo *repository.CollectionUpdateRepository
}

func NewCollectionToolsService(
	coinRepo *repository.CoinRepository,
	proposalRepo *repository.CollectionUpdateRepository,
) *CollectionToolsService {
	return &CollectionToolsService{
		coinRepo:     coinRepo,
		proposalRepo: proposalRepo,
	}
}

// SearchMyCollection searches the user's active collection by filters.
// Returns up to 'limit' matching coins (default 5, max 20).
func (s *CollectionToolsService) SearchMyCollection(userID uint, query string, limit *int) ([]CollectionCoinSummary, error) {
	searchLimit := 5
	if limit != nil {
		if *limit > 20 {
			searchLimit = 20
		} else if *limit > 0 {
			searchLimit = *limit
		}
	}

	filters := repository.OwnedCoinFilters{
		Search: strings.TrimSpace(query),
	}

	lower := strings.ToLower(query)
	// Category
	if strings.Contains(lower, "roman") {
		filters.Category = string(models.CategoryRoman)
	} else if strings.Contains(lower, "greek") {
		filters.Category = string(models.CategoryGreek)
	} else if strings.Contains(lower, "byzantine") {
		filters.Category = string(models.CategoryByzantine)
	} else if strings.Contains(lower, "modern") {
		filters.Category = string(models.CategoryModern)
	}

	// Material
	if strings.Contains(lower, "gold") {
		filters.Material = string(models.MaterialGold)
	} else if strings.Contains(lower, "silver") {
		filters.Material = string(models.MaterialSilver)
	} else if strings.Contains(lower, "bronze") {
		filters.Material = string(models.MaterialBronze)
	} else if strings.Contains(lower, "copper") {
		filters.Material = string(models.MaterialCopper)
	} else if strings.Contains(lower, "electrum") {
		filters.Material = string(models.MaterialElectrum)
	}

	// Era
	if strings.Contains(lower, "ancient") {
		filters.Era = string(models.EraAncient)
	} else if strings.Contains(lower, "medieval") {
		filters.Era = string(models.EraMedieval)
	}

	// Status
	if strings.Contains(lower, "wishlist") {
		v := true
		filters.Wishlist = &v
	}
	if strings.Contains(lower, "sold") {
		v := true
		filters.Sold = &v
	}

	coins, err := s.coinRepo.ListOwnedByFilters(userID, filters, searchLimit)
	if err != nil {
		return nil, err
	}

	summaries := make([]CollectionCoinSummary, 0, len(coins))
	for _, coin := range coins {
		summaries = append(summaries, toCoinSummary(coin))
	}

	return summaries, nil
}

// GetCoin retrieves a single coin by ID from the user's collection.
func (s *CollectionToolsService) GetCoin(userID uint, coinID uint) (*CollectionCoinSummary, error) {
	coin, err := s.coinRepo.FindByID(coinID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCoinNotFound
		}
		return nil, err
	}

	summary := toCoinSummary(*coin)
	return &summary, nil
}

// CollectionSummary returns aggregate statistics for the user's collection.
func (s *CollectionToolsService) CollectionSummary(userID uint) (*CollectionAggregateSummary, error) {
	stats, err := s.coinRepo.GetStats(userID)
	if err != nil {
		return nil, err
	}

	return &CollectionAggregateSummary{
		TotalCoins:       stats.TotalCoins,
		TotalWishlist:    stats.TotalWishlist,
		TotalSold:        stats.TotalSold,
		TotalCurrentUSD:  stats.Values.TotalCurrentValue,
		TotalPurchaseUSD: stats.Values.TotalPurchasePrice,
	}, nil
}

// TopCoinsByValue returns the top coins by current value (default 3, max 10).
func (s *CollectionToolsService) TopCoinsByValue(userID uint, limit *int) ([]CollectionCoinSummary, error) {
	valueLimit := 3
	if limit != nil {
		if *limit > 10 {
			valueLimit = 10
		} else if *limit > 0 {
			valueLimit = *limit
		}
	}

	coins, err := s.coinRepo.TopOwnedByCurrentValue(userID, valueLimit)
	if err != nil {
		return nil, err
	}

	summaries := make([]CollectionCoinSummary, 0, len(coins))
	for _, coin := range coins {
		summaries = append(summaries, toCoinSummary(coin))
	}

	return summaries, nil
}

// ProposeUpdate creates a write proposal for the given coin with allowlisted field changes.
// Returns a proposal with a token that must be used to commit the changes.
func (s *CollectionToolsService) ProposeUpdate(userID uint, coinID uint, changes map[string]any) (*CollectionProposalPreview, error) {
	// Validate that only allowlisted fields are present
	allowlistedFields := map[string]bool{
		"grade":         true,
		"currentValue":  true,
		"notes":         true,
		"tags":          true,
		"referenceText": true,
		"referenceUrl":  true,
	}

	for key := range changes {
		if !allowlistedFields[key] {
			return nil, fmt.Errorf("%w: field '%s' is not allowlisted", ErrInvalidFieldChanges, key)
		}
	}

	if len(changes) == 0 {
		return nil, fmt.Errorf("%w: no changes provided", ErrInvalidFieldChanges)
	}

	coin, err := s.coinRepo.FindByID(coinID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCoinNotFound
		}
		return nil, err
	}

	fields := sortedMapKeys(changes)
	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return nil, err
	}
	fieldsJSON, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}

	proposalID, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	tokenPlain, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(15 * time.Minute)
	proposal := &models.CollectionUpdateProposal{
		ID:            "cup_" + proposalID,
		UserID:        userID,
		CoinID:        coin.ID,
		TokenHash:     sha256Hex(tokenPlain),
		Status:        models.CollectionUpdateProposalPending,
		ChangesJSON:   string(changesJSON),
		ChangedFields: string(fieldsJSON),
		ExpiresAt:     expiresAt,
	}

	if err := s.proposalRepo.CreateProposal(proposal); err != nil {
		return nil, err
	}

	return &CollectionProposalPreview{
		ProposalID:    proposal.ID,
		ProposalToken: tokenPlain,
		CoinID:        coin.ID,
		CoinName:      coin.Name,
		ChangedFields: fields,
		Changes:       changes,
		ExpiresAt:     expiresAt,
	}, nil
}

// CommitUpdate commits a previously created proposal with explicit confirmation.
// Uses "collection_chat" as the journal source (internal).
func (s *CollectionToolsService) CommitUpdate(
	userID uint,
	proposalID string,
	proposalToken string,
	confirm bool,
) (*CommitCollectionProposalResult, error) {
	return s.commitProposalWithSource(userID, proposalID, proposalToken, confirm, "collection_chat", nil)
}

// CommitUpdateExternal commits a proposal from an external tool server with API key metadata.
func (s *CollectionToolsService) CommitUpdateExternal(
	userID uint,
	proposalID string,
	proposalToken string,
	confirm bool,
	apiKeyID uint,
	apiKeyName string,
	apiKeyCapabilities string,
) (*CommitCollectionProposalResult, error) {
	metadata := map[string]any{
		"apiKeyId":           apiKeyID,
		"apiKeyName":         apiKeyName,
		"apiKeyCapabilities": apiKeyCapabilities,
	}
	return s.commitProposalWithSource(userID, proposalID, proposalToken, confirm, "external_tool_server", metadata)
}

// CommitProposal is the underlying implementation for CommitUpdate.
// KEPT for backwards compatibility if frontend still uses public commit/cancel endpoints.
func (s *CollectionToolsService) CommitProposal(
	userID uint,
	proposalID string,
	proposalToken string,
	confirm bool,
) (*CommitCollectionProposalResult, error) {
	return s.commitProposalWithSource(userID, proposalID, proposalToken, confirm, "collection_chat", nil)
}

// commitProposalWithSource is the core commit implementation that accepts a journal source and optional metadata.
func (s *CollectionToolsService) commitProposalWithSource(
	userID uint,
	proposalID string,
	proposalToken string,
	confirm bool,
	journalSource string,
	metadata map[string]any,
) (*CommitCollectionProposalResult, error) {
	if !confirm {
		return nil, ErrProposalConfirmationReq
	}

	proposal, err := s.proposalRepo.FindOwnedProposal(userID, proposalID)
	if err != nil {
		return nil, err
	}

	if proposal.Status != models.CollectionUpdateProposalPending {
		return nil, ErrProposalStateConflict
	}

	now := time.Now().UTC()
	if now.After(proposal.ExpiresAt) {
		_ = s.proposalRepo.MarkExpired(userID, proposalID, now)
		return nil, ErrProposalStateConflict
	}

	if sha256Hex(proposalToken) != proposal.TokenHash {
		return nil, ErrProposalTokenInvalid
	}

	var changes map[string]any
	if err := json.Unmarshal([]byte(proposal.ChangesJSON), &changes); err != nil {
		return nil, err
	}

	var changedFields []string
	if err := json.Unmarshal([]byte(proposal.ChangedFields), &changedFields); err != nil {
		return nil, err
	}

	err = s.coinRepo.DB().Transaction(func(tx *gorm.DB) error {
		txCoinRepo := s.coinRepo.WithTx(tx)
		txProposalRepo := s.proposalRepo.WithTx(tx)

		coin, err := txCoinRepo.FindByID(proposal.CoinID, userID)
		if err != nil {
			return err
		}

		if err := applyAllowedFieldChanges(tx, coin, userID, changes); err != nil {
			return err
		}

		if err := txProposalRepo.MarkCommitted(userID, proposal.ID, now); err != nil {
			return err
		}

		if err := txCoinRepo.CreateJournalEntry(&models.CoinJournal{
			CoinID: coin.ID,
			UserID: userID,
			Entry:  buildJournalEntry(journalSource, changes, metadata),
		}); err != nil {
			return err
		}

		return txCoinRepo.RecordValueSnapshot(userID)
	})
	if err != nil {
		return nil, err
	}

	return &CommitCollectionProposalResult{
		ProposalID:    proposal.ID,
		Status:        string(models.CollectionUpdateProposalCommitted),
		CoinID:        proposal.CoinID,
		ChangedFields: changedFields,
		JournalSource: journalSource,
		Message:       "Update committed.",
	}, nil
}

// CancelProposal cancels a pending proposal. KEPT if frontend still uses it.
func (s *CollectionToolsService) CancelProposal(
	userID uint,
	proposalID string,
) (*CancelCollectionProposalResult, error) {
	proposal, err := s.proposalRepo.FindOwnedProposal(userID, proposalID)
	if err != nil {
		return nil, err
	}
	if proposal.Status != models.CollectionUpdateProposalPending {
		return nil, ErrProposalStateConflict
	}
	if err := s.proposalRepo.MarkCancelled(userID, proposalID, time.Now().UTC()); err != nil {
		return nil, err
	}
	return &CancelCollectionProposalResult{
		ProposalID: proposalID,
		Status:     string(models.CollectionUpdateProposalCancelled),
		Message:    "Proposal cancelled.",
	}, nil
}

// ---- Helper functions ----

func applyAllowedFieldChanges(tx *gorm.DB, coin *models.Coin, userID uint, changes map[string]any) error {
	updates := map[string]interface{}{}
	for key, value := range changes {
		switch key {
		case "grade":
			updates["grade"] = strings.TrimSpace(fmt.Sprintf("%v", value))
		case "currentValue":
			switch v := value.(type) {
			case float64:
				updates["current_value"] = &v
			case string:
				if parsed, err := strconv.ParseFloat(v, 64); err == nil {
					updates["current_value"] = &parsed
				} else {
					return fmt.Errorf("invalid current value")
				}
			default:
				return fmt.Errorf("invalid current value")
			}
		case "notes":
			updates["notes"] = strings.TrimSpace(fmt.Sprintf("%v", value))
		case "referenceText":
			updates["reference_text"] = strings.TrimSpace(fmt.Sprintf("%v", value))
		case "referenceUrl":
			updates["reference_url"] = strings.TrimSpace(fmt.Sprintf("%v", value))
		case "tags":
			// handled after scalar updates
		default:
			return fmt.Errorf("field %s is not allowlisted", key)
		}
	}

	if len(updates) > 0 {
		if err := tx.Model(&models.Coin{}).
			Where("id = ? AND user_id = ?", coin.ID, userID).
			Updates(updates).Error; err != nil {
			return err
		}
	}

	if rawTags, ok := changes["tags"]; ok {
		if err := replaceCoinTags(tx, coin.ID, userID, rawTags); err != nil {
			return err
		}
	}

	return nil
}

func replaceCoinTags(tx *gorm.DB, coinID, userID uint, rawTags any) error {
	tagsAny, ok := rawTags.([]any)
	if !ok {
		if tagsString, ok := rawTags.([]string); ok {
			tagsAny = make([]any, 0, len(tagsString))
			for _, t := range tagsString {
				tagsAny = append(tagsAny, t)
			}
		}
	}
	if !ok && tagsAny == nil {
		return fmt.Errorf("invalid tags format")
	}

	tagNames := make([]string, 0, len(tagsAny))
	seen := map[string]struct{}{}
	for _, item := range tagsAny {
		name := strings.TrimSpace(fmt.Sprintf("%v", item))
		if name == "" {
			continue
		}
		lower := strings.ToLower(name)
		if _, exists := seen[lower]; exists {
			continue
		}
		seen[lower] = struct{}{}
		tagNames = append(tagNames, name)
	}

	if err := tx.Where("coin_id = ?", coinID).Delete(&models.CoinTag{}).Error; err != nil {
		return err
	}

	for _, name := range tagNames {
		var tag models.Tag
		err := tx.Where("user_id = ? AND LOWER(name) = LOWER(?)", userID, name).First(&tag).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tag = models.Tag{
				UserID: userID,
				Name:   name,
			}
			if err := tx.Create(&tag).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if err := tx.Create(&models.CoinTag{CoinID: coinID, TagID: tag.ID}).Error; err != nil {
			return err
		}
	}

	return nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func toCoinSummary(coin models.Coin) CollectionCoinSummary {
	return CollectionCoinSummary{
		ID:           coin.ID,
		Name:         coin.Name,
		Category:     string(coin.Category),
		Era:          string(coin.Era),
		Ruler:        coin.Ruler,
		Material:     string(coin.Material),
		CurrentValue: coin.CurrentValue,
	}
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func buildJournalEntry(source string, changes map[string]any, metadata map[string]any) string {
	keys := sortedMapKeys(changes)
	entry := fmt.Sprintf("%s: committed updates (%s)", source, strings.Join(keys, ", "))
	
	// For external sources, append API key metadata if present
	if source == "external_tool_server" && metadata != nil {
		if apiKeyID, ok := metadata["apiKeyId"].(uint); ok {
			if apiKeyName, ok := metadata["apiKeyName"].(string); ok {
				entry += fmt.Sprintf(" [API key #%d '%s']", apiKeyID, apiKeyName)
			}
		}
	}
	
	return entry
}
