package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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
)

var (
	coinIDPattern         = regexp.MustCompile(`(?i)\b(?:coin\s*#|coin\s+id\s*)(\d+)\b`)
	topNPattern           = regexp.MustCompile(`(?i)\btop\s+(\d+)\b`)
	valueUpdatePattern    = regexp.MustCompile(`(?i)\b(?:current\s+value|value)\s+(?:to|as|=)\s*\$?\s*([0-9]+(?:\.[0-9]{1,2})?)`)
	gradeUpdatePattern    = regexp.MustCompile(`(?i)\bgrade\s+(?:to|as|=)\s+([a-z0-9+\-\/ ]{1,40})`)
	notesUpdatePattern    = regexp.MustCompile(`(?i)\bnotes?\s+(?:to|as|=)\s+(.+)$`)
	refTextUpdatePattern  = regexp.MustCompile(`(?i)\breference\s+text\s+(?:to|as|=)\s+(.+)$`)
	refURLUpdatePattern   = regexp.MustCompile(`(?i)\breference\s+url\s+(?:to|as|=)\s+(\S+)$`)
	tagsUpdatePattern     = regexp.MustCompile(`(?i)\btags?\s+(?:to|as|=)\s+(.+)$`)
	disallowedFieldTokens = regexp.MustCompile(`(?i)\b(name|category|material|denomination|mint|purchase\s+price|purchase\s+date|sold\s+price|sold\s+date|obverse|reverse|rarity|weight|diameter|images?)\b`)
)

type CollectionChatContext struct {
	Route        string `json:"route,omitempty"`
	ActiveCoinID *uint  `json:"activeCoinId,omitempty"`
}

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

type CollectionReadResult struct {
	ResultType string                      `json:"resultType"`
	Total      int64                       `json:"total,omitempty"`
	Coins      []CollectionCoinSummary     `json:"coins,omitempty"`
	Aggregate  *CollectionAggregateSummary `json:"aggregate,omitempty"`
}

type CollectionDisambiguation struct {
	Message    string                  `json:"message"`
	Candidates []CollectionCoinSummary `json:"candidates"`
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

type CollectionChatResponse struct {
	Kind           string                     `json:"kind"`
	Message        string                     `json:"message"`
	ReadResult     *CollectionReadResult      `json:"readResult,omitempty"`
	Disambiguation *CollectionDisambiguation  `json:"disambiguation,omitempty"`
	Proposal       *CollectionProposalPreview `json:"proposal,omitempty"`
	ErrorCode      string                     `json:"errorCode,omitempty"`
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

// ShouldHandleCollection returns true when the prompt is a collection read/write intent.
func (s *CollectionToolsService) ShouldHandleCollection(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}

	searchTokens := []string{
		"for sale", "buy", "dealer", "listing", "vcoins", "ma-shops",
		"coin show", "upcoming show", "auction", "near me",
	}
	for _, token := range searchTokens {
		if strings.Contains(lower, token) {
			return false
		}
	}

	collectionTokens := []string{
		"my collection", "my coins", "i own", "holdings", "wishlist", "sold coins",
		"coin #", "coin id", "how many", "top ", "total value", "update ", "change ", "set ",
		"note ", "notes ", "grade ", "reference ",
	}
	for _, token := range collectionTokens {
		if strings.Contains(lower, token) {
			return true
		}
	}

	return false
}

func (s *CollectionToolsService) HandleChat(userID uint, message string, ctx *CollectionChatContext) (*CollectionChatResponse, error) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return &CollectionChatResponse{
			Kind:      "validation_error",
			Message:   "Please enter a message.",
			ErrorCode: "empty_message",
		}, nil
	}

	if isWriteIntent(trimmed) {
		return s.handleWriteIntent(userID, trimmed, ctx)
	}

	return s.handleReadIntent(userID, trimmed, ctx)
}

func (s *CollectionToolsService) CommitProposal(
	userID uint,
	proposalID string,
	proposalToken string,
	confirm bool,
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
			Entry:  buildCollectionChatJournalEntry(changes),
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
		JournalSource: "collection_chat",
		Message:       "Update committed.",
	}, nil
}

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

func (s *CollectionToolsService) handleReadIntent(
	userID uint,
	message string,
	ctx *CollectionChatContext,
) (*CollectionChatResponse, error) {
	if id, ok := extractCoinID(message, ctx); ok {
		coin, err := s.coinRepo.FindByID(id, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &CollectionChatResponse{
					Kind:       "read_result",
					Message:    "I couldn't find that coin in your collection.",
					ReadResult: &CollectionReadResult{ResultType: "coin", Total: 0},
				}, nil
			}
			return nil, err
		}
		msg := fmt.Sprintf(
			"**%s** — %s %s, %s. Current value: %s.",
			coin.Name, coin.Category, coin.Denomination, coin.Ruler, formatCurrency(coin.CurrentValue),
		)
		return &CollectionChatResponse{
			Kind:    "read_result",
			Message: msg,
			ReadResult: &CollectionReadResult{
				ResultType: "coin",
				Total:      1,
				Coins:      []CollectionCoinSummary{toCoinSummary(*coin)},
			},
		}, nil
	}

	lower := strings.ToLower(message)
	if strings.Contains(lower, "top") && strings.Contains(lower, "value") {
		limit := extractTopLimit(message)
		coins, err := s.coinRepo.TopOwnedByCurrentValue(userID, limit)
		if err != nil {
			return nil, err
		}
		if len(coins) == 0 {
			return &CollectionChatResponse{
				Kind:       "read_result",
				Message:    "I couldn't find any valued coins in your active collection.",
				ReadResult: &CollectionReadResult{ResultType: "top_value", Total: 0},
			}, nil
		}
		summaries := make([]CollectionCoinSummary, 0, len(coins))
		lines := make([]string, 0, len(coins))
		for idx, coin := range coins {
			summaries = append(summaries, toCoinSummary(coin))
			lines = append(lines, fmt.Sprintf("%d. %s — %s", idx+1, coin.Name, formatCurrency(coin.CurrentValue)))
		}
		return &CollectionChatResponse{
			Kind:    "read_result",
			Message: "Top coins by current value:\n" + strings.Join(lines, "\n"),
			ReadResult: &CollectionReadResult{
				ResultType: "top_value",
				Total:      int64(len(summaries)),
				Coins:      summaries,
			},
		}, nil
	}

	if strings.Contains(lower, "total value") || strings.Contains(lower, "collection summary") || strings.Contains(lower, "summary") {
		stats, err := s.coinRepo.GetStats(userID)
		if err != nil {
			return nil, err
		}
		msg := fmt.Sprintf(
			"You have %d active coins, %d wishlist coins, and %d sold coins. Current active value is %s.",
			stats.TotalCoins, stats.TotalWishlist, stats.TotalSold, formatCurrencyValue(stats.Values.TotalCurrentValue),
		)
		return &CollectionChatResponse{
			Kind:    "read_result",
			Message: msg,
			ReadResult: &CollectionReadResult{
				ResultType: "aggregate",
				Aggregate: &CollectionAggregateSummary{
					TotalCoins:       stats.TotalCoins,
					TotalWishlist:    stats.TotalWishlist,
					TotalSold:        stats.TotalSold,
					TotalCurrentUSD:  stats.Values.TotalCurrentValue,
					TotalPurchaseUSD: stats.Values.TotalPurchasePrice,
				},
			},
		}, nil
	}

	filters := parseOwnedFilters(message)
	if strings.Contains(lower, "how many") || strings.HasPrefix(lower, "count ") {
		count, err := s.coinRepo.CountOwnedByFilters(userID, filters)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return &CollectionChatResponse{
				Kind:       "read_result",
				Message:    "No coins matched that query in your collection.",
				ReadResult: &CollectionReadResult{ResultType: "count", Total: 0},
			}, nil
		}
		return &CollectionChatResponse{
			Kind:    "read_result",
			Message: fmt.Sprintf("You have %d coin(s) matching that query.", count),
			ReadResult: &CollectionReadResult{
				ResultType: "count",
				Total:      count,
			},
		}, nil
	}

	coins, err := s.coinRepo.ListOwnedByFilters(userID, filters, 5)
	if err != nil {
		return nil, err
	}
	if len(coins) == 0 {
		return &CollectionChatResponse{
			Kind:       "read_result",
			Message:    "No coins matched that query in your collection.",
			ReadResult: &CollectionReadResult{ResultType: "query", Total: 0},
		}, nil
	}

	summaries := make([]CollectionCoinSummary, 0, len(coins))
	lines := make([]string, 0, len(coins))
	for _, coin := range coins {
		summaries = append(summaries, toCoinSummary(coin))
		lines = append(lines, fmt.Sprintf("- #%d %s (%s, %s)", coin.ID, coin.Name, coin.Category, coin.Ruler))
	}

	return &CollectionChatResponse{
		Kind:    "read_result",
		Message: "Here are matching coins from your collection:\n" + strings.Join(lines, "\n"),
		ReadResult: &CollectionReadResult{
			ResultType: "query",
			Total:      int64(len(summaries)),
			Coins:      summaries,
		},
	}, nil
}

func (s *CollectionToolsService) handleWriteIntent(
	userID uint,
	message string,
	ctx *CollectionChatContext,
) (*CollectionChatResponse, error) {
	if disallowedFieldTokens.MatchString(message) {
		return &CollectionChatResponse{
			Kind:      "validation_error",
			Message:   "This chat update can only change: grade, current value, notes, tags, reference text, and reference URL.",
			ErrorCode: "non_allowlisted_field",
		}, nil
	}

	changes := parseAllowedFieldChanges(message)
	if len(changes) == 0 {
		return &CollectionChatResponse{
			Kind:      "validation_error",
			Message:   "I couldn't detect an allowed field update. Try: `set coin #123 grade to VF`.",
			ErrorCode: "missing_allowed_fields",
		}, nil
	}

	targetCoin, disambiguation, err := s.resolveTargetCoin(userID, message, ctx)
	if err != nil {
		return nil, err
	}
	if disambiguation != nil {
		return &CollectionChatResponse{
			Kind:           "disambiguation",
			Message:        disambiguation.Message,
			Disambiguation: disambiguation,
		}, nil
	}
	if targetCoin == nil {
		return &CollectionChatResponse{
			Kind:      "validation_error",
			Message:   "Please specify which coin to update (for example: `update coin #123 notes to ...`).",
			ErrorCode: "missing_target",
		}, nil
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
		CoinID:        targetCoin.ID,
		TokenHash:     sha256Hex(tokenPlain),
		Status:        models.CollectionUpdateProposalPending,
		ChangesJSON:   string(changesJSON),
		ChangedFields: string(fieldsJSON),
		ExpiresAt:     expiresAt,
	}
	if err := s.proposalRepo.CreateProposal(proposal); err != nil {
		return nil, err
	}

	return &CollectionChatResponse{
		Kind:    "proposal",
		Message: fmt.Sprintf("I prepared an update proposal for **%s**. Review and confirm to apply it.", targetCoin.Name),
		Proposal: &CollectionProposalPreview{
			ProposalID:    proposal.ID,
			ProposalToken: tokenPlain,
			CoinID:        targetCoin.ID,
			CoinName:      targetCoin.Name,
			ChangedFields: fields,
			Changes:       changes,
			ExpiresAt:     expiresAt,
		},
	}, nil
}

func (s *CollectionToolsService) resolveTargetCoin(
	userID uint,
	message string,
	ctx *CollectionChatContext,
) (*models.Coin, *CollectionDisambiguation, error) {
	if id, ok := extractCoinID(message, ctx); ok {
		coin, err := s.coinRepo.FindByID(id, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &CollectionDisambiguation{
					Message:    "I couldn't find that coin in your collection.",
					Candidates: []CollectionCoinSummary{},
				}, nil
			}
			return nil, nil, err
		}
		return coin, nil, nil
	}

	targetName := extractCoinNameHint(message)
	if targetName == "" {
		return nil, nil, nil
	}

	candidates, err := s.coinRepo.FindOwnedByNameCandidates(userID, targetName, 5)
	if err != nil {
		return nil, nil, err
	}
	if len(candidates) == 0 {
		return nil, &CollectionDisambiguation{
			Message:    "I couldn't find a matching coin name. Try using a coin ID like `coin #123`.",
			Candidates: []CollectionCoinSummary{},
		}, nil
	}
	if len(candidates) == 1 {
		coin, err := s.coinRepo.FindByID(candidates[0].ID, userID)
		return coin, nil, err
	}

	summaries := make([]CollectionCoinSummary, 0, len(candidates))
	for _, coin := range candidates {
		summaries = append(summaries, toCoinSummary(coin))
	}
	return nil, &CollectionDisambiguation{
		Message:    "I found multiple possible coins. Select one to continue.",
		Candidates: summaries,
	}, nil
}

func extractCoinID(message string, ctx *CollectionChatContext) (uint, bool) {
	if match := coinIDPattern.FindStringSubmatch(message); len(match) == 2 {
		value, err := strconv.ParseUint(match[1], 10, 32)
		if err == nil {
			return uint(value), true
		}
	}
	if ctx != nil && ctx.ActiveCoinID != nil {
		lower := strings.ToLower(message)
		if strings.Contains(lower, "this coin") || strings.Contains(lower, "this one") {
			return *ctx.ActiveCoinID, true
		}
	}
	return 0, false
}

func extractTopLimit(message string) int {
	if match := topNPattern.FindStringSubmatch(message); len(match) == 2 {
		if n, err := strconv.Atoi(match[1]); err == nil && n > 0 {
			if n > 10 {
				return 10
			}
			return n
		}
	}
	return 3
}

func extractCoinNameHint(message string) string {
	lower := strings.ToLower(message)
	fieldMarkers := []string{" notes", " note", " grade", " value", " tags", " tag", " reference"}
	start := strings.Index(lower, "my ")
	if start == -1 {
		return ""
	}
	remaining := message[start+3:]
	remainingLower := lower[start+3:]
	end := len(remaining)
	for _, marker := range fieldMarkers {
		if idx := strings.Index(remainingLower, marker); idx != -1 && idx < end {
			end = idx
		}
	}
	hint := strings.TrimSpace(remaining[:end])
	hint = strings.Trim(hint, `"'.`)
	if len(hint) < 2 {
		return ""
	}
	return hint
}

func parseOwnedFilters(message string) repository.OwnedCoinFilters {
	lower := strings.ToLower(message)
	filters := repository.OwnedCoinFilters{
		Search: strings.TrimSpace(extractCoinNameHint(message)),
	}

	switch {
	case strings.Contains(lower, "roman"):
		filters.Category = string(models.CategoryRoman)
	case strings.Contains(lower, "greek"):
		filters.Category = string(models.CategoryGreek)
	case strings.Contains(lower, "byzantine"):
		filters.Category = string(models.CategoryByzantine)
	case strings.Contains(lower, "modern"):
		filters.Category = string(models.CategoryModern)
	}

	switch {
	case strings.Contains(lower, "gold"):
		filters.Material = string(models.MaterialGold)
	case strings.Contains(lower, "silver"):
		filters.Material = string(models.MaterialSilver)
	case strings.Contains(lower, "bronze"):
		filters.Material = string(models.MaterialBronze)
	case strings.Contains(lower, "copper"):
		filters.Material = string(models.MaterialCopper)
	case strings.Contains(lower, "electrum"):
		filters.Material = string(models.MaterialElectrum)
	}

	switch {
	case strings.Contains(lower, "ancient"):
		filters.Era = string(models.EraAncient)
	case strings.Contains(lower, "medieval"):
		filters.Era = string(models.EraMedieval)
	case strings.Contains(lower, "modern"):
		filters.Era = string(models.EraModern)
	}

	if strings.Contains(lower, "wishlist") {
		v := true
		filters.Wishlist = &v
	}
	if strings.Contains(lower, "sold") {
		v := true
		filters.Sold = &v
	}

	return filters
}

func parseAllowedFieldChanges(message string) map[string]any {
	changes := map[string]any{}

	if match := valueUpdatePattern.FindStringSubmatch(message); len(match) == 2 {
		if v, err := strconv.ParseFloat(match[1], 64); err == nil {
			changes["currentValue"] = v
		}
	}
	if match := gradeUpdatePattern.FindStringSubmatch(message); len(match) == 2 {
		changes["grade"] = strings.TrimSpace(match[1])
	}
	if match := notesUpdatePattern.FindStringSubmatch(message); len(match) == 2 {
		changes["notes"] = strings.TrimSpace(match[1])
	}
	if match := refTextUpdatePattern.FindStringSubmatch(message); len(match) == 2 {
		changes["referenceText"] = strings.TrimSpace(match[1])
	}
	if match := refURLUpdatePattern.FindStringSubmatch(message); len(match) == 2 {
		changes["referenceUrl"] = strings.TrimSpace(match[1])
	}
	if match := tagsUpdatePattern.FindStringSubmatch(message); len(match) == 2 {
		tags := splitTags(match[1])
		if len(tags) > 0 {
			changes["tags"] = tags
		}
	}

	return changes
}

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

func isWriteIntent(message string) bool {
	lower := strings.ToLower(message)
	writeTokens := []string{
		"update", "change", "set", "edit", "mark",
		"notes", "note", "grade", "value", "reference", "tags",
	}
	for _, token := range writeTokens {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func splitTags(raw string) []string {
	raw = strings.ReplaceAll(raw, " and ", ",")
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(strings.Trim(part, `"'`))
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
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

func formatCurrency(value *float64) string {
	if value == nil {
		return "n/a"
	}
	return formatCurrencyValue(*value)
}

func formatCurrencyValue(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func buildCollectionChatJournalEntry(changes map[string]any) string {
	keys := sortedMapKeys(changes)
	return fmt.Sprintf("collection_chat: committed updates (%s)", strings.Join(keys, ", "))
}
