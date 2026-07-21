package services

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupWishlistSearchAlertService(t *testing.T) (*WishlistSearchAlertService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.User{}, &models.StorageLocation{}, &models.Coin{}, &models.CoinImage{}, &models.CoinReference{}, &models.ValueSnapshot{}, &models.AppSetting{}, &models.AvailabilityRun{}, &models.AvailabilityResult{}, &models.WishlistSearchAlert{}, &models.AlertRun{}, &models.AlertCandidate{}, &models.CandidateProvenance{}, &models.CandidateReviewAction{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	coinSvc := NewCoinService(repository.NewCoinRepository(db), nil).
		WithStorageLocationSupport(repository.NewStorageLocationRepository(db))
	return NewWishlistSearchAlertService(repository.NewWishlistSearchAlertRepository(db)).
		WithCoinCreation(coinSvc), db
}

func setupWishlistSearchAlertDiscoveryService(t *testing.T, handler http.HandlerFunc) (*WishlistSearchAlertService, *gorm.DB, func()) {
	t.Helper()
	svc, db := setupWishlistSearchAlertService(t)
	server := httptest.NewServer(handler)
	settingsRepo := repository.NewSettingsRepository(db)
	if err := db.Create(&models.AppSetting{Key: SettingAIProvider, Value: "ollama"}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	svc.WithDiscovery(NewAgentProxy(server.URL, "internal-token", NewLogger(10)), NewSettingsService(settingsRepo))
	return svc, db, server.Close
}
func validAlertInput() WishlistSearchAlertInput {
	active := true
	return WishlistSearchAlertInput{
		Name:     "Domitian denarius under $300",
		Cadence:  "manual",
		IsActive: &active,
		Criteria: WishlistAlertCriteriaInput{
			RulerOrIssuer: "Domitian",
			CoinType:      "Denarius",
			PriceMax:      alertFloatPtr(300),
			Currency:      "USD",
			SourceFilters: []string{"https://www.vcoins.com/store"},
			Keywords:      "RIC Minerva",
		},
	}
}

func alertFloatPtr(v float64) *float64 { return &v }
func alertIntPtr(v int) *int           { return &v }

func TestWishlistSearchAlertService_CRUDScopesToOwner(t *testing.T) {
	svc, _ := setupWishlistSearchAlertService(t)
	created, err := svc.CreateAlert(1, validAlertInput())
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	if created.UserID != 1 || created.SourceFilters[0] != "www.vcoins.com" {
		t.Fatalf("unexpected created alert: %+v", created)
	}
	if _, err := svc.GetAlert(created.ID, 2); !errors.Is(err, ErrWishlistSearchAlertNotFound) {
		t.Fatalf("cross-owner get error = %v", err)
	}
	list, total, err := svc.ListAlerts(1, nil, 1, 20)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("list got len=%d total=%d err=%v", len(list), total, err)
	}
	updatedInput := validAlertInput()
	updatedInput.Name = "Updated discovery alert"
	updatedInput.Cadence = "weekly"
	updated, err := svc.UpdateAlert(created.ID, 1, updatedInput)
	if err != nil {
		t.Fatalf("owner update: %v", err)
	}
	if updated.Name != "Updated discovery alert" || updated.Cadence != models.WishlistAlertCadenceWeekly {
		t.Fatalf("unexpected update: %+v", updated)
	}
	if _, err := svc.SetAlertActive(created.ID, 1, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if err := svc.DeleteAlert(created.ID, 1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.GetAlert(created.ID, 1); !errors.Is(err, ErrWishlistSearchAlertNotFound) {
		t.Fatalf("get deleted error = %v", err)
	}
}

func TestWishlistSearchAlertService_RunNowPersistsCandidatesAndSuppressesDuplicates(t *testing.T) {
	agent := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search/alerts" {
			t.Fatalf("unexpected agent path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates":[{
				"source_url":"https://www.vcoins.com/en/stores/example/123?utm_source=x",
				"source_name":"VCoins Example",
				"title":"Domitian AR Denarius",
				"observed_price":225.0,
				"observed_currency":"USD",
				"reason_for_match":"Title and price match the alert.",
				"last_seen_at":"2026-06-29T17:00:10Z",
				"provenance_status":"verified",
				"fields":{"ruler":"Domitian"},
				"provenance":[{
					"field":"source_url",
					"value":"https://www.vcoins.com/en/stores/example/123",
					"source_url":"https://www.vcoins.com/en/stores/example/123",
					"observed_at":"2026-06-29T17:00:10Z",
					"confidence":"high",
					"verification_state":"verified"
				}]
			}],
			"warnings":[],
			"partial":false
		}`))
	}
	svc, db, cleanup := setupWishlistSearchAlertDiscoveryService(t, agent)
	defer cleanup()
	alert, err := svc.CreateAlert(1, validAlertInput())
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	savedCoin := models.Coin{UserID: 1, Name: "Saved wishlist coin", IsWishlist: true, ReferenceURL: "https://dealer.example/saved", ListingStatus: "available", ListingCheckReason: "seeded"}
	if err := db.Create(&savedCoin).Error; err != nil {
		t.Fatalf("seed saved wishlist coin: %v", err)
	}
	firstQueued, err := svc.RunNow(alert.ID, 1, RunAlertInput{MaxCandidates: 20})
	if err != nil {
		t.Fatalf("queue first run: %v", err)
	}
	if firstQueued.Status != models.AlertRunStatusQueued || firstQueued.ResultCount != 0 {
		t.Fatalf("first run should return queued before agent processing: %+v", firstQueued)
	}
	if err := svc.ProcessRun(firstQueued.RunID); err != nil {
		t.Fatalf("process first run: %v", err)
	}
	first, err := svc.GetRun(alert.ID, firstQueued.RunID, 1)
	if err != nil {
		t.Fatalf("load first run: %v", err)
	}
	if first.Status != models.AlertRunStatusCompleted || first.NewCount != 1 || first.DuplicateCount != 0 {
		t.Fatalf("unexpected first run: %+v", first)
	}
	if len(first.Candidates) != 1 || len(first.Candidates[0].Provenance) == 0 {
		t.Fatalf("first run did not preserve candidate provenance: %+v", first.Candidates)
	}
	if first.Candidates[0].LifecycleState != models.AlertCandidateStateActive || first.Candidates[0].ProvenanceStatus != models.CandidateProvenanceVerified {
		t.Fatalf("first run persisted wrong lifecycle/provenance state: %+v", first.Candidates[0])
	}
	if first.Candidates[0].Fields["ruler"] != "Domitian" {
		t.Fatalf("first run did not preserve source-backed fields: %+v", first.Candidates[0].Fields)
	}
	var storedRun models.AlertRun
	if err := db.First(&storedRun, firstQueued.RunID).Error; err != nil {
		t.Fatalf("load stored run: %v", err)
	}
	if storedRun.CriteriaSnapshot == "" || storedRun.CompletedAt == nil {
		t.Fatalf("run missing audit metadata: %+v", storedRun)
	}
	secondQueued, err := svc.RunNow(alert.ID, 1, RunAlertInput{MaxCandidates: 20})
	if err != nil {
		t.Fatalf("queue second run: %v", err)
	}
	if err := svc.ProcessRun(secondQueued.RunID); err != nil {
		t.Fatalf("process second run: %v", err)
	}
	second, err := svc.GetRun(alert.ID, secondQueued.RunID, 1)
	if err != nil {
		t.Fatalf("load second run: %v", err)
	}
	if second.NewCount != 0 || second.DuplicateCount != 1 {
		t.Fatalf("duplicate was not suppressed: %+v", second)
	}
	var availabilityRuns, availabilityResults int64
	db.Model(&models.AvailabilityRun{}).Count(&availabilityRuns)
	db.Model(&models.AvailabilityResult{}).Count(&availabilityResults)
	if availabilityRuns != 0 || availabilityResults != 0 {
		t.Fatalf("alert run touched availability tables: runs=%d results=%d", availabilityRuns, availabilityResults)
	}
	var after models.Coin
	if err := db.First(&after, savedCoin.ID).Error; err != nil {
		t.Fatalf("reload saved coin: %v", err)
	}
	if after.ListingStatus != "available" || after.ListingCheckedAt != nil || after.ListingCheckReason != "seeded" {
		t.Fatalf("alert run mutated listing status fields: %+v", after)
	}
}

func TestWishlistSearchAlertService_RunNowEnqueuesWithoutWaitingForAgent(t *testing.T) {
	agentCalled := false
	svc, _, cleanup := setupWishlistSearchAlertDiscoveryService(t, func(w http.ResponseWriter, r *http.Request) {
		agentCalled = true
		t.Fatal("agent should not be called until a worker processes the queued run")
	})
	defer cleanup()
	alert, err := svc.CreateAlert(1, validAlertInput())
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	result, err := svc.RunNow(alert.ID, 1, RunAlertInput{})
	if err != nil {
		t.Fatalf("queue run: %v", err)
	}
	if result.Status != models.AlertRunStatusQueued || result.CompletedAt != nil {
		t.Fatalf("expected queued run response, got %+v", result)
	}
	if agentCalled {
		t.Fatal("agent was called synchronously during RunNow")
	}
}

func TestWishlistSearchAlertService_RunNowFiltersSoldAndOutOfRangeCandidates(t *testing.T) {
	agent := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates":[
				{
					"source_url":"https://www.vcoins.com/valid",
					"source_name":"VCoins",
					"title":"Aspendos AR Stater",
					"observed_price":225.0,
					"observed_currency":"USD",
					"reason_for_match":"Title and price match the alert.",
					"last_seen_at":"2026-06-29T17:00:10Z",
					"provenance_status":"verified",
					"fields":{"material":"Silver"},
					"provenance":[{"field":"observed_price","value":"US$ 225.00","source_url":"https://www.vcoins.com/valid","observed_at":"2026-06-29T17:00:10Z","confidence":"high","verification_state":"verified"}]
				},
				{
					"source_url":"https://www.vcoins.com/sold",
					"source_name":"VCoins",
					"title":"Aspendos AR Stater",
					"observed_price":225.0,
					"observed_currency":"USD",
					"reason_for_match":"Title and price match the alert.",
					"last_seen_at":"2026-06-29T17:00:10Z",
					"provenance_status":"verified",
					"fields":{"availability":"Sold"},
					"provenance":[{"field":"availability","value":"Sold","source_url":"https://www.vcoins.com/sold","observed_at":"2026-06-29T17:00:10Z","confidence":"high","verification_state":"verified"}]
				},
				{
					"source_url":"https://www.vcoins.com/too-expensive",
					"source_name":"VCoins",
					"title":"Aspendos AR Stater",
					"observed_price":680.0,
					"observed_currency":"USD",
					"reason_for_match":"Title matches the alert.",
					"last_seen_at":"2026-06-29T17:00:10Z",
					"provenance_status":"verified",
					"fields":{"material":"Silver"},
					"provenance":[{"field":"observed_price","value":"US$ 680.00","source_url":"https://www.vcoins.com/too-expensive","observed_at":"2026-06-29T17:00:10Z","confidence":"high","verification_state":"verified"}]
				},
				{
					"source_url":"https://www.vcoins.com/too-cheap",
					"source_name":"VCoins",
					"title":"Aspendos AR Stater",
					"observed_price":25.0,
					"observed_currency":"USD",
					"reason_for_match":"Title matches the alert.",
					"last_seen_at":"2026-06-29T17:00:10Z",
					"provenance_status":"verified",
					"fields":{"material":"Silver"},
					"provenance":[{"field":"observed_price","value":"US$ 25.00","source_url":"https://www.vcoins.com/too-cheap","observed_at":"2026-06-29T17:00:10Z","confidence":"high","verification_state":"verified"}]
				},
				{
					"source_url":"https://www.vcoins.com/no-price",
					"source_name":"VCoins",
					"title":"Aspendos AR Stater",
					"reason_for_match":"Title matches the alert.",
					"last_seen_at":"2026-06-29T17:00:10Z",
					"provenance_status":"verified",
					"fields":{"material":"Silver"},
					"provenance":[{"field":"source_url","value":"https://www.vcoins.com/no-price","source_url":"https://www.vcoins.com/no-price","observed_at":"2026-06-29T17:00:10Z","confidence":"high","verification_state":"verified"}]
				}
			],
			"warnings":[],
			"partial":false
		}`))
	}
	svc, _, cleanup := setupWishlistSearchAlertDiscoveryService(t, agent)
	defer cleanup()
	input := validAlertInput()
	input.Criteria.PriceMin = alertFloatPtr(50)
	input.Criteria.PriceMax = alertFloatPtr(350)
	alert, err := svc.CreateAlert(1, input)
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	queued, err := svc.RunNow(alert.ID, 1, RunAlertInput{MaxCandidates: 20})
	if err != nil {
		t.Fatalf("queue run: %v", err)
	}
	if err := svc.ProcessRun(queued.RunID); err != nil {
		t.Fatalf("process run: %v", err)
	}
	run, err := svc.GetRun(alert.ID, queued.RunID, 1)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run.Status != models.AlertRunStatusPartial || run.NewCount != 1 || run.ResultCount != 1 {
		t.Fatalf("expected only the valid candidate to persist as partial run, got %+v", run)
	}
	if len(run.Candidates) != 1 || run.Candidates[0].SourceURL != "https://www.vcoins.com/valid" {
		t.Fatalf("unexpected persisted candidates: %+v", run.Candidates)
	}
	if len(run.PartialWarnings) != 4 {
		t.Fatalf("expected four omission warnings, got %+v", run.PartialWarnings)
	}
}

func TestWishlistSearchAlertService_RunNowRejectsDisabledAndRunningAlert(t *testing.T) {
	svc, db, cleanup := setupWishlistSearchAlertDiscoveryService(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("agent should not be called")
	})
	defer cleanup()
	disabledInput := validAlertInput()
	active := false
	disabledInput.IsActive = &active
	disabled, err := svc.CreateAlert(1, disabledInput)
	if err != nil {
		t.Fatalf("create disabled alert: %v", err)
	}
	if _, err := svc.RunNow(disabled.ID, 1, RunAlertInput{}); !errors.Is(err, ErrWishlistSearchAlertDisabled) {
		t.Fatalf("disabled run error = %v", err)
	}
	alert, err := svc.CreateAlert(1, validAlertInput())
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	if err := db.Create(&models.AlertRun{
		AlertID: alert.ID, UserID: 1, TriggerType: models.AlertRunTriggerManual,
		Status: models.AlertRunStatusRunning, StartedAt: alert.CreatedAt, CriteriaSnapshot: "{}",
	}).Error; err != nil {
		t.Fatalf("seed running run: %v", err)
	}
	if _, err := svc.RunNow(alert.ID, 1, RunAlertInput{}); !errors.Is(err, ErrWishlistSearchAlertRunLimited) {
		t.Fatalf("running run error = %v", err)
	}
}

func TestWishlistSearchAlertService_RunNowKeepsRunningLockBeyondThirtySeconds(t *testing.T) {
	if wishlistAlertDiscoveryTimeout <= 30*time.Second {
		t.Fatalf("discovery timeout should allow normal agent runs beyond 30 seconds, got %s", wishlistAlertDiscoveryTimeout)
	}

	svc, db, cleanup := setupWishlistSearchAlertDiscoveryService(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("agent should not be called while a prior run is still locked")
	})
	defer cleanup()
	alert, err := svc.CreateAlert(1, validAlertInput())
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	if err := db.Create(&models.AlertRun{
		AlertID: alert.ID, UserID: 1, TriggerType: models.AlertRunTriggerManual,
		Status: models.AlertRunStatusRunning, StartedAt: time.Now().Add(-2 * time.Minute), CriteriaSnapshot: "{}",
	}).Error; err != nil {
		t.Fatalf("seed running run: %v", err)
	}
	if _, err := svc.RunNow(alert.ID, 1, RunAlertInput{}); !errors.Is(err, ErrWishlistSearchAlertRunLimited) {
		t.Fatalf("running run beyond 30 seconds error = %v", err)
	}
}

func TestWishlistSearchAlertService_DismissRestoreConvertAndDuplicateWarning(t *testing.T) {
	svc, db := setupWishlistSearchAlertService(t)
	alert, err := svc.CreateAlert(1, validAlertInput())
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	run := &models.AlertRun{AlertID: alert.ID, UserID: 1, TriggerType: models.AlertRunTriggerManual, Status: models.AlertRunStatusCompleted, StartedAt: models.WishlistSearchAlert{}.CreatedAt, CriteriaSnapshot: "{}"}
	run.StartedAt = alert.CreatedAt
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	candidate := &models.AlertCandidate{
		UserID: 1, AlertID: alert.ID, RunID: run.ID, SourceURL: "https://dealer.example/item/1",
		CanonicalSourceURL: "https://dealer.example/item/1", Title: "Domitian Denarius", NormalizedTitle: "domitian denarius",
		ReasonForMatch: "matches", LastSeenAt: alert.CreatedAt, FirstSeenAt: alert.CreatedAt,
		ProvenanceStatus: models.CandidateProvenanceVerified, LifecycleState: models.AlertCandidateStateActive,
		DuplicateKey: DuplicateKey(alert.ID, "https://dealer.example/item/1", "domitian denarius", nil, "USD"),
	}
	if err := db.Create(candidate).Error; err != nil {
		t.Fatalf("seed candidate: %v", err)
	}
	dismissed, err := svc.DismissCandidate(alert.ID, candidate.ID, 1, DismissCandidateInput{Reason: "duplicate"})
	if err != nil || dismissed.LifecycleState != models.AlertCandidateStateDismissed {
		t.Fatalf("dismiss got candidate=%+v err=%v", dismissed, err)
	}
	restored, err := svc.RestoreCandidate(alert.ID, candidate.ID, 1)
	if err != nil || restored.LifecycleState != models.AlertCandidateStateActive {
		t.Fatalf("restore got candidate=%+v err=%v", restored, err)
	}
	if err := db.Create(&models.Coin{UserID: 1, Name: "Existing", IsWishlist: true, ReferenceURL: candidate.SourceURL}).Error; err != nil {
		t.Fatalf("seed duplicate wishlist: %v", err)
	}
	input := ConvertCandidateInput{Coin: models.Coin{Name: "Domitian Denarius", Category: models.CategoryRoman, Era: models.EraAncient, ReferenceURL: candidate.SourceURL}}
	result, err := svc.ConvertCandidate(alert.ID, candidate.ID, 1, input)
	if !errors.Is(err, ErrWishlistSearchAlertDuplicate) || result == nil || len(result.Warnings) == 0 {
		t.Fatalf("expected duplicate warning, result=%+v err=%v", result, err)
	}
	var preAckCoinCount int64
	if err := db.Model(&models.Coin{}).Where("user_id = ? AND is_wishlist = ?", 1, true).Count(&preAckCoinCount).Error; err != nil {
		t.Fatalf("count wishlist before duplicate acknowledgement: %v", err)
	}
	if preAckCoinCount != 1 {
		t.Fatalf("duplicate warning created a wishlist item before acknowledgement; count=%d", preAckCoinCount)
	}
	var preAckCandidate models.AlertCandidate
	if err := db.First(&preAckCandidate, candidate.ID).Error; err != nil {
		t.Fatalf("reload candidate before duplicate acknowledgement: %v", err)
	}
	if preAckCandidate.LifecycleState != models.AlertCandidateStateActive || preAckCandidate.ConvertedCoinID != nil {
		t.Fatalf("duplicate warning mutated candidate before acknowledgement: %+v", preAckCandidate)
	}
	otherUserLocation := models.StorageLocation{UserID: 2, Name: "Other user's tray"}
	if err := db.Create(&otherUserLocation).Error; err != nil {
		t.Fatalf("seed other user storage location: %v", err)
	}
	input.Coin.StorageLocationID = &otherUserLocation.ID
	input.AcknowledgeDuplicateWarning = true
	if _, err := svc.ConvertCandidate(alert.ID, candidate.ID, 1, input); !errors.Is(err, ErrStorageLocationNotFound) {
		t.Fatalf("cross-owner storage location conversion error = %v, want %v", err, ErrStorageLocationNotFound)
	}
	var invalidStorageCandidate models.AlertCandidate
	if err := db.First(&invalidStorageCandidate, candidate.ID).Error; err != nil {
		t.Fatalf("reload candidate after invalid storage location: %v", err)
	}
	if invalidStorageCandidate.LifecycleState != models.AlertCandidateStateActive || invalidStorageCandidate.ConvertedCoinID != nil {
		t.Fatalf("invalid storage location mutated candidate: %+v", invalidStorageCandidate)
	}
	input.Coin.StorageLocationID = nil
	input.AcknowledgeDuplicateWarning = true
	result, err = svc.ConvertCandidate(alert.ID, candidate.ID, 1, input)
	if err != nil {
		t.Fatalf("acknowledged convert: %v", err)
	}
	if !result.Coin.IsWishlist || result.Coin.SourceAlertCandidateID == nil || *result.Coin.SourceAlertCandidateID != candidate.ID {
		t.Fatalf("converted coin missing traceability: %+v", result.Coin)
	}
	var converted models.AlertCandidate
	if err := db.First(&converted, candidate.ID).Error; err != nil {
		t.Fatalf("reload converted candidate: %v", err)
	}
	if converted.LifecycleState != models.AlertCandidateStateConverted || converted.ConvertedCoinID == nil {
		t.Fatalf("candidate was not marked converted after acknowledged save: %+v", converted)
	}
	if _, err := svc.RestoreCandidate(alert.ID, candidate.ID, 1); !errors.Is(err, ErrWishlistSearchAlertCandidateState) {
		t.Fatalf("converted candidate restore error = %v, want %v", err, ErrWishlistSearchAlertCandidateState)
	}
}
func TestWishlistSearchAlertService_ValidationAndNoAvailabilitySideEffects(t *testing.T) {
	svc, db := setupWishlistSearchAlertService(t)
	if err := db.Create(&models.Coin{UserID: 1, Name: "Saved wishlist coin", IsWishlist: true, ListingStatus: "available"}).Error; err != nil {
		t.Fatalf("seed coin: %v", err)
	}

	tests := []struct {
		name string
		edit func(*WishlistSearchAlertInput)
		want error
	}{
		{name: "empty criteria", edit: func(in *WishlistSearchAlertInput) { in.Criteria = WishlistAlertCriteriaInput{Currency: "USD"} }, want: ErrWishlistSearchAlertNoCriteria},
		{name: "bad price range", edit: func(in *WishlistSearchAlertInput) {
			in.Criteria.PriceMin = alertFloatPtr(400)
			in.Criteria.PriceMax = alertFloatPtr(300)
		}, want: ErrWishlistSearchAlertPriceRange},
		{name: "bad date range", edit: func(in *WishlistSearchAlertInput) {
			in.Criteria.DateFrom = alertIntPtr(100)
			in.Criteria.DateTo = alertIntPtr(90)
		}, want: ErrWishlistSearchAlertDateRange},
		{name: "bad cadence", edit: func(in *WishlistSearchAlertInput) { in.Cadence = "hourly" }, want: ErrWishlistSearchAlertCadence},
		{name: "bad source", edit: func(in *WishlistSearchAlertInput) { in.Criteria.SourceFilters = []string{"localhost"} }, want: ErrWishlistSearchAlertSourceFilter},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := validAlertInput()
			tc.edit(&in)
			if _, err := svc.CreateAlert(1, in); !errors.Is(err, tc.want) {
				t.Fatalf("create error = %v, want %v", err, tc.want)
			}
		})
	}
	if _, err := svc.CreateAlert(1, validAlertInput()); err != nil {
		t.Fatalf("valid create: %v", err)
	}
	var coinCount, runCount, resultCount int64
	db.Model(&models.Coin{}).Count(&coinCount)
	db.Model(&models.AvailabilityRun{}).Count(&runCount)
	db.Model(&models.AvailabilityResult{}).Count(&resultCount)
	if coinCount != 1 || runCount != 0 || resultCount != 0 {
		t.Fatalf("alert CRUD side effects: coins=%d runs=%d results=%d", coinCount, runCount, resultCount)
	}
	var coin models.Coin
	db.First(&coin)
	if coin.ListingStatus != "available" {
		t.Fatalf("listing status mutated: %q", coin.ListingStatus)
	}
}

// TestConvertCandidate_CoinWithNonZeroReferenceIDsDoesNotConflict proves the
// wishlist invariant on the exact agent conversion path:
//   - ConvertCandidateInput.Coin carries References with non-zero IDs from
//     existing rows (as the agent might send from source/database data).
//   - Conversion must succeed with HTTP 201.
//   - The converted wishlist coin must have ZERO persisted reference rows.
//   - The original reference row must remain untouched.
func TestConvertCandidate_CoinWithNonZeroReferenceIDsDoesNotConflict(t *testing.T) {
	svc, db := setupWishlistSearchAlertService(t)

	// Seed an existing reference row so we have a real occupied primary key.
	existingCoin := models.Coin{UserID: 1, Name: "Existing Coin", IsWishlist: false}
	if err := db.Create(&existingCoin).Error; err != nil {
		t.Fatalf("seed existing coin: %v", err)
	}
	existingRef := models.CoinReference{CoinID: existingCoin.ID, Catalog: "RIC", Number: "1"}
	if err := db.Create(&existingRef).Error; err != nil {
		t.Fatalf("seed existing ref: %v", err)
	}
	if existingRef.ID == 0 {
		t.Fatal("seed produced ID=0; pre-condition broken")
	}

	alert, err := svc.CreateAlert(1, validAlertInput())
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	run := &models.AlertRun{
		AlertID: alert.ID, UserID: 1,
		TriggerType: models.AlertRunTriggerManual, Status: models.AlertRunStatusCompleted,
		StartedAt: alert.CreatedAt, CriteriaSnapshot: "{}",
	}
	if err := db.Create(run).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	candidate := &models.AlertCandidate{
		UserID: 1, AlertID: alert.ID, RunID: run.ID,
		SourceURL:          "https://dealer.example/ref-id-regression",
		CanonicalSourceURL: "https://dealer.example/ref-id-regression",
		Title:              "Domitian Denarius RIC 1", NormalizedTitle: "domitian denarius ric 1",
		ReasonForMatch: "regression test", LastSeenAt: alert.CreatedAt, FirstSeenAt: alert.CreatedAt,
		ProvenanceStatus: models.CandidateProvenanceVerified, LifecycleState: models.AlertCandidateStateActive,
		DuplicateKey: DuplicateKey(alert.ID, "https://dealer.example/ref-id-regression", "domitian denarius ric 1", nil, "USD"),
	}
	if err := db.Create(candidate).Error; err != nil {
		t.Fatalf("seed candidate: %v", err)
	}

	// Build the input the way the agent/frontend would: coin includes References
	// with IDs that came from the source/database (e.g. looked up from a prior
	// similar coin). This is the exact payload that caused the regression.
	input := ConvertCandidateInput{
		Coin: models.Coin{
			Name:     "Domitian Denarius RIC 1",
			Category: models.CategoryRoman,
			Era:      models.EraAncient,
			References: []models.CoinReference{
				{ID: existingRef.ID, Catalog: "RIC", Number: "1"},
			},
		},
	}

	result, err := svc.ConvertCandidate(alert.ID, candidate.ID, 1, input)
	if err != nil {
		t.Fatalf("ConvertCandidate with non-zero reference ID failed: %v", err)
	}
	if !result.Coin.IsWishlist {
		t.Error("converted coin should be a wishlist coin")
	}

	// Wishlist coins discard references — no coin_references rows should exist
	// for the newly created coin.
	var refCount int64
	if err := db.Model(&models.CoinReference{}).Where("coin_id = ?", result.Coin.ID).Count(&refCount).Error; err != nil {
		t.Fatalf("count converted coin references: %v", err)
	}
	if refCount != 0 {
		t.Errorf("expected 0 references on wishlist coin, got %d", refCount)
	}

	// The existing reference row must remain untouched and still owned by the
	// original coin — it must not have been reassigned or deleted.
	var unchanged models.CoinReference
	if err := db.First(&unchanged, existingRef.ID).Error; err != nil {
		t.Fatalf("existing reference missing after conversion: %v", err)
	}
	if unchanged.CoinID != existingCoin.ID {
		t.Errorf("existing reference CoinID mutated to %d, want %d", unchanged.CoinID, existingCoin.ID)
	}
}
