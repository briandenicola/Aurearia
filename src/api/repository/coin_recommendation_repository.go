package repository

import (
	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CoinRecommendationRepository struct {
	db *gorm.DB
}

func NewCoinRecommendationRepository(db *gorm.DB) *CoinRecommendationRepository {
	return &CoinRecommendationRepository{db: db}
}

func (r *CoinRecommendationRepository) GetCoinByID(coinID, userID uint) (*models.Coin, error) {
	var coin models.Coin
	if err := r.db.Scopes(OwnedByID(coinID, userID)).
		Preload("Tags").
		Preload("Sets").
		First(&coin).Error; err != nil {
		return nil, err
	}
	return &coin, nil
}

func (r *CoinRecommendationRepository) ListOwnedCoinsWithMemberships(userID uint) ([]models.Coin, error) {
	var coins []models.Coin
	err := r.db.Scopes(OwnedBy(userID)).
		Preload("Tags").
		Preload("Sets").
		Find(&coins).Error
	return coins, err
}

func (r *CoinRecommendationRepository) ListOwnedManualSets(userID uint) ([]models.CoinSet, error) {
	var sets []models.CoinSet
	err := r.db.Model(&models.CoinSet{}).
		Where("user_id = ? AND set_type IN ?", userID, []string{string(models.CoinSetTypeStandard), string(models.CoinSetTypeGoal)}).
		Order("name ASC").
		Find(&sets).Error
	return sets, err
}

func (r *CoinRecommendationRepository) ListOwnedTags(userID uint) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Model(&models.Tag{}).Scopes(OwnedBy(userID)).Order("name ASC").Find(&tags).Error
	return tags, err
}

func (r *CoinRecommendationRepository) ListByCoin(coinID, userID uint) ([]models.CoinRecommendation, error) {
	var recs []models.CoinRecommendation
	err := r.db.Where("coin_id = ? AND user_id = ?", coinID, userID).
		Order("score DESC, id ASC").
		Find(&recs).Error
	return recs, err
}

func (r *CoinRecommendationRepository) UpsertMany(recommendations []models.CoinRecommendation) error {
	if len(recommendations) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "coin_id"},
			{Name: "target_type"},
			{Name: "target_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"score":      gorm.Expr("excluded.score"),
			"confidence": gorm.Expr("excluded.confidence"),
			"reasons":    gorm.Expr("excluded.reasons"),
			"status":     string(models.RecommendationStatusPending),
		}),
	}).Create(&recommendations).Error
}

func (r *CoinRecommendationRepository) GetByID(recID, coinID, userID uint) (*models.CoinRecommendation, error) {
	var rec models.CoinRecommendation
	if err := r.db.Where("id = ? AND coin_id = ? AND user_id = ?", recID, coinID, userID).First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *CoinRecommendationRepository) UpdateStatus(recID uint, status models.RecommendationStatus) error {
	return r.db.Model(&models.CoinRecommendation{}).
		Where("id = ?", recID).
		Update("status", status).Error
}

func (r *CoinRecommendationRepository) CreateFeedback(feedback *models.RecommendationFeedback) error {
	return r.db.Create(feedback).Error
}
