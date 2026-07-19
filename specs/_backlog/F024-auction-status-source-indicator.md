---
id: F024
title: "Show whether a lot's Won/Lost status was auto-detected or manually set"
status: backlog
priority: P2
effort: S
value: 3
risk: 2
owner: unassigned
created: 2026-07-19
updated: 2026-07-19
---

# F024 — Show whether a lot's Won/Lost status was auto-detected or manually set

## Summary

CNG lots can now auto-resolve to Won/Lost when the provider reports the lot
closed (see commit `0943a56` on `claude/auction-functionality-review-17b5cj`).
NumisBids lots cannot (F022, blocked on F021) — they still require a manual
override. A Won/Lost lot in the UI looks identical regardless of which path set
it, which could read as inconsistent or make a NumisBids user wonder why their
lots never "just resolve themselves" the way CNG ones do.

## Acceptance criteria

- [ ] `AuctionLot` records whether its current status was set by sync
      (`"sync"`) or by an explicit user action (`"manual"`).
- [ ] Repository `upsert()`'s auto-transitions (watching→bidding,
      watching→passed, →won/lost) tag the write as `"sync"`.
- [ ] `AuctionLotService.UpdateStatus` (the manual override) tags the write as
      `"manual"`.
- [ ] `AuctionLotCard.vue` and `AuctionLotDetailModal.vue` show a small
      indicator (e.g. an icon + tooltip) next to the status badge, but only for
      `won`/`lost` — the source of `watching`/`bidding`/`passed` isn't
      interesting to the user.
- [ ] New field is additive (nullable/defaulted), no backfill required for
      existing rows.

## Constitution alignment

- Principle I (Clear Layered Architecture) — the field is set at the
  repository/service layer where each transition actually happens, not
  inferred later from other data.
- Principle III (Strict Types and Explicit Contracts) — use a typed enum
  (`AuctionLotStatusSource`), not a free-form string.
- Principle VI (Consistent User Experience) — the indicator should read as an
  small, secondary hint, not compete with the status badge itself.

## Open questions

- [ ] Exact copy/icon for "sync" vs "manual" — needs a design decision, not
      just an engineering one.
- [ ] Should this also cover `passed`, once F022 possibly adds a real
      auto-detected "sold to someone else" signal for NumisBids?

## History

- 2026-07-19: created (status: backlog) — split out from open UI gaps noted at
  the end of the CNG rebuild work on issue #482.
