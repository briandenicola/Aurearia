package services

import (
	"sort"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// ImperialFigureSlot pairs a curated RomanImperialFigure with the user's
// matching coin, if any. A nil Coin means the figure is unowned.
type ImperialFigureSlot struct {
	Figure            models.RomanImperialFigure `json:"figure"`
	Coin              *models.Coin               `json:"coin"`
	Coins             []models.Coin              `json:"coins"`
	HighlightedCoinID *uint                      `json:"highlightedCoinId"`
}

// DynastyProgress groups a category's figures by dynasty/era with completion counts.
type DynastyProgress struct {
	Dynasty string               `json:"dynasty"`
	Owned   int                  `json:"owned"`
	Total   int                  `json:"total"`
	Figures []ImperialFigureSlot `json:"figures"`
}

// CategoryProgress is the completion result for one or more combined roles
// (e.g. the "other figures" category combines role:caesar and role:other).
type CategoryProgress struct {
	Roles      []models.ImperialFigureRole `json:"roles"`
	Owned      int                         `json:"owned"`
	Total      int                         `json:"total"`
	Percentage float64                     `json:"percentage"`
	Dynasties  []DynastyProgress           `json:"dynasties"`
}

// EmperorTrackerResult is the full F028 /stats/emperors payload: the primary
// (always-on) emperor goal plus its V1 suggestions, and any of the three
// optional categories the user has enabled.
type EmperorTrackerResult struct {
	Emperor     CategoryProgress             `json:"emperor"`
	Suggestions []models.RomanImperialFigure `json:"suggestions"`
	Usurpers    *CategoryProgress            `json:"usurpers,omitempty"`
	Empresses   *CategoryProgress            `json:"empresses,omitempty"`
	Other       *CategoryProgress            `json:"other,omitempty"`
}

// rarityTierOrder ranks RarityTier from most to least attainable, used to
// sort the V1 "what to pursue next" suggestions list.
var rarityTierOrder = map[models.RarityTier]int{
	models.RarityTierCommon:   0,
	models.RarityTierScarce:   1,
	models.RarityTierRare:     2,
	models.RarityTierVeryRare: 3,
}

// EmperorTrackerService computes per-user collection-completion progress
// against the curated RomanImperialFigure dataset (see F028). Progress is
// computed live per-request, the same way AuctionLotService.Recommend/
// MarketSignal compute their results, rather than maintaining a stored table.
type EmperorTrackerService struct {
	figureRepo    *repository.RomanImperialFigureRepository
	coinRepo      *repository.CoinRepository
	highlightRepo *repository.RomanImperialFigureHighlightRepository
}

// NewEmperorTrackerService creates a new EmperorTrackerService.
func NewEmperorTrackerService(figureRepo *repository.RomanImperialFigureRepository, coinRepo *repository.CoinRepository, highlightRepo *repository.RomanImperialFigureHighlightRepository) *EmperorTrackerService {
	return &EmperorTrackerService{figureRepo: figureRepo, coinRepo: coinRepo, highlightRepo: highlightRepo}
}

// Progress computes completion for the given role(s), combining them into a
// single category when more than one role is passed.
func (s *EmperorTrackerService) Progress(userID uint, roles ...models.ImperialFigureRole) (CategoryProgress, error) {
	figures, err := s.figureRepo.ListByRoles(roles...)
	if err != nil {
		return CategoryProgress{}, err
	}
	owned, err := s.ownedCoinsByFigureID(userID)
	if err != nil {
		return CategoryProgress{}, err
	}
	highlights, err := s.highlightCoinIDsByFigureID(userID)
	if err != nil {
		return CategoryProgress{}, err
	}
	progress := buildCategoryProgress(figures, owned, highlights)
	progress.Roles = roles
	return progress, nil
}

// Suggestions returns the user's missing role:emperor figures sorted with
// the most attainable (by RarityTier) first, tie-broken by chronological
// SortOrder — a static sort over already-loaded data, no agent/network call
// (F028 V1 scope; a market-data-assisted V2 is a documented future stretch).
func (s *EmperorTrackerService) Suggestions(userID uint, limit int) ([]models.RomanImperialFigure, error) {
	progress, err := s.Progress(userID, models.ImperialFigureRoleEmperor)
	if err != nil {
		return nil, err
	}

	missing := make([]models.RomanImperialFigure, 0)
	for _, dynasty := range progress.Dynasties {
		for _, slot := range dynasty.Figures {
			if slot.Coin == nil {
				missing = append(missing, slot.Figure)
			}
		}
	}
	sort.SliceStable(missing, func(i, j int) bool {
		ri, rj := rarityTierOrder[missing[i].RarityTier], rarityTierOrder[missing[j].RarityTier]
		if ri != rj {
			return ri < rj
		}
		return missing[i].SortOrder < missing[j].SortOrder
	})
	if limit > 0 && len(missing) > limit {
		missing = missing[:limit]
	}
	return missing, nil
}

// FullProgress computes the primary emperor progress + V1 suggestions, plus
// progress for any optional categories the user has enabled.
func (s *EmperorTrackerService) FullProgress(userID uint, includeUsurpers, includeEmpresses, includeOther bool, suggestionLimit int) (EmperorTrackerResult, error) {
	emperor, err := s.Progress(userID, models.ImperialFigureRoleEmperor)
	if err != nil {
		return EmperorTrackerResult{}, err
	}
	suggestions, err := s.Suggestions(userID, suggestionLimit)
	if err != nil {
		return EmperorTrackerResult{}, err
	}
	result := EmperorTrackerResult{Emperor: emperor, Suggestions: suggestions}

	if includeUsurpers {
		p, err := s.Progress(userID, models.ImperialFigureRoleUsurper)
		if err != nil {
			return EmperorTrackerResult{}, err
		}
		result.Usurpers = &p
	}
	if includeEmpresses {
		p, err := s.Progress(userID, models.ImperialFigureRoleEmpress)
		if err != nil {
			return EmperorTrackerResult{}, err
		}
		result.Empresses = &p
	}
	if includeOther {
		p, err := s.Progress(userID, models.ImperialFigureRoleCaesar, models.ImperialFigureRoleOther)
		if err != nil {
			return EmperorTrackerResult{}, err
		}
		result.Other = &p
	}
	return result, nil
}

// SetHighlight stores a user's chosen tray display coin for a matched figure.
func (s *EmperorTrackerService) SetHighlight(userID, figureID, coinID uint) error {
	coins, err := s.coinRepo.ListMatchedImperialFigures(userID)
	if err != nil {
		return err
	}
	for _, coin := range coins {
		if coin.ID == coinID && coin.RomanImperialFigureID != nil && *coin.RomanImperialFigureID == figureID {
			return s.highlightRepo.Upsert(&models.RomanImperialFigureHighlight{
				UserID:                userID,
				RomanImperialFigureID: figureID,
				CoinID:                coinID,
			})
		}
	}
	return repository.ErrRecordNotFound
}

// ClearHighlight removes the user's chosen display coin for a figure.
func (s *EmperorTrackerService) ClearHighlight(userID, figureID uint) error {
	return s.highlightRepo.Delete(userID, figureID)
}

func (s *EmperorTrackerService) ownedCoinsByFigureID(userID uint) (map[uint][]models.Coin, error) {
	coins, err := s.coinRepo.ListMatchedImperialFigures(userID)
	if err != nil {
		return nil, err
	}
	byFigureID := make(map[uint][]models.Coin, len(coins))
	for i := range coins {
		coin := coins[i]
		if coin.RomanImperialFigureID == nil {
			continue
		}
		byFigureID[*coin.RomanImperialFigureID] = append(byFigureID[*coin.RomanImperialFigureID], coin)
	}
	return byFigureID, nil
}

func (s *EmperorTrackerService) highlightCoinIDsByFigureID(userID uint) (map[uint]uint, error) {
	highlights, err := s.highlightRepo.ListForUser(userID)
	if err != nil {
		return nil, err
	}
	byFigureID := make(map[uint]uint, len(highlights))
	for _, highlight := range highlights {
		byFigureID[highlight.RomanImperialFigureID] = highlight.CoinID
	}
	return byFigureID, nil
}

func highlightedCoin(coins []models.Coin, selectedCoinID uint) (*models.Coin, *uint) {
	if len(coins) == 0 {
		return nil, nil
	}
	for i := range coins {
		if selectedCoinID != 0 && coins[i].ID == selectedCoinID {
			return &coins[i], &coins[i].ID
		}
	}
	return &coins[0], &coins[0].ID
}

func buildCategoryProgress(figures []models.RomanImperialFigure, owned map[uint][]models.Coin, highlights map[uint]uint) CategoryProgress {
	dynastyIndex := make(map[string]int)
	dynasties := make([]DynastyProgress, 0)
	totalOwned := 0

	for _, figure := range figures {
		coins := owned[figure.ID]
		if coins == nil {
			coins = []models.Coin{}
		}
		coin, highlightedCoinID := highlightedCoin(coins, highlights[figure.ID])
		idx, ok := dynastyIndex[figure.Dynasty]
		if !ok {
			idx = len(dynasties)
			dynastyIndex[figure.Dynasty] = idx
			dynasties = append(dynasties, DynastyProgress{Dynasty: figure.Dynasty})
		}
		dynasties[idx].Total++
		dynasties[idx].Figures = append(dynasties[idx].Figures, ImperialFigureSlot{
			Figure:            figure,
			Coin:              coin,
			Coins:             coins,
			HighlightedCoinID: highlightedCoinID,
		})
		if coin != nil {
			dynasties[idx].Owned++
			totalOwned++
		}
	}

	percentage := 0.0
	if len(figures) > 0 {
		percentage = float64(totalOwned) / float64(len(figures)) * 100
	}
	return CategoryProgress{
		Owned:      totalOwned,
		Total:      len(figures),
		Percentage: percentage,
		Dynasties:  dynasties,
	}
}
