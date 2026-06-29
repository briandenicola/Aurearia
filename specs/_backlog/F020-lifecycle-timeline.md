---
id: F020
title: "Add lifecycle timeline and trade journal"
status: triaged
priority: P2
effort: L
value: 4
risk: 3
owner: Maximus
created: 2026-06-29
updated: 2026-06-29
---

# F020 — Add lifecycle timeline and trade journal

**GitHub issue**: #358

## Summary

Create a Lifecycle Timeline that records meaningful events in a coin's ownership journey: acquisition, attribution changes, conservation, valuation updates, moves, loans, trades, sale, and notes. Done means the collector can view and add journaled events per coin and across the collection without confusing this with the existing historical/statistical timeline.

## Acceptance criteria

- [ ] User can add, edit, and delete user-scoped lifecycle events for a coin with date, event type, notes, optional value/cost, optional source/dealer, and optional attachments or links.
- [ ] Coin detail shows a chronological lifecycle timeline and the collection-level view can filter events by type/date/coin.
- [ ] Trade/sale/acquisition events integrate with existing purchase price, current value, sold status, and storage/location workflows without silent data conflicts.
- [ ] Existing `/stats/timeline` historical/statistical timeline remains distinct in navigation and terminology.
- [ ] Lifecycle data remains private unless a future sharing spec explicitly exposes selected fields.

## Plan outline

1. Define lifecycle event taxonomy and which events synchronize with existing coin fields versus remain journal-only.
2. Add user-scoped backend lifecycle event storage and service rules for coin ownership, value fields, sold/acquired state, and attachments/links.
3. Add coin-detail timeline UI plus a collection-level Lifecycle view under an appropriate navigation parent.
4. Validate field synchronization and privacy boundaries with targeted regression tests.

## Task breakdown

- Backend: add `CoinLifecycleEvent` model in `src\api\models\`, repository methods in `src\api\repository\`, service rules in `src\api\services\`, handlers in `src\api\handlers\`, AutoMigrate in `src\api\database\database.go`, and protected routes in `src\api\main.go`.
- Frontend: add lifecycle components under `src\web\src\components\coin\` or `src\web\src\components\lifecycle\`, collection-level page under `src\web\src\pages\`, router/nav entries, API client methods, and TypeScript types.
- Testing: cover same-user coin validation, event CRUD, synchronization with purchase/sold/value fields, collection filters, and mobile timeline rendering.
- Documentation: update architecture/API docs and user workflow notes; clearly distinguish Lifecycle Timeline from Stats Timeline.

## Constitution alignment

- §0 Hierarchy of Authority — backlog card is source #6 and must promote before any schema/API work.
- Principle I (Clear Layered Architecture) — event writes and field synchronization belong in services, not handlers.
- Principle V (Security, Auth, and Privacy by Default) — lifecycle notes, prices, dealer data, and attachments are private and user-scoped.
- Principle VI (Consistent User Experience) — timeline UI must reuse existing page, card, chip, button, and mobile patterns.
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [ ] Which event types are required in v1, and are custom event types allowed?
- [ ] Should acquisition/trade/sale events update coin fields automatically, prompt the user, or remain journal-only?
- [ ] Where should the collection-level Lifecycle view live: Collection submenu, Stats submenu, or its own route?

## Notes

Frame this as "Lifecycle Timeline" to avoid collision with the existing Stats Timeline. Trade Journal is a subset of lifecycle events, not a separate top-level feature.

## History

- 2026-06-29: created (status: triaged).
