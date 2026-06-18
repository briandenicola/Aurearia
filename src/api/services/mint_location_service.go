package services

import (
	"errors"
	"strings"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"gorm.io/gorm"
)

const maxMintLocationTextLength = 128

var (
	ErrMintLocationNotFound      = errors.New("mint location not found")
	ErrMintLocationNameRequired  = errors.New("display name is required")
	ErrMintLocationDuplicate     = errors.New("a mint location with this display name already exists")
	ErrMintLocationLatInvalid    = errors.New("lat must be between -90 and 90")
	ErrMintLocationLngInvalid    = errors.New("lng must be between -180 and 180")
	ErrMintLocationAliasInvalid  = errors.New("aliases must not be blank")
	ErrMintLocationRegionInvalid = errors.New("region must be at most 128 characters")
	ErrMintLocationNameTooLong   = errors.New("display name must be at most 128 characters")
	ErrMintLocationAliasTooLong  = errors.New("aliases must be at most 128 characters")
)

// MintLocationInput contains editable mint-location fields.
type MintLocationInput struct {
	DisplayName string
	Lat         float64
	Lng         float64
	Region      string
	Aliases     []string
}

// MintLocationService manages global mint-location rules.
type MintLocationService struct {
	repo *repository.MintLocationRepository
}

// NewMintLocationService creates a new MintLocationService.
func NewMintLocationService(repo *repository.MintLocationRepository) *MintLocationService {
	return &MintLocationService{repo: repo}
}

// List returns all mint locations.
func (s *MintLocationService) List() ([]models.MintLocation, error) {
	return s.repo.List()
}

// Create validates and creates a mint location.
func (s *MintLocationService) Create(input MintLocationInput) (*models.MintLocation, error) {
	_, location, err := s.validateInput(input)
	if err != nil {
		return nil, err
	}

	if err := s.ensureLookupKeysAvailable(location, 0); err != nil {
		return nil, err
	}

	if err := s.repo.Create(location); err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrMintLocationDuplicate
		}
		return nil, err
	}
	return location, nil
}

// Update validates and updates a mint location.
func (s *MintLocationService) Update(id uint, input MintLocationInput) (*models.MintLocation, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMintLocationNotFound
		}
		return nil, err
	}

	_, location, err := s.validateInput(input)
	if err != nil {
		return nil, err
	}

	if err := s.ensureLookupKeysAvailable(location, existing.ID); err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"display_name":    location.DisplayName,
		"normalized_name": location.NormalizedName,
		"lat":             location.Lat,
		"lng":             location.Lng,
		"region":          location.Region,
		"aliases":         location.Aliases,
	}
	if err := s.repo.Update(existing, updates); err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrMintLocationDuplicate
		}
		return nil, err
	}
	return s.repo.FindByID(id)
}

// Delete removes a mint location.
func (s *MintLocationService) Delete(id uint) error {
	if err := s.repo.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMintLocationNotFound
		}
		return err
	}
	return nil
}

func (s *MintLocationService) validateInput(input MintLocationInput) (string, *models.MintLocation, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return "", nil, ErrMintLocationNameRequired
	}
	if len(displayName) > maxMintLocationTextLength {
		return "", nil, ErrMintLocationNameTooLong
	}
	if input.Lat < -90 || input.Lat > 90 {
		return "", nil, ErrMintLocationLatInvalid
	}
	if input.Lng < -180 || input.Lng > 180 {
		return "", nil, ErrMintLocationLngInvalid
	}
	region := strings.TrimSpace(input.Region)
	if len(region) > maxMintLocationTextLength {
		return "", nil, ErrMintLocationRegionInvalid
	}

	aliases, err := normalizeMintAliases(input.Aliases, displayName)
	if err != nil {
		return "", nil, err
	}
	normalized := models.NormalizeMintLocationName(displayName)
	location := &models.MintLocation{
		DisplayName:    displayName,
		NormalizedName: normalized,
		Lat:            input.Lat,
		Lng:            input.Lng,
		Region:         region,
		Aliases:        models.StringList(aliases),
	}
	return normalized, location, nil
}

func (s *MintLocationService) ensureLookupKeysAvailable(candidate *models.MintLocation, excludeID uint) error {
	candidateKeys := mintLocationLookupKeys(candidate)
	locations, err := s.repo.List()
	if err != nil {
		return err
	}
	for _, location := range locations {
		if location.ID == excludeID {
			continue
		}
		for key := range mintLocationLookupKeys(&location) {
			if candidateKeys[key] {
				return ErrMintLocationDuplicate
			}
		}
	}
	return nil
}

func mintLocationLookupKeys(location *models.MintLocation) map[string]bool {
	keys := make(map[string]bool, len(location.Aliases)+1)
	normalizedName := location.NormalizedName
	if normalizedName == "" {
		normalizedName = models.NormalizeMintLocationName(location.DisplayName)
	}
	if normalizedName != "" {
		keys[normalizedName] = true
	}
	for _, alias := range location.Aliases {
		normalizedAlias := models.NormalizeMintLocationName(alias)
		if normalizedAlias != "" {
			keys[normalizedAlias] = true
		}
	}
	return keys
}

func normalizeMintAliases(values []string, displayName string) ([]string, error) {
	aliases := make([]string, 0, len(values))
	seen := map[string]bool{models.NormalizeMintLocationName(displayName): true}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, ErrMintLocationAliasInvalid
		}
		if len(trimmed) > maxMintLocationTextLength {
			return nil, ErrMintLocationAliasTooLong
		}
		normalized := models.NormalizeMintLocationName(trimmed)
		if normalized == "" {
			return nil, ErrMintLocationAliasInvalid
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		aliases = append(aliases, trimmed)
	}
	return aliases, nil
}

func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
