---
title: Portfolio Monthly Valuation Snapshots
id: F009
status: promoted
priority: P2
effort: M
value: 4
risk: 2
owner: unassigned
created: 2026-05-28
updated: 2026-05-28
---

## Summary

Capture an immutable monthly snapshot of every active coin's current valuation so the portfolio page can render appreciation/depreciation trends over time (per coin, per category, per era, and portfolio-wide). Complements the existing real-time value-over-time view by guaranteeing a sampled baseline that does not depend on ad-hoc page visits.

## Acceptance Criteria

- On the first day of each month (UTC), a scheduled job captures `(coin_id, snapshot_date, current_value, currency)` for every active, non-wishlist coin owned by every user opted into snapshots
- Snapshots are immutable — historical valuations are not retroactively rewritten when a coin's `currentValue` field changes
- Portfolio page renders a trend chart (line, monthly granularity) with toggles for: total portfolio, by category, by era, by single coin
- When a coin is added mid-month, it appears in the next snapshot; no backfill
- When a coin is sold or deleted, its snapshot history is retained (read-only) and excluded from "active portfolio" totals going forward
- Admin can manually trigger a snapshot run (debug/recovery)
- Per-user opt-in toggle in Settings → Account (default on for new accounts)

## Constitution Alignment

**Principle II (Layered Architecture):** New `valuation_snapshots` repository + service; scheduled job lives alongside existing schedulers (e.g., `CoinOfDayScheduler`).
**Principle VI (Data Integrity & Immutability):** Snapshots are insert-only; no UPDATE path exposed.
**Principle VII (Schema-Driven Contracts):** New DTO for trend response; chart endpoint returns aggregated, paginated time series.
**Principle XV (Supply Chain & CI Integrity):** No new external dependency; reuse existing charting library on the frontend.

## Implementation Notes

- New model: `ValuationSnapshot { ID, UserID, CoinID, SnapshotDate (DATE, indexed), Value (numeric), Currency }`; unique index `(UserID, CoinID, SnapshotDate)`
- Scheduler: `services/valuation_snapshot_scheduler.go` — runs daily, no-ops unless it's the configured monthly day; idempotent via the unique index
- Endpoints:
  - `GET /portfolio/trend?granularity=monthly&group_by=category|era|coin|none&from=YYYY-MM&to=YYYY-MM` (auth required)
  - `POST /admin/valuation-snapshots/run` (admin) — manual trigger
- Admin settings (new): `ValuationSnapshotEnabled`, `ValuationSnapshotDayOfMonth` (1–28), `ValuationSnapshotTime` (HH:MM UTC)
- User field (new): `User.ValuationSnapshotEnabled` (default `true`)
- Frontend: extend portfolio page with monthly trend chart; reuse design tokens (`--accent-gold` for line, `--text-muted` for axis)
- Storage cost: ~150 bytes/snapshot × N coins × 12 months/year; bounded and predictable

## Open Questions

- Default day-of-month: 1st? Or last calendar day to capture month-end value?
- Trend chart: line, area, or candlestick (min/max via daily real-time fetches between snapshots)?
- Should we expose a CSV download of the snapshot series for power users? (Note: §8 Q4 closed CSV as a No for export — this is read-only analytical, not a coin export, so different concern.)

## Dependencies

- Existing scheduler infrastructure (`services/coin_of_day_scheduler.go`) as pattern reference
- Existing portfolio review pipeline (uses current values; this adds historical depth)

## References

- PRD §8 Q2 (Portfolio valuation tracking — resolved Yes, 2026-05-28)
- F003 (Portfolio Review) — complementary, not blocking
