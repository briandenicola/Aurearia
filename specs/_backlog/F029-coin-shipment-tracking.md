---
id: F029
title: "Track coin shipments from purchase/auction win through delivery"
status: triaged
priority: P2
effort: L
value: 4
risk: 3
owner: unassigned
created: 2026-08-02
updated: 2026-08-02
---

# F029 — Track coin shipments from purchase/auction win through delivery

## Summary

When a collector moves a coin from wishlist to purchased, or converts a won auction
lot into an owned coin, there is today no record of how the coin physically gets to
them — carrier, tracking number, transit status, delivery confirmation. Done means a
collector can attach carrier + tracking number to a coin (optionally right after
purchase/auction-conversion, or any time later), see live shipment status without
leaving the app, and get notified on key status changes (out for delivery,
delivered, delivery exception), for USPS, UPS, and FedEx at MVP.

## Acceptance criteria

- [ ] User can add a shipment (carrier, tracking number, optional notes) to a coin,
      either from the post-purchase/post-auction-conversion flow or from the coin
      detail page at any later time. All fields optional/skippable, matching the
      existing `PurchaseModal.vue` UX convention.
- [ ] Coin detail page shows a "Shipping" section with carrier, tracking number
      (linking to the carrier's tracking page), current status, and a chronological
      scan-event timeline.
- [ ] Shipment status auto-updates via a tracking aggregator API (e.g. EasyPost or
      AfterShip) covering USPS, UPS, and FedEx behind one integration — not three
      separate direct carrier integrations, and not Parcel.app (which has no public
      server-to-server API and is not viable as a backend integration).
- [ ] User receives an in-app notification (and Pushover push, if enabled) when a
      shipment status changes to out-for-delivery, delivered, or an exception/return.
- [ ] Manual status entry/override remains possible even without aggregator
      configuration (admin hasn't set an API key yet, or a carrier/tracking number
      the aggregator doesn't recognize).

## Constitution alignment

- Principle I (Clear Layered Architecture) — new `Shipment`/`ShipmentEvent` models,
  repository, service, and handler trio; shipment creation stays decoupled from
  `CoinService.PurchaseCoin` / `AuctionLotService.ConvertToCoin` rather than being
  bolted into those transactions.
- Principle V (Security, Auth, and Privacy by Default) — shipment data is
  user-scoped; the aggregator webhook receiver is unauthenticated but must verify
  the provider's signature before trusting payloads; aggregator API key stored as an
  admin-configured `AppSetting`, not a per-user credential.
- §17 Quality Gate, §21 Definition of Done.

## Plan outline

1. Data model: `Shipment` (carrier, tracking number/URL, status, provider tracker
   ID, dates) + `ShipmentEvent` (scan history) in `src/api/models/`, added to
   `AutoMigrate` in `src/api/database/database.go`.
2. `TrackingProvider` interface + one concrete aggregator adapter (EasyPost or
   AfterShip), following the existing HTTP-client/service patterns in
   `services/scraper_transport.go` and `services/pushover_service.go`; API key via
   `SettingsService`/`AppSetting`.
3. Webhook receiver (`POST /webhooks/tracking/:provider`, signature-verified) plus a
   polling-fallback `ShipmentTrackingScheduler` with run log + admin manual trigger,
   built exactly per `.squad/skills/adding-scheduled-job/SKILL.md`.
4. `NotificationService.NotifyShipmentStatusChanged` alongside the existing
   `Notify*` methods in `services/notification_service.go`.
5. Protected CRUD routes/handlers for shipments; coin-detail frontend section
   (`CoinDetailShippingPage.vue` on the existing `CoinDetailSectionPageShell.vue`
   pattern) plus an optional "add tracking" step hooked into `PurchaseModal.vue` and
   the auction-lot-convert flow.
6. Tests: repository/service tests on in-memory SQLite; fixture-based provider tests
   per `.squad/skills/external-service-scraping-with-fixtures/SKILL.md` (no live
   calls in CI); frontend Vitest component tests; a Playwright smoke test for
   purchase → add tracking → see status.

## Effort estimate (from planning spike)

Roughly **L** (~2–2.5 weeks, one engineer), broken into four independently useful
phases — see full spike write-up in Notes below:

1. Data model + manual CRUD (carrier deep-links only, no external API) — M, 3–5 days
2. Aggregator integration (live status, webhooks) — M, 3–5 days
3. Polling-fallback scheduler + notifications — S/M, 2–3 days
4. Frontend polish + e2e coverage — S, 1–2 days

Phase 1 alone is shippable on its own (manual tracking-number entry with carrier
deep links, zero external dependency) if a faster first release is preferred, with
live aggregator status added as a fast-follow.

## Open questions

- [ ] Which aggregator: EasyPost (simpler tracking-only API, generous free tier) vs.
      AfterShip (broader carrier coverage, different pricing)? Either comfortably
      covers a personal collection app's volume.
- [ ] Should a shipment link directly to an `AuctionLot` (for provenance), or only
      to the resulting `Coin`?
- [ ] Is one-shipment-per-coin enough for the MVP UI, or must split
      shipments/returns/reships be supported from day one? (The data model already
      allows multiple `Shipment` rows per coin; this is a UI-scope question only.)
- [ ] Overlap with **F020 (Lifecycle Timeline)**: that backlog card's "acquisition"
      event type could eventually subsume or link to shipment records. Whichever of
      F020/F029 is promoted first should account for the other so they don't diverge.

## Notes

Full planning-spike design (data model field lists, carrier/aggregator integration
architecture, trigger-point UX, webhook + polling design, API surface, frontend
component plan, and phased effort table) was produced 2026-08-02 and is captured in
full above; ping the requester (Brian) for the original spike conversation if more
detail is needed than fits in this card.

Carrier-sourcing decision: confirmed to use a tracking aggregator API rather than
direct USPS/UPS/FedEx integrations or Parcel.app — Parcel.app has no public API
viable for a backend integration.

## History

- 2026-08-02: created (status: triaged) from a planning spike; carrier-sourcing
  approach (aggregator API) confirmed with requester.
