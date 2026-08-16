# Implementation Plan: Optional Nomisma.org Authority Linking for Global Mint Locations

**Branch**: `343-nomisma-mint-authority-linking` | **Date**: 2026-08-14 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/343-nomisma-mint-authority-linking/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

**Scope note**: This plan covers **Phase 1 only** — Nomisma.org global-mint
authority linking, as specified. A **Phase 2 (Deferred)** section at the end
of this document records the OCRE/RPC catalog-authority extension seam for a
future spec; it is explicitly out of scope here and must not expand Phase 1's
design, schema, or client boundary.

Admins can optionally search Nomisma.org's live reconciliation service for a
candidate authority concept matching a **global** `MintLocation`, explicitly
confirm exactly one candidate, and persist a durable concept URI plus minimal
provenance (matched label, confirmation timestamp) on that record. Wherever a
linked mint's details are shown, the app displays a visible
"Source: Nomisma.org · CC BY 4.0" attribution linking to the specific concept
and the CC BY 4.0 license. There is no bulk ingestion, background sync, or
SPARQL browsing — every lookup is on-demand, admin-only, and scoped to global
mints; private (user-owned) mints are never searchable or transmitted.

Technical approach: add a narrow, typed `NomismaClient` HTTP boundary (mirrors
the existing `GeocodeService`/`NumistaClient` conventions — bounded timeout,
input validation, typed outcomes for unavailable/no-match/invalid-response,
no credential needed), a small bounded in-memory TTL cache for search
responses only (mirrors `numista_cache.go`, simplified — no singleflight
needed at this call volume), two new nullable+one string field on the
existing `MintLocation` GORM model (additive `AutoMigrate`, no destructive
migration), new admin-only handler routes for search/link/unlink layered
under the existing `MintLocationService`, and additive Vue admin UI (extends
`AdminCoinPropertiesSection.vue`'s global mint management) plus a shared
`NomismaAttribution` display used in Mint Map and any other surface that
already renders a linked global mint.

## Technical Context

**Language/Version**: Go 1.x (API, `src/api`), TypeScript 5 + Vue 3 (`src/web`)
**Primary Dependencies**: Gin, GORM (SQLite), stdlib `net/http` for the new
Nomisma client (no new third-party HTTP/reconciliation library — same
convention as `GeocodeService` and `NumistaClient`); Vue 3 Composition API,
Vitest, Axios (`src/web/src/api/client.ts`)
**Storage**: SQLite via GORM `AutoMigrate` — additive nullable columns on
existing `mint_locations` table; no new table (provenance is minimal enough
to fit inline, consistent with FR-003's "durable URI + matched label +
confirmation timestamp" scope)
**Testing**: `go test ./...` (table-driven service/handler/client tests with
`httptest.Server` fixtures, no live Nomisma calls), `vue-tsc --build`,
`npm run build`, Vitest component/unit tests (`src/web/src/**/__tests__/`)
**Target Platform**: Self-hosted Docker deployment (existing 3-service
architecture: API, web, agent) — this feature touches API and web only
**Project Type**: Web application (Go API + Vue SPA), per ADR 0002
three-service architecture
**Performance Goals**: On-demand search only; no throughput target beyond
existing admin-tooling expectations — bounded by Nomisma's own reconciliation
service response time
**Constraints**: Search request timeout bounded (mirrors
`geocodeRequestTimeout = 8s` convention); bounded result count per search;
cache is short-lived and search-only (never a substitute for the persisted
confirmed link, per FR-011 and the Assumptions section); admin-only exposure,
same-origin API only (browser never calls Nomisma directly)
**Scale/Scope**: Single-tenant personal collection app; expected admin usage
is occasional curation of a bounded global mint list (tens to low hundreds of
records), not a bulk workflow

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Rationale |
|------|--------|-----------|
| **§0 Hierarchy of Authority** | PASS | This plan defers to the spec's recorded clarification (no pre-validation gate; reconciliation assumed the intended mechanism) and to the PRD non-goal on reference-database replication (see below). No conflict found requiring escalation. |
| **Principle I — Clear Layered Architecture** | PASS | New work follows `Handler → Service → Repository → Database`. `NomismaClient` (HTTP boundary) is owned by a service, not the handler; `MintLocationHandler` stays a thin adapter; `MintLocationRepository`/`MintLocationService` gain no new persistence responsibilities beyond two new columns. |
| **Principle II — Service Boundary Separation** | PASS | Nomisma HTTP calls live behind a single `NomismaClient` interface, never called from handlers or repositories directly — mirrors `GeocodeService`/`NumistaClient` boundary ownership (ADR 0007). |
| **Principle III — Strict Types and Explicit Contracts** | PASS | New DTOs (`NomismaCandidate`, typed error kinds) are explicit; no `any`/`interface{}` leakage to the frontend; Swagger annotations required on new handler methods (§21.10). |
| **Principle IV — Simple Complete Changes** | PASS | Reuses existing `GeocodeService`-style client shape rather than porting the full Numista cache/telemetry/scoring stack, which would be disproportionate for a single-field, admin-only, on-demand lookup. Two nullable columns instead of a new join table, consistent with FR-003's minimal-provenance requirement. |
| **Principle V — Security, Auth, and Privacy by Default** | PASS | Search/link/unlink routes require admin auth (mirrors `/admin/mint-locations` middleware); private mints have no route path into this feature (FR-006); no credential to protect (Nomisma reconciliation is unauthenticated), so the risk surface is input validation and same-origin exposure, both addressed in research.md. |
| **Principle VI — Consistent User Experience** | PASS | Reuses existing admin mint-management panel and Mint Map/drawer surfaces rather than introducing a new page; attribution string and placement match the pattern already used for other external-source links. |
| **Principle VII/§17 Quality Gate** | PASS (planned) | `go vet`, `go test ./...`, `vue-tsc --build`, `npm run build` all required in Definition of Done; Swagger regeneration required for new admin routes. |
| **Principle VIII — Documented Decisions** | PASS (planned) | An ADR will be added under `docs/adr/` for the Nomisma client boundary/cache decision if the implementing PR makes a material design choice, per §21.12 — deferred to implementation, not blocking planning. |
| **Principle IX — Automated Enforcement** | PASS | Architecture test coverage already enforces layering (`architecture_test.go`); no new exemption needed. |
| **PRD §4 Non-Goal ("do not replicate numismatic reference catalogs")** | RECONCILED | This feature stores only a reference (URI + matched label + timestamp) to a single external concept per confirmed link — never a copy of Nomisma's dataset, partner corpora, or a browsable catalog. It is the same "link to them, don't replicate them" pattern already used for Numista/ACSearch elsewhere in the product. See research.md for the explicit reconciliation. |
| **Constitution §17 workflow-contract check** | PASS (planned) | Touches a shared surface (`MintLocation`, Mint Map, admin mint panel); plan requires regression tests proving existing coordinates/aliases/ownership/legacy-backfill behavior (338) is unchanged by this additive feature. |
| **Constitution §21 Definition of Done** | PASS (planned) | Enumerated fully in quickstart.md's rollout checklist. |

No violations requiring the Complexity Tracking table.

## Project Structure

### Documentation (this feature)

```text
specs/343-nomisma-mint-authority-linking/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md         # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── nomisma-authority-linking.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
src/api/
├── models/
│   └── mint_location.go            # extend: NomismaURI, NomismaLabel, NomismaLinkedAt (nullable)
├── services/
│   ├── nomisma_client.go           # NEW: typed reconciliation HTTP boundary
│   ├── nomisma_client_test.go      # NEW: httptest fixtures, no live Nomisma
│   ├── nomisma_cache.go            # NEW: bounded in-memory TTL cache for search responses
│   ├── nomisma_cache_test.go       # NEW
│   ├── mint_location_service.go    # extend: LinkNomisma / UnlinkNomisma (admin-only, global-only)
│   └── mint_location_service_test.go
├── handlers/
│   ├── mint_location.go            # extend: SearchNomisma / LinkNomisma / UnlinkNomisma admin routes
│   └── mint_location_handler_test.go
├── repository/
│   └── mint_location_repository.go # no interface change; existing FindByID/Update reused
├── database/
│   └── database.go                 # AutoMigrate already includes MintLocation{} — additive columns only
└── main.go                         # DI wiring: NewNomismaClient, NewNomismaCache/LookupService, route registration

src/web/src/
├── api/client.ts                   # extend: searchNomismaMintCandidates, linkNomismaMintLocation, unlinkNomismaMintLocation
├── types/index.ts                  # extend: MintLocation gains nomismaUri/nomismaLabel/nomismaLinkedAt; NomismaCandidate type
├── components/
│   ├── admin/AdminCoinPropertiesSection.vue   # extend: per-global-mint Nomisma search/confirm/unlink controls
│   ├── admin/__tests__/AdminCoinPropertiesSection.test.ts
│   ├── mint/NomismaAttribution.vue             # NEW: shared "Source: Nomisma.org · CC BY 4.0" display
│   ├── mint/__tests__/NomismaAttribution.test.ts
│   └── map/MintCoinDrawer.vue                  # extend: render NomismaAttribution when group.mint is linked
└── utils/mintMap.ts                # extend: MintReference type carries the new optional fields through unchanged
```

**Structure Decision**: Existing web application structure (Go API +
Vue SPA under `src/api` / `src/web`, per ADR 0002's three-service
architecture). This feature is additive within both existing trees; no new
top-level project, service, or directory is introduced. All new backend
files live beside their existing `mint_location_*` and `numista_*`/`geocode_*`
siblings so ownership and test conventions stay consistent.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No entries — the Constitution Check above found no violations requiring
justification.

## Phase 2 (Deferred, Out of Scope): OCRE/RPC Catalog-Authority Extension

**Status: Explicitly deferred. Not designed, not scheduled, not part of this
plan's Phase 1 implementation. Included here only to record the extension
seam so a future spec doesn't have to rediscover it.**

Per additional user direction, this plan intentionally does **not** expand
Phase 1 (Nomisma global-mint linking) to also cover
[OCRE](https://numismatics.org/ocre/apis) (Online Coins of the Roman Empire,
American Numismatic Society) or
[RPC Online](https://rpc.ashmus.ox.ac.uk/introduction) (Roman Provincial
Coinage, Ashmolean Museum / University of Oxford). These are **not**
interchangeable with Nomisma and MUST NOT be folded into the
`NomismaClient`/`MintLocation` design built in Phase 1:

| | Nomisma (Phase 1, this plan) | OCRE / RPC (Phase 2, deferred) |
|---|---|---|
| **What it identifies** | A controlled-vocabulary *concept* (a mint, a person, a material — here, specifically a mint) | A specific *coin type/catalog entry* (an OCRE `ric.*` ID, an RPC `rpc.*` ID) |
| **Aurearia entity it would attach to** | `MintLocation` (global, admin-curated place) | `CoinReference` (per-coin catalog citation — see `214-structured-numismatic-catalog-references`) |
| **API contract** | OpenRefine-compatible reconciliation service, unauthenticated, JSON query/result envelope (see research.md §1) | OCRE exposes its own API surface (`numismatics.org/ocre/apis`) separate from Nomisma's reconciliation protocol; RPC Online (`rpc.ashmus.ox.ac.uk`) has its own separate API/data model. Neither has been API-contract-reviewed for this repo yet. |
| **Licensing/attribution** | CC BY 4.0, confirmed and already encoded in Phase 1's attribution string | **Unconfirmed for this repo.** OCRE (ANS) and RPC Online (Ashmolean/Oxford) are separate institutions with their own terms; nothing in this plan assumes they share Nomisma's CC BY 4.0 license, and Phase 2 MUST NOT reuse the `NomismaAttribution.vue` string/wording without its own license research and (if different) its own attribution component. |
| **Existing Aurearia behavior today** | None prior to this feature | **Already present, but shallow**: `src/agent/app/tools/numismatic_authority.py`'s `lookup_authority_uri()` does *best-effort URL templating* only — it builds a plausible `https://numismatics.org/ocre/id/{ric-number}` or `https://rpc.ashmus.ox.ac.uk/id/{rpc-number}` string (or a `results?q=`/`search?q=` fallback search link) from a catalog/volume/number the AI agent already parsed. It never calls an OCRE/RPC API, never validates the link resolves, never reconciles a free-text description against a candidate list, and carries no confirm/attribution workflow. This is a Python-agent-side (`src/agent/`) convenience helper for `CoinReference.URI`, not a Go-API authority-linking feature, and is unaffected by this plan. |

**Extension seam for a future Phase 2** (informational only — not designed
here):
- A future spec would need its own `/speckit.specify` → `/speckit.clarify`
  → `/speckit.plan` cycle, not an addendum to this one, because it targets a
  different entity (`CoinReference`, not `MintLocation`) and a materially
  different licensing/API-review question.
- Required pre-work before any implementation: (1) read and cite
  `numismatics.org/ocre/apis`'s actual documented contract (query shape,
  rate limits, auth if any); (2) read and cite RPC Online's actual API/
  license terms at `rpc.ashmus.ox.ac.uk`; (3) confirm each dataset's license
  independently — do not assume CC BY 4.0 carries over from Nomisma; (4)
  decide whether OCRE/RPC linking upgrades the existing
  `lookup_authority_uri()` heuristic (agent-side, Python) or introduces a
  new Go-side typed client analogous to (but not shared with)
  `NomismaClient` — this is a design decision for that future plan, not
  this one.
- If Phase 2 is ever scheduled, it MUST reconcile against the same PRD §4
  non-goal this plan already reconciles against (Principle IV,
  `docs/prd.md` §4) — OCRE/RPC are exactly the kind of "numismatic reference
  catalog" the PRD says the app links to but does not replicate, so the same
  "reference only, no ingestion" discipline applies there too.
- This plan's `NomismaClient`/`nomisma_cache.go`/`MintLocation` schema
  additions are deliberately **not** made generic or provider-pluggable in
  anticipation of Phase 2 — Constitution Principle IV disfavors speculative
  abstraction for a use case that hasn't been specified yet.
