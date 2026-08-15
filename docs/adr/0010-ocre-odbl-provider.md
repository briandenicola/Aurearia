# ADR 0010: OCRE ODbL 1.0 Automated Coin-Type Provider (Nomisma SPARQL)

Date: 2026-08-15
Status: Accepted

## Context

Feature 344 shipped the Deep Analysis pipeline with OCRE (Online Coins of the
Roman Empire, American Numismatic Society) staged as a `not_automated` typed
stub behind validation gate **G-OCRE** (deferred task **T155**). The provider
node, the citation host allowlist entry (`numismatics.org`), the provider
catalog entry (`{Provider:"ocre", Automatable:false, Reason:
"pending_license_validation"}`), the admin setting
`SettingDeepIdentificationOCREEnabled` (default `false`), and the Go-owned
internal-tool/job-token/budget boundary all already exist and were designed to
be switched on with no contract respecification.

Feature 345 opens G-OCRE and implements OCRE as a first-class **automated**
Roman Imperial coin-type authority provider. This ADR records the two material
design choices that require a waiver/decision under Constitution §22: (1) a new
outbound third-party data integration and (2) a new data-license posture
(ODC-ODbL 1.0 with attribution to the American Numismatic Society), which is
distinct from every license already in the codebase.

This decision is deliberately scoped **narrower and different** from
[ADR 0009](0009-nomisma-authority-linking.md): ADR 0009 covers Nomisma's
**reconciliation** endpoint (`/apis/reconcile`, CC BY 4.0) for single-admin mint
linking. This ADR covers the Nomisma **SPARQL triplestore** (`/query`) for
automated coin-type candidate retrieval under **ODbL 1.0 / ANS**. ADR 0009 §
"Negative and trade-offs" explicitly warned that this Phase-2 work "cannot assume
this ADR's CC BY 4.0 attribution wording applies" — this ADR discharges that.

## Decision

### Automated route: Nomisma SPARQL over GET (validated live)

Automated OCRE access is a **fixed-template, parameter-bound Nomisma SPARQL
query** against `https://nomisma.org/query` (OCRE URI prefix
`http://numismatics.org/ocre/id/`). No OCRE front-end scraping, no OCRE-hosted
APIs (observed HTTP 500 in F344 research), and no arbitrary/user-supplied SPARQL.

The wire contract was verified live on 2026-08-15:

- **`GET https://nomisma.org/query?query=<url-encoded SPARQL>`** with headers
  `Accept: application/sparql-results+json` and a **non-default `User-Agent`**
  returns HTTP 200 with standard SPARQL 1.1 Query Results JSON.
- **`POST` is blocked with HTTP 403** at the Cloudflare edge, as is a GET with a
  default/empty `User-Agent`. Therefore the client uses **GET with a fixed
  non-default `User-Agent`** — the spec's "POST/GET per validated endpoint" is
  resolved to GET.

### Typed HTTP boundary

`OCREClient` (interface) / `HTTPOCREClient` (implementation) in
`src/api/services/ocre_client.go` is the **only** OCRE/Nomisma-triplestore HTTP
boundary, mirroring `nomisma_client.go`. Bounded 8 s timeout, 1 MiB response cap,
and every outcome mapped to a typed `OCREErrorKind` (`unavailable`, `timeout`,
`no_match`, `invalid_response`, `invalid_request`, `cancelled`) — never a raw
error. A caller/client deadline or any `net.Error.Timeout()` maps to `timeout`
(wire `timeout` → Python `timed_out`), keeping degradation typed end-to-end. Fixed
SPARQL templates (`ocre_query.go`) interpolate **only** application-validated
Nomisma id slugs inside `<...>` URI brackets (regex-validated, length-bounded);
free legend/inscription text never enters SPARQL, so injection cannot alter query
structure. Deterministic, LLM-free scoring/ranking (`ocre_scoring.go`) produces
stable candidate ordering and explanations.

### Internal-tool boundary, budget, cache

A new internal endpoint `POST /api/internal/tools/ocre_search` is added to the
existing job-token group; the Python provider node calls it and **never** talks
to Nomisma directly (Principle II). A per-job `ocre` call budget (setting
`DeepIdentificationOCRECallBudget`, default 3) is enforced by the shared budget
tracker; over-budget degrades to `quota_limited`, never an error. A bounded TTL
cache (`ocre_cache.go`, mirroring `nomisma_cache.go`) keyed on the fully-bound
parameter set caches `no_match` (negative caching) but never transient failures;
the key incorporates the enable-flag generation so a toggle never reuses stale
results. The endpoint never returns 5xx for an upstream problem.

### License, attribution, and share-alike posture

OCRE type data is **ODC-ODbL 1.0** with share-alike on derivative *databases* and
attribution to the **American Numismatic Society**. This feature persists only
**bounded per-job evidence** (a few type links/labels/matched fields per
analysis) in the existing `DeepIdentificationProviderRun`/proposal structures —
it **never** builds, caches, or stores an OCRE reference database, mirror, or
corpus, and never uses OCRE images. Under the authorized product interpretation,
the ODbL share-alike **database** obligation is therefore not triggered;
attribution and the license link are nonetheless rendered on every surface where
OCRE evidence appears (report, proposal, provider status, export) via a dedicated
`OCREAttribution.vue`, kept textually and visually **distinct** from Nomisma
(CC BY 4.0), Numista, and RPC. The exact attribution string is:

> Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic
> Society — ODbL 1.0.

with the canonical type link (`https://numismatics.org/ocre/id/...`) and license
link (`https://opendatacommons.org/licenses/odbl/1-0/`).

### Default-off, admin-gated, reversible

The existing `SettingDeepIdentificationOCREEnabled` stays default `false` and
gates whether the catalog entry is `Automatable`. Flag off ⇒ `not_automated`,
zero SPARQL calls, no attribution. No database migration is introduced.

## Alternatives Considered

- **POST SPARQL** (the conventional verb): rejected — blocked (403) at the
  Nomisma CDN; GET is the validated working path.
- **OCRE-hosted APIs / front-end scraping**: rejected by spec and unstable
  (HTTP 500 observed).
- **Reuse the ADR 0009 `nomisma_search` tool / CC BY attribution**: rejected —
  different endpoint, protocol (SPARQL vs reconcile), license (ODbL vs CC BY),
  allowlist semantics, and telemetry key; conflating them would misattribute the
  license.
- **LLM-ranked OCRE candidates**: rejected — violates deterministic-ranking
  success criterion (SC-005) and the no-LLM-citation-invention rule.
- **Persisting an OCRE reference database/corpus for reuse**: rejected by spec
  (FR-018/FR-021) and would trigger ODbL share-alike database obligations.
- **New DB table/migration for OCRE evidence**: rejected — the existing bounded
  provider-run/proposal structures suffice; additive-only.

## Consequences

### Positive

- Roman Imperial coins gain authoritative, citation-backed type candidates from
  the canonical ANS source, within the proven F344 pipeline and bounds.
- Injection-proof (slug-only URI binding), deterministic, and robust (typed
  partial outcomes; a single provider failure never fails a job).
- Reversible and safe: default-off, no migration, flag-off = zero calls.

### Negative and trade-offs

- Depends on Nomisma community infrastructure (no SLA); mitigated by default-off,
  bounded budget/timeout, and typed degradation.
- GET-only transport is a CDN constraint, not a protocol preference; if Nomisma
  later blocks the custom `User-Agent` the provider degrades to `unavailable`.
- Slug coverage (ruler/mint/denomination/material → Nomisma id) is
  application-owned and finite; unmapped inputs yield `no_match` rather than a
  guess (acceptable, honest).

## Security and Privacy

- No credentials; the Nomisma SPARQL endpoint is public.
- Requests are bounded by timeout and response size; only validated slugs are
  templated — adversarial input cannot alter query structure (SC-010).
- The internal tool is job-token authenticated; Python holds no DB handle, no
  API keys, no upstream URL.
- Telemetry records status/timing/counts/cache hit-miss only — never user notes
  or full legend text (FR-022).
- No raw upstream body or internal error is surfaced to the client — only typed
  status.

## Rollback

Set `DeepIdentificationOCREEnabled=false` for an instant disable (zero calls,
`not_automated`), or revert the additive code. No schema change to undo; the
in-memory OCRE cache is discarded safely on restart. Bounded per-job evidence
already persisted is inert and unaffected.

## Related

- [Feature 345 specification](../../specs/345-ocre-deep-analysis-provider/spec.md)
- [Feature 345 plan](../../specs/345-ocre-deep-analysis-provider/plan.md)
- [Feature 345 research (live SPARQL validation)](../../specs/345-ocre-deep-analysis-provider/research.md)
- [Feature 345 contract](../../specs/345-ocre-deep-analysis-provider/contracts/ocre-provider.md)
- [ADR 0009: Nomisma.org Authority Linking](0009-nomisma-authority-linking.md)
- [ADR 0007: Shared Numista Lookup Boundary](0007-shared-numista-lookup.md)
- [Feature 344 internal contract](../../specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md)
