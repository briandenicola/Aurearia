# Research: Coin Copilot Collector Curator

## R1. Profile storage

- **Decision**: Add one `CollectorProfile` row per authenticated owner. Keep all
  optional preferences and collecting goals in that row, using bounded JSON
  string arrays for the list values.
- **Rationale**: The information is private owner data and does not belong in
  global `AppSetting`. One row is enough for the lightweight scope and follows
  the existing Go/GORM/SQLite persistence boundary.
- **Alternatives considered**: Add columns to `User` (mixes account identity
  and optional Coin Copilot context); separate goal/history tables (unneeded
  lifecycle complexity); Python memory or a new persistence service (violates
  the stateless agent boundary).

## R2. Profile API shape

- **Decision**: Add authenticated `GET /api/collector-profile` and
  `PUT /api/collector-profile`. GET returns neutral empty values when no row
  exists; PUT performs one complete validated replacement.
- **Rationale**: A small dedicated resource keeps profile privacy and
  validation explicit. Full replacement makes clearing values predictable and
  prevents partial invalid updates.
- **Alternatives considered**: Extend `/api/user/profile` (would enlarge an
  already broad account contract and risk exposing context through
  `/api/auth/me`); field-by-field endpoints (unnecessary); admin settings
  endpoints (wrong ownership).

## R3. Profile bounds

- **Decision**: Support nullable budget minimum/maximum, optional three-letter
  uppercase currency, and bounded unique arrays for preferred periods,
  preferred categories, excluded categories, preferred dealers, and
  collecting goals.
  Use small limits documented in `data-model.md`.
- **Rationale**: Bounds protect request size and prompt size while preserving
  every field required by the spec. Empty fields are neutral and never
  inferred.
- **Alternatives considered**: Unbounded free-form JSON (weak validation);
  normalized taxonomy tables (too large for optional advisory context);
  currency conversion (explicitly out of scope).

## R4. Curator composition

- **Decision**: Keep the existing Coin Copilot harness and its current
  `collection_summary`, `portfolio_review`, and `gap_analysis` tools. Add the
  profile as optional request context and update the existing planner prompt so
  curator requests compose those capabilities.
- **Rationale**: The three capabilities already exist in
  `copilot_collection_tools.py`, `coin_copilot.py`,
  `portfolio_review.py`, and `gap_analysis.py`. The feature needs synthesis
  and context, not another graph or platform.
- **Alternatives considered**: A new curator agent/tool family (duplicates the
  harness); direct Python database reads (forbidden); a separate curator REST
  workflow (unnecessary public surface).

## R5. Curator write boundary

- **Decision**: Profile context is advisory data only. It contains no action
  URL, credential, approval flag, or tool authority. Curator requests may
  create only normal Coin Copilot run/checkpoint/event records.
- **Rationale**: The current tool allowlist is read-only for the three required
  capabilities. Keeping writes outside the agent makes the read-only promise
  testable.
- **Alternatives considered**: Conversational save commands, generic action
  tools, or Python calls to `POST /api/coins` (all violate the spec).

## R6. Typed dealer eligibility

- **Decision**: A Copilot card is eligible only when the validated typed result
  says:

  ```text
  capability = market_search
  item.kind = dealer_listing
  item.verificationState = verified
  item.availability = available
  ```

  Any absent, partial, unknown, sold, malformed, or auction value is
  ineligible.
- **Rationale**: These fields already exist in the internal
  `CopilotSpecialistEvidence`/Pydantic `DealerListing`. The active spec names
  this exact boundary.
- **Alternatives considered**: Parse rendered `facts` text (fragile);
  infer from URL/provider/title (unsafe); recheck through an auction or new
  browser subsystem (excluded).

## R7. Public specialist projection

- **Decision**: Extend `CopilotSpecialistPublicEvidence` and the matching
  TypeScript type with optional typed dealer fields already present in the
  validated internal result: description, dealer name, listed price, currency,
  availability, ruler, denomination, era, and material.
- **Rationale**: The current public projection drops these fields and retains
  them only as display strings. The browser needs typed values for eligibility
  and `CoinSuggestion` mapping. This is an additive projection, not a provider
  or auction change.
- **Alternatives considered**: Parse `facts` strings (not a contract); post
  the card to a new server action endpoint (rejected architecture); expose raw
  provider payloads (privacy and contract violation).

## R8. Wishlist UI integration

- **Decision**: `CopilotRunProgress.vue` emits an eligible item from an explicit
  **Add to Wishlist** button. `CoinSearchChat.vue` adapts or forwards it to the
  existing `useCoinSearchChat.addToWishlist` flow.
- **Rationale**: The existing composable already owns `addingIdx`, `addedSet`,
  category/era confirmation, payload mapping, coin creation, and image
  attachment. Reuse avoids a second write workflow.
- **Alternatives considered**: A new review modal, staged action, or new
  composable (duplicates shipped behavior); model-triggered save (forbidden).

## R9. Dealer result to `CoinSuggestion`

- **Decision**: Adapt only typed dealer fields:
  title → name; description → description; dealer name → source name; listed
  price/currency → display price for the existing parser; source URL → source
  URL; ruler/denomination/era/material → matching fields. Category may be empty
  and follows the existing safe fallback/confirmation behavior. Image URL may
  be empty because the existing flow first attempts source-page scraping.
- **Rationale**: `buildWishlistCoinPayload` already bounds and maps only the
  accepted wishlist fields and sets `isWishlist: true`.
- **Alternatives considered**: Map `facts` text, auction fields, purchase
  fields, storage, sold state, or arbitrary provider metadata (all semantically
  wrong).

## R10. Category, era, and image behavior

- **Decision**: Reuse `resolveCategoryAndEra` and
  `CategoryEraConfirmModal.vue`. After successful `createCoin`, retain the
  current source scrape → proxy → upload sequence and fallback to typed image
  URL if available. Image failure is reported as a warning and never retries
  coin creation.
- **Rationale**: These are shipped safeguards and match the rewritten spec.
- **Alternatives considered**: New validation endpoints or image pipeline
  (duplicate); making image success transactional with coin creation (changes
  existing semantics).

## R11. Duplicate behavior

- **Decision**: Preserve `addingIdx` and `addedSet` for in-browser repeated
  clicks. Reuse `CoinRepository.FindWishlistByReferenceURL` inside the existing
  `CoinService.CreateCoin` transaction for wishlist coins with a non-empty
  reference URL. A duplicate returns the existing clear conflict behavior and
  does not create another coin.
- **Rationale**: The repository already has the owner-scoped lookup, and
  `CreateCoin` already owns the canonical transaction. The SQLite connection
  uses immediate transactions, allowing the lookup and insert to serialize
  without an action table.
- **Alternatives considered**: Wishlist action/outcome tables (rejected);
  frontend-only protection (cannot cover retries/tabs); a new source identity
  model (disproportionate).

## R12. Auction exclusion

- **Decision**: Do not modify or call auction subsystem models, repositories,
  services, routes, configuration, providers, or UI. `auction_search` and
  `auction_lot` fail the UI eligibility predicate.
- **Rationale**: The rewritten spec makes auction work an absolute exclusion.
  The existing specialist renderer may continue displaying auction evidence,
  but Feature 363 adds no action to it.
- **Alternatives considered**: Reusing auction cards or auction status for
  wishlist creation (explicitly rejected).

## R13. ADR decision

- **Decision**: Delete ADR 0018 and do not replace it.
- **Rationale**: The reduced design does not alter service boundaries, add a
  new provider, introduce a semantic workflow platform, or require a complex
  migration. Existing architectural rules fully determine the implementation.
- **Alternatives considered**: Rewrite ADR 0018 around the smaller scope
  (unnecessary documentation overhead under Principle IV).

## Clarification resolution

All technical choices needed by the rewritten spec are resolved. There are no
remaining `NEEDS CLARIFICATION` items.
