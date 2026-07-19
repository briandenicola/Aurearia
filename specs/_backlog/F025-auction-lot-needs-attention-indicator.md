---
id: F025
title: "Flag lots whose auction has ended but status was never confirmed"
status: backlog
priority: P1
effort: S
value: 4
risk: 2
owner: unassigned
created: 2026-07-19
updated: 2026-07-19
---

# F025 — Flag lots whose auction has ended but status was never confirmed

## Summary

A lot can sit in `watching` or `bidding` indefinitely after its real-world
auction has closed: CNG auto-resolves Won/Lost on the next sync (F022 work),
but NumisBids cannot yet (blocked on F021), and even for CNG a lot only
resolves on its *next* sync — there's a window, and if sync is ever disabled
or fails, a lot can stay stale. Nothing in the UI today tells the user "this
one needs you to go check and confirm" — they'd only notice by chance while
browsing.

## Acceptance criteria

- [ ] A lot is flagged "needs attention" when its close time
      (`auctionEndTime`, falling back to `saleDate`) is in the past AND its
      status is still `watching` or `bidding`.
- [ ] Flag is visible on `AuctionLotCard.vue` (so it's visible in list views,
      not just when a lot happens to be opened) and in
      `AuctionLotDetailModal.vue`.
- [ ] `AuctionsPage.vue` can filter/sort by this flag, so a user can find every
      lot needing attention in one place rather than scanning the whole list.
- [ ] Computed client-side from data already returned by the API — no backend
      change should be required for the base case (it's a pure function of
      `auctionEndTime`/`saleDate`/`status` against "now").

## Constitution alignment

- Principle IV (Simple Complete Changes) — purely derived UI state; no new
  persisted field, no migration.
- Principle VI (Consistent User Experience) — should read as a clear call to
  action, not just another badge competing with status/source (F024) or the
  Winning/Outbid indicator.

## Open questions

- [ ] Should there be a grace period (e.g. don't flag until N hours past
      close, since providers sometimes take a little time to report the final
      outcome) or is "past close time, still watching/bidding" sufficient?
- [ ] Does this belong as a filter option alongside the existing status filter
      on `AuctionsPage.vue`, or a separate always-visible counter/banner?

## History

- 2026-07-19: created (status: backlog) — split out from open UI gaps noted at
  the end of the CNG rebuild work on issue #482.
