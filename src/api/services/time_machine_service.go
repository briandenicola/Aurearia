package services

import (
	"errors"
	"sort"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	// ErrTimeMachineFutureDate is returned for a requested date after today.
	// The collection's future is not a thing we can report on.
	ErrTimeMachineFutureDate = errors.New("date is in the future")
)

// topCoinLimit bounds the "largest holdings" list returned per snapshot.
const topCoinLimit = 5

// TimeMachineService reconstructs a collection's state on a past date.
type TimeMachineService struct {
	repo  *repository.TimeMachineRepository
	clock func() time.Time
}

func NewTimeMachineService(repo *repository.TimeMachineRepository) *TimeMachineService {
	return &TimeMachineService{repo: repo, clock: time.Now}
}

// WithClock overrides the time source. Tests use it to make "today" deterministic.
func (s *TimeMachineService) WithClock(clock func() time.Time) *TimeMachineService {
	s.clock = clock
	return s
}

// BreakdownEntry is one slice of a categorical breakdown.
type BreakdownEntry struct {
	Label string  `json:"label"`
	Count int     `json:"count"`
	Value float64 `json:"value"`
}

// TopCoin is one of the most valuable holdings on the requested date.
type TopCoin struct {
	ID               uint    `json:"id"`
	Name             string  `json:"name"`
	Value            float64 `json:"value"`
	ValueFromHistory bool    `json:"valueFromHistory"`
}

// ValueBasis reports how each coin's value was determined, so the UI can be
// explicit that early dates lean on purchase price rather than valuations.
type ValueBasis struct {
	FromValuationHistory int `json:"fromValuationHistory"`
	FromPurchasePrice    int `json:"fromPurchasePrice"`
}

// TimeMachineSnapshot is the collection as it stood on AsOfDate.
type TimeMachineSnapshot struct {
	AsOfDate       string           `json:"asOfDate"`
	CoinCount      int              `json:"coinCount"`
	TotalValue     float64          `json:"totalValue"`
	TotalInvested  float64          `json:"totalInvested"`
	UnrealizedGain float64          `json:"unrealizedGain"`
	ByCategory     []BreakdownEntry `json:"byCategory"`
	ByMaterial     []BreakdownEntry `json:"byMaterial"`
	ByEra          []BreakdownEntry `json:"byEra"`
	TopCoins       []TopCoin        `json:"topCoins"`
	AcquiredInYear int              `json:"acquiredInYear"`
	HealthScore    *int             `json:"healthScore"`
	ValueBasis     ValueBasis       `json:"valueBasis"`
	// UndatedCoinCount is how many owned coins have no purchase date and so
	// appear at no point on the timeline. Surfaced rather than hidden: a
	// non-zero value means the numbers below understate the collection.
	UndatedCoinCount int `json:"undatedCoinCount"`
}

// TimeMachineBounds is the addressable range of the timeline.
type TimeMachineBounds struct {
	EarliestDate string `json:"earliestDate"`
	LatestDate   string `json:"latestDate"`
	HasData      bool   `json:"hasData"`
}

const dateLayout = "2006-01-02"

// GetSnapshot reconstructs the collection as of the given date.
//
// The date is interpreted as end-of-day so that a coin purchased on the
// requested date is included, which is what a user scrubbing to "the day I
// bought the Trajan" expects.
func (s *TimeMachineService) GetSnapshot(userID uint, asOf time.Time) (*TimeMachineSnapshot, error) {
	endOfDay := endOfDayUTC(asOf)
	if endOfDay.After(endOfDayUTC(s.clock())) {
		return nil, ErrTimeMachineFutureDate
	}

	coins, err := s.repo.GetCollectionAsOf(userID, endOfDay)
	if err != nil {
		return nil, err
	}
	undated, err := s.repo.CountUndatedCoins(userID)
	if err != nil {
		return nil, err
	}
	healthScore, err := s.repo.GetHealthScoreAsOf(userID, endOfDay)
	if err != nil {
		return nil, err
	}

	snapshot := &TimeMachineSnapshot{
		AsOfDate:         endOfDay.Format(dateLayout),
		CoinCount:        len(coins),
		HealthScore:      healthScore,
		UndatedCoinCount: int(undated),
		ByCategory:       []BreakdownEntry{},
		ByMaterial:       []BreakdownEntry{},
		ByEra:            []BreakdownEntry{},
		TopCoins:         []TopCoin{},
	}

	byCategory := map[string]*BreakdownEntry{}
	byMaterial := map[string]*BreakdownEntry{}
	byEra := map[string]*BreakdownEntry{}
	yearStart := endOfDay.AddDate(-1, 0, 0)

	for _, coin := range coins {
		snapshot.TotalValue += coin.ValueAsOf
		if coin.PurchasePrice != nil {
			snapshot.TotalInvested += *coin.PurchasePrice
		}
		if coin.ValueFromHistory {
			snapshot.ValueBasis.FromValuationHistory++
		} else {
			snapshot.ValueBasis.FromPurchasePrice++
		}
		if coin.PurchaseDate != nil && coin.PurchaseDate.After(yearStart) {
			snapshot.AcquiredInYear++
		}

		accumulate(byCategory, coin.Category, coin.ValueAsOf)
		accumulate(byMaterial, coin.Material, coin.ValueAsOf)
		accumulate(byEra, coin.Era, coin.ValueAsOf)
	}

	snapshot.UnrealizedGain = snapshot.TotalValue - snapshot.TotalInvested
	snapshot.ByCategory = sortedBreakdown(byCategory)
	snapshot.ByMaterial = sortedBreakdown(byMaterial)
	snapshot.ByEra = sortedBreakdown(byEra)

	// The repository already orders by value descending.
	for i, coin := range coins {
		if i >= topCoinLimit {
			break
		}
		snapshot.TopCoins = append(snapshot.TopCoins, TopCoin{
			ID:               coin.ID,
			Name:             coin.Name,
			Value:            coin.ValueAsOf,
			ValueFromHistory: coin.ValueFromHistory,
		})
	}

	return snapshot, nil
}

// GetBounds returns the timeline's addressable range: the first acquisition on
// record through today. HasData is false when nothing is dated, in which case
// the UI should explain that rather than render an empty scrubber.
func (s *TimeMachineService) GetBounds(userID uint) (*TimeMachineBounds, error) {
	bounds, err := s.repo.GetBounds(userID)
	if err != nil {
		return nil, err
	}
	today := s.clock().UTC().Format(dateLayout)
	if bounds.EarliestPurchase == nil {
		return &TimeMachineBounds{EarliestDate: today, LatestDate: today, HasData: false}, nil
	}
	return &TimeMachineBounds{
		EarliestDate: bounds.EarliestPurchase.UTC().Format(dateLayout),
		LatestDate:   today,
		HasData:      true,
	}, nil
}

// ParseDate parses a YYYY-MM-DD timeline date.
func ParseDate(value string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, value, time.UTC)
}

func endOfDayUTC(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
}

func accumulate(into map[string]*BreakdownEntry, label string, value float64) {
	if label == "" {
		label = "Unspecified"
	}
	entry, ok := into[label]
	if !ok {
		entry = &BreakdownEntry{Label: label}
		into[label] = entry
	}
	entry.Count++
	entry.Value += value
}

// sortedBreakdown orders by count descending then label, so a scrubbed timeline
// keeps a stable ordering between adjacent dates instead of reshuffling.
func sortedBreakdown(from map[string]*BreakdownEntry) []BreakdownEntry {
	out := make([]BreakdownEntry, 0, len(from))
	for _, entry := range from {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Label < out[j].Label
	})
	return out
}
