package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Outcomes a Coin Copilot wishlist proposal can report in addition to the
// "ready", "needs_review" and "duplicate" outcomes of Analyze.
const (
	WishlistURLOutcomeInvalidURL  = "invalid_url"
	WishlistURLOutcomeUnreadable  = "unreadable"
	WishlistURLOutcomeUnavailable = "unavailable"
)

var ErrWishlistURLExtractionUnavailable = errors.New("wishlist URL extraction unavailable")

// Copilot tool results are persisted under a small byte budget, so each field
// keeps one short evidence quote; the review form uses only the values.
const copilotWishlistEvidenceMaxChars = 200

const (
	wishlistURLInvalidWarning     = "The link is not a public HTTP(S) listing URL."
	wishlistURLUnreadableWarning  = "The listing page could not be read. The site may block automated access or need a browser to show the coin."
	wishlistURLUnavailableWarning = "Listing analysis is unavailable right now. Check the AI provider settings or try again later."
)

// ProposeForCopilot prepares the same reviewable proposal as "Add from URL"
// for Coin Copilot. A bad link or unreadable page is reported as an outcome
// rather than an error so one listing cannot fail a whole Copilot run. It
// never writes: the owner confirms the proposal in the app before anything is
// added to the wishlist.
func (s *WishlistURLService) ProposeForCopilot(ctx context.Context, userID uint, rawURL string) (*WishlistURLAnalysis, error) {
	canonicalURL, err := CanonicalizeWishlistURL(rawURL)
	if err != nil {
		return wishlistURLFailedProposal(WishlistURLOutcomeInvalidURL, "", wishlistURLInvalidWarning), nil
	}
	result, err := s.Analyze(ctx, userID, rawURL)
	if err == nil {
		trimWishlistHypothesisEvidence(result.Hypothesis)
		return result, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	switch {
	case errors.Is(err, ErrWishlistURLInvalid), IsOutboundTargetBlockedError(err):
		return wishlistURLFailedProposal(WishlistURLOutcomeInvalidURL, "", wishlistURLInvalidWarning), nil
	case errors.Is(err, ErrWishlistURLFetchFailed):
		return wishlistURLFailedProposal(WishlistURLOutcomeUnreadable, canonicalURL, wishlistURLUnreadableWarning), nil
	case errors.Is(err, ErrWishlistURLExtractionUnavailable):
		return wishlistURLFailedProposal(WishlistURLOutcomeUnavailable, canonicalURL, wishlistURLUnavailableWarning), nil
	default:
		return nil, fmt.Errorf("prepare Copilot wishlist proposal: %w", err)
	}
}

func wishlistURLFailedProposal(outcome, sourceURL, warning string) *WishlistURLAnalysis {
	return &WishlistURLAnalysis{
		SchemaVersion: WishlistURLProposalSchemaVersion,
		Outcome:       outcome,
		SourceURL:     sourceURL,
		Warnings:      []string{warning},
	}
}

func trimWishlistHypothesisEvidence(hypothesis *WishlistURLHypothesis) {
	if hypothesis == nil {
		return
	}
	for _, field := range []*WishlistURLHypothesisField{
		hypothesis.Name, hypothesis.Category, hypothesis.Ruler, hypothesis.Denomination,
		hypothesis.Material, hypothesis.Mint, hypothesis.DateRange, hypothesis.Era,
		hypothesis.ObverseLegend, hypothesis.ObverseDescription, hypothesis.ReverseLegend,
		hypothesis.ReverseDescription, hypothesis.Weight, hypothesis.Diameter, hypothesis.Grade,
		hypothesis.RarityRating, hypothesis.References, hypothesis.Notes, hypothesis.CoinType,
		hypothesis.ListingStatus, hypothesis.ListedPrice, hypothesis.Currency, hypothesis.DealerName,
	} {
		if field == nil || len(field.Evidence) == 0 {
			continue
		}
		quote := field.Evidence[0]
		if runes := []rune(quote); len(runes) > copilotWishlistEvidenceMaxChars {
			quote = string(runes[:copilotWishlistEvidenceMaxChars])
		}
		field.Evidence = []string{quote}
	}
}

// PublicCopilotWishlistProposal re-validates a proposal relayed through the
// agent service before the browser renders it as an "Add to wishlist" form.
func PublicCopilotWishlistProposal(raw json.RawMessage) (*WishlistURLAnalysis, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var proposal WishlistURLAnalysis
	if err := decoder.Decode(&proposal); err != nil {
		return nil, ErrInvalidCopilotFrame
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, ErrInvalidCopilotFrame
	}
	if proposal.SchemaVersion != WishlistURLProposalSchemaVersion ||
		len(proposal.PageTitle) > 500 || len(proposal.Warnings) > 20 {
		return nil, ErrInvalidCopilotFrame
	}
	for _, warning := range proposal.Warnings {
		if warning == "" || len(warning) > 500 {
			return nil, ErrInvalidCopilotFrame
		}
	}
	if proposal.SourceURL != "" {
		if canonical, err := CanonicalizeWishlistURL(proposal.SourceURL); err != nil || canonical != proposal.SourceURL {
			return nil, ErrInvalidCopilotFrame
		}
	}
	for _, image := range []string{proposal.ImageURL, proposal.ReverseImageURL} {
		if image == "" {
			continue
		}
		if _, err := ValidateOutboundURL(image); err != nil {
			return nil, ErrInvalidCopilotFrame
		}
	}
	hasProposal := proposal.Hypothesis != nil
	hasDuplicate := proposal.ExistingCoinID != nil
	switch proposal.Outcome {
	case "ready", "needs_review":
		if !hasProposal || hasDuplicate || proposal.SourceURL == "" {
			return nil, ErrInvalidCopilotFrame
		}
	case "duplicate":
		if hasProposal || !hasDuplicate || *proposal.ExistingCoinID == 0 || proposal.SourceURL == "" {
			return nil, ErrInvalidCopilotFrame
		}
	case WishlistURLOutcomeInvalidURL, WishlistURLOutcomeUnreadable, WishlistURLOutcomeUnavailable:
		if hasProposal || hasDuplicate {
			return nil, ErrInvalidCopilotFrame
		}
	default:
		return nil, ErrInvalidCopilotFrame
	}
	return &proposal, nil
}
