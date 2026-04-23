package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

// CatCount holds a category and its coin count for portfolio summary.
type CatCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// MatCount holds a material and its coin count for portfolio summary.
type MatCount struct {
	Material string `json:"material"`
	Count    int    `json:"count"`
}

// PortfolioEraCount holds an era and its coin count for portfolio summary.
type PortfolioEraCount struct {
	Era   string `json:"era"`
	Count int    `json:"count"`
}

// PortfolioRulerCount holds a ruler and its coin count for portfolio summary.
type PortfolioRulerCount struct {
	Ruler string `json:"ruler"`
	Count int    `json:"count"`
}

// TopCoin holds a top-valued coin's summary fields.
type TopCoin struct {
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	CurrentValue *float64 `json:"currentValue"`
	Ruler        string   `json:"ruler"`
	Era          string   `json:"era"`
	Grade        string   `json:"grade"`
}

// PortfolioSummary holds all aggregated portfolio data.
type PortfolioSummary struct {
	TotalCoins    int64                `json:"totalCoins"`
	TotalValue    float64              `json:"totalValue"`
	TotalInvested float64              `json:"totalInvested"`
	Categories    []CatCount           `json:"categories"`
	Materials     []MatCount           `json:"materials"`
	Eras          []PortfolioEraCount  `json:"eras"`
	Rulers        []PortfolioRulerCount `json:"rulers"`
	TopCoins      []TopCoin            `json:"topCoins"`
}

// AgentRepository encapsulates database operations for the agent handler.
type AgentRepository struct {
	db *gorm.DB
}

// NewAgentRepository creates a new AgentRepository.
func NewAgentRepository(db *gorm.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

// GetPortfolioSummary returns aggregated collection data for AI portfolio analysis.
func (r *AgentRepository) GetPortfolioSummary(userID uint) (*PortfolioSummary, error) {
	activeFilter := "user_id = ? AND is_wishlist = ? AND is_sold = ?"

	var totalCoins int64
	r.db.Model(&models.Coin{}).Where(activeFilter, userID, false, false).Count(&totalCoins)

	var totalValue float64
	r.db.Model(&models.Coin{}).Where(activeFilter, userID, false, false).
		Select("COALESCE(SUM(current_value), 0)").Scan(&totalValue)

	var totalInvested float64
	r.db.Model(&models.Coin{}).Where(activeFilter, userID, false, false).
		Select("COALESCE(SUM(purchase_price), 0)").Scan(&totalInvested)

	var categories []CatCount
	r.db.Model(&models.Coin{}).
		Select("category, COUNT(*) as count").
		Where(activeFilter, userID, false, false).
		Group("category").Order("count DESC").Find(&categories)

	var materials []MatCount
	r.db.Model(&models.Coin{}).
		Select("material, COUNT(*) as count").
		Where(activeFilter, userID, false, false).
		Group("material").Order("count DESC").Find(&materials)

	var eras []PortfolioEraCount
	r.db.Model(&models.Coin{}).
		Select("era, COUNT(*) as count").
		Where(activeFilter+" AND era != ''", userID, false, false).
		Group("era").Order("count DESC").Limit(15).Find(&eras)

	var rulers []PortfolioRulerCount
	r.db.Model(&models.Coin{}).
		Select("ruler, COUNT(*) as count").
		Where(activeFilter+" AND ruler != ''", userID, false, false).
		Group("ruler").Order("count DESC").Limit(15).Find(&rulers)

	var topCoins []TopCoin
	r.db.Model(&models.Coin{}).
		Select("name, category, current_value, ruler, era, grade").
		Where(activeFilter+" AND current_value IS NOT NULL", userID, false, false).
		Order("current_value DESC").Limit(10).Find(&topCoins)

	return &PortfolioSummary{
		TotalCoins:    totalCoins,
		TotalValue:    totalValue,
		TotalInvested: totalInvested,
		Categories:    categories,
		Materials:     materials,
		Eras:          eras,
		Rulers:        rulers,
		TopCoins:      topCoins,
	}, nil
}

// FindCoinForUser finds a coin by ID and user ID.
func (r *AgentRepository) FindCoinForUser(coinID uint, userID uint) (*models.Coin, error) {
	var coin models.Coin
	if err := r.db.Where("id = ? AND user_id = ?", coinID, userID).First(&coin).Error; err != nil {
		return nil, err
	}
	return &coin, nil
}

// RecordValueHistory creates a coin value history entry.
func (r *AgentRepository) RecordValueHistory(entry *models.CoinValueHistory) error {
	return r.db.Create(entry).Error
}

// CreateJournalEntry creates a coin journal entry.
func (r *AgentRepository) CreateJournalEntry(entry *models.CoinJournal) error {
	return r.db.Create(entry).Error
}
