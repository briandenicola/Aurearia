package services

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
)

const (
	maxMintLocationTextLength = 128
	maxNomismaQueryLength     = 200
	maxNomismaLabelLength     = 256
	nomismaAuthorityHost      = "nomisma.org"
)

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
	ErrMintLocationInUse         = errors.New("mint location is in use")

	ErrMintLocationNomismaQueryInvalid = errors.New("query must be non-blank and at most 200 characters")
	ErrMintLocationNomismaURIInvalid   = errors.New("uri must be an absolute http(s) URL on the nomisma.org host")
	ErrMintLocationNomismaLabelInvalid = errors.New("label is required and must be at most 256 characters")
)

// MintLocationInput contains editable mint-location fields.
type MintLocationInput struct {
	DisplayName string
	Lat         float64
	Lng         float64
	Region      string
	Aliases     []string
}

// MintLocationService manages global (admin-curated) and private
// (per-user) mint-location rules.
type MintLocationService struct {
	repo         *repository.MintLocationRepository
	nomisma      NomismaClient
	nomismaCache *NomismaCache
}

// NewMintLocationService creates a new MintLocationService.
func NewMintLocationService(repo *repository.MintLocationRepository) *MintLocationService {
	return &MintLocationService{repo: repo}
}

// WithNomisma enables the Nomisma.org authority-linking search/link/unlink
// methods, wiring in the typed client and its bounded search cache. Mirrors
// MintLocationHandler.WithGeocoding's optional-dependency shape.
func (s *MintLocationService) WithNomisma(client NomismaClient, cache *NomismaCache) *MintLocationService {
	s.nomisma = client
	s.nomismaCache = cache
	return s
}

// NomismaSearchOutcome is the typed result of a Nomisma search,
// distinguishing ok/no_match/unavailable so callers never string-match an
// error.
type NomismaSearchOutcome struct {
	Status     NomismaSearchStatus
	Candidates []NomismaCandidate
}

// List returns every mint location a user may pick from: the global
// (admin-curated) list plus that user's own private ones.
func (s *MintLocationService) List(userID uint) ([]models.MintLocation, error) {
	return s.repo.ListVisibleTo(userID)
}

// CreateGlobal validates and creates a global mint location (admin only).
func (s *MintLocationService) CreateGlobal(input MintLocationInput) (*models.MintLocation, error) {
	_, location, err := s.validateInput(input)
	if err != nil {
		return nil, err
	}

	if err := s.ensureLookupKeysAvailable(location, 0, nil); err != nil {
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

// CreatePrivate validates and creates a mint location private to userID.
func (s *MintLocationService) CreatePrivate(userID uint, input MintLocationInput) (*models.MintLocation, error) {
	_, location, err := s.validateInput(input)
	if err != nil {
		return nil, err
	}
	location.UserID = &userID

	if err := s.ensureLookupKeysAvailable(location, 0, &userID); err != nil {
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

// UpdateGlobal validates and updates a global mint location (admin only).
// Rejects the request if id refers to a private mint.
func (s *MintLocationService) UpdateGlobal(id uint, input MintLocationInput) (*models.MintLocation, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrMintLocationNotFound
		}
		return nil, err
	}
	if existing.UserID != nil {
		return nil, ErrMintLocationNotFound
	}

	return s.update(existing, input, nil)
}

// UpdatePrivate validates and updates a private mint location owned by userID.
func (s *MintLocationService) UpdatePrivate(id, userID uint, input MintLocationInput) (*models.MintLocation, error) {
	existing, err := s.repo.FindOwnedByID(id, userID)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrMintLocationNotFound
		}
		return nil, err
	}

	return s.update(existing, input, &userID)
}

func (s *MintLocationService) update(existing *models.MintLocation, input MintLocationInput, ownerUserID *uint) (*models.MintLocation, error) {
	_, location, err := s.validateInput(input)
	if err != nil {
		return nil, err
	}

	if err := s.ensureLookupKeysAvailable(location, existing.ID, ownerUserID); err != nil {
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
	return s.repo.FindByID(existing.ID)
}

// findGlobalMintLocation resolves id to a global (UserID == nil) mint
// location, or ErrMintLocationNotFound otherwise - reused by
// SearchNomisma/LinkNomismaGlobal/UnlinkNomismaGlobal so a private mint
// never has a reachable code path into Nomisma (FR-006, User Story 4).
func (s *MintLocationService) findGlobalMintLocation(id uint) (*models.MintLocation, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return nil, ErrMintLocationNotFound
		}
		return nil, err
	}
	if existing.UserID != nil {
		return nil, ErrMintLocationNotFound
	}
	return existing, nil
}

// SearchNomisma looks up Nomisma.org authority candidates for a global mint
// location's admin-typed query. Never returns a hard error for an upstream
// Nomisma failure - that outcome is represented as NomismaSearchUnavailable
// so mint/coin CRUD stays fully usable (FR-007).
func (s *MintLocationService) SearchNomisma(id uint, query string) (NomismaSearchOutcome, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" || len([]rune(trimmed)) > maxNomismaQueryLength {
		return NomismaSearchOutcome{}, ErrMintLocationNomismaQueryInvalid
	}
	if _, err := s.findGlobalMintLocation(id); err != nil {
		return NomismaSearchOutcome{}, err
	}
	if s.nomisma == nil {
		return NomismaSearchOutcome{Status: NomismaSearchUnavailable, Candidates: []NomismaCandidate{}}, nil
	}

	if s.nomismaCache != nil {
		if status, candidates, ok := s.nomismaCache.Get(trimmed); ok {
			return NomismaSearchOutcome{Status: status, Candidates: candidates}, nil
		}
	}

	candidates, kind, err := s.nomisma.Search(context.Background(), trimmed, 0)
	if err != nil || kind == NomismaErrorUnavailable || kind == NomismaErrorInvalidResponse {
		// Never cached - a transient outage must not get "stuck" for the TTL.
		return NomismaSearchOutcome{Status: NomismaSearchUnavailable, Candidates: []NomismaCandidate{}}, nil
	}

	status := NomismaSearchOK
	if kind == NomismaErrorNoMatch || len(candidates) == 0 {
		status = NomismaSearchNoMatch
		candidates = []NomismaCandidate{}
	}
	if s.nomismaCache != nil {
		s.nomismaCache.Set(trimmed, status, candidates)
	}
	return NomismaSearchOutcome{Status: status, Candidates: candidates}, nil
}

// LinkNomismaGlobal confirms exactly one Nomisma candidate for a global
// mint location, replacing any existing link. Sets NomismaURI/NomismaLabel/
// NomismaLinkedAt together in one repo.Update call touching only those
// three columns - DisplayName/Lat/Lng/Region/Aliases are never part of this
// update's column set (FR-005).
func (s *MintLocationService) LinkNomismaGlobal(id uint, uri, label string) (*models.MintLocation, error) {
	existing, err := s.findGlobalMintLocation(id)
	if err != nil {
		return nil, err
	}
	if !isValidNomismaURI(uri) {
		return nil, ErrMintLocationNomismaURIInvalid
	}
	trimmedLabel := strings.TrimSpace(label)
	if trimmedLabel == "" || len([]rune(trimmedLabel)) > maxNomismaLabelLength {
		return nil, ErrMintLocationNomismaLabelInvalid
	}

	now := time.Now().UTC()
	updates := map[string]interface{}{
		"nomisma_uri":       uri,
		"nomisma_label":     trimmedLabel,
		"nomisma_linked_at": now,
	}
	if err := s.repo.Update(existing, updates); err != nil {
		return nil, err
	}
	return s.repo.FindByID(existing.ID)
}

// UnlinkNomismaGlobal clears an existing Nomisma link on a global mint
// location. Idempotent - unlinking an already-unlinked mint is a no-op
// success, not an error.
func (s *MintLocationService) UnlinkNomismaGlobal(id uint) (*models.MintLocation, error) {
	existing, err := s.findGlobalMintLocation(id)
	if err != nil {
		return nil, err
	}
	if existing.NomismaURI == nil {
		return existing, nil
	}

	updates := map[string]interface{}{
		"nomisma_uri":       nil,
		"nomisma_label":     "",
		"nomisma_linked_at": nil,
	}
	if err := s.repo.Update(existing, updates); err != nil {
		return nil, err
	}
	return s.repo.FindByID(existing.ID)
}

func isValidNomismaURI(uri string) bool {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return false
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	host := parsed.Hostname()
	return host == nomismaAuthorityHost || strings.HasSuffix(host, "."+nomismaAuthorityHost)
}

// DeleteGlobal removes a global mint location (admin only). Rejects the
// request if id refers to a private mint, or if any coin still references it.
func (s *MintLocationService) DeleteGlobal(id uint) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		if repository.IsRecordNotFound(err) {
			return ErrMintLocationNotFound
		}
		return err
	}
	if existing.UserID != nil {
		return ErrMintLocationNotFound
	}
	return s.deleteIfUnused(id)
}

// DeletePrivate removes a private mint location owned by userID, unless
// any coin still references it.
func (s *MintLocationService) DeletePrivate(id, userID uint) error {
	if _, err := s.repo.FindOwnedByID(id, userID); err != nil {
		if repository.IsRecordNotFound(err) {
			return ErrMintLocationNotFound
		}
		return err
	}
	return s.deleteIfUnused(id)
}

func (s *MintLocationService) deleteIfUnused(id uint) error {
	count, err := s.repo.CountCoinsUsing(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrMintLocationInUse
	}
	if err := s.repo.Delete(id); err != nil {
		if repository.IsRecordNotFound(err) {
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

// ensureLookupKeysAvailable checks the candidate's name/aliases don't
// collide with an existing mint location. ownerUserID nil scopes the check
// to global entries only (admin create/update); non-nil scopes it to
// global entries plus that user's own private ones (self-service
// create/update) - never against another user's private entries.
func (s *MintLocationService) ensureLookupKeysAvailable(candidate *models.MintLocation, excludeID uint, ownerUserID *uint) error {
	candidateKeys := mintLocationLookupKeys(candidate)
	var locations []models.MintLocation
	var err error
	if ownerUserID != nil {
		locations, err = s.repo.ListVisibleTo(*ownerUserID)
	} else {
		locations, err = s.repo.ListGlobal()
	}
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
