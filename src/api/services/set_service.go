package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

const maxSetsPerUser = 100
const maxSetNameLength = 80

var (
	ErrInvalidSetOrder = errors.New("ordered coin IDs must exactly match current set members")
	ErrSmartSetOrder   = errors.New("cannot manually reorder smart sets")
)

// SetService handles business logic for coin sets.
type SetService struct {
	repo      *repository.SetRepository
	tagRepo   *repository.TagRepository
	notifRepo *repository.NotificationRepository
}

// NewSetService creates a new SetService.
func NewSetService(repo *repository.SetRepository, tagRepo *repository.TagRepository, notifRepo ...*repository.NotificationRepository) *SetService {
	var notifications *repository.NotificationRepository
	if len(notifRepo) > 0 {
		notifications = notifRepo[0]
	}
	return &SetService{
		repo:      repo,
		tagRepo:   tagRepo,
		notifRepo: notifications,
	}
}

// ListSets returns all sets for a user with summary data.
func (s *SetService) ListSets(userID uint) ([]map[string]interface{}, error) {
	if err := s.repo.MigrateTagsToSets(userID); err != nil {
		return nil, err
	}

	sets, err := s.repo.List(userID)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(sets))
	for _, set := range sets {
		summary, err := s.repo.GetSetSummary(set.ID, userID)
		if err != nil {
			return nil, err
		}

		var completion interface{}
		if set.SetType == models.CoinSetTypeDefined || set.SetType == models.CoinSetTypeGoal {
			if c, err := s.repo.GetSetCompletion(set.ID, userID); err == nil {
				completion = c["completionPercentage"]
			}
		}

		setData := map[string]interface{}{
			"id":                   set.ID,
			"name":                 set.Name,
			"color":                set.Color,
			"icon":                 set.Icon,
			"setType":              set.SetType,
			"coinCount":            summary["coinCount"],
			"totalValue":           summary["totalValue"],
			"completionPercentage": completion,
			"valueChangePercent":   nil, // Will be populated in US3
		}
		result = append(result, setData)
	}

	return result, nil
}

// GetSetDetail returns detailed set information with aggregates.
func (s *SetService) GetSetDetail(setID, userID uint) (map[string]interface{}, error) {
	set, err := s.repo.GetByID(setID, userID)
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.GetSetSummary(setID, userID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":                   set.ID,
		"name":                 set.Name,
		"description":          set.Description,
		"color":                set.Color,
		"icon":                 set.Icon,
		"setType":              set.SetType,
		"parentSetId":          set.ParentSetID,
		"targetCompletionDate": set.TargetCompletionDate,
		"coinCount":            summary["coinCount"],
		"totalValue":           summary["totalValue"],
		"totalInvested":        summary["totalInvested"],
		"avgValuePerCoin":      summary["avgValuePerCoin"],
		"highestValueCoinId":   summary["highestValueCoinId"],
		"completionPercentage": nil,
	}
	if set.SetType == models.CoinSetTypeDefined || set.SetType == models.CoinSetTypeGoal {
		if c, err := s.repo.GetSetCompletion(set.ID, userID); err == nil {
			result["completionPercentage"] = c["completionPercentage"]
		}
	}

	return result, nil
}

// CreateSet creates a new set with validation.
func (s *SetService) CreateSet(userID uint, input map[string]interface{}) (*models.CoinSet, error) {
	// Validate name
	name, ok := input["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	name = strings.TrimSpace(name)
	if len(name) > maxSetNameLength {
		return nil, fmt.Errorf("name must be %d characters or less", maxSetNameLength)
	}

	// Check max sets limit
	count, err := s.repo.CountByUser(userID)
	if err != nil {
		return nil, err
	}
	if count >= maxSetsPerUser {
		return nil, fmt.Errorf("maximum of %d sets allowed", maxSetsPerUser)
	}

	// Check case-insensitive uniqueness
	exists, err := s.repo.ExistsByName(userID, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("a set with this name already exists")
	}

	// Validate set type
	setType, ok := input["setType"].(string)
	if !ok || setType == "" {
		setType = string(models.CoinSetTypeOpen)
	}
	if setType != string(models.CoinSetTypeOpen) && setType != string(models.CoinSetTypeDefined) && setType != string(models.CoinSetTypeSmart) && setType != string(models.CoinSetTypeGoal) {
		return nil, fmt.Errorf("invalid set type")
	}

	var smartCriteria *models.JSONObject
	if raw, ok := input["smartCriteria"].(map[string]interface{}); ok && raw != nil {
		if err := ValidateSmartCriteria(raw); err != nil {
			return nil, err
		}
		criteria := models.JSONObject(raw)
		smartCriteria = &criteria
	}
	if setType == string(models.CoinSetTypeSmart) && smartCriteria == nil {
		return nil, fmt.Errorf("smart criteria is required for smart sets")
	}

	// Validate parent set ID if provided
	if parentSetID, ok := input["parentSetId"].(float64); ok && parentSetID > 0 {
		parentID := uint(parentSetID)
		parent, err := s.repo.GetByID(parentID, userID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("parent set not found")
			}
			return nil, err
		}
		// Check for cycle (simplified - just check direct parent)
		if parent.ParentSetID != nil && *parent.ParentSetID == parentID {
			return nil, fmt.Errorf("parent set cycle detected")
		}
	}

	// Create the set
	set := &models.CoinSet{
		UserID:        userID,
		Name:          name,
		Description:   getStringValue(input, "description"),
		Color:         getStringValueOrDefault(input, "color", "#6b7280"),
		Icon:          getStringValue(input, "icon"),
		SetType:       models.CoinSetType(setType),
		SmartCriteria: smartCriteria,
	}
	if rawDate := getStringValue(input, "targetCompletionDate"); rawDate != "" {
		t, err := time.Parse("2006-01-02", rawDate)
		if err != nil {
			return nil, fmt.Errorf("target completion date must be YYYY-MM-DD")
		}
		set.TargetCompletionDate = &t
	}

	if parentSetID, ok := input["parentSetId"].(float64); ok && parentSetID > 0 {
		pid := uint(parentSetID)
		set.ParentSetID = &pid
	}

	if err := s.repo.Create(set); err != nil {
		return nil, err
	}
	if templateID := getStringValue(input, "templateId"); templateID != "" {
		template := GetTemplateByID(templateID)
		if template == nil {
			return nil, fmt.Errorf("template not found")
		}
		if err := s.repo.CreateTargetsForSet(set.ID, CopyTemplateToCoinSetTargets(template, set.ID)); err != nil {
			return nil, err
		}
	}

	return set, nil
}

// UpdateSet updates a set with validation.
func (s *SetService) UpdateSet(setID, userID uint, updates map[string]interface{}) (*models.CoinSet, error) {
	set, err := s.repo.GetByID(setID, userID)
	if err != nil {
		return nil, err
	}

	// Validate name if provided
	if name, ok := updates["name"].(string); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("name cannot be empty")
		}
		if len(name) > maxSetNameLength {
			return nil, fmt.Errorf("name must be %d characters or less", maxSetNameLength)
		}
		// Check uniqueness (excluding current set)
		exists, err := s.repo.ExistsByName(userID, name)
		if err != nil {
			return nil, err
		}
		if exists && strings.ToLower(name) != strings.ToLower(set.Name) {
			return nil, fmt.Errorf("a set with this name already exists")
		}
		updates["name"] = name
	}

	if err := s.repo.Update(set, updates); err != nil {
		return nil, err
	}

	return s.repo.GetByID(setID, userID)
}

// DeleteSet deletes a set and its memberships.
func (s *SetService) DeleteSet(setID, userID uint) error {
	return s.repo.Delete(setID, userID)
}

// AddCoinToSet adds a coin to a manual set with validation.
func (s *SetService) AddCoinToSet(coinID, setID, userID uint, notes string) error {
	// Verify the set is not a smart set
	set, err := s.repo.GetByID(setID, userID)
	if err != nil {
		return err
	}
	if set.SetType == models.CoinSetTypeSmart {
		return fmt.Errorf("cannot manually add coins to smart sets")
	}

	return s.repo.AddCoinToSet(coinID, setID, userID, notes)
}

func (s *SetService) GetCompletion(setID, userID uint) (map[string]interface{}, error) {
	return s.repo.GetSetCompletion(setID, userID)
}

func (s *SetService) CreateSnapshot(setID, userID uint) (*models.CoinSetValuationSnapshot, error) {
	var completion *float64
	set, err := s.repo.GetByID(setID, userID)
	if err != nil {
		return nil, err
	}
	if set.SetType == models.CoinSetTypeDefined || set.SetType == models.CoinSetTypeGoal {
		c, err := s.repo.GetSetCompletion(setID, userID)
		if err == nil {
			if pct, ok := c["completionPercentage"].(float64); ok {
				completion = &pct
			}
		}
	}
	snapshot, err := s.repo.CreateSnapshot(setID, userID, completion)
	if err != nil {
		return nil, err
	}
	_ = s.evaluateMilestones(snapshot)
	return snapshot, nil
}

func (s *SetService) CreateSnapshotsForAllUsers() error {
	users, err := s.repo.ListUsersWithSets()
	if err != nil {
		return err
	}
	for _, userID := range users {
		ids, err := s.repo.GetAllSetIDsForUser(userID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if _, err := s.CreateSnapshot(id, userID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SetService) GetTrends(setID, userID uint, rangeKey string) ([]models.CoinSetValuationSnapshot, error) {
	since := snapshotSince(rangeKey)
	return s.repo.GetSnapshots(setID, userID, since)
}

func (s *SetService) GetAnalytics(setID, userID uint) (map[string]interface{}, error) {
	detail, err := s.GetSetDetail(setID, userID)
	if err != nil {
		return nil, err
	}
	totalValue, _ := detail["totalValue"].(float64)
	totalInvested, _ := detail["totalInvested"].(float64)
	var roi *float64
	if totalInvested > 0 {
		v := ((totalValue - totalInvested) / totalInvested) * 100
		roi = &v
	}
	snapshots, err := s.repo.GetSnapshots(setID, userID, nil)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"roiPercent":              roi,
		"bestPerformerCoinId":     detail["highestValueCoinId"],
		"worstPerformerCoinId":    nil,
		"acquisitionRatePerMonth": acquisitionRate(snapshots),
		"projectedCompletionDate": nil,
	}, nil
}

func (s *SetService) CompareSets(userID uint, setIDs []uint, rangeKey string) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0, len(setIDs))
	for _, setID := range setIDs {
		set, err := s.repo.GetByID(setID, userID)
		if err != nil {
			return nil, err
		}
		snapshots, err := s.GetTrends(setID, userID, rangeKey)
		if err != nil {
			return nil, err
		}
		var startValue, endValue float64
		if len(snapshots) > 0 {
			startValue = snapshots[0].TotalValue
			endValue = snapshots[len(snapshots)-1].TotalValue
		} else {
			summary, err := s.repo.GetSetSummary(setID, userID)
			if err != nil {
				return nil, err
			}
			endValue, _ = summary["totalValue"].(float64)
			startValue = endValue
		}
		valueChange := endValue - startValue
		var pct float64
		if startValue > 0 {
			pct = (valueChange / startValue) * 100
		}
		results = append(results, map[string]interface{}{
			"setId":              set.ID,
			"name":               set.Name,
			"startValue":         startValue,
			"endValue":           endValue,
			"valueChange":        valueChange,
			"valueChangePercent": pct,
			"completionChange":   nil,
		})
	}
	return results, nil
}

func (s *SetService) PreviewSmartSet(userID uint, criteria map[string]interface{}) (map[string]interface{}, error) {
	if err := ValidateSmartCriteria(criteria); err != nil {
		return nil, err
	}
	coins, err := s.repo.GetCoinsMatchingCriteria(userID, criteria)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(coins))
	var totalValue float64
	for _, coin := range coins {
		ids = append(ids, coin.ID)
		if coin.CurrentValue != nil {
			totalValue += *coin.CurrentValue
		}
	}
	return map[string]interface{}{
		"coinIds":    ids,
		"coinCount":  len(coins),
		"totalValue": totalValue,
	}, nil
}

func (s *SetService) evaluateMilestones(snapshot *models.CoinSetValuationSnapshot) error {
	if s.notifRepo == nil {
		return nil
	}
	alerts, err := s.repo.GetEnabledMilestoneAlerts(snapshot.SetID, snapshot.UserID)
	if err != nil {
		return err
	}
	for _, alert := range alerts {
		if alert.LastTriggeredAt != nil {
			continue
		}
		var current float64
		switch alert.Metric {
		case "total_value":
			current = snapshot.TotalValue
		case "completion_percentage":
			if snapshot.CompletionPercentage == nil {
				continue
			}
			current = *snapshot.CompletionPercentage
		case "coin_count":
			current = float64(snapshot.CoinCount)
		default:
			continue
		}
		triggered := (alert.Direction == "crosses_above" && current >= alert.Threshold) || (alert.Direction == "crosses_below" && current <= alert.Threshold)
		if !triggered {
			continue
		}
		if err := s.notifRepo.Create(&models.Notification{
			UserID:      snapshot.UserID,
			Type:        "set_milestone",
			Title:       "Set milestone reached",
			Message:     fmt.Sprintf("A coin set reached the %s milestone.", alert.Metric),
			ReferenceID: snapshot.SetID,
		}); err != nil {
			return err
		}
		if err := s.repo.MarkMilestoneAlertTriggered(alert.ID); err != nil {
			return err
		}
	}
	return nil
}

func snapshotSince(rangeKey string) *time.Time {
	now := time.Now()
	var since time.Time
	switch rangeKey {
	case "1m":
		since = now.AddDate(0, -1, 0)
	case "3m":
		since = now.AddDate(0, -3, 0)
	case "1y", "":
		since = now.AddDate(-1, 0, 0)
	case "all":
		return nil
	default:
		since = now.AddDate(-1, 0, 0)
	}
	return &since
}

func acquisitionRate(snapshots []models.CoinSetValuationSnapshot) *float64 {
	if len(snapshots) < 2 {
		return nil
	}
	days := snapshots[len(snapshots)-1].SnapshotDate.Sub(snapshots[0].SnapshotDate).Hours() / 24
	if days <= 0 {
		return nil
	}
	v := float64(snapshots[len(snapshots)-1].CoinCount-snapshots[0].CoinCount) / (days / 30)
	return &v
}

// RemoveCoinFromSet removes a coin from a manual set with validation.
func (s *SetService) RemoveCoinFromSet(coinID, setID, userID uint) error {
	// Verify the set is not a smart set
	set, err := s.repo.GetByID(setID, userID)
	if err != nil {
		return err
	}
	if set.SetType == models.CoinSetTypeSmart {
		return fmt.Errorf("cannot manually remove coins from smart sets")
	}

	return s.repo.RemoveCoinFromSet(coinID, setID, userID)
}

// GetCoinsInSet returns all coins in a set.
func (s *SetService) GetCoinsInSet(setID, userID uint) ([]models.Coin, error) {
	return s.repo.GetCoinsInSet(setID, userID)
}

// ReorderCoinsInSet saves the manual order for every current member of a non-smart set.
func (s *SetService) ReorderCoinsInSet(setID, userID uint, coinIDs []uint) error {
	set, err := s.repo.GetByID(setID, userID)
	if err != nil {
		return err
	}
	if set.SetType == models.CoinSetTypeSmart {
		return ErrSmartSetOrder
	}

	seen := make(map[uint]struct{}, len(coinIDs))
	for _, coinID := range coinIDs {
		if coinID == 0 {
			return ErrInvalidSetOrder
		}
		if _, exists := seen[coinID]; exists {
			return ErrInvalidSetOrder
		}
		seen[coinID] = struct{}{}
	}

	if err := s.repo.ReorderCoinsInSet(setID, userID, coinIDs); err != nil {
		if errors.Is(err, repository.ErrInvalidSetOrder) {
			return ErrInvalidSetOrder
		}
		return err
	}
	return nil
}

// MapTagToSet returns set representation of a tag for compatibility.
func (s *SetService) MapTagToSet(tag *models.Tag) map[string]interface{} {
	return map[string]interface{}{
		"id":                   tag.ID,
		"name":                 tag.Name,
		"color":                tag.Color,
		"icon":                 "",
		"setType":              models.CoinSetTypeOpen,
		"coinCount":            0, // Will be populated separately
		"totalValue":           0.0,
		"completionPercentage": nil,
		"valueChangePercent":   nil,
	}
}

// Helper functions
func getStringValue(input map[string]interface{}, key string) string {
	if val, ok := input[key].(string); ok {
		return strings.TrimSpace(val)
	}
	return ""
}

func getStringValueOrDefault(input map[string]interface{}, key, defaultVal string) string {
	if val, ok := input[key].(string); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

// CreateSetFromTemplate creates a defined or goal set from a template.
func (s *SetService) CreateSetFromTemplate(userID uint, input map[string]interface{}) (*models.CoinSet, error) {
	// Get template ID
	templateID, ok := input["templateId"].(string)
	if !ok || templateID == "" {
		return nil, fmt.Errorf("templateId is required")
	}

	// Get template
	template := GetTemplateByID(templateID)
	if template == nil {
		return nil, fmt.Errorf("template not found")
	}

	// Determine set type (default to defined)
	setType := string(models.CoinSetTypeDefined)
	if st, ok := input["setType"].(string); ok && st == string(models.CoinSetTypeGoal) {
		setType = string(models.CoinSetTypeGoal)
	}

	// Create the set
	setInput := map[string]interface{}{
		"name":        input["name"],
		"description": fmt.Sprintf("%s (from template: %s)", template.Description, template.Name),
		"setType":     setType,
	}
	if color, ok := input["color"]; ok {
		setInput["color"] = color
	}
	if icon, ok := input["icon"]; ok {
		setInput["icon"] = icon
	}
	if targetDate, ok := input["targetCompletionDate"]; ok {
		setInput["targetCompletionDate"] = targetDate
	}

	set, err := s.CreateSet(userID, setInput)
	if err != nil {
		return nil, err
	}

	// Copy template targets to set targets
	targets := CopyTemplateToCoinSetTargets(template, set.ID)
	if err := s.repo.CreateTargetsForSet(set.ID, targets); err != nil {
		// Rollback - delete the set
		s.repo.Delete(set.ID, userID)
		return nil, fmt.Errorf("failed to create targets: %w", err)
	}

	return set, nil
}

// CreateSetFromCSV creates a defined or goal set from CSV target import.
func (s *SetService) CreateSetFromCSV(userID uint, input map[string]interface{}, csvContent string) (*models.CoinSet, error) {
	// Parse CSV
	importer := NewSetTargetImport()
	targets, err := importer.ParseCSV(strings.NewReader(csvContent))
	if err != nil {
		return nil, err
	}

	// Validate targets
	if err := importer.ValidateTargets(targets); err != nil {
		return nil, err
	}

	// Determine set type (default to defined)
	setType := string(models.CoinSetTypeDefined)
	if st, ok := input["setType"].(string); ok && st == string(models.CoinSetTypeGoal) {
		setType = string(models.CoinSetTypeGoal)
	}

	// Create the set
	setInput := map[string]interface{}{
		"name":        input["name"],
		"description": input["description"],
		"setType":     setType,
	}
	if color, ok := input["color"]; ok {
		setInput["color"] = color
	}
	if icon, ok := input["icon"]; ok {
		setInput["icon"] = icon
	}

	set, err := s.CreateSet(userID, setInput)
	if err != nil {
		return nil, err
	}

	// Create targets
	if err := s.repo.CreateTargetsForSet(set.ID, targets); err != nil {
		// Rollback - delete the set
		s.repo.Delete(set.ID, userID)
		return nil, fmt.Errorf("failed to create targets: %w", err)
	}

	return set, nil
}
