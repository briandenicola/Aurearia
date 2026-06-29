# Research: Wishlist Search Alerts for Acquisition Ideas

## Decision: Go API owns alert persistence, lifecycle, and conversion

**Rationale**: The constitution requires clear Handler → Service → Repository → Database ownership in the Go API and a stateless Python agent. Existing availability checks already store runs/results in Go models and update wishlist coin status from Go services. Search alerts similarly need durable user scoping, criteria snapshots, run history, candidate lifecycle, duplicate suppression, and explicit wishlist conversion, all of which are application business decisions rather than agent responsibilities.

**Alternatives considered**:
- Store candidates in Python agent memory or files: rejected because the agent is stateless and must not own database state.
- Convert agent results directly to wishlist items from Python: rejected because it bypasses explicit user review, Go service validation, transactions, and owner scoping.
- Reuse `AvailabilityRun` / `AvailabilityResult`: rejected because availability validates already-saved wishlist URLs while alerts discover new acquisition candidates.

## Decision: Use a typed stateless agent discovery contract behind `AgentProxy`

**Rationale**: Existing `AgentProxy.CheckAvailability()` demonstrates a non-streaming typed Go-to-Python call. Existing `/api/search/coins` is SSE/chat-oriented and returns suggestion JSON embedded in assistant messages, which is not an ideal durable service contract for persistence. v1 should add or adapt a typed non-streaming endpoint such as `/api/search/alerts` that accepts an alert criteria snapshot and returns candidates plus field-level provenance. The agent remains stateless and returns only source-backed facts, partial/unverified indicators, and reasons for match.

**Alternatives considered**:
- Parse current SSE chat output in the Go API: rejected because it is brittle, mixes chat UX with persistence, and weakens contract tests.
- Add custom per-dealer scrapers for each marketplace: rejected for v1 per feature constraints and maintenance/rate-limit risk.
- Let Vue call Python directly: rejected by service-boundary rules.

## Decision: MVP uses manual Run Now and in-app candidate review; scheduling/notifications are deferred

**Rationale**: The feature requires cadence metadata but explicitly allows push/email/digest/scheduled delivery to be deferred. Existing availability scheduler is oriented around checking saved wishlist URLs for all users and notifications for unavailable saved items. Reusing it for discovery now risks conflating workflows and increasing outbound traffic before per-alert run limits and review noise controls are proven.

**Alternatives considered**:
- Implement scheduled alert execution in v1 using `AvailabilityScheduler` patterns: deferred because the MVP value is manual discovery/review and current scheduler semantics do not match per-alert discovery.
- Send Pushover/email/digest notifications when candidates are found: deferred because in-app review is MVP and notifications could imply unverified availability.
- Ignore cadence entirely: rejected because cadence preference is part of the collector-facing alert model and should be stored for future scheduling.

## Decision: Duplicate suppression key uses canonical source URL strongest, plus normalized title, observed price, and alert ID

**Rationale**: The feature specification requires duplicate suppression using canonical source URL + normalized title + observed price + alert ID, URL strongest. Source URL canonicalization handles redirects, tracking parameters, and dealer URL variants, while title/price/alert identity helps suppress repeats when URLs vary or are missing. Duplicate relationships to existing wishlist items and converted candidates must be shown before allowing duplicate wishlist creation.

**Alternatives considered**:
- URL-only identity: rejected because dealer redirects and changed URLs can create duplicates.
- Global duplicate suppression across all alerts: rejected because the same candidate can be independently relevant to multiple alerts with separate review states.
- Fuzzy title-only matching: rejected as too noisy without URL/price/alert context.

## Decision: Candidate conversion to wishlist is explicit and source-backed

**Rationale**: Existing wishlist items are normal `Coin` records with `IsWishlist=true` and fields such as `Name`, `Ruler`, `Era`, `Material`, `Grade`, `PurchasePrice`, `ReferenceURL`, and `ReferenceText`. Candidate conversion should prefill only fields backed by candidate provenance, require collector review/input for missing or uncertain required fields, and then create a normal wishlist item in a Go transaction while linking the new coin back to the candidate for auditability.

**Alternatives considered**:
- Auto-create wishlist items for all candidates: rejected by spec and because it removes collector control.
- Prefill inferred fields as if verified: rejected because source facts must not be invented.
- Store converted items in separate candidate-only tables: rejected because future availability checks must operate on normal wishlist items after conversion.

## Decision: Preserve and regression-test existing wishlist availability architecture

**Rationale**: Existing availability checks read already-saved wishlist URLs, create `AvailabilityRun`/`AvailabilityResult`, update `Coin.ListingStatus`, and may notify users when saved listings become unavailable. Search alerts must never write those availability tables or update listing status during discovery. Regression coverage should prove that manual alert runs create only alert runs/candidates and that existing availability checks still validate saved wishlist URLs only.

**Alternatives considered**:
- Share availability result rows with alert candidates: rejected because it would blur history and UI semantics.
- Run availability keyword checks before saving every candidate: deferred unless needed as source provenance; discovery should rely on agent/search provenance and candidate lifecycle, not existing saved URL availability logic.

## Decision: Safe outbound HTTP rules remain mandatory for discovery

**Rationale**: Existing Python search tools use `safe_get()` and `validate_public_outbound_url()` to block metadata/local/private targets and validate redirects. Discovery expands outbound fetch volume, so v1 must keep these rules and add Go-side run limits/caps before any scheduling. User-facing errors should identify failed/rate-limited/partial runs without exposing internal agent configuration or secrets.

**Alternatives considered**:
- Fetch arbitrary URLs from alert source filters without validation: rejected for SSRF and privacy risk.
- Disable redirects to simplify canonicalization: rejected because dealer URLs commonly redirect; safe redirect validation already exists.
