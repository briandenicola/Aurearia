# Data Model: Collector Curator, Watchlist, and Provenance

## Relationship overview

```text
User 1 ── 0..1 CollectorProfile 1 ── * CollectingGoal
  │
  ├── Coin (collection or wishlist) / Availability / Auction records
  ├── CoinCopilotRun ── checkpoints containing bounded read-only snapshots
  └── WishlistActionStage 1 ── 0..1 WishlistActionOutcome ── 1 Coin (wishlist)
```

Recommendation, evaluation, risk finding, and profile snapshot are strict
contract values retained inside existing owner-bound Coin Copilot
checkpoint/event limits. They are not new tables.

## 1. CollectorProfile

Exactly one private preference root per owner.

| Field | Storage/type | Rules |
|---|---|---|
| `id` | uint PK | internal |
| `user_id` | uint, unique index | owner derived server-side; cascade/delete per existing user policy |
| `budget_min` / `budget_max` | nullable decimal | finite, `0..100000000`, min ≤ max |
| `currency` | char(3) | uppercase supported code; default owner display currency or `USD` |
| `favorite_periods_json` | bounded JSON array | 0..20 unique normalized strings, each 1..100 |
| `disliked_categories_json` | bounded JSON array | 0..20 unique normalized strings, each 1..100 |
| `preferred_dealers_json` | bounded JSON array | 0..50 unique normalized strings, each 1..200 |
| `version` | uint64 | starts 1; increments once per successful atomic update |
| `created_at` / `updated_at` | timestamp | UTC |

Unique index: `ux_collector_profiles_user_id(user_id)`.

Normalization trims Unicode whitespace, collapses internal whitespace and uses
case-folded comparison while preserving first accepted display spelling.
Unknown JSON fields/types or duplicate normalized values reject the whole save.
A missing row projects neutral defaults and version `0`; it is not persisted
until first save.

## 2. CollectingGoal

| Field | Storage/type | Rules |
|---|---|---|
| `id` | UUID/text PK | owner-generated cryptographically random identity; never model-generated |
| `user_id` | uint, indexed | duplicated owner for direct scoping |
| `collector_profile_id` | uint FK, indexed | parent |
| `title` | string(200) | trimmed, 1..200 |
| `description` | string(1000), nullable | untrusted data |
| `priority` | enum | `low`, `medium`, `high` |
| `state` | enum | `active`, `inactive` |
| `version` | uint64 | increments when goal content changes |
| `created_at` / `updated_at` | timestamp | UTC |

Indexes:
`ix_collecting_goals_user_state(user_id,state)` and unique
`ux_collecting_goals_profile_id_id(collector_profile_id,id)`. Maximum 50 goals
is enforced inside the profile update transaction. Omitting an existing goal
from a full-save request means delete only after optimistic profile-version
validation; a restore is a new explicit save before retention cleanup.

## 3. ProfileSnapshot (contract value)

| Field | Rules |
|---|---|
| `profile_version` | exact version read in one Go transaction; `0` for defaults |
| `profile_digest` | SHA-256 over canonical bounded profile/goal values |
| budgets/currency/preferences | normalized values, no invented defaults beyond documented neutral currency |
| `goals[]` | active goals only unless contract explicitly requests inactive; id/version/title/description/priority |
| `captured_at` | Go timestamp |

The snapshot is passed to Python and retained only within the existing
Coin Copilot completed-tool result. A run never rereads part of the profile.

## 4. CollectorRecommendation (contract value)

| Field | Rules |
|---|---|
| `id` | bounded run-local opaque id |
| `kind` | `strength`, `theme`, `gap`, `acquisition_idea` |
| `observed_facts[]` | typed owner-scoped collection fact refs |
| `recommendation` | bounded text explicitly labeled recommendation |
| `profile_effects[]` | `included`, `excluded`, `deprioritized`, `neutral` plus profile field/goal id |
| `confidence` | `low`, `medium`, `high` |
| `why_this_matters` | bounded text |
| `citations[]` | internal fact refs or validated URLs only |
| `limitations[]` | 1..10 bounded strings |

At most 20 recommendations, deterministic order by category then rank/id.

## 5. WatchlistEvaluation (contract value)

Input union:

- `wishlist_coin`: positive owned `coin_id`;
- `collecting_goal`: owned active goal id/version;
- `market_result`: owner-bound run/execution/tool-call plus
  `dealer_listing|auction_lot` canonical source identity/digest.

Output contains closed statuses for `goal_fit`, `budget_fit`,
`preference_fit`, `dealer_fit`, `owned_duplicate`, `wishlist_duplicate`,
`collection_coverage`, `source_quality`, `listing_state`, and
`price_comparability`: `match`, `conflict`, `unknown`, `not_applicable`.
It includes input origin, authoritative state refs/timestamps, evidence,
confidence, why-it-matters and limitations. At most 20 candidates. Currency
comparison is `unknown` when currencies differ and no existing source-backed
conversion is provided.

## 6. RiskFinding (contract value)

| Field | Rules |
|---|---|
| `kind` | `missing_provenance`, `unverified_claim`, `conflicting_claim`, `duplicate_image`, `broken_source_link`, `documentation_gap` |
| `tier` | `review_low`, `review_medium`, `review_high` |
| `needs_review` | literal `true`; rendered copy must contain “needs review” |
| `observed_facts[]` | facts only, each with evidence ref |
| `source_claims[]` | attributed claims, never rewritten as fact |
| `conflicts[]` | all supported sides |
| `recommendation` | bounded collector review step |
| `evidence[]` | validated citations/immutable internal refs |
| `confidence` | `low`, `medium`, `high` |
| `why_this_matters` | bounded |
| `limitations[]` | at least one |
| `dependency_state` | `baseline`, `feature_362`, `attribution_evidence_unavailable` |

At most 20 findings. `duplicate_image` additionally requires an existing
comparison evidence id, method/basis, compared image digests and limitations;
no images or provider payloads are copied.

## 7. WishlistActionStage

Immutable review snapshot; never a `Coin`.

| Field | Storage/type | Rules |
|---|---|---|
| `id` | opaque UUID/text PK | random; returned to owner |
| `user_id` | uint, indexed | server-derived |
| `version` | uint | literal `1`; edits create another stage |
| `source_kind` | enum | `dealer_listing`, `auction_lot` |
| `source_run_id` / `source_execution_id` / `source_tool_call_id` | bounded ids | retained owner-bound evidence locator |
| `source_result_identity_digest` | SHA-256 | canonical provider/source identity |
| `source_result_digest` | SHA-256 | exact retained normalized result |
| `provider` | closed registered provider id | not free-form |
| `source_url` / `canonical_source_url` | strings ≤2048 | validated HTTPS registered host |
| `observed_at` | timestamp | required |
| `listing_state` | closed Feature 361 state | preserved, not guessed |
| `price` / `currency` | nullable decimal/char(3) | paired when known; listing price only |
| `confidence` / `verification_state` | closed enums | preserved |
| `mapping_json` | strict bounded JSON | only allowed destination fields and provenance refs |
| `evidence_json` | strict bounded JSON | field provenance, limitations, duplicate warnings; no raw payload |
| `request_fingerprint` | SHA-256 | canonical source + mapping + evidence + owner-independent schema version |
| `stage_idempotency_key_hash` | SHA-256 | plaintext key never stored |
| `expires_at` / `created_at` | timestamp | immutable; max 15 minutes |

Indexes:

- unique `ux_wishlist_stage_owner_key(user_id,stage_idempotency_key_hash)`;
- `ix_wishlist_stage_owner_expiry(user_id,expires_at)`;
- `ix_wishlist_stage_owner_source(user_id,source_result_identity_digest)`.

Stage replay with same key/fingerprint returns this row. Key/fingerprint
mismatch conflicts. Cancel/close does not mutate the row; expiry is derived
from time.

## 8. WishlistActionOutcome

Immutable durable confirmation/audit result.

| Field | Storage/type | Rules |
|---|---|---|
| `id` | uint PK | internal |
| `user_id` | uint, indexed | server-derived |
| `stage_id` | UUID FK, unique | exact stage |
| `request_fingerprint` | SHA-256 | must equal stage |
| `confirmation_idempotency_key_hash` | SHA-256 | plaintext not stored |
| `source_kind` / `source_result_identity_digest` | copied closed identity | concurrency uniqueness |
| `outcome` | enum | `created`, `existing` |
| `coin_id` | uint FK | owned wishlist coin |
| `mapped_fields_json` | bounded string array | names only, no private values |
| `confirmed_at` | timestamp | UTC |

Indexes:

- unique `ux_wishlist_outcome_owner_confirm_key(user_id,confirmation_idempotency_key_hash)`;
- unique `ux_wishlist_outcome_owner_source(user_id,source_kind,source_result_identity_digest)`;
- unique `ux_wishlist_outcome_stage(stage_id)`.

Outcome and wishlist coin are inserted in one transaction. `existing` may be
used when the same owner/source is already linked by a prior outcome; manually
created URL duplicates return a review conflict before confirmation rather than
silently link unrelated data.

## 9. Existing Coin mapping

| Stage field | `Coin` destination | Rule |
|---|---|---|
| proven title | `Name` | required, sanitized |
| proven numismatic identity | matching `Denomination`, `Ruler`, `Era`, `Mint`, `Material`, `Grade` | only field-level-proven values passing existing validation |
| source URL | `ReferenceURL` | validated/canonicalized through existing reference path |
| supported observed listing price | `CurrentValue` | never `PurchasePrice`; no conversion |
| supported listing state | existing listing-status fields | only if destination enum supports exact meaning |
| constant | `IsWishlist=true` | required |

Never map purchase date/price/location, invoice/SKU, sold fields, storage,
visibility, images, notes, uncited catalog/provenance data, credentials or raw
provider fields. Existing manual wishlist items are never updated by this
action.

## 10. State transitions

```text
Profile: missing(default v0) -> saved(v1) -> updated(vN)
         saved(vN) -- stale vN-1 --> 409/no change
         flag disabled -> inert/read-safe defaults

Read-only run: accepted -> running -> completed|partial|paused|cancelled|failed
               replay -> same checkpoint result, no new provider work

Stage: created -> confirmed outcome
       created -> expires (derived, no mutation)
       created -> abandoned/cancelled (no outcome, no Coin)
       edited -> new immutable stage

Confirmation: validate -> transaction(created Coin + outcome + journal)
              replay -> same outcome
              expired/cancelled/foreign/stale/tampered/duplicate -> no write
```

Cancellation is checked before each capability, after awaits, before staging,
and immediately before the confirmation transaction. Once the transaction
commits, cancellation returns the stable idempotent outcome rather than
compensating/deleting the coin.

## 11. Migration, defaults, rollback, retention and privacy

Migration order:

1. `CollectorProfile`;
2. `CollectingGoal` and ownership indexes;
3. `WishlistActionStage`;
4. `WishlistActionOutcome` and unique idempotency/source indexes;
5. enable no flags automatically.

No destructive backfill. Existing users have virtual neutral profile defaults.
Rollback first disables flags, then rolls back code; additive tables remain
inert. Do not drop tables/columns until retention/export and downgrade
compatibility are explicitly designed in a later migration.

All four tables are private owner data. They are excluded from public/follower
queries, public gallery, cross-user analytics and privacy-unsafe logs. Account
export/deletion must include/remove them according to existing owner-data
policy. Stage cleanup may delete expired unconfirmed stages after the existing
short-lived action retention window; confirmed stage/outcome evidence follows
audit retention and must remain bounded.
