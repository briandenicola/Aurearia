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
