package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/models"
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

type storageLocationCreateRequest struct {
	Name      string `json:"name" binding:"required"`
	SortOrder int    `json:"sortOrder"`
}

type storageLocationUpdateRequest struct {
	Name      *string `json:"name"`
	SortOrder *int    `json:"sortOrder"`
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
//	@Description	Creates a user-owned storage location.
//	@Tags			Storage Locations
//	@Accept			json
//	@Produce		json
//	@Param			body	body		storageLocationCreateRequest	true	"Storage location data"
//	@Success		201		{object}	models.StorageLocation
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/storage-locations [post]
func (h *StorageLocationHandler) Create(c *gin.Context) {
	userID := c.GetUint("userId")
	var body storageLocationCreateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Name is required", err)
		return
	}
	location, err := h.svc.Create(userID, body.Name, body.SortOrder)
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
//	@Description	Updates a user-owned storage location.
//	@Tags			Storage Locations
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Storage location ID"
//	@Param			body	body		storageLocationUpdateRequest	true	"Storage location updates"
//	@Success		200		{object}	models.StorageLocation
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
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
	var body storageLocationUpdateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	location, err := h.svc.Update(uint(id), userID, body.Name, body.SortOrder)
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
//	@Failure		409	{object}	ErrorResponse
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
		respondError(c, http.StatusBadRequest, services.ErrStorageLocationNameInvalid.Error(), err)
		return true
	case errors.Is(err, services.ErrStorageLocationDuplicate):
		respondError(c, http.StatusConflict, services.ErrStorageLocationDuplicate.Error(), err)
		return true
	case errors.Is(err, services.ErrStorageLocationLimit):
		respondError(c, http.StatusConflict, services.ErrStorageLocationLimit.Error(), err)
		return true
	case errors.Is(err, services.ErrStorageLocationNotFound):
		respondError(c, http.StatusNotFound, "Storage location not found", err)
		return true
	case errors.Is(err, services.ErrStorageLocationInUse):
		message := fmt.Sprintf("Storage location is used by %d coin(s); reassign those coins before deleting it", count)
		respondError(c, http.StatusConflict, message, err)
		return true
	default:
		return false
	}
}
