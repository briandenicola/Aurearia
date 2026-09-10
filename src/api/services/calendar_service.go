package services

import (
	"errors"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var ErrCalendarEventNotFound = errors.New("calendar event not found")

type CalendarService struct {
	repo        *repository.AuctionEventRepository
	quickAccess *QuickAccessService
}

func NewCalendarService(repo *repository.AuctionEventRepository, quickAccess *QuickAccessService) *CalendarService {
	return &CalendarService{repo: repo, quickAccess: quickAccess}
}

func (s *CalendarService) Create(event *models.AuctionEvent) error {
	event.Origin = models.AuctionEventOriginManual
	return s.repo.Create(event)
}

func (s *CalendarService) Update(id, userID uint, apply func(*models.AuctionEvent)) error {
	event, err := s.repo.GetByID(id, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return ErrCalendarEventNotFound
		}
		return err
	}
	apply(event)
	return s.repo.Update(event)
}

func (s *CalendarService) Delete(id, userID uint) error {
	return s.repo.RunInTransaction(func(tx *repository.Transaction) error {
		txRepo := s.repo.WithTransaction(tx)
		if _, err := txRepo.GetByID(id, userID); err != nil {
			if repository.IsRecordNotFound(err) {
				return ErrCalendarEventNotFound
			}
			return err
		}
		if s.quickAccess != nil {
			if err := s.quickAccess.RemoveTargetInTx(tx, userID, models.QuickAccessTargetCalendarEvent, id); err != nil {
				return err
			}
		}
		return txRepo.Delete(id, userID)
	})
}
