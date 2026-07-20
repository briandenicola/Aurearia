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

// SyncAllConfiguredUsers refreshes watchlists for every user with auction credentials
// configured, regardless of notification preferences (F026) — this keeps CurrentBid/status
// fresh in the background even for users who haven't set up Pushover.
func (s *AuctionWatchlistSyncService) SyncAllConfiguredUsers() AuctionWatchlistSyncStats {
	stats := AuctionWatchlistSyncStats{}
	users, err := s.userRepo.ListUsersWithAuctionCredentials()
	if err != nil {
		s.warn("Failed to list users with auction credentials: %v", err)
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

	// NumisBids is a reduced-functionality provider: the watchlist page carries
	// everything we need (image, title, sale name/date, starting price, watchlist ID)
	// without any per-lot HTTP requests. The site exposes no max-bid, winning-bidder,
	// or won/lost outcome signal, so CNG-style auto-detection is not applicable — lots
	// remain in Watching until the sale date passes, then flip to Passed. Manual status
	// override is required to record a Won or Lost result. See F022.
	now := time.Now()
	synced := 0
	for _, wl := range s.nbSvc.ParseWatchlist(raw) {
		status := models.AuctionStatusWatching
		saleDate := ParseSaleDate(wl.SaleDate)
		if saleDate != nil && saleDate.Before(now) {
			status = models.AuctionStatusPassed
		}
		lot := models.AuctionLot{
			NumisBidsURL: wl.URL,
			Source:       models.AuctionSourceNumisBids,
			SourceURL:    wl.URL,
			SourceLotID:  wl.SourceLotID,
			SourceSaleID: wl.SourceSaleID,
			SaleID:       wl.SaleID,
			LotNumber:    wl.LotNumber,
			Title:        wl.Title,
			ImageURL:     wl.ImageURL,
			Estimate:     wl.Estimate,
			Currency:     firstNonBlank(wl.Currency, "USD"),
			SaleName:     wl.SaleName,
			SaleDate:     saleDate,
			// AuctionEndTime must be set even though NumisBids only gives us a coarse
			// sale-wide date (not a precise per-lot close time, unlike CNG's
			// extended_end_time — see F021/F022): bid reminders (bidReminderDue in
			// auction_alert_service.go) hard-require AuctionEndTime and silently never
			// fire without it. A coarse deadline is strictly better than a reminder that
			// can never fire at all.
			AuctionEndTime: saleDate,
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

func (s *AuctionWatchlistSyncService) syncCNG(user *models.User) (int, error) {
	password, err := s.decryptStoredCredential(user, "cng_password", user.CNGPassword)
	if err != nil {
		return 0, err
	}
	client, err := s.cngSvc.Login(user.CNGUsername, password)
	if err != nil {
		return 0, err
	}

	// Used to detect whether a closed lot was won: compared against each lot's winning
	// bidder. Absence (e.g. a transient refresh-me failure) degrades gracefully — sync
	// still proceeds, it just can't auto-resolve won/lost for any newly-closed lots this run.
	customerRowID, err := s.cngSvc.CurrentCustomerRowID(client)
	if err != nil {
		s.warn("Could not determine CNG customer ID for user %d; won/lost auto-detection skipped this sync: %v", user.ID, err)
	}

	// The watched-lots list page already carries full bid detail (current bid, the user's own
	// max bid, and — once closed — the winning bidder) for every lot; no per-lot follow-up
	// request is needed.
	lots, err := s.cngSvc.FetchWatchlistLots(client)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	synced := 0
	for _, wl := range lots {
		auctionEndTime := ParseCNGDate(wl.SaleDate)

		// Presence of an absentee (max) bid means the user has placed a bid on this lot.
		status := models.AuctionStatusWatching
		if wl.MaxBid != nil {
			status = models.AuctionStatusBidding
		}

		var winningBid *float64
		switch {
		case wl.ProviderStatus != "" && wl.ProviderStatus != "active":
			// CNG reports the lot as closed. Resolve the real outcome instead of guessing
			// from end-time: a lot we were only watching (never bid on) is simply passed;
			// one we bid on is won or lost depending on who the final bid belongs to.
			switch {
			case wl.MaxBid == nil:
				status = models.AuctionStatusPassed
			case customerRowID != "" && wl.WinningCustomerRowID == customerRowID:
				status = models.AuctionStatusWon
				winningBid = firstNonNilFloat(wl.SoldPrice, wl.CurrentBid)
			default:
				status = models.AuctionStatusLost
			}
		case auctionEndTime != nil && auctionEndTime.Before(now):
			// Fallback for the rare case the provider status field itself is unavailable.
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
			MaxBid:         wl.MaxBid,
			WinningBid:     winningBid,
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
