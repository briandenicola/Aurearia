package handlers

import (
	"errors"
	"net/http"

	"github.com/briandenicola/ancient-coins-api/models"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
)

// MintLocationHandler handles mint-location HTTP requests, covering both
// admin-curated global entries and self-service private entries.
type MintLocationHandler struct {
	svc     *services.MintLocationService
	geocode *services.GeocodeService
}

// NewMintLocationHandler creates a new MintLocationHandler.
func NewMintLocationHandler(svc *services.MintLocationService) *MintLocationHandler {
	return &MintLocationHandler{svc: svc}
}

// WithGeocoding enables the geocode-candidates endpoint used by the
// create-mint flow to suggest coordinates for a typed name.
func (h *MintLocationHandler) WithGeocoding(geocode *services.GeocodeService) *MintLocationHandler {
	h.geocode = geocode
	return h
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

// List returns every mint location visible to the authenticated user: the
// global (admin-curated) list plus that user's own private ones.
//
//	@Summary		List mint locations
//	@Description	Returns global mint locations plus the authenticated user's own private ones.
//	@Tags			Mint Locations
//	@Produce		json
//	@Success		200	{object}	mintLocationListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/mint-locations [get]
func (h *MintLocationHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")
	locations, err := h.svc.List(userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to list mint locations", err)
		return
	}
	c.JSON(http.StatusOK, mintLocationListResponse{MintLocations: locations})
}

// Create adds a new global mint location (admin only).
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
	location, err := h.svc.CreateGlobal(input)
	if err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, location)
}

// Update modifies a global mint location (admin only).
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
	location, err := h.svc.UpdateGlobal(id, input)
	if err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusOK, location)
}

// Delete removes a global mint location (admin only).
//
//	@Summary		Delete mint location
//	@Description	Deletes a global mint location. Fails if any coin still references it. Admin only.
//	@Tags			Mint Locations
//	@Produce		json
//	@Param			id	path		int	true	"Mint location ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/mint-locations/{id} [delete]
func (h *MintLocationHandler) Delete(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteGlobal(id); err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: "Mint location deleted successfully"})
}

// CreatePrivate adds a new mint location private to the authenticated user.
//
//	@Summary		Create a private mint location
//	@Description	Creates a mint location visible only to the authenticated user.
//	@Tags			Mint Locations
//	@Accept			json
//	@Produce		json
//	@Param			body	body		mintLocationRequest	true	"Mint location data"
//	@Success		201		{object}	models.MintLocation
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/mint-locations [post]
func (h *MintLocationHandler) CreatePrivate(c *gin.Context) {
	userID := c.GetUint("userId")
	input, ok := bindMintLocationRequest(c)
	if !ok {
		return
	}
	location, err := h.svc.CreatePrivate(userID, input)
	if err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, location)
}

// UpdatePrivate modifies a mint location owned by the authenticated user.
//
//	@Summary		Update a private mint location
//	@Description	Updates a mint location owned by the authenticated user.
//	@Tags			Mint Locations
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Mint location ID"
//	@Param			body	body		mintLocationRequest	true	"Mint location data"
//	@Success		200		{object}	models.MintLocation
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/mint-locations/{id} [put]
func (h *MintLocationHandler) UpdatePrivate(c *gin.Context) {
	userID := c.GetUint("userId")
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	input, ok := bindMintLocationRequest(c)
	if !ok {
		return
	}
	location, err := h.svc.UpdatePrivate(id, userID, input)
	if err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusOK, location)
}

// DeletePrivate removes a mint location owned by the authenticated user.
//
//	@Summary		Delete a private mint location
//	@Description	Deletes a mint location owned by the authenticated user. Fails if any of their coins still reference it.
//	@Tags			Mint Locations
//	@Produce		json
//	@Param			id	path		int	true	"Mint location ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		409	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/mint-locations/{id} [delete]
func (h *MintLocationHandler) DeletePrivate(c *gin.Context) {
	userID := c.GetUint("userId")
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeletePrivate(id, userID); err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: "Mint location deleted successfully"})
}

type geocodeMintResponse struct {
	Candidates []services.GeocodeCandidate `json:"candidates"`
}

// Geocode looks up coordinate candidates for a typed place name, for the
// create-mint flow. Never errors on a network/lookup failure or a query
// with no matches - it responds with an empty candidate list either way, so
// the frontend can fall back to manual pin placement without a dead end.
//
//	@Summary		Geocode a mint name
//	@Description	Looks up coordinate candidates for a place name via OpenStreetMap Nominatim. Only the typed name is sent - no coin, collection, or account data.
//	@Tags			Mint Locations
//	@Produce		json
//	@Param			query	query		string	true	"Place name to look up"
//	@Success		200		{object}	geocodeMintResponse
//	@Failure		401		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/mint-locations/geocode [get]
func (h *MintLocationHandler) Geocode(c *gin.Context) {
	query := c.Query("query")
	if h.geocode == nil {
		c.JSON(http.StatusOK, geocodeMintResponse{Candidates: []services.GeocodeCandidate{}})
		return
	}
	candidates, err := h.geocode.Search(query)
	if err != nil {
		candidates = []services.GeocodeCandidate{}
	}
	c.JSON(http.StatusOK, geocodeMintResponse{Candidates: candidates})
}

type nomismaSearchResponse struct {
	Status     services.NomismaSearchStatus `json:"status"`
	Candidates []services.NomismaCandidate  `json:"candidates"`
}

// SearchNomisma looks up Nomisma.org authority candidates for a global mint
// location, for admin review before an explicit confirm/link. Never returns
// a 5xx for an upstream Nomisma failure - that's surfaced as a 200 with
// status "unavailable" so the admin panel and mint/coin CRUD stay usable.
//
//	@Summary		Search Nomisma candidates for a global mint location
//	@Description	Searches Nomisma.org's reconciliation service for authority candidates matching the given query. Admin only. Never fails for an upstream Nomisma outage - surfaced as status "unavailable".
//	@Tags			Mint Locations
//	@Produce		json
//	@Param			id		path		int		true	"Mint location ID"
//	@Param			query	query		string	true	"Text to search Nomisma for"
//	@Success		200		{object}	nomismaSearchResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/mint-locations/{id}/nomisma/search [get]
func (h *MintLocationHandler) SearchNomisma(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	query := c.Query("query")
	outcome, err := h.svc.SearchNomisma(id, query)
	if err != nil {
		handleMintLocationError(c, err)
		return
	}
	candidates := outcome.Candidates
	if candidates == nil {
		candidates = []services.NomismaCandidate{}
	}
	c.JSON(http.StatusOK, nomismaSearchResponse{Status: outcome.Status, Candidates: candidates})
}

type nomismaLinkRequest struct {
	URI   string `json:"uri" binding:"required"`
	Label string `json:"label" binding:"required"`
}

// LinkNomisma confirms exactly one Nomisma candidate for a global mint
// location, replacing any existing link.
//
//	@Summary		Link a global mint location to a Nomisma authority concept
//	@Description	Confirms a Nomisma.org candidate for a global mint location, persisting its URI/label/timestamp. Admin only. Replaces any existing link.
//	@Tags			Mint Locations
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Mint location ID"
//	@Param			body	body		nomismaLinkRequest	true	"Confirmed Nomisma candidate"
//	@Success		200		{object}	models.MintLocation
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/mint-locations/{id}/nomisma [post]
func (h *MintLocationHandler) LinkNomisma(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var body nomismaLinkRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid Nomisma link request", err)
		return
	}
	location, err := h.svc.LinkNomismaGlobal(id, body.URI, body.Label)
	if err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusOK, location)
}

// UnlinkNomisma removes a confirmed Nomisma link from a global mint
// location. Idempotent - unlinking an already-unlinked mint is a 200
// success, not an error.
//
//	@Summary		Remove a Nomisma authority link from a global mint location
//	@Description	Clears a global mint location's Nomisma URI/label/timestamp. Admin only. Idempotent.
//	@Tags			Mint Locations
//	@Produce		json
//	@Param			id	path		int	true	"Mint location ID"
//	@Success		200	{object}	MessageResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/mint-locations/{id}/nomisma [delete]
func (h *MintLocationHandler) UnlinkNomisma(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if _, err := h.svc.UnlinkNomismaGlobal(id); err != nil {
		handleMintLocationError(c, err)
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: "Nomisma link removed"})
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
	case errors.Is(err, services.ErrMintLocationInUse):
		respondError(c, http.StatusConflict, services.ErrMintLocationInUse.Error(), err)
	case errors.Is(err, services.ErrMintLocationNameRequired),
		errors.Is(err, services.ErrMintLocationNameTooLong),
		errors.Is(err, services.ErrMintLocationLatInvalid),
		errors.Is(err, services.ErrMintLocationLngInvalid),
		errors.Is(err, services.ErrMintLocationAliasInvalid),
		errors.Is(err, services.ErrMintLocationAliasTooLong),
		errors.Is(err, services.ErrMintLocationRegionInvalid),
		errors.Is(err, services.ErrMintLocationNomismaQueryInvalid),
		errors.Is(err, services.ErrMintLocationNomismaURIInvalid),
		errors.Is(err, services.ErrMintLocationNomismaLabelInvalid):
		respondError(c, http.StatusBadRequest, err.Error(), err)
	default:
		respondError(c, http.StatusInternalServerError, "Failed to process mint location request", err)
	}
}
