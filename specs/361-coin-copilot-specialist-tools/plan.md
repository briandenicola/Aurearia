# Implementation Plan: Coin Copilot Specialist Market Tools

**Branch**: `beta` (existing branch; no branch creation or switch) | **Date**: 2026-09-18 | **Spec**: `specs/361-coin-copilot-specialist-tools/spec.md`
**Input**: Feature 361 / issue #723, extending the Feature 359 durable read-only harness

## Summary

Extend Coin Copilot with exactly four read-only model-callable capabilities:
`market_search`, `auction_search`, `price_trends`, and `similar_lots`. Each
capability adapts the existing canonical Python specialist team and its
existing outbound-provider boundary into strict Pydantic input/output models.
The Coin Copilot loop remains bounded; independent read-only calls execute in
deterministic groups of at most three by default, while a specialist invocation
consumes one existing run-wide tool call and receives no independent budget.

Go remains the durable authority. It expands the execution allowlist, validates
the normalized specialist result again, stores it in the existing bounded
checkpoint representation, and emits an additive public projection on the
existing `tool_completed` event. Vue renders outcome, warnings, sources,
confidence, and verification state in the existing drawer. No table, public
route, run state, retention rule, or legacy-chat contract changes.

## Technical Context

**Language/Version**: Go 1.26.6, Python 3.12, Vue 3.5 / TypeScript 5.9
**Primary Dependencies**: Gin, GORM, SQLite, FastAPI, Pydantic, LangGraph/LangChain, httpx, Vue 3
**Storage**: Existing Go-owned SQLite Coin Copilot tables; no new table or column; Python remains stateless
**Testing**: `go test ./...`, `go test -race ./...`, `go vet ./...`, `ruff check app/ tests/`, `pytest tests/ -v`, `npm run type-check`, `npm run test`, `npm run build`, `task openapi` drift check
**Target Platform**: Existing two-container self-hosted web/PWA deployment
**Project Type**: Go API + Vue SPA + stateless Python agent service
**Performance Goals**: Every specialist call settles inside the existing 120-second default/150-second maximum execution budget; cancellation is checked before the specialist call and after each awaited provider operation; replay performs zero provider calls; UI remains responsive while rendering at most 10 evidence items per specialist result
**Constraints**: Exactly four new read-only capabilities; at most 3 concurrent tool calls by default and 5 for accepted snapshots; 12 run-wide tool calls by default; 32 KiB persisted result per call by default; 64 KiB public event maximum; strict source validation; no arbitrary HTTP; no provider-native/raw payload persistence; no chain-of-thought; owner/run/execution binding; feature flag and legacy fallback preserved
**Scale/Scope**: Personal-scale single-node deployment; one active run per owner by default; existing queue depth 16; at most 10 normalized evidence items and 10 warnings per specialist result

There are no unresolved technical-context items. Phase 0 decisions are recorded
in `research.md`.

## Constitution Check

### Pre-design gate

| Gate | Status | Design evidence |
|---|---|---|
| Principle I — layered architecture | PASS | No public handler or repository change is required. Go result validation remains in `services`; any thin projection logic is called by the existing worker/service. |
| Principle II — service boundaries | PASS | Python performs inference/provider access but remains DB-free; Go owns durable checkpoints/events/auth/cancellation; Vue calls only Go. |
| Principle III — strict types/contracts | PASS | Pydantic, Go DTO validators, TypeScript discriminated unions, shared JSON fixtures, SSE documentation, and OpenAPI drift checks are explicit deliverables. |
| Principle IV — simple complete changes | PASS | The design adapts the four existing specialist teams/provider functions and extends the existing Coin Copilot tool client instead of adding a second router or duplicate searches. |
| Principle V — security/privacy | PASS | Capability-specific source policies, safe outbound access, prompt-injection-as-data handling, credential binding, owner scoping, redaction, and bounded results fail closed. |
| Principle VI — consistent UX | PASS | The existing responsive drawer and design tokens render source evidence; no new app surface or approval UI is introduced. |
| Principle VII — CI/release integrity | PASS | No dependency is required; all Go/Python/Vue gates, race tests, generated Swagger, and root OpenAPI synchronization are planned. |
| Principle VIII — documented decisions | PASS | This plan and its research/contracts record the extension. ADR 0016 remains authoritative because durable ownership and service boundaries do not change. |
| Principle IX / §17 | PASS | Contract, malformed-provider, injection, cancellation, replay, owner-isolation, fallback, and exact UI-path tests are required. |
| §21 Definition of Done | PASS | The plan includes documentation, API drift, architecture, security, full build, lint, unit, integration, and workflow regression gates. |

No constitution violation or waiver is required.

### Post-design gate

The Phase 1 design preserves the same boundaries and passes all gates:

- no new persistence object or Python database access;
- no new public HTTP route or event name;
- one additive optional `specialistResult` projection on `tool_completed`;
- strict validation in Python and Go before persistence/publication;
- network access only through the canonical specialist/provider boundaries;
- existing feature flag, model preflight, owner checks, execution credentials,
  cancellation, replay, retention, and legacy fallback remain unchanged.

## Project Structure

### Documentation for this feature

```text
specs/361-coin-copilot-specialist-tools/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/
│   └── requirements.md
└── contracts/
    ├── specialist-tools.md
    ├── public-sse-extension.md
    └── openapi-impact.md
```

### Planned source changes

```text
src/agent/
├── app/
│   ├── models/
│   │   ├── requests.py
│   │   └── responses.py
│   ├── teams/
│   │   ├── coin_copilot.py
│   │   ├── coin_search.py
│   │   ├── auction_search.py
│   │   ├── price_trends.py
│   │   ├── similar_lots.py
│   │   └── specialist_contracts.py       # typed adapters/normalization only
│   └── tools/
│       ├── copilot_collection_tools.py   # generalized bounded dispatcher
│       ├── search.py                     # existing dealer/provider boundary
│       └── numisbids.py                  # existing auction-provider boundary
└── tests/
    ├── fixtures/coin_copilot/
    ├── test_coin_copilot_contract.py
    ├── test_coin_copilot_harness.py
    ├── test_coin_copilot_security.py
    ├── test_coin_copilot_architecture.py
    └── test_coin_copilot_specialists.py

src/api/
├── services/
│   ├── coin_copilot_contract.go
│   ├── coin_copilot_worker.go
│   └── *_test.go
├── handlers/
│   └── coin_copilot_internal_tools_test.go
├── integration/
│   └── coin_copilot_seam_test.go
└── docs/
    ├── docs.go
    ├── swagger.json
    └── swagger.yaml

src/web/src/
├── types/agent.ts
├── api/endpoints/agent.ts
├── composables/
│   ├── useCoinCopilot.ts
│   └── __tests__/useCoinCopilot.test.ts
└── components/
    ├── chat/CopilotRunProgress.vue
    └── __tests__/CoinSearchChat.copilot.test.ts

docs/
├── features/coin-copilot.md
├── api-reference.md
├── testing.md
└── openapi.json
```

**Structure Decision**: Keep provider-specific search/fetch behavior in the
four existing teams and `app/tools`. Add one typed adapter/normalization module
for use by Coin Copilot and, where practical, have legacy team formatting call
the same normalized runner. Do not add Go provider endpoints, another Python
router, another persistence model, or duplicate web-search code.

## Architecture Flow

```text
Vue existing CoinSearchChat drawer
  └─ existing public Coin Copilot REST/SSE
       └─ Go CoinCopilotService / worker
            ├─ mints existing owner+run+execution+tool credential
            ├─ sends 10-tool allowlist to stateless Python
            ├─ validates normalized specialist results
            ├─ persists bounded result in existing checkpoint
            └─ publishes existing tool_completed event with optional
               specialistResult public projection
                    │
                    v
Python bounded Coin Copilot loop (deterministic groups, 3 concurrent by default)
  ├─ collection tools → existing Go internal callback routes
  └─ exactly four specialist adapters
       ├─ market_search  → existing coin_search team/provider functions
       ├─ auction_search → existing auction_search team/NumisBids functions
       ├─ price_trends   → existing price_trends search/analysis functions
       └─ similar_lots   → existing similar_lots search/scoring functions
             └─ existing safe outbound/provider boundaries only
```

## Phase 0 — Research

`research.md` resolves:

1. capability names and their mapping to existing specialist teams;
2. reuse/refactor boundary that prevents duplicate search logic;
3. strict normalized result and field-provenance shape;
4. aggregate outcome and partial-failure rules;
5. URL/source policy and prompt-injection controls;
6. budget, result-count, timeout, cancellation, and replay semantics;
7. public SSE/Vue projection and OpenAPI compatibility;
8. observability, rollout, and test strategy.

## Phase 1 — Design and Contracts

### Data design

- Reuse `CoinCopilotCheckpoint.completed_tools[].result`; do not migrate
  SQLite.
- Add strict discriminated Python and Go DTOs for specialist inputs, provider
  attempts, evidence items, price-trend summary, provenance, warnings, and
  truncation metadata.
- Store only normalized evidence and safe provider outcomes. Raw pages, raw
  search responses, prompts, credentials, and provider exceptions are not
  persisted.
- Derive a bounded public `specialistResult` projection from the already
  validated result. Existing non-specialist events remain byte-compatible.

### Contract design

- `contracts/specialist-tools.md`: Python tool schemas, normalization rules,
  provider outcome aggregation, URL validation, cancellation, and failure
  mapping.
- `contracts/public-sse-extension.md`: additive `tool_completed` payload and
  replay/UI rules.
- `contracts/openapi-impact.md`: no new route; generated Swagger/root OpenAPI
  synchronization and SSE documentation obligations.

### Agent context

Run:

```powershell
& .specify/scripts/powershell/update-agent-context.ps1 -AgentType copilot
```

The script may update the standard Copilot context file, preserving manual
content between its markers.

## Phase 2 — Implementation Plan

Planning stops after this phase; no source implementation is performed.

### 2.1 Lock shared contracts and fixtures

1. Add canonical valid fixtures for all four tool inputs/results and the
   additive public event projection.
2. Add invalid fixtures for extra fields, unsafe/credential-bearing URLs,
   missing provenance, unsupported outcome states, duplicate source identity,
   oversized arrays/text, and hidden-reasoning fields.
3. Make Python and Go deserialize the same fixtures before changing runtime
   behavior.

### 2.2 Adapt the existing specialist teams

1. Extract callable typed runners from `coin_search.py`,
   `auction_search.py`, `price_trends.py`, and `similar_lots.py`.
2. Keep each team's existing search/fetch/analysis functions as the canonical
   implementation; the adapter supplies cancellation/deadline hooks and
   converts outputs to the shared result envelope.
3. Keep legacy supervisor nodes calling those same canonical runners and
   preserve their legacy user-facing response behavior.
4. Route all network work through existing `search.py`, `numisbids.py`, and
   configured model-search boundaries; reject URLs outside the registered
   source policy.

### 2.3 Extend the stateless Python harness

1. Expand the fixed allowlist from six to ten tools with only:
   `market_search`, `auction_search`, `price_trends`, and `similar_lots`.
2. Add strict input/result Pydantic models and specialist runner dispatch.
3. Count each specialist invocation once against the existing run-wide tool
   budget; apply the same result byte bound and digest logic.
4. Check cancellation before dispatch and after every awaited provider
   operation; do not emit completion/checkpoint frames after cancellation.
5. Feed only normalized data to the model under the existing explicit
   untrusted-data delimiter. Update the system prompt to require source URL,
   observation time, confidence, verification state, outcome, and limitations.
6. Hydrate completed specialist results from checkpoints so reconnect/resume
   does not repeat provider work.

### 2.4 Extend Go validation without changing ownership

1. Expand `CoinCopilotAllowedTools`; leave
   `copilotCallbackTools` and the registered internal callback routes unchanged.
2. Continue minting the existing fresh execution credential with the expanded
   fixed allowlist; specialist names authorize Python-local dispatch only and
   have no Go HTTP route.
3. Add strict Go validation for each normalized specialist result before
   accepting `tool_completed` or checkpoint state.
4. Canonicalize/deduplicate source identities, apply existing sanitization and
   32 KiB bound, and reject mismatched tool/result kinds.
5. Build the bounded additive `specialistResult` public projection and persist
   it through the existing append-before-publish transaction path.
6. Preserve existing cancellation/status predicates so late specialist frames
   lose to cancellation and never persist.

### 2.5 Integrate the existing Vue drawer

1. Extend TypeScript event guards and types for optional
   `specialistResult`.
2. Keep the run lifecycle, reconnect cursor, and de-duplication logic
   unchanged.
3. In `CopilotRunProgress.vue`, show capability label, aggregate outcome,
   warnings, truncation notice, and bounded source links with confidence and
   verification state. Use existing tokens/classes and 44px controls.
4. Continue rendering the final response through the existing safe Markdown
   path; source links in the structured projection must be validated and use
   safe external-link attributes.
5. Add no proposal, approval, save, bid, watch, or deep-identification action.

### 2.6 Documentation, OpenAPI, and rollout

1. Update Coin Copilot feature/API/testing docs with the four capabilities,
   evidence semantics, provider degradation, and explicit non-goals.
2. Update Swagger annotations/descriptions only where needed to describe the
   additive event projection; add no public route.
3. Run `task openapi` and require zero drift among `docs/openapi.json` and the
   generated Go Swagger artifacts.
4. Keep `CoinCopilotEnabled=false` as the rollout default. Disabling it stops
   new starts/resumes and selects legacy chat; existing runs remain
   readable/cancellable.
5. Add privacy-safe metrics/log fields: capability, provider id, provider
   outcome, aggregate outcome, duration, result count, bytes, truncation,
   digest, and run/execution ids. Exclude query text, result content, URLs,
   prompts, credentials, and raw exceptions.

## Test Strategy

### Go

- Shared fixture decoding and strict rejection of unknown/malformed fields.
- Tool/result-kind matching and exact ten-tool execution allowlist.
- Proof that Go internal Copilot routes remain the existing four collection
  callbacks and contain no specialist, write, arbitrary HTTP, or deep-ID route.
- Source URL scheme, embedded credentials, provider/source policy, provenance,
  deduplication, outcome aggregation, and payload-bound tests.
- Cancellation-vs-provider-completion race with zero late checkpoint/event.
- Restart/resume/replay seam test proving a completed specialist call is not
  executed again.
- Foreign and unknown run IDs remain indistinguishable.
- Existing feature-off/unsupported-model preflight creates no run and makes no
  specialist call.
- `go test -race ./...`, route/OpenAPI drift, and architecture tests.

### Python

- For each capability: success, no-match, timeout, transport failure,
  provider unavailable, malformed output, partial mixed outcome, and
  deterministic deduplication.
- Strict input/result Pydantic validation, maximum counts/string sizes, URL
  source policy, currency/price-basis separation, trend sufficiency, and
  similarity explanation.
- Existing specialist functions are invoked exactly once; no duplicated
  provider/search implementation in the Copilot module.
- Prompt-injection strings, token-shaped strings, unsupported claims, and
  fabricated URLs are removed/rejected and never interpreted as instructions.
- Cancellation before dispatch and after every awaited provider operation.
- Checkpoint hydration skips completed calls; top-level tool/iteration/time
  counters remain cumulative.
- Architecture guard remains DB/filesystem/shell free.

### Vue

- Runtime event guard accepts the additive projection and rejects invalid
  URLs, enums, oversized lists, and malformed provenance.
- Complete, partial, no-match, and unavailable states render accessibly.
- Every displayed claim links to its source and displays confidence and
  verification state; warnings and truncation remain visible.
- Reconnect/de-duplication preserves one result card per tool call.
- Cancellation, paused/resume, terminal close, mobile/PWA layout, and legacy
  fallback regressions remain green.

### Full quality gate

```powershell
Push-Location src\api
go build ./...
go vet ./...
go test ./...
go test -race ./...
Pop-Location

Push-Location src\agent
ruff check app/ tests/
pytest tests/ -v
Pop-Location

Push-Location src\web
npm run type-check
npm run test
npm run build
Pop-Location

task openapi
git diff --exit-code -- src/api/docs/docs.go src/api/docs/swagger.json src/api/docs/swagger.yaml docs/openapi.json
```

## Complexity Tracking

No constitution violation is introduced. The only new abstraction is a shared
typed specialist-result adapter, necessary to keep the four existing teams
canonical while giving the durable harness a uniform contract. A second router,
provider layer, database model, or public endpoint would be less simple and is
explicitly rejected.
