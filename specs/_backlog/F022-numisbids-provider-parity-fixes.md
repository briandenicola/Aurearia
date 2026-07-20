---
id: F022
title: "Verify and remediate NumisBids provider gaps"
status: backlog
priority: P1
effort: M
value: 4
risk: 3
owner: unassigned
created: 2026-07-19
updated: 2026-07-19
---

# F022 — Verify and remediate NumisBids provider gaps

## Summary

This is the remediation counterpart to F021 (ground-truth audit), not a promise
that NumisBids can reach the same automation level as CNG. The CNG side of
issue #482 was rebuilt after a live-account audit found the scraper reading dead
JSON field names, a redundant per-lot HTTP request, a wrong end-time field, and no
way to auto-detect a lot's outcome once it closed — all fixed and verified against
a real account (see `claude/auction-functionality-review-17b5cj` session history).
NumisBids uses a different scraping approach entirely (HTML/regex against
`numisbids_service.go`, not a JSON API), so none of the CNG code changes apply
directly, but the same category of drift is unverified there. This card is the
actual fix work once F021's ground-truth pass identifies what, if anything, is
available or wrong — it should not start until F021 has real fixture data to work
from. If NumisBids does not expose max-bid or final outcome signals, the correct
resolution is documentation and UI clarity that NumisBids remains manual for
those fields.

## Acceptance criteria

- [ ] Every regex/extraction in `numisbids_service.go` matches real, current
      NumisBids markup (per F021's fixtures), not assumptions from the original
      integration.
- [ ] `CurrentBid` reflects the real live/final bid; `MaxBid` reflects the user's
      real bid ceiling if NumisBids exposes one (confirm in F021 first — it may
      not have an equivalent concept to CNG's `absentee_bid`).
- [x] `AuctionEndTime` is populated at all for NumisBids lots — fixed without
      needing live data (was a pure wiring gap: `syncNumisBids` parsed a sale
      date and never assigned it to the field `bidReminderDue()` requires, so
      bid reminders could never fire for any NumisBids lot). See History.
- [ ] Sale/lot end-time *precision* is confirmed correct for multi-lot sales
      (confirm in F021 whether NumisBids has CNG's per-lot vs. sale-wide
      end-time distinction at all — the current fix uses whatever coarse date
      `ParseSaleDate` extracts, which is a bare calendar date with no per-lot
      time component).
- [ ] If NumisBids exposes any closed-lot outcome signal (winner identity, final
      price, lifecycle status), wire it into `syncNumisBids` for the same
      auto won/lost detection CNG now has, using
      `auction_watchlist_sync_service.go`'s `syncCNG` as the reference
      implementation (repository `upsert()` already supports the transition
      generically — no repository changes should be needed).
- [ ] If NumisBids has no such signal, document that explicitly rather than
      leaving it silently unhandled — NumisBids lots will keep needing a manual
      status override, and the UI should not imply otherwise.
- [ ] Regression tests built from F021's real fixtures, not hand-authored HTML
      that could encode the same wrong assumptions again.

## Constitution alignment

- Principle I (Clear Layered Architecture) — fixes stay inside
  `services/numisbids_service.go` and `auction_watchlist_sync_service.go`; the
  repository/service status-transition rules added for CNG are already
  provider-agnostic and should not need modification.
- Principle III (Strict Types and Explicit Contracts) — replace assumed markup
  shapes with ones verified against real responses.
- Principle IX (Automated Enforcement Over Manual Memory) — regression tests
  derived from real fixtures so this can't silently drift again.

## Open questions

- [ ] Blocked on F021 producing real fixture data — do not start estimation or
      implementation before that lands.
- [ ] Does NumisBids even have a "your max bid" / autobid concept, or is bidding
      handled entirely off-platform relative to what the watchlist page reports?
- [ ] Should the UI (per the auction-functionality-review session's open UI-gap
      notes) distinguish "auto-resolved" lots from "needs manual confirmation"
      lots once CNG and NumisBids diverge in this capability?

## Notes

Reference implementation for the pattern to follow (once ground truth is known):
commit `0943a56` on `claude/auction-functionality-review-17b5cj` — rebuilt
`cng_auction_service.go`'s field mapping, dropped a redundant per-lot scrape,
fixed a lot-vs-sale end-time bug, and added won/lost auto-detection by comparing
a closed lot's winning bidder ID against the logged-in user's own customer ID.

## History

- 2026-07-19: created (status: backlog) — split out as the fix-work counterpart
  to F021 so the two don't get conflated into one unbounded card.
- 2026-07-19: fixed the `AuctionEndTime` wiring gap (see acceptance criteria) —
  this was safe to do without live data since it was a plumbing omission, not a
  markup-verification question. `numisbidsBase` was also changed from a `const`
  to a `var` (mirroring `numisbidsLoginURL`/`numisbidsWatchlistURL`) purely to
  make this testable against a local server. Covered by
  `TestAuctionWatchlistSyncService_SyncNumisBidsSetsAuctionEndTime`. Everything
  else on this card (the `MaxBid`/auto-bidding-detection gap, end-time
  precision, and any won/lost signal) remains blocked on F021.
