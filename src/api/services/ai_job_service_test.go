package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type fakeAIJobAgent struct {
	analysisResponse  string
	gradingResponse   string
	valuationResponse string
}

func (f fakeAIJobAgent) AnalyzeCoin(ctx context.Context, req AnalyzeProxyRequest) (string, error) {
	if f.analysisResponse == "" {
		return "", errors.New("analysis not configured")
	}
	return f.analysisResponse, nil
}

func (f fakeAIJobAgent) GradeCoin(ctx context.Context, req GradeProxyRequest) (string, error) {
	if f.gradingResponse == "" {
		return "", errors.New("grading not configured")
	}
	return f.gradingResponse, nil
}

func (f fakeAIJobAgent) CollectPortfolioReview(ctx context.Context, req PortfolioReviewProxyRequest) (string, error) {
	if f.valuationResponse == "" {
		return "", errors.New("valuation not configured")
	}
	return f.valuationResponse, nil
}

func newAIJobServiceTestDB(t *testing.T) (*gorm.DB, *AIJobService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Coin{}, &models.CoinImage{}, &models.AppSetting{}, &models.AIJob{}, &models.CoinJournal{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	db.Create(&models.AppSetting{Key: SettingAIProvider, Value: "ollama"})
	db.Create(&models.AppSetting{Key: SettingOllamaModel, Value: "llava"})
	db.Create(&models.User{Username: "tester", Email: "tester@example.com", PasswordHash: "x"})

	settingsSvc := NewSettingsService(repository.NewSettingsRepository(db))
	svc := NewAIJobService(
		repository.NewAIJobRepository(db),
		fakeAIJobAgent{
			analysisResponse:  "obverse analysis text",
			gradingResponse:   "**Estimated Grade: VF-20** (Confidence: Medium)",
			valuationResponse: "```json\n{\"estimatedValue\":321,\"confidence\":\"high\",\"reasoning\":\"Comparable listings support the estimate.\",\"comparables\":[{\"source\":\"VCoins\",\"price\":\"$321\",\"url\":\"https://example.com\"}]}\n```",
		},
		repository.NewUserRepository(db),
		settingsSvc,
		nil,
		NewLogger(10),
	)
	return db, svc
}

func createAIJobTestCoin(t *testing.T, db *gorm.DB, userID uint) models.Coin {
	t.Helper()
	coin := models.Coin{
		Name:         "Test Denarius",
		UserID:       userID,
		Category:     models.CategoryRoman,
		Era:          models.EraAncient,
		Material:     models.MaterialSilver,
		Denomination: "Denarius",
		Ruler:        "Trajan",
	}
	if err := db.Create(&coin).Error; err != nil {
		t.Fatalf("create coin: %v", err)
	}
	return coin
}

func addAIJobTestImage(t *testing.T, db *gorm.DB, coinID uint, side models.ImageType) {
	t.Helper()
	imageDir := filepath.Join("uploads", "ai-job-test")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("create image dir: %v", err)
	}
	imageName := string(side) + ".bin"
	imagePath := filepath.Join("ai-job-test", imageName)
	fullPath := filepath.Join("uploads", imagePath)
	if err := os.WriteFile(fullPath, []byte("image-bytes"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(fullPath)
		_ = os.Remove(imageDir)
	})
	if err := db.Create(&models.CoinImage{CoinID: coinID, FilePath: imagePath, ImageType: side}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
}

func TestAIJobServiceDuplicatePrevention(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	coin := createAIJobTestCoin(t, db, 1)
	addAIJobTestImage(t, db, coin.ID, models.ImageTypeObverse)

	first, created, err := svc.EnqueueAnalysis(1, coin.ID, "obverse")
	if err != nil || !created {
		t.Fatalf("first enqueue = job %v, created %v, err %v", first, created, err)
	}
	second, created, err := svc.EnqueueAnalysis(1, coin.ID, "obverse")
	if err != nil {
		t.Fatalf("second enqueue: %v", err)
	}
	if created {
		t.Fatal("expected duplicate enqueue to return existing job")
	}
	if first.ID != second.ID {
		t.Fatalf("expected same job id, got %d and %d", first.ID, second.ID)
	}
}

func TestAIJobServiceUserScopedJobLookup(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	coin := createAIJobTestCoin(t, db, 1)
	addAIJobTestImage(t, db, coin.ID, models.ImageTypeObverse)
	job, _, err := svc.EnqueueAnalysis(1, coin.ID, "obverse")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := svc.GetJob(2, job.ID); !repository.IsRecordNotFound(err) {
		t.Fatalf("expected other user lookup to be not found, got %v", err)
	}
}

func TestAIJobServiceAnalysisJobUpdatesCoinResult(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	coin := createAIJobTestCoin(t, db, 1)
	addAIJobTestImage(t, db, coin.ID, models.ImageTypeObverse)
	job, _, err := svc.EnqueueAnalysis(1, coin.ID, "obverse")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	svc.processJob(job.ID)

	var updated models.Coin
	if err := db.First(&updated, coin.ID).Error; err != nil {
		t.Fatalf("load coin: %v", err)
	}
	if updated.ObverseAnalysis != "obverse analysis text" {
		t.Fatalf("obverse analysis = %q", updated.ObverseAnalysis)
	}
	stored, err := svc.GetJob(1, job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if stored.Status != models.AIJobStatusCompleted || stored.Result == "" {
		t.Fatalf("job status/result = %s/%q", stored.Status, stored.Result)
	}
}

func TestAIJobServiceValueEstimateJournalsWithoutApplyingValue(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	coin := createAIJobTestCoin(t, db, 1)
	job, _, err := svc.EnqueueValueEstimate(1, coin.ID)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	svc.processJob(job.ID)

	var journal models.CoinJournal
	if err := db.Where("coin_id = ? AND user_id = ?", coin.ID, uint(1)).First(&journal).Error; err != nil {
		t.Fatalf("expected journal entry: %v", err)
	}
	if journal.Entry != "AI Value Estimate: $321.00 (high confidence)" {
		t.Fatalf("journal entry = %q", journal.Entry)
	}
	var updated models.Coin
	if err := db.First(&updated, coin.ID).Error; err != nil {
		t.Fatalf("load coin: %v", err)
	}
	if updated.CurrentValue != nil {
		t.Fatalf("current value was unexpectedly applied: %v", *updated.CurrentValue)
	}
	stored, err := svc.GetJob(1, job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if stored.Status != models.AIJobStatusCompleted || stored.Result == "" {
		t.Fatalf("job status/result = %s/%q", stored.Status, stored.Result)
	}
}

func TestAIJobServiceCoinGradingRequiresOwnedCoinWithImage(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	coin := createAIJobTestCoin(t, db, 1)

	if _, _, err := svc.EnqueueCoinGrading(1, coin.ID); !errors.Is(err, ErrAIJobNoImagesForGrading) {
		t.Fatalf("expected no-image grading error, got %v", err)
	}

	addAIJobTestImage(t, db, coin.ID, models.ImageTypeObverse)
	if _, _, err := svc.EnqueueCoinGrading(2, coin.ID); !repository.IsRecordNotFound(err) {
		t.Fatalf("expected owner-scoped lookup to hide another user's coin, got %v", err)
	}
}

func TestAIJobServiceCoinGradingStoresReportWithoutUpdatingCoinGrade(t *testing.T) {
	db, svc := newAIJobServiceTestDB(t)
	coin := createAIJobTestCoin(t, db, 1)
	coin.Grade = "F-12"
	if err := db.Save(&coin).Error; err != nil {
		t.Fatalf("save grade: %v", err)
	}
	addAIJobTestImage(t, db, coin.ID, models.ImageTypeObverse)

	job, _, err := svc.EnqueueCoinGrading(1, coin.ID)
	if err != nil {
		t.Fatalf("enqueue grading: %v", err)
	}
	svc.processJob(job.ID)

	var updated models.Coin
	if err := db.First(&updated, coin.ID).Error; err != nil {
		t.Fatalf("load coin: %v", err)
	}
	if updated.Grade != "F-12" {
		t.Fatalf("coin grade was unexpectedly updated: %q", updated.Grade)
	}
	stored, err := svc.GetJob(1, job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if stored.Status != models.AIJobStatusCompleted {
		t.Fatalf("job status = %s, want completed (error=%q)", stored.Status, stored.ErrorMessage)
	}
	if stored.Result != `{"gradingReport":"**Estimated Grade: VF-20** (Confidence: Medium)"}` {
		t.Fatalf("job result = %q", stored.Result)
	}
}
