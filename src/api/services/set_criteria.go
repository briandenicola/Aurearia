package services

import (
	"fmt"
	"slices"
	"strings"
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

// defaultSuggestedCategories is used when the caller has no admin-defined
// CoinCategories list available (e.g. settings not wired), preserving the
// previous fixed suggestions rather than showing none at all.
var defaultSuggestedCategories = []string{"Roman", "Greek", "Byzantine"}

// GetSuggestedCriteria returns the built-in suggested smart set starters.
// Category-based suggestions are built from categories, the caller's
// current admin-defined CoinCategories list, so a customized category list
// is reflected here instead of a fixed Roman/Greek/Byzantine set.
func GetSuggestedCriteria(categories []string) []SuggestedCriteria {
	if len(categories) == 0 {
		categories = defaultSuggestedCategories
	}
	trueVal := true
	suggestions := []SuggestedCriteria{
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
	}
	for _, category := range categories {
		suggestions = append(suggestions, SuggestedCriteria{
			ID:          "category-" + slugifyCriteriaID(category),
			Name:        category + " Collection",
			Description: fmt.Sprintf("All coins in the %s category", category),
			Criteria:    singleRule("category", "eq", category),
		})
	}
	return append(suggestions,
		SuggestedCriteria{
			ID:          "wishlist",
			Name:        "Wishlist",
			Description: "All coins on your wishlist",
			Criteria:    singleRule("isWishlist", "eq", trueVal),
		},
		SuggestedCriteria{
			ID:          "sold-items",
			Name:        "Sold Items",
			Description: "All coins you have sold",
			Criteria:    singleRule("isSold", "eq", trueVal),
		},
		SuggestedCriteria{
			ID:          "high-value",
			Name:        "High-Value Coins",
			Description: "Coins with a current value of at least $100",
			Criteria:    singleRule("currentValue", "gte", float64(100)),
		},
		SuggestedCriteria{
			ID:          "private-coins",
			Name:        "Private Coins",
			Description: "All coins marked as private",
			Criteria:    singleRule("isPrivate", "eq", trueVal),
		},
	)
}

// slugifyCriteriaID turns a category name into a stable, URL/ID-safe
// suffix (lowercase, non-alphanumeric runs collapsed to a single hyphen).
func slugifyCriteriaID(value string) string {
	var b strings.Builder
	lastHyphen := true // avoid a leading hyphen
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
		} else if !lastHyphen {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
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
