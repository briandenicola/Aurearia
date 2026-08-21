package services

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrRecommendationNotFound = errors.New("recommendation not found")
	ErrRecommendationTarget   = errors.New("recommendation target is invalid")
)

const (
	minRecommendationSampleSize      = 2
	maxRecommendationsPerCoin        = 12
	// medium confidence allows thematic tags (same category/era/material, different ruler)
	// to score high enough after the ruler weight was reduced from 0.45 to 0.30.
	requiredRecommendationConfidence = "medium"
)

type CoinRecommendationItem struct {
	ID         uint                            `json:"id"`
	TargetType models.RecommendationTargetType `json:"targetType"`
	TargetID   uint                            `json:"targetId"`
	TargetName string                          `json:"targetName"`
	Score      float64                         `json:"score"`
	Confidence string                          `json:"confidence"`
	Reasons    []string                        `json:"reasons"`
	Status     models.RecommendationStatus     `json:"status"`
}

type CoinRecommendationService struct {
	repo    *repository.CoinRecommendationRepository
	tagRepo *repository.TagRepository
	setRepo *repository.SetRepository
}

func NewCoinRecommendationService(
	repo *repository.CoinRecommendationRepository,
	tagRepo *repository.TagRepository,
	setRepo *repository.SetRepository,
) *CoinRecommendationService {
	return &CoinRecommendationService{
		repo:    repo,
		tagRepo: tagRepo,
		setRepo: setRepo,
	}
}

func (s *CoinRecommendationService) ListForCoin(coinID, userID uint) ([]CoinRecommendationItem, error) {
	coin, err := s.repo.GetCoinByID(coinID, userID)
	if err != nil {
		return nil, err
	}

	coins, err := s.repo.ListOwnedCoinsWithMemberships(userID)
	if err != nil {
		return nil, err
	}
	sets, err := s.repo.ListOwnedManualSets(userID)
	if err != nil {
		return nil, err
	}
	tags, err := s.repo.ListOwnedTags(userID)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.ListByCoin(coinID, userID)
	if err != nil {
		return nil, err
	}
	existingByTarget := make(map[string]models.CoinRecommendation, len(existing))
	for _, rec := range existing {
		existingByTarget[recommendationTargetKey(rec.TargetType, rec.TargetID)] = rec
	}

	profiles := buildTargetProfiles(coins, coinID)
	alreadyInSet := make(map[uint]bool, len(coin.Sets))
	for _, set := range coin.Sets {
		alreadyInSet[set.ID] = true
	}
	alreadyTagged := make(map[uint]bool, len(coin.Tags))
	for _, tag := range coin.Tags {
		alreadyTagged[tag.ID] = true
	}

	generated := make([]CoinRecommendationItem, 0, 24)
	for _, set := range sets {
		if alreadyInSet[set.ID] {
			continue
		}
		profile, ok := profiles.sets[set.ID]
		if !ok || profile.sampleSize < minRecommendationSampleSize {
			continue
		}
		score, reasons := scoreCoinAgainstProfile(coin, profile, "set")
		if score <= 0 {
			continue
		}
		confidence := confidenceTier(score)
		if !confidenceMeetsMinimum(confidence, requiredRecommendationConfidence) {
			continue
		}
		key := recommendationTargetKey(models.RecommendationTargetTypeSet, set.ID)
		if existingRec, ok := existingByTarget[key]; ok && existingRec.Status == models.RecommendationStatusRejected {
			continue
		}
		generated = append(generated, CoinRecommendationItem{
			TargetType: models.RecommendationTargetTypeSet,
			TargetID:   set.ID,
			TargetName: set.Name,
			Score:      score,
			Confidence: confidence,
			Reasons:    reasons,
			Status:     models.RecommendationStatusPending,
		})
	}

	for _, tag := range tags {
		if alreadyTagged[tag.ID] {
			continue
		}
		profile, ok := profiles.tags[tag.ID]
		if !ok || profile.sampleSize < minRecommendationSampleSize {
			continue
		}
		score, reasons := scoreCoinAgainstProfile(coin, profile, "tag")
		if score <= 0 {
			continue
		}
		confidence := confidenceTier(score)
		if !confidenceMeetsMinimum(confidence, requiredRecommendationConfidence) {
			continue
		}
		key := recommendationTargetKey(models.RecommendationTargetTypeTag, tag.ID)
		if existingRec, ok := existingByTarget[key]; ok && existingRec.Status == models.RecommendationStatusRejected {
			continue
		}
		generated = append(generated, CoinRecommendationItem{
			TargetType: models.RecommendationTargetTypeTag,
			TargetID:   tag.ID,
			TargetName: tag.Name,
			Score:      score,
			Confidence: confidence,
			Reasons:    reasons,
			Status:     models.RecommendationStatusPending,
		})
	}

	sort.Slice(generated, func(i, j int) bool {
		if generated[i].Score == generated[j].Score {
			if generated[i].TargetType == generated[j].TargetType {
				return generated[i].TargetName < generated[j].TargetName
			}
			return generated[i].TargetType < generated[j].TargetType
		}
		return generated[i].Score > generated[j].Score
	})
	if len(generated) > maxRecommendationsPerCoin {
		generated = generated[:maxRecommendationsPerCoin]
	}

	toPersist := make([]models.CoinRecommendation, 0, len(generated))
	for _, item := range generated {
		toPersist = append(toPersist, models.CoinRecommendation{
			UserID:     userID,
			CoinID:     coinID,
			TargetType: item.TargetType,
			TargetID:   item.TargetID,
			Score:      item.Score,
			Confidence: item.Confidence,
			Reasons:    models.StringList(item.Reasons),
			Status:     models.RecommendationStatusPending,
		})
	}
	if err := s.repo.UpsertMany(toPersist); err != nil {
		return nil, err
	}

	refreshed, err := s.repo.ListByCoin(coinID, userID)
	if err != nil {
		return nil, err
	}

	setNameByID := make(map[uint]string, len(sets))
	for _, set := range sets {
		setNameByID[set.ID] = set.Name
	}
	tagNameByID := make(map[uint]string, len(tags))
	for _, tag := range tags {
		tagNameByID[tag.ID] = tag.Name
	}

	result := make([]CoinRecommendationItem, 0, len(refreshed))
	for _, rec := range refreshed {
		if rec.Status != models.RecommendationStatusPending {
			continue
		}
		if !confidenceMeetsMinimum(rec.Confidence, requiredRecommendationConfidence) {
			continue
		}
		targetName := ""
		switch rec.TargetType {
		case models.RecommendationTargetTypeSet:
			targetName = setNameByID[rec.TargetID]
		case models.RecommendationTargetTypeTag:
			targetName = tagNameByID[rec.TargetID]
		}
		if targetName == "" {
			continue
		}
		result = append(result, CoinRecommendationItem{
			ID:         rec.ID,
			TargetType: rec.TargetType,
			TargetID:   rec.TargetID,
			TargetName: targetName,
			Score:      rec.Score,
			Confidence: rec.Confidence,
			Reasons:    []string(rec.Reasons),
			Status:     rec.Status,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Score > result[j].Score })
	return result, nil
}

func (s *CoinRecommendationService) Accept(coinID, recID, userID uint) error {
	rec, err := s.repo.GetByID(recID, coinID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return ErrRecommendationNotFound
		}
		return err
	}

	switch rec.TargetType {
	case models.RecommendationTargetTypeSet:
		if err := s.setRepo.AddCoinToSet(coinID, rec.TargetID, userID, "accepted recommendation"); err != nil {
			return err
		}
	case models.RecommendationTargetTypeTag:
		if err := s.tagRepo.AttachToCoin(coinID, rec.TargetID, userID); err != nil {
			return err
		}
	default:
		return ErrRecommendationTarget
	}

	if err := s.repo.UpdateStatus(rec.ID, models.RecommendationStatusAccepted); err != nil {
		return err
	}
	return s.repo.CreateFeedback(&models.RecommendationFeedback{
		RecommendationID: rec.ID,
		UserID:           userID,
		Action:           models.RecommendationFeedbackActionAccept,
		Context:          models.StringMap{"targetType": string(rec.TargetType), "targetId": fmt.Sprintf("%d", rec.TargetID)},
	})
}

func (s *CoinRecommendationService) Reject(coinID, recID, userID uint) error {
	rec, err := s.repo.GetByID(recID, coinID, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return ErrRecommendationNotFound
		}
		return err
	}
	if err := s.repo.UpdateStatus(rec.ID, models.RecommendationStatusRejected); err != nil {
		return err
	}
	return s.repo.CreateFeedback(&models.RecommendationFeedback{
		RecommendationID: rec.ID,
		UserID:           userID,
		Action:           models.RecommendationFeedbackActionReject,
		Context:          models.StringMap{"targetType": string(rec.TargetType), "targetId": fmt.Sprintf("%d", rec.TargetID)},
	})
}

type recommendationProfile struct {
	sampleSize        int
	rulerCount        map[string]int
	categoryCount     map[string]int
	eraCount          map[string]int
	mintCount         map[string]int
	denominationCount map[string]int
	materialCount     map[string]int
}

type recommendationProfiles struct {
	sets map[uint]*recommendationProfile
	tags map[uint]*recommendationProfile
}

func newRecommendationProfile() *recommendationProfile {
	return &recommendationProfile{
		rulerCount:        make(map[string]int),
		categoryCount:     make(map[string]int),
		eraCount:          make(map[string]int),
		mintCount:         make(map[string]int),
		denominationCount: make(map[string]int),
		materialCount:     make(map[string]int),
	}
}

func buildTargetProfiles(coins []models.Coin, excludeCoinID uint) recommendationProfiles {
	profiles := recommendationProfiles{
		sets: make(map[uint]*recommendationProfile),
		tags: make(map[uint]*recommendationProfile),
	}
	for _, coin := range coins {
		if coin.ID == excludeCoinID {
			continue
		}
		for _, set := range coin.Sets {
			profile, ok := profiles.sets[set.ID]
			if !ok {
				profile = newRecommendationProfile()
				profiles.sets[set.ID] = profile
			}
			addCoinToProfile(profile, coin)
		}
		for _, tag := range coin.Tags {
			profile, ok := profiles.tags[tag.ID]
			if !ok {
				profile = newRecommendationProfile()
				profiles.tags[tag.ID] = profile
			}
			addCoinToProfile(profile, coin)
		}
	}
	return profiles
}

func addCoinToProfile(profile *recommendationProfile, coin models.Coin) {
	profile.sampleSize++
	incrementIfPresent(profile.rulerCount, coin.Ruler)
	// Skip "Other" category/material — they are "unknown" values and should not
	// contribute to similarity scoring (mirrors coinHasEnoughMetadata).
	if coin.Category != "" && coin.Category != "Other" {
		incrementIfPresent(profile.categoryCount, string(coin.Category))
	}
	incrementIfPresent(profile.eraCount, string(coin.Era))
	incrementIfPresent(profile.mintCount, coin.Mint)
	incrementIfPresent(profile.denominationCount, coin.Denomination)
	if coin.Material != "" && coin.Material != "Other" {
		incrementIfPresent(profile.materialCount, string(coin.Material))
	}
}

func incrementIfPresent(bucket map[string]int, raw string) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return
	}
	bucket[value]++
}

func scoreCoinAgainstProfile(coin *models.Coin, profile *recommendationProfile, targetLabel string) (float64, []string) {
	score := 0.0
	reasons := make([]string, 0, 3)
	addFeatureScore := func(feature string, bucket map[string]int, raw string, weight float64) {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" || profile.sampleSize == 0 {
			return
		}
		ratio := float64(bucket[value]) / float64(profile.sampleSize)
		if ratio <= 0 {
			return
		}
		score += ratio * weight
		if ratio >= 0.6 {
			reasons = append(reasons, fmt.Sprintf("%s matches %d/%d coins in this %s", feature, bucket[value], profile.sampleSize, targetLabel))
		}
	}

	// Weights sum to 1.0.  Ruler is 0.30 (was 0.45) so thematic tags whose coins
	// span multiple rulers can still qualify at medium confidence when category,
	// era, and/or material strongly agree.
	addFeatureScore("Ruler", profile.rulerCount, coin.Ruler, 0.30)
	if coin.Category != "" && coin.Category != "Other" {
		addFeatureScore("Category", profile.categoryCount, string(coin.Category), 0.20)
	}
	addFeatureScore("Era", profile.eraCount, string(coin.Era), 0.20)
	addFeatureScore("Mint", profile.mintCount, coin.Mint, 0.15)
	addFeatureScore("Denomination", profile.denominationCount, coin.Denomination, 0.075)
	if coin.Material != "" && coin.Material != "Other" {
		addFeatureScore("Material", profile.materialCount, string(coin.Material), 0.075)
	}

	if len(reasons) == 0 && score > 0 {
		reasons = append(reasons, "Multiple metadata fields align with this target's existing coins")
	}
	return score, reasons
}

func confidenceTier(score float64) string {
	switch {
	case score >= 0.7:
		return "high"
	case score >= 0.45:
		return "medium"
	default:
		return "low"
	}
}

// confidenceMeetsMinimum returns true when tier is at or above the minimum required level.
// Ordered: high > medium > low.
func confidenceMeetsMinimum(tier, minimum string) bool {
	order := map[string]int{"high": 2, "medium": 1, "low": 0}
	return order[tier] >= order[minimum]
}

func recommendationTargetKey(targetType models.RecommendationTargetType, targetID uint) string {
	return fmt.Sprintf("%s:%d", targetType, targetID)
}
