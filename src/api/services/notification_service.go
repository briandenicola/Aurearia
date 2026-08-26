package services

import (
	"fmt"
	"html"
	"net/url"
	"sort"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

// NotificationService handles creating and managing notifications.
type NotificationService struct {
	notifRepo   *repository.NotificationRepository
	socialRepo  *repository.SocialRepository
	userRepo    *repository.UserRepository
	pushoverSvc *PushoverService
	logger      *Logger
}

const NotificationTypeFollowRequest = "follow_request"

const (
	NotificationTypeAuctionPriceAlert  = "auction_price_alert"
	NotificationTypeAuctionBidReminder = "auction_bid_reminder"
	NotificationTypeAuctionEndingSoon  = "auction_ending_soon"
	// NotificationTypeAuctionLotsTracked is fired by the background watchlist sync when it
	// picks up lots it was not tracking before (newly watched, or newly bid on). One
	// notification covers every new lot found in a single sync run — see
	// NotifyAuctionLotsTracked.
	NotificationTypeAuctionLotsTracked = "auction_lots_tracked"
	NotificationTypeShipmentStatus     = "shipment_status"
	// NotificationTypeAvailabilityRun is the terminal-outcome notification created for every
	// terminal child AvailabilityRun (owner/scheduled/admin-triggered), in addition to (never
	// instead of) any per-coin wishlist_unavailable notifications fired during the same run (D6).
	NotificationTypeAvailabilityRun = "wishlist_availability_run"
	// NotificationTypePurchaseReminder is fired by the daily reminder scheduler when a
	// pending reminder's remind_date has arrived. ReferenceID = reminder.ID, ReferenceURL = /coin/{coinID}.
	NotificationTypePurchaseReminder = "purchase_reminder"
)

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	notifRepo *repository.NotificationRepository,
	socialRepo *repository.SocialRepository,
	userRepo *repository.UserRepository,
	pushoverSvc *PushoverService,
	logger *Logger,
) *NotificationService {
	return &NotificationService{
		notifRepo:   notifRepo,
		socialRepo:  socialRepo,
		userRepo:    userRepo,
		pushoverSvc: pushoverSvc,
		logger:      logger,
	}
}

// NotifyWishlistUnavailable creates a notification when a wishlist coin
// is detected as no longer available.
func (s *NotificationService) NotifyWishlistUnavailable(userID uint, coin models.Coin, reason string) {
	coinName := coin.Name
	if coinName == "" {
		coinName = "Unnamed coin"
	}

	title := "Wishlist item unavailable"
	message := fmt.Sprintf("%s appears to no longer be available. %s", coinName, reason)

	n := &models.Notification{
		UserID:       userID,
		Type:         "wishlist_unavailable",
		Title:        title,
		Message:      message,
		ReferenceID:  coin.ID,
		ReferenceURL: coin.ReferenceURL,
	}

	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create wishlist notification for user %d, coin %d: %v", userID, coin.ID, err)
	}

	go s.sendPushover(userID, title, message, coin.ReferenceURL)
}

// NotifyAvailabilityRunTerminal creates the owner-facing terminal-outcome notification for a
// single completed or failed child AvailabilityRun. Exactly one is created per terminal child
// run, regardless of how many (if any) per-coin wishlist_unavailable notifications also fired
// for that run — this is purely additive and never gates or replaces the per-coin call (D6).
// The message is always generic: no URLs, no query text, no internal error details (FR-015);
// it may mention up to 3 newly-unavailable coin names (plus "and N more") for readability.
func (s *NotificationService) NotifyAvailabilityRunTerminal(userID uint, run *models.AvailabilityRun, newlyUnavailableCoinNames []string) {
	title := "Wishlist availability check complete"
	message := fmt.Sprintf(
		"Checked %d item(s): %d available, %d unavailable, %d unknown.",
		run.CoinsChecked, run.Available, run.Unavailable, run.Unknown,
	)

	if run.Status == models.AvailabilityRunStatusFailed {
		title = "Wishlist availability check failed"
		message = models.GenericAvailabilityFailureMessage
	} else if len(newlyUnavailableCoinNames) > 0 {
		message = fmt.Sprintf("%s %s", message, summarizeUnavailableCoinNames(newlyUnavailableCoinNames))
	}

	refURL := fmt.Sprintf("/wishlist/availability-runs/%d", run.ID)
	n := &models.Notification{
		UserID:       userID,
		Type:         NotificationTypeAvailabilityRun,
		Title:        title,
		Message:      message,
		ReferenceID:  run.ID,
		ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create availability run notification for user %d, run %d: %v", userID, run.ID, err)
	}

	go s.sendPushover(userID, title, message, refURL)
}

// NotifyAdminCycleChildFailure notifies the admin who triggered a cycle that one of its
// per-owner child runs failed. The message is generic (owner username + cycle ID only) — no
// URLs, no query text, no internal error details (FR-012, FR-015).
func (s *NotificationService) NotifyAdminCycleChildFailure(adminID uint, ownerUsername string, cycleID uint) {
	if adminID == 0 {
		return
	}
	if ownerUsername == "" {
		ownerUsername = "a user"
	}

	title := "Wishlist availability check failed"
	message := fmt.Sprintf("The wishlist availability check for %s failed (cycle #%d). %s",
		ownerUsername, cycleID, models.GenericAvailabilityFailureMessage)
	refURL := fmt.Sprintf("/admin/availability-cycles/%d", cycleID)

	n := &models.Notification{
		UserID:       adminID,
		Type:         NotificationTypeAvailabilityRun,
		Title:        title,
		Message:      message,
		ReferenceID:  cycleID,
		ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create admin cycle-child-failure notification for admin %d, cycle %d: %v", adminID, cycleID, err)
	}

	go s.sendPushover(adminID, title, message, refURL)
}

// summarizeUnavailableCoinNames formats up to 3 newly-unavailable coin names plus an
// "and N more" suffix for the remainder, for use in the generic availability-run summary.
func summarizeUnavailableCoinNames(names []string) string {
	const maxShown = 3
	if len(names) == 0 {
		return ""
	}
	shown := names
	suffix := ""
	if len(names) > maxShown {
		shown = names[:maxShown]
		suffix = fmt.Sprintf(", and %d more", len(names)-maxShown)
	}
	return fmt.Sprintf("Newly unavailable: %s%s.", strings.Join(shown, ", "), suffix)
}

// NotifyShipmentStatusTransition creates an in-app shipment milestone notification
// and best-effort Pushover push when available.
func (s *NotificationService) NotifyShipmentStatusTransition(
	userID uint,
	coinID uint,
	shipmentID uint,
	previousStatus models.ShipmentStatus,
	currentStatus models.ShipmentStatus,
) {
	title := "Shipment update"
	message := fmt.Sprintf(
		"Shipment for coin #%d changed from %s to %s.",
		coinID,
		formatShipmentStatusLabel(previousStatus),
		formatShipmentStatusLabel(currentStatus),
	)
	refURL := fmt.Sprintf("/coin/%d", coinID)

	n := &models.Notification{
		UserID:       userID,
		Type:         NotificationTypeShipmentStatus,
		Title:        title,
		Message:      message,
		ReferenceID:  shipmentID,
		ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create shipment notification for user %d, shipment %d: %v", userID, shipmentID, err)
	}
	go s.sendPushover(userID, title, message, refURL)
}

// NotifyNewCoin creates notifications for all accepted followers when a user
// adds a new coin to their collection (non-wishlist only).
func (s *NotificationService) NotifyNewCoin(ownerID uint, coin models.Coin) {
	if coin.IsWishlist {
		return
	}

	followers, err := s.socialRepo.GetAcceptedFollowerIDs(ownerID)
	if err != nil {
		s.logger.Error("notifications", "Failed to get followers for user %d: %v", ownerID, err)
		return
	}

	if len(followers) == 0 {
		return
	}

	// Look up the owner's username for the message
	ownerName := fmt.Sprintf("User #%d", ownerID)
	if user, err := s.socialRepo.GetUserByID(ownerID); err == nil && user != nil {
		ownerName = user.Username
	}

	coinName := coin.Name
	if coinName == "" {
		coinName = "a new coin"
	}

	for _, followerID := range followers {
		n := &models.Notification{
			UserID:      followerID,
			Type:        "friend_new_coin",
			Title:       "New coin added",
			Message:     fmt.Sprintf("%s added %s to their collection.", ownerName, coinName),
			ReferenceID: coin.ID,
		}
		if err := s.notifRepo.Create(n); err != nil {
			s.logger.Error("notifications", "Failed to notify follower %d about coin %d: %v", followerID, coin.ID, err)
		}
		go s.sendPushover(followerID, "New coin added", fmt.Sprintf("%s added %s to their collection.", ownerName, coinName), "")
	}

	s.logger.Debug("notifications", "Notified %d followers about new coin %d from user %d", len(followers), coin.ID, ownerID)
}

// NotifyFollowRequest creates a notification for a user who received a new
// follower request.
func (s *NotificationService) NotifyFollowRequest(followerID, targetID uint) {
	if followerID == 0 || targetID == 0 || followerID == targetID {
		return
	}

	followerName := fmt.Sprintf("User #%d", followerID)
	if user, err := s.socialRepo.GetUserByID(followerID); err == nil && user != nil {
		followerName = user.Username
	}

	title := "New follower request"
	message := fmt.Sprintf("%s requested to follow you.", followerName)

	n := &models.Notification{
		UserID:       targetID,
		Type:         NotificationTypeFollowRequest,
		Title:        title,
		Message:      message,
		ReferenceID:  followerID,
		ReferenceURL: "/followers",
	}

	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create follow-request notification for user %d from follower %d: %v", targetID, followerID, err)
		return
	}

	go s.sendPushover(targetID, title, message, "/followers")
}

// NotifyAuctionPriceAlert creates an in-app notification when a tracked lot's price alert
// crosses its target, and best-effort attempts a Pushover push if the user has one
// configured. Pushover delivery failing (or not being configured) never prevents the in-app
// notification — this fixes a gap where users without Pushover configured got no
// notification for auction events at all (specs/_backlog/F027).
func (s *NotificationService) NotifyAuctionPriceAlert(userID uint, lot models.AuctionLot, targetPrice float64) {
	title := "Auction Price Alert"
	message := fmt.Sprintf(
		"%s\n%s\nTarget: %.2f %s\nCurrent bid: %s",
		auctionLotTitle(lot),
		auctionLotLabel(lot),
		targetPrice,
		auctionCurrency(lot.Currency),
		formatAuctionBid(lot.CurrentBid, lot.Currency),
	)
	refURL := auctionLotURL(lot)

	n := &models.Notification{
		UserID: userID, Type: NotificationTypeAuctionPriceAlert, Title: title, Message: message,
		ReferenceID: lot.ID, ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create price alert notification for user %d, lot %d: %v", userID, lot.ID, err)
	}
	go s.sendPushoverWithURLTitle(userID, title, message, refURL, "View auction lot")
}

// NotifyAuctionBidReminder creates an in-app notification for a bid reminder that has come
// due, and best-effort attempts a Pushover push. See NotifyAuctionPriceAlert for why Pushover
// is best-effort here.
func (s *NotificationService) NotifyAuctionBidReminder(userID uint, lot models.AuctionLot, minutesBefore int) {
	title := "Auction Bid Reminder"
	message := fmt.Sprintf(
		"%s\n%s\nReminder: %d minutes before close\nCurrent bid: %s",
		auctionLotTitle(lot),
		auctionLotLabel(lot),
		minutesBefore,
		formatAuctionBid(lot.CurrentBid, lot.Currency),
	)
	refURL := auctionLotURL(lot)

	n := &models.Notification{
		UserID: userID, Type: NotificationTypeAuctionBidReminder, Title: title, Message: message,
		ReferenceID: lot.ID, ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create bid reminder notification for user %d, lot %d: %v", userID, lot.ID, err)
	}
	go s.sendPushoverWithURLTitle(userID, title, message, refURL, "View auction lot")
}

// NotifyAuctionEndingSoon creates a single in-app notification consolidating every lot a user
// is bidding on that closes within the ending-soon window. Pushover delivery for this event
// is handled separately by AuctionEndingScheduler (which manages its own per-day dedup state);
// this method only handles the in-app side.
func (s *NotificationService) NotifyAuctionEndingSoon(userID uint, lots []models.AuctionLot) {
	if len(lots) == 0 {
		return
	}
	message := fmt.Sprintf("%d auction(s) you are bidding on end within 24 hours:\n\n", len(lots))
	for _, lot := range lots {
		auctionHouse := lot.AuctionHouse
		if auctionHouse == "" {
			auctionHouse = "Unknown House"
		}
		saleName := lot.SaleName
		if saleName == "" {
			saleName = "Sale"
		}
		message += fmt.Sprintf("- %s - %s (Lot %d)\n", auctionHouse, saleName, lot.LotNumber)
	}
	n := &models.Notification{
		UserID:  userID,
		Type:    NotificationTypeAuctionEndingSoon,
		Title:   "Auctions Ending Soon",
		Message: message,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create ending-soon notification for user %d: %v", userID, err)
	}
}

// NotifyAuctionLotsTracked creates a single in-app notification, plus a single rich-HTML
// Pushover push, for every lot a background watchlist sync started tracking in one run —
// whether it landed on the watchlist or was picked up as one the user is already bidding on.
// Batching is the whole point: a sync that discovers a dozen new lots must produce one
// notification, not a dozen (F031). Pushover delivery stays best-effort and never gates the
// in-app notification, per F027.
func (s *NotificationService) NotifyAuctionLotsTracked(userID uint, lots []models.AuctionLot) {
	if len(lots) == 0 {
		return
	}

	title := auctionLotsTrackedTitle(len(lots))
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Auction sync started tracking %d new lot(s):\n\n", len(lots)))
	for i, lot := range lots {
		if i == auctionLotsTrackedInAppLimit {
			builder.WriteString(fmt.Sprintf("… and %d more\n", len(lots)-i))
			break
		}
		builder.WriteString(fmt.Sprintf("%s\n%s\n\n", auctionLotTitle(lot), auctionLotTrackingLabel(lot)))
	}

	// A single new lot can deep-link straight to itself; a batch can only sensibly land on
	// the auctions list, since no one page shows an arbitrary set of lots.
	refURL := "/auctions"
	var refID uint
	if len(lots) == 1 {
		refURL = auctionLotAppPath(lots[0].ID)
		refID = lots[0].ID
	}

	n := &models.Notification{
		UserID:       userID,
		Type:         NotificationTypeAuctionLotsTracked,
		Title:        title,
		Message:      strings.TrimRight(builder.String(), "\n"),
		ReferenceID:  refID,
		ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create auction lots tracked notification for user %d: %v", userID, err)
	}

	go s.sendPushoverMessage(userID, buildAuctionLotsTrackedPushoverMessage(lots, s.publicAppBaseURL()))
}

// NotifyCoinOfDay creates an in-app notification and Pushover alert for the
// user's daily featured coin. The ReferenceID points to the FeaturedCoin record
// so the frontend can open the dedicated modal.
func (s *NotificationService) NotifyCoinOfDay(userID uint, featuredCoinID, coinID uint, coinName, summary string) {
	if coinName == "" {
		coinName = "Today's coin"
	}

	title := "Coin of the Day"
	message := coinName
	if summary != "" {
		// Keep notification message short — the modal shows the full summary.
		preview := summary
		if len(preview) > 140 {
			preview = preview[:137] + "..."
		}
		message = fmt.Sprintf("%s — %s", coinName, preview)
	}

	n := &models.Notification{
		UserID:      userID,
		Type:        "coin_of_day",
		Title:       title,
		Message:     message,
		ReferenceID: featuredCoinID,
	}

	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create coin-of-day notification for user %d: %v", userID, err)
	}

	go s.sendPushoverMessage(userID, buildCoinOfDayPushoverMessage(title, coinID, coinName, summary, s.publicAppBaseURL()))
}

// NotifyPurchaseReminder creates an in-app notification when a purchase reminder's
// remind_date arrives. ReferenceID is the reminder ID; ReferenceURL deep-links to
// the coin detail page. Pushover is best-effort.
func (s *NotificationService) NotifyPurchaseReminder(userID, reminderID, coinID uint, coinName string) {
	if coinName == "" {
		coinName = "Unnamed coin"
	}
	title := "Purchase Reminder"
	message := fmt.Sprintf("%s is on your wishlist — time to buy!", coinName)
	refURL := fmt.Sprintf("/coin/%d", coinID)
	n := &models.Notification{
		UserID:       userID,
		Type:         NotificationTypePurchaseReminder,
		Title:        title,
		Message:      message,
		ReferenceID:  reminderID,
		ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create purchase reminder notification for user %d, reminder %d: %v", userID, reminderID, err)
	}
	go s.sendPushover(userID, title, message, refURL)
}

// NotifyAIJobCompleted creates a notification when an asynchronous AI job completes.
func (s *NotificationService) NotifyAIJobCompleted(userID, jobID, coinID uint, coinName, jobType string) {
	if coinName == "" {
		coinName = "coin"
	}
	label := formatAIJobType(jobType)
	title := fmt.Sprintf("AI %s complete", label)
	message := fmt.Sprintf("%s is ready.", coinName)
	refURL := fmt.Sprintf("/coin/%d", coinID)
	n := &models.Notification{
		UserID:       userID,
		Type:         "ai_job_completed",
		Title:        title,
		Message:      message,
		ReferenceID:  jobID,
		ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create AI job completion notification for user %d, job %d: %v", userID, jobID, err)
	}
	go s.sendPushover(userID, title, message, refURL)
}

// NotifyAIJobFailed creates a notification when an asynchronous AI job fails.
func (s *NotificationService) NotifyAIJobFailed(userID, jobID, coinID uint, jobType, reason string) {
	label := formatAIJobType(jobType)
	title := fmt.Sprintf("AI %s failed", label)
	message := fmt.Sprintf("AI %s could not be completed.", label)
	if reason != "" {
		message = fmt.Sprintf("%s Please check AI provider configuration and try again.", message)
	}
	refURL := fmt.Sprintf("/coin/%d", coinID)
	n := &models.Notification{
		UserID:       userID,
		Type:         "ai_job_failed",
		Title:        title,
		Message:      message,
		ReferenceID:  jobID,
		ReferenceURL: refURL,
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create AI job failure notification for user %d, job %d: %v", userID, jobID, err)
	}
	go s.sendPushover(userID, title, message, refURL)
}

// NotifyValuationRunComplete creates an in-app notification when a background valuation run completes.
func (s *NotificationService) NotifyValuationRunComplete(userID, runID uint, checked, updated, skipped, errors int) {
	title := "Valuation complete"
	message := fmt.Sprintf("Checked: %d | Updated: %d | Skipped: %d | Errors: %d", checked, updated, skipped, errors)
	n := &models.Notification{
		UserID:       userID,
		Type:         "valuation_complete",
		Title:        title,
		Message:      message,
		ReferenceID:  runID,
		ReferenceURL: "/stats/value-trends",
	}
	if err := s.notifRepo.Create(n); err != nil {
		s.logger.Error("notifications", "Failed to create valuation completion notification for user %d, run %d: %v", userID, runID, err)
	}
}

// NotifyAPIKeyRotationRequired creates a single actionable notification that lists
// active API key names that must be recreated.
func (s *NotificationService) NotifyAPIKeyRotationRequired(userID uint, keyNames []string) error {
	if len(keyNames) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(keyNames))
	seen := make(map[string]struct{}, len(keyNames))
	for _, keyName := range keyNames {
		name := strings.TrimSpace(keyName)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)

	n := &models.Notification{
		UserID:       userID,
		Type:         NotificationTypeAPIKeyRotationRequired,
		Title:        "Action required: Recreate API keys",
		Message:      fmt.Sprintf("Recreate these API keys in Settings: %s", strings.Join(normalized, ", ")),
		ReferenceURL: "/settings",
	}
	if err := s.notifRepo.ReplaceByUserAndType(n); err != nil {
		s.logger.Error("notifications", "Failed to create API key rotation notification for user %d: %v", userID, err)
		return err
	}
	return nil
}

// sendPushover checks if the user has Pushover enabled and sends a push notification.
func (s *NotificationService) sendPushover(userID uint, title, message, refURL string) {
	s.sendPushoverMessage(userID, PushoverMessage{
		Title:   title,
		Message: message,
		URL:     refURL,
	})
}

// sendPushoverWithURLTitle behaves like sendPushover but also sets a Pushover URL title so the
// action link shows a readable label (e.g. "View auction lot") instead of the raw URL.
func (s *NotificationService) sendPushoverWithURLTitle(userID uint, title, message, refURL, urlTitle string) {
	s.sendPushoverMessage(userID, PushoverMessage{
		Title:    title,
		Message:  message,
		URL:      refURL,
		URLTitle: urlTitle,
	})
}

// sendPushoverMessage checks if the user has Pushover enabled and sends a push notification.
func (s *NotificationService) sendPushoverMessage(userID uint, message PushoverMessage) {
	if s.pushoverSvc == nil || s.userRepo == nil {
		return
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return
	}

	if !user.PushoverEnabled || user.PushoverUserKey == "" {
		return
	}

	message.UserKey = user.PushoverUserKey
	if err := s.pushoverSvc.SendMessage(message); err != nil {
		s.logger.Error("pushover", "Failed to send Pushover notification to user %d: %v", userID, err)
	}
}

func (s *NotificationService) publicAppBaseURL() string {
	if s == nil || s.pushoverSvc == nil || s.pushoverSvc.settingsSvc == nil {
		return ""
	}
	return s.pushoverSvc.settingsSvc.GetSetting(SettingPublicAppURL)
}

func buildCoinOfDayPushoverMessage(title string, coinID uint, coinName, summary, publicAppBaseURL string) PushoverMessage {
	if coinName == "" {
		coinName = "Today's coin"
	}

	body := fmt.Sprintf("<b>%s</b>", html.EscapeString(coinName))
	if summary != "" {
		body = fmt.Sprintf("%s — %s", body, html.EscapeString(truncateRunes(summary, 140)))
	}

	coinURL := buildCoinOfDayURL(publicAppBaseURL, coinID)
	if coinURL != "" {
		body = fmt.Sprintf("%s — <a href=\"%s\">Open coin</a>", body, html.EscapeString(coinURL))
	}

	return PushoverMessage{
		Title:   title,
		Message: body,
		URL:     coinURL,
		HTML:    true,
	}
}

// auctionLotsTrackedInAppLimit caps how many lots the in-app notification body lists before
// collapsing the rest into an "… and N more" line. The Pushover body has its own, tighter cap
// (pushoverMessageLimit) because the API rejects oversized messages outright.
const auctionLotsTrackedInAppLimit = 10

// buildAuctionLotsTrackedPushoverMessage renders the batched new-lot push as Pushover HTML:
// per lot, the coin name in bold, then the auction house/sale, lot number and how the lot is
// being tracked, then a fully-qualified link into the app's auction view for that exact lot.
// Every interpolated value is HTML-escaped — lot titles are scraped from third-party auction
// sites, so they can carry markup. Lots are dropped (with a trailing summary line) once the
// body would exceed Pushover's message length limit, mirroring the watch-bid digest, so a
// large sync can never produce a message the API rejects.
func buildAuctionLotsTrackedPushoverMessage(lots []models.AuctionLot, publicAppBaseURL string) PushoverMessage {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Auction sync started tracking %d new lot(s):\n\n", len(lots)))

	for i, lot := range lots {
		entry := fmt.Sprintf(
			"<b>%s</b>\n%s\n",
			html.EscapeString(auctionLotTitle(lot)),
			html.EscapeString(auctionLotTrackingLabel(lot)),
		)
		if lotURL := buildAuctionLotAppURL(publicAppBaseURL, lot.ID); lotURL != "" {
			entry += fmt.Sprintf("<a href=\"%s\">View lot in Aurearia</a>\n", html.EscapeString(lotURL))
		}
		entry += "\n"

		omittedNote := fmt.Sprintf("… %d more lot(s) omitted\n", len(lots)-i)
		if builder.Len()+len(entry) > pushoverMessageLimit-len(omittedNote) {
			builder.WriteString(omittedNote)
			break
		}
		builder.WriteString(entry)
	}

	// The notification-level action link mirrors the in-app reference: straight to the lot
	// when there is exactly one, otherwise the auctions list.
	actionURL := buildPublicAppURL(publicAppBaseURL, "/auctions")
	urlTitle := "Open auctions"
	if len(lots) == 1 {
		if lotURL := buildAuctionLotAppURL(publicAppBaseURL, lots[0].ID); lotURL != "" {
			actionURL = lotURL
			urlTitle = "View auction lot"
		}
	}

	return PushoverMessage{
		Title:    auctionLotsTrackedTitle(len(lots)),
		Message:  strings.TrimRight(builder.String(), "\n"),
		URL:      actionURL,
		URLTitle: urlTitle,
		HTML:     true,
	}
}

func auctionLotsTrackedTitle(count int) string {
	if count == 1 {
		return "New Auction Lot Tracked"
	}
	return fmt.Sprintf("%d New Auction Lots Tracked", count)
}

// auctionLotTrackingLabel adds how the lot is being tracked to the shared auction/lot label,
// so a lot synced as an active bid is distinguishable from one that is only being watched.
func auctionLotTrackingLabel(lot models.AuctionLot) string {
	// NumisBids watchlist rows carry a sale name but no auction house; naming the provider
	// reads better in a notification than auctionLotLabel's generic "Auction" fallback. The
	// lot is a copy, so this only affects the rendered label.
	if strings.TrimSpace(lot.AuctionHouse) == "" {
		lot.AuctionHouse = auctionSourceLabel(lot.Source)
	}
	return fmt.Sprintf("%s — %s", auctionLotLabel(lot), auctionLotTrackingVerb(lot))
}

func auctionSourceLabel(source models.AuctionSource) string {
	switch source {
	case models.AuctionSourceCNG:
		return "CNG Auctions"
	case models.AuctionSourceNumisBids:
		return "NumisBids"
	default:
		// Unknown source: leave it blank so auctionLotLabel applies its own fallback.
		return ""
	}
}

func auctionLotTrackingVerb(lot models.AuctionLot) string {
	if lot.Status == models.AuctionStatusBidding {
		return "Bidding"
	}
	return "Watching"
}

// auctionLotAppPath is the in-app (relative) route that opens a single lot. The auctions page
// reads the lot query parameter and opens that lot's detail modal.
func auctionLotAppPath(lotID uint) string {
	if lotID == 0 {
		return "/auctions"
	}
	return fmt.Sprintf("/auctions?lot=%d", lotID)
}

func formatAIJobType(jobType string) string {
	switch jobType {
	case "analysis":
		return "analysis"
	case "value_estimate":
		return "value estimate"
	default:
		return "AI job"
	}
}

func buildCoinOfDayURL(publicAppBaseURL string, coinID uint) string {
	return buildPublicAppURL(publicAppBaseURL, fmt.Sprintf("/coin/%d", coinID))
}

// buildAuctionLotAppURL returns the fully-qualified link to a lot inside the app, or "" when
// no valid public app URL is configured — a relative link in a push notification is broken on
// the device, so it is better to omit the link entirely (same rule as coin-of-the-day).
func buildAuctionLotAppURL(publicAppBaseURL string, lotID uint) string {
	return buildPublicAppURL(publicAppBaseURL, auctionLotAppPath(lotID))
}

// buildPublicAppURL joins an app-relative path onto the configured public app URL, returning
// "" unless that setting is a usable absolute http(s) URL.
func buildPublicAppURL(publicAppBaseURL, path string) string {
	base := strings.TrimRight(strings.TrimSpace(publicAppBaseURL), "/")
	if base == "" {
		return ""
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ""
	}
	return base + path
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-3]) + "..."
}

func formatShipmentStatusLabel(status models.ShipmentStatus) string {
	value := strings.ReplaceAll(string(status), "_", " ")
	if value == "" {
		return "unknown"
	}
	words := strings.Fields(value)
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}
