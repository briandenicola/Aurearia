# Research: Collector Curator, Watchlist, and Provenance

## R1. Dedicated profile storage

- **Decision**: Add one `CollectorProfile` per owner plus separate
  `CollectingGoal` rows. Store bounded preference lists as validated JSON on
  the profile; update profile and complete goal set atomically with optimistic
  `version`.
- **Rationale**: Current `AppSetting` is global/admin configuration and current
  `User` fields are account/privacy/integration toggles. Neither safely models
  private collecting intent. Separate goal rows provide stable owner-generated
  identities and lifecycle timestamps without adding 50 opaque objects to one
  blob.
- **Alternatives considered**: Admin settings (privacy/ownership mismatch);
  more `User` columns (schema sprawl and mixed responsibility); Python memory
  (stateless boundary violation); one unvalidated JSON document (weak
  identities and constraints).

## R2. Snapshot persistence

- **Decision**: Do not add a profile-snapshot table. Put the validated bounded
  profile version, normalized values, digest, and relevant goal versions in
  the existing Coin Copilot completed-tool checkpoint/result that already
  explains each read-only run.
- **Rationale**: Checkpoints are owner-bound, bounded, replayable, digested and
  already retain tool facts. Recommendations need explainability for the run,
  not a second mutable profile history.
- **Alternatives considered**: New snapshot/event store (duplicate
  durability); reread current profile during inference (mixed-version race);
  Python cache (statelessness violation).

## R3. Curator composition

- **Decision**: Add one fixed read-only curator capability that consumes the
  existing collection summary, portfolio review and gap-analysis projections.
  Go supplies one owner/profile snapshot; Python ranks and explains without new
  collection queries.
- **Rationale**: Current `AgentRepository` portfolio summary,
  `CollectionToolsService`, and Coin Copilot collection tools are canonical.
  The feature needs synthesis, not another agent or repository.
- **Alternatives considered**: A new curator graph/platform (redundant);
  direct Python database reads (forbidden); recomputing portfolio gaps in Vue
  (contract drift).

## R4. Watchlist input boundary

- **Decision**: Accept only discriminated, Go-resolved snapshots of an owned
  wishlist coin, active collecting goal, or retained eligible Feature 361
  `dealer_listing`/`auction_lot`.
- **Rationale**: This implements the settled scope and composes authoritative
  wishlist availability, auction state, collection coverage and specialist
  evidence without new provider work.
- **Alternatives considered**: Direct URL or saved-search ingestion (deferred);
  model-supplied identifiers (cross-user risk); fresh scraping during
  evaluation (new automation and stale/replay ambiguity).

## R5. Eligible market result

- **Decision**: Go re-resolves an item by owner-bound run, execution, tool-call,
  canonical source identity and retained result digest. Eligibility requires a
  successful retained Feature 361 dealer/auction item, approved provider and
  HTTPS host, field provenance, non-cancelled run, valid observation timestamp,
  and freshness policy. Raw provider responses are never accepted.
- **Rationale**: Vue state and model prose are not authority. Feature 361's
  normalized specialist result already supplies the minimum evidence.
- **Alternatives considered**: Post the card JSON back and trust it (tampering);
  accept final Markdown (untyped); call the provider again at confirmation
  (new side effect and race).

## R6. Risk vocabulary

- **Decision**: Closed kinds are `missing_provenance`,
  `unverified_claim`, `conflicting_claim`, `duplicate_image`,
  `broken_source_link`, and `documentation_gap`. Closed tiers are
  `review_low`, `review_medium`, `review_high`; all output contains the literal
  phrase “needs review” and separates observed facts, source claims, conflicts,
  recommendations, evidence, confidence, why-it-matters and limitations.
- **Rationale**: The schema makes cautious language testable and prevents
  inference from becoming an accusation or authenticity determination.
- **Alternatives considered**: Free-form severity/labels (unsafe drift); hide
  low confidence (conceals uncertainty); fraud/authenticity scores (out of
  scope and harmful).

## R7. Feature 362 dependency

- **Decision**: Ship profile, curator, watchlist, baseline documentation/link
  review, and confirmed wishlist action before Feature 362. Gate only findings
  that require persisted Deep Analysis attribution, provider contradictions,
  proposal confidence or citations. After 362, consume its validated
  projection without reinterpretation.
- **Rationale**: The baseline has independent value and no need to block on
  attribution handoff. Missing 362 evidence is not negative evidence.
- **Alternatives considered**: Block the entire suite (unnecessary coupling);
  infer attribution from existing coin text (unsupported); copy Deep raw JSON
  (contract/privacy drift).

## R8. Wishlist mapping

- **Decision**: Map only proven destination-valid fields: title→`Name`;
  supported denomination/ruler/era/mint/material/grade fields to the matching
  `Coin` fields; source URL→`ReferenceURL`; observed supported listing price→
  `CurrentValue`; and supported listing state→existing listing-status fields.
  Always set `IsWishlist=true`. Preserve currency/provider/source identity/
  observation/provenance/evidence/limitations in the linked stage/outcome, not
  unrelated `Coin` fields.
- **Rationale**: Current normal creation is
  `AddCoinPage.vue` → `stores/coins.ts:addCoin` → `handlers/coins.go:Create` →
  `CoinService.CreateCoin`. Reusing its validation and transaction seam keeps
  wishlist semantics while a narrow DTO prevents accidental population of
  purchase/private/manual fields.
- **Alternatives considered**: Submit the full Add Coin form DTO (overbroad);
  put provider metadata in notes/purchase location (semantic corruption);
  set `PurchasePrice` from a listing (not a purchase); copy images (duplicate
  and unsafe).

## R9. Stage and audit schema

- **Decision**: Add immutable `WishlistActionStage` and
  `WishlistActionOutcome`. Stage stores the exact bounded normalized evidence
  and mapping snapshot with expiry/fingerprint. Outcome is inserted only by
  confirmation in the same transaction as canonical coin creation and journal
  append. Unique owner/source and idempotency indexes provide replay safety.
- **Rationale**: Coin Copilot checkpoints can establish source evidence but are
  prunable and cannot represent user confirmation. Coin journals describe a
  created coin but cannot authorize or linearize creation.
- **Alternatives considered**: Mutable staged coin (partial write before
  consent); session/local storage (tamper/restart unsafe); checkpoint-only
  confirmation (wrong authority); journal-only idempotency (post-write race).

## R10. Explicit mutation boundary

- **Decision**: Only Vue's rendered result-card button may start staging.
  Stage and confirm are authenticated public Go endpoints, never Coin Copilot
  callback tools. Confirmation requires a second event and exact stage version.
  Python receives neither route nor DTO nor generic credential.
- **Rationale**: This proves mutation is outside model tool selection and
  separates recommendation from consent.
- **Alternatives considered**: Conversational “save this” intent (ambiguous);
  model-generated action URL (self-approval path); Python calling public APIs
  with a user token (credential/boundary violation).

## R11. Idempotency and duplicate policy

- **Decision**: Hash idempotency keys at rest. Same owner/key/fingerprint
  returns the original stage/outcome; key reuse with another fingerprint is
  `409`. Confirmation serializes by unique
  `(user_id, source_kind, source_result_identity_digest)` and rechecks existing
  wishlist normalized URL/source linkage before calling `CoinService`.
- **Rationale**: This covers browser retries, two tabs, replay and retained
  source duplicates while preserving a warning for manually created wishlist
  items.
- **Alternatives considered**: Title matching only (false positives); frontend
  de-duplication (racy); acknowledge-and-force duplicate creation (violates
  stable source identity).

## R12. URL, citation and provider data

- **Decision**: Reuse specialist registered-host rules,
  `validate_outbound_url`/`safe_get`, redirect revalidation, and wishlist
  canonicalization. Require HTTPS, no user-info, no private/loopback/link-local/
  metadata targets, max 2,048 characters, and citation/source match. Normalize
  only for comparison. Treat all provider/goal text as delimited untrusted data
  and reject instruction/token-shaped content from promoted factual fields.
- **Rationale**: Existing controls are stronger and better tested than a new
  validator; display URLs and canonical identities serve different purposes.
- **Alternatives considered**: Trust model citations (fabrication/SSRF); invent
  home-page URLs (false evidence); duplicate weaker validation in Vue.

## R13. Duplicate images and stale listings

- **Decision**: Feature 363 does not compare or fetch new images. It can cite a
  duplicate-image finding only from an existing defensible comparison record
  with algorithm/match basis and limitations. Listing state remains an
  observation with timestamp; current wishlist availability/auction records
  override specialist prose, and stale/contradictory state lowers confidence.
- **Rationale**: Absence of a match or fresh listing proves nothing, and the
  feature cannot add scraping/image automation.
- **Alternatives considered**: Perceptual hashing provider images here (new
  automation/dependency); treat last observed listing as current (stale claim);
  choose one conflict silently.

## R14. Flags and rollback

- **Decision**: Use five default-off global settings following
  `SettingsService` patterns. Migration is additive and non-destructive.
  Disable later slices independently; keep profiles, stages, outcomes and
  existing completed results readable/inert.
- **Rationale**: Staged rollout and quick rollback cannot require destructive
  migration or disable existing collection/wishlist/chat behavior.
- **Alternatives considered**: One all-or-nothing flag (poor isolation);
  delete rows on rollback (data loss); store profile in global settings.

## R15. Observability and privacy

- **Decision**: Log identifiers/digests, closed outcomes, counts, duration,
  versions and truncation only. Exclude preference/goal text, budgets,
  collection facts, evidence text, URLs, prompts, provider payloads,
  credentials and raw errors. Audit rows contain only the bounded evidence
  necessary to explain the owner's durable action.
- **Rationale**: Diagnostics do not justify leaking private collection intent
  or provider content.
- **Alternatives considered**: Full payload logging (privacy/prompt-secret
  risk); no audit linkage (unexplainable assisted write).

## R16. Current code and UI composition

- **Decision**: Extend `SettingsPage.vue`/settings component conventions,
  `CopilotRunProgress.vue`'s `specialistResult` card,
  `CoinCopilotSpecialistResult`, and modal patterns. Use the current Vue API
  client and Go protected routes.
- **Rationale**: These are the shipped account, evidence and confirmation
  surfaces. New standalone pages or stores are unnecessary.
- **Alternatives considered**: New curator app/page (fragmentation); browser
  direct Python call (forbidden); hardcoded styles/icons (constitution
  violation).

## R17. OpenAPI and contract drift

- **Decision**: Add profile, read-only workflow, stage and confirmation public
  REST contracts with Swagger annotations. Internal Python execution mirrors
  are documented but are not public paths. Regenerate generated Swagger and
  `docs/openapi.json`; run route-drift and shared fixture tests.
- **Rationale**: Public Go authority must be discoverable and generated from
  one source without exposing internal callback topology.
- **Alternatives considered**: SSE-only prose (weak contract); publish Python
  endpoints (wrong boundary); hand-edit generated files (drift).

## Clarification resolution

All planning questions are resolved by the binding product decisions, current
code evidence, and upstream feature contracts. There are no remaining
`NEEDS CLARIFICATION` items.
