# Data Model: Improved Numista Lookup

**Date**: 2026-08-11  
**Spec**: [spec.md](./spec.md)

## Model boundaries

Only the selected Quick Capture reference is authoritative persisted data.
Lookup requests, candidates, scores, cache entries, and telemetry are typed
runtime value objects. Provider JSON is private to `HTTPNumistaClient` and is
never persisted or returned directly.

## Runtime value objects

### `NumistaLookupPath`

Enum: `direct`, `photo`.

- Required on every broad lookup and telemetry event.
- It describes the initiating application workflow, not a provider parameter.

### `NumistaEvidence`

| Field | Type | Rules |
|---|---|---|
| `title` | string? | trim; max 200 |
| `issuer` | string? | ruler or issuing authority; max 200 |
| `denomination` | string? | max 100 |
| `mint` | string? | max 200 |
| `dateText` | string? | source range/era text; max 100 |
| `material` | string? | max 100 |
| `obverseInscription` | string? | accepted for scoring, max 500 |
| `reverseInscription` | string? | accepted for scoring, max 500 |
| `visibleText` | string? | photo label/other text, max 500 |
| `exactNumistaId` | integer? | positive |

At least the effective query must be non-empty; evidence may be empty when the
collector manually enters a query. The exact strings are request-scoped and
must not enter telemetry/cache diagnostics. Normalized internal evidence is
bounded to 100 distinct tokens.

### `NumistaLookupRequest`

| Field | Type | Rules |
|---|---|---|
| `query` | string | trim non-empty; max 500 |
| `path` | `NumistaLookupPath` | required |
| `evidence` | `NumistaEvidence` | required object, may contain no fields |

The response repeats `query` exactly after surrounding whitespace is trimmed.
Internal canonicalization does not change the displayed query.

### `NumistaCandidate`

Application-owned candidate; no provider payload passthrough.

| Field | Type | Rules |
|---|---|---|
| `id` | integer | positive stable Numista type ID |
| `canonicalUrl` | string | generated from ID; HTTPS Numista host |
| `title` | string | non-empty after mapping |
| `issuer` | string? | safe provider text |
| `denomination` | string? | usually detail-stage |
| `mint` | string? | usually detail-stage |
| `minYear` / `maxYear` | integer? | astronomical year; `min <= max` when both |
| `yearDisplay` | string? | application-formatted BCE/CE-safe display |
| `material` | string? | usually detail-stage |
| `obverseInscription` | string? | bounded display detail |
| `reverseInscription` | string? | bounded display detail |
| `obverseThumbnail` / `reverseThumbnail` | string? | validated HTTPS image URL or omitted |
| `providerPosition` | integer | zero-based stable fallback |
| `enrichmentState` | enum | see below |
| `assessment` | `RelevanceAssessment` | always present |

Malformed candidates without positive ID or title are discarded. Missing
optional fields do not discard a candidate.

### `EnrichmentState`

Enum:

- `not_requested`: broad result only;
- `enriched`: fresh provider detail applied;
- `cached`: fresh detail cache applied;
- `failed`: detail attempted but broad candidate retained.

### `RelevanceAssessment`

| Field | Type | Rules |
|---|---|---|
| `scoringVersion` | string | `numista-v1` |
| `score` | integer | 0–100 |
| `band` | enum | `strong`, `possible`, `weak` |
| `reasons` | array of `RelevanceReason` | stable order by weight then code |

`RelevanceReason` has:

- `field`: one of `exact_id`, `title`, `issuer`, `denomination`, `mint`,
  `date`, `material`, `inscription`;
- `kind`: `match`, `conflict`, or `unavailable`;
- `code`: stable machine code such as `date_overlap`, `material_conflict`,
  `candidate_value_missing`;
- `label`: short safe collector-facing text. It must not echo full inscription
  or visible-label values.

Score formula:

```text
weighted = Σ(weight[field] × similarity[field])
applicable = Σ(weight[field]) for request-present fields
score = applicable == 0
  ? 50
  : clamp(round(50 + 50 × weighted / applicable), 0, 100)
```

Similarities are deterministic values in `[-1,1]`. Fixed v1 weights are
35/15/12/12/10/8/5/3 for exact ID/title/issuer/denomination/mint/date/material/
inscription. Sorting is score descending, exact-ID match, enrichment
completeness, normalized title, numeric ID, provider position.

### `LookupStatus`

Enum and meaning:

| Status | Meaning |
|---|---|
| `success` | one or more usable broad candidates |
| `empty` | configured/reachable search completed with zero usable candidates |
| `unconfigured` | no current server-side API key; checked before cache |
| `quota-limited` | provider returned 429 |
| `timeout` | application/provider deadline exceeded |
| `unavailable` | invalid credential, network/provider/malformed response, or internal lookup capability unhealthy |

Caller cancellation ends the HTTP request and records a safe cancelled
operation internally; it is not returned as a misleading provider status.

### `NumistaLookupOutcome`

| Field | Type | Rules |
|---|---|---|
| `status` | `LookupStatus` | required |
| `effectiveQuery` | string | exact trimmed submitted query |
| `candidates` | array | always present; empty on non-success |
| `guidanceCode` | string? | safe stable UI mapping |
| `retryAfterSeconds` | integer? | positive only when observed |
| `cache` | `CacheMetadata`? | present for cached/fresh successful or empty search |
| `stage` | enum | `broad`, `enriched` |

Expected domain statuses use HTTP 200. Invalid request syntax/ranges use 400.

### `CacheMetadata`

| Field | Type | Rules |
|---|---|---|
| `hit` | boolean | true only when a fresh entry was reused |
| `createdAt` | timestamp | UTC |
| `expiresAt` | timestamp | UTC, after creation |
| `ageSeconds` | integer | non-negative |

Search and detail cache entries are separate generic runtime records containing
mapped application values, creation/expiry, and a SHA-256 key. Entries are
bounded and disposable; expired entries are removed rather than served stale.

### `NumistaEnrichmentRequest`

| Field | Type | Rules |
|---|---|---|
| `query` | string | same rules as broad request |
| `path` | enum | required |
| `evidence` | object | same rules |
| `candidates` | broad candidate array | 1–50; IDs unique |

The service reranks first, enriches at most the configured leading subset
(default five), and returns all candidates. The client cannot expand the limit
by reordering or duplicating IDs.

### `NumistaTelemetryEvent`

In-memory only:

`occurredAt`, `path`, `operation`, `status`, `cacheHit`, `refreshed`,
`elapsedMs`, `candidateCount`, `detailAttemptCount`, `detailSuccessCount`,
`detailFailureCount`, `retryCount`, optional `retryAfterSeconds`, and
`correlationDigest`.

Prohibited fields: API key, query, evidence strings, images, raw provider
request/response, raw errors. The ring defaults to 500 events.

### `NumistaHealthSummary`

Admin-only aggregate:

- `configured`, `configurationValid`;
- `lastOutcome`, `lastCheckedAt`;
- status counts for the rolling ring;
- broad/detail request counts;
- cache hit and refresh counts/rate;
- p50/p95 elapsed milliseconds;
- enrichment attempted/succeeded/failed;
- `lastQuotaLimitedAt`, optional `lastRetryAfterSeconds`.

An empty ring is valid and returns zero counts/null timestamps.

## Persisted entity

### `QuickCaptureDraftReference`

One optional selected Numista reference for a Quick Capture draft.

| Column | Type | Constraints |
|---|---|---|
| `id` | uint | primary key |
| `draft_id` | uint | not null, unique index, FK logical ownership |
| `user_id` | uint | not null, owner index |
| `catalog` | varchar(32) | not null; service accepts only `Numista` |
| `number` | varchar(128) | not null; canonical positive decimal type ID |
| `uri` | varchar(2000) | not null; generated canonical URL |
| `created_at` | timestamp | UTC |
| `updated_at` | timestamp | UTC |

Relationships:

```text
User 1 ── * QuickCaptureDraft
QuickCaptureDraft 1 ── 0..1 QuickCaptureDraftReference
QuickCaptureDraft 1 ── * QuickCaptureDraftImage

promotion:
QuickCaptureDraftReference 0..1 ──copy──> CoinReference 0..1
```

The draft relation is returned as:

```json
{
  "selectedNumistaReference": {
    "catalog": "Numista",
    "number": "12345",
    "uri": "https://en.numista.com/catalogue/pieces12345.html"
  }
}
```

Internal IDs/user IDs of the selected-reference row need not be exposed.

## Validation and write semantics

### Selection validation

1. Selection is absent, or all of `catalog`, `number`, `uri` are present.
2. Catalog compares case-insensitively to `Numista` and is stored canonically.
3. Number parses as a positive integer and is stored in minimal decimal form.
4. URI is ignored/reconstructed from the number, then compared if supplied;
   mismatches are rejected.
5. Existing `CoinReferenceService` catalog/dedup rules apply before copy.
6. Owner ID always comes from JWT context, never request data.

### Draft create

- No selection: create no child row.
- Valid selection: create draft, lifecycle event, images, and reference in one
  repository transaction.
- Any failure rolls back all rows. File cleanup follows the existing draft
  failure path.

### Draft update

- Selection omitted: preserve current row.
- `clearSelectedNumista=true`: delete owner/draft-scoped row.
- Valid selection supplied: upsert/replace the one row.
- Clear plus selection: 400.
- Scalar/image/reference/event changes share one transaction.
- Validation failure leaves the prior selection unchanged.

### Draft lifecycle

```text
active ──update selection──> active
active ──clear selection───> active
active ──discard───────────> discarded (reference retained with draft history)
active ──CAS claim─────────> promoting
promoting ──transaction success──> promoted
```

Only active drafts can change selection. A candidate disappearing from later
searches does not mutate the stored selection.

### Promotion

Within `PromoteDraftTransaction`:

1. claim active owner-scoped draft (`active` → `promoting`);
2. reload draft, images and optional selected reference;
3. create collection/wishlist coin;
4. if selection exists, create exactly one mapped `CoinReference`;
5. transfer images and record value snapshot;
6. set promoted coin/timestamp/status;
7. append promoted lifecycle event;
8. commit.

Any failure rolls back all database changes. Repeated promotion of an already
promoted draft returns the existing coin and performs no copy. The existing
unique coin-reference index is the final duplication guard.

## Migration and rollback

- Add the new model to `AutoMigrate`; SQLite creates a new table/index only.
- No existing row is modified and no user backfill is required.
- Migration tests open a pre-feature schema, insert active/promoted drafts,
  run migration, and verify both remain readable with no selection.
- New code must treat missing relation as no selection.
- Rolling back to the old binary ignores the additive table. Promoted
  `CoinReference` rows retain their existing format and remain readable.
- A future destructive removal requires a separate migration; this feature
  does not drop data.
