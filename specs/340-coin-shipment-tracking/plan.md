# Implementation Plan: Coin Shipment Tracking (Direct Carrier APIs)

**Branch**: `claude/coin-shipment-tracking-n24fhx` | **Date**: 2026-08-02 | **Spec**: `specs/340-coin-shipment-tracking/spec.md`
**Input**: Feature specification from `specs/340-coin-shipment-tracking/spec.md`

## Summary

Add shipment tracking to the collection app so a collector can attach a carrier and
tracking number to a coin — optionally right after a wishlist-to-purchase
transition or an auction-lot-to-coin conversion, or any time later — and see status
update automatically without leaving the app. Carrier status is sourced through
**direct integrations with each carrier's own API** (USPS, UPS, FedEx), each behind
its own credential flow, normalized to one shared status vocabulary by a common
`CarrierClient` interface in the Go API's service layer. No third-party tracking
aggregator is used. MVP scope is manual + automatic CRUD, a polling scheduler as the
sole sync mechanism (carrier-native webhooks deferred), in-app/Pushover
notifications on meaningful status transitions, and admin-visible run history.
Carrier-native webhooks, multi-shipment UI, and return/reship workflows are
deferred past v1.

## Technical Context

**Language/Version**: Go 1.26 API (`src/api/`), Vue 3 + TypeScript + Pinia + Vite PWA (`src/web/`) — no Python agent involvement, this feature has no AI/LLM component
**Primary Dependencies**: Gin, GORM, pure-Go SQLite driver, axios API client; new: three carrier-specific HTTP/OAuth2 clients (USPS APIs, UPS Tracking API, FedEx Track API)
**Storage**: SQLite via GORM AutoMigrate; new Go-owned tables `Shipment`, `ShipmentEvent`, `ShipmentTrackingRun`; per-carrier credentials stored as existing `AppSetting` rows (no new credential table)
**Testing**: Go `go test -v ./...` and `go vet ./...` (repository/service tests on in-memory SQLite; fixture-based tests per carrier client per `.squad/skills/external-service-scraping-with-fixtures/SKILL.md`, no live carrier calls in CI); Vue `npm run build` / strict `vue-tsc --build`; Vitest component tests; Playwright smoke test
**Target Platform**: Self-hosted web service + browser/PWA app, existing Docker/Taskfile topology
**Project Type**: Two-service web application slice (Go REST API + Vue SPA/PWA); no agent service changes
**Performance Goals**: Scheduled tracking-check cycle completes for all non-terminal shipments without one carrier's failure/rate-limit blocking another carrier's shipments in the same run; shipment CRUD endpoints respond within existing coin-endpoint latency norms
**Constraints**: Preserve layered architecture (Handler → Service → Repository → Database); no aggregator/third-party tracking service; each carrier's credentials independently optional (feature must degrade to manual-only per unconfigured carrier, not fail); no invented status/event data; shipment data owner-scoped; shipment creation MUST NOT be coupled into `CoinService.PurchaseCoin` / `AuctionLotService.ConvertToCoin` transactions
**Scale/Scope**: Single-collector/self-hosted usage; v1 assumes low shipment volume (well under any carrier API's free/base rate limits); one shipment per coin is sufficient for v1 UI though the data model supports more

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Evidence / Plan |
|------|--------|-----------------|
| I. Clear Layered Architecture | PASS | `Shipment`/`ShipmentEvent`/`ShipmentTrackingRun` models import only stdlib; all GORM queries live in a new `repository/shipment_repository.go`; `CarrierClient` implementations and status-normalization/notification logic live in `services/`; handlers stay thin CRUD adapters with Swagger annotations. Shipment CRUD and status updates use transactions where multi-row (e.g. status change + event append). |
| II. Service Boundary Separation | PASS | No Python agent involvement — this feature is entirely Go API + Vue SPA. Vue calls only `/api/*` via `src/web/src/api/client.ts`. |
| III. Strict Types and Explicit Contracts | PASS | New Go structs, TypeScript interfaces (`Shipment`, `ShipmentEvent`), and Swagger annotations on new handlers. Each `CarrierClient` implementation maps its carrier's response shape into the shared typed `ShipmentStatus` enum — no untyped passthrough of carrier JSON to the frontend. |
| IV. Simple Complete Changes | PASS | v1 is proportional: manual CRUD + direct-API polling + notifications + admin run visibility. Carrier-native webhooks, multi-shipment UI, and return/reship workflows are explicitly deferred (see spec Assumptions). |
| V. Security/Auth/Privacy | PASS | All shipment endpoints require auth and are scoped to the owning collector via coin ownership. Carrier API credentials are admin-only `AppSetting`s, never returned in shipment API responses. Internal carrier-API errors are logged server-side only; clients get generic messages. |
| VI. Consistent UX | PASS | New `CoinDetailShippingPage.vue` follows the existing `CoinDetailSectionPageShell.vue` pattern used by Journal/Valuation/Notes; optional "add tracking" step in `PurchaseModal.vue` matches its existing optional-field convention; dark theme, design tokens, `lucide-vue-next` icons, PWA-compatible. |
| VII/IX/§17 Quality Gate | PASS | Plan requires Go unit/repository/service tests, fixture-based tests per carrier client (no live external calls in CI), frontend component tests, and a Playwright smoke test for purchase → add tracking → see status. |
| VIII/§19 Documentation | PASS | This plan, `research.md`, and `data-model.md` are generated in the feature directory. An ADR is warranted for the "new third-party services" trigger (Constitution §VIII) since this introduces three new external API dependencies (USPS, UPS, FedEx) — see Complexity Tracking. |

**Initial Gate Result**: PASS — no unjustified constitutional violations. One ADR is
recommended (not a violation) because the feature adds new third-party service
dependencies, per §VIII.

## Project Structure

### Documentation (this feature)

```text
specs/340-coin-shipment-tracking/
├── spec.md
├── plan.md               # this file
├── research.md            # Phase 0 output
├── data-model.md          # Phase 1 output
└── tasks.md               # Future /speckit.tasks output; not created by this plan
```

### Source Code (repository root)

```text
src/api/
├── models/
│   └── shipment.go                       # Shipment, ShipmentEvent, ShipmentCarrier, ShipmentStatus, ShipmentTrackingRun
├── repository/
│   └── shipment_repository.go            # all GORM queries; owner-scoped via existing coin-ownership scopes
├── services/
│   ├── carrier_client.go                 # CarrierClient interface + shared ShipmentStatus normalization helpers
│   ├── usps_carrier_client.go            # USPS Tracking API adapter (OAuth2 client credentials)
│   ├── ups_carrier_client.go             # UPS Tracking API adapter (OAuth2 client credentials)
│   ├── fedex_carrier_client.go           # FedEx Track API adapter (OAuth2 client credentials)
│   ├── shipment_service.go               # CRUD, status-change application, carrier dispatch
│   ├── shipment_tracking_scheduler.go    # polling loop + run log, per .squad/skills/adding-scheduled-job/SKILL.md
│   ├── settings_service.go               # add per-carrier credential + enabled settings constants (existing file)
│   └── notification_service.go           # add NotifyShipmentStatusChanged (existing file)
├── handlers/
│   ├── shipments.go                      # protected CRUD handlers for /coins/:id/shipments, /shipments/:id
│   └── shipment_tracking_admin.go        # admin run-history + manual-trigger handlers
├── database/
│   └── database.go                       # AutoMigrate Shipment, ShipmentEvent, ShipmentTrackingRun (existing file)
└── main.go                               # wire repo/service/scheduler/handlers/routes (existing file)

src/web/
├── src/api/client.ts                     # createShipment/getShipments/updateShipment/deleteShipment (existing file)
├── src/types/index.ts                    # Shipment, ShipmentEvent, ShipmentCarrier, ShipmentStatus types (existing file)
├── src/pages/CoinDetailShippingPage.vue  # new coin-detail section, on CoinDetailSectionPageShell.vue
├── src/components/coin/ShipmentTracker.vue  # carrier badge, status pill, tracking link, event timeline
├── src/components/PurchaseModal.vue      # extend with optional post-purchase "add tracking" step (existing file)
└── src/router/index.ts                   # coin-detail child route for the new Shipping tab (existing file)
```

**Structure Decision**: This is a Go API + Vue SPA feature only; no Python agent
changes. New Go work follows the existing model/repository/service/handler layering
used by `AuctionLot`/`AuctionLotService` (closest existing analogue: an external
status concept normalized into an internal enum, tied to a `Coin`). New scheduler
work follows the `.squad/skills/adding-scheduled-job/SKILL.md` recipe exactly, as
used by `AvailabilityScheduler`/`ValuationScheduler`/`AuctionEndingScheduler`. New
frontend work reuses the existing coin-detail section-page shell used by
Journal/Valuation/Notes tabs.

## Existing-Code Findings

- `CoinService.PurchaseCoin` (`src/api/services/coin_service.go:371`) and
  `AuctionLotService.ConvertToCoin` (`src/api/services/auction_lot_service.go:86`)
  are the two acquisition-event entry points this feature hooks into at the UI
  layer only — neither transaction is modified.
- `AuctionLot` (`src/api/models/auction_lot.go`) is the closest existing pattern for
  an externally-sourced status normalized into an internal enum
  (`AuctionLotStatus`) with a `StatusSource` field distinguishing sync vs. manual —
  reused directly for `ShipmentStatus`/`StatusSource`.
- `PushoverService` (`src/api/services/pushover_service.go`) and
  `NumisBids`/`CNG` scraper services (`src/api/services/scraper_transport.go`) are
  the existing patterns for a small HTTP-client-wrapping service struct reading
  credentials from `SettingsService` — reused for each `CarrierClient`
  implementation.
- `.squad/skills/adding-scheduled-job/SKILL.md` gives the exact scaffolding for a
  new scheduler with run log + admin manual-trigger endpoint, already used by
  `AvailabilityScheduler`, `ValuationScheduler`, and `AuctionEndingScheduler` — the
  `ShipmentTrackingScheduler` follows it directly, including the "manual trigger +
  run log" extension section.
- `.squad/skills/external-service-scraping-with-fixtures/SKILL.md` gives the
  fixture-based testing pattern already used for `numisbids_service.go` /
  `cng_auction_service.go` — reused for all three carrier clients so CI never makes
  live USPS/UPS/FedEx calls.
- `NotificationService` (`src/api/services/notification_service.go`) has an
  established `Notify*` method shape (in-app row + best-effort async Pushover push)
  — `NotifyShipmentStatusChanged` follows the same shape as
  `NotifyAuctionPriceAlert`.
- `CoinDetailSectionPageShell.vue` and its siblings
  (`CoinDetailJournalPage.vue`, `CoinDetailValuationPage.vue`, etc.) are the
  existing coin-detail tab pattern — `CoinDetailShippingPage.vue` follows it
  directly.
- `PurchaseModal.vue` establishes the "all fields optional, skippable" UX
  convention this feature's post-purchase tracking step must match.
- No existing per-carrier or multi-provider HTTP-client abstraction exists yet in
  the codebase; `CarrierClient` is a new interface, not a reuse of `AuctionSource`
  (which is a string enum on one model, not a service-layer interface) — modeled
  instead on the interface-of-adapters shape without an existing direct analogue.

## Phase 0: Research

Completed in `research.md`. Key decisions:
- Three independent `CarrierClient` implementations (USPS/UPS/FedEx) behind one
  interface, each owning its own OAuth2/API-key flow and response normalization.
- Polling scheduler is the sole sync mechanism at MVP; carrier-native webhooks
  deferred per carrier as a fast-follow.
- Per-carrier credentials stored as `AppSetting` rows, admin-configured, each
  carrier independently optional.
- Shipment creation stays decoupled from `PurchaseCoin`/`ConvertToCoin`; hooked in
  only at the UI layer as an optional follow-on step.
- Notification triggers limited to out-for-delivery/delivered/exception to avoid
  noise on intermediate transit-status changes.

## Phase 1: Design & Contracts

Completed artifacts:
- `data-model.md`

Post-design constitution re-check:

| Gate | Status | Notes |
|------|--------|-------|
| Layered ownership | PASS | `CarrierClient` adapters and status/notification decisions live in services; `ShipmentRepository` owns all queries; handlers stay thin. |
| Security/privacy | PASS | Credential storage pattern (`AppSetting`, admin-only) and owner-scoping are explicit in `data-model.md`. |
| Scope proportionality | PASS | Deferred webhook/multi-shipment/reship scope is documented in the spec's Assumptions and carried through unchanged. |

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Item | Why Needed | Simpler Alternative Rejected Because |
|------|------------|---------------------------------------|
| Three separate `CarrierClient` implementations instead of one aggregator client | Explicit product decision: no third-party tracking aggregator | A single aggregator integration (EasyPost/AfterShip) was the original spike recommendation and is materially less integration work, but was rejected per requester's explicit preference for direct carrier APIs |
| New third-party service dependencies (USPS, UPS, FedEx developer APIs) trigger Constitution §VIII "new third-party services" ADR requirement | Not a violation, but should be logged as an ADR before/alongside implementation, covering credential handling, rate-limit posture, and graceful degradation when a carrier is unconfigured | N/A — this is a documentation step, not an architectural exception |

## Stop Point

Planning stops after Phase 1 artifact generation. Implementation tasks are
intentionally not generated here; run `/speckit.tasks` after this plan is accepted.
