---
id: F034
title: "Notify when the provider reports you outbid on a lot"
status: backlog
priority: P1
effort: S
value: 5
risk: 2
owner: unassigned
created: 2026-08-31
updated: 2026-08-31
---

# F034 — Notify when the provider reports you outbid on a lot

## Summary

The Auctions page shows an `OUTBID` badge and a "Needs attention" flag on a lot
someone has outbid you on, but nothing tells you — the badge is computed in the
browser at render time and the server never evaluates the condition, so no
notification of any kind exists. The user only learns they were outbid by
opening the app and scrolling to the lot, which for a lot closing overnight is
the same as not learning at all. This is the most time-critical auction event
the app tracks and the only one it stayed silent on. "Done" is: losing the lead
on a lot you are bidding on notifies you once, while there is still time to
respond.

## Acceptance criteria

- [x] The background watchlist sync notifies when the provider first reports
      someone else holding the winning bid on a lot the user is bidding on.
- [x] The notification fires once per time the lead is lost — not once per sync
      while the user remains behind.
- [x] Retaking the lead re-arms it: a later loss of the lead notifies again.
- [x] Outbid state is decided by the provider's own winning-bidder identity, not
      by comparing max bid against current bid.
- [x] Several lots outbid in one sync run produce one batched notification, not
      one per lot, matching the newly-tracked and now-bidding pushes.
- [x] The notification names each lot, where the bidding stands (current high
      bid and the user's max bid), and how long is left before the lot closes.
- [x] The Pushover push is rich HTML with the lot number linked to the auction
      site and a link into the app; scraped titles and sale names are escaped.
- [x] Users without Pushover configured still get the in-app notification.
- [x] A lot that closes is won, lost or passed — never left sitting as outbid.

## Constitution alignment

- Principle I (Clear Layered Architecture) — detection in
  `services/auction_watchlist_sync_service.go`, the not-outbid → outbid edge
  reported by the repository that owns the stored value (extending F031's
  `AuctionLotUpsertResult.PreviousStatus` precedent), composition in
  `services/notification_service.go`. No handler change.
- Principle IV (Simple Complete Changes) — one added column, one predicate, one
  notification, wired into the sync that already runs.
- Principle V (Security, Auth, and Privacy by Default) — scraped lot titles and
  sale names are HTML-escaped into the push, and links are emitted only for
  absolute http(s) URLs (F031's rule).
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [x] Compare max bid against current bid, as the UI badge does? — No. CNG
      reports the winning bidder's own customer id, and the sync already reads it
      to resolve won/lost at close; using it live is authoritative. Under proxy
      bidding the amounts lie in both directions: a ceiling above the current bid
      can still be losing, and one below it can still be leading.
- [x] What if the provider names no winning bidder, or the user's own id is
      unavailable? — Not outbid. Silence beats a false alarm telling someone they
      lost a lot they are winning.
- [x] Should the notification repeat while the user stays behind? — No. A lot can
      sit outbid for days (the reported one had nine), and an hourly scheduler
      would send it dozens of times. Once per loss of the lead, re-armed by
      retaking it.
- [ ] Should the Auctions page's `OUTBID` badge switch to the stored server flag
      now that one exists? It would agree with the notification and be right
      under proxy bidding, but the client-side heuristic is the only signal for
      NumisBids lots, so the badge would need a fallback. Left out of this card:
      the badge was not what was broken.

## Notes

Reported directly: a CNG Keystone lot showed `OUTBID` and "Needs attention" in
the PWA with no notification on either surface. The badge comes from
`AuctionLotCard.vue`'s `biddingIndicator`, and "Needs attention" from
`utils/auctionLot.ts` — both browser-side, both invisible to the server.

Why the neighbouring notifications did not cover it: `auction_lots_bidding`
fires once on the watching → bidding transition and is explicitly guarded
against re-firing while the lot stays in bidding, which is exactly the window
where being outbid happens; `auction_price_alert` only fires for a target the
user configured on that specific lot and knows nothing about their max bid; and
sync sets `bidding` whether the user is winning or losing, so no status
transition existed to hang a notification on.

Implemented as `AuctionLot.IsOutbid` (provider truth, persisted),
`AuctionLotUpsertResult.BecameOutbid` (the edge, reported by the repository that
owns the previous value), `outbidByProvider` (the predicate), and
`NotificationService.NotifyAuctionLotsOutbid` (batched in-app notification plus
one rich-HTML push per sync run). Only CNG can drive this: NumisBids exposes no
bid signal at all (F022), so its lots never set the flag.

Operational note: outbid notifications ride on the background watchlist sync,
which runs from the auction-alert scheduler (hourly by default) and the
watch-bid digest scheduler. If neither is enabled in Admin → Schedules, no sync
runs in the background and nothing fires.

## History

- 2026-08-31: created (status: backlog) — requested feature; implemented and
  unit-tested (the full lose/hold/retake/lose-again cycle end to end through a
  stubbed CNG watchlist, the predicate's truth table, the repository edge, push
  rendering, escaping, and the closes-in wording). Status left at `backlog`
  pending Lead triage per `_backlog/README.md` — implementation does not
  self-advance status.
