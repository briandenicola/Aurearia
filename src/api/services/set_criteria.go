package services

import (
	"fmt"
	"slices"
)

var allowedCriteriaFields = []string{
	"material", "category", "denomination", "ruler", "era", "mint", "grade",
	"currentValue", "purchasePrice", "purchaseDate", "createdAt", "isWishlist",
	"isSold", "isPrivate",
}

var allowedCriteriaOps = []string{
	"eq", "neq", "contains", "startsWith", "in", "between", "gte", "lte", "isNull", "isNotNull",
}

// ValidateSmartCriteria validates the restricted smart-set criteria tree.
func ValidateSmartCriteria(criteria map[string]interface{}) error {
	if len(criteria) == 0 {
		return fmt.Errorf("smart criteria is required")
	}
	return validateCriteriaNode(criteria)
}

func validateCriteriaNode(node map[string]interface{}) error {
	if op, ok := node["operator"].(string); ok {
		if op != "and" && op != "or" {
			return fmt.Errorf("criteria operator must be and or or")
		}
		rules, ok := node["rules"].([]interface{})
		if !ok || len(rules) == 0 {
			return fmt.Errorf("criteria groups require rules")
		}
		for _, raw := range rules {
			child, ok := raw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("criteria rule must be an object")
			}
			if err := validateCriteriaNode(child); err != nil {
				return err
			}
		}
		return nil
	}

	field, _ := node["field"].(string)
	ruleOp, _ := node["op"].(string)
	if !slices.Contains(allowedCriteriaFields, field) {
		return fmt.Errorf("criteria field %q is not allowed", field)
	}
	if !slices.Contains(allowedCriteriaOps, ruleOp) {
		return fmt.Errorf("criteria operator %q is not allowed", ruleOp)
	}
	return nil
}

// SuggestedCriteria is a built-in starter template for common smart sets.
type SuggestedCriteria struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Criteria    map[string]interface{} `json:"criteria"`
}

// GetSuggestedCriteria returns the built-in suggested smart set starters.
func GetSuggestedCriteria() []SuggestedCriteria {
	trueVal := true
	return []SuggestedCriteria{
		{
			ID:          "silver-coins",
			Name:        "Silver Coins",
			Description: "All coins made of silver in your collection",
			Criteria:    singleRule("material", "eq", "Silver"),
		},
		{
			ID:          "gold-coins",
			Name:        "Gold Coins",
			Description: "All coins made of gold in your collection",
			Criteria:    singleRule("material", "eq", "Gold"),
		},
		{
			ID:          "bronze-coins",
			Name:        "Bronze Coins",
			Description: "All coins made of bronze in your collection",
			Criteria:    singleRule("material", "eq", "Bronze"),
		},
		{
			ID:          "roman-collection",
			Name:        "Roman Collection",
			Description: "All coins in the Roman category",
			Criteria:    singleRule("category", "eq", "Roman"),
		},
		{
			ID:          "greek-collection",
			Name:        "Greek Collection",
			Description: "All coins in the Greek category",
			Criteria:    singleRule("category", "eq", "Greek"),
		},
		{
			ID:          "byzantine-collection",
			Name:        "Byzantine Collection",
			Description: "All coins in the Byzantine category",
			Criteria:    singleRule("category", "eq", "Byzantine"),
		},
		{
			ID:          "wishlist",
			Name:        "Wishlist",
			Description: "All coins on your wishlist",
			Criteria:    singleRule("isWishlist", "eq", trueVal),
		},
		{
			ID:          "sold-items",
			Name:        "Sold Items",
			Description: "All coins you have sold",
			Criteria:    singleRule("isSold", "eq", trueVal),
		},
		{
			ID:          "high-value",
			Name:        "High-Value Coins",
			Description: "Coins with a current value of at least $100",
			Criteria:    singleRule("currentValue", "gte", float64(100)),
		},
		{
			ID:          "private-coins",
			Name:        "Private Coins",
			Description: "All coins marked as private",
			Criteria:    singleRule("isPrivate", "eq", trueVal),
		},
	}
}

func singleRule(field, op string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"operator": "and",
		"rules": []interface{}{
			map[string]interface{}{
				"field": field,
				"op":    op,
				"value": value,
			},
		},
	}
}
