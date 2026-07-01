package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

type AuctionWatchlistSyncStats struct {
	UsersChecked int
	LotsSynced   int
	Errors       int
}

type AuctionWatchlistSyncService struct {
	auctionRepo *repository.AuctionLotRepository
	userRepo    *repository.UserRepository
	nbSvc       *NumisBidsService
	cngSvc      *CNGAuctionService
	credentials *CredentialEncryptionService
	logger      *Logger
}

func NewAuctionWatchlistSyncService(
	auctionRepo *repository.AuctionLotRepository,
	userRepo *repository.UserRepository,
	nbSvc *NumisBidsService,
	cngSvc *CNGAuctionService,
	credentials *CredentialEncryptionService,
	logger *Logger,
) *AuctionWatchlistSyncService {
	if credentials == nil {
		credentials = NewDisabledCredentialEncryptionService()
	}
	return &AuctionWatchlistSyncService{
		auctionRepo: auctionRepo,
		userRepo:    userRepo,
		nbSvc:       nbSvc,
		cngSvc:      cngSvc,
		credentials: credentials,
		logger:      logger,
	}
}

func (s *AuctionWatchlistSyncService) SyncDigestEligibleUsers() AuctionWatchlistSyncStats {
	stats := AuctionWatchlistSyncStats{}
	users, err := s.userRepo.ListAuctionWatchDigestEligible()
	if err != nil {
		s.warn("Failed to list auction digest users: %v", err)
		stats.Errors++
		return stats
	}

	for i := range users {
		stats.UsersChecked++
		synced, err := s.SyncUser(&users[i])
		stats.LotsSynced += synced
		if err != nil {
			stats.Errors++
			s.warn("Scheduled auction watchlist sync failed for user %d: %v", users[i].ID, err)
		}
	}
	return stats
}

func (s *AuctionWatchlistSyncService) SyncUser(user *models.User) (int, error) {
	if user == nil {
		return 0, fmt.Errorf("user is required")
	}

	total := 0
	var errs []string
	if user.NumisBidsUsername != "" && user.NumisBidsPassword != "" {
		synced, err := s.syncNumisBids(user)
		total += synced
		if err != nil {
			errs = append(errs, fmt.Sprintf("numisbids: %v", err))
		}
	}
	if user.CNGUsername != "" && user.CNGPassword != "" {
		synced, err := s.syncCNG(user)
		total += synced
		if err != nil {
			errs = append(errs, fmt.Sprintf("cng: %v", err))
		}
	}
	if len(errs) > 0 {
		return total, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return total, nil
}

func (s *AuctionWatchlistSyncService) syncNumisBids(user *models.User) (int, error) {
	password, err := s.decryptStoredCredential(user, "numis_bids_password", user.NumisBidsPassword)
	if err != nil {
		return 0, err
	}
	client, err := s.nbSvc.Login(user.NumisBidsUsername, password)
	if err != nil {
		return 0, err
	}
	raw, err := s.nbSvc.FetchWatchlist(client)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	synced := 0
	for _, wl := range s.nbSvc.ParseWatchlist(raw) {
		if details, err := s.nbSvc.ScrapeLotPage(wl.URL); err == nil {
			if details.ImageURL != "" {
				wl.ImageURL = details.ImageURL
			}
			wl.AuctionHouse = details.AuctionHouse
			wl.SaleName = details.SaleName
			wl.SaleDate = details.SaleDate
			wl.Description = details.Description
			wl.CurrentBid = details.CurrentBid
			if details.Currency != "" {
				wl.Currency = details.Currency
			}
			if details.LotNumber > 0 {
				wl.LotNumber = details.LotNumber
			}
		} else {
			s.warn("Could not refresh NumisBids lot page for scheduled sync user %d url=%s: %v", user.ID, wl.URL, err)
		}

		status := models.AuctionStatusWatching
		saleDate := ParseSaleDate(wl.SaleDate)
		if saleDate != nil && saleDate.Before(now) {
			status = models.AuctionStatusPassed
		}
		lot := models.AuctionLot{
			NumisBidsURL: wl.URL,
			Source:       models.AuctionSourceNumisBids,
			SourceURL:    wl.URL,
			SourceSaleID: wl.SourceSaleID,
			SaleID:       wl.SaleID,
			LotNumber:    wl.LotNumber,
			Title:        wl.Title,
			Description:  wl.Description,
			ImageURL:     wl.ImageURL,
			Estimate:     wl.Estimate,
			CurrentBid:   wl.CurrentBid,
			Currency:     firstNonBlank(wl.Currency, "USD"),
			AuctionHouse: wl.AuctionHouse,
			SaleName:     wl.SaleName,
			SaleDate:     saleDate,
			Status:       status,
			UserID:       user.ID,
		}
		if _, err := s.auctionRepo.UpsertWithCalendarEvent(&lot); err != nil {
			return synced, err
		}
		synced++
	}

	s.auctionRepo.MarkPastAuctionsAsPassed(user.ID, now)
	return synced, nil
}

func (s *AuctionWatchlistSyncService) syncCNG(user *models.User) (int, error) {
	password, err := s.decryptStoredCredential(user, "cng_password", user.CNGPassword)
	if err != nil {
		return 0, err
	}
	client, err := s.cngSvc.Login(user.CNGUsername, password)
	if err != nil {
		return 0, err
	}
	lots, err := s.cngSvc.FetchWatchlistLots(client)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	synced := 0
	for _, wl := range lots {
		status := models.AuctionStatusWatching
		auctionEndTime := ParseCNGDate(wl.SaleDate)
		if auctionEndTime != nil && auctionEndTime.Before(now) {
			status = models.AuctionStatusPassed
		}
		lot := models.AuctionLot{
			NumisBidsURL:   strings.TrimSpace(wl.URL),
			Source:         models.AuctionSourceCNG,
			SourceURL:      strings.TrimSpace(wl.URL),
			SourceLotID:    wl.SourceLotID,
			SourceSaleID:   firstNonBlank(wl.SourceSaleID, wl.SaleID),
			SaleID:         wl.SaleID,
			LotNumber:      wl.LotNumber,
			Title:          wl.Title,
			Description:    wl.Description,
			ImageURL:       wl.ImageURL,
			Estimate:       wl.Estimate,
			CurrentBid:     wl.CurrentBid,
			Currency:       firstNonBlank(wl.Currency, "USD"),
			AuctionHouse:   wl.AuctionHouse,
			SaleName:       wl.SaleName,
			AuctionEndTime: auctionEndTime,
			Status:         status,
			UserID:         user.ID,
		}
		if _, err := s.auctionRepo.UpsertWithCalendarEvent(&lot); err != nil {
			return synced, err
		}
		synced++
	}

	s.auctionRepo.MarkPastAuctionsAsPassed(user.ID, now)
	return synced, nil
}

func (s *AuctionWatchlistSyncService) decryptStoredCredential(user *models.User, field string, stored string) (string, error) {
	plain, wasEncrypted, err := s.credentials.DecryptStringWithAAD(stored, AuctionCredentialAAD(user.ID, field))
	if err != nil {
		return "", err
	}
	if s.credentials.Enabled() && !wasEncrypted && stored != "" {
		encrypted, err := s.credentials.EncryptStringWithAAD(plain, AuctionCredentialAAD(user.ID, field))
		if err != nil {
			s.warn("Failed to encrypt legacy auction credential for user %d: %v", user.ID, err)
			return plain, nil
		}
		if encrypted != plain {
			if err := s.userRepo.UpdateField(user, field, encrypted); err != nil {
				s.warn("Failed to save encrypted legacy auction credential for user %d: %v", user.ID, err)
			}
		}
	}
	return plain, nil
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *AuctionWatchlistSyncService) warn(format string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Warn("auction-watch-sync", format, args...)
	}
}
