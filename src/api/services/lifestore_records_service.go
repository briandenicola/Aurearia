package services

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// LifestoreRecordsSchema names the shape of GET /api/lifestore/records, so a
// reader can tell a future, incompatible shape apart.
const LifestoreRecordsSchema = "aurearia.lifestore.records/1"

const (
	lifestoreDefaultLimit = 100
	lifestoreMaxLimit     = 500
)

// ErrInvalidLifestorePage is returned for a page token this server did not give out.
var ErrInvalidLifestorePage = errors.New("invalid lifestore page token")

// LifestoreRecordsService lists everything a user owns for the life store, a
// read-only service that keeps a household-wide copy and reads it in full.
type LifestoreRecordsService struct {
	coins *repository.CoinRepository
}

// NewLifestoreRecordsService creates a LifestoreRecordsService.
func NewLifestoreRecordsService(coins *repository.CoinRepository) *LifestoreRecordsService {
	return &LifestoreRecordsService{coins: coins}
}

// LifestoreRecordsPage is one page of the feed.
type LifestoreRecordsPage struct {
	Schema  string            `json:"schema" example:"aurearia.lifestore.records/1"`
	Records []LifestoreRecord `json:"records"`
	// Next is the token for the following page; empty on the last page.
	Next string `json:"next" example:"coin:142"`
}

// LifestoreRecord is one record: today every record is a coin.
type LifestoreRecord struct {
	Kind      string        `json:"kind" example:"coin"`
	ID        string        `json:"id" example:"142"`
	UpdatedAt time.Time     `json:"updatedAt"`
	Data      LifestoreCoin `json:"data"`
}

// LifestoreCoin is a coin with what belongs to it. Amounts are in the
// collection's own currency; Aurearia does not store one.
type LifestoreCoin struct {
	ID                    uint                      `json:"id"`
	Name                  string                    `json:"name"`
	Category              string                    `json:"category"`
	Denomination          string                    `json:"denomination"`
	Ruler                 string                    `json:"ruler"`
	Era                   string                    `json:"era"`
	DateRange             string                    `json:"dateRange"`
	Mint                  string                    `json:"mint"`
	Material              string                    `json:"material"`
	WeightGrams           *float64                  `json:"weightGrams"`
	DiameterMm            *float64                  `json:"diameterMm"`
	Grade                 string                    `json:"grade"`
	RarityRating          string                    `json:"rarityRating"`
	ObverseInscription    string                    `json:"obverseInscription"`
	ReverseInscription    string                    `json:"reverseInscription"`
	ObverseDescription    string                    `json:"obverseDescription"`
	ReverseDescription    string                    `json:"reverseDescription"`
	Notes                 string                    `json:"notes"`
	PurchasePrice         *float64                  `json:"purchasePrice"`
	PurchaseDate          *time.Time                `json:"purchaseDate"`
	PurchaseLocation      string                    `json:"purchaseLocation"`
	VendorInvoice         string                    `json:"vendorInvoice"`
	CurrentValue          *float64                  `json:"currentValue"`
	CurrentValueUpdatedAt *time.Time                `json:"currentValueUpdatedAt"`
	IsWishlist            bool                      `json:"isWishlist"`
	IsSold                bool                      `json:"isSold"`
	SoldPrice             *float64                  `json:"soldPrice"`
	SoldDate              *time.Time                `json:"soldDate"`
	SoldTo                string                    `json:"soldTo"`
	IsPrivate             bool                      `json:"isPrivate"`
	StorageLocation       string                    `json:"storageLocation"`
	Tags                  []string                  `json:"tags"`
	Sets                  []string                  `json:"sets"`
	References            []LifestoreCoinReference  `json:"references"`
	Images                int                       `json:"images"`
	Journal               []LifestoreJournalEntry   `json:"journal"`
	ValueHistory          []LifestoreValueRecording `json:"valueHistory"`
	CreatedAt             time.Time                 `json:"createdAt"`
	UpdatedAt             time.Time                 `json:"updatedAt"`
}

// LifestoreCoinReference is a catalog attribution.
type LifestoreCoinReference struct {
	Catalog string `json:"catalog"`
	Volume  string `json:"volume"`
	Number  string `json:"number"`
	URI     string `json:"uri"`
}

// LifestoreJournalEntry is one dated journal entry.
type LifestoreJournalEntry struct {
	Entry     string    `json:"entry"`
	CreatedAt time.Time `json:"createdAt"`
}

// LifestoreValueRecording is one recorded value.
type LifestoreValueRecording struct {
	Value      float64   `json:"value"`
	Confidence string    `json:"confidence"`
	Source     string    `json:"source"`
	RecordedAt time.Time `json:"recordedAt"`
}

// Page returns the user's records after the page token, in a stable order.
// An empty token starts at the beginning. limit is clamped to 1..500 and
// defaults to 100.
func (s *LifestoreRecordsService) Page(userID uint, token string, limit int) (*LifestoreRecordsPage, error) {
	after, err := parseLifestorePage(token)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = lifestoreDefaultLimit
	}
	if limit > lifestoreMaxLimit {
		limit = lifestoreMaxLimit
	}
	// One extra row says whether another page follows.
	coins, err := s.coins.PageForLifestore(userID, after, limit+1)
	if err != nil {
		return nil, err
	}
	page := &LifestoreRecordsPage{Schema: LifestoreRecordsSchema, Records: []LifestoreRecord{}}
	if len(coins) > limit {
		coins = coins[:limit]
		page.Next = "coin:" + strconv.FormatUint(uint64(coins[len(coins)-1].ID), 10)
	}
	ids := make([]uint, len(coins))
	for i, c := range coins {
		ids[i] = c.ID
	}
	journals, err := s.coins.JournalsForCoins(userID, ids)
	if err != nil {
		return nil, err
	}
	history, err := s.coins.ValueHistoryForCoins(userID, ids)
	if err != nil {
		return nil, err
	}
	journalByCoin := map[uint][]LifestoreJournalEntry{}
	for _, j := range journals {
		journalByCoin[j.CoinID] = append(journalByCoin[j.CoinID], LifestoreJournalEntry{Entry: j.Entry, CreatedAt: j.CreatedAt})
	}
	historyByCoin := map[uint][]LifestoreValueRecording{}
	for _, h := range history {
		historyByCoin[h.CoinID] = append(historyByCoin[h.CoinID], LifestoreValueRecording{Value: h.Value, Confidence: h.Confidence, Source: h.Source, RecordedAt: h.RecordedAt})
	}
	for _, c := range coins {
		page.Records = append(page.Records, LifestoreRecord{
			Kind:      "coin",
			ID:        strconv.FormatUint(uint64(c.ID), 10),
			UpdatedAt: c.UpdatedAt,
			Data:      lifestoreCoin(c, journalByCoin[c.ID], historyByCoin[c.ID]),
		})
	}
	return page, nil
}

// parseLifestorePage reads "coin:<id>", the only token Page gives out.
func parseLifestorePage(token string) (uint, error) {
	if token == "" {
		return 0, nil
	}
	kind, id, ok := strings.Cut(token, ":")
	if !ok || kind != "coin" {
		return 0, ErrInvalidLifestorePage
	}
	n, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return 0, ErrInvalidLifestorePage
	}
	return uint(n), nil
}

func lifestoreCoin(c models.Coin, journal []LifestoreJournalEntry, history []LifestoreValueRecording) LifestoreCoin {
	out := LifestoreCoin{
		ID: c.ID, Name: c.Name, Category: string(c.Category), Denomination: c.Denomination, Ruler: c.Ruler,
		Era: string(c.Era), DateRange: c.DateRange, Mint: c.Mint, Material: string(c.Material),
		WeightGrams: c.WeightGrams, DiameterMm: c.DiameterMm, Grade: c.Grade, RarityRating: c.RarityRating,
		ObverseInscription: c.ObverseInscription, ReverseInscription: c.ReverseInscription,
		ObverseDescription: c.ObverseDescription, ReverseDescription: c.ReverseDescription, Notes: c.Notes,
		PurchasePrice: c.PurchasePrice, PurchaseDate: c.PurchaseDate, PurchaseLocation: c.PurchaseLocation,
		VendorInvoice: c.VendorInvoice, CurrentValue: c.CurrentValue, CurrentValueUpdatedAt: c.CurrentValueUpdatedAt,
		IsWishlist: c.IsWishlist, IsSold: c.IsSold, SoldPrice: c.SoldPrice, SoldDate: c.SoldDate, SoldTo: c.SoldTo,
		IsPrivate: c.IsPrivate, Images: len(c.Images), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
		Tags: []string{}, Sets: []string{}, References: []LifestoreCoinReference{},
		Journal: []LifestoreJournalEntry{}, ValueHistory: []LifestoreValueRecording{},
	}
	if c.MintLocation != nil && c.MintLocation.DisplayName != "" {
		out.Mint = c.MintLocation.DisplayName
	}
	if c.StorageLocation != nil {
		out.StorageLocation = c.StorageLocation.Name
	}
	for _, t := range c.Tags {
		out.Tags = append(out.Tags, t.Name)
	}
	for _, s := range c.Sets {
		out.Sets = append(out.Sets, s.Name)
	}
	for _, r := range c.References {
		out.References = append(out.References, LifestoreCoinReference{Catalog: r.Catalog, Volume: r.Volume, Number: r.Number, URI: r.URI})
	}
	if journal != nil {
		out.Journal = journal
	}
	if history != nil {
		out.ValueHistory = history
	}
	return out
}
