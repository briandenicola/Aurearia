package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// MintLocationHandler handles mint-location HTTP requests.
type MintLocationHandler struct {
	svc *services.MintLocationService
}

// NewMintLocationHandler creates a new MintLocationHandler.
func NewMintLocationHandler(svc *services.MintLocationService) *MintLocationHandler {
	return &MintLocationHandler{svc: svc}
}

type mintLocationListResponse struct {
	MintLocations []models.MintLocation `json:"mintLocations"`
}

type mintLocationRequest struct {
	DisplayName string   `json:"displayName" binding:"required"`
	Lat         *float64 `json:"lat" binding:"required"`
	Lng         *float64 `json:"lng" binding:"required"`
	Region      string   `json:"region"`
	Aliases     []string `json:"aliases"`
}

// List returns all mint locations.
//
//	@Summary		List mint locations
//	@Description	Returns all global mint locations for authenticated users.
//	@Tags			Mint Locations
//	@Produce		json
//	@Success		200	{object}	mintLocationListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/mint-locations [get]
func (h *MintLocationHandler) List(c *gin.Context) {
	locations, err := h.svc.List()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list mint locations", err)
		return
	}
	c.JSON(http.StatusOK, mintLocationListResponse{MintLocations: locations})
}

// Create adds a new mint location (admin only).
//
//	@Summary		Create mint location
//	@Description	Creates a global mint location. Admin only.
//	@Tags			Mint Locations
//	@Accept			json
//	@Produce		json
//	@Param			body	body		mintLocationRequest	true	"Mint location data"
//	@Success		201		{object}	models.MintLocation
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/mint-locations [post]
func (h *MintLocationHandler) Create(c *gin.Context) {
	input, ok := bindMintLocationRequest(c)
	if !ok {
		return
	}
	location, err := h.svc.Create(input)
	if err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, location)
}

// Update modifies a mint location (admin only).
//
//	@Summary		Update mint location
//	@Description	Updates a global mint location. Admin only.
//	@Tags			Mint Locations
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Mint location ID"
//	@Param			body	body		mintLocationRequest	true	"Mint location data"
//	@Success		200		{object}	models.MintLocation
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/mint-locations/{id} [put]
func (h *MintLocationHandler) Update(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	input, ok := bindMintLocationRequest(c)
	if !ok {
		return
	}
	location, err := h.svc.Update(id, input)
	if err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusOK, location)
}

// Delete removes a mint location (admin only).
//
//	@Summary		Delete mint location
//	@Description	Deletes a global mint location. Coins with no remaining matching mint location appear as unattributed in the map. Admin only.
//	@Tags			Mint Locations
//	@Produce		json
//	@Param			id	path		int	true	"Mint location ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/mint-locations/{id} [delete]
func (h *MintLocationHandler) Delete(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: "Mint location deleted successfully"})
}

func bindMintLocationRequest(c *gin.Context) (services.MintLocationInput, bool) {
	var body mintLocationRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid mint location request", err)
		return services.MintLocationInput{}, false
	}
	return services.MintLocationInput{
		DisplayName: body.DisplayName,
		Lat:         *body.Lat,
		Lng:         *body.Lng,
		Region:      body.Region,
		Aliases:     body.Aliases,
	}, true
}

func handleMintLocationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrMintLocationNotFound):
		respondError(c, http.StatusNotFound, "Mint location not found", err)
	case errors.Is(err, services.ErrMintLocationDuplicate):
		respondError(c, http.StatusConflict, services.ErrMintLocationDuplicate.Error(), err)
	case errors.Is(err, services.ErrMintLocationNameRequired),
		errors.Is(err, services.ErrMintLocationNameTooLong),
		errors.Is(err, services.ErrMintLocationLatInvalid),
		errors.Is(err, services.ErrMintLocationLngInvalid),
		errors.Is(err, services.ErrMintLocationAliasInvalid),
		errors.Is(err, services.ErrMintLocationAliasTooLong),
		errors.Is(err, services.ErrMintLocationRegionInvalid):
		respondError(c, http.StatusBadRequest, err.Error(), err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to process mint location request", err)
	}
}
