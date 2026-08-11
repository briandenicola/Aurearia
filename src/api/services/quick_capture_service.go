package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

var (
	ErrQuickCaptureNotFound              = errors.New("quick capture draft not found")
	ErrQuickCaptureMinimumIdentity       = errors.New("add a working title, note, or image before saving a draft")
	ErrQuickCaptureInvalidPrice          = errors.New("purchase price must be zero or greater")
	ErrQuickCaptureDraftAlreadyPromoted  = errors.New("quick capture draft has already been promoted")
	ErrQuickCaptureDraftConcurrentAction = errors.New("draft is currently being promoted or is not in an active state")
	ErrQuickCaptureInvalidReference      = errors.New("selected Numista reference is invalid")
	ErrQuickCaptureReferenceConflict     = errors.New("selected Numista reference cannot be supplied while clearing it")
)

type QuickCapturePromotionTarget string

const (
	QuickCapturePromotionTargetCollection QuickCapturePromotionTarget = "collection"
	QuickCapturePromotionTargetWishlist   QuickCapturePromotionTarget = "wishlist"
)

// QuickCapturePromotionValidationError carries per-field validation messages for promotion.
type QuickCapturePromotionValidationError struct {
	Fields map[string]string
}

func (e *QuickCapturePromotionValidationError) Error() string {
	return "complete required fields before promotion"
}

type QuickCaptureImageUpload struct {
	Filename  string
	Data      []byte
	ImageType string
	IsPrimary bool
}

type CreateQuickCaptureDraftInput struct {
	UserID                   uint
	WorkingTitle             string
	DateRange                string
	Era                      string
	AcquisitionSource        string
	PurchasePrice            *float64
	Notes                    string
	Source                   string
	NGCCertNumber            string
	NGCLookupURL             string
	NGCGrade                 string
	LabelText                string
	AIConfidence             string
	SelectedNumistaReference *models.SelectedNumistaReference
	Images                   []QuickCaptureImageUpload
}

// UpdateQuickCaptureDraftInput carries all fields for a draft update.
// String fields always replace the current value (send current value to preserve it).
// RemoveImageIDsRaw is a comma-separated list of image IDs to remove.
type UpdateQuickCaptureDraftInput struct {
	UserID                   uint
	WorkingTitle             string
	DateRange                string
	Era                      string
	AcquisitionSource        string
	PurchasePrice            *float64
	PurchasePriceSet         bool // true means PurchasePrice was explicitly provided (even if nil)
	Notes                    string
	Source                   string
	NGCCertNumber            string
	NGCLookupURL             string
	NGCGrade                 string
	LabelText                string
	AIConfidence             string
	SelectedNumistaReference *models.SelectedNumistaReference
	SelectedNumistaProvided  bool
	ClearSelectedNumista     bool
	RemoveImageIDsRaw        string // e.g. "3,7,12"
	ReplaceObverse           bool
	ReplaceReverse           bool
	NewImages                []QuickCaptureImageUpload
}

// PromoteOverrides are optional coin-field overrides provided by the user at promotion time.
type PromoteOverrides struct {
	Name             *string
	Category         *string
	Material         *string
	Era              *string
	PurchasePrice    *float64
	PurchasePriceSet bool
	PurchaseLocation *string
	Notes            *string
}

// PromoteDraftInput holds promotion request data.
type PromoteDraftInput struct {
	Confirm   bool
	Target    QuickCapturePromotionTarget
	Overrides PromoteOverrides
}

// PromoteDraftResult is returned after a successful (or idempotent) promotion.
type PromoteDraftResult struct {
	DraftID         uint
	CoinID          uint
	AlreadyPromoted bool
	Target          QuickCapturePromotionTarget
}

type QuickCaptureService struct {
	repo         *repository.QuickCaptureRepository
	uploadDir    string
	coinSvc      *CoinService
	referenceSvc *CoinReferenceService
}

func NewQuickCaptureService(repo *repository.QuickCaptureRepository, uploadDir string) *QuickCaptureService {
	return &QuickCaptureService{repo: repo, uploadDir: uploadDir}
}

// WithCoinValidation wires in CoinService's era/category allow-list so draft
// promotion honors the same admin-defined values (and built-in defaults) as
// direct coin create/update, instead of a separately hardcoded check.
func (s *QuickCaptureService) WithCoinValidation(coinSvc *CoinService) *QuickCaptureService {
	s.coinSvc = coinSvc
	return s
}

func (s *QuickCaptureService) WithReferenceValidation(referenceSvc *CoinReferenceService) *QuickCaptureService {
	s.referenceSvc = referenceSvc
	return s
}

func (s *QuickCaptureService) CreateDraft(input CreateQuickCaptureDraftInput) (*models.QuickCaptureDraft, error) {
	if err := validateQuickCaptureIdentity(input.WorkingTitle, input.Notes, len(input.Images)); err != nil {
		return nil, err
	}
	if input.PurchasePrice != nil && *input.PurchasePrice < 0 {
		return nil, ErrQuickCaptureInvalidPrice
	}
	selectedReference, err := s.normalizeSelectedNumistaReference(input.UserID, input.SelectedNumistaReference)
	if err != nil {
		return nil, err
	}

	normalizedImages := make([]struct {
		upload    QuickCaptureImageUpload
		ext       string
		imageType models.ImageType
	}, 0, len(input.Images))
	for _, image := range input.Images {
		ext, err := NormalizeImageExt(filepath.Ext(image.Filename))
		if err != nil {
			return nil, err
		}
		if err := ValidateImageData(image.Data); err != nil {
			return nil, err
		}
		imageType, err := NormalizeImageType(image.ImageType)
		if err != nil {
			return nil, err
		}
		normalizedImages = append(normalizedImages, struct {
			upload    QuickCaptureImageUpload
			ext       string
			imageType models.ImageType
		}{upload: image, ext: ext, imageType: imageType})
	}

	draft := &models.QuickCaptureDraft{
		UserID:            input.UserID,
		WorkingTitle:      strings.TrimSpace(input.WorkingTitle),
		DateRange:         strings.TrimSpace(input.DateRange),
		Era:               strings.TrimSpace(input.Era),
		AcquisitionSource: strings.TrimSpace(input.AcquisitionSource),
		PurchasePrice:     input.PurchasePrice,
		Notes:             strings.TrimSpace(input.Notes),
		Source:            strings.TrimSpace(input.Source),
		NGCCertNumber:     strings.TrimSpace(input.NGCCertNumber),
		NGCLookupURL:      strings.TrimSpace(input.NGCLookupURL),
		NGCGrade:          strings.TrimSpace(input.NGCGrade),
		LabelText:         strings.TrimSpace(input.LabelText),
		AIConfidence:      strings.TrimSpace(input.AIConfidence),
		Status:            models.QuickCaptureDraftStatusActive,
	}
	var writtenFiles []string
	cleanupWrittenFiles := func() {
		for _, filePath := range writtenFiles {
			fullPath := filepath.Join(s.uploadDir, filepath.FromSlash(filePath))
			_ = os.Remove(fullPath)
			_ = os.Remove(filepath.Dir(fullPath))
		}
	}

	createdEvent := &models.DraftLifecycleEvent{
		UserID:    input.UserID,
		EventType: models.DraftLifecycleEventCreated,
		Message:   "Quick Capture draft created",
		CreatedAt: time.Now().UTC(),
	}
	err = s.repo.CreateDraftWithImages(draft, createdEvent, selectedReference, func(draftID uint) ([]models.QuickCaptureDraftImage, error) {
		images := make([]models.QuickCaptureDraftImage, 0, len(normalizedImages))
		for index, normalized := range normalizedImages {
			filePath, err := s.saveDraftImageFile(draftID, normalized.upload.Data, normalized.ext, normalized.imageType)
			if err != nil {
				return nil, err
			}
			writtenFiles = append(writtenFiles, filePath)
			images = append(images, models.QuickCaptureDraftImage{
				DraftID:      draftID,
				UserID:       input.UserID,
				FilePath:     filePath,
				ImageType:    normalized.imageType,
				IsPrimary:    normalized.upload.IsPrimary,
				DisplayOrder: index,
			})
		}
		return images, nil
	})
	if err != nil {
		cleanupWrittenFiles()
		return nil, err
	}

	return s.repo.GetDraftForOwner(draft.ID, input.UserID)
}

func (s *QuickCaptureService) ListDrafts(userID uint, status models.QuickCaptureDraftStatus, page, limit int) ([]models.QuickCaptureDraft, int64, error) {
	if !models.IsValidQuickCaptureDraftStatus(status) || status == models.QuickCaptureDraftStatusPromoting {
		status = models.QuickCaptureDraftStatusActive
	}
	return s.repo.ListDraftsForOwner(userID, status, page, limit)
}

func (s *QuickCaptureService) GetDraft(userID, draftID uint) (*models.QuickCaptureDraft, error) {
	draft, err := s.repo.GetDraftForOwner(draftID, userID)
	if err != nil {
		return nil, ErrQuickCaptureNotFound
	}
	return draft, nil
}

func validateQuickCaptureIdentity(title, notes string, imageCount int) error {
	if strings.TrimSpace(title) == "" && strings.TrimSpace(notes) == "" && imageCount == 0 {
		return ErrQuickCaptureMinimumIdentity
	}
	return nil
}

// UpdateDraft applies field and image changes to an active draft in one transaction.
func (s *QuickCaptureService) UpdateDraft(userID, draftID uint, input UpdateQuickCaptureDraftInput) (*models.QuickCaptureDraft, error) {
	// Load current draft to assess remaining identity after removals
	current, err := s.repo.GetDraftForOwner(draftID, userID)
	if err != nil {
		return nil, ErrQuickCaptureNotFound
	}
	if current.Status != models.QuickCaptureDraftStatusActive {
		return nil, ErrQuickCaptureNotFound
	}
	if input.ClearSelectedNumista && input.SelectedNumistaProvided {
		return nil, ErrQuickCaptureReferenceConflict
	}
	var referenceMutation repository.DraftReferenceMutation
	if input.ClearSelectedNumista {
		referenceMutation.Clear = true
	} else if input.SelectedNumistaProvided {
		selectedReference, err := s.normalizeSelectedNumistaReference(userID, input.SelectedNumistaReference)
		if err != nil {
			return nil, err
		}
		referenceMutation.Replace = selectedReference
	}

	// Parse remove-by-ID list
	var removeByIDs []uint
	for _, part := range strings.Split(input.RemoveImageIDsRaw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 32)
		if err == nil {
			removeByIDs = append(removeByIDs, uint(id))
		}
	}

	// Determine remove-by-type only when a replacement image of that type is present.
	hasNewObverse := false
	hasNewReverse := false
	for _, img := range input.NewImages {
		switch strings.ToLower(strings.TrimSpace(img.ImageType)) {
		case string(models.ImageTypeObverse):
			hasNewObverse = true
		case string(models.ImageTypeReverse):
			hasNewReverse = true
		}
	}
	var removeByTypes []models.ImageType
	if input.ReplaceObverse && hasNewObverse {
		removeByTypes = append(removeByTypes, models.ImageTypeObverse)
	}
	if input.ReplaceReverse && hasNewReverse {
		removeByTypes = append(removeByTypes, models.ImageTypeReverse)
	}

	// Estimate remaining image count for minimum-identity check
	removeIDSet := make(map[uint]bool, len(removeByIDs))
	for _, id := range removeByIDs {
		removeIDSet[id] = true
	}
	removeTypeSet := make(map[models.ImageType]bool, len(removeByTypes))
	for _, t := range removeByTypes {
		removeTypeSet[t] = true
	}
	remainingImages := 0
	for _, img := range current.Images {
		if !removeIDSet[img.ID] && !removeTypeSet[img.ImageType] {
			remainingImages++
		}
	}
	remainingImages += len(input.NewImages)

	if input.PurchasePrice != nil && *input.PurchasePrice < 0 {
		return nil, ErrQuickCaptureInvalidPrice
	}

	// Validate minimum identity after update
	if err := validateQuickCaptureIdentity(input.WorkingTitle, input.Notes, remainingImages); err != nil {
		return nil, err
	}

	// Validate and save new image files
	newImageRecords := make([]models.QuickCaptureDraftImage, 0, len(input.NewImages))
	var writtenFiles []string
	cleanupWritten := func() {
		for _, p := range writtenFiles {
			fullPath := filepath.Join(s.uploadDir, filepath.FromSlash(p))
			_ = os.Remove(fullPath)
		}
	}

	for idx, img := range input.NewImages {
		ext, err := NormalizeImageExt(filepath.Ext(img.Filename))
		if err != nil {
			cleanupWritten()
			return nil, err
		}
		if err := ValidateImageData(img.Data); err != nil {
			cleanupWritten()
			return nil, err
		}
		imageType, err := NormalizeImageType(img.ImageType)
		if err != nil {
			cleanupWritten()
			return nil, err
		}
		filePath, err := s.saveDraftImageFile(draftID, img.Data, ext, imageType)
		if err != nil {
			cleanupWritten()
			return nil, err
		}
		writtenFiles = append(writtenFiles, filePath)
		newImageRecords = append(newImageRecords, models.QuickCaptureDraftImage{
			DraftID:      draftID,
			UserID:       userID,
			FilePath:     filePath,
			ImageType:    imageType,
			IsPrimary:    img.IsPrimary,
			DisplayOrder: len(current.Images) - len(removeByIDs) + idx,
		})
	}

	// Build scalar field updates map
	fieldUpdates := map[string]interface{}{
		"working_title":      strings.TrimSpace(input.WorkingTitle),
		"date_range":         strings.TrimSpace(input.DateRange),
		"era":                strings.TrimSpace(input.Era),
		"acquisition_source": strings.TrimSpace(input.AcquisitionSource),
		"notes":              strings.TrimSpace(input.Notes),
	}
	if input.PurchasePriceSet {
		fieldUpdates["purchase_price"] = input.PurchasePrice
	}

	lifecycleEvent := &models.DraftLifecycleEvent{
		UserID:    userID,
		EventType: models.DraftLifecycleEventUpdated,
		Message:   "Draft updated",
		CreatedAt: time.Now().UTC(),
	}

	updated, removedPaths, err := s.repo.UpdateDraftTransaction(
		draftID, userID,
		fieldUpdates,
		removeByIDs,
		removeByTypes,
		newImageRecords,
		lifecycleEvent,
		referenceMutation,
	)
	if err != nil {
		cleanupWritten()
		return nil, err
	}

	// Remove files for deleted images (best-effort)
	for _, p := range removedPaths {
		fullPath := filepath.Join(s.uploadDir, filepath.FromSlash(p))
		_ = os.Remove(fullPath)
	}

	return updated, nil
}

func (s *QuickCaptureService) normalizeSelectedNumistaReference(
	userID uint,
	selected *models.SelectedNumistaReference,
) (*models.QuickCaptureDraftReference, error) {
	if selected == nil {
		return nil, nil
	}
	if err := selected.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuickCaptureInvalidReference, err)
	}
	normalized := models.CoinReference{
		Catalog: selected.Catalog,
		Number:  selected.Number,
		URI:     selected.URI,
	}
	if s.referenceSvc != nil {
		var err error
		normalized, err = s.referenceSvc.NormalizeAndValidateOne(normalized)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrQuickCaptureInvalidReference, err)
		}
	}
	return &models.QuickCaptureDraftReference{
		UserID:  userID,
		Catalog: normalized.Catalog,
		Number:  normalized.Number,
		URI:     normalized.URI,
	}, nil
}

// DiscardDraft marks an active draft as discarded and records a lifecycle event.
// Idempotent: already-discarded drafts are returned as-is.
// Returns ErrQuickCaptureDraftAlreadyPromoted for promoted drafts.
func (s *QuickCaptureService) DiscardDraft(userID, draftID uint) (*models.QuickCaptureDraft, error) {
	draft, err := s.repo.GetDraftForOwner(draftID, userID)
	if err != nil {
		return nil, ErrQuickCaptureNotFound
	}
	if draft.Status == models.QuickCaptureDraftStatusDiscarded {
		return draft, nil
	}
	if draft.Status == models.QuickCaptureDraftStatusPromoted ||
		draft.Status == models.QuickCaptureDraftStatusPromoting {
		return nil, ErrQuickCaptureDraftAlreadyPromoted
	}
	discarded, err := s.repo.DiscardDraft(draftID, userID)
	if err != nil {
		return nil, ErrQuickCaptureNotFound
	}
	_ = s.repo.AddLifecycleEvent(&models.DraftLifecycleEvent{
		DraftID:   draftID,
		UserID:    userID,
		EventType: models.DraftLifecycleEventDiscarded,
		Message:   "Quick Capture draft discarded",
		CreatedAt: time.Now().UTC(),
	})
	return discarded, nil
}

// PromoteDraft validates and transactionally promotes a draft into a normal Coin.
// Idempotent: returns the existing coin if already promoted.
func (s *QuickCaptureService) PromoteDraft(userID, draftID uint, input PromoteDraftInput) (*PromoteDraftResult, error) {
	if !input.Confirm {
		return nil, &QuickCapturePromotionValidationError{
			Fields: map[string]string{"confirm": "confirm must be true to promote"},
		}
	}
	target, err := normalizeQuickCapturePromotionTarget(input.Target)
	if err != nil {
		return nil, &QuickCapturePromotionValidationError{
			Fields: map[string]string{"target": "target must be collection or wishlist"},
		}
	}

	draft, err := s.repo.GetDraftForOwner(draftID, userID)
	if err != nil {
		return nil, ErrQuickCaptureNotFound
	}

	// Idempotent: already promoted
	if draft.Status == models.QuickCaptureDraftStatusPromoted && draft.PromotedCoinID != nil {
		existingTarget := target
		if promotedCoin, err := s.repo.GetCoinForOwner(*draft.PromotedCoinID, userID); err == nil {
			existingTarget = targetForPromotedCoin(promotedCoin)
		}
		return &PromoteDraftResult{
			DraftID:         draftID,
			CoinID:          *draft.PromotedCoinID,
			AlreadyPromoted: true,
			Target:          existingTarget,
		}, nil
	}

	// Discarded → not eligible
	if draft.Status == models.QuickCaptureDraftStatusDiscarded {
		return nil, ErrQuickCaptureDraftConcurrentAction
	}
	// Promoting → concurrent conflict
	if draft.Status == models.QuickCaptureDraftStatusPromoting {
		return nil, ErrQuickCaptureDraftConcurrentAction
	}

	// Build coin from draft fields + overrides
	coin := s.buildCoinFromDraft(draft, input.Overrides)
	coin.IsWishlist = target == QuickCapturePromotionTargetWishlist

	// Validate minimum coin requirements
	if fieldErrors := s.validateCoinMinimumForPromotion(coin); len(fieldErrors) > 0 {
		return nil, &QuickCapturePromotionValidationError{Fields: fieldErrors}
	}

	// Transactional promotion
	_, createdCoin, err := s.repo.PromoteDraftTransaction(draftID, userID, coin)
	if err != nil {
		if errors.Is(err, repository.ErrDraftNotClaimable) {
			return nil, ErrQuickCaptureDraftConcurrentAction
		}
		return nil, err
	}

	return &PromoteDraftResult{
		DraftID:         draftID,
		CoinID:          createdCoin.ID,
		AlreadyPromoted: false,
		Target:          target,
	}, nil
}

func normalizeQuickCapturePromotionTarget(target QuickCapturePromotionTarget) (QuickCapturePromotionTarget, error) {
	switch QuickCapturePromotionTarget(strings.ToLower(strings.TrimSpace(string(target)))) {
	case "", QuickCapturePromotionTargetCollection:
		return QuickCapturePromotionTargetCollection, nil
	case QuickCapturePromotionTargetWishlist:
		return QuickCapturePromotionTargetWishlist, nil
	default:
		return "", fmt.Errorf("invalid quick capture promotion target")
	}
}

func targetForPromotedCoin(coin *models.Coin) QuickCapturePromotionTarget {
	if coin != nil && coin.IsWishlist {
		return QuickCapturePromotionTargetWishlist
	}
	return QuickCapturePromotionTargetCollection
}

// buildCoinFromDraft constructs a Coin struct from a draft and promotion overrides.
func (s *QuickCaptureService) buildCoinFromDraft(draft *models.QuickCaptureDraft, overrides PromoteOverrides) *models.Coin {
	name := stringOverride(overrides.Name, draft.WorkingTitle)
	notes := stringOverride(overrides.Notes, draft.Notes)
	purchaseLocation := stringOverride(overrides.PurchaseLocation, draft.AcquisitionSource)

	category := models.Category(stringOverride(overrides.Category, ""))
	if category == "" {
		category = models.CategoryOther
	}
	material := models.Material(stringOverride(overrides.Material, ""))
	if material == "" {
		material = models.MaterialOther
	}
	era := models.Era(stringOverride(overrides.Era, draft.Era))

	var purchasePrice *float64
	if overrides.PurchasePriceSet {
		purchasePrice = overrides.PurchasePrice
	} else {
		purchasePrice = draft.PurchasePrice
	}

	return &models.Coin{
		UserID:           draft.UserID,
		Name:             name,
		Category:         category,
		Material:         material,
		Era:              era,
		Notes:            notes,
		PurchaseLocation: purchaseLocation,
		PurchasePrice:    purchasePrice,
		CurrentValue:     purchasePrice,
	}
}

func stringOverride(value *string, fallback string) string {
	if value != nil {
		return strings.TrimSpace(*value)
	}
	return strings.TrimSpace(fallback)
}

// validateCoinMinimumForPromotion checks that the built coin satisfies minimum
// create rules. Returns a map of fieldName → error message for any invalid
// fields. Era/category are checked against the same allow-list as direct coin
// create/update (via WithCoinValidation) when it's wired in; otherwise falls
// back to the built-in defaults only, so promotion can never be laxer than
// coin creation, only as strict as the caller allows.
func (s *QuickCaptureService) validateCoinMinimumForPromotion(coin *models.Coin) map[string]string {
	fieldErrors := map[string]string{}
	if strings.TrimSpace(coin.Name) == "" {
		fieldErrors["name"] = "Name is required"
	}
	if coin.Era != "" {
		if err := s.validateEraForPromotion(coin.Era); err != nil {
			fieldErrors["era"] = "Era must be one of your configured eras"
		}
	}
	if coin.Category != "" {
		if err := s.validateCategoryForPromotion(coin.Category); err != nil {
			fieldErrors["category"] = "Category must be one of your configured categories"
		}
	}
	return fieldErrors
}

func (s *QuickCaptureService) validateEraForPromotion(era models.Era) error {
	if s.coinSvc != nil {
		return s.coinSvc.validateCoinEra(era)
	}
	if _, ok := builtInCoinEras[era]; ok {
		return nil
	}
	return ErrCoinInvalidEra
}

func (s *QuickCaptureService) validateCategoryForPromotion(category models.Category) error {
	if s.coinSvc != nil {
		return s.coinSvc.validateCoinCategory(category)
	}
	if _, ok := builtInCoinCategories[category]; ok {
		return nil
	}
	return ErrCoinInvalidCategory
}

func (s *QuickCaptureService) saveDraftImageFile(draftID uint, fileData []byte, ext string, imageType models.ImageType) (string, error) {
	draftDir := filepath.Join(s.uploadDir, fmt.Sprintf("quick-capture-draft-%d", draftID))
	if err := os.MkdirAll(draftDir, 0755); err != nil {
		return "", ErrDirectoryCreation
	}
	filename := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), imageType, ext)
	filePath := filepath.Clean(filepath.Join(draftDir, filename))
	rel, err := filepath.Rel(filepath.Clean(draftDir), filePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", ErrFileSave
	}
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return "", ErrFileSave
	}
	return filepath.ToSlash(filepath.Join(fmt.Sprintf("quick-capture-draft-%d", draftID), filename)), nil
}
