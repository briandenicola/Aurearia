package handlers

import (
	"time"

	"github.com/briandenicola/ancient-coins-api/models"
)

type CoinReferenceRequest struct {
	Catalog       string `json:"catalog" binding:"max=32"`
	Volume        string `json:"volume" binding:"max=64"`
	Number        string `json:"number" binding:"max=128"`
	InvoiceNumber string `json:"invoiceNumber" binding:"max=64"`
	URI           string `json:"uri" binding:"max=2000"`
}

type CoinCreateRequest struct {
	Name                  string                 `json:"name" binding:"max=200"`
	Category              models.Category        `json:"category"`
	Denomination          string                 `json:"denomination" binding:"max=200"`
	Ruler                 string                 `json:"ruler" binding:"max=200"`
	RomanImperialFigureID *uint                  `json:"romanImperialFigureId"`
	Era                   models.Era             `json:"era" binding:"omitempty,max=64"`
	Mint                  string                 `json:"mint" binding:"max=200"`
	Material              models.Material        `json:"material"`
	WeightGrams           *float64               `json:"weightGrams"`
	DiameterMm            *float64               `json:"diameterMm"`
	Grade                 string                 `json:"grade" binding:"max=100"`
	ObverseInscription    string                 `json:"obverseInscription" binding:"max=1000"`
	ReverseInscription    string                 `json:"reverseInscription" binding:"max=1000"`
	ObverseDescription    string                 `json:"obverseDescription" binding:"max=2000"`
	ReverseDescription    string                 `json:"reverseDescription" binding:"max=2000"`
	RarityRating          string                 `json:"rarityRating" binding:"max=100"`
	PurchasePrice         *float64               `json:"purchasePrice"`
	CurrentValue          *float64               `json:"currentValue"`
	PurchaseDate          *time.Time             `json:"purchaseDate"`
	PurchaseLocation      string                 `json:"purchaseLocation" binding:"max=500"`
	Notes                 string                 `json:"notes" binding:"max=5000"`
	ReferenceURL          string                 `json:"referenceUrl" binding:"max=2000"`
	ReferenceText         string                 `json:"referenceText" binding:"max=2000"`
	IsWishlist            bool                   `json:"isWishlist"`
	IsSold                bool                   `json:"isSold"`
	SoldPrice             *float64               `json:"soldPrice"`
	SoldDate              *time.Time             `json:"soldDate"`
	SoldTo                string                 `json:"soldTo"`
	StorageLocationID     *uint                  `json:"storageLocationId"`
	IsPrivate             bool                   `json:"isPrivate"`
	References            []CoinReferenceRequest `json:"references"`
}

type CoinUpdateRequest struct {
	Name                  *string                `json:"name" binding:"omitempty,max=200"`
	Category              *models.Category       `json:"category"`
	Denomination          *string                `json:"denomination" binding:"omitempty,max=200"`
	Ruler                 *string                `json:"ruler" binding:"omitempty,max=200"`
	RomanImperialFigureID *uint                  `json:"romanImperialFigureId"`
	Era                   *models.Era            `json:"era" binding:"omitempty,max=64"`
	Mint                  *string                `json:"mint" binding:"omitempty,max=200"`
	Material              *models.Material       `json:"material"`
	WeightGrams           *float64               `json:"weightGrams"`
	DiameterMm            *float64               `json:"diameterMm"`
	Grade                 *string                `json:"grade" binding:"omitempty,max=100"`
	ObverseInscription    *string                `json:"obverseInscription" binding:"omitempty,max=1000"`
	ReverseInscription    *string                `json:"reverseInscription" binding:"omitempty,max=1000"`
	ObverseDescription    *string                `json:"obverseDescription" binding:"omitempty,max=2000"`
	ReverseDescription    *string                `json:"reverseDescription" binding:"omitempty,max=2000"`
	RarityRating          *string                `json:"rarityRating" binding:"omitempty,max=100"`
	PurchasePrice         *float64               `json:"purchasePrice"`
	CurrentValue          *float64               `json:"currentValue"`
	PurchaseDate          *time.Time             `json:"purchaseDate"`
	PurchaseLocation      *string                `json:"purchaseLocation" binding:"omitempty,max=500"`
	Notes                 *string                `json:"notes" binding:"omitempty,max=5000"`
	ReferenceURL          *string                `json:"referenceUrl" binding:"omitempty,max=2000"`
	ReferenceText         *string                `json:"referenceText" binding:"omitempty,max=2000"`
	IsWishlist            *bool                  `json:"isWishlist"`
	IsSold                *bool                  `json:"isSold"`
	SoldPrice             *float64               `json:"soldPrice"`
	SoldDate              *time.Time             `json:"soldDate"`
	SoldTo                *string                `json:"soldTo"`
	StorageLocationID     *uint                  `json:"storageLocationId"`
	IsPrivate             *bool                  `json:"isPrivate"`
	References            []CoinReferenceRequest `json:"references"`
}

func (r CoinCreateRequest) toCoin(userID uint) models.Coin {
	romanImperialFigureID := r.RomanImperialFigureID
	if r.Category != models.CategoryRoman {
		romanImperialFigureID = nil
	}
	return models.Coin{
		Name:                  r.Name,
		Category:              r.Category,
		Denomination:          r.Denomination,
		Ruler:                 r.Ruler,
		RomanImperialFigureID: romanImperialFigureID,
		Era:                   r.Era,
		Mint:                  r.Mint,
		Material:              r.Material,
		WeightGrams:           r.WeightGrams,
		DiameterMm:            r.DiameterMm,
		Grade:                 r.Grade,
		ObverseInscription:    r.ObverseInscription,
		ReverseInscription:    r.ReverseInscription,
		ObverseDescription:    r.ObverseDescription,
		ReverseDescription:    r.ReverseDescription,
		RarityRating:          r.RarityRating,
		PurchasePrice:         r.PurchasePrice,
		CurrentValue:          r.CurrentValue,
		PurchaseDate:          r.PurchaseDate,
		PurchaseLocation:      r.PurchaseLocation,
		Notes:                 r.Notes,
		ReferenceURL:          r.ReferenceURL,
		ReferenceText:         r.ReferenceText,
		IsWishlist:            r.IsWishlist,
		IsSold:                r.IsSold,
		SoldPrice:             r.SoldPrice,
		SoldDate:              r.SoldDate,
		SoldTo:                r.SoldTo,
		StorageLocationID:     r.StorageLocationID,
		IsPrivate:             r.IsPrivate,
		UserID:                userID,
		References:            mapCoinReferenceRequests(r.References),
	}
}

func appendUpdateField(fields []string, field string) []string {
	for _, existing := range fields {
		if existing == field {
			return fields
		}
	}
	return append(fields, field)
}

func (r CoinUpdateRequest) toCoin(existing *models.Coin, storageLocationProvided bool, nullableScalarProvided map[string]bool) (models.Coin, []string) {
	updates := models.Coin{ID: existing.ID, UserID: existing.UserID}
	updateFields := make([]string, 0, 32)
	targetCategory := existing.Category
	if r.Name != nil {
		updates.Name = *r.Name
		updateFields = append(updateFields, "Name")
	}
	if r.Category != nil {
		updates.Category = *r.Category
		targetCategory = *r.Category
		updateFields = append(updateFields, "Category")
	}
	if r.Denomination != nil {
		updates.Denomination = *r.Denomination
		updateFields = append(updateFields, "Denomination")
	}
	if r.Ruler != nil {
		updates.Ruler = *r.Ruler
		updateFields = append(updateFields, "Ruler")
	}
	if r.RomanImperialFigureID != nil || nullableScalarProvided["RomanImperialFigureID"] {
		updates.RomanImperialFigureID = r.RomanImperialFigureID
		updateFields = appendUpdateField(updateFields, "RomanImperialFigureID")
	}
	if r.Era != nil {
		updates.Era = *r.Era
		updateFields = append(updateFields, "Era")
	}
	if r.Mint != nil {
		updates.Mint = *r.Mint
		updateFields = append(updateFields, "Mint")
	}
	if r.Material != nil {
		updates.Material = *r.Material
		updateFields = append(updateFields, "Material")
	}
	if r.WeightGrams != nil || nullableScalarProvided["WeightGrams"] {
		updates.WeightGrams = r.WeightGrams
		updateFields = append(updateFields, "WeightGrams")
	}
	if r.DiameterMm != nil || nullableScalarProvided["DiameterMm"] {
		updates.DiameterMm = r.DiameterMm
		updateFields = append(updateFields, "DiameterMm")
	}
	if r.Grade != nil {
		updates.Grade = *r.Grade
		updateFields = append(updateFields, "Grade")
	}
	if r.ObverseInscription != nil {
		updates.ObverseInscription = *r.ObverseInscription
		updateFields = append(updateFields, "ObverseInscription")
	}
	if r.ReverseInscription != nil {
		updates.ReverseInscription = *r.ReverseInscription
		updateFields = append(updateFields, "ReverseInscription")
	}
	if r.ObverseDescription != nil {
		updates.ObverseDescription = *r.ObverseDescription
		updateFields = append(updateFields, "ObverseDescription")
	}
	if r.ReverseDescription != nil {
		updates.ReverseDescription = *r.ReverseDescription
		updateFields = append(updateFields, "ReverseDescription")
	}
	if r.RarityRating != nil {
		updates.RarityRating = *r.RarityRating
		updateFields = append(updateFields, "RarityRating")
	}
	if r.PurchasePrice != nil || nullableScalarProvided["PurchasePrice"] {
		updates.PurchasePrice = r.PurchasePrice
		updateFields = append(updateFields, "PurchasePrice")
	}
	if r.CurrentValue != nil || nullableScalarProvided["CurrentValue"] {
		updates.CurrentValue = r.CurrentValue
		updateFields = append(updateFields, "CurrentValue")
	}
	if r.PurchaseDate != nil || nullableScalarProvided["PurchaseDate"] {
		updates.PurchaseDate = r.PurchaseDate
		updateFields = append(updateFields, "PurchaseDate")
	}
	if r.PurchaseLocation != nil {
		updates.PurchaseLocation = *r.PurchaseLocation
		updateFields = append(updateFields, "PurchaseLocation")
	}
	if r.Notes != nil {
		updates.Notes = *r.Notes
		updateFields = append(updateFields, "Notes")
	}
	if r.ReferenceURL != nil {
		updates.ReferenceURL = *r.ReferenceURL
		updateFields = append(updateFields, "ReferenceURL")
	}
	if r.ReferenceText != nil {
		updates.ReferenceText = *r.ReferenceText
		updateFields = append(updateFields, "ReferenceText")
	}
	if r.IsWishlist != nil {
		updates.IsWishlist = *r.IsWishlist
		updateFields = append(updateFields, "IsWishlist")
	}
	if r.IsSold != nil {
		updates.IsSold = *r.IsSold
		updateFields = append(updateFields, "IsSold")
	}
	if r.SoldPrice != nil || nullableScalarProvided["SoldPrice"] {
		updates.SoldPrice = r.SoldPrice
		updateFields = append(updateFields, "SoldPrice")
	}
	if r.SoldDate != nil || nullableScalarProvided["SoldDate"] {
		updates.SoldDate = r.SoldDate
		updateFields = append(updateFields, "SoldDate")
	}
	if r.SoldTo != nil {
		updates.SoldTo = *r.SoldTo
		updateFields = append(updateFields, "SoldTo")
	}
	if storageLocationProvided {
		updates.StorageLocationID = r.StorageLocationID
	}
	if r.IsPrivate != nil {
		updates.IsPrivate = *r.IsPrivate
		updateFields = append(updateFields, "IsPrivate")
	}
	if r.References != nil {
		updates.References = mapCoinReferenceRequests(r.References)
	}
	if targetCategory != models.CategoryRoman {
		updates.RomanImperialFigureID = nil
		updateFields = appendUpdateField(updateFields, "RomanImperialFigureID")
	}
	return updates, updateFields
}

func mapCoinReferenceRequests(requests []CoinReferenceRequest) []models.CoinReference {
	if requests == nil {
		return nil
	}
	refs := make([]models.CoinReference, 0, len(requests))
	for _, ref := range requests {
		refs = append(refs, models.CoinReference{
			Catalog:       ref.Catalog,
			Volume:        ref.Volume,
			Number:        ref.Number,
			InvoiceNumber: ref.InvoiceNumber,
			URI:           ref.URI,
		})
	}
	return refs
}
