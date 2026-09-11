package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// StorageLocationHandler handles storage-location HTTP requests.
type StorageLocationHandler struct {
	svc *services.StorageLocationService
}

// NewStorageLocationHandler creates a new StorageLocationHandler.
func NewStorageLocationHandler(svc *services.StorageLocationService) *StorageLocationHandler {
	return &StorageLocationHandler{svc: svc}
}

type storageLocationListResponse struct {
	StorageLocations []models.StorageLocation `json:"storageLocations"`
}

type StorageLocationCreateRequest struct {
	Name      string `json:"name" binding:"required" minLength:"1" maxLength:"100"`
	Type      string `json:"type" enums:"standard,tray" default:"standard"`
	Rows      *int   `json:"rows" minimum:"1" maximum:"20" extensions:"x-nullable"`
	Columns   *int   `json:"columns" minimum:"1" maximum:"20" extensions:"x-nullable"`
	SortOrder int    `json:"sortOrder"`
}

type StorageLocationUpdateRequest struct {
	Name      *string `json:"name" minLength:"1" maxLength:"100"`
	Rows      *int    `json:"rows" minimum:"1" maximum:"20" extensions:"x-nullable"`
	Columns   *int    `json:"columns" minimum:"1" maximum:"20" extensions:"x-nullable"`
	SortOrder *int    `json:"sortOrder"`
}

type storageTrayListResponse struct {
	Trays []repository.TrayAggregate `json:"trays"`
}

// List returns all storage locations for the authenticated user.
//
//	@Summary		List storage locations
//	@Description	Returns all storage locations belonging to the authenticated user.
//	@Tags			Storage Locations
//	@Produce		json
//	@Success		200	{object}	storageLocationListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/storage-locations [get]
func (h *StorageLocationHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")
	locations, err := h.svc.List(userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list storage locations", err)
		return
	}
	c.JSON(http.StatusOK, storageLocationListResponse{StorageLocations: locations})
}

// Create adds a new storage location for the authenticated user.
//
//	@Summary		Create storage location
//	@Description	Creates a user-owned Standard Location or Coin Tray. Type defaults to standard; trays require rows and columns from 1 through 20, with at most 400 slots.
//	@Tags			Storage Locations
//	@Accept			json
//	@Produce		json
//	@Param			body	body		StorageLocationCreateRequest	true	"Storage location data"
//	@Success		201		{object}	models.StorageLocation
//	@Failure		400		{object}	ValidationErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		409		{object}	StorageLocationConflictErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/storage-locations [post]
func (h *StorageLocationHandler) Create(c *gin.Context) {
	userID := c.GetUint("userId")
	var body StorageLocationCreateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Name is required", err)
		return
	}
	location, err := h.svc.CreateTyped(userID, services.StorageLocationWrite{
		Name: body.Name, Type: body.Type, Rows: body.Rows, Columns: body.Columns, SortOrder: body.SortOrder,
	})
	if err != nil {
		if handleStorageLocationError(c, err, 0) {
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to create storage location", err)
		return
	}
	c.JSON(http.StatusCreated, location)
}

// Update modifies a storage location.
//
//	@Summary		Update storage location
//	@Description	Renames a user-owned location or resizes an empty tray. Nullable dimensions are validated against the persisted location type.
//	@Tags			Storage Locations
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Storage location ID"
//	@Param			body	body		StorageLocationUpdateRequest	true	"Storage location updates"
//	@Success		200		{object}	models.StorageLocation
//	@Failure		400		{object}	ValidationErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		409		{object}	StorageLocationConflictErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/storage-locations/{id} [put]
func (h *StorageLocationHandler) Update(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid storage location ID"})
		return
	}
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(bodyBytes, &raw)
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	var body StorageLocationUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	_, rowsSet := raw["rows"]
	_, columnsSet := raw["columns"]
	location, err := h.svc.UpdateTyped(uint(id), userID, services.StorageLocationUpdate{
		Name: body.Name, Rows: body.Rows, Columns: body.Columns,
		RowsSet: rowsSet, ColumnsSet: columnsSet, SortOrder: body.SortOrder,
	})
	if err != nil {
		if handleStorageLocationError(c, err, 0) {
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to update storage location", err)
		return
	}
	c.JSON(http.StatusOK, location)
}

// Delete removes an unused storage location.
//
//	@Summary		Delete storage location
//	@Description	Deletes a storage location. Deletion is blocked while any coins still reference it.
//	@Tags			Storage Locations
//	@Produce		json
//	@Param			id	path		int	true	"Storage location ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	LocationReferencedErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/storage-locations/{id} [delete]
func (h *StorageLocationHandler) Delete(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid storage location ID"})
		return
	}
	count, err := h.svc.Delete(uint(id), userID)
	if err != nil {
		if handleStorageLocationError(c, err, count) {
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to delete storage location", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Storage location deleted"})
}

func handleStorageLocationError(c *gin.Context, err error, count int64) bool {
	switch {
	case errors.Is(err, services.ErrStorageLocationNameInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": services.ErrStorageLocationNameInvalid.Error(), "code": "validation_error", "field": "name"})
		return true
	case errors.Is(err, services.ErrStorageLocationTypeInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "validation_error", "field": "type"})
		return true
	case errors.Is(err, services.ErrStorageLocationDimensions):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "validation_error", "field": "rows"})
		return true
	case errors.Is(err, services.ErrStorageLocationDuplicate):
		c.JSON(http.StatusConflict, gin.H{
			"error": services.ErrStorageLocationDuplicate.Error(), "code": "duplicate_location",
			"message": "Choose a different storage location name.",
		})
		return true
	case errors.Is(err, services.ErrStorageLocationLimit):
		c.JSON(http.StatusConflict, gin.H{
			"error": services.ErrStorageLocationLimit.Error(), "code": "location_limit",
			"message": services.ErrStorageLocationLimit.Error(),
		})
		return true
	case errors.Is(err, services.ErrStorageLocationNotFound):
		respondError(c, http.StatusNotFound, "Storage location not found", err)
		return true
	case errors.Is(err, services.ErrStorageLocationInUse):
		message := fmt.Sprintf("Storage location is used by %d coin(s); reassign those coins before deleting it", count)
		c.JSON(http.StatusConflict, gin.H{"error": message, "message": message, "code": "location_referenced", "count": count})
		return true
	case errors.Is(err, services.ErrTrayOccupied):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "message": err.Error(), "code": "tray_occupied"})
		return true
	default:
		return false
	}
}

// Occupancy returns privacy-preserving occupied slot numbers for an owned tray.
//
//	@Summary		Get tray occupancy
//	@Description	Returns one-based occupied slot numbers for an owned Coin Tray. coinId may identify the current owned coin whose slot remains selectable.
//	@Tags			Storage Locations
//	@Produce		json
//	@Param			id path int true "Storage location ID" minimum(1)
//	@Param			coinId query int false "Current owned coin ID" minimum(1)
//	@Success		200 {object} repository.TrayOccupancy
//	@Failure		400 {object} ValidationErrorResponse
//	@Failure		401 {object} ErrorResponse
//	@Failure		404 {object} ErrorResponse
//	@Failure		500 {object} ErrorResponse
//	@Security		BearerAuth
//	@Router			/storage-locations/{id}/occupancy [get]
func (h *StorageLocationHandler) Occupancy(c *gin.Context) {
	locationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || locationID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid storage location ID", "code": "validation_error"})
		return
	}
	var coinID *uint
	if raw := c.Query("coinId"); raw != "" {
		value, parseErr := strconv.ParseUint(raw, 10, 32)
		if parseErr != nil || value == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID", "code": "validation_error", "field": "coinId"})
			return
		}
		parsed := uint(value)
		coinID = &parsed
	}
	result, err := h.svc.Occupancy(uint(locationID), c.GetUint("userId"), coinID)
	if err != nil {
		if errors.Is(err, services.ErrStorageLocationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Storage location not found"})
			return
		}
		respondError(c, http.StatusInternalServerError, "Failed to read tray occupancy", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListTrays returns every owned tray with minimal positioned coin data.
//
//	@Summary		List physical storage trays
//	@Tags			Storage Locations
//	@Produce		json
//	@Success		200 {object} storageTrayListResponse
//	@Failure		401 {object} ErrorResponse
//	@Failure		500 {object} ErrorResponse
//	@Security		BearerAuth
//	@Router			/storage-trays [get]
func (h *StorageLocationHandler) ListTrays(c *gin.Context) {
	trays, err := h.svc.ListTrayAggregates(c.GetUint("userId"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list storage trays", err)
		return
	}
	c.JSON(http.StatusOK, storageTrayListResponse{Trays: trays})
}
