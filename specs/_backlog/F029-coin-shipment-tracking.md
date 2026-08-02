---
id: F029
title: "Track coin shipments from purchase/auction win through delivery"
status: promoted
priority: P2
effort: XL
value: 4
risk: 4
owner: unassigned
created: 2026-08-02
updated: 2026-08-02
---

# F029 — Track coin shipments from purchase/auction win through delivery

**Promoted to**: specs/340-coin-shipment-tracking/
**GitHub issue**: #577

## Summary

When a collector moves a coin from wishlist to purchased, or converts a won auction
lot into an owned coin, there is today no record of how the coin physically gets to
them — carrier, tracking number, transit status, delivery confirmation. Done means a
collector can attach carrier + tracking number to a coin (optionally right after
purchase/auction-conversion, or any time later), see live shipment status without
leaving the app, and get notified on key status changes (out for delivery,
delivered, delivery exception), for USPS, UPS, and FedEx at MVP — via **direct
integrations with each carrier's own API**, not a third-party aggregator.

## Acceptance criteria

- [ ] User can add a shipment (carrier, tracking number, optional notes) to a coin,
      either from the post-purchase/post-auction-conversion flow or from the coin
      detail page at any later time. All fields optional/skippable, matching the
      existing `PurchaseModal.vue` UX convention.
- [ ] Coin detail page shows a "Shipping" section with carrier, tracking number
      (linking to the carrier's tracking page), current status, and a chronological
      scan-event timeline.
- [ ] Shipment status auto-updates via **direct** USPS, UPS, and FedEx API
      integrations — each carrier authenticated/configured separately (no
      third-party tracking aggregator, and not Parcel.app, which has no public
      server-to-server API and is not viable as a backend integration).
- [ ] User receives an in-app notification (and Pushover push, if enabled) when a
      shipment status changes to out-for-delivery, delivered, or an exception/return.
- [ ] Manual status entry/override remains possible for any carrier, including when
      that carrier's API credentials aren't configured yet (e.g. still waiting on
      production API approval).

## Constitution alignment

- Principle I (Clear Layered Architecture) — new `Shipment`/`ShipmentEvent` models,
  repository, service, and handler trio; one `CarrierClient` interface with three
  concrete implementations (USPS/UPS/FedEx) behind a shared `ShipmentService`;
  shipment creation stays decoupled from `CoinService.PurchaseCoin` /
  `AuctionLotService.ConvertToCoin` rather than being bolted into those
  transactions.
- Principle V (Security, Auth, and Privacy by Default) — shipment data is
  user-scoped; each carrier's OAuth/API credentials stored as admin-configured
  `AppSetting`s (not per-user), separately for sandbox vs. production.
- §17 Quality Gate, §21 Definition of Done.

## Plan outline

1. Data model: `Shipment` (carrier, tracking number/URL, status, dates) +
   `ShipmentEvent` (scan history) in `src/api/models/`, added to `AutoMigrate` in
   `src/api/database/database.go`.
2. `CarrierClient` interface + three concrete adapters — `USPSClient`, `UPSClient`,
   `FedExClient` — each wrapping that carrier's own OAuth2/API-key flow and
   normalizing its response shape to the shared `ShipmentStatus` enum, following the
   existing HTTP-client/service patterns in `services/scraper_transport.go` and
   `services/pushover_service.go`. Credentials per carrier via
   `SettingsService`/`AppSetting`.
3. `ShipmentTrackingScheduler` polling loop (primary sync mechanism at MVP — carrier
   webhook support and setup requirements differ enough across USPS/UPS/FedEx that
   webhooks are deferred past MVP) with run log + admin manual trigger, built
   exactly per `.squad/skills/adding-scheduled-job/SKILL.md`.
4. `NotificationService.NotifyShipmentStatusChanged` alongside the existing
   `Notify*` methods in `services/notification_service.go`.
5. Protected CRUD routes/handlers for shipments; coin-detail frontend section
   (`CoinDetailShippingPage.vue` on the existing `CoinDetailSectionPageShell.vue`
   pattern) plus an optional "add tracking" step hooked into `PurchaseModal.vue` and
   the auction-lot-convert flow.
6. Tests: repository/service tests on in-memory SQLite; fixture-based tests per
   carrier client per `.squad/skills/external-service-scraping-with-fixtures/SKILL.md`
   (no live calls in CI); frontend Vitest component tests; a Playwright smoke test
   for purchase → add tracking → see status.

## Effort estimate (from planning spike, revised for direct carrier APIs)

Roughly **XL** (~3–3.5 weeks engineering, one engineer), broken into four phases:

1. Data model + manual CRUD (carrier deep-links only, no external API) — M, 3–5 days
2. Direct carrier integrations: USPS + UPS + FedEx clients, per-carrier credential
   settings, status normalization, fixture tests for all three — **L, 6–9 days**
   (this phase roughly doubled vs. the aggregator approach, since it's three
   separate auth flows, response schemas, and rate-limit strategies instead of one)
3. Polling scheduler (sole sync mechanism at MVP) + notifications — S/M, 2–3 days
4. Frontend polish + e2e coverage — S, 1–2 days

Phase 1 is shippable on its own (manual tracking-number entry with carrier deep
links, zero external dependency). **Calendar-time risk beyond engineering effort**:
UPS and FedEx both require a developer-portal application + approval for production
API access (USPS is typically self-serve); that approval lead time is outside
engineering's control and should be kicked off as early as possible, in parallel
with Phase 1.

## Open questions

- [ ] Confirm each carrier's specific API product/plan and terms of service permit
      this use case (personal tracking display, no resale of tracking data) before
      registering developer accounts.
- [ ] Should a shipment link directly to an `AuctionLot` (for provenance), or only
      to the resulting `Coin`?
- [ ] Is one-shipment-per-coin enough for the MVP UI, or must split
      shipments/returns/reships be supported from day one? (The data model already
      allows multiple `Shipment` rows per coin; this is a UI-scope question only.)
- [ ] Overlap with **F020 (Lifecycle Timeline)**: that backlog card's "acquisition"
      event type could eventually subsume or link to shipment records. Whichever of
      F020/F029 is promoted first should account for the other so they don't diverge.

## Notes

Full planning-spike design (data model field lists, trigger-point UX, API surface,
frontend component plan) was produced 2026-08-02; see
`specs/340-coin-shipment-tracking/spec.md` for the promoted spec, which supersedes
this card as the source of truth going forward.

Carrier-sourcing decision (revised 2026-08-02): direct USPS/UPS/FedEx API
integrations, not a tracking aggregator — the original spike recommended an
aggregator (EasyPost/AfterShip) for lower integration effort, but the requester
opted for direct carrier APIs instead. Parcel.app remains ruled out (no public
server-to-server API).

## History

- 2026-08-02: created (status: triaged) from a planning spike; carrier-sourcing
  approach (aggregator API) initially recommended.
- 2026-08-02: carrier-sourcing decision revised to direct USPS/UPS/FedEx APIs per
  requester; promoted (status: promoted) to `specs/340-coin-shipment-tracking/`;
  linked GitHub issue #577.
