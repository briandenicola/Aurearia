# Phase 0 Research: Deep Agentic Coin Identification

**Feature**: 344-deep-agentic-coin-identification
**Date**: 2026-08-15
**Input**: `specs/344-deep-agentic-coin-identification/spec.md`
**Status**: Complete — no unresolved `NEEDS CLARIFICATION` remain (see §12)

This document resolves every open technical unknown from `plan.md` →
Technical Context, records provider/license findings with citations, and
documents the four decisions for which the repository has **no existing
precedent** (persisted SSE event retention, cancellation propagation,
provider calls in Go vs Python, draft bridging).

---

## 1. Existing code inventory (what we reuse, what we add)

All paths below were read or grepped in this repository; none are invented.

### 1.1 Go API (`src/api/`)

| Path | What exists today | Role in F344 |
|---|---|---|
| `src/api/handlers/coin_lookup.go` | `CoinLookupHandler.Lookup` → `POST /api/coins/lookup`, multipart field `images`, `fileToDataURI` from `handlers/helpers.go` | **Unchanged.** Fast path (FR-001). Reused as the "quick evidence" producer for deep runs. |
| `src/api/services/coin_lookup_service.go` | `CoinLookupService.Lookup(ctx, userID, CoinLookupRequest) (*CoinLookupResponse, error)`; `LookupExtractedData`, `NGCData`, `NumistaEvidence`, `ProposedNumistaQuery` | Reused **read-only** to build normalized quick-lookup context for a deep job. No signature change. |
| `src/api/models/ai_job.go` | `AIJob` (`analysis`/`value_estimate`/`coin_grading`; `queued/running/completed/failed`), coin-bound (`CoinID` non-optional in practice) | **Not mutated.** New sibling domain instead (see §5). Pattern donor only. |
| `src/api/repository/ai_job_repository.go` | `EnqueueOrFindActive`, `ClaimQueued`, `Complete`, `Fail`, `RecoverStaleJobs(timeout)` | Pattern donor for `DeepIdentificationRepository` (idempotency + stale recovery). |
| `src/api/services/ai_job_service.go` | In-process `queue chan uint` + `StartWorkers(n)`, `aiJobStaleTimeout=1h`, `aiJobAnalyzeTimeout=5m`, narrow `AIJobAgent` interface | Pattern donor for `DeepIdentificationService` worker pool and narrow agent interface. |
| `src/api/services/agent_proxy.go` | `AgentProxy{streamClient(no timeout), requestClient(5m)}`, `attachInternalCredential` → `X-Internal-Service-Token`, `proxySSE(ctx,w,path,payload)`, `collectSSE(ctx,path,payload)` | **Extended** with `StreamDeepIdentification(ctx, req, onEvent func(AgentDeepEvent) error) error` — a *callback* SSE consumer (new; neither pure proxy nor collect-only). |
| `src/api/handlers/agent.go` | `AgentHandler.ChatStream` → `POST /api/agent/chat`, mints `services.InternalTokenService.Mint(userID)` for Python→Go callbacks | Pattern donor for internal-token minting; **not** the SSE model we use for F344 (see §6). |
| `src/api/services/internal_token_service.go` | `Mint(userID)`, `Verify(token)` | Reused verbatim for provider tool callbacks. |
| `src/api/main.go:804-811` + `handlers/internal_tools.go` + `middleware.InternalTokenRequired` | `/api/internal/tools/*` group (`search_my_collection`, `get_coin`, …) callable by Python with a minted token | **Extended** with provider tools (`numista_search`, `numista_detail`, `nomisma_search`) — this is the resolution of "provider calls in Go vs Python" (§4). |
| `src/api/services/image_service.go` | `MaxImageUploadBytes = 20MB`, `NormalizeImageExt` allowlist `.jpg/.jpeg/.png/.gif/.webp`, `ValidateImageData` (magic bytes via `http.DetectContentType`), `UploadImage`, `DeleteImage`, `ResolveAuthorizedMediaPath` | Reused for deep-job artifact validation/storage/serving; new job-scoped storage dir + delete-on-terminal. |
| `src/api/models/coin.go:26-32,99` | `ImageType` = `obverse|reverse|detail|other`; `CoinImage{FilePath, ImageType, IsPrimary}` | Saved-image reuse source; role vocabulary extended (job-local `hint` role, never a `CoinImage`). |
| `src/api/models/quick_capture_draft.go`, `services/quick_capture_service.go` (`PromoteDraft`) | Draft → coin promotion incl. lifecycle audit events | **Reused as-is** for new-intake confirm (FR-033). |
| `src/api/models/coin_intake_draft.go`, `services/coin_intake_service.go` | AI intake draft + commit | Considered and rejected as the F344 bridge (§7). |
| `src/api/services/coin_service.go` (`UpdateCoinWithFields`) | Coin edit write path | **Reused as-is** for saved-coin proposal apply (FR-033). |
| `src/api/services/numista_client.go`, `numista_cache.go`, `numista_lookup_service.go`, `numista_query.go`, `numista_telemetry.go`; `models/numista.go` | F341 shared Numista boundary + TTL cache + coalescing + redacted telemetry (ADR 0007) | Reused **in Go** behind internal provider tools. |
| `src/api/services/nomisma_client.go`, `nomisma_cache.go`; `services/mint_location_service.go` | F343 Nomisma reconciliation client, typed `NomismaErrorKind`, never-5xx status contract (ADR 0009) | Reused **in Go** behind internal provider tools. |
| `src/api/database/database.go:36` | Single additive `AutoMigrate(...)` list; `database/migration_test.go` conventions | New models appended; `TestDeepIdentificationModelsAutoMigrate` added. |
| `src/api/architecture_test.go` | Layer matrix; `allowedServiceGORMFiles` map (transaction exceptions) | Gate to satisfy; add entry only if a transaction is orchestrated in the service (see plan.md Complexity Tracking). |
| `src/api/services/settings_service.go` | `AppSetting` key constants + `Default*` fallbacks; `ResolveLLMConfig()` | New `SettingDeepIdentification*` keys. |
| `src/api/route_openapi_drift_test.go`, `openapi_nullability_test.go` | Route/Swagger drift gate | New handlers need swaggo annotations + `task openapi`. |

### 1.2 Python agent (`src/agent/`)

| Path | What exists today | Role in F344 |
|---|---|---|
| `src/agent/app/routes.py` | Single `APIRouter(prefix="/api")`; `POST /api/analyze`, `/api/grade`, `/api/intake/draft`, SSE `POST /api/search/coins`, `/api/portfolio/review` | **Extended** with `POST /api/deep-identify/stream` (SSE). |
| `src/agent/app/streaming.py` | `format_sse(data)`, `stream_graph_events(graph, input, config)`, `_STATUS_MESSAGES`, `UserFacingTextSanitizer`/`sanitize_user_facing_text` (redacts JWT/Bearer) | Reused for sanitization; F344 emits an **application-owned typed envelope** rather than raw LangGraph event translation (§6.3). |
| `src/agent/app/supervisor.py` | `StateGraph(MessagesState)` + router node + `_RECURSION_CONFIG = {"recursion_limit": settings.max_supervisor_iterations}` (25) | Pattern donor for bounded graph config. |
| `src/agent/app/teams/coin_analysis.py` (`_build_image_contents`), `coin_intake.py` (`generate_intake_draft`), `json_extraction.py` (`extract_json_payload`) | Vision message construction, strict JSON extraction with graceful `ValidationError` fallback | Reused by the deep graph's vision-evidence node and synthesis node. |
| `src/agent/app/llm/provider.py` (`get_chat_model`), `app/llm/retry.py` (`ainvoke_with_retry`, tenacity 3 attempts) | Provider abstraction (anthropic/ollama) + retry | Reused unchanged. |
| `src/agent/app/models/requests.py` | `StrictRequestModel` (`extra="forbid"`), `LLMConfig`, `CoinData`, bounded `StringConstraints` | New `DeepIdentifyRequest` family follows the same strictness. |
| `src/agent/app/models/responses.py` | `IntakeDraftResponse`, `IntakeEvidenceItem`, `CandidateReference` | New `ProviderEvidence`/`DeepSynthesis` models follow the same shape rules. |
| `src/agent/app/tools/collection_tools.py` | `build_collection_tools(tools_base_url, internal_token)` → `StructuredTool` POSTing `/api/internal/tools/*` with `Authorization: Bearer <minted>` and `httpx.Timeout(connect=5, read=20, …)` | **Direct precedent** for new `build_provider_tools(...)` (§4). |
| `src/agent/app/security.py` | `InternalServiceAuthMiddleware` validating `x-internal-service-token` (`secrets.compare_digest`), bypass `{/health,/ready}` | Applies unchanged to the new endpoint. |
| `src/agent/app/outbound.py` | `validate_outbound_url`, `validate_public_outbound_url`, `safe_get` (no auto-redirect, max 5 validated hops) | Any direct provider fetch from Python must go through this; F344 avoids direct provider egress entirely (§4). |
| `src/agent/app/config.py` | `Settings` env prefix `AGENT_`, `max_supervisor_iterations=25`, `verification_timeout=10` | New `AGENT_DEEP_*` bounds. |
| `src/agent/tests/test_streaming.py` (`FakeGraph`, `collect_sse`) | SSE unit-test harness without a real LLM | Direct donor for deep-graph SSE tests. |

### 1.3 Vue SPA (`src/web/`)

| Path | What exists today | Role in F344 |
|---|---|---|
| `src/web/src/pages/CoinLookupPage.vue` (507 lines, states `capture/analyzing/results`) | Quick Identify Coin; `lookupCoin(images)`; `createQuickCaptureDraft`; `InlineCameraCapturePanel`, `NumistaLookupPanel` | **Extended by one secondary CTA only**; deep UI lives in new page/components to avoid an oversized-page regression (largest page today: `SetDetailPage.vue` 720 lines). |
| `src/web/src/api/client.ts` | Single axios instance + 401 refresh/replay; `agentChatStream` = `fetch` + `ReadableStream` + `Authorization` header + `fetchWithAuthRetry` | Donor for the deep SSE reader; new typed wrappers added here (all API access must go through this file, Principle III). |
| `src/web/src/composables/useCoinSearchChat.ts` (575 lines) | Stream lifecycle composable over `agentChatStream` | Pattern donor for `useDeepIdentificationStream`. |
| `src/web/src/components/coin/CoinAIAnalysis.vue` | Inline AI-job polling, `isTerminalStatus/isFailedStatus`, resume-on-mount via `getCoinAIJobs(activeOnly)` | Pattern donor for resume-on-mount and terminal-state classification; also the saved-coin entry point neighbourhood. |
| `src/web/src/components/quick-capture/QuickCaptureImageSlots.vue`, `components/InlineCameraCapturePanel.vue`, `CameraCaptureModal.vue`, `composables/useImageProcessor.ts` | Obverse/reverse slot capture + processing | Reused for required obverse/reverse and hint image inputs. |
| `src/web/src/pages/SetProposalReviewPage.vue` (510 lines) | Proposal review layout: status hero, recommendation chips, issue cards, accept/adjust | Direct UX donor for the deep report/proposal review screen. |
| `src/web/src/composables/useToast.ts`, `useDialog.ts`, `usePwa.ts`, `useNetworkQuality.ts` | Toasts, confirm dialogs, PWA/network signals | Reused (disconnect indicator uses `useNetworkQuality`). |
| `src/web/src/assets/styles/variables.css` | Design tokens (`--accent-gold`, `--bg-card`, `--text-muted`, …) | Mandatory (Principle VI); no hardcoded visual values. |

---

## 2. Provider contracts, licensing, and staging gates

Research performed 2026-08-15 against primary sources. **Where a source could
not be fetched directly, that is stated explicitly.**

### 2.1 Nomisma.org — automatable (MVP)

- **APIs (documented, unauthenticated)**: SPARQL `http://nomisma.org/query`;
  REST `http://nomisma.org/apis/getLabel|getRdf|getMints`; OpenRefine
  reconciliation `http://nomisma.org/apis/reconcile`; per-concept content
  negotiation (`Accept: application/ld+json`).
  Sources: `http://nomisma.org/documentation/apis/`,
  `http://nomisma.org/documentation/sparql/`.
- **Auth / limits**: none required; **no published rate limits** (ANS
  community infrastructure, no SLA) → we must self-throttle.
- **License**: Nomisma concepts **CC BY**; aggregated partner datasets
  **ODC-ODbL** (`http://nomisma.org/datasets`). Attribution required.
- **F344 decision**: MVP automated provider node, executed through the
  **existing Go client** `src/api/services/nomisma_client.go` (F343/ADR 0009),
  which already returns typed `NomismaErrorKind` and never 5xx. UI shows
  "Data: Nomisma.org (CC BY)" attribution on any Nomisma-derived claim.

### 2.2 Numista — automatable (MVP), with license constraints

- **API**: `https://api.numista.com/v3/`; `GET /types` (search),
  `GET /types/{id}` (detail), `POST /types/search_by_image` (paid only).
  Header `Numista-API-Key`. Docs `https://en.numista.com/api/doc/index.php`.
- **Limits**: max **10 simultaneous requests** per account; free plan
  **2,000 requests/month**; paid plan has a **15:1 detail-to-search ratio cap**.
  Source: `https://en.numista.com/api/pricing.php`,
  `https://en.numista.com/api/license.php`.
- **License**: mandatory visible attribution ("Source: Numista") **and**
  visible N# per result; no redistribution as a database/API; **catalogue data
  other than N# and catalogue metadata must not be persistently cached**
  (metadata ≤ 7 days), *except for private personal projects*.
- **F344 decision**: MVP automated provider node via the **existing Go client**
  `src/api/services/numista_client.go` + `numista_cache.go` (F341/ADR 0007), so
  the existing quota-aware, coalescing, TTL-bounded, redacted-telemetry
  boundary is not duplicated. **Compliance constraint recorded**: the persisted
  deep report stores N#, canonical URL, and the owner-facing claim text needed
  for review; it must not become a durable mirror of Numista type detail. This
  app is self-hosted and single-owner ("private personal project"), which is the
  license carve-out we rely on; this is called out in plan.md → ADR-0010 scope.
  Free-tier 2,000/month is the practical ceiling → per-job Numista call budget
  is bounded (max 1 search + ≤3 details per job, §8).

### 2.3 NGC — **not automatable**; OCR + link-out only

- **No public API exists.** Only the human web tool
  `https://www.ngccoin.com/verify/` (cert format `XXXXXXX-XXX`).
- **Terms of Website Use** (`https://www.ngccoin.com/legal/terms-of-use/`, §2
  Prohibited Uses) explicitly forbid: *"Use any robot, spider, or other
  automatic device, process, or means to access the Website for any purpose"*.
- **F344 decision**: NGC is a **`not_automated` provider node**. It contributes
  only (a) the cert number already extracted by the existing OCR path
  (`CoinLookupService.extractNGCCert` / `LookupExtractedData.NGC`) and (b) a
  canonical `https://www.ngccoin.com/verify/` link-out rendered via
  `src/web/src/components/SafeExternalLink.vue`. **No HTTP request is made to
  ngccoin.com by this feature.** This matches spec Assumptions and Non-Goals.

### 2.4 OCRE — technically automatable, **gated** (post-MVP)

- Access is via the Nomisma triplestore (`http://nomisma.org/query`, URI prefix
  `http://numismatics.org/ocre/id/`) and per-record content negotiation; bulk
  RDF at `http://numismatics.org/ocre/nomisma.rdf`.
- **License: ODC-ODbL** with share-alike obligations on derivative *databases*;
  attribution to the American Numismatic Society.
- Observed during research: `numismatics.org/ocre/apis` returned HTTP 500 and
  the OCRE web front end was intermittently unavailable; Nomisma SPARQL is the
  stable path. No documented rate limits.
- **F344 decision**: ship as **`not_automated` in MVP**, behind validation gate
  **G-OCRE** (§9). It remains a first-class node in the target architecture and
  needs **no contract respecification** to switch on — only a provider adapter
  plus an ODbL/attribution review recorded in the ADR.

### 2.5 RPC Online — **blocked**, manual reference only

- **No documented API of any kind**; `rpc.ashmus.ox.ac.uk` returned **HTTP 403
  to non-browser clients** during research (terms page not directly fetchable;
  license characterisation below comes from indexed/secondary sources citing
  `https://rpc.ashmus.ox.ac.uk/terms`).
- **License: CC BY-NC-SA 4.0** for text/data (non-commercial, share-alike);
  images carry separate third-party rights requiring individual clearance.
- **F344 decision**: RPC is **`unavailable`/manual-reference only**. No
  automated access is planned or implemented. Gate **G-RPC** (§9) requires
  written permission or a documented API before any adapter work starts.
  **This is not a contradiction with the spec** — spec Assumptions and
  Non-Goals already stage OCRE/RPC and forbid scraping undocumented endpoints.

### 2.6 Contradiction check (STOP condition)

None found. The provider realities (NGC no-API + robot prohibition; RPC no-API
+ NC license + 403 blocking; OCRE ODbL + flaky front end) are **already**
encoded in spec Assumptions ("Provider phase boundaries (Phase 1 / MVP
honesty)") and Non-Goals ("Scraping or reverse-engineering undocumented NGC or
RPC Online endpoints… out of scope"). FR-025's `not_automated` / failed /
unavailable statuses exist precisely to represent this. **Planning proceeds.**

The one item to watch (recorded, non-blocking): Numista's "do not persistently
cache catalogue data" clause vs. our persisted report. Mitigation above; must be
restated in ADR 0010 and reviewed if this app ever ceases to be a private
personal deployment.

---

## 3. Decision: new sibling job domain (not `AIJob` mutation)

- **Decision**: create `DeepIdentificationJob` + append-only
  `DeepIdentificationEvent` + `DeepIdentificationProviderRun` +
  `DeepIdentificationArtifact` in `src/api/models/`, siblings to
  `models.AIJob`.
- **Rationale (Principle IV — simple, complete, proportional; Principle I)**:
  `AIJob` is coin-bound (`CoinID` + `Side`), has a single opaque `Result` text
  column, four statuses, no event log, no artifacts, no provider set, no retry
  lineage, and its composite index
  `idx_ai_jobs_user_coin_type_side_status` encodes coin-bound idempotency.
  F344 requires optional coin linkage (new intake), six statuses incl.
  `partial`/`cancelled`, replayable event sequence, per-provider run rows,
  ephemeral artifacts, cancellation requests, and input-fingerprint
  idempotency. Retrofitting these onto `AIJob` would add ~10 nullable columns
  and change the meaning of an existing index used by shipped flows
  (`repository.EnqueueOrFindActive`), i.e. a *larger* and riskier change with
  cross-feature blast radius (§21.7 workflow contracts) than an additive
  sibling table set.
- **Alternatives rejected**: (a) extend `AIJob` — rejected above; (b) reuse
  `CoinIntakeDraft` as the job carrier — rejected, it is a draft not a job and
  has no event/replay semantics; (c) event-sourced generic job framework —
  rejected as disproportionate for one feature on single-node SQLite.

---

## 4. Decision: provider calls execute in **Go**, invoked by Python as internal tools

The spec's binding architecture says Python is stateless inference/provider
routing; the Constitution (Principle II) says Go holds no LLM logic and Python
holds no DB/secrets. Provider access sits between those.

- **Decision**: **provider data fetch stays in Go**, exposed to Python through
  the already-shipped internal tool-server channel:
  `main.go:804` `internal := r.Group("/api/internal/tools")` +
  `middleware.InternalTokenRequired(internalTokenSvc)` +
  `handlers/internal_tools.go`. New internal endpoints:
  `POST /api/internal/tools/numista_search`, `/numista_detail`,
  `/nomisma_search`. Python gains
  `src/agent/app/tools/provider_tools.py::build_provider_tools(tools_base_url,
  internal_token, allowed_providers)` modelled exactly on
  `src/agent/app/tools/collection_tools.py`.
- **Rationale**:
  1. Numista requires an API key that lives in `AppSetting`
     (`SettingNumistaAPIKey`) — sending it to a stateless Python service would
     widen secret exposure (Principle V) and duplicate the F341 quota/TTL/
     coalescing/telemetry boundary (ADR 0007).
  2. Nomisma's typed `NomismaErrorKind` + never-5xx contract (ADR 0009) already
     exists in Go; duplicating it in Python creates two divergent status
     vocabularies for the same upstream.
  3. Python stays stateless: it holds no keys, no cache, no DB; it calls a
     typed, authenticated, owner-scoped Go tool with a short-lived minted token
     — the exact precedent set by F012/F217 collection tools.
  4. The LLM never chooses an arbitrary URL: provider tools are a fixed,
     enum-bounded StructuredTool set; free-form web fetching
     (`app/tools/search.py`, `app/outbound.py`) is **not** wired into this graph.
- **Alternatives rejected**: (a) duplicate Numista/Nomisma HTTP clients in
  Python — rejected (secret spread, two caches, two status vocabularies,
  license-quota accounting in two places); (b) Go orchestrates providers and
  calls Python only for synthesis — rejected because router/evaluator/synthesis
  are LLM-driven and Principle II forbids agent logic in Go; (c) Python calls
  providers directly via `safe_get` with an egress allowlist — rejected for
  Numista (key) and unnecessary for Nomisma given (2).

---

## 5. Decision: Go owns SSE; Python→Go is an internal stream

- **Decision**: Python streams typed events to Go over SSE (
  `POST /api/deep-identify/stream`); the Go worker **persists each event** as a
  `DeepIdentificationEvent` row and republishes to an in-process broker. The
  browser connects to a **Go-owned** SSE endpoint
  `GET /api/deep-identification/jobs/{id}/events?since=N` (also honouring
  `Last-Event-ID`), which **replays from the database first**, then follows live.
- **Rationale**: FR-009/FR-015/FR-016 require background continuation and
  gap-free resume, which a pass-through byte proxy (`AgentProxy.proxySSE`)
  cannot provide — the Python stream is bound to one request. Constitution
  Principle II ("SSE streams flow Python → Go → Vue") is still honoured
  directionally; Go simply persists and re-serves rather than blindly relaying.
  This deviation is recorded in plan.md → Complexity Tracking and ADR 0010.
- **Alternatives rejected**: (a) direct proxy — cannot resume, dies with the
  client (violates FR-009); (b) polling only (like `CoinAIAnalysis.vue`) —
  fails FR-015's "as they occur" and yields poor 5-minute UX; (c) WebSockets —
  new transport, no precedent in the repo, disproportionate (Principle IV).

---

## 6. Decision (no precedent): persisted SSE event retention

- **Decision**: `DeepIdentificationEvent` rows are retained **24 hours after
  the job reaches a terminal state** (setting
  `deep_identification_event_retention_hours`, default 24), while the
  **report/draft is retained 90 days** (setting
  `deep_identification_result_retention_days`, default 90). A reconnect asking
  for a pruned `since` receives an `event: stream_truncated` frame carrying the
  current job status + earliest retained sequence, then the retained tail —
  never an empty 200 and never a bare error (FR-017, edge case "pruned event
  ID").
- **Rationale**: 24 h comfortably covers "closed laptop lid"/mobile-network
  scenarios named in spec Assumptions while bounding SQLite growth on a
  single-node personal deployment (a 5-minute job emits O(50–200) events).
  Decoupling result retention satisfies FR-034 explicitly.
- **Alternatives rejected**: indefinite retention (unbounded SQLite growth,
  violates "bounded retention"); in-memory ring buffer only (loses replay
  across restart, violates FR-012); retention tied to hint-image cleanup
  (explicitly forbidden by FR-034).

---

## 7. Decision (no precedent): cancellation propagation

- **Decision**: three-layer cooperative cancellation.
  1. `POST /api/deep-identification/jobs/{id}/cancel` writes
     `cancel_requested_at` **without** changing status (single UPDATE, owner
     scoped).
  2. The Go worker owning the job holds a `context.CancelFunc`; the handler
     signals it through an in-process registry keyed by job ID. Cancelling the
     context terminates the HTTP request to Python, which aborts the graph run
     (FastAPI/`httpx` client disconnect).
  3. A worker that is *not* on this process (post-restart edge) still converges:
     the claim/heartbeat loop checks `cancel_requested_at` between graph
     phases and stops issuing new provider work.
  - **Race resolution**: the terminal transition is a single conditional UPDATE
    `... WHERE id = ? AND status IN ('queued','running')`; the first writer
    wins and `RowsAffected` decides. If cancel lands first, a later natural
    completion is **discarded** (report not stored, per FR-019/"in-flight
    provider calls are abandoned"); if completion lands first, cancel returns
    `409` with the settled terminal state. Exactly one terminal event is ever
    appended, because the event append happens in the same transaction as the
    winning UPDATE.
- **Rationale**: reuses Go's standard `context` cancellation and the existing
  claim pattern (`ClaimQueued`) without adding a message bus; the conditional
  UPDATE gives a single source of truth on SQLite (no advisory locks needed).
- **Alternatives rejected**: killing the Python process/graph out-of-band (no
  mechanism, Python is stateless and shared); status-column-only cancellation
  without a separate `cancel_requested_at` (loses the "requested but not yet
  settled" state and makes the race unresolvable).

---

## 8. Decision (no precedent): draft bridging — **no new draft model**

- **Decision**: the deep result is stored **on the job** (`report_json`,
  `proposal_json`) and applied through existing write paths:
  - **New intake**: on confirm, Go maps the proposal to the existing
    `QuickCaptureDraft` create/update payload (same normalization
    `src/web/src/utils/coinLookupDraft.ts` performs today for the quick path),
    then the owner promotes via the shipped
    `services.QuickCaptureService.PromoteDraft`.
  - **Saved coin**: on confirm, the selected fields are applied through
    `services.CoinService.UpdateCoinWithFields(existing, updates,
    updateFields, userID, source, …)` with `source = "deep_identification"`.
  - The job records `applied_draft_id` / `applied_coin_id` / `applied_at` for
    traceability (SC-009).
- **Rationale**: FR-033 forbids a third write implementation.
  `CoinIntakeDraft` was evaluated as a bridge and rejected: it carries its own
  `drafted/confirmed/discarded/expired` lifecycle and its own commit path
  (`services/coin_intake_service.go`), so routing through it would mean two
  drafts (intake draft **and** quick-capture draft) for one confirmation, or a
  second promotion implementation — the opposite of Principle IV.
- **Alternatives rejected**: (a) new `DeepIdentificationDraft` model — third
  write path, rejected by FR-033; (b) writing coins directly from the deep
  service — forbidden by FR-031/FR-035.

---

## 9. Runtime budget, bounds, and provider quotas

Target: terminal state ≤ 5 minutes (FR-014, SC-002). Allocation (all values are
settings, not product commitments — spec Assumptions):

| Phase | Budget | Bound source |
|---|---|---|
| Quick-lookup context build (Go, reuses `CoinLookupService.Lookup`) | 45 s | `deep_identification_context_timeout_seconds` |
| Router node (1 LLM call) | 25 s | `AGENT_DEEP_ROUTER_TIMEOUT` |
| Provider fan-out (≤ 4 providers, ≤ 2 concurrent) | 100 s wall | `AGENT_DEEP_PROVIDER_TIMEOUT=45`, `AGENT_DEEP_MAX_CONCURRENCY=2`, `AGENT_DEEP_MAX_PROVIDERS=4` |
| Contradiction/provenance evaluator (1 LLM call) | 40 s | `AGENT_DEEP_EVAL_TIMEOUT` |
| Synthesis + typed proposal (1 LLM call) | 60 s | `AGENT_DEEP_SYNTH_TIMEOUT` |
| Slack / persistence | 30 s | — |
| **Hard caps** | Python `AGENT_DEEP_TOTAL_TIMEOUT=280 s`; Go `deepJobHardTimeout = 300 s` (`context.WithTimeout`) | FR-014 |

- Graph recursion limit `AGENT_DEEP_RECURSION_LIMIT = 12` (cf. supervisor's 25).
- **Numista budget per job**: ≤ 1 search + ≤ 3 detail calls (respects the 15:1
  detail:search ratio and keeps the free tier's 2,000/month meaningful);
  concurrency never exceeds the account's 10 simultaneous-request cap because
  Go-side fan-out is ≤ 2.
- **Nomisma budget per job**: ≤ 3 reconciliation/SPARQL calls, self-throttled
  (no documented upstream limit).
- **Per-user active job limit**: 1 (`deep_identification_max_active_per_user`);
  a second start returns the in-flight job (FR-007) or `429` if the inputs
  differ.
- **Global worker concurrency**: 2 (`deep_identification_worker_count`), queue
  depth 32 with backpressure → `503 job_queue_full` (never silent drop).

---

## 10. Image pipeline decisions

- **Transport**: `multipart/form-data` on job creation with named parts
  `obverse`, `reverse`, `hints[]` (max 3), plus `notes` and
  `providers[]`. Rejects a request supplying files under the wrong role
  (edge case: mislabeled hint).
- **Validation**: reuse `services.ValidateImageData` (magic bytes),
  `NormalizeImageExt` (allowlist), `MaxImageUploadBytes` (20 MB/file), and
  Gin's `MaxMultipartMemory` cap; total request cap 4 files.
- **Saved-image reuse**: when `coinId` is supplied and a role is omitted, Go
  resolves that coin's `CoinImage` rows of `ImageTypeObverse`/`ImageTypeReverse`
  owned by the caller. If either is missing → `422` (FR-003, US2 AC2).
- **Storage**: job artifacts under `<uploadDir>/deep-jobs/job-<id>/` (new,
  parallel to `coin-<id>/`), owner-scoped, served only via
  `ResolveAuthorizedMediaPath`. Hint images are **never** written to
  `coin_images` and have no coin-image endpoint.
- **Transfer to Python**: base64 data URIs in the request body (exactly the
  existing `handlers/helpers.go::fileToDataURI` + `_build_image_contents`
  convention). No signed URLs, no Python filesystem access.
- **EXIF/privacy**: no EXIF parsing exists in the repo today; F344 does not add
  metadata extraction. Images are re-encoded through the existing variant
  pipeline for coin-face artifacts; hint images are stored raw and deleted at
  terminal. Documented stance: metadata is neither read nor forwarded beyond
  the raw bytes already handled by existing uploads.
- **Cleanup**: hint artifacts deleted on **any** terminal transition (FR-021/
  FR-030) inside the same terminal transaction's post-commit hook; a startup
  sweep plus an hourly janitor deletes (a) hint artifacts of terminal jobs,
  (b) orphaned directories with no job row, (c) directories for jobs older than
  the result-retention window. Restart safety: file deletion is idempotent and
  `artifact.deleted_at` marks the DB truth.

---

## 11. Testing decisions

- **No real provider calls in CI** — Go provider tools are tested against
  `httptest` servers exactly as `numista_client_test.go`/`nomisma_client_test.go`
  do; Python graph tests use a `FakeGraph`/fake tool-callable pattern
  (`tests/test_streaming.py`); Vue tests stub `client.ts` wrappers.
- **Determinism**: LLM nodes are exercised through injected fake chat models
  (no network); routing/merge/contradiction logic is pure-function tested.
- **Fast-path regression**: an explicit test asserting `POST /api/coins/lookup`
  request/response shape and that no `DeepIdentificationJob` row is created
  (SC-008/FR-001).

---

## 12. Resolved Technical Context unknowns

| Unknown (plan.md) | Resolution | Section |
|---|---|---|
| Where do provider HTTP calls execute? | Go, behind `/api/internal/tools/*` | §4 |
| How is SSE resumable across disconnects/restarts? | Go persists events, serves own SSE with `since`/`Last-Event-ID` | §5, §6 |
| Event retention window? | 24 h post-terminal; result 90 d | §6 |
| Cancellation mechanism? | `cancel_requested_at` + context registry + conditional terminal UPDATE | §7 |
| Draft bridge model? | None — reuse QuickCaptureDraft promote / CoinService.UpdateCoinWithFields | §8 |
| Which providers are automatable at MVP? | Nomisma, Numista; NGC OCR+link-out; OCRE/RPC gated | §2 |
| 5-minute budget allocation? | Table in §9 | §9 |
| Image transport/limits/cleanup? | multipart + data URIs, 20 MB/file, ≤4 files, delete on terminal | §10 |
| Frontend streaming approach? | `fetch` + `ReadableStream` reader like `agentChatStream` | §1.3 |
| EXIF stance? | No EXIF handling exists; none added | §10 |
