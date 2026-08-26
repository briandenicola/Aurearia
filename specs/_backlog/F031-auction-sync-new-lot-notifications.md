---
id: F031
title: "Notify on lots the background auction sync starts tracking"
status: backlog
priority: P2
effort: S
value: 4
risk: 2
owner: unassigned
created: 2026-08-26
updated: 2026-08-26
---

# F031 — Notify on lots the background auction sync starts tracking

## Summary

The background watchlist sync (`AuctionWatchlistSyncService`) silently picks up
lots a user has added to a NumisBids or CNG watchlist, or started bidding on,
elsewhere — on the provider's own site. Until the user next opens the Auctions
page they have no idea the app began tracking anything. Existing auction
notifications only cover lots *already* tracked (price alerts, bid reminders,
ending-soon, the watch-bid digest); nothing announces the tracking itself.
"Done" is: a sync run that starts tracking new lots tells the user once, with
enough detail to recognise each coin and open it in the app.

## Acceptance criteria

- [x] A background sync that starts tracking one or more new lots produces
      exactly one notification per user per run — several new lots are batched
      into a single push, never one push per lot.
- [x] A sync where nothing is newly tracked produces no notification, including
      re-syncs over an unchanged watchlist.
- [x] Lots newly tracked as *watching* and lots newly tracked as *bidding* both
      notify, and the notification says which of the two each lot is.
- [x] The Pushover push is rich HTML and names, per lot: the coin name, the
      auction (house and sale), the lot number, and a fully-qualified link to
      that lot in the app.
- [x] Lots that a first sync picks up already closed (passed/won/lost) do not
      notify — nothing new is being tracked.
- [x] The push body stays within Pushover's message limit for arbitrarily large
      batches, trimming with a summary line rather than being rejected.
- [x] Lot titles scraped from provider sites are HTML-escaped in the push body.
- [x] Users without Pushover configured still get the in-app notification, and
      the sync itself still runs and refreshes lot data either way (F026/F027).

## Constitution alignment

- Principle I (Clear Layered Architecture) — notification composition lives in
  `services/notification_service.go`, detection of newly tracked lots in
  `services/auction_watchlist_sync_service.go` (reusing the repository's
  existing `AuctionLotUpsertResult.Created` signal); no new repository query,
  no handler change.
- Principle IV (Simple Complete Changes) — one notification path, one deep
  link, wired only into the background sync.
- Principle V (Security, Auth, and Privacy by Default) — scraped third-party
  lot titles are HTML-escaped before entering the Pushover HTML body, and the
  deep link is only emitted when `PublicAppURL` is a valid absolute http(s)
  URL (same rule coin-of-the-day already applies).

## Open questions

- [x] Should the *manual* "Sync Watchlists" button notify too? — No. The user
      is looking at the Auctions page and already gets on-screen sync results;
      a push telling them what they just watched happen is noise. Only the
      background sync (alert scheduler and watch-bid digest scheduler) notifies.
- [x] What should the app link point at, given there is no per-lot route? —
      `/auctions?lot=<id>`, a new deep-link parameter on the auctions page that
      opens that lot's detail modal, fetching the lot by id so it works even
      when the current status filter excludes it.
- [ ] Should a lot that transitions watching → bidding on a *later* sync (the
      user placed a bid on the provider's site after the lot was already
      tracked) also notify? Out of scope here — this card covers lots the sync
      newly starts tracking, not status changes on lots it already tracks.

## Notes

Implemented as `NotificationService.NotifyAuctionLotsTracked`, which creates one
in-app notification (`auction_lots_tracked`) and one batched HTML Pushover push
per sync run. `AuctionWatchlistSyncService` collects lots whose upsert reported
`Created` and whose status is watching or bidding, across both providers, and
notifies once at the end of `SyncUser` — so a user with both NumisBids and CNG
configured gets a single push, not one per provider. Notification wiring is
optional (`WithNotifications`), keeping F026's rule that sync never depends on
notification configuration.

## History

- 2026-08-26: created (status: backlog) — requested feature; implemented and
  unit-tested (message builder, batching/trim, escaping, sync detection,
  re-sync silence, web deep link). Status left at `backlog` pending Lead triage
  per `_backlog/README.md` — implementation does not self-advance status.
