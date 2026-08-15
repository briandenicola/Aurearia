# ADR 0009: Nomisma.org Authority Linking for Global Mint Locations

Date: 2026-08-14
Status: Accepted

## Context

Feature 343 (Phase 1) adds an optional, admin-confirmed link from a global
(non-private) `MintLocation` record to a Nomisma.org authority concept, with
visible CC BY 4.0 attribution. Nomisma is a single-admin, occasional,
click-driven workflow — not a high-volume enrichment pipeline like Numista
(ADR 0007) — so this decision deliberately keeps the boundary simpler:
no request coalescing, no retry policy, and a much smaller bounded cache.

Phase 2 (OCRE/RPC catalogue enrichment) is explicitly out of scope for this
feature and this ADR; it is a separate, future decision with its own
licensing research per `research.md`.

This decision implements Feature 343 FR-001–FR-0xx under Constitution
Principles I, III, IV, V, VIII, IX, §17, and §21.

## Decision

### Typed HTTP boundary

`NomismaClient` (interface) / `HTTPNomismaClient` (implementation) in
`src/api/services/nomisma_client.go` is the only Nomisma HTTP boundary. It
calls Nomisma's reconciliation endpoint (`https://nomisma.org/apis/reconcile`)
with a bounded timeout and response-size limit, and maps every outcome to a
typed `NomismaErrorKind` (`unavailable`, `no_match`, `invalid_response`,
`invalid_request`, `cancelled`) — never a generic/untyped error, and never a
raw provider error surfaced to callers. The request/response wire shape was
verified live on 2026-08-14: the `queries` URL parameter's value is the
query-identifier map directly (no outer `"queries"` key), the response is
that same map at the top level (no `"results"` wrapper), each candidate's
short `"id"` is expanded to `http://nomisma.org/id/<id>`, and `"match"` is
decoded from a JSON string rather than a JSON boolean.

### Bounded cache, no coalescing

`NomismaCache` in `src/api/services/nomisma_cache.go` is a 200-entry,
10-minute TTL, in-memory cache keyed on a SHA-256 digest of the normalized
query. `no_match` results are cached (negative caching); `unavailable`
results are never cached, so a transient outage does not "stick." There is no
singleflight/coalescing layer, unlike Numista's cache — Nomisma lookups are
single-admin-triggered and low-frequency, so duplicate concurrent loads are an
acceptable and simpler trade-off than the added complexity would justify.

### Service and handler layering

`MintLocationService` gains an optional `WithNomisma(...)` dependency (mirrors
the existing `WithGeocoding` chainable-setter pattern) and three methods —
`SearchNomisma`, `LinkNomismaGlobal`, `UnlinkNomismaGlobal` — all sharing one
`findGlobalMintLocation()` guard that 404s for any mint with a non-nil
`UserID` (private mint), so this feature can never expose or mutate a private
user's mint data. Link/unlink touch **only** `NomismaURI`, `NomismaLabel`, and
`NomismaLinkedAt` via a single `repo.Update` call with an explicit column map
— name, coordinates, region, and aliases are never read or written by these
code paths.

`MintLocationHandler` exposes three admin-only routes:

```text
GET    /admin/mint-locations/:id/nomisma/search?query=...
POST   /admin/mint-locations/:id/nomisma        {uri, label}
DELETE /admin/mint-locations/:id/nomisma
```

An upstream Nomisma failure (outage, timeout, malformed response) is always a
`200 {"status": "unavailable", "candidates": []}` — never a `5xx` — so mint
and coin CRUD are unaffected by a Nomisma outage. No candidate is ever
pre-selected; confirming a link requires an explicit admin action per
candidate.

### Additive schema, no backfill

`MintLocation` gains three additive, nullable/omit-empty columns:
`NomismaURI`, `NomismaLabel`, `NomismaLinkedAt`. GORM `AutoMigrate` adds them
without any backfill or reconciliation of existing rows — every existing mint
location is unaffected until an admin explicitly links it.

### Attribution

A single shared `NomismaAttribution.vue` component renders exactly
`Source: Nomisma.org · CC BY 4.0`, with `Nomisma.org` linking to the specific
confirmed concept URI (not the Nomisma homepage) and `CC BY 4.0` linking to
the Creative Commons license page, reusing the existing `SafeExternalLink.vue`
safe-URL pattern. It is rendered in the admin mint-management panel and the
Mint Map drawer (`MintCoinDrawer.vue`) whenever `nomismaUri` is present.

## Alternatives Considered

- Reuse `NumistaCache`'s singleflight/coalescing design: rejected as
  disproportionate for a single-admin, occasional workflow with no realistic
  concurrent-duplicate-request pressure.
- Silently match/backfill Nomisma links from name similarity: rejected — the
  spec requires an explicit admin confirmation step for every link; no
  automatic matching is ever performed.
- Extend `NumistaClient` to also call Nomisma: rejected — different
  provider, protocol (reconciliation vs. Numista's REST API), and license
  terms; a separate typed client keeps error/outage handling protocol-correct
  and avoids conflating two independently-evolving external dependencies.
- Allow linking private (per-user) mint locations: rejected by spec — Nomisma
  linking is global-authority-only; exposing it on private mints would leak
  user-specific mint data into an admin-only, cross-user feature.

## Consequences

### Positive

- Admins get a low-friction way to add authoritative external context to
  global mint locations without any risk of an outage blocking coin/mint CRUD.
- The shared `NomismaAttribution.vue` component keeps the attribution string
  and link targets consistent everywhere it is rendered.
- The additive-only schema and global-only guard make this feature safe to
  roll back (see Rollback) without touching existing data.

### Negative and trade-offs

- No coalescing means two admins searching the same query concurrently will
  both hit Nomisma (acceptable given the workflow's low frequency).
- The in-memory cache does not coordinate across API replicas and is lost on
  restart — acceptable since Nomisma lookups are cheap to repeat.
- Phase 2 (OCRE/RPC) cannot assume this ADR's CC BY 4.0 attribution wording
  applies to a different institution's license; a future ADR must confirm
  Phase 2's own licensing terms before reusing any UI pattern from this one.

## Security and Privacy

- No credentials are involved; Nomisma's reconciliation API is public.
- Requests are bounded by timeout and response size; queries are validated
  (non-blank, length-bounded) before being sent.
- All three Nomisma routes are admin-only and 404 for any private mint,
  regardless of caller.
- No raw provider response body, stack trace, or internal error is ever
  returned to the client — only the typed `NomismaErrorKind`-derived status.

## Rollback

Because the new columns are additive/nullable and no backfill runs, an older
API build ignores them and continues to serve existing mint locations
unaffected. Restarting the API discards the in-memory Nomisma cache safely.
Do not drop the new columns during an emergency rollback; destructive removal
requires a separate migration and ADR.

## Related

- [Feature 343 specification](../../specs/343-nomisma-mint-authority-linking/spec.md)
- [Feature 343 plan](../../specs/343-nomisma-mint-authority-linking/plan.md)
- [Feature 343 data model](../../specs/343-nomisma-mint-authority-linking/data-model.md)
- [Feature 343 contract](../../specs/343-nomisma-mint-authority-linking/contracts/nomisma-authority-linking.md)
- [ADR 0007: Shared Numista Lookup Boundary](0007-shared-numista-lookup.md)
- [ADR 0001](0001-record-architecture-decisions.md)
