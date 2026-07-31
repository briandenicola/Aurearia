package services

import (
	"testing"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCoinRecommendationServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coin{},
		&models.Tag{},
		&models.CoinTag{},
		&models.CoinSet{},
		&models.CoinSetMembership{},
		&models.CoinRecommendation{},
		&models.RecommendationFeedback{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestCoinRecommendationService_ListForCoin_SuggestsSetAndTag(t *testing.T) {
	db := setupCoinRecommendationServiceDB(t)
	createTestUserRecord(t, db, 1)

	targetCoin := models.Coin{Name: "Julia Domna denarius", Ruler: "Julia Domna", Category: models.CategoryRoman, Era: models.EraAncient, UserID: 1}
	peerCoin1 := models.Coin{Name: "Domna 1", Ruler: "Julia Domna", Category: models.CategoryRoman, Era: models.EraAncient, UserID: 1}
	peerCoin2 := models.Coin{Name: "Domna 2", Ruler: "Julia Domna", Category: models.CategoryRoman, Era: models.EraAncient, UserID: 1}
	if err := db.Create(&[]models.Coin{targetCoin, peerCoin1, peerCoin2}).Error; err != nil {
		t.Fatalf("failed to create coins: %v", err)
	}

	var coins []models.Coin
	if err := db.Order("id ASC").Find(&coins).Error; err != nil {
		t.Fatalf("failed to reload coins: %v", err)
	}
	targetCoin = coins[0]
	peerCoin1 = coins[1]
	peerCoin2 = coins[2]

	set := models.CoinSet{UserID: 1, Name: "Women of the Severan Dynasty", SetType: models.CoinSetTypeGoal}
	tag := models.Tag{UserID: 1, Name: "Severan Women", Color: "#c9a84c"}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("failed to create set: %v", err)
	}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}
	if err := db.Create(&[]models.CoinSetMembership{
		{SetID: set.ID, CoinID: peerCoin1.ID},
		{SetID: set.ID, CoinID: peerCoin2.ID},
	}).Error; err != nil {
		t.Fatalf("failed to create set memberships: %v", err)
	}
	if err := db.Create(&[]models.CoinTag{
		{CoinID: peerCoin1.ID, TagID: tag.ID},
		{CoinID: peerCoin2.ID, TagID: tag.ID},
	}).Error; err != nil {
		t.Fatalf("failed to create tag memberships: %v", err)
	}

	svc := NewCoinRecommendationService(
		repository.NewCoinRecommendationRepository(db),
		repository.NewTagRepository(db),
		repository.NewSetRepository(db),
	)

	recommendations, err := svc.ListForCoin(targetCoin.ID, 1)
	if err != nil {
		t.Fatalf("ListForCoin returned error: %v", err)
	}
	if len(recommendations) < 2 {
		t.Fatalf("expected at least 2 recommendations, got %d", len(recommendations))
	}

	foundSet := false
	foundTag := false
	for _, rec := range recommendations {
		if rec.TargetType == models.RecommendationTargetTypeSet && rec.TargetID == set.ID {
			foundSet = true
		}
		if rec.TargetType == models.RecommendationTargetTypeTag && rec.TargetID == tag.ID {
			foundTag = true
		}
	}
	if !foundSet {
		t.Fatalf("expected set recommendation for set %d", set.ID)
	}
	if !foundTag {
		t.Fatalf("expected tag recommendation for tag %d", tag.ID)
	}
}

func TestCoinRecommendationService_AcceptAndRejectFlow(t *testing.T) {
	db := setupCoinRecommendationServiceDB(t)
	createTestUserRecord(t, db, 1)

	targetCoin := models.Coin{Name: "Julia Domna", Ruler: "Julia Domna", Category: models.CategoryRoman, Era: models.EraAncient, UserID: 1}
	peerCoin1 := models.Coin{Name: "Domna 1", Ruler: "Julia Domna", Category: models.CategoryRoman, Era: models.EraAncient, UserID: 1}
	peerCoin2 := models.Coin{Name: "Domna 2", Ruler: "Julia Domna", Category: models.CategoryRoman, Era: models.EraAncient, UserID: 1}
	if err := db.Create(&[]models.Coin{targetCoin, peerCoin1, peerCoin2}).Error; err != nil {
		t.Fatalf("failed to create coins: %v", err)
	}
	var coins []models.Coin
	if err := db.Order("id ASC").Find(&coins).Error; err != nil {
		t.Fatalf("failed to reload coins: %v", err)
	}
	targetCoin = coins[0]
	peerCoin1 = coins[1]
	peerCoin2 = coins[2]

	set := models.CoinSet{UserID: 1, Name: "Severans", SetType: models.CoinSetTypeGoal}
	tag := models.Tag{UserID: 1, Name: "Julia", Color: "#c9a84c"}
	if err := db.Create(&set).Error; err != nil {
		t.Fatalf("failed to create set: %v", err)
	}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}
	if err := db.Create(&[]models.CoinSetMembership{
		{SetID: set.ID, CoinID: peerCoin1.ID},
		{SetID: set.ID, CoinID: peerCoin2.ID},
	}).Error; err != nil {
		t.Fatalf("failed to create set memberships: %v", err)
	}
	if err := db.Create(&[]models.CoinTag{
		{CoinID: peerCoin1.ID, TagID: tag.ID},
		{CoinID: peerCoin2.ID, TagID: tag.ID},
	}).Error; err != nil {
		t.Fatalf("failed to create tag memberships: %v", err)
	}

	svc := NewCoinRecommendationService(
		repository.NewCoinRecommendationRepository(db),
		repository.NewTagRepository(db),
		repository.NewSetRepository(db),
	)
	recommendations, err := svc.ListForCoin(targetCoin.ID, 1)
	if err != nil {
		t.Fatalf("ListForCoin returned error: %v", err)
	}
	if len(recommendations) == 0 {
		t.Fatal("expected at least one recommendation")
	}

	var setRecID uint
	var tagRecID uint
	for _, rec := range recommendations {
		if rec.TargetType == models.RecommendationTargetTypeSet {
			setRecID = rec.ID
		}
		if rec.TargetType == models.RecommendationTargetTypeTag {
			tagRecID = rec.ID
		}
	}
	if setRecID == 0 || tagRecID == 0 {
		t.Fatalf("expected both set and tag recommendations; set=%d tag=%d", setRecID, tagRecID)
	}

	if err := svc.Accept(targetCoin.ID, setRecID, 1); err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	var setMembershipCount int64
	if err := db.Model(&models.CoinSetMembership{}).Where("set_id = ? AND coin_id = ?", set.ID, targetCoin.ID).Count(&setMembershipCount).Error; err != nil {
		t.Fatalf("failed counting set memberships: %v", err)
	}
	if setMembershipCount != 1 {
		t.Fatalf("expected accepted recommendation to add set membership, got %d rows", setMembershipCount)
	}

	if err := svc.Reject(targetCoin.ID, tagRecID, 1); err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}
	var rejected models.CoinRecommendation
	if err := db.First(&rejected, tagRecID).Error; err != nil {
		t.Fatalf("failed loading recommendation: %v", err)
	}
	if rejected.Status != models.RecommendationStatusRejected {
		t.Fatalf("expected recommendation status rejected, got %s", rejected.Status)
	}
}

func createTestUserRecord(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	user := models.User{
		ID:           userID,
		Username:     "tester",
		PasswordHash: "hash",
		Email:        "tester@example.com",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
}
