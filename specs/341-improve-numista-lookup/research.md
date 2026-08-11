# Phase 0 Research: Improved Numista Lookup

**Date**: 2026-08-11  
**Spec**: [spec.md](./spec.md)

All technical-context questions are resolved. The official Numista OpenAPI
document consulted during planning is v3.33 at
`https://en.numista.com/api/doc/swagger.yaml?v=3.33`.

## 1. Shared client boundary

**Decision**: Introduce one application-owned `NumistaClient` interface and
`HTTPNumistaClient` implementation, composed by a `NumistaLookupService`.
Direct and photo paths depend on the lookup service, never on `http.Client` or
provider JSON.

**Rationale**: The current direct handler proxies untyped JSON while
`CoinLookupService` independently implements a second provider client. A
single boundary makes errors, cancellation, caching, scoring, and DTOs
consistent and gives tests a live-provider-free seam. It also restores
Constitution Principle I because provider/business behavior leaves the
handler.

**Alternatives considered**:

- Keep two clients and share helper functions: rejected because policy and
  status behavior could still diverge.
- Put provider HTTP in a repository: rejected because repositories own
  database access, not outbound HTTP.
- Let Vue call Numista: rejected by Constitution Principle II and because it
  would expose the API key.

## 2. Provider endpoints and error taxonomy

**Decision**: Use official v3 `GET /types` for broad search and
`GET /types/{type_id}` for detail. Authenticate only with
`Numista-API-Key`. Treat documented 429 as `quota-limited`; use an explicit
typed taxonomy for invalid request, unauthorized, quota, timeout,
unavailable, malformed response, and cancellation. Parse standard
`Retry-After` if present but do not assume undocumented quota headers.

**Rationale**: Numista OpenAPI v3.33 documents both endpoints, search count up
to 100, and 400/401/429 responses. It does not promise remaining-quota
headers. The application can accurately expose observed quota events without
inventing a remaining allowance.

**Alternatives considered**:

- Scrape the Numista website: rejected as brittle and outside the API contract.
- Report an estimated remaining monthly quota: rejected because other clients
  may share the key and no authoritative response header is documented.
- Forward provider status/body: rejected by Principles III/V and the safe-error
  requirement.

## 3. Retry and timeout policy

**Decision**: Make all calls context-aware. Search defaults to four seconds,
detail to three seconds. Retry once only for connection resets and
502/503/504, with short context-aware jitter. Never retry 400, 401/403, 429,
malformed JSON, or caller cancellation.

**Rationale**: Search must terminate inside the five-second p95 target, while
idempotent transient GET failures deserve one bounded recovery attempt.
Automatically retrying 429 wastes a shared allowance and delays accurate user
guidance.

**Alternatives considered**:

- Generic three-attempt exponential retry: rejected for latency/quota cost.
- Fifteen-second current timeout: rejected because it cannot meet NFR-001.
- No retry: rejected because a single transient gateway/reset should not
  produce avoidable unavailability.

## 4. Two-stage browser contract

**Decision**: Make broad lookup and detail enrichment separate authenticated
application requests. The broad response is rendered immediately; Vue then
submits only the service-ranked leading IDs to enrichment. The server
revalidates and caps IDs and returns reranked full candidates.

**Rationale**: A single synchronous response cannot guarantee that usable
broad results are visible before detail calls. SSE or background jobs are
unnecessary for at most five bounded reads. Two ordinary typed REST calls are
simple, cancellable, independently cacheable, and satisfy NFR-003.

**Alternatives considered**:

- Search and enrich in one request: rejected because details delay first paint.
- SSE: rejected as disproportionate and creates reconnect/state complexity.
- Persisted asynchronous jobs: rejected because detail data is disposable and
  the workflow is short-lived.
- Have Vue call provider details directly: rejected due to key exposure and
  service-boundary violation.

## 5. Editable photo query and NGC-first behavior

**Decision**: Photo analysis returns a typed proposed query and evidence but
does not automatically call Numista. If a usable NGC result exists, the
proposal remains suppressed unless the collector explicitly opens Numista
lookup. Otherwise the user may edit and submit the proposal through the same
broad lookup route as direct lookup.

**Rationale**: Searching during image analysis violates FR-005 because the
collector cannot edit before the first search. It also spends quota on noisy
vision output. This decision preserves the established NGC-first rule.

**Alternatives considered**:

- Eager search followed by editable retry: rejected because the first
  provider request remains unreviewed.
- Send the images to Numista image search: rejected as out of scope, paid-plan
  dependent, and a new privacy surface.

## 6. Application scoring

**Decision**: Use a versioned, injected v1 scoring configuration with fixed
weights: exact ID 35, title 15, issuer/ruler 12, denomination 12, mint 10, date
8, material 5, inscription/visible text 3. Normalize each applicable
dimension to `[-1,1]`, map the weighted result around neutral 50 to `[0,100]`,
and apply a fully specified stable tie-break.

**Rationale**: Fixed versioned weights are reproducible, testable, and safer
than hidden provider ordering. Centering missing information at neutral
distinguishes absence from conflict. Exact ID dominates only when the request
actually carries one. Structured reason codes support concise explanations
without echoing sensitive text.

**Alternatives considered**:

- Provider ordering only: rejected by FR-007.
- LLM scoring: rejected as nondeterministic, costly, and prohibited Go-side
  agent logic.
- Admin-editable weights at launch: rejected as premature complexity and a
  hard-to-support config contract. The scorer remains injectable for future
  versions.
- Edit distance over one concatenated string: rejected because it cannot
  explain field conflicts or date compatibility.

## 7. Normalization and dates

**Decision**: Preserve the exact user query for display, while internal
matching uses Unicode NFKC, case folding, punctuation/whitespace collapse,
token deduplication and strict length/token bounds. Parse signed astronomical
years and BCE/BC/CE/AD intervals; compare interval overlap. Unparseable dates
are unavailable rather than mismatched.

**Rationale**: Ancient dates and damaged inscriptions require conservative
normalization. Treating parse failure as conflict would create false
certainty. Preserving source text meets FR-006.

**Alternatives considered**:

- Rewrite the visible query into normalized text: rejected because collectors
  need to see exactly what they submitted.
- Remove all non-ASCII characters: rejected because it damages mixed-script
  legends.
- Infer uncertain date semantics aggressively: rejected because ranking is
  decision support, not attribution.

## 8. Cache implementation and identity

**Decision**: Use bounded in-memory TTL caches with independent search/detail
TTLs, same-key in-flight coalescing, and injected clock. Cache successful
searches including empty results and valid details; never cache failures. Hash
canonical keys with SHA-256. Check current configuration before reading cache.

**Rationale**: Cache data is explicitly disposable and does not need cross-
restart durability. In-memory storage avoids schema cleanup and keeps provider
payloads out of SQLite. Independent keys/TTLs match the distinct freshness
requirements. Configuration-before-cache prevents a removed key from being
masked by old data.

**Alternatives considered**:

- SQLite cache: rejected as unnecessary authoritative-looking persistence and
  migration/cleanup complexity.
- Browser cache: rejected because it cannot share quota savings across users
  and paths.
- Redis: rejected as a fourth deployable dependency for a self-hosted personal
  app.
- Cache errors/429: rejected because it could prolong outages or stale retry
  guidance.

## 9. Telemetry and quota visibility

**Decision**: Maintain a bounded in-memory rolling telemetry ring and expose
aggregates from an admin-only endpoint. Record only path, operation, status,
cache/refreshed flags, duration, counts, retry and observed retry-after, plus a
truncated non-reversible correlation digest. Do not claim remaining quota.

**Rationale**: This supplies recent operational health without introducing a
high-volume event table or retaining user text. It meets FR-025/026 while
respecting NFR-005/006 and Principle V.

**Alternatives considered**:

- Log full queries for debugging: rejected due to inscriptions/label privacy.
- Persist every event: rejected as disproportionate and creates retention
  obligations.
- Reuse collection-health summary: rejected because it models coin quality,
  not external-service operations.

## 10. Selected-reference data model

**Decision**: Add a one-to-one `QuickCaptureDraftReference` row with unique
draft ID, owner ID, fixed `Numista` catalog, number, canonical URI and
timestamps. Create/replace/remove it through Quick Capture repository
transactions. Copy it to `CoinReference` inside promotion.

**Rationale**: A normalized child row enforces “at most one” and avoids a
cluster of nullable draft columns. It maps exactly to the existing structured
reference contract and makes transactional exact-once copy straightforward.
No backfill is needed.

**Alternatives considered**:

- Store every candidate: rejected by FR-015.
- Store selection in notes/JSON: rejected because it bypasses validation and
  typing.
- Add Numista-specific columns directly to `Coin`: rejected because
  `CoinReference` is already authoritative.
- Infer the top candidate at promotion: rejected by explicit-selection
  requirements.

## 11. Compatibility

**Decision**: Add new typed POST lookup/enrichment contracts, retain the
existing GET search as a deprecated adapter preserving `{count,types}`, and
add photo/Quick Capture fields additively. New clients write only the new
selection contract. Remove no route or field in this feature.

**Rationale**: Backend-first rollout supports old and new SPA assets while
moving internal behavior to one service. Existing coins/drafts/references
remain readable, and rollback ignores the additive table.

**Alternatives considered**:

- Change GET response shape in place: rejected because cached/old SPA assets
  would fail.
- Flag-day removal: rejected as unnecessary deployment risk.
- Keep legacy duplicate implementation: rejected because it defeats the
  shared-client requirement.

## 12. Security and canonical links

**Decision**: Keep the key only in existing admin settings/server memory.
Bound query/evidence/IDs and response bodies. Generate canonical links from a
validated numeric ID (`https://en.numista.com/catalogue/pieces{id}.html`)
rather than trusting provider/user URLs. Protect lookup with normal auth and
health/configuration with admin authorization.

**Rationale**: Numeric reconstruction prevents arbitrary link persistence and
SSRF/open-redirect input. Safe application error mapping prevents raw internal
or credential leakage.

**Alternatives considered**:

- Accept arbitrary selected URLs from Vue: rejected as unsafe and unnecessary.
- Return invalid-key details to every collector: rejected because
  configuration specifics are administrator-only.

## 13. Testing and documentation

**Decision**: Use pure scorer/cache tests, fake interfaces and `httptest` for
Go, repository migration/transaction tests, Vue component/page tests, and
cross-workflow integration tests. Regenerate all Swagger/OpenAPI artifacts,
update shipped feature/API/deployment/Quick Capture docs, and add an ADR.

**Rationale**: Constitution Principles III, VIII, IX, §17 and §21 require
typed contracts, durable material decisions, and exact workflow regression
coverage. No test requires a real key or external network.

**Alternatives considered**:

- Manual provider testing only: rejected as flaky and not enforceable.
- Snapshot-only frontend tests: rejected because selection, keyboard,
  progressive state and retry behavior require interaction assertions.
