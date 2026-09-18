# Research: Coin Copilot Specialist Market Tools

## R1. Capability names and canonical implementations

- **Decision**: Add exactly `market_search`, `auction_search`,
  `price_trends`, and `similar_lots`. Map them respectively to the existing
  `coin_search`, `auction_search`, `price_trends`, and `similar_lots` teams.
- **Rationale**: `market_search` is distinct from
  `search_my_collection`, while the mappings preserve the existing specialist
  implementations as the canonical domain workflows.
- **Alternatives considered**: Expose `coin_search` as the public tool name
  (too easily confused with owner collection search); add one generic
  `web_search` tool (overbroad and violates the exact four-capability scope).

## R2. Reuse boundary instead of a second implementation

- **Decision**: Extract typed callable runners from the existing four team
  modules. Both legacy supervisor nodes and Coin Copilot adapters call those
  runners. Provider search, fetch, parsing, trend analysis, and similarity
  analysis remain in the existing team/tool modules.
- **Rationale**: A narrow adapter can add strict schemas, cancellation hooks,
  and normalized outcomes without duplicating provider logic or replacing the
  legacy router.
- **Alternatives considered**: Copy team logic into
  `coin_copilot.py` (rejected as duplicate search logic); call legacy graph
  nodes and parse their final Markdown (rejected because prose is not a
  reliable typed contract).

## R3. Specialist execution and budgets

- **Decision**: A specialist invocation is one top-level Coin Copilot tool
  call. It consumes the existing run-wide iteration, tool-call, wall-clock,
  sequential-concurrency, and persisted-result budgets. Internal provider
  attempts are fixed by the canonical team, capped, share the same deadline,
  and do not create a second retry or token budget.
- **Rationale**: This preserves Feature 359 accounting while allowing a
  specialist to perform its established bounded search/fetch pipeline.
- **Alternatives considered**: Charge each internal HTTP request as a Copilot
  tool call (leaks provider implementation into the planner); give each
  specialist an independent budget (allows multiplication of work and violates
  FR-006).

## R4. Normalized result envelope

- **Decision**: All four capabilities return one strict envelope containing
  `schema_version`, `capability`, `outcome`, bounded `items`, bounded
  `provider_attempts`, `warnings`, and explicit truncation metadata. Price
  trends also include a typed `trend`; similar lots include match reasons and
  material differences.
- **Rationale**: A common envelope lets Python, Go, checkpoints, SSE, and Vue
  agree on failure and provenance semantics while discriminated item kinds
  preserve capability-specific fields.
- **Alternatives considered**: Persist Markdown (cannot validate provenance);
  four unrelated envelopes (more drift); one loose dictionary (fails strict
  typing).

## R5. Field-level provenance and source identity

- **Decision**: Every evidence item requires a validated canonical source URL,
  `observed_at`, confidence, verification state, provider id, and a bounded
  `provenance` list naming the fields supported by that source. Details absent
  from provider evidence remain null/empty and are omitted from field
  provenance. Canonical source identity is the normalized URL after removing
  fragments and normalizing host/default port; query parameters are preserved
  unless the provider boundary already defines safe tracking-parameter removal.
- **Rationale**: Item-level metadata alone does not prove which price, date,
  dealer, sale, or similarity attribute was observed. Field provenance makes
  unsupported enrichment detectable.
- **Alternatives considered**: Trust LLM-generated citations (not
  deterministic); use title/lot number as identity (collisions); rewrite URLs
  into guessed canonical forms (risks invention).

## R6. URL and provider-source policy

- **Decision**: Accept only `https` source URLs with a hostname, no user-info,
  no local/private/metadata target, and a hostname allowed by the capability's
  registered provider boundary. Preserve the provider URL except for safe
  canonical comparison. Redirects are revalidated at every hop by the existing
  outbound helper. A source failing validation cannot support a verified item.
- **Rationale**: This extends the existing outbound controls and prevents SSRF,
  credential-bearing links, and fabricated cross-provider attribution.
- **Alternatives considered**: Accept any model-returned URL (unsafe);
  maintain a second URL validator in Coin Copilot (drift); rewrite invalid
  links to provider home pages (fabricates provenance).

## R7. Provider and aggregate outcomes

- **Decision**: Provider attempts use `success`, `no_match`, `timeout`,
  `failure`, `unavailable`, or `malformed`. Aggregate outcome is:
  - `complete` when at least one valid item exists and no provider failed,
    timed out, was unavailable, or was malformed;
  - `partial` when at least one valid item exists and any attempted/eligible
    provider degraded;
  - `no_match` when all completed eligible providers report success/no-match
    and no valid item exists;
  - `unavailable` when no valid item exists and provider degradation prevents
    a reliable no-match conclusion.
- **Rationale**: The rule is deterministic and preserves valid evidence
  without disguising incomplete coverage.
- **Alternatives considered**: Fail the entire tool on one provider failure
  (throws away evidence); treat failure as no match (misleading); let the LLM
  choose the status (non-deterministic).

## R8. Deduplication and conflicts

- **Decision**: Deduplicate by canonical source identity. Merge only fields
  with identical values or strictly stronger verification. Preserve the
  strongest provenance, record conflicting observations as a warning, and do
  not count duplicates as independent samples.
- **Rationale**: The same lot may surface through multiple search paths.
  Deterministic deduplication prevents false corroboration and inflated trend
  samples.
- **Alternatives considered**: Deduplicate by title (false merges); keep all
  copies (inflates evidence); silently choose the newest value (hides conflict).

## R9. Price-trend sufficiency

- **Decision**: `rising`, `stable`, or `declining` requires at least three
  verified completed-sale observations in the same currency and price basis,
  across at least two sale dates spanning 30 days. Otherwise the trend state is
  `unknown`. Never convert currencies without a source-backed conversion
  already supplied by an approved provider; separate incomparable groups and
  explain the limitation.
- **Rationale**: This creates a testable minimum for a directional claim and
  avoids silently mixing hammer, premium-inclusive, estimate, bid, and listing
  prices.
- **Alternatives considered**: Use one or two observations (too fragile);
  perform implicit exchange-rate conversion (new provider/dependency and
  unsupported evidence); use active estimates as sales (semantically wrong).

## R10. Similar-lot ranking

- **Decision**: Reuse the existing similar-lots scorer, but require each
  normalized candidate to expose a bounded score, explicit
  `matched_attributes`, and `material_differences`. Sort by score descending
  with canonical source identity as a deterministic tie-breaker. Omit
  candidates without minimum identifying evidence or a valid URL.
- **Rationale**: Users can understand why a lot is similar, and deterministic
  ordering makes replay and tests stable.
- **Alternatives considered**: Hide the score/reasons (fails transparency);
  duplicate a new Go/Vue ranking algorithm (breaks the canonical team
  boundary).

## R11. Prompt injection and model synthesis

- **Decision**: Treat all provider text as untrusted data, sanitize
  instruction-shaped/token-shaped content before persistence, validate the
  normalized schema, and pass it back to the model only inside the existing
  explicit untrusted-data delimiter. Provider content cannot choose tools,
  modify the allowlist, request credentials, change owner/run scope, or bypass
  cancellation. Final-answer instructions require citations and forbid claims
  outside normalized fields.
- **Rationale**: External pages are adversarial input, not trusted prompts.
- **Alternatives considered**: Prompt-only defense without schema validation
  (insufficient); omit provider descriptions entirely (unnecessarily reduces
  useful evidence).

## R12. Cancellation, resume, and replay

- **Decision**: Check cancellation before specialist dispatch and after every
  awaited provider/model operation. Go remains authoritative and rejects late
  frames. The existing checkpoint stores the normalized completed result and
  call id; resumed execution hydrates that result and never reruns it solely
  because of disconnect/restart.
- **Rationale**: This directly preserves Feature 359's race and replay
  guarantees.
- **Alternatives considered**: Rely only on HTTP disconnect (late work can
  complete); cache in Python (violates statelessness); rerun for display
  (duplicates external work and evidence).

## R13. Public SSE and Vue representation

- **Decision**: Keep all ten existing event names. Add an optional,
  strictly-bounded `specialistResult` projection to `tool_completed` only for
  the four specialist tools. It contains normalized display evidence, never
  raw provider content or tool arguments. Vue renders it in the existing
  progress surface; final Markdown remains supplementary.
- **Rationale**: An additive field preserves old consumers while making source,
  confidence, verification, partial failure, and truncation visible and
  replayable without a new endpoint.
- **Alternatives considered**: New SSE event name (unnecessary compatibility
  surface); final Markdown only (weakly typed and hard to verify); new result
  endpoint/table (unnecessary durable surface).

## R14. Bounds

- **Decision**: Retain the existing 32 KiB per-tool persisted result and 64 KiB
  event limits. Before byte truncation, cap: query 500 characters, 10 evidence
  items, 10 provider attempts, 10 warnings, 20 provenance entries per item,
  2,048-character URLs, 300-character titles, 500-character descriptions or
  warning text, and 20 match/difference attributes. Deterministic list
  truncation occurs before the existing byte-bound fallback; metadata records
  original/persisted bytes, digest, and truncation.
- **Rationale**: Structural bounds improve useful degradation and keep the
  generic 32 KiB fallback as a final safety net.
- **Alternatives considered**: Only byte truncation (may discard all useful
  evidence); new configurable limits (unnecessary admin surface and config
  drift).

## R15. Go internal tool boundary

- **Decision**: Do not add Go internal HTTP endpoints for the specialist
  capabilities. The existing four callback routes remain collection-only.
  Specialist names are accepted in the execution manifest/checkpoint contract
  but dispatch only to Python's fixed in-process adapters, which themselves use
  the existing provider boundaries.
- **Rationale**: Go does not own these providers and must contain no agent
  logic. No route means an execution token cannot turn a specialist name into
  arbitrary Go-side network authority.
- **Alternatives considered**: Proxy provider HTTP through Go (moves AI/provider
  responsibility); add a generic fetch callback (violates least privilege).

## R16. OpenAPI and contract drift

- **Decision**: Add no public REST path. Document the additive
  `tool_completed.specialistResult` SSE projection in the API reference and
  feature contract. If Swagger models describe event payloads, update them
  additively; always run `task openapi` and the route-drift test so
  `src/api/docs/*` and `docs/openapi.json` remain generated from one source.
- **Rationale**: OpenAPI remains truthful without pretending internal Python
  tools are public endpoints.
- **Alternatives considered**: Publish internal tools in public OpenAPI
  (security/confusion); hand-edit generated files (guaranteed drift).

## R17. Observability and rollout

- **Decision**: Log/measure capability, provider id, safe outcome code,
  duration, evidence count, bytes, truncation, digest, run id, and execution id.
  Never log query text, result content, source URLs, prompts, credentials, or
  raw exceptions. Keep the existing default-off flag and fail-closed
  capability preflight; no additional feature flag is introduced.
- **Rationale**: Existing rollback remains sufficient and metrics can diagnose
  provider degradation without collecting user or market content.
- **Alternatives considered**: Per-specialist flags (scope/config expansion);
  raw provider logging (privacy and secret risk).
