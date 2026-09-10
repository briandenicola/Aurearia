package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QuickAccessRepository struct {
	db *gorm.DB
}

type QuickAccessCoinRow struct {
	ID              uint
	Name            string
	IsWishlist      bool
	IsSold          bool
	PrimaryImageURL string
}

type QuickAccessSetRow struct {
	ID      uint
	Name    string
	SetType models.CoinSetType
	Color   string
	Icon    string
}

type QuickAccessAuctionLotRow struct {
	ID             uint
	Title          string
	Status         models.AuctionLotStatus
	AuctionHouse   string
	SaleDate       *time.Time
	AuctionEndTime *time.Time
	ImageURL       string
}

type QuickAccessCalendarEventRow struct {
	ID           uint
	Title        string
	AuctionHouse string
	StartDate    *time.Time
	EndDate      *time.Time
	Origin       models.AuctionEventOrigin
}

func NewQuickAccessRepository(db *gorm.DB) *QuickAccessRepository {
	return &QuickAccessRepository{db: db}
}

func (r *QuickAccessRepository) WithTx(tx *gorm.DB) *QuickAccessRepository {
	return &QuickAccessRepository{db: tx}
}

func (r *QuickAccessRepository) WithTransaction(tx *Transaction) *QuickAccessRepository {
	return &QuickAccessRepository{db: tx.db}
}

func (r *QuickAccessRepository) RunInTransaction(fn func(tx *Transaction) error) error {
	return r.db.Transaction(func(db *gorm.DB) error {
		return fn(NewTransaction(db))
	})
}

func (r *QuickAccessRepository) CreateOrGet(pin *models.QuickAccessPin) (bool, error) {
	result := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "target_type"}, {Name: "target_id"}},
		DoNothing: true,
	}).Create(pin)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		return true, nil
	}
	return false, r.db.
		Where("user_id = ? AND target_type = ? AND target_id = ?", pin.UserID, pin.TargetType, pin.TargetID).
		First(pin).Error
}

func (r *QuickAccessRepository) Get(userID uint, targetType models.QuickAccessTargetType, targetID uint) (*models.QuickAccessPin, error) {
	var pin models.QuickAccessPin
	err := r.db.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).First(&pin).Error
	if err != nil {
		return nil, err
	}
	return &pin, nil
}

func (r *QuickAccessRepository) List(userID uint) ([]models.QuickAccessPin, error) {
	pins := make([]models.QuickAccessPin, 0)
	err := r.db.Scopes(OwnedBy(userID)).Order("pinned_at DESC").Order("id DESC").Find(&pins).Error
	return pins, err
}

func (r *QuickAccessRepository) CountByType(userID uint, targetType models.QuickAccessTargetType) (int64, error) {
	var count int64
	err := r.db.Model(&models.QuickAccessPin{}).
		Where("user_id = ? AND target_type = ?", userID, targetType).
		Count(&count).Error
	return count, err
}

func (r *QuickAccessRepository) Delete(userID uint, targetType models.QuickAccessTargetType, targetID uint) error {
	return r.db.
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Delete(&models.QuickAccessPin{}).Error
}

func (r *QuickAccessRepository) DeleteTargets(userID uint, targetType models.QuickAccessTargetType, targetIDs []uint) error {
	if len(targetIDs) == 0 {
		return nil
	}
	return r.db.
		Where("user_id = ? AND target_type = ? AND target_id IN ?", userID, targetType, targetIDs).
		Delete(&models.QuickAccessPin{}).Error
}

func (r *QuickAccessRepository) SetCoinSetPinnedAt(setID, userID uint, pinnedAt *time.Time) error {
	result := r.db.Model(&models.CoinSet{}).Scopes(OwnedByID(setID, userID)).Update("pinned_at", pinnedAt)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *QuickAccessRepository) FindCoin(userID, id uint) (*QuickAccessCoinRow, error) {
	rows, err := r.FindCoins(userID, []uint{id})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *QuickAccessRepository) FindCoins(userID uint, ids []uint) ([]QuickAccessCoinRow, error) {
	rows := make([]QuickAccessCoinRow, 0)
	if len(ids) == 0 {
		return rows, nil
	}
	err := r.db.Model(&models.Coin{}).
		Select(`coins.id, coins.name, coins.is_wishlist, coins.is_sold,
			COALESCE((SELECT file_path FROM coin_images
				WHERE coin_images.coin_id = coins.id
				ORDER BY coin_images.is_primary DESC, coin_images.id ASC LIMIT 1), '') AS primary_image_url`).
		Scopes(OwnedBy(userID)).
		Where("coins.id IN ?", ids).
		Scan(&rows).Error
	return rows, err
}

func (r *QuickAccessRepository) FindCoinSet(userID, id uint) (*QuickAccessSetRow, error) {
	rows, err := r.FindCoinSets(userID, []uint{id})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *QuickAccessRepository) FindCoinSets(userID uint, ids []uint) ([]QuickAccessSetRow, error) {
	rows := make([]QuickAccessSetRow, 0)
	if len(ids) == 0 {
		return rows, nil
	}
	err := r.db.Model(&models.CoinSet{}).
		Select("id, name, set_type, color, icon").
		Scopes(OwnedBy(userID)).
		Where("id IN ?", ids).
		Scan(&rows).Error
	return rows, err
}

func (r *QuickAccessRepository) FindAuctionLot(userID, id uint) (*QuickAccessAuctionLotRow, error) {
	rows, err := r.FindAuctionLots(userID, []uint{id})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *QuickAccessRepository) FindAuctionLots(userID uint, ids []uint) ([]QuickAccessAuctionLotRow, error) {
	rows := make([]QuickAccessAuctionLotRow, 0)
	if len(ids) == 0 {
		return rows, nil
	}
	err := r.db.Model(&models.AuctionLot{}).
		Select("id, title, status, auction_house, sale_date, auction_end_time, image_url").
		Scopes(OwnedBy(userID)).
		Where("id IN ?", ids).
		Scan(&rows).Error
	return rows, err
}

func (r *QuickAccessRepository) FindCalendarEvent(userID, id uint) (*QuickAccessCalendarEventRow, error) {
	rows, err := r.FindCalendarEvents(userID, []uint{id})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &rows[0], nil
}

func (r *QuickAccessRepository) FindCalendarEvents(userID uint, ids []uint) ([]QuickAccessCalendarEventRow, error) {
	rows := make([]QuickAccessCalendarEventRow, 0)
	if len(ids) == 0 {
		return rows, nil
	}
	err := r.db.Model(&models.AuctionEvent{}).
		Select("id, title, auction_house, start_date, end_date, origin").
		Scopes(OwnedBy(userID)).
		Where("id IN ?", ids).
		Scan(&rows).Error
	return rows, err
}
