# Quickstart: Implement and Verify Improved Numista Lookup

## Prerequisites

- Branch: `341-improve-numista-lookup`
- Go toolchain matching `src/api/go.mod` (Go 1.26.5)
- Node/npm for `src/web`
- No real Numista key is required for tests
- Read the constitution, active spec, plan, research and data model first

Treat the amended `spec.md` as authoritative. Do not use a live provider in
automated tests.

## Recommended implementation order

### 1. Build the typed backend foundation

1. Add application DTOs/enums in `src/api/models/numista.go`.
2. Add `NumistaClient`, private provider DTOs, outbound HTTP configuration and
   typed errors in `services/numista_client.go`.
3. Add bounded TTL/coalescing cache with injected clock.
4. Add the pure versioned scorer and reason codes.
5. Add rolling redacted telemetry.
6. Add validated Numista settings/defaults.

Run package-level tests after each seam:

```powershell
Set-Location src/api
go test ./services -run "Test(HTTPNumistaClient|NumistaCache|NumistaScorer|NumistaTelemetry)" -count=1
go vet ./...
```

Client tests must use a fake `RoundTripper` or `httptest.Server` and assert the
API key never appears in errors, cache keys, responses, or telemetry.

### 2. Add broad lookup and enrichment contracts

1. Implement `NumistaLookupService`.
2. Replace handler-owned HTTP in `handlers/numista.go`.
3. Add typed POST lookup/enrichment routes and the deprecated GET adapter.
4. Inject one shared service from `main.go`.
5. Add admin health endpoint under the existing admin group.
6. Regenerate Swagger/OpenAPI.

Expected local behavior with a fake provider:

```text
POST /api/numista/lookup
  -> broad success/empty or explicit terminal status
  -> cache metadata

POST /api/numista/enrich
  -> all broad candidates retained
  -> only configured leading subset detailed
  -> deterministic rerank

GET /api/numista/search?q=...
  -> legacy {count,types} response
```

Validate:

```powershell
Set-Location src/api
go test ./handlers ./services -run "Test.*Numista" -count=1
go test ./... -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI -count=1
```

### 3. Integrate photo analysis

Change `CoinLookupService` to inject the shared query builder/lookup types.
For a non-NGC photo it returns typed evidence and a proposed query without a
provider call. For a usable NGC cert it preserves NGC-first behavior. Remove
top-candidate inference and never produce Numista candidate references.

Validate:

```powershell
Set-Location src/api
go test ./services ./handlers -run "Test.*CoinLookup" -count=1
```

Required cases: no image, NGC result, no-NGC typed proposal, noisy/empty
evidence, cancellation, and legacy additive fields.

### 4. Add selected-reference persistence

1. Add `QuickCaptureDraftReference` and AutoMigrate entry.
2. Extend draft create/read/update DTOs and repository transactions.
3. Extend promotion transaction to copy the selected reference.
4. Preserve selection on omitted updates/validation errors; support explicit
   clear and replace.

Validate:

```powershell
Set-Location src/api
go test ./database -run "Test.*QuickCapture.*Migration" -count=1
go test ./repository ./services ./handlers -run "Test.*QuickCapture" -count=1
```

Inspect transaction tests for both collection and wishlist, no selection,
owner isolation, invalid canonical URL, rollback, two concurrent promotions,
and repeated idempotent promotion.

### 5. Implement the Vue workflows

1. Add exact shared DTOs to `types/index.ts` and calls to `api/client.ts`.
2. Create the reusable lookup panel and pure query/status helpers.
3. Integrate direct coin selection with existing reference creation.
4. Integrate photo broad-first lookup and draft save.
5. Integrate draft resume/replace/remove.
6. Add admin telemetry/configuration UI.

Validate:

```powershell
Set-Location src/web
npm run test -- --run
npm run type-check
npm run build
```

Targeted tests must assert:

- proposed query includes only non-empty evidence and remains editable;
- the exact edited query reaches the API on first search and retry;
- broad results render while enrichment is pending;
- all six statuses and role-safe guidance;
- cached/fresh indicators;
- deterministic rerank does not silently change explicit selection;
- selection absent/outside latest results/replace/remove;
- draft save/resume and promotion validation failure preserve selection;
- keyboard selection, focus, `aria-live`, non-color explanations;
- narrow mobile viewport does not overflow.

### 5A. Reconcile the approved canonical-placement UX

Implement this P1 amendment after the completed T001–T053 foundation and before
declaring the MVP user workflow complete. It does not wait for Phase 6 caching/
telemetry or Phase 7 enrichment.

1. Compose saved-coin lookup inside
   `src/web/src/components/coin/CoinReferencesSection.vue`, with compact
   `Search Numista` beside `Add Reference`.
2. Collapse the inline panel only after selected-reference persistence succeeds
   and the refreshed reference appears.
3. Remove the full panel from
   `src/web/src/components/coin/CoinActionsPanel.vue`; retain at most a compact
   contextual link to Catalog References.
4. In `src/web/src/pages/CoinLookupPage.vue`, keep non-NGC lookup contextual,
   add `Also search Numista` beneath usable NGC results, and assert revealing
   the panel makes zero Numista requests before explicit submission.
5. Rename the initial action to `Analyze Photos`; keep `Save as Draft`.
6. Render `Numista #<identifier>` in
   `src/web/src/components/quick-capture/QuickCaptureDraftCard.vue` from the
   existing owner-scoped list response.
7. Do not add a route, sidebar item, top-level menu item, schema migration,
   endpoint, cache/telemetry behavior, or enrichment behavior.

Targeted validation:

```powershell
Set-Location src/api
go test ./repository ./services ./handlers -run "Test.*QuickCapture.*(List|Numista)" -count=1

Set-Location ..\web
npm run test -- --run src/components/coin/__tests__/CoinReferencesSection.test.ts src/components/coin/__tests__/CoinActionsPanel.test.ts src/pages/__tests__/CoinLookupPage.test.ts src/components/quick-capture/__tests__/QuickCaptureDraftCard.test.ts
npm run type-check
npm run build
```

### 6. Update contracts and lower-authority docs

Regenerate all repository-standard Swagger files and update:

- `docs/features/numista-integration.md`
- `docs/quick-capture.md`
- `docs/api-reference.md`
- `docs/deployment.md`
- `docs/openapi.json`
- a Nygard ADR for shared client/cache/migration behavior

Do not copy the outdated claim that photo lookup automatically attaches or
generates Numista references. The active spec requires explicit selection.

## Manual integration walkthrough

Use a local stub server through the injected test-only base URL or a
development key. Never commit the key.

1. **Direct**: open a coin with ruler, denomination, mint, date, material and
   inscriptions. Confirm the proposed query, edit it, search, observe broad
   results, then enriched reasons. Select one and explicitly add it. Verify one
   `CoinReference`.
2. **Photo**: upload non-NGC photos. Confirm no Numista request happens before
   editing/submitting the proposed query. Select one result and save a Quick
   Capture draft.
3. **Resume**: edit unrelated draft fields, cause one validation failure, and
   retry lookup where the selected ID is absent. Confirm selection persists
   until explicitly removed/replaced.
4. **Promote**: promote once to collection and once in a separate case to
   wishlist. Retry promotion. Verify exactly one selected reference.
5. **Statuses**: simulate empty, missing key, 429 with Retry-After, deadline,
   and 503/malformed response. Confirm distinct safe guidance and retained
   query.
6. **Caching**: repeat equivalent normalized searches and details before/after
   fake-clock expiry. Confirm fresh hit indicators and provider call counts.
7. **Partial detail failure**: fail some/all details. Confirm every broad
   candidate remains selectable and overall search is not `empty`.
8. **Admin**: verify status counts, p50/p95, cache/enrichment/quota signals and
   that no key/query/inscription/label text is visible.
9. **NGC**: upload a usable NGC slab and confirm Numista is not automatically
   requested. Activate `Also search Numista`, confirm the editable panel is
   revealed with no request, then submit explicitly.
10. **Canonical placement**: on a saved coin, confirm `Search Numista` is beside
    `Add Reference`, expands inline, persists through the structured-reference
    API, collapses after success, and is absent as a full panel from Actions.
11. **Draft list**: save a selected Numista reference and confirm the list card
    shows `Numista #<identifier>`; confirm an unselected draft has no chip.
12. **Labels/navigation/mobile**: confirm `Analyze Photos`, `Save as Draft`, no
    new top-level navigation, keyboard disclosure/focus behavior, and no 375 px
    horizontal overflow.

## Full quality gate

From `src/api`:

```powershell
go build ./...
go vet ./...
go test ./... -count=1
```

From `src/web`:

```powershell
npm run test -- --run
npm run type-check
npm run build
```

From repository root, run applicable Taskfile contract/documentation checks
after inspecting `task --list`. Python checks are not required unless Python
files change. Before completion, also verify:

- no secret or generated build artifact in the diff;
- OpenAPI route-drift test passes;
- all new service methods have unit tests;
- exact workflow/blast-radius tests cover direct, photo, Quick Capture,
  collection, wishlist and idempotency;
- PR notes cite Constitution Principles I–IX as applicable, §17 and §21.

## Phase 8 beta verification evidence

**Run:** 2026-08-11 on `beta` at `011c635`.

### Contracts and task checks

- `task openapi` — PASS. Regenerated
  `src/api/docs/docs.go`, `src/api/docs/swagger.json`,
  `src/api/docs/swagger.yaml`, and `docs/openapi.json`.
- A second `task openapi` followed by
  `git diff --exit-code -- src/api/docs/docs.go src/api/docs/swagger.json src/api/docs/swagger.yaml docs/openapi.json`
  — PASS; generation is byte-stable.
- The planning contract's eight `/api/...` routes all match generated Swagger
  after applying generated `basePath: /api`; methods also match.
- `task --list` — PASS. `openapi` is the only applicable contract/docs task.

### Quality gates

- From `src/api`:
  `go build ./...`; `go vet ./...`; `go test ./... -count=1`;
  `go test -run TestArchitecture ./... -count=1`;
  `go test -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI ./... -count=1`
  — PASS. The full suite includes the uncommitted `integration` package.
- From `src/web`:
  `npm.cmd run test -- --run`; `npm.cmd run type-check`;
  `npx.cmd vue-tsc --build`; `npm.cmd run build` — PASS.
  Vitest: 112 files, 689 tests. PowerShell blocked the unsigned `npm.ps1`
  shim, so the equivalent Windows `npm.cmd`/`npx.cmd` shims were used.
- From `src/agent`: `ruff check app/ tests/` — PASS;
  `pytest tests/ -v` — PASS, 189 tests with 9 existing LangGraph
  deprecation warnings. The local interpreter was Python 3.14.3 rather than
  the CI-pinned 3.12, but no environment exception affected the result.

### Walkthrough evidence

The non-interactive beta verification used deterministic fake-provider API
integration tests and Vue interaction tests; it did not use a real Numista
credential.

1. Direct selected-only persistence and deduplication — PASS:
   `TestNumistaWorkflowDirectPersistsOnlyExplicitSelection`.
2. Non-NGC photo proposal, explicit selection, and draft save contract — PASS:
   compatibility/integration tests plus `CoinLookupPage.test.ts`.
3. Resume, validation failure, outside-result retention, replace/remove —
   PASS in the full `QuickCaptureDraftPage.test.ts` suite.
4. Collection/wishlist promotion, no-selection, and retry idempotency — PASS:
   `TestNumistaWorkflowPhotoDraftPromotionCollectionWishlistNoSelectionAndRetry`.
5. Six status presentations, retained query/selection, and safe guidance —
   PASS in `NumistaLookupPanel.status.test.ts` and backend status tests.
6. Fresh-cache reuse, expiry behavior, call reduction, and latency budgets —
   PASS in integration/cache tests; deterministic workload achieved at least
   80% broad-call reduction, uncached p95 <=5s, and fresh-cache p95 <=1s.
   The SC-002 ranking benchmark contains 24 distinct known-coin fixtures
   spanning Roman imperial, Greek/Hellenistic, Byzantine, and medieval
   issues. Every fixture uses six plausible candidates, three deterministic
   input permutations, and noisy evidence across title/ruler, denomination,
   date, mint, material, and inscriptions where available. No fixture supplies
   `ExactNumistaID`. The correct candidate ranked top-three in 24/24 fixtures
   (100%), above the required 85%, while every run retained the default
   five-detail ceiling. The benchmark also requires the correct candidate to
   emit match reasons for every supplied scorer field and to outscore fourth
   place, so removing a scoring dimension or collapsing discrimination fails
   with fixture/permutation/rank/score/reason diagnostics.
7. Partial/all detail failure retention and bounded enrichment — PASS in
   `NumistaLookupPanel.enrichment.test.ts` and backend enrichment tests.
8. Admin settings, status/cache/enrichment/quota metrics, and redaction —
   PASS in `AdminSystemSection.numista.test.ts` and security/health tests.
9. NGC-first behavior and zero eager Numista request before explicit submit —
   PASS in compatibility tests and `CoinLookupPage.test.ts`.
10. Canonical Catalog References placement, collapse after refreshed
    persistence, and no full Actions panel — PASS in
    `CoinReferencesSection.test.ts` and `CoinActionsPanel.test.ts`.
11. Draft-list `Numista #<identifier>` chip and no-selection omission — PASS
    in `QuickCaptureDraftCard.test.ts` and draft-list tests.
12. `Analyze Photos`, `Save as Draft`, no navigation destination, keyboard
    disclosure/focus, and 375 px containment — PASS through Vue interaction
    and structural responsive assertions.

Residual: no interactive physical-browser or live-provider walkthrough was
performed. The repository has no Feature 341 Playwright scenario or committed
development credential/stub-server launcher; all provider-independent
workflow requirements above are covered by deterministic tests.

### Security scans

- Installed repository workflow-equivalent local tools through existing
  Windows package support:
  `winget install --id Gitleaks.Gitleaks --exact ...` (8.30.1) and
  `winget install --id AquaSecurity.Trivy --exact ...` (0.73.0).
- `gitleaks git . --config .gitleaks.toml --redact --no-banner` — PASS,
  1,083 commits, no leaks.
- `gitleaks dir . --config .gitleaks.toml --redact --no-banner` — PASS,
  no leaks, including uncommitted integration tests.
- `trivy image --scanners vuln,secret --severity HIGH,CRITICAL --exit-code 1 --no-progress bjd145/ancient-coins:beta`
  — PASS. Alpine packages: 0 High/Critical; Go binary: 0 High/Critical.
  Docker/Podman was unavailable locally, so Trivy scanned the published beta
  production image rather than rebuilding the Dockerfile on this workstation.

### Release record corrections and immutable exceptions

- Merge commit `a8f59b3bf7e2479e1083ee21f0737369c89c3a91` inaccurately
  described T087–T096 as “Quick Capture promotion.” The authoritative task
  list identifies them as Phase 5A / User Story 6 canonical catalog-reference
  placement and contextual lookup UX. Transactional Quick Capture promotion
  belongs to User Story 2, principally T030–T031 and T038–T040.
- Public history is not rewritten. The following matrix was verified with
  `git show -s --format=fuller`, Git's `%(trailers:...)` formatter, and
  `git interpret-trailers --parse`. `Copilot-Session` is repository metadata,
  not a Constitution requirement; its absence is recorded but is not a gate.

| Commit(s) | Subject prefix | Required Copilot co-author | Optional session |
| --- | --- | --- | --- |
| `460dbfc` | `docs:` compliant | **Not parseable**: the message contains `- Co-authored-by: ...` as a list item, not a Git trailer | Absent |
| `31cb603` | **Nonconventional `scribe:`** | Present and parseable | Absent |
| `6e274c2`, `ec54469`, `395cdba`, `a8f4fdd`, `dd694dc`, `c03b860`, `83bfdd1`, `57f6f6d`, `dc40277`, `3972dc2`, `56b6abe`, `a793b04`, `a19bf4a`, `503d309` | Compliant allowed prefixes | Present and parseable | Present and parseable |
| `97bba2e`, `6726b09`, `011c635` | Compliant allowed prefixes | Present and parseable | Absent |
| `8e77500` | **Nonconventional `merge:`** | Absent | Absent |
| `a8f59b3` | **Nonconventional `merge:`** | Present and parseable | Present and parseable |

- The immutable exceptions are therefore three nonconventional subjects
  (`31cb603`, `8e77500`, `a8f59b3`) plus the unparseable required co-author
  trailer on `460dbfc`. No amend, rebase, reset, or force-push is permitted on
  public `beta`.
- [ADR 0008: Feature 341 Immutable Public-History Waiver](../../docs/adr/0008-feature-341-immutable-public-history-waiver.md)
  is the accepted §22 authority for this exact matrix. Brian explicitly
  approved the recommended ADR on 2026-08-11.
- T086 is **complete**: the accepted waiver, exact exception matrix, full
  quality-gate and workflow evidence above, and prospective PR/release
  enforcement satisfy its verification acceptance. The future PR must
  disclose the ADR and all four immutable exceptions. This task state does
  not claim Maximus final clearance; his reviewer block remains until he
  explicitly re-reviews and clears it.

### Phase 8 task reconciliation

| Tasks | State | Evidence |
| --- | --- | --- |
| T073–T076 | Complete | Four deterministic integration suites under `src/api/integration/`; SC-002 is 24 fixtures × 6 candidates × 3 permutations, 24/24 top-three at an 85% threshold, with no exact-ID leakage; focused and full Go gates pass. |
| T077 | Complete | Canonical `task openapi` is byte-stable and the route-drift test passes. |
| T078–T080 | Complete | Commit `011c635` reconciles feature/Quick Capture docs, API/deployment rollout, and Nygard ADR 0007. |
| T081–T083 | Complete | Full Go, frontend (689 Vitest), and Python (189 pytest) gates pass; the Python 3.14.3 local-tooling variance is recorded above. |
| T084 | Complete | Taskfile/OpenAPI checks and all 12 provider-independent walkthrough contracts pass deterministically; the physical-browser/live-provider residual is explicit above. |
| T085 | Complete | Working-tree/history Gitleaks scans pass with path-and-placeholder-scoped allowances; published beta Trivy scan reports 0 High/Critical. |
| T086 | Complete | History audit identifies three immutable nonconventional subjects and one malformed required co-author trailer. Accepted ADR 0008 authorizes only those exceptions; the matrix is transparent and future commit/PR enforcement remains mandatory. Maximus final clearance is still pending as a separate reviewer step. |
