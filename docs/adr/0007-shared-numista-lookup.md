# ADR 0007: Shared Numista Lookup Boundary

Date: 2026-08-11
Status: Accepted

## Context

Feature 341 replaces a direct handler-level catalog proxy and a separate photo
path that lacked shared query, error, ranking, caching, privacy, and
persistence semantics. Numista uses an instance credential and finite shared
allowance, while Aurearia requires broad-first results, deterministic
decision-support ranking, explicit selection, and rollback-safe draft
retention.

This decision implements Feature 341 FR-001–FR-029 under Constitution
Principles I, III, IV, V, VIII, IX, §17, and §21.

## Decision

### Shared application boundary

The Go API constructs one process-wide composition in `main.go`:

```text
NumistaHandler
  -> NumistaLookupService
       -> NumistaClient
       -> NumistaCache
       -> NumistaScorer
       -> NumistaTelemetry
```

`NumistaClient` is the only Numista HTTP boundary. Provider DTOs remain
private; handlers and Vue consume application-owned typed contracts. Calls are
context-aware and bounded, and the key is sent server-side only as
`Numista-API-Key`. Canonical links are reconstructed from validated IDs.

Broad lookup and enrichment are separate authenticated operations. Broad
lookup preserves the submitted visible query. Enrichment trims surrounding
whitespace, reranks the complete submitted set, and fetches details for a
server-selected bounded subset with concurrency two. Detail failure retains
the broad candidate.

### Cache and coalescing

Search and detail results use separate bounded in-memory TTL namespaces
(defaults: 500 searches, 5,000 details; 24 and 168 hours). Successful empty
searches and valid details are cacheable; provider failures are not. Expired
entries are deleted rather than served stale.

Equivalent misses coalesce onto one owned provider load. Fresh stored hits,
coalesced waiters, provider loads, and configuration bypasses remain distinct.
Credential/configuration checks occur before cache reads. Cache keys are
SHA-256 identities and contain no credential.

### Deterministic scoring

The pure `numista-v1` scorer owns ranking and explanations across exact ID,
title, issuer, denomination, mint, date, material, and inscription/visible
text. Missing candidate evidence is neutral. Stable ordering is score,
exact-ID match, enrichment completeness, normalized title, numeric ID, then
provider position. Scores are decision support, not attribution.

### Publication-owned telemetry

Telemetry is published only after cache ownership is known. The owning loader
publishes provider status, retry, latency, failure, and enrichment counts.
Coalesced waiters publish coalescing only; caller cancellation publishes
cancellation only. Superseded, orphaned, or late non-owning work publishes no
provider aggregate.

The 500-event in-memory ring records typed operational fields and a truncated
SHA-256 digest. It never records keys, queries, evidence text, images, provider
bodies, raw errors, or user identity. Admin health exposes sparse owned status
counts, provider-load latency, cache/coalescing, failures, cancellations,
enrichment, and observed 429 timing. It does not estimate remaining quota.

### Additive selected-reference migration

Quick Capture adds `quick_capture_draft_references`, a one-to-zero-or-one,
owner-scoped child row containing canonical `Catalog`, `Number`, and `URI`.
No existing row is rewritten or backfilled.

Draft transactions persist only explicit selection. Omission preserves it, a
complete canonical pair replaces it, and explicit clear removes it. Promotion
copies exactly that row into existing `coin_references` inside the existing
collection/wishlist transaction. Existing validation, ownership,
deduplication, rollback, and idempotency remain authoritative.

### Compatibility and rollout

Typed POST lookup/enrichment routes are additive. Deprecated
`GET /api/numista/search` remains for one compatibility release with its
legacy `{count,types}` shape. Photo and Quick Capture fields are additive, so
the backend deploys before the SPA.

## Alternatives Considered

- Keep separate clients: rejected because policy would continue to drift.
- Put provider HTTP in handlers/repositories: rejected by layered ownership.
- Let the browser call Numista: rejected because it exposes the shared key.
- Combine broad search and details: rejected because details delay first paint.
- Use SSE/background jobs: rejected as disproportionate for bounded details.
- Use SQLite/Redis cache: rejected because data is disposable and Redis adds a
  deployable dependency.
- Use provider ordering or LLM ranking: rejected as unowned or
  nondeterministic.
- Persist all candidates or infer the first: rejected by explicit selection.
- Add Numista columns directly to coins/drafts: rejected because structured
  `CoinReference` is authoritative.
- Flag-day replacement: rejected because compatibility lowers rollout risk.

## Consequences

### Positive

- Direct and photo workflows share one typed, live-provider-free test seam.
- Fresh reuse and coalescing conserve allowance without hiding stale state.
- Deterministic explanations improve review without claiming attribution.
- Publication ownership prevents fan-in/cancellation from corrupting metrics.
- Additive persistence and the legacy adapter permit independent SPA rollback.

### Negative and trade-offs

- Cache and telemetry disappear on restart and do not coordinate across API
  replicas.
- The compatibility period temporarily maintains POST and legacy GET.
- Enrichment consumes extra provider calls, currently capped at five.
- Material scoring changes require a future versioned decision.
- The additive table remains after rollback unless a separate destructive
  migration is approved.

## Security and Privacy

- Credentials remain server-side and appear only in the provider header.
- Request/body, ID, candidate-count, timeout, and response-size bounds limit
  abuse.
- Canonical Numista and HTTPS image validation prevent arbitrary links.
- Lookup requires authentication; health requires admin authorization.
- Guidance exposes no credentials, raw provider errors, or admin-only detail.
- Cache identities and telemetry omit full queries, inscriptions, label text,
  images, and user identity.

## Rollback

Roll back the SPA independently first. An older API ignores the additive table;
existing coins and promoted references remain readable. Restarting discards
cache and telemetry safely. Do not drop the table during emergency rollback;
destructive removal requires a separate migration and ADR.

## Related

- [Feature 341 specification](../../specs/341-improve-numista-lookup/spec.md)
- [Feature 341 plan](../../specs/341-improve-numista-lookup/plan.md)
- [Feature 341 data model](../../specs/341-improve-numista-lookup/data-model.md)
- [Numista integration guide](../features/numista-integration.md)
- [API reference](../api-reference.md#numista)
- [Deployment guide](../deployment.md#numista-backend-first-rollout)
- [ADR 0001](0001-record-architecture-decisions.md)
