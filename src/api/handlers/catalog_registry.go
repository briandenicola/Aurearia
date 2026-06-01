package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// CatalogRegistryHandler handles catalog registry HTTP requests.
type CatalogRegistryHandler struct {
	svc *services.CatalogRegistryService
}

// NewCatalogRegistryHandler creates a new CatalogRegistryHandler.
func NewCatalogRegistryHandler(svc *services.CatalogRegistryService) *CatalogRegistryHandler {
	return &CatalogRegistryHandler{svc: svc}
}

type catalogListResponse struct {
	Catalogs []models.CatalogRegistry `json:"catalogs"`
}

type catalogCreateRequest struct {
	Catalog        string      `json:"catalog" binding:"required"`
	DisplayName    string      `json:"displayName" binding:"required"`
	Era            models.Era  `json:"era" binding:"required"`
	VolumeRequired bool        `json:"volumeRequired"`
}

type catalogUpdateRequest struct {
	Catalog        string      `json:"catalog" binding:"required"`
	DisplayName    string      `json:"displayName" binding:"required"`
	Era            models.Era  `json:"era" binding:"required"`
	VolumeRequired bool        `json:"volumeRequired"`
}

// List returns all catalog registry entries.
//
//	@Summary		List catalogs
//	@Description	Returns all catalog registry entries.
//	@Tags			Catalogs
//	@Produce		json
//	@Success		200	{object}	catalogListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalogs [get]
func (h *CatalogRegistryHandler) List(c *gin.Context) {
	catalogs, err := h.svc.List()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list catalogs", err)
		return
	}
	c.JSON(http.StatusOK, catalogListResponse{Catalogs: catalogs})
}

// Create adds a new catalog registry entry (admin only).
//
//	@Summary		Create catalog
//	@Description	Creates a new catalog registry entry.
//	@Tags			Catalogs
//	@Accept			json
//	@Produce		json
//	@Param			body	body		catalogCreateRequest	true	"Catalog data"
//	@Success		201		{object}	models.CatalogRegistry
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/catalogs [post]
func (h *CatalogRegistryHandler) Create(c *gin.Context) {
	var body catalogCreateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	entry := models.CatalogRegistry{
		Catalog:        body.Catalog,
		DisplayName:    body.DisplayName,
		Era:            body.Era,
		VolumeRequired: body.VolumeRequired,
	}

	created, err := h.svc.Create(entry)
	if err != nil {
		h.handleCatalogError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// Update modifies an existing catalog registry entry (admin only).
//
//	@Summary		Update catalog
//	@Description	Updates an existing catalog registry entry.
//	@Tags			Catalogs
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Catalog ID"
//	@Param			body	body		catalogUpdateRequest	true	"Catalog data"
//	@Success		200		{object}	models.CatalogRegistry
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/catalogs/{id} [put]
func (h *CatalogRegistryHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid catalog ID", err)
		return
	}

	var body catalogUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	entry := models.CatalogRegistry{
		Catalog:        body.Catalog,
		DisplayName:    body.DisplayName,
		Era:            body.Era,
		VolumeRequired: body.VolumeRequired,
	}

	updated, err := h.svc.Update(uint(id), entry)
	if err != nil {
		h.handleCatalogError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// Delete removes a catalog registry entry (admin only).
//
//	@Summary		Delete catalog
//	@Description	Deletes a catalog registry entry if not in use.
//	@Tags			Catalogs
//	@Produce		json
//	@Param			id	path		int	true	"Catalog ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/catalogs/{id} [delete]
func (h *CatalogRegistryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid catalog ID", err)
		return
	}

	if err := h.svc.Delete(uint(id)); err != nil {
		h.handleCatalogError(c, err)
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Catalog deleted successfully"})
}

func (h *CatalogRegistryHandler) handleCatalogError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrCatalogNotFound):
		respondError(c, http.StatusNotFound, err.Error(), err)
	case errors.Is(err, services.ErrCatalogDuplicate):
		respondError(c, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, services.ErrCatalogInUse):
		respondError(c, http.StatusConflict, err.Error(), err)
	case errors.Is(err, services.ErrCatalogInvalidEra):
		respondError(c, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, services.ErrCatalogCodeRequired):
		respondError(c, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, services.ErrCatalogNameRequired):
		respondError(c, http.StatusBadRequest, err.Error(), err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to process catalog request", err)
	}
}
