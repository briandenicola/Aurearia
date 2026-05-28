---
title: Wishlist and Availability Check
id: F002
status: promoted
priority: P1
effort: M
value: 4
risk: 3
owner: Cassius
created: 2026-05-28
updated: 2026-05-28
---

## Summary

Wishlist tracking with per-coin URL availability verification. System polls URLs via HTTP + keyword heuristics, escalates ambiguous results to AI agent Team 6 for analysis. Admins configure scheduled checks (start time + repeat interval).

## Acceptance Criteria

- When user adds coin to wishlist, it appears in dedicated Wish List page (not main collection)
- When "Check Availability" is clicked, system visits each reference URL and returns HTTP code + keyword verdict (sold/available/unknown)
- When verdict is ambiguous, AI agent (Team 6) analyzes and returns final verdict
- When availability check runs scheduled, run history persists with per-coin drill-down (URL, status, reason, HTTP code, AI used)
- When coin status changes to unavailable, UI shows red "Unavailable" overlay and green/amber dots for available/unknown

## Constitution Alignment

**Principle II (Dependency Injection):** Availability check is a cross-package service orchestrating repos + AI agent proxy.
**Principle XVIII (Agent Operating Rules):** Verification agents confirm URLs are live before downstream use; only tool-returned data passes downstream.

## Implementation Notes

- Wishlist flag on Coin model; coins with flag appear only in `/wishlist` gallery
- Availability check endpoints: `POST /wishlist/check`, `GET /admin/availability-checks` (history)
- Admins configure: `AvailabilityCheckEnabled`, `AvailabilityCheckStartTime`, `AvailabilityCheckInterval`
- Run history stored; per-coin results: URL, status, reason, HTTP code, AI agent flag
- Ambiguous results escalated to `app/availability_check.py` agent team

## Open Questions

None — feature shipped.

## Notes

Retroactive card created 2026-05-28 for governance traceability under Constitution §0 (Hierarchy) Phase 2. Purchase modal styling implemented; dismissed coins can clear status indicators.
