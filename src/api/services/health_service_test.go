package services

import (
	"strings"
	"testing"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupHealthServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	err = db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.CoinImage{},
		&models.CollectionHealthSnapshot{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func setupHealthService(t *testing.T, db *gorm.DB) *HealthService {
	t.Helper()
	repo := repository.NewHealthRepository(db)
	logger := NewLogger(100)
	return NewHealthService(repo, logger)
}

// --- T011: Scoring unit tests for weights, grade thresholds, freshness buckets, and empty collections ---

func TestHealthGradeFromScore_GradeThresholds(t *testing.T) {
	tests := []struct {
		score int
		want  HealthGrade
	}{
		{100, HealthGradeA},
		{95, HealthGradeA},
		{90, HealthGradeA},
		{89, HealthGradeB},
		{85, HealthGradeB},
		{80, HealthGradeB},
		{79, HealthGradeC},
		{75, HealthGradeC},
		{70, HealthGradeC},
		{69, HealthGradeD},
		{65, HealthGradeD},
		{60, HealthGradeD},
		{59, HealthGradeF},
		{30, HealthGradeF},
		{0, HealthGradeF},
	}

	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			got := HealthGradeFromScore(tt.score)
			if got != tt.want {
				t.Errorf("HealthGradeFromScore(%d) = %s, want %s", tt.score, got, tt.want)
			}
		})
	}
}

func TestClampHealthScore_BoundaryConditions(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{-10, 0},
		{-1, 0},
		{0, 0},
		{50, 50},
		{100, 100},
		{101, 100},
		{150, 100},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := ClampHealthScore(tt.input)
			if got != tt.want {
				t.Errorf("ClampHealthScore(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultHealthWeights_FixedValues(t *testing.T) {
	weights := DefaultHealthWeights()

	if weights.Metadata != 40 {
		t.Errorf("Metadata weight = %d, want 40", weights.Metadata)
	}
	if weights.ImageCoverage != 20 {
		t.Errorf("ImageCoverage weight = %d, want 20", weights.ImageCoverage)
	}
	if weights.ValuationFreshness != 20 {
		t.Errorf("ValuationFreshness weight = %d, want 20", weights.ValuationFreshness)
	}
	if weights.AICoverage != 20 {
		t.Errorf("AICoverage weight = %d, want 20", weights.AICoverage)
	}

	total := weights.Metadata + weights.ImageCoverage + weights.ValuationFreshness + weights.AICoverage
	if total != 100 {
		t.Errorf("total weights = %d, want 100", total)
	}
}

func TestGetCollectionHealthSummary_EmptyCollection(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	user := models.User{Username: "empty", Email: "empty@test.com"}
	db.Create(&user)

	summary, err := svc.GetCollectionHealthSummary(user.ID)
	if err != nil {
		t.Fatalf("GetCollectionHealthSummary failed: %v", err)
	}

	if summary.Score != 0 {
		t.Errorf("expected score=0 for empty collection, got %d", summary.Score)
	}
	if summary.Grade != HealthGradeF {
		t.Errorf("expected grade=F for empty collection, got %s", summary.Grade)
	}
	if summary.EligibleCoinCount != 0 {
		t.Errorf("expected eligibleCoinCount=0, got %d", summary.EligibleCoinCount)
	}
	if summary.Trend30D.Status != "unavailable" {
		t.Errorf("expected trend status=unavailable, got %s", summary.Trend30D.Status)
	}
}

func TestGetCollectionHealthSummary_WithActiveCoins(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	user := models.User{Username: "collector", Email: "collector@test.com"}
	db.Create(&user)

	// Create active coins
	for i := 0; i < 3; i++ {
		db.Create(&models.Coin{
			Name:       "Test Coin",
			Category:   models.CategoryRoman,
			Material:   models.MaterialSilver,
			UserID:     user.ID,
			IsWishlist: false,
			IsSold:     false,
		})
	}

	summary, err := svc.GetCollectionHealthSummary(user.ID)
	if err != nil {
		t.Fatalf("GetCollectionHealthSummary failed: %v", err)
	}

	if summary.EligibleCoinCount != 3 {
		t.Errorf("expected eligibleCoinCount=3, got %d", summary.EligibleCoinCount)
	}
	if summary.Weights.Metadata != 40 {
		t.Errorf("expected metadata weight=40, got %d", summary.Weights.Metadata)
	}
}

// --- T015: Trend-direction and delta unit tests ---

func TestGetCollectionHealthSummary_TrendUnavailableNoSnapshot(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	user := models.User{Username: "newuser", Email: "new@test.com"}
	db.Create(&user)

	db.Create(&models.Coin{
		Name:     "Coin",
		Category: models.CategoryRoman,
		UserID:   user.ID,
	})

	summary, err := svc.GetCollectionHealthSummary(user.ID)
	if err != nil {
		t.Fatalf("GetCollectionHealthSummary failed: %v", err)
	}

	if summary.Trend30D.Status != "unavailable" {
		t.Errorf("expected trend status=unavailable when no baseline, got %s", summary.Trend30D.Status)
	}
	if summary.Trend30D.Delta != nil {
		t.Errorf("expected nil delta, got %v", summary.Trend30D.Delta)
	}
	if summary.Trend30D.Direction != "unavailable" {
		t.Errorf("expected direction=unavailable, got %s", summary.Trend30D.Direction)
	}
}

func TestSnapshotUserHealth_PersistsSnapshot(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	user := models.User{Username: "snapper", Email: "snap@test.com"}
	db.Create(&user)

	db.Create(&models.Coin{
		Name:     "Coin",
		Category: models.CategoryRoman,
		UserID:   user.ID,
	})

	snapshotDate := time.Now().Truncate(24 * time.Hour)
	err := svc.SnapshotUserHealth(user.ID, snapshotDate)
	if err != nil {
		t.Fatalf("SnapshotUserHealth failed: %v", err)
	}

	var snapshot models.CollectionHealthSnapshot
	db.Where("user_id = ? AND snapshot_date = ?", user.ID, snapshotDate).First(&snapshot)
	if snapshot.ID == 0 {
		t.Fatal("expected snapshot to be persisted")
	}
	if snapshot.EligibleCoinCount != 1 {
		t.Errorf("expected eligibleCoinCount=1, got %d", snapshot.EligibleCoinCount)
	}
}

// --- T028: Service tests for missing checklist key mapping and quick-action hints ---

func TestListCoinHealth_EmptyCollection(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	user := models.User{Username: "empty", Email: "empty@test.com"}
	db.Create(&user)

	list, err := svc.ListCoinHealth(user.ID, 1, 25, "all")
	if err != nil {
		t.Fatalf("ListCoinHealth failed: %v", err)
	}

	if len(list.Coins) != 0 {
		t.Errorf("expected 0 coins, got %d", len(list.Coins))
	}
	if list.Pagination.Total != 0 {
		t.Errorf("expected total=0, got %d", list.Pagination.Total)
	}
}

func TestListCoinHealth_WithCoins(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	user := models.User{Username: "collector", Email: "collector@test.com"}
	db.Create(&user)

	for i := 0; i < 5; i++ {
		db.Create(&models.Coin{
			Name:     "Test Coin",
			Category: models.CategoryRoman,
			UserID:   user.ID,
		})
	}

	list, err := svc.ListCoinHealth(user.ID, 1, 25, "all")
	if err != nil {
		t.Fatalf("ListCoinHealth failed: %v", err)
	}

	if len(list.Coins) != 5 {
		t.Errorf("expected 5 coins, got %d", len(list.Coins))
	}
	if list.Pagination.Total != 5 {
		t.Errorf("expected total=5, got %d", list.Pagination.Total)
	}
	if list.Pagination.Page != 1 {
		t.Errorf("expected page=1, got %d", list.Pagination.Page)
	}
	if list.Pagination.Limit != 25 {
		t.Errorf("expected limit=25, got %d", list.Pagination.Limit)
	}
}

func TestListCoinHealth_PaginationRespected(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	user := models.User{Username: "collector", Email: "collector@test.com"}
	db.Create(&user)

	for i := 0; i < 30; i++ {
		db.Create(&models.Coin{
			Name:     "Test Coin",
			Category: models.CategoryRoman,
			UserID:   user.ID,
		})
	}

	list, err := svc.ListCoinHealth(user.ID, 1, 10, "all")
	if err != nil {
		t.Fatalf("ListCoinHealth failed: %v", err)
	}

	if len(list.Coins) != 10 {
		t.Errorf("expected 10 coins on page 1, got %d", len(list.Coins))
	}
	if list.Pagination.Total != 30 {
		t.Errorf("expected total=30, got %d", list.Pagination.Total)
	}

	list, err = svc.ListCoinHealth(user.ID, 2, 10, "all")
	if err != nil {
		t.Fatalf("ListCoinHealth page 2 failed: %v", err)
	}

	if len(list.Coins) != 10 {
		t.Errorf("expected 10 coins on page 2, got %d", len(list.Coins))
	}
}

// --- T040: Admin aggregate metric unit tests ---

func TestGetAdminHealthSummary_EmptySystem(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	summary, err := svc.GetAdminHealthSummary()
	if err != nil {
		t.Fatalf("GetAdminHealthSummary failed: %v", err)
	}

	if summary.MedianScore != 0 {
		t.Errorf("expected medianScore=0, got %d", summary.MedianScore)
	}
	if summary.LowScorePercentage != 0 {
		t.Errorf("expected lowScorePercentage=0, got %f", summary.LowScorePercentage)
	}
	if summary.EligibleCoinCount != 0 {
		t.Errorf("expected eligibleCoinCount=0, got %d", summary.EligibleCoinCount)
	}
	if summary.LowScoreThreshold != HealthLowScoreThreshold {
		t.Errorf("expected lowScoreThreshold=%d, got %d", HealthLowScoreThreshold, summary.LowScoreThreshold)
	}
}

func TestGetAdminHealthSummary_WithCoins(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	user := models.User{Username: "admin_test", Email: "admin@test.com"}
	db.Create(&user)

	for i := 0; i < 3; i++ {
		db.Create(&models.Coin{
			Name:     "Test Coin",
			Category: models.CategoryRoman,
			UserID:   user.ID,
		})
	}

	summary, err := svc.GetAdminHealthSummary()
	if err != nil {
		t.Fatalf("GetAdminHealthSummary failed: %v", err)
	}

	// Note: stub implementation returns 0, but structure is validated
	if summary.LowScoreThreshold != HealthLowScoreThreshold {
		t.Errorf("expected lowScoreThreshold=%d, got %d", HealthLowScoreThreshold, summary.LowScoreThreshold)
	}
	if summary.TopMissingFields == nil {
		t.Error("expected TopMissingFields to be initialized")
	}
}

// --- Fixture tests for deterministic test data ---

func TestFixtureHealthWeights_MatchesDefaults(t *testing.T) {
	fixture := fixtureHealthWeights()
	defaults := DefaultHealthWeights()

	if fixture.Metadata != defaults.Metadata {
		t.Errorf("fixture metadata = %d, defaults = %d", fixture.Metadata, defaults.Metadata)
	}
	if fixture.ImageCoverage != defaults.ImageCoverage {
		t.Errorf("fixture imageCoverage = %d, defaults = %d", fixture.ImageCoverage, defaults.ImageCoverage)
	}
	if fixture.ValuationFreshness != defaults.ValuationFreshness {
		t.Errorf("fixture valuationFreshness = %d, defaults = %d", fixture.ValuationFreshness, defaults.ValuationFreshness)
	}
	if fixture.AICoverage != defaults.AICoverage {
		t.Errorf("fixture aiCoverage = %d, defaults = %d", fixture.AICoverage, defaults.AICoverage)
	}
}

func TestFixtureCollectionHealthSummary_ValidStructure(t *testing.T) {
	delta := 5
	summary := fixtureCollectionHealthSummary(75, &delta)

	if summary.Score != 75 {
		t.Errorf("expected score=75, got %d", summary.Score)
	}
	if summary.Grade != HealthGradeC {
		t.Errorf("expected grade=C, got %s", summary.Grade)
	}
	if summary.EligibleCoinCount != 3 {
		t.Errorf("expected eligibleCoinCount=3, got %d", summary.EligibleCoinCount)
	}
	if summary.Trend30D.Status != "available" {
		t.Errorf("expected trend status=available, got %s", summary.Trend30D.Status)
	}
	if summary.Trend30D.Delta == nil || *summary.Trend30D.Delta != 5 {
		t.Errorf("expected delta=5, got %v", summary.Trend30D.Delta)
	}
}

func TestFixtureCoinHealthItem_ValidStructure(t *testing.T) {
	item := fixtureCoinHealthItem(10, 82)

	if item.CoinID != 10 {
		t.Errorf("expected coinID=10, got %d", item.CoinID)
	}
	if item.Score != 82 {
		t.Errorf("expected score=82, got %d", item.Score)
	}
	if item.Grade != HealthGradeB {
		t.Errorf("expected grade=B, got %s", item.Grade)
	}
	if len(item.MissingItems) != 1 {
		t.Errorf("expected 1 missing item, got %d", len(item.MissingItems))
	}
	if item.MissingItems[0].Dimension != ChecklistDimensionMetadata {
		t.Errorf("expected dimension=metadata, got %s", item.MissingItems[0].Dimension)
	}
	if len(item.QuickActions) != 1 {
		t.Errorf("expected 1 quick action, got %d", len(item.QuickActions))
	}
	if item.QuickActions[0] != HealthActionEditMetadata {
		t.Errorf("expected action=edit_metadata, got %s", item.QuickActions[0])
	}
}

// --- T012: Valuation freshness scoring with CurrentValueUpdatedAt ---

func TestScoreCoinValuationFreshness_WithCurrentValueUpdatedAt(t *testing.T) {
	svc := &HealthService{}
	now := time.Now()

	tests := []struct {
		name              string
		currentValue      *float64
		updatedAt         *time.Time
		purchaseDate      *time.Time
		expectedScore     int
		expectedChecklist bool // should valuation.freshness appear?
	}{
		{
			name:              "Recent valuation (today) scores 100, no checklist item",
			currentValue:      ptrFloat(100.0),
			updatedAt:         ptrTime(now),
			purchaseDate:      ptrTime(now.AddDate(-1, 0, 0)), // old purchase
			expectedScore:     100,
			expectedChecklist: false,
		},
		{
			name:              "Valuation 20 days old scores 100",
			currentValue:      ptrFloat(100.0),
			updatedAt:         ptrTime(now.AddDate(0, 0, -20)),
			purchaseDate:      ptrTime(now.AddDate(-1, 0, 0)),
			expectedScore:     100,
			expectedChecklist: false,
		},
		{
			name:              "Valuation 60 days old scores 80",
			currentValue:      ptrFloat(100.0),
			updatedAt:         ptrTime(now.AddDate(0, 0, -60)),
			purchaseDate:      ptrTime(now.AddDate(-1, 0, 0)),
			expectedScore:     80,
			expectedChecklist: false,
		},
		{
			name:              "Valuation 120 days old scores 60",
			currentValue:      ptrFloat(100.0),
			updatedAt:         ptrTime(now.AddDate(0, 0, -120)),
			purchaseDate:      ptrTime(now.AddDate(-1, 0, 0)),
			expectedScore:     60,
			expectedChecklist: false,
		},
		{
			name:              "Valuation 200 days old scores 35",
			currentValue:      ptrFloat(100.0),
			updatedAt:         ptrTime(now.AddDate(0, 0, -200)),
			purchaseDate:      ptrTime(now.AddDate(-1, 0, 0)),
			expectedScore:     35,
			expectedChecklist: true, // >180 days triggers checklist
		},
		{
			name:              "Valuation 400 days old scores 0",
			currentValue:      ptrFloat(100.0),
			updatedAt:         ptrTime(now.AddDate(0, 0, -400)),
			purchaseDate:      ptrTime(now.AddDate(-1, 0, 0)),
			expectedScore:     0,
			expectedChecklist: true,
		},
		{
			name:              "Nil CurrentValueUpdatedAt falls back to PurchaseDate (legacy behavior)",
			currentValue:      ptrFloat(100.0),
			updatedAt:         nil,
			purchaseDate:      ptrTime(now.AddDate(0, 0, -60)),
			expectedScore:     80,
			expectedChecklist: false,
		},
		{
			name:              "Nil CurrentValueUpdatedAt with old PurchaseDate scores 35",
			currentValue:      ptrFloat(100.0),
			updatedAt:         nil,
			purchaseDate:      ptrTime(now.AddDate(0, 0, -200)),
			expectedScore:     35,
			expectedChecklist: true,
		},
		{
			name:              "No CurrentValue scores 0",
			currentValue:      nil,
			updatedAt:         ptrTime(now),
			purchaseDate:      ptrTime(now),
			expectedScore:     0,
			expectedChecklist: false, // different checklist item: valuation.value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coin := &repository.EligibleCoinRow{
				CoinID:                1,
				CurrentValue:          tt.currentValue,
				CurrentValueUpdatedAt: tt.updatedAt,
				PurchaseDate:          tt.purchaseDate,
			}

			score := svc.scoreCoinValuationFreshness(coin)
			if score != tt.expectedScore {
				t.Errorf("score = %d, want %d", score, tt.expectedScore)
			}

			// Check checklist generation
			checklist := svc.generateCoinChecklist(coin)
			hasValuationFreshnessItem := false
			for _, item := range checklist {
				if item.Key == "valuation.freshness" {
					hasValuationFreshnessItem = true
					break
				}
			}
			if hasValuationFreshnessItem != tt.expectedChecklist {
				t.Errorf("valuation.freshness in checklist = %v, want %v", hasValuationFreshnessItem, tt.expectedChecklist)
			}
		})
	}
}

// --- AI Coverage Scoring and Checklist Tests ---

// Combined (legacy) AIAnalysis is NOT counted toward coverage; only per-side
// obverse/reverse analysis matters.
func TestScoreCoinAICoverage_CombinedAnalysisOnly(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	coin := &repository.EligibleCoinRow{
		AIAnalysis:      "Full combined analysis of both obverse and reverse",
		ObverseAnalysis: "",
		ReverseAnalysis: "",
	}

	score := svc.scoreCoinAICoverage(coin)
	if score != 0 {
		t.Errorf("combined analysis only: score = %d, want 0 (combined not counted)", score)
	}

	checklist := svc.generateCoinChecklist(coin)
	hasAnalysisItem := false
	for _, item := range checklist {
		if item.Key == "ai.analysis" {
			hasAnalysisItem = true
		}
		if item.Key == "ai.coverage" {
			t.Errorf("combined analysis only: should have ai.analysis, not ai.coverage")
		}
	}
	if !hasAnalysisItem {
		t.Errorf("combined analysis only: expected ai.analysis checklist item")
	}
}

func TestScoreCoinAICoverage_BothPerSideOnly(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	coin := &repository.EligibleCoinRow{
		AIAnalysis:      "",
		ObverseAnalysis: "Obverse shows portrait",
		ReverseAnalysis: "Reverse shows eagle",
	}

	score := svc.scoreCoinAICoverage(coin)
	if score != 100 {
		t.Errorf("both per-side only: score = %d, want 100", score)
	}

	checklist := svc.generateCoinChecklist(coin)
	for _, item := range checklist {
		if item.Key == "ai.analysis" || item.Key == "ai.coverage" {
			t.Errorf("both per-side only: unexpected checklist item %s", item.Key)
		}
	}
}

func TestScoreCoinAICoverage_OnlyObverseNoReverse(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	coin := &repository.EligibleCoinRow{
		AIAnalysis:      "",
		ObverseAnalysis: "Obverse shows portrait",
		ReverseAnalysis: "",
	}

	score := svc.scoreCoinAICoverage(coin)
	if score != 50 {
		t.Errorf("obverse only: score = %d, want 50", score)
	}

	checklist := svc.generateCoinChecklist(coin)
	hasCoverageItem := false
	for _, item := range checklist {
		if item.Key == "ai.coverage" {
			hasCoverageItem = true
		}
		if item.Key == "ai.analysis" {
			t.Errorf("obverse only: should not have ai.analysis item")
		}
	}
	if !hasCoverageItem {
		t.Errorf("obverse only: expected ai.coverage checklist item")
	}
}

func TestScoreCoinAICoverage_NoAnalysisAtAll(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	coin := &repository.EligibleCoinRow{
		AIAnalysis:      "",
		ObverseAnalysis: "",
		ReverseAnalysis: "",
	}

	score := svc.scoreCoinAICoverage(coin)
	if score != 0 {
		t.Errorf("no analysis: score = %d, want 0", score)
	}

	checklist := svc.generateCoinChecklist(coin)
	hasAnalysisItem := false
	for _, item := range checklist {
		if item.Key == "ai.analysis" {
			hasAnalysisItem = true
		}
		if item.Key == "ai.coverage" {
			t.Errorf("no analysis: should have ai.analysis, not ai.coverage")
		}
	}
	if !hasAnalysisItem {
		t.Errorf("no analysis: expected ai.analysis checklist item")
	}
}

// Combined analysis plus one per-side: combined ignored, so this scores as
// "one side only" (obverse done, reverse missing).
func TestScoreCoinAICoverage_CombinedPlusOneSide(t *testing.T) {
	db := setupHealthServiceTestDB(t)
	svc := setupHealthService(t, db)

	coin := &repository.EligibleCoinRow{
		AIAnalysis:      "Full combined analysis",
		ObverseAnalysis: "Additional obverse detail",
		ReverseAnalysis: "",
	}

	score := svc.scoreCoinAICoverage(coin)
	if score != 50 {
		t.Errorf("combined + obverse only: score = %d, want 50", score)
	}

	checklist := svc.generateCoinChecklist(coin)
	hasCoverageItem := false
	for _, item := range checklist {
		if item.Key == "ai.coverage" {
			hasCoverageItem = true
			if !strings.Contains(item.Label, "reverse") {
				t.Errorf("combined + obverse only: label should name missing reverse side, got %q", item.Label)
			}
		}
		if item.Key == "ai.analysis" {
			t.Errorf("combined + obverse only: should not have ai.analysis item")
		}
	}
	if !hasCoverageItem {
		t.Errorf("combined + obverse only: expected ai.coverage checklist item")
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
