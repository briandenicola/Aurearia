package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CoinReferenceHandler handles structured reference CRUD endpoints.
type CoinReferenceHandler struct {
	repo           *repository.CoinReferenceRepository
	svc            *services.CoinReferenceService
	migrationSvc   *services.ReferenceMigrationService
}

// NewCoinReferenceHandler creates a new CoinReferenceHandler.
func NewCoinReferenceHandler(
	repo *repository.CoinReferenceRepository,
	svc *services.CoinReferenceService,
	migrationSvc *services.ReferenceMigrationService,
) *CoinReferenceHandler {
	return &CoinReferenceHandler{repo: repo, svc: svc, migrationSvc: migrationSvc}
}

// List returns all references for a coin.
//
//	@Summary		List coin references
//	@Description	Returns all structured references for a user-owned coin.
//	@Tags			Coin References
//	@Produce		json
//	@Param			id	path		int	true	"Coin ID"
//	@Success		200	{array}		CoinReferenceDTO
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/references [get]
func (h *CoinReferenceHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if !h.repo.CoinExists(coinID, userID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	refs, err := h.repo.ListByCoin(coinID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list references"})
		return
	}
	c.JSON(http.StatusOK, refs)
}

// Create adds a structured reference to a coin.
//
//	@Summary		Create coin reference
//	@Description	Adds a structured catalog reference to a user-owned coin.
//	@Tags			Coin References
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int							true	"Coin ID"
//	@Param			body	body		CoinReferenceUpsertRequest	true	"Reference payload"
//	@Success		201		{object}	CoinReferenceDTO
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/references [post]
func (h *CoinReferenceHandler) Create(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if !h.repo.CoinExists(coinID, userID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	var ref models.CoinReference
	if err := c.ShouldBindJSON(&ref); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	ref.ID = 0
	ref.CoinID = coinID

	normalized, err := h.svc.NormalizeAndValidateOne(ref)
	if err != nil {
		h.respondReferenceValidationError(c, err)
		return
	}

	if err := h.repo.Create(&normalized); err != nil {
		if isUniqueConstraintError(err) {
			respondError(c, http.StatusBadRequest, services.ErrReferenceDuplicate.Error(), err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reference"})
		return
	}
	c.JSON(http.StatusCreated, normalized)
}

// Update modifies a structured reference for a coin.
//
//	@Summary		Update coin reference
//	@Description	Updates one structured reference on a user-owned coin.
//	@Tags			Coin References
//	@Accept			json
//	@Produce		json
//	@Param			id			path		int							true	"Coin ID"
//	@Param			referenceId	path		int							true	"Reference ID"
//	@Param			body		body		CoinReferenceUpsertRequest	true	"Reference payload"
//	@Success		200			{object}	CoinReferenceDTO
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/references/{referenceId} [put]
func (h *CoinReferenceHandler) Update(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	refID, ok := parseID(c, "referenceId")
	if !ok {
		return
	}

	existing, err := h.repo.GetByID(refID, coinID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Reference not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load reference"})
		return
	}

	var updates models.CoinReference
	if err := c.ShouldBindJSON(&updates); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	updates.CoinID = coinID
	normalized, err := h.svc.NormalizeAndValidateOne(updates)
	if err != nil {
		h.respondReferenceValidationError(c, err)
		return
	}

	if err := h.repo.Update(existing, map[string]interface{}{
		"catalog":   normalized.Catalog,
		"volume":    normalized.Volume,
		"number":    normalized.Number,
		"certainty": normalized.Certainty,
		"uri":       normalized.URI,
	}); err != nil {
		if isUniqueConstraintError(err) {
			respondError(c, http.StatusBadRequest, services.ErrReferenceDuplicate.Error(), err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update reference"})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// Delete removes a structured reference from a coin.
//
//	@Summary		Delete coin reference
//	@Description	Deletes one structured reference from a user-owned coin.
//	@Tags			Coin References
//	@Produce		json
//	@Param			id			path		int	true	"Coin ID"
//	@Param			referenceId	path		int	true	"Reference ID"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/references/{referenceId} [delete]
func (h *CoinReferenceHandler) Delete(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, ok := parseID(c, "id")
	if !ok {
		return
	}
	refID, ok := parseID(c, "referenceId")
	if !ok {
		return
	}

	rows, err := h.repo.Delete(refID, coinID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete reference"})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reference not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Reference deleted"})
}

func (h *CoinReferenceHandler) respondReferenceValidationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrReferenceCatalogRequired),
		errors.Is(err, services.ErrReferenceNumberRequired),
		errors.Is(err, services.ErrReferenceVolumeRequired),
		errors.Is(err, services.ErrReferenceUnknownCatalog),
		errors.Is(err, services.ErrReferenceDuplicate):
		respondError(c, http.StatusBadRequest, err.Error(), err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to validate reference", err)
	}
}

// MigrateLegacy migrates legacy rarity_rating fields to structured references for the authenticated user.
//
//	@Summary		Migrate legacy references
//	@Description	Migrates legacy rarity_rating text to structured CoinReference records for the authenticated user's coins. Non-destructive operation.
//	@Tags			Coin References
//	@Produce		json
//	@Success		200	{object}	MigrationResultDTO
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/references/migrate-legacy [post]
func (h *CoinReferenceHandler) MigrateLegacy(c *gin.Context) {
	userID := c.GetUint("userId")

	result, err := h.migrationSvc.MigrateLegacyReferences(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Migration failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}
