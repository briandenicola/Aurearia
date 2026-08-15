# Implementation Plan: OCRE Automated Deep Analysis Provider

**Branch**: `345-ocre-deep-analysis-provider` | **Date**: 2026-08-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/345-ocre-deep-analysis-provider/spec.md`

## Summary

Open validation gate **G-OCRE** (deferred task **T155**) and turn the existing
Feature 344 `not_automated` OCRE stub into a first-class **automated Roman
Imperial coin-type authority provider**, entirely within the already-shipped
Deep Analysis architecture. The single new outbound integration is a **fixed-
template, parameter-bound Nomisma SPARQL query** against `https://nomisma.org/query`
(OCRE URI prefix `http://numismatics.org/ocre/id/`), routed through the existing
Go-owned internal-tool / job-token / call-budget boundary. The Python agent node
stops emitting the trivial stub and instead calls a new internal tool
`ocre_search`; it never talks to Nomisma directly. Results become application-
owned typed candidate claims (canonical `numismatics.org` citation, RIC-style
label, matched fields, deterministic confidence, explanation) merged through the
existing `ProviderEvidence`/`DeepSynthesis` contracts — **with no change to the
generic provider contract**. OCRE ODbL 1.0 / American Numismatic Society
attribution renders, distinct from Nomisma/Numista, on every surface. Everything
is gated by the existing default-`false` admin setting
`SettingDeepIdentificationOCREEnabled`; flag-off means zero SPARQL calls and a
`not_automated` outcome. A new **ADR 0010** records the G-OCRE / ODbL posture.

**Research outcome (blocking question resolved):** the Nomisma SPARQL contract
was validated live during planning (see [research.md](./research.md)). `GET
https://nomisma.org/query?query=<url-encoded>` with a non-default `User-Agent`
and `Accept: application/sparql-results+json` returns HTTP 200 with standard
SPARQL 1.1 JSON. **POST is blocked (HTTP 403, Cloudflare)** — so the client uses
**GET**. Implementation is **not blocked**.

## Technical Context

**Language/Version**: Go 1.26.x (API), Python 3.12 / FastAPI + LangGraph (agent), Vue 3 + TypeScript (SPA)
**Primary Dependencies**: Gin, GORM (Go); httpx, Pydantic, LangGraph (Python); Vite, vue-tsc (web). No new third-party dependency — the OCRE client is stdlib `net/http` + `encoding/json`, mirroring `services/nomisma_client.go`.
**Storage**: Existing SQLite/Postgres via GORM. No new table; bounded OCRE evidence persists in the existing `DeepIdentificationProviderRun.ClaimsJSON` and the job proposal document JSON. No schema migration required (see Constitution Check + data-model).
**Testing**: `go test ./...` (httptest fake Nomisma), `pytest` (fake internal tool), `vue-tsc --build` + vitest; one **manual, CI-excluded** live smoke test against `https://nomisma.org/query`.
**Target Platform**: Self-hosted three-service deployment (Go API, Python agent, Vue SPA) behind Docker Compose.
**Project Type**: Web application — three services (Constitution §II). Structure Decision below.
**Performance Goals**: Bounded per-job OCRE work: ≤ per-job call budget (default 3), request timeout (default 8 s, matching Nomisma client), response-size cap (1 MiB), result cap (default 5 distinct types). No throughput target — single-owner, admin-triggered.
**Constraints**: Never exceed provider bounds; deterministic ranking; injection-proof binding; ODbL attribution on every surface; no OCRE images; no corpus/DB; Python holds no DB handle / no direct upstream HTTP (Principle II).
**Scale/Scope**: One provider node + one internal tool + one Go SPARQL client + one attribution component; ~5 backend files, ~3 agent files, ~2 web surfaces changed. Estimated **S–M**.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle / Section | Assessment | Verdict |
|---|---|---|
| **I. Layered architecture** | New `OCREClient` interface + `HTTPOCREClient` in `services/`; handler stays thin (`DeepProviderToolsHandler.OCRESearch`); no SQL in handler; DI via constructor. | ✅ Pass |
| **II. Service boundary** | Python agent calls only the Go internal tool `ocre_search`; **no** direct Python HTTP to Nomisma (`app/outbound.py` unused for this path). Go contains no LLM logic. Scoring is deterministic Go code, not LLM. | ✅ Pass |
| **III. Strict types & contracts** | Typed `OCREErrorKind`; Pydantic tool response model; Swagger annotations on the new internal handler; Vue nullable-safe rendering. Generic `ProviderEvidence`/`ProviderClaim` unchanged. | ✅ Pass |
| **IV. Simple complete proportional** | Reuses the Nomisma-client / Nomisma-cache / budget-tracker patterns verbatim; adds one closed provider path. No speculative abstraction. | ✅ Pass |
| **VIII/IX (schemas, arch tests)** | Additive-only, no new model; `architecture_test.go` layering respected. | ✅ Pass |
| **§17 Quality Gate** | Workflow-contract check: touched shared surface = Deep Analysis provider fan-out + citation allowlist + settings; targeted regressions listed in data-model/quickstart. | ✅ Pass |
| **§21 Definition of Done** | ADR added (0010); regression tests per exact path (injection/timeout/malformed/cap/cache/flag/routing/attribution); no secrets; Swagger; new service method unit-tested. | ✅ Pass |
| **§22 Amendment / ADR** | New third-party data-license posture (ODbL 1.0 / ANS) + gate opening ⇒ **ADR required**. ADR 0010 (Proposed) authored alongside this plan. | ✅ Pass |

**No violations. Complexity Tracking table intentionally empty.**

Distinctness from ADR 0009 (Nomisma reconcile) is explicit: ADR 0009 covers the
CC BY 4.0 **reconciliation** endpoint (`/apis/reconcile`, GET, JSON candidate
map) for admin mint linking; ADR 0010 covers the **SPARQL triplestore**
(`/query`, GET, SPARQL-results JSON) under **ODbL 1.0 / ANS** for automated
coin-type candidates. Different endpoint, protocol, license, and consumer.

## Project Structure

### Documentation (this feature)

```text
specs/345-ocre-deep-analysis-provider/
├── plan.md              # This file
├── research.md          # Phase 0 — Nomisma SPARQL contract (validated live), scoring, injection posture
├── data-model.md        # Phase 1 — entities, persistence reuse, status mapping (no migration)
├── quickstart.md        # Phase 1 — enable/verify + test matrix + manual smoke test
├── contracts/
│   └── ocre-provider.md  # Phase 1 — ocre_search internal tool + SPARQL template + candidate schema
└── checklists/
    └── requirements.md   # Pre-existing spec checklist
```

Related ADR (outside specs tree): `docs/adr/0010-ocre-odbl-provider.md` (Proposed).

### Source Code (repository root)

```text
src/api/                                         # Go API
├── services/
│   ├── ocre_client.go            # NEW  OCREClient / HTTPOCREClient — fixed SPARQL over GET
│   ├── ocre_client_test.go       # NEW  httptest fake Nomisma: injection/timeout/malformed/cap
│   ├── ocre_query.go             # NEW  fixed template + slug-binding + escaping + result parse
│   ├── ocre_query_test.go        # NEW  SC-010 structural-invariance (injection) fixtures
│   ├── ocre_cache.go             # NEW  bounded TTL cache (mirrors nomisma_cache.go), negative-cache
│   ├── ocre_scoring.go           # NEW  deterministic candidate scoring/ranking/explanations
│   ├── ocre_scoring_test.go      # NEW  determinism + tie-break + ambiguity + no-match
│   └── deep_identification_pipeline_runner.go   # EDIT catalog entry OCRE conditional on flag
├── handlers/
│   ├── internal_tools.go         # EDIT add OCRESearch handler + request/response DTOs + Swagger
│   └── internal_tools_test.go    # EDIT budget/flag/allowlist handler tests
├── models/
│   └── deep_identification_provider_run.go      # (no change — reuse status enum + ClaimsJSON)
└── main.go                        # EDIT register POST /api/internal/tools/ocre_search; wire OCRE client+cache

src/agent/app/                                    # Python agent
├── teams/deep_identification/
│   ├── providers/ocre.py         # EDIT stub → automated node (calls tools.ocre_search, builds claims)
│   ├── graph.py                  # EDIT move "ocre" from _TRIVIAL to automated fan-out when automatable
│   └── router.py                 # (no change — automatable OCRE now flows through normal selection)
├── tools/provider_tools.py       # EDIT add async ocre_search(...) wrapper
└── tests/
    ├── test_deep_identification_ocre.py         # NEW node: contributed/no_match/timeout/malformed/flag
    └── test_deep_identification_router.py       # EDIT OCRE selection for Roman-Imperial evidence + override

src/web/src/                                      # Vue SPA
├── components/deep-identification/
│   ├── OCREAttribution.vue       # NEW  fixed ODbL 1.0 / ANS attribution + SafeExternalLink (distinct)
│   ├── DeepReportPanel.vue       # EDIT render OCRE claims + OCREAttribution when ocre evidence present
│   └── DeepProposalEditor.vue    # EDIT surface OCRE attribution in the draft proposal
└── (existing admin settings surface)  # EDIT expose OCRE enable toggle + health/outcome-class
```

**Structure Decision**: Web application / three-service layout (Constitution §II).
The change is deliberately confined to the OCRE provider seam in each service —
the Go internal-tool boundary, the Python provider node, and the Vue attribution
surface — reusing every existing Deep Analysis mechanism (fan-out, budget,
citation allowlist, settings, SSE privacy). No new service, package, or table.

## Complexity Tracking

> No Constitution violations. Table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| — | — | — |

## Phased Delivery (S–M, ready for `/speckit.tasks`)

Sequenced for **beta**; each phase is independently reviewable and leaves the
build green with the flag **off** (no behavior change until enabled).

- **Phase A — Go SPARQL boundary (core, no UI):** `ocre_query.go` (template +
  slug-binding + parse), `ocre_client.go` (GET, timeout, size cap, typed
  `OCREErrorKind`), `ocre_cache.go` (TTL + negative cache), `ocre_scoring.go`
  (deterministic rank/explanations). Full offline-fixture unit tests incl.
  SC-010 injection invariance. **No routing yet.**
- **Phase B — Internal tool + wiring:** `OCRESearch` handler (job-token, budget,
  allowlist re-check, never-5xx), register route in `main.go`, make the pipeline
  catalog entry OCRE `Automatable` conditional on `settings.OCREEnabled`. Handler
  tests (budget/flag/allowlist).
- **Phase C — Python node + routing:** `provider_tools.ocre_search`, rewrite
  `providers/ocre.py` to build typed claims from tool output, move OCRE into the
  automated fan-out set in `graph.py`, ensure router selects OCRE for Roman-
  Imperial evidence and honors explicit override. Node + router tests.
- **Phase D — Attribution & admin UX:** `OCREAttribution.vue` (distinct ODbL/ANS),
  wire into report + proposal + provider-status + export; admin enable toggle and
  health/outcome-class view using existing settings patterns. Web tests.
- **Phase E — ADR, docs, gates, release:** finalize ADR 0010 (Proposed→Accepted on
  merge), update `docs/adr/README.md` index + feature docs, run full §17/§21
  gates, manual live smoke test (excluded from CI), beta release note.

**Effort:** S–M (~5 Go files, ~3 agent files, ~2 web surfaces, 1 ADR).
**Risk:** Low–Medium. Primary risks: (1) Nomisma community infra has no SLA →
mitigated by typed partial outcomes + default-off; (2) SPARQL slug mapping
coverage (ruler/mint/denomination/material → Nomisma id) → mitigated by
application-owned closed slug maps + graceful `no_match`; (3) ODbL share-alike
interpretation → bounded per-job evidence only (no DB), always-on attribution,
recorded in ADR 0010 (remaining non-blocking legal note below).

## Constitution Re-Check (post-design)

Re-evaluated after Phase 1 artifacts: no new violation introduced. Generic
provider contract untouched (ODbL specifics carried in a fixed attribution
string + UI-side license link keyed on `provider == "ocre"`, not by widening
`ProviderEvidence`). Additive, reversible (flag-off), ADR-backed. ✅ Pass.
