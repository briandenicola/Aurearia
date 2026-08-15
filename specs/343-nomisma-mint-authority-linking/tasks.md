---

description: "Task list for 343-nomisma-mint-authority-linking (Phase 1 only)"
---

# Tasks: Optional Nomisma.org Authority Linking for Global Mint Locations

**Input**: Design documents from `specs/343-nomisma-mint-authority-linking/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/nomisma-authority-linking.md, quickstart.md
**Branch**: `343-nomisma-mint-authority-linking`

**Scope note**: This file covers **Phase 1 only** (Nomisma.org global-mint
authority linking), per plan.md's explicit scope note. The Phase 2 (Deferred)
OCRE/RPC catalog-authority extension recorded at the end of plan.md and in
research.md §9 is informational only — **no task below implements, tests, or
touches OCRE/RPC code**, and none may be added without a new
`/speckit.specify` cycle for that separate feature.

**Tests**: Included throughout, matching this repo's existing convention of
co-located `_test.go` / `__tests__/*.test.ts` files for every new
service/handler/component (quickstart.md §1–§2; constitution §21 items 3, 9).

**Organization**: Tasks are grouped by user story (spec.md) to enable
independent implementation and testing of each story, after a shared
Foundational phase that all four stories depend on.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on
  incomplete tasks)
- **[Story]**: US1 (P1), US2 (P2), US3 (P2), US4 (P1) — maps to spec.md's
  User Story 1–4
- All file paths are exact, existing repository paths confirmed from
  plan.md's Project Structure section, or new files plan.md explicitly names

---

## Phase 1: Setup

**Purpose**: Confirm a clean, green baseline before any Phase 1 change; no
new dependencies are required (research.md §3: stdlib `net/http` only, no
new third-party HTTP/reconciliation library).

- [X] T001 On branch `343-nomisma-mint-authority-linking`, verify the
      pre-change baseline is green: `go build ./...`, `go vet ./...`, and
      `go test ./...` from `src/api`, plus `npm run build` from `src/web`
- [X] T002 [P] Confirm no `go.mod`/`package.json` changes are required for
      this feature (research.md §3/§4: stdlib `net/http` for the Nomisma
      client, no new cache/HTTP library) — record this confirmation in the
      PR description, no file changes expected

**Checkpoint**: Baseline confirmed green; safe to begin Foundational work.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared schema, client, cache, DI wiring, and shared frontend
types that every user story (US1–US4) depends on. **No user story task may
start until this phase is complete.**

- [X] T003 [P] Add `NomismaURI *string` (`gorm:"type:varchar(256)"`,
      `json:"nomismaUri,omitempty"`), `NomismaLabel string`
      (`gorm:"type:varchar(256)"`, `json:"nomismaLabel,omitempty"`), and
      `NomismaLinkedAt *time.Time` (`json:"nomismaLinkedAt,omitempty"`)
      fields to `src/api/models/mint_location.go`, per data-model.md's
      "New fields" table (additive, nullable, absence means "not linked")
- [X] T004 Add a code comment beside the `AutoMigrate` call in
      `src/api/database/database.go` (near line 32, mirroring the existing
      additive-migration comment near line 50) documenting that the three
      new `MintLocation.Nomisma*` columns are additive-only, require no
      backfill, and introduce no destructive migration (data-model.md
      "Migration"/"Rollback" sections)
- [X] T005 [P] Create `src/api/services/nomisma_client.go`: `NomismaCandidate`
      struct (`URI`, `Label`, `Score`, `Match`), `NomismaErrorKind` typed
      constants (`unavailable`, `no_match`, `invalid_response`,
      `invalid_request`, `cancelled`), a `NomismaClient` interface
      (`Search(ctx context.Context, query string, limit int)
      ([]NomismaCandidate, NomismaErrorKind, error)`), and an
      `HTTPNomismaClient` implementation calling
      `http://nomisma.org/apis/reconcile` (OpenRefine-compatible
      reconciliation JSON per research.md §1) with a bounded request timeout
      constant (mirror `geocodeRequestTimeout` in
      `src/api/services/geocode_service.go`) and a bounded response-body
      read (mirror `numistaResponseLimit` in
      `src/api/services/numista_client.go`)
- [X] T006 [P] Create `src/api/services/nomisma_cache.go`: a bounded
      in-memory TTL cache for search responses only, keyed by a SHA-256
      digest of the normalized (trimmed, lower-cased, whitespace-collapsed)
      query text (mirrors `numista_cache.go`'s key-identity convention), a
      fixed 10-minute TTL, a 200-entry bound with LRU-by-expiry eviction;
      cache a zero-result search as a negative entry for the same TTL;
      **never** cache `unavailable`/`invalid_response` outcomes
      (research.md §4, FR-011)
- [X] T007 [P] Create `src/api/services/nomisma_client_test.go` with
      `httptest.Server` fixtures covering `ok`, `no_match`, timeout,
      connection-refused, non-2xx, malformed-JSON, and cancelled-context
      outcomes; assert no test call ever reaches the real `nomisma.org` host
      (quickstart.md §1 constraint)
- [X] T008 [P] Create `src/api/services/nomisma_cache_test.go` covering
      hit/miss/expiry, 200-entry eviction, negative-entry TTL, and
      no-caching-of-failure behavior
- [X] T009 Wire `HTTPNomismaClient`/`NomismaCache` construction into
      `src/api/main.go` beside the existing admin mint-location wiring
      (near line 704's `mintLocationRepo`/`mintLocationSvc`/
      `mintLocationHandler` construction), passing both into
      `services.NewMintLocationService(...)` (extend its constructor or add
      a `.WithNomisma(...)` option mirroring `WithGeocoding` on
      `MintLocationHandler` in `src/api/handlers/mint_location.go`)
- [X] T010 [P] Add `NomismaCandidate`, `NomismaSearchStatus`
      (`'ok' | 'no_match' | 'unavailable'`), and `NomismaSearchResponse`
      types to `src/web/src/types/index.ts`, and extend the existing
      `MintLocation` interface (line 556) with optional `nomismaUri`,
      `nomismaLabel`, `nomismaLinkedAt` fields per
      contracts/nomisma-authority-linking.md's frontend contract
- [X] T011 [P] Extend `MintReference` in `src/web/src/utils/mintMap.ts`
      (and `buildMintLookup`/`findMintReference`) so the three new optional
      Nomisma fields flow through unchanged from `MintLocation` to
      `MintReference`/`MintGroup`

**Checkpoint**: Schema, typed client, cache, DI, and shared frontend types
exist. No story-specific behavior yet. All four user stories can now begin.

---

## Phase 3: User Story 1 - Admin links a global mint to its Nomisma concept (Priority: P1) 🎯 MVP

**Goal**: An admin can search Nomisma for a global mint, review candidates,
explicitly confirm exactly one, and the mint location persists the concept
URI/label/timestamp; every surface showing that mint then displays
"Source: Nomisma.org · CC BY 4.0" linking to the concept and license.

**Independent Test**: Per spec.md User Story 1 — open a global mint's
management view, trigger a Nomisma search, select a candidate, confirm, and
verify the mint now persists the URI and shows attribution.

- [X] T012 [P] [US1] Add `MintLocationService.SearchNomisma(id uint, query
      string) (NomismaSearchOutcome, error)` in
      `src/api/services/mint_location_service.go`: validate a non-blank,
      ≤200-char query (else a new `ErrMintLocationNomismaQueryInvalid`),
      resolve the target via `FindByID` reusing the existing
      `existing.UserID != nil → ErrMintLocationNotFound` guard (mirrors
      `UpdateGlobal` at lines 95-109), check the cache, else call
      `NomismaClient.Search`, cache the result, and return the
      `NomismaSearchOutcome` (`ok`/`no_match`/`unavailable`) per
      data-model.md's `NomismaSearchOutcome` table
- [X] T013 [US1] Add `MintLocationService.LinkNomismaGlobal(id uint, uri,
      label string) (*models.MintLocation, error)` in the same file:
      validate `uri` parses as an absolute `http`/`https` URL under the
      `nomisma.org` host and `label` is non-blank and ≤256 chars (else new
      `ErrMintLocationNomismaURIInvalid`/`ErrMintLocationNomismaLabelInvalid`),
      reuse the global-only guard, then set `NomismaURI`/`NomismaLabel`/
      `NomismaLinkedAt = now()` together via **one** `repo.Update` call
      touching only those three columns — never `DisplayName`, `Lat`,
      `Lng`, `Region`, or `Aliases` (contracts/nomisma-authority-linking.md
      §2, FR-005, data-model.md invariants)
- [X] T014 [P] [US1] Add table-driven tests in
      `src/api/services/mint_location_service_test.go` for `SearchNomisma`
      (ok / no_match / unavailable / invalid-query / private-mint-404) and
      `LinkNomismaGlobal` (happy path, invalid URI host, blank/too-long
      label, private-mint-404, replaces an existing link)
- [X] T015 [US1] Add `MintLocationHandler.SearchNomisma(c *gin.Context)` in
      `src/api/handlers/mint_location.go`: `GET
      /admin/mint-locations/:id/nomisma/search?query=`, Swagger annotations
      (tag `Mint Locations`, mirror the existing `Geocode` handler doc block
      at lines ~258-267), `200` response `{status, candidates}` (candidates
      always present, possibly empty) per
      contracts/nomisma-authority-linking.md §1, `400` for invalid query,
      `404` for unknown/private mint, **never** a `5xx` for an upstream
      Nomisma failure
- [X] T016 [US1] Add `MintLocationHandler.LinkNomisma(c *gin.Context)` in
      the same file: `POST /admin/mint-locations/:id/nomisma`, Swagger
      annotations, binds `{uri, label}`, `200` returns the updated
      `models.MintLocation`, `400`/`404`/`500` mapped per
      contracts/nomisma-authority-linking.md §2
- [X] T017 [P] [US1] Add handler tests in
      `src/api/handlers/mint_location_handler_test.go` for `SearchNomisma`/
      `LinkNomisma` happy paths, asserting the exact response shapes above
- [X] T018 [US1] Register `GET /admin/mint-locations/:id/nomisma/search` and
      `POST /admin/mint-locations/:id/nomisma` on the existing `admin`
      route group in `src/api/main.go`, beside the existing
      `admin.POST("/mint-locations", ...)` /
      `admin.PUT("/mint-locations/:id", ...)` registrations near line 706
- [X] T019 [P] [US1] Add `searchNomismaMintCandidates(id, query)` and
      `linkNomismaMintLocation(id, uri, label)` functions to
      `src/web/src/api/client.ts`, alongside the existing mint-location
      functions (lines 415-427), per
      contracts/nomisma-authority-linking.md's frontend contract
- [X] T020 [P] [US1] Create `src/web/src/components/mint/NomismaAttribution.vue`:
      props `uri: string`, `label: string`; renders exactly
      "Source: Nomisma.org · CC BY 4.0", with `Nomisma.org` linking to
      `uri` and `CC BY 4.0` linking to
      `https://creativecommons.org/licenses/by/4.0/`
- [X] T021 [P] [US1] Create
      `src/web/src/components/mint/__tests__/NomismaAttribution.test.ts`
      asserting the exact attribution string and both link targets verbatim
      (SC-002: "never an unattributed or generic reference")
- [X] T022 [US1] Extend `src/web/src/components/admin/AdminCoinPropertiesSection.vue`'s
      global mint list: a per-mint "Search Nomisma" trigger, a rendered
      candidate list (label, score, match — none pre-selected), and a
      single explicit "Confirm" action per candidate that calls
      `linkNomismaMintLocation`; render `NomismaAttribution` once a mint is
      linked
- [X] T023 [P] [US1] Create
      `src/web/src/components/admin/__tests__/AdminCoinPropertiesSection.test.ts`
      covering the search-trigger → candidates-render → confirm →
      attribution-shown flow, mocking `searchNomismaMintCandidates`/
      `linkNomismaMintLocation`
- [X] T024 [US1] Extend `src/web/src/components/map/MintCoinDrawer.vue` to
      render `NomismaAttribution` when `group.mint.nomismaUri` is present
- [X] T025 [P] [US1] Add assertions to
      `src/web/src/components/map/__tests__/MintCoinDrawer.test.ts`
      covering attribution rendering when a group's mint is Nomisma-linked,
      and its absence when not
- [X] T026 [US1] Manually execute quickstart.md §3 steps 1–4 (search →
      confirm → attribution renders on both the admin panel and the Mint
      Map drawer) against a local run and record the result in the PR
      description
      **RESOLVED (2026-08-14):** The contract defect from the earlier
      BLOCKED note (double-wrapped `queries` request param + `results`-
      wrapped response parsing) is fixed in
      `src/api/services/nomisma_client.go`. The live wire contract was
      verified directly against `https://nomisma.org/apis/reconcile`
      (`Invoke-WebRequest` for `Roma`/`Rome`, 2026-08-14) and is:
      - Request: the `queries` URL param's value is the query-id map
        directly — `{"q1":{"query":"Roma","limit":5}}` — no outer
        `"queries"` key.
      - Response: the query-id map at the **top level** — `{"q1":{"result":
        [{"id":"rome","name":"Rome","type":[...],"score":1,"match":
        "false"}]}}` — no `"results"` wrapper.
      - Each result's `"id"` is Nomisma's short local id (e.g. `"rome"`),
        **not** a full URI — the client expands it to
        `http://nomisma.org/id/<id>` (matching the convention documented
        for Nomisma's own `getLabel`/`getRdf` REST helpers) before
        returning it as `NomismaCandidate.URI`.
      - `"match"` is encoded as a **JSON string** (`"true"`/`"false"`), not
        a JSON boolean; the client decodes this via a small custom
        `UnmarshalJSON` and exposes it as `NomismaCandidate.Match bool`.
      Fixtures/tests (`nomisma_client_test.go`) were updated to the
      verified shape, and two regression tests were added —
      `TestHTTPNomismaClient_Search_RequestShapeMatchesLiveContract` (fails
      if the request is ever double-wrapped again) and
      `TestHTTPNomismaClient_Search_ResponseShapeMatchesLiveContract`
      (fails if response parsing reverts to expecting a `results`
      wrapper) — both offline via `httptest.Server`, no live calls in CI.
      Live happy-path re-verified end-to-end against an isolated local API
      + temp SQLite DB (deleted after the run) and the real
      `nomisma.org` HTTPS host: registered an admin user, created a global
      mint location, `GET .../nomisma/search?query=Rome` returned real
      candidates (`http://nomisma.org/id/rome`, score 1, plus four other
      real Nomisma concepts), `POST .../nomisma` linked
      `http://nomisma.org/id/rome` and a follow-up `GET
      /mint-locations` confirmed `nomismaUri`/`nomismaLabel`/
      `nomismaLinkedAt` persisted with `displayName`/`lat`/`lng`/`region`/
      `aliases` unchanged, `DELETE .../nomisma` cleared all three Nomisma
      fields (idempotent on a second call) while leaving the other fields
      untouched. Frontend attribution rendering (admin panel + Mint Map
      drawer) is verified via the existing
      `AdminCoinPropertiesSection.test.ts` / `MintCoinDrawer.test.ts`
      component tests (T023/T025) plus `NomismaAttribution.test.ts`'s
      exact-string assertion — no browser automation tool is available in
      this environment, so the rendered DOM was not visually re-driven
      live; this is the stated limitation for the UI-rendering portion of
      this task, backed by the passing component tests and the confirmed
      live API response shape those components consume.

**Checkpoint**: User Story 1 is fully functional and independently
testable/deployable as the MVP.

---

## Phase 4: User Story 2 - Admin changes or removes an existing Nomisma link (Priority: P2)

**Goal**: An admin can replace a confirmed link with a different candidate,
or explicitly unlink it, without altering the mint location's name,
coordinates, aliases, or coin associations.

**Independent Test**: Per spec.md User Story 2 — on a linked mint, re-search
and confirm a different candidate (or unlink), and verify only the Nomisma
fields change.

- [X] T027 [US2] Add `MintLocationService.UnlinkNomismaGlobal(id uint)
      (*models.MintLocation, error)` in
      `src/api/services/mint_location_service.go`: reuse the global-only
      guard, idempotently clear `NomismaURI`/`NomismaLabel`/
      `NomismaLinkedAt` via **one** `repo.Update` call touching only those
      three columns; a no-op success (not an error) if already unlinked
      (contracts/nomisma-authority-linking.md §3)
- [X] T028 [P] [US2] Add tests in
      `src/api/services/mint_location_service_test.go` for
      `UnlinkNomismaGlobal` (clears an existing link, idempotent no-op on
      an already-unlinked mint, private-mint-404) and a `LinkNomismaGlobal`
      "replace" test asserting `DisplayName`/`Lat`/`Lng`/`Region`/`Aliases`
      are byte-for-byte unchanged after replacing an existing link
- [X] T029 [US2] Add `MintLocationHandler.UnlinkNomisma(c *gin.Context)` in
      `src/api/handlers/mint_location.go`: `DELETE
      /admin/mint-locations/:id/nomisma`, Swagger annotations, `200`
      `MessageResponse{"message": "Nomisma link removed"}`, `404` for
      unknown/private mint
- [X] T030 [P] [US2] Add handler tests in
      `src/api/handlers/mint_location_handler_test.go` for `UnlinkNomisma`
      (removes an existing link, idempotent double-unlink, private-mint-404)
- [X] T031 [US2] Register `DELETE /admin/mint-locations/:id/nomisma` on the
      `admin` route group in `src/api/main.go`, beside T018's registrations
- [X] T032 [P] [US2] Add `unlinkNomismaMintLocation(id)` to
      `src/web/src/api/client.ts` per
      contracts/nomisma-authority-linking.md's frontend contract
- [X] T033 [US2] Extend `src/web/src/components/admin/AdminCoinPropertiesSection.vue`:
      on an already-linked mint, allow re-search/confirm-a-different-candidate
      (replace) and an explicit "Unlink" action with a confirmation step
      before calling `unlinkNomismaMintLocation`
- [X] T034 [P] [US2] Extend
      `src/web/src/components/admin/__tests__/AdminCoinPropertiesSection.test.ts`
      with replace-link and unlink flows, asserting the attribution
      updates/disappears and the mint's name/coordinates/aliases fields in
      the rendered form state are untouched

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Nomisma is unavailable, has no match, or match is ambiguous (Priority: P2)

**Goal**: Outage, zero-result, and ambiguous-candidate outcomes degrade
predictably; no link is ever silently created; normal mint/coin CRUD is
unaffected.

**Independent Test**: Per spec.md User Story 3 — simulate an outage/timeout,
a zero-result search, and a multi-candidate ambiguous search; confirm no
silent link, a clear status, and unaffected mint/coin CRUD in each case.

- [X] T035 [P] [US3] Add outage/no-match/ambiguous-candidate fixture tests
      to `src/api/services/nomisma_client_test.go`: network error,
      connection-refused, non-2xx, and timeout all map to `unavailable`;
      zero results map to `no_match`; multiple similarly-scored candidates
      are returned as-is (never auto-selected, `match` reflects Nomisma's
      own flag only)
- [X] T036 [P] [US3] Add tests in
      `src/api/services/mint_location_service_test.go` and
      `src/api/handlers/mint_location_handler_test.go` asserting
      `SearchNomisma` never surfaces a `5xx` for an upstream failure —
      always `200 {"status": "unavailable", "candidates": []}` — and that
      no link is persisted when the admin closes the search without
      confirming
- [X] T037 [US3] Extend `src/web/src/components/admin/AdminCoinPropertiesSection.vue`'s
      search-result UI with explicit "no match found" and "lookup
      unavailable" status messaging; no candidate is ever pre-selected on
      an ambiguous multi-candidate result — the admin must always click
      Confirm
- [X] T038 [P] [US3] Add ARIA live-region and keyboard-navigable
      candidate-list assertions, plus a mobile-viewport layout check, to
      `src/web/src/components/admin/__tests__/AdminCoinPropertiesSection.test.ts`
      for the no-match/unavailable/ambiguous states, matching this
      component's existing accessibility conventions
- [X] T039 [P] [US3] Add assertions to
      `src/web/src/components/mint/__tests__/NomismaAttribution.test.ts`
      confirming the attribution link is a real `<a href>` with a visible
      focus state (accessible/consistent with other external-source links
      in this app), not a JS-only click handler
- [X] T040 [US3] Manually execute quickstart.md §3 steps 7–8 (zero-result
      query; simulated Nomisma outage) and record the observed "no match
      found"/"lookup unavailable" behavior in the PR description
      **Verified (2026-08-14):** Zero-result — live `GET
      /admin/mint-locations/{id}/nomisma/search` against a running API
      (temp SQLite DB, temp admin user) with a real gibberish query
      returned `200 {"status":"no_match","candidates":[]}` against the
      live `nomisma.org` host; mint remained fully readable/editable
      afterward (no link created, no CRUD disruption). Outage — exercised
      at the real socket level (not mocked): `go test ./services/... -run
      TestHTTPNomismaClient_Search_(ConnectionRefused|Timeout|Non2xx)`
      pass by making genuine failing HTTP calls (closed port / hung conn /
      real 500 via `httptest`), and
      `TestMintLocationHandler_SearchNomisma_UnavailableNeverSurfaces5xx`
      confirms the handler never surfaces a 5xx for that outcome — this
      mirrors quickstart's own suggested outage technique ("point the
      client at a closed port / stub a 500"). Note: the no-match result
      above is also consistent with the request/response contract bug
      logged against T026; the no_match/unavailable *handling* itself
      (status mapping, no 5xx, mint stays usable) is independently
      verified regardless of that bug.
- [X] T041 [US3] Add a test in
      `src/api/services/mint_location_service_test.go` confirming mint/coin
      CRUD (`UpdateGlobal`, and a coin create/update referencing a mint with
      Nomisma unavailable) is unaffected when `NomismaClient.Search` errors
      — no shared code path fails (FR-015, SC-004)

**Checkpoint**: User Stories 1, 2, and 3 all work independently.

---

## Phase 6: User Story 4 - Private user mints are never sent to Nomisma (Priority: P1)

**Goal**: No Nomisma search/link/unlink code path or UI control is ever
reachable for a private (user-owned) mint location.

**Independent Test**: Per spec.md User Story 4 — confirm no Nomisma
search/link UI or API path is reachable for a private mint, by any user or
admin.

- [X] T042 [P] [US4] Add tests in
      `src/api/services/mint_location_service_test.go` asserting
      `SearchNomisma`/`LinkNomismaGlobal`/`UnlinkNomismaGlobal` all return
      `ErrMintLocationNotFound` for a private (`UserID != nil`) mint ID,
      using a fake `NomismaClient` test double that fails the test if
      `Search` is ever invoked for that ID
- [X] T043 [P] [US4] Add handler-level tests in
      `src/api/handlers/mint_location_handler_test.go` asserting
      `GET`/`POST`/`DELETE` on `.../nomisma...` all return `404` for a
      private mint ID and that the injected `NomismaClient` is never
      invoked for that request
- [X] T044 [US4] Audit
      `src/web/src/components/admin/AdminCoinPropertiesSection.vue` to
      confirm the Nomisma search/link/unlink controls added in T022/T033
      render only for entries in the **global** mint list — never for any
      private-mint editing path within the same component
- [X] T045 [P] [US4] Add or verify a test in
      `src/web/src/components/admin/__tests__/AdminCoinPropertiesSection.test.ts`
      asserting no Nomisma control renders for a private mint list item, if
      the component ever receives private mints in its props
- [X] T046 [US4] Grep-audit `src/web/src/components/**/CreateMintModal.vue`,
      the private "My Mints" list in `SettingsDataSection` (or equivalent
      settings component), and the coin form's mint picker to confirm none
      of them expose a Nomisma control (no code change is expected — these
      surfaces never receive Nomisma UI per plan.md's scope); record the
      result in the PR description
- [X] T047 [US4] Manually execute quickstart.md §4's private-mint
      authorization checks (direct API attempts against a known private
      mint ID for all three new routes) and record the `404` + no-outbound-
      Nomisma-call result in the PR description
      **Verified (2026-08-14):** Live curl/`Invoke-WebRequest` against a
      running API (temp SQLite DB, temp admin user) targeting a
      self-owned private mint id returned `404` for
      `GET .../nomisma/search`, `POST .../nomisma`, and
      `DELETE .../nomisma`. No-outbound-call confirmed two ways: (1)
      latency — the private-mint 404 returned in ~52ms vs. ~156ms for the
      equivalent real call on a global mint (real HTTPS round trip to
      `nomisma.org`), consistent with the guard short-circuiting before
      any client is invoked; (2) existing
      `TestMintLocationHandler_Nomisma_PrivateMintReturns404AndNeverCallsClient`
      uses a `fakeNomismaClient{failIfCalled: true}` that fails the test
      immediately if `Search` is ever invoked — re-run green
      (`go test ./handlers/... -run Nomisma`).

**Checkpoint**: All four user stories (US1–US4) are independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, full quality gates, final review, and commit/push
— applies across all completed stories.

- [X] T048 [P] Regenerate Swagger docs (`swag init`, or this repo's
      documented equivalent) so `src/api/docs/swagger.json` and
      `src/api/docs/swagger.yaml` include the three new admin routes; note
      in the PR that this repo has no root `openapi.yaml` to additionally
      update (only `src/api/docs/swagger.*` exist) if that remains true,
      per quickstart.md §8 and constitution §21 item 11
- [X] T049 [P] Add `docs/adr/0009-nomisma-authority-linking.md` documenting
      the `NomismaClient`/`nomisma_cache.go` boundary decision (research.md
      §3/§4), status `ACCEPTED`, following the existing format in
      `docs/adr/0007-shared-numista-lookup.md` (constitution §21 item 12,
      quickstart.md §8)
- [X] T050 [P] Verify quickstart.md §5's migration/rollback checklist: a
      fresh `AutoMigrate` against an existing SQLite database with
      pre-existing `mint_locations` rows adds the three nullable columns
      without error; rerun the existing `backfillCoinMintLocations`-covering
      tests in `src/api/database` and confirm they remain green and
      untouched (feature 338 regression)
- [X] T051 [P] Verify quickstart.md §6's observability constraint: any new
      logging around Nomisma search/link/unlink logs outcome/status/latency
      only — never raw query text, matched label, or the requesting user's
      identity beyond standard request logging (mirrors ADR 0007's rule)
- [X] T052 Run the full backend Quality Gate from `src/api`: `go build
      ./...`, `go vet ./...`, `go test -run TestArchitecture ./...`, and
      `go test ./...` — all green, with zero live-network calls to
      `nomisma.org` in any test (constitution §17, §21 items 1–3)
- [X] T053 [P] Run the full frontend Quality Gate from `src/web`: `npx
      vue-tsc --build` and `npm run build` (constitution §17, §21 item 4)
- [X] T054 [P] Run targeted frontend tests from `src/web`: `npm run test --
      NomismaAttribution AdminCoinPropertiesSection MintCoinDrawer`
      (quickstart.md §2)
- [X] T055 Review the full diff against plan.md's Project Structure list and
      confirm zero changes were made to
      `src/agent/app/tools/numismatic_authority.py` or any OCRE/RPC-named
      file — Phase 2 (plan.md's Phase 2 section, research.md §9) remains
      entirely untouched
- [X] T056 Check off all completed items in this file
      (`specs/343-nomisma-mint-authority-linking/tasks.md`, constitution
      §21 item 13) and, if any cross-cutting decision emerged during
      implementation, add an entry under `.squad/decisions/inbox/`
      (constitution §21 item 14)
- [ ] T057 Prepare a Conventional Commits-formatted commit for the feature
      branch, citing the touched Constitution Principles/sections (I, II,
      III, IV, V, VI, VIII, §17, §21) in the commit body or PR description,
      and including the `Co-authored-by: Copilot
      <223556219+Copilot@users.noreply.github.com>` trailer (this work is
      AI-assisted; constitution §17)
- [ ] T058 Push the `343-nomisma-mint-authority-linking` branch only, with a
      **non-force** push (`git push origin
      343-nomisma-mint-authority-linking`) — no `--force`/
      `--force-with-lease`, and no other branch touched

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup — **blocks all user stories**
  (schema, `NomismaClient`, `NomismaCache`, DI wiring, shared frontend types
  must exist before any story-specific handler/route/UI work).
- **User Story 1 (Phase 3, P1)**: Depends only on Foundational. This is the
  MVP.
- **User Story 2 (Phase 4, P2)**: Depends on Foundational; also depends on
  US1's `LinkNomismaGlobal` (T013) and the admin UI shell (T022) existing,
  since "replace" and "unlink" extend the same service method and component.
- **User Story 3 (Phase 5, P2)**: Depends on Foundational; extends US1's
  client/service/UI (T012, T015, T022) with additional outcome-handling
  tests and status UI — does not require US2.
- **User Story 4 (Phase 6, P1)**: Depends on Foundational; its tests exercise
  the guard already reused by T012/T013/T027, so it should run after those
  exist, but the guard logic itself requires no new production code — this
  phase is almost entirely tests/audits and can be done any time after
  Foundational + T012/T013.
- **Polish (Phase 7)**: Depends on all four user stories being complete.

### User Story Dependencies

- **US1 (P1)**: No dependency on other stories — the true MVP slice.
- **US2 (P2)**: Structurally extends US1's `LinkNomismaGlobal` and admin UI
  shell; not independently deployable before US1 exists, but independently
  *testable* per its own Independent Test criteria once US1 is in place.
- **US3 (P2)**: Extends US1's client/service/UI with additional tests and
  status messaging; independently testable via its own fixtures once US1's
  `SearchNomisma`/UI shell exist.
- **US4 (P1)**: Independently testable at any point after Foundational +
  T012/T013/T027 exist (it only adds tests/audits against already-required
  guards) — no new production code of its own.

### Critical Path

T001/T002 → T003 → T004 → T005 → T006 → T007/T008 → T009 → T010 → T011 →
**T012 → T013 → T015 → T016 → T018 → T019 → T020 → T022 → T024** (US1 MVP
path) → T027 → T029 → T031 → T033 (US2) → T052/T053 (final gates) → T057 →
T058.

The bolded segment is the minimum path to a demoable MVP (User Story 1).
US2/US3/US4 can each be built and merged as independent increments after
US1, in any order, since none of their production-code tasks depend on each
other (only on Foundational + relevant US1 pieces as noted above).

### Within Each Phase

- Tests are written alongside (not strictly before) implementation in this
  plan, matching this repo's existing table-driven-test convention; both the
  test and implementation task for a given method/handler/component must be
  green before the phase checkpoint is considered met.
- Service layer before handler layer before route registration before
  frontend client before frontend components (Handler → Service →
  Repository → Database per constitution Principle I).

---

## Parallelizable Groups

- **Foundational**: T003, T005, T006, T007, T008, T010, T011 can all run in
  parallel (different files); T004 depends on T003; T009 depends on
  T005/T006/T003.
- **US1**: T014, T017, T019, T020, T021, T023, T025 can run in parallel with
  each other once their respective non-`[P]` prerequisite (T012/T013, T015/
  T016, T018) is done.
- **US2**: T028, T030, T032, T034 can run in parallel once T027/T029/T031
  are done.
- **US3**: T035, T036, T038, T039 can run in parallel; all depend only on
  US1 existing.
- **US4**: T042, T043, T045 can run in parallel; T044/T046/T047 are
  audits/manual steps that can run alongside them.
- **Polish**: T048, T049, T050, T051, T053, T054 can all run in parallel;
  T052 (backend gate) has no file conflicts with any of them and can run
  concurrently; T055/T056/T057/T058 are sequential and must run last, in
  that order.

### Parallel Example: User Story 1

```bash
# Backend tests, once T012/T013 land:
Task: "Add table-driven tests for SearchNomisma/LinkNomismaGlobal in src/api/services/mint_location_service_test.go"
Task: "Add handler tests for SearchNomisma/LinkNomisma in src/api/handlers/mint_location_handler_test.go"

# Frontend, independent files:
Task: "Add searchNomismaMintCandidates/linkNomismaMintLocation to src/web/src/api/client.ts"
Task: "Create src/web/src/components/mint/NomismaAttribution.vue"
Task: "Create src/web/src/components/mint/__tests__/NomismaAttribution.test.ts"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational (blocks everything).
3. Complete Phase 3: User Story 1.
4. **STOP and VALIDATE**: run quickstart.md §3 steps 1–4 independently.
5. Demo/merge as the MVP increment.

### Incremental Delivery

1. Setup + Foundational → green baseline + shared plumbing ready.
2. Add User Story 1 → validate independently → merge (MVP).
3. Add User Story 4 → validate independently → merge (closes the privacy
   gap early, since it is P1 alongside US1).
4. Add User Story 2 → validate independently → merge.
5. Add User Story 3 → validate independently → merge.
6. Polish → full gates, ADR, Swagger sync, final review, commit, push.

---

## Notes

- No task in this file may create, modify, or delete any OCRE/RPC-named
  file, or generalize `NomismaClient`/`nomisma_cache.go` into a
  provider-pluggable abstraction — plan.md's Phase 2 section and
  research.md §9 are explicit that this is deliberately deferred and MUST
  NOT be anticipated here (Principle IV).
- Every new service method has ≥ 1 unit test (constitution §21 item 9);
  every new/modified public handler has Swagger annotations (constitution
  §21 item 10).
- Commit after each phase checkpoint or logical group; do not batch all 58
  tasks into a single commit if a natural per-story boundary exists.
- Avoid: vague tasks, same-file conflicts inside a `[P]` group, and
  cross-story production-code dependencies that would break a story's
  independent testability (only test/audit tasks in US4 are ordered after
  other stories' code, per the Dependencies section above).
