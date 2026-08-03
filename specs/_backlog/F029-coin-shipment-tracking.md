---
id: F029
title: "Track coin shipments via direct USPS/UPS/FedEx APIs"
status: promoted
priority: P1
effort: L
value: 5
risk: 4
owner: Maximus
created: 2026-08-03
updated: 2026-08-03
---

# F029 — Track coin shipments via direct USPS/UPS/FedEx APIs

**Promoted to**: specs\340-coin-shipment-tracking\

## Summary

When a coin is purchased or converted from a won auction lot, collectors currently track shipping outside the app. This feature adds first-party shipment tracking tied to coins, including carrier + tracking details, in-app status/timeline, direct carrier API sync (USPS/UPS/FedEx), and transition notifications while preserving manual entry/override when carrier credentials are unavailable.

## Acceptance criteria

- [ ] Collector can add shipment details (carrier, tracking number, optional notes) from post-purchase and post-auction-conversion flows, and from coin detail later.
- [ ] Coin detail displays shipment carrier, tracking number link, current status, and chronological scan-event timeline.
- [ ] Shipment status auto-sync supports direct USPS, UPS, and FedEx APIs with separate admin-configured credentials.
- [ ] Collector receives in-app notifications (and optional Pushover notifications) for out-for-delivery, delivered, and exception/return transitions.
- [ ] Manual status entry/override remains available for all carriers, including when API credentials are not configured.

## Constitution alignment

- Principle I (Clear Layered Architecture) — shipment persistence/integration remains Handler → Service → Repository.
- Principle IV (Simple, Complete, Proportional) — direct carrier integrations behind a shared internal contract without over-coupling purchase flows.
- Principle V (Security, Auth, and Privacy by Default) — carrier credentials are admin-only settings; shipment data is strictly user-scoped.
- Principle VI (Consistent User Experience) — coin-level shipment UI follows existing coin detail patterns in desktop and PWA modes.
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [ ] Polling cadence and retry policy per carrier (single global default vs per-carrier controls).
- [ ] Retention window for historical scan events and whether old events are pruned.
- [ ] Whether shipment timeline should surface raw carrier event text in addition to normalized status labels.

## Notes

- This card intentionally uses direct carrier APIs (USPS/UPS/FedEx), not a third-party aggregator.
- Existing scheduler + notification patterns in `src/api/services` should be reused for polling and transition notifications.

## History

- 2026-08-03: promoted to `specs\340-coin-shipment-tracking\`.
- 2026-08-03: created (status: triaged).
