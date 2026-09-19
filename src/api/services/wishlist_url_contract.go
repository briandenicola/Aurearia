package services

import (
	"context"
	"encoding/json"
)

const WishlistURLProposalSchemaVersion = 1

type WishlistURLHypothesisField struct {
	Value      string   `json:"value"`
	Confidence float64  `json:"confidence"`
	Evidence   []string `json:"evidence"`
}

type WishlistURLHypothesis struct {
	Name               *WishlistURLHypothesisField `json:"name"`
	Category           *WishlistURLHypothesisField `json:"category"`
	Ruler              *WishlistURLHypothesisField `json:"ruler"`
	Denomination       *WishlistURLHypothesisField `json:"denomination"`
	Material           *WishlistURLHypothesisField `json:"material"`
	Mint               *WishlistURLHypothesisField `json:"mint"`
	DateRange          *WishlistURLHypothesisField `json:"dateRange"`
	Era                *WishlistURLHypothesisField `json:"era"`
	ObverseLegend      *WishlistURLHypothesisField `json:"obverseInscription"`
	ObverseDescription *WishlistURLHypothesisField `json:"obverseDescription"`
	ReverseLegend      *WishlistURLHypothesisField `json:"reverseInscription"`
	ReverseDescription *WishlistURLHypothesisField `json:"reverseDescription"`
	Weight             *WishlistURLHypothesisField `json:"weightGrams"`
	Diameter           *WishlistURLHypothesisField `json:"diameterMm"`
	Grade              *WishlistURLHypothesisField `json:"grade"`
	RarityRating       *WishlistURLHypothesisField `json:"rarityRating"`
	References         *WishlistURLHypothesisField `json:"references"`
	Notes              *WishlistURLHypothesisField `json:"notes"`
	CoinType           *WishlistURLHypothesisField `json:"coin_type"`
	ListingStatus      *WishlistURLHypothesisField `json:"listingStatus"`
	ListedPrice        *WishlistURLHypothesisField `json:"listedPrice"`
	Currency           *WishlistURLHypothesisField `json:"currency"`
	DealerName         *WishlistURLHypothesisField `json:"dealerName"`
	Observations       string                      `json:"observations"`
	Legible            bool                        `json:"legible"`
}

type WishlistURLExtractionRequest struct {
	LLM          LLMConfig       `json:"llm"`
	SourceURL    string          `json:"source_url"`
	PageTitle    string          `json:"page_title"`
	PageText     string          `json:"page_text"`
	PageMetadata json.RawMessage `json:"page_metadata"`
}

type WishlistURLExtractionResponse struct {
	Hypothesis WishlistURLHypothesis `json:"hypothesis"`
	Warnings   []string              `json:"warnings"`
}

type WishlistURLExtractor interface {
	ExtractWishlistURL(context.Context, WishlistURLExtractionRequest) (*WishlistURLExtractionResponse, error)
}
