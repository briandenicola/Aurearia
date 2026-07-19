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

- [x] A lot is flagged "needs attention" when its close time
      (`auctionEndTime`, falling back to `saleDate`) is in the past AND its
      status is still `watching` or `bidding`. Implemented as a pure function,
      `auctionLotNeedsAttention()` in `src/web/src/utils/auctionLot.ts`.
- [x] Flag is visible on `AuctionLotCard.vue` and `AuctionLotDetailModal.vue`.
- [x] `AuctionsPage.vue` has a "Needs Attention" toggle chip (with a live count)
      that filters the currently-loaded list down to flagged lots.
- [x] Computed entirely client-side — no backend change, no migration.

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
- 2026-07-19: implemented. Open questions (grace period; filter vs. banner UX)
  resolved pragmatically for V1: no grace period (flags immediately once close
  time passes — simpler, and a false-negative window is worse than a
  false-positive one here), and a filter chip approach since it composes
  with the existing status/source filters rather than adding a separate UI
  surface. Left at `backlog` rather than self-advancing to
  `triaged`/`promoted`, per this repo's Lead-driven workflow.
