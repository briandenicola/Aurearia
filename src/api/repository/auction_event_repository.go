package repository

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"gorm.io/gorm"
)

type AuctionEventRepository struct {
	db *gorm.DB
}

func NewAuctionEventRepository(db *gorm.DB) *AuctionEventRepository {
	return &AuctionEventRepository{db: db}
}

func (r *AuctionEventRepository) Create(event *models.AuctionEvent) error {
	return r.db.Create(event).Error
}

func (r *AuctionEventRepository) Update(event *models.AuctionEvent) error {
	return r.db.Save(event).Error
}

func (r *AuctionEventRepository) Delete(id uint, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.AuctionEvent{}).Error
}

func (r *AuctionEventRepository) GetByID(id uint, userID uint) (*models.AuctionEvent, error) {
	var event models.AuctionEvent
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&event).Error
	return &event, err
}

func (r *AuctionEventRepository) ListByUser(userID uint) ([]models.AuctionEvent, error) {
	var events []models.AuctionEvent
	err := r.db.Where("user_id = ?", userID).Order("start_date ASC").Find(&events).Error
	return events, err
}

// GetCalendar returns all auction lots and events for a user within a date range.
func (r *AuctionEventRepository) GetCalendar(userID uint, start, end time.Time) ([]models.AuctionLot, []models.AuctionEvent, error) {
	var lots []models.AuctionLot
	lotsQuery := r.db.Where("user_id = ?", userID)
	lotsQuery = lotsQuery.Where(
		r.db.Where("sale_date BETWEEN ? AND ?", start, end).
			Or("auction_end_time BETWEEN ? AND ?", start, end),
	)
	if err := lotsQuery.Order("sale_date ASC").Find(&lots).Error; err != nil {
		return nil, nil, err
	}

	var events []models.AuctionEvent
	eventsQuery := r.db.Where("user_id = ?", userID).Where(
		r.db.Where("start_date BETWEEN ? AND ?", start, end).
			Or("end_date BETWEEN ? AND ?", start, end).
			Or("start_date IS NULL"),
	)
	if err := eventsQuery.Order("start_date ASC").Find(&events).Error; err != nil {
		return nil, nil, err
	}

	return lots, events, nil
}
