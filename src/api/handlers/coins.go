package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/briandenicola/ancient-coins-api/repository"
	"github.com/briandenicola/ancient-coins-api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// allowedListSortFields is the handler-level allowlist of sort fields for the
// coin list endpoint. Matches the repository's allowedSortFields map.
// "random" uses a seeded deterministic shuffle (see ?seed= query param).
var allowedListSortFields = map[string]bool{
	"created_at":    true,
	"updated_at":    true,
	"current_value": true,
	"purchase_date": true,
	"name":          true,
	"random":        true,
}

var nullableCoinUpdateScalarFields = map[string]string{
	"purchasePrice": "PurchasePrice",
	"currentValue":  "CurrentValue",
	"purchaseDate":  "PurchaseDate",
	"soldPrice":     "SoldPrice",
	"soldDate":      "SoldDate",
	"weightGrams":   "WeightGrams",
	"diameterMm":    "DiameterMm",
}

func nullableScalarFieldPresence(raw map[string]json.RawMessage) map[string]bool {
	present := make(map[string]bool, len(nullableCoinUpdateScalarFields))
	for jsonField, modelField := range nullableCoinUpdateScalarFields {
		if _, ok := raw[jsonField]; ok {
			present[modelField] = true
		}
	}
	return present
}

type CoinHandler struct {
	repo   *repository.CoinRepository
	svc    *services.CoinService
	logger *services.Logger
}

func NewCoinHandler(repo *repository.CoinRepository, svc *services.CoinService, logger *services.Logger) *CoinHandler {
	return &CoinHandler{repo: repo, svc: svc, logger: logger}
}

// PurchaseRequest holds optional details when purchasing a wishlist coin.
type PurchaseRequest struct {
	PurchasePrice    *float64 `json:"purchasePrice"`
	PurchaseDate     string   `json:"purchaseDate"`
	PurchaseLocation string   `json:"purchaseLocation"`
}

// List returns a paginated list of coins for the authenticated user.
//
//	@Summary		List coins
//	@Description	Returns a paginated, filterable list of coins belonging to the authenticated user.
//	@Tags			Coins
//	@Produce		json
//	@Param			category	query		string	false	"Filter by category (Roman, Greek, Byzantine, Modern, Other)"
//	@Param			era			query		string	false	"Filter by era (ancient, medieval, modern)"
//	@Param			search		query		string	false	"Search across name, denomination, ruler, era, mint, inscriptions, notes"
//	@Param			wishlist	query		string	false	"Filter by wishlist status"	Enums(true, false)
//	@Param			sold		query		string	false	"Filter by sold status"	Enums(true, false)
//	@Param			page		query		int		false	"Page number"	default(1)
//	@Param			limit		query		int		false	"Items per page (max 100)"	default(50)
//	@Param			sort		query		string	false	"Sort field"	Enums(created_at, updated_at, current_value)	default(updated_at)
//	@Param			order		query		string	false	"Sort order"	Enums(asc, desc)	default(desc)
//	@Success		200			{object}	CoinListResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins [get]
func (h *CoinHandler) List(c *gin.Context) {
	userID := c.GetUint("userId")

	// Validate page
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page must be an integer >= 1"})
		return
	}

	// Validate limit
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer between 1 and 100"})
		return
	}

	// Validate sort field against allowlist (defense-in-depth against SQL injection)
	sortField := c.DefaultQuery("sort", "updated_at")
	if !allowedListSortFields[sortField] {
		allowed := make([]string, 0, len(allowedListSortFields))
		for k := range allowedListSortFields {
			allowed = append(allowed, k)
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("sort must be one of: %s", strings.Join(allowed, ", ")),
		})
		return
	}

	// Validate sort order
	sortOrder := c.DefaultQuery("order", "desc")
	if sortOrder != "asc" && sortOrder != "desc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order must be 'asc' or 'desc'"})
		return
	}

	filters := repository.CoinListFilters{
		Category:  c.Query("category"),
		Era:       c.Query("era"),
		Search:    c.Query("search"),
		SortField: sortField,
		SortOrder: sortOrder,
		Page:      page,
		Limit:     limit,
	}
	// Parse optional seed for deterministic random sort.
	// Use strconv.Atoi to ensure it's a real integer (defense against SQL injection).
	if sortField == "random" {
		if seedStr := c.Query("seed"); seedStr != "" {
			if seed, err := strconv.Atoi(seedStr); err == nil {
				filters.Seed = &seed
			}
		}
		if filters.Seed == nil {
			// Default seed if none provided — based on current minute to vary slightly.
			s := int(time.Now().Unix() % 1000000)
			filters.Seed = &s
		}
	}
	if w := c.Query("wishlist"); w == "true" {
		v := true
		filters.Wishlist = &v
	} else if w == "false" {
		v := false
		filters.Wishlist = &v
	}
	if s := c.Query("sold"); s == "true" {
		v := true
		filters.Sold = &v
	} else if s == "false" {
		v := false
		filters.Sold = &v
	}
	if t := c.Query("tag"); t != "" {
		if tagID, err := strconv.ParseUint(t, 10, strconv.IntSize); err == nil {
			v := uint(tagID)
			filters.TagID = &v
		}
	}
	if s := c.Query("set"); s != "" {
		if setID, err := strconv.ParseUint(s, 10, strconv.IntSize); err == nil {
			v := uint(setID)
			filters.SetID = &v
		}
	}

	coins, total, err := h.repo.List(userID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch coins"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"coins": coins,
		"total": total,
		"page":  filters.Page,
		"limit": filters.Limit,
	})
}

// Get returns a single coin by ID for the authenticated user.
//
//	@Summary		Get a coin
//	@Description	Returns a single coin with its images, owned by the authenticated user.
//	@Tags			Coins
//	@Produce		json
//	@Param			id	path		int	true	"Coin ID"
//	@Success		200	{object}	models.Coin
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id} [get]
func (h *CoinHandler) Get(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	coin, err := h.repo.FindByID(uint(id), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch coin"})
		return
	}

	c.JSON(http.StatusOK, coin)
}

// Create adds a new coin for the authenticated user.
//
//	@Summary		Create a coin
//	@Description	Creates a new coin record for the authenticated user.
//	@Tags			Coins
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CoinCreateRequest	true	"Coin data"
//	@Success		201		{object}	models.Coin
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins [post]
func (h *CoinHandler) Create(c *gin.Context) {
	logger := h.logger
	userID := c.GetUint("userId")

	var req CoinCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	coin := req.toCoin(userID)

	logger.Debug("coins", "Creating coin '%s' for user %d", coin.Name, userID)

	if err := h.svc.CreateCoin(&coin); err != nil {
		if handleCoinMutationError(c, err) {
			return
		}
		logger.Error("coins", "Failed to create coin: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create coin"})
		return
	}

	logger.Info("coins", "Created coin %d '%s' for user %d", coin.ID, coin.Name, userID)
	c.JSON(http.StatusCreated, coin)
}

// Update modifies an existing coin owned by the authenticated user.
//
//	@Summary		Update a coin
//	@Description	Updates an existing coin record. Only the coin owner can update it.
//	@Tags			Coins
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int			true	"Coin ID"
//	@Param			body	body		CoinUpdateRequest	true	"Updated coin data"
//	@Success		200		{object}	models.Coin
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id} [put]
func (h *CoinHandler) Update(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	existing, err := h.repo.FindByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	var raw map[string]json.RawMessage
	storageLocationProvided := false
	nullableScalarProvided := map[string]bool{}
	if err := json.Unmarshal(bodyBytes, &raw); err == nil {
		_, storageLocationProvided = raw["storageLocationId"]
		nullableScalarProvided = nullableScalarFieldPresence(raw)
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var req CoinUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	updates, updateFields := req.toCoin(existing, storageLocationProvided, nullableScalarProvided)

	source := c.Query("source")
	if err := h.svc.UpdateCoinWithFields(existing, &updates, updateFields, userID, source, storageLocationProvided); err != nil {
		if handleCoinMutationError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update coin"})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// Purchase marks a wishlist coin as purchased (moves it to the collection).
//
//	@Summary		Mark coin as purchased
//	@Description	Sets isWishlist to false, moving the coin from wishlist to collection. Optionally accepts purchase details.
//	@Tags			Coins
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Coin ID"
//	@Param			body	body		PurchaseRequest			false	"Optional purchase details"
//	@Success		200		{object}	models.Coin
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/purchase [post]
func (h *CoinHandler) Purchase(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	coin, err := h.repo.FindByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	if !coin.IsWishlist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coin is not on the wishlist"})
		return
	}

	// Apply optional purchase details from request body
	var req PurchaseRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
			return
		}
	}
	if req.PurchasePrice != nil {
		coin.PurchasePrice = req.PurchasePrice
	}
	if req.PurchaseDate != "" {
		if t, err := time.Parse("2006-01-02", req.PurchaseDate); err == nil {
			coin.PurchaseDate = &t
		}
	}
	if req.PurchaseLocation != "" {
		coin.PurchaseLocation = req.PurchaseLocation
	}

	if err := h.svc.PurchaseCoin(coin, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as purchased"})
		return
	}

	c.JSON(http.StatusOK, coin)
}

// Sell marks a collection coin as sold with a sale price.
//
//	@Summary		Mark coin as sold
//	@Description	Sets isSold to true with a sold price and date, moving the coin from collection to sold gallery.
//	@Tags			Coins
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Coin ID"
//	@Param			body	body		object{soldPrice float64}	true	"Sale details"
//	@Success		200		{object}	models.Coin
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/sell [post]
func (h *CoinHandler) Sell(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	var body struct {
		SoldPrice *float64 `json:"soldPrice"`
		SoldTo    string   `json:"soldTo"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	coin, err := h.repo.FindByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{
		"is_sold":    true,
		"sold_price": body.SoldPrice,
		"sold_date":  &now,
		"sold_to":    body.SoldTo,
	}
	if err := h.svc.SellCoin(coin, updates, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as sold"})
		return
	}

	c.JSON(http.StatusOK, coin)
}

// Delete removes a coin and its associated images.
//
//	@Summary		Delete a coin
//	@Description	Deletes a coin and all associated image records. Only the coin owner can delete it.
//	@Tags			Coins
//	@Produce		json
//	@Param			id	path		int	true	"Coin ID"
//	@Success		200	{object}	CoinDeletedResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id} [delete]
func (h *CoinHandler) Delete(c *gin.Context) {
	userID := c.GetUint("userId")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	rows, err := h.svc.DeleteCoin(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete coin"})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Coin deleted"})
}

// Stats returns aggregate statistics for the authenticated user's collection.
//
//	@Summary		Get collection statistics
//	@Description	Returns counts by category/material, total/average values, and wishlist count.
//	@Tags			Coins
//	@Produce		json
//	@Success		200	{object}	StatsResponse
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/stats [get]
func (h *CoinHandler) Stats(c *gin.Context) {
	userID := c.GetUint("userId")

	stats, err := h.repo.GetStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"totalCoins":    stats.TotalCoins,
		"totalWishlist": stats.TotalWishlist,
		"totalSold":     stats.TotalSold,
		"byCategory":    stats.ByCategory,
		"byMaterial":    stats.ByMaterial,
		"byGrade":       stats.ByGrade,
		"byEra":         stats.ByEra,
		"byRuler":       stats.ByRuler,
		"byPriceRange":  stats.ByPriceRange,
		"values":        stats.Values,
		"soldValues":    stats.SoldValues,
	})
}

// Distribution returns the era × category cross-tabulation for the collection heat map.
//
//	@Summary		Get collection distribution
//	@Description	Returns era × category counts for the heat map visualization.
//	@Tags			Coins
//	@Produce		json
//	@Success		200	{array}		repository.DistributionCell
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/stats/distribution [get]
func (h *CoinHandler) Distribution(c *gin.Context) {
	userID := c.GetUint("userId")
	cells, err := h.repo.GetDistribution(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch distribution"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cells": cells})
}

// Suggestions returns distinct values for autocomplete fields.
//
//	@Summary		Get autocomplete suggestions
//	@Description	Returns distinct values for the specified field, optionally filtered by a search query. Limited to 20 results.
//	@Tags			Coins
//	@Produce		json
//	@Param			field	query		string	true	"Field to get suggestions for"	Enums(name, denomination, purchaseLocation)
//	@Param			q		query		string	false	"Search filter"
//	@Success		200		{array}		string
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/suggestions [get]
func (h *CoinHandler) Suggestions(c *gin.Context) {
	userID := c.GetUint("userId")
	field := c.Query("field")
	q := c.Query("q")

	var column string
	switch field {
	case "name":
		column = "name"
	case "denomination":
		column = "denomination"
	case "ruler":
		column = "ruler"
	case "era":
		column = "era"
	case "purchaseLocation":
		column = "purchase_location"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid field"})
		return
	}

	values, err := h.repo.Suggestions(userID, column, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch suggestions"})
		return
	}
	c.JSON(http.StatusOK, values)
}

// ValueHistory returns historical value snapshots for the authenticated user.
//
//	@Summary		Get value history
//	@Description	Returns value snapshots over time for charting collection value trends.
//	@Tags			Coins
//	@Produce		json
//	@Success		200	{array}		models.ValueSnapshot
//	@Failure		401	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/value-history [get]
func (h *CoinHandler) ValueHistory(c *gin.Context) {
	userID := c.GetUint("userId")

	snapshots, err := h.repo.GetValueHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch value history"})
		return
	}

	c.JSON(http.StatusOK, snapshots)
}

// CoinValueHistory returns value history entries for a specific coin.
//
//	@Summary		Get coin value history
//	@Description	Returns value tracking entries for a coin, ordered oldest first.
//	@Tags			Coins
//	@Produce		json
//	@Param			id	path		int	true	"Coin ID"
//	@Success		200	{array}		models.CoinValueHistory
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/coins/{id}/value-history [get]
func (h *CoinHandler) CoinValueHistory(c *gin.Context) {
	userID := c.GetUint("userId")
	coinID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin ID"})
		return
	}

	exists, err := h.repo.CoinExists(uint(coinID), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify coin"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Coin not found"})
		return
	}

	entries, err := h.repo.GetCoinValueHistory(uint(coinID), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch value history"})
		return
	}

	c.JSON(http.StatusOK, entries)
}

func handleCoinMutationError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, services.ErrReferenceCatalogRequired),
		errors.Is(err, services.ErrReferenceNumberRequired),
		errors.Is(err, services.ErrReferenceVolumeRequired),
		errors.Is(err, services.ErrReferenceUnknownCatalog),
		errors.Is(err, services.ErrReferenceDuplicate),
		errors.Is(err, services.ErrStorageLocationNotFound),
		errors.Is(err, services.ErrCoinInvalidEra):
		respondError(c, http.StatusBadRequest, err.Error(), err)
		return true
	case isUniqueConstraintError(err):
		respondError(c, http.StatusBadRequest, services.ErrReferenceDuplicate.Error(), err)
		return true
	default:
		return false
	}
}
