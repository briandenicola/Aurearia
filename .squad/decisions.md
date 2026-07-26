# Squad Decisions

## Active Decisions

### Decision: Sets Type Refinement + Goal Completion Formula

**Date:** 2026-07-26  
**Agent:** Cassius  
**Status:** IMPLEMENTED

## Context
Set semantics were refined: legacy `defined` is removed, `open` is renamed to `standard`, and Goal completion no longer uses target matching.

## Decision
- Normalize legacy set type values in DB migration logic:
  - `defined` -> `goal`
  - `open` -> `standard`
  - legacy `dynamic` -> `tracker` with `creation_mode='dynamic'`
- Add `creation_mode` on `coin_sets` (`manual` default, `dynamic` allowed only for `tracker` sets).
- Update Goal completion to: `collection_items / (collection_items + wishlist_items)` using set memberships + `coins.is_wishlist`.
- Keep tag-to-set migration idempotent and additive so newly tagged coins join existing migrated sets.

## Validation
- `go test ./repository -run "TestSetRepository_"`
- `go test ./services -run "TestSetService_CreateSet"`
- `go test ./database -run "TestMigrateCoinSetTypes_NormalizesLegacyValues"`
- `go test ./handlers -run "TestSetHandler_ReorderCoins"`
- `go test ./testutil`

---

### Decision: Frontend set-type normalization during Standard/Goal migration

**Date:** 2026-07-26  
**Agent:** Aurelia  
**Status:** IMPLEMENTED

## Context
Set APIs are moving from legacy `open`/`defined` to `standard`/`goal`, with Tracker and Dynamic Tracker creation mode added. During rollout, frontend may receive mixed legacy/new set type values.

## Decision
Frontend now writes only `standard`, `goal`, `smart`, and `tracker`, while treating `open` and `defined` as legacy read aliases through a shared `normalizeCoinSetType()` helper in `src/web/src/types/index.ts`.

Membership and workflow gates branch on normalized values:
- Tracker and Smart sets are non-manual membership in Set Detail and Coin Tags surfaces.
- Completion panel loads for normalized `goal` and `tracker`.
- Collection set filter includes normalized `standard` sets.

## Rationale
This keeps UI behavior stable during mixed-contract deployments, prevents accidental legacy writes, and centralizes compatibility logic to avoid drift between components.

---

### Decision: Tracker set creation mode contract alignment

**Date:** 2026-07-26  
**Agent:** Maximus  
**Status:** IMPLEMENTED

## Context
`SetCreationWizard` emitted `trackerCreationMode`, but backend set creation reads `creationMode`. Tracker sets created through the dynamic flow therefore persisted with backend default `manual` mode.

## Decision
Align frontend payload contract to backend by emitting `creationMode` in `CreateCoinSetRequest` and wizard submit payload. Keep prompt behavior unchanged, and add a focused frontend regression test asserting dynamic tracker submission emits `creationMode: "dynamic"` and not `trackerCreationMode`.

## Validation
- `npm.cmd run test -- src/components/sets/__tests__/SetCreationWizard.test.ts`
- `npm.cmd run type-check`

## Alignment
- Principle III: explicit typed frontend/backend contract field match
- Principle IV: smallest complete fix with targeted regression coverage

---

### Decision: Coin Grading as AI Analysis Sub-Action

**Date:** 2026-07-02
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context
Coin grading is an AI-assisted per-coin workflow that returns an estimate report without mutating the saved coin grade.

## Decision
The user-facing grading entry point lives inside `CoinAIAnalysis.vue` beside obverse/reverse analysis instead of chat or a new store. It uses the existing async AI job polling pattern, requires both obverse and reverse images before start, displays `gradingReport` in-place, and includes permanent limitation copy that the estimate is not professional certification and does not update `Coin.grade` automatically.

## Alignment
- Principle III: typed API contract and frontend type-check passed.
- Principle IV: reused existing AI job polling surface without a new store.
- Principle VI: token-based in-panel UI with no emoji and existing button hierarchy.

---

### Decision: Coin Grading ships as a dedicated coin-detail AI job

**Date:** 2026-07-01
**Agent:** Maximus
**Status:** IMPLEMENTED
**Issue:** #374

## Context

Coin grading exists in `src/agent/app/teams/coin_grading.py` and the supervisor router advertises a `grading` intent, but the streaming chat path does not carry images. Routing grading requests through chat currently lands on a passthrough/dead-end unless a caller injects a grading node.

## Decision

Implement Coin Grading as a dedicated authenticated coin-detail action backed by the existing AI job system, not as image-capable chat attachments for this slice.

Recommended surface:
- Add a `Grade Coin` action on the coin detail AI Analysis page, using existing stored obverse/reverse/detail images.
- Queue `AIJobTypeCoinGrading` through Go, pass owner-scoped coin context and image bytes to Python `/api/grade`, then store the report in the job result.
- Do not write the estimated grade into `Coin.Grade` automatically. Offer an explicit `Apply to Grade` follow-up only after the user reviews the confidence/limitations.
- Remove supervisor grading advertising/dead-end behavior unless/until chat supports image attachments.

## Rationale

This keeps the feature aligned with Constitution Principle I/II: Vue calls Go, Go owns auth/image access/job persistence, Python remains stateless and receives only bounded per-request context. Reusing AI jobs matches the existing analysis/value UX, avoids introducing image-capable chat contracts prematurely, and provides clear regression points for no-image, success, and model-failure paths.

## Validation expected

- Agent: request model and `/api/grade` route tests for no-image, success, and graph failure.
- Go: service/job tests for no images, successful grading result persistence, and agent failure status.
- Frontend: component tests for disabled/no-image state, queued/success polling, failure display, and confidence/limitations copy.
- Contract: regenerate OpenAPI and run route drift test for any new Go route.

---

### Decision: Coin Grading Revision

**Date:** 2026-07-02
**Agent:** Maximus
**Status:** IMPLEMENTED

Coin grading remains exposed through the dedicated `/api/grade` agent endpoint and Go `POST /coins/:id/grade` AI job workflow only. Collection chat no longer advertises or routes to a `grading` supervisor capability until that path has a real wired implementation instead of a passthrough dead-end.

---

### Decision: Coin Grading Backend Contract

**Date:** 2026-07-01
**Agent:** Cassius
**Status:** IMPLEMENTED
**Issue:** #374

## Decision

Issue #374 uses a dedicated async AI job endpoint, `POST /api/coins/:id/grade`, backed by Python `POST /api/grade`.

## Contract

- Go endpoint returns `202` with the normal `services.AIJobSubmissionResponse`.
- New job type is `coin_grading`.
- Python grading response is `{ "report": string }`.
- Completed Go job result is JSON `{ "gradingReport": string }`.
- The workflow requires at least one owner-scoped coin image; image-less coins return `400 {"error":"No image available for grading"}`.
- The saved `Coin.Grade` field is not updated automatically.

## Rationale

This preserves the existing AI job polling/notification contract, avoids chat attachment coupling, and keeps grade updates explicitly user-controlled.

---

### Decision: Auction Sync Auto-Creates In-App Calendar Events

**Date:** 2026-07-01
**Agent:** Cassius
**Status:** IMPLEMENTED

## Context

`/auctions/sync` upserts NumisBids and CNG watchlist lots, but newly tracked active lots were not linked to in-app calendar entries despite `AuctionLot.EventID` and `AuctionEvent` already existing.

## Decision

Add repository-level `UpsertWithCalendarEvent` for sync paths only. It creates an `AuctionEvent` and links `AuctionLot.EventID` in the same transaction only when the source-aware upsert inserts a new lot with status `watching` or `bidding`. Existing lots update without new events, and `passed`/`won`/`lost` lots do not auto-create events.

## Validation

- `go test -v .\repository -run "TestAuctionLotRepository_Upsert"`
- `go test -v .\handlers -run "TestAuctionLotHandlerUpdateStatus"`
- `go test ./...`

## Alignment

- Principle I: GORM and multi-step create/link live in repository transaction.
- Principle IV: Small, source-aware extension of existing upsert/sync workflow.
- §17: Targeted regression coverage plus full Go API tests pass.

---

### Decision: Cassius Scraper Transport Helper

**Date:** 2026-07-01
**Agent:** Cassius
**Status:** IMPLEMENTED

## Context

Issue #373 starts with auditing shared scraper behavior across NumisBids and CNG. Both providers need authenticated HTTP session mechanics, but their login payloads, auth verification rules, URL safety, pagination, parsing, and provider-specific sentinel errors must remain provider-owned.

## Decision

Added a package-private shared helper in `src/api/services/scraper_transport.go` for cookie-jar client creation, request/header construction, form POST construction, request execution, status checks, response body read/close behavior, and request error wrapping. The first segment intentionally does not refactor `NumisBidsService` or `CNGAuctionService` to use it yet.

## Validation

- `go test -v ./services -run "Test(NewScraper|DoScraper|ReadScraper|CNGAuctionService|CanonicalCNG|ParseWatchlist|WatchlistDiagnostics|FetchWatchlist|Login|VerifyAuthentication)"`

## Alignment

- Principle I: helper stays in service layer and remains HTTP-provider agnostic.
- Principle IV: simple, focused extraction without broad provider refactor.
- §21: new helper methods have focused regression coverage.

---

### Decision: User-Initiated Camera Start

**Date:** 2026-06-30
**Agent:** Aurelia
**Status:** IMPLEMENTED

## Context

iOS/PWA users should not see a camera permission prompt just by opening Add Coin agentic mode or Identify Coin. The app still needs to preserve the guided live camera experience once the user intentionally starts capture.

## Decision

`src/web/src/pages/AddCoinPage.vue` and `src/web/src/pages/CoinLookupPage.vue` no longer start camera streams from page mount, agentic mode entry, or Identify Coin retake. Both pages show a clear "Start Camera" placeholder action that calls `startCamera()` only from a user tap. Existing upload-library actions remain available, shutter buttons stay disabled until `cameraReady`, and Add Coin continues stopping active streams when leaving agentic mode.

## Validation

- `npm.cmd run test -- src/pages/__tests__/CoinLookupPage.test.ts src/__tests__/ui-patterns.test.ts`
- `npm.cmd run type-check`

## Alignment

- Principle III: Vue strict type-check passed.
- Principle IV: Simple complete change across both affected camera entry points.
- Principle VI: Preserves existing dark, token-based camera UI and upload fallback.

---
