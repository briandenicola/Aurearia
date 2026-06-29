# Data Model: Wishlist Search Alerts

## Overview

The feature adds Go-owned persistence for discovery alerts and acquisition candidates while preserving existing wishlist availability models. `WishlistSearchAlert`, `AlertRun`, `AlertCandidate`, `CandidateProvenance`, and `CandidateReviewAction` are new alert-domain entities. Existing `Coin` remains the wishlist item model after explicit conversion.

## Entity: WishlistSearchAlert

Collector-owned saved discovery criteria.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `user_id` | uint | yes | Owner; indexed; every query scoped with `OwnedBy(userID)` |
| `name` | string | yes | Display name, max 200 |
| `ruler_or_issuer` | string | no | Criteria |
| `coin_type` | string | no | Denomination/type keywords |
| `date_from` / `date_to` | int? | no | Year range; support BCE with negative values if existing UI allows |
| `mint` | string | no | Criteria |
| `material` | string | no | Match existing material vocabulary where possible |
| `grade_or_condition` | string | no | Free-form condition/grade criteria |
| `price_min` / `price_max` | decimal? | no | Non-negative; min <= max |
| `currency` | string | no | Default `USD` unless criteria specify otherwise |
| `dealer_preference` | string | no | Preferred dealer/source text |
| `source_filters` | JSON string/list | no | Allowed source/domain filters after validation |
| `keywords` | string | no | Free-text search terms |
| `notes` | text | no | Collector-facing notes |
| `cadence` | string | yes | `manual`, `daily`, `weekly`, `monthly`; v1 stores only |
| `is_active` | bool | yes | Disabled alerts cannot run |
| `last_run_at` | time? | no | Convenience display metadata |
| `created_at` / `updated_at` / `deleted_at` | time | yes | Soft delete preferred to preserve history |

### Validation

- Must be authenticated and owner-scoped.
- At least one meaningful search criterion beyond cadence/name is required.
- `price_min <= price_max` when both present.
- `date_from <= date_to` when both present.
- `cadence` must be one of supported enum values; only manual execution is promised in v1.
- Source/domain filters must be normalized hostnames or source labels; malformed URLs/domains rejected.

## Entity: AlertCriteriaSnapshot

Immutable copy of alert criteria stored on each run.

Implementation choice: store as JSON text on `AlertRun.criteria_snapshot` plus optional typed helper DTO. The snapshot must include all fields used to build the agent request, the alert name, active state at run time, and run limit settings.

## Entity: AlertRun

Execution record for manual and future scheduled discovery.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `alert_id` | uint | yes | FK to `WishlistSearchAlert` |
| `user_id` | uint | yes | Owner; duplicate for fast scoping |
| `trigger_type` | string | yes | `manual` in v1; future `scheduled` |
| `status` | string | yes | `queued`, `running`, `completed`, `failed`, `partial`, `rate_limited`, `cancelled` |
| `started_at` | time | yes | Set before agent call |
| `completed_at` | time? | no | Set when terminal |
| `duration_ms` | int64 | no | Terminal runs |
| `criteria_snapshot` | JSON text | yes | Immutable criteria |
| `result_count` | int | yes | Persisted candidate count |
| `new_count` | int | yes | New active candidates |
| `duplicate_count` | int | yes | Suppressed/updated duplicates |
| `dismissed_count` | int | yes | Candidates already dismissed and re-observed |
| `partial_warnings` | JSON/text | no | Non-secret user-facing warnings |
| `error_message` | string | no | Sanitized generic message |
| `rate_limit_status` | string | no | e.g., `ok`, `per_alert_limited`, `per_user_limited`, `source_limited` |
| `created_at` | time | yes | Audit |

## Entity: AlertCandidate

Source-backed acquisition idea produced by a run.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `user_id` | uint | yes | Owner |
| `alert_id` | uint | yes | Source alert |
| `run_id` | uint | yes | First or latest run that observed it |
| `source_url` | string | yes | Exact source URL from agent result |
| `canonical_source_url` | string | no | Canonicalized after safe redirect/URL normalization |
| `source_name` | string | no | Only if source-backed |
| `title` | string | yes | Source title or partial title from result |
| `normalized_title` | string | yes | Lowercase/punctuation-collapsed for duplicate key |
| `observed_price` | decimal? | no | Null when absent |
| `observed_currency` | string | no | Default/unknown explicit |
| `reason_for_match` | text | yes | Agent/source-backed reason |
| `last_seen_at` | time | yes | Updated on duplicate observations |
| `first_seen_at` | time | yes | Created timestamp semantics |
| `provenance_status` | string | yes | `verified`, `partial`, `unverified` |
| `lifecycle_state` | string | yes | `active`, `dismissed`, `converted`, `suppressed`, `needs_review` |
| `duplicate_key` | string | yes | Hash or string from alert ID + URL/title/price |
| `duplicate_of_candidate_id` | uint? | no | Relationship for suppressed duplicates |
| `matching_wishlist_coin_id` | uint? | no | Existing or converted wishlist relationship |
| `converted_coin_id` | uint? | no | New wishlist item after explicit conversion |
| `dismissal_reason` | string | no | Latest dismissal reason if dismissed |
| `created_at` / `updated_at` | time | yes | Audit |

### Duplicate key rules

1. Compute canonical URL if safe and available; remove tracking parameters and normalize scheme/host/path.
2. Normalize title by trimming, lowercasing, collapsing whitespace, and removing low-value punctuation.
3. Normalize observed price to numeric amount + uppercase currency when present.
4. Key includes `alert_id` to preserve per-alert review state.
5. Canonical URL is the strongest signal: same alert + same canonical URL updates/suppresses even if title or price changes; changed title/price should be recorded as provenance/change history.

## Entity: CandidateProvenance

Evidence for candidate facts.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `candidate_id` | uint | yes | FK |
| `field` | string | yes | `source_url`, `title`, `observed_price`, `source_name`, `availability_text`, `reason_for_match`, etc. |
| `value` | text | yes | Observed value |
| `source_url` | string | yes | Evidence URL |
| `observed_at` | time | yes | Agent/source observation time |
| `confidence` | string | yes | `high`, `medium`, `low` |
| `verification_state` | string | yes | `verified`, `partial`, `unverified` |
| `notes` | text | no | Non-secret details |

## Entity: CandidateReviewAction

Collector action history.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | uint | yes | Primary key |
| `candidate_id` | uint | yes | FK |
| `user_id` | uint | yes | Actor/owner |
| `action` | string | yes | `dismissed`, `restored`, `converted`, `criteria_adjusted`, `duplicate_warning_acknowledged` |
| `reason` | string | no | e.g., `irrelevant`, `duplicate`, `price_too_high`, `poor_provenance`, `other` |
| `metadata` | JSON text | no | Conversion coin ID, previous/new state, criteria fields changed |
| `created_at` | time | yes | Audit |

## Existing Entity: WishlistItem (`Coin`)

`Coin` already represents wishlist items with `is_wishlist=true`. Candidate conversion should create a normal `Coin` record through existing service validation and transactions.

### Conversion mapping

| Candidate field | Coin field | Rule |
|-----------------|------------|------|
| `title` | `name` | Prefill only if title provenance is present |
| `source_url` | `reference_url` | Required for conversion unless collector supplies a URL |
| `source_name` | `purchase_location` or `reference_text` | Use only if source-backed |
| `observed_price` | `purchase_price` and/or `current_value` | Prefill as wishlist target/listing price if source-backed; user reviews |
| Alert criteria ruler/material/mint/grade | corresponding coin fields | Prefill as criteria-derived suggestions only when clearly source-backed or user confirms |
| Candidate provenance summary | `notes` / `reference_text` | Preserve source-backed context without inventing details |

### Traceability

Preferred: add nullable `source_alert_candidate_id` to `Coin` for converted wishlist items. If avoiding a `Coin` schema change, store traceability on `AlertCandidate.converted_coin_id`; however, a `Coin` FK is stronger for auditability from the wishlist item.

## Relationships

- `User` 1 → many `WishlistSearchAlert`
- `WishlistSearchAlert` 1 → many `AlertRun`
- `AlertRun` 1 → many `AlertCandidate`
- `AlertCandidate` 1 → many `CandidateProvenance`
- `AlertCandidate` 1 → many `CandidateReviewAction`
- `AlertCandidate` 0/1 → 1 `Coin` after conversion
- Existing `AvailabilityRun` / `AvailabilityResult` remain unrelated except through eventual converted `Coin.ReferenceURL`

## State Transitions

```text
Alert: active → disabled → active
Alert: active/disabled → deleted (soft delete; no future runs)

Run: queued → running → completed
Run: queued/running → failed
Run: running → partial
Run: queued → rate_limited

Candidate: active → dismissed
Candidate: dismissed → active (restore)
Candidate: active/needs_review → converted
Candidate: active → suppressed (duplicate)
Candidate: active → needs_review (missing/uncertain conversion fields)
```

## Separation from availability checks

- Alert runs must not create `AvailabilityRun` or `AvailabilityResult` records.
- Alert discovery must not update `Coin.ListingStatus`, `ListingCheckedAt`, or `ListingCheckReason`.
- Existing availability checks continue to read `Coin` records where `is_wishlist=true` and `reference_url` is present.
- After conversion, the created wishlist `Coin` participates in normal availability checks on future availability runs.
