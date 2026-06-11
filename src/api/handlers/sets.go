package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetHandler handles set-related HTTP requests.
type SetHandler struct {
	repo    *repository.SetRepository
	service *services.SetService
}

// NewSetHandler creates a new SetHandler.
func NewSetHandler(repo *repository.SetRepository, service *services.SetService) *SetHandler {
	return &SetHandler{
		repo:    repo,
		service: service,
	}
}

// List returns all sets for the authenticated user with summary data.
//
//	@Summary		List user sets
//	@Description	Get all sets for the authenticated user with aggregate summaries
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	object{sets=[]object}
//	@Failure		401	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Router			/sets [get]
func (h *SetHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")
	sets, err := h.service.ListSets(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sets"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sets": sets})
}

// Create creates a new set for the authenticated user.
//
//	@Summary		Create a set
//	@Description	Create a new coin set
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		object{name=string,description=string,color=string,icon=string,setType=string,parentSetId=number}	true	"Set creation data"
//	@Success		201		{object}	object
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		409		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Router			/sets [post]
func (h *SetHandler) Create(c *gin.Context) {
	userID := c.GetUint("userId")

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	set, err := h.service.CreateSet(userID, input)
	if err != nil {
		if err.Error() == "a set with this name already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get full details with summary
	detail, err := h.service.GetSetDetail(set.ID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get set details"})
		return
	}

	c.JSON(http.StatusCreated, detail)
}

// CreateFromCSV creates a defined or goal set from CSV target definitions.
//
//	@Summary		Create set from CSV
//	@Description	Create a defined or goal set using custom CSV target definitions
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		object{name=string,description=string,color=string,setType=string,csv=string}	true	"Set data and CSV content"
//	@Success		201		{object}	object
//	@Failure		400		{object}	object{error=string}
//	@Router			/sets/import-csv [post]
func (h *SetHandler) CreateFromCSV(c *gin.Context) {
	userID := c.GetUint("userId")
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
		SetType     string `json:"setType"`
		CSV         string `json:"csv"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if body.CSV == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV content is required"})
		return
	}
	input := map[string]interface{}{
		"name":        body.Name,
		"description": body.Description,
		"color":       body.Color,
		"setType":     body.SetType,
	}
	set, err := h.service.CreateSetFromCSV(userID, input, body.CSV)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	detail, err := h.service.GetSetDetail(set.ID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get set details"})
		return
	}
	c.JSON(http.StatusCreated, detail)
}

// Get returns detailed information for a specific set.
//
//	@Summary		Get set details
//	@Description	Get detailed information and aggregates for a specific set
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Set ID"
//	@Success		200	{object}	object
//	@Failure		401	{object}	object{error=string}
//	@Failure		404	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Router			/sets/{id} [get]
func (h *SetHandler) Get(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}

	detail, err := h.service.GetSetDetail(uint(id), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Set not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get set details"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// Update updates a set's properties.
//
//	@Summary		Update a set
//	@Description	Update set metadata
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int													true	"Set ID"
//	@Param			body	body		object{name=string,description=string,color=string}	true	"Updated fields"
//	@Success		200		{object}	object
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		404		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Router			/sets/{id} [put]
func (h *SetHandler) Update(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	_, err = h.service.UpdateSet(uint(id), userID, updates)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Set not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get updated details with summary
	detail, err := h.service.GetSetDetail(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get set details"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// Delete removes a set and its memberships.
//
//	@Summary		Delete a set
//	@Description	Delete a set and all its memberships
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Set ID"
//	@Success		200	{object}	object{message=string}
//	@Failure		401	{object}	object{error=string}
//	@Failure		404	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Router			/sets/{id} [delete]
func (h *SetHandler) Delete(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}

	if err := h.service.DeleteSet(uint(id), userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Set not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete set"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Set deleted"})
}

// GetCoins returns all coins in a set.
//
//	@Summary		List coins in a set
//	@Description	Get all coins that belong to a specific set
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Set ID"
//	@Success		200	{object}	object{coins=[]object}
//	@Failure		401	{object}	object{error=string}
//	@Failure		404	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Router			/sets/{id}/coins [get]
func (h *SetHandler) GetCoins(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}

	coins, err := h.service.GetCoinsInSet(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get coins"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coins": coins})
}

// AddCoin adds a coin to a manual set.
//
//	@Summary		Add coin to set
//	@Description	Add a coin to a manual set (not smart sets)
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"Set ID"
//	@Param			body	body		object{coinId=int,notes=string}	true	"Coin to add"
//	@Success		200		{object}	object{message=string}
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		404		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Router			/sets/{id}/coins [post]
func (h *SetHandler) AddCoin(c *gin.Context) {
	userID := c.GetUint("userId")
	setID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}

	var body struct {
		CoinID uint   `json:"coinId" binding:"required"`
		Notes  string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coin ID is required"})
		return
	}

	if err := h.service.AddCoinToSet(body.CoinID, uint(setID), userID, body.Notes); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Set or coin not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Coin added to set"})
}

// ReorderCoins saves the manual coin order for a set.
//
//	@Summary		Reorder set coins
//	@Description	Persist the full manual order for all coins in a non-smart set
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"Set ID"
//	@Param			body	body		object{coinIds=[]int}	true	"Ordered coin IDs"
//	@Success		200		{object}	object{message=string}
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		404		{object}	object{error=string}
//	@Router			/sets/{id}/coins/order [put]
func (h *SetHandler) ReorderCoins(c *gin.Context) {
	userID := c.GetUint("userId")
	setID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}

	var body struct {
		CoinIDs []uint `json:"coinIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coinIds is required"})
		return
	}

	if err := h.service.ReorderCoinsInSet(uint(setID), userID, body.CoinIDs); err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Set not found"})
		case errors.Is(err, services.ErrInvalidSetOrder):
			c.JSON(http.StatusBadRequest, gin.H{"error": "coinIds must exactly match current set members"})
		case errors.Is(err, services.ErrSmartSetOrder):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Smart sets cannot be manually reordered"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder set coins"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Set coin order saved"})
}

// RemoveCoin removes a coin from a manual set.
//
//	@Summary		Remove coin from set
//	@Description	Remove a coin from a manual set (not smart sets)
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int	true	"Set ID"
//	@Param			coinId	path		int	true	"Coin ID"
//	@Success		200		{object}	object{message=string}
//	@Failure		400		{object}	object{error=string}
//	@Failure		401		{object}	object{error=string}
//	@Failure		404		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Router			/sets/{id}/coins/{coinId} [delete]
func (h *SetHandler) RemoveCoin(c *gin.Context) {
	userID := c.GetUint("userId")
	setID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}
	coinID, err := strconv.ParseUint(c.Param("coinId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	if err := h.service.RemoveCoinFromSet(uint(coinID), uint(setID), userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Set or coin not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Coin removed from set"})
}

// GetTemplates returns all available set templates.
//
//	@Summary		List set templates
//	@Description	Get all available built-in set templates for popular collecting series
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	object{templates=[]object}
//	@Failure		401	{object}	object{error=string}
//	@Router			/sets/templates [get]
func (h *SetHandler) GetTemplates(c *gin.Context) {
	templates := services.GetTemplates()
	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// GetCompletion returns completion metrics for a defined or goal set.
//
//	@Summary		Get set completion
//	@Description	Get completion metrics including target count, completed count, and missing targets
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Set ID"
//	@Success		200	{object}	object{totalTargets=int,completedTargets=int,completionPercentage=number,missingTargets=[]object}
//	@Failure		400	{object}	object{error=string}
//	@Failure		401	{object}	object{error=string}
//	@Failure		404	{object}	object{error=string}
//	@Failure		500	{object}	object{error=string}
//	@Router			/sets/{id}/completion [get]
func (h *SetHandler) GetCompletion(c *gin.Context) {
	userID := c.GetUint("userId")
	setID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}

	completion, err := h.service.GetCompletion(uint(setID), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Set not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get completion metrics"})
		return
	}

	c.JSON(http.StatusOK, completion)
}

// CreateSnapshot captures a manual valuation snapshot.
//
//	@Summary		Create set snapshot
//	@Description	Capture a manual valuation snapshot for a set
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"Set ID"
//	@Success		201	{object}	object
//	@Router			/sets/{id}/snapshot [post]
func (h *SetHandler) CreateSnapshot(c *gin.Context) {
	userID := c.GetUint("userId")
	setID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}
	snapshot, err := h.service.CreateSnapshot(uint(setID), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create snapshot"})
		return
	}
	c.JSON(http.StatusCreated, snapshot)
}

// GetTrends returns set valuation snapshots for a range.
//
//	@Summary		Get set trends
//	@Description	Get valuation snapshots for a set
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	int		true	"Set ID"
//	@Param			range	query	string	false	"Range: 1m, 3m, 1y, all"
//	@Success		200		{object}	object{snapshots=[]object}
//	@Router			/sets/{id}/trends [get]
func (h *SetHandler) GetTrends(c *gin.Context) {
	userID := c.GetUint("userId")
	setID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}
	snapshots, err := h.service.GetTrends(uint(setID), userID, c.DefaultQuery("range", "1y"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get trends"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": snapshots})
}

// GetAnalytics returns aggregate analytics for a set.
//
//	@Summary		Get set analytics
//	@Description	Get ROI and velocity analytics for a set
//	@Tags			sets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	int	true	"Set ID"
//	@Success		200	{object}	object
//	@Router			/sets/{id}/analytics [get]
func (h *SetHandler) GetAnalytics(c *gin.Context) {
	userID := c.GetUint("userId")
	setID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid set ID"})
		return
	}
	analytics, err := h.service.GetAnalytics(uint(setID), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics"})
		return
	}
	c.JSON(http.StatusOK, analytics)
}

// CompareSets compares multiple sets over a range.
//
//	@Summary		Compare sets
//	@Description	Compare set performance over a selected range
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	object{setIds=[]int,range=string}	true	"Sets to compare"
//	@Success		200		{object}	object{sets=[]object}
//	@Router			/sets/compare [post]
func (h *SetHandler) CompareSets(c *gin.Context) {
	userID := c.GetUint("userId")
	var body struct {
		SetIDs []uint `json:"setIds" binding:"required"`
		Range  string `json:"range"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.SetIDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least two setIds are required"})
		return
	}
	sets, err := h.service.CompareSets(userID, body.SetIDs, body.Range)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compare sets"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sets": sets})
}

// PreviewSmartSet previews coins matched by smart criteria.
//
//	@Summary		Preview smart set
//	@Description	Preview matching coins before saving a smart set
//	@Tags			sets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	object	true	"Smart criteria"
//	@Success		200		{object}	object
//	@Router			/sets/preview-smart [post]
func (h *SetHandler) PreviewSmartSet(c *gin.Context) {
	userID := c.GetUint("userId")
	var criteria map[string]interface{}
	if err := c.ShouldBindJSON(&criteria); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid criteria"})
		return
	}
	preview, err := h.service.PreviewSmartSet(userID, criteria)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, preview)
}
