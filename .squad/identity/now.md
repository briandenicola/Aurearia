---
updated_at: 2026-08-20T22:52:31Z
focus_area: Feature 355 — Wishlist Purchase Reminders (timezone hotfix merged; beta release pending all gates)
active_issues: []
handoff_commit: pending
---

# What We're Focused On

**Feature 355 — Wishlist Purchase Reminders (Production-Ready, Beta Pending)**

## Status Summary

Feature 355 implementation COMPLETE. Core backend + frontend DONE. Route BLOCK (B1) cleared; wishlist badge (NB1) resolved. Admin schedule UI (T037-T038) DONE. **Production Timezone Defect Fixed** (2026-08-20): Alpine container lacking zoneinfo caused `time.LoadLocation` to fail for valid IANA zones. Hotfix: embedded IANA database via stdlib `time/tzdata` blank import. Approved by Maximus. **Beta release pending** final operational gates (T034 frontend validation, T035 scheduler regression test).

## Critical Hotfix: Timezone Portability (2026-08-20)

### Defect
Production Alpine containers lack `/usr/share/zoneinfo`. Feature 355's timezone validation (`time.LoadLocation`) failed for all valid IANA zones (e.g., `America/Chicago`), returning HTTP 400 for legitimate user inputs.

### Solution
Added `_ "time/tzdata"` blank import to `src/api/main.go`. Standard library package self-registers full IANA timezone database at init. Binary becomes self-contained; no OS zoneinfo dependency.

### Status
- **Implementation**: DONE (Cassius)
- **Review**: APPROVED (Maximus)
- **Regression Test**: 8 valid zones + 3 invalid rejections (catches removal on Alpine)
- **Supply Chain**: Zero risk (stdlib only)
- **Binary Size**: +450 KB (negligible)
- **Docker**: No changes needed

### Files Changed
- `src/api/main.go`: +1 line (blank import)
- `src/api/timezone_embed_test.go`: New test (package main)

### Next: Non-Blocking Post-Merge Hardening
Add CI container test to verify regression on Alpine (recommended but not gating).

## Implementation Deliverables (Complete)

### Feature 355 Core Capability

**Wishlist Purchase Reminders**
- Date-based reminders on wishlist coins with IANA timezone snapshot (now portable to Alpine)
- Daily scheduler with catch-up for overdue reminders; durable idempotency via status gate
- In-app notification (type `purchase_reminder`) with deep-link to coin detail; Pushover best-effort
- Auto-cancel on any IsWishlist -> false transition within the same transaction
- Inline modal + badge on existing wishlist/detail surfaces; no new route
- One active reminder per (coin, user); upsert on re-POST
- Admin Settings schedule UI: enable/disable toggle + start-time input (DONE — T037-T038)

## Ownership

- **Cassius**: Backend (model, repo, service, handler, scheduler) — DONE; hotfix approved
- **Aurelia**: Frontend (modal, badge, composable, API client, admin schedule UI) — DONE
- **Brutus**: Regression QA — PENDING (T034, T035 required before beta→main merge)
- **Maximus**: Architecture review — COMPLETE + hotfix approval

## Operational Gates (Before Beta→Main Merge)

1. **T034**: Full frontend build validation (`npm run type-check`, `npm run build`)
2. **T035**: Brutus scheduler regression test

Both are **operational gates** (not architectural blockers). Architecture review is complete.

## Next Steps

1. **IMMEDIATE**: Run operational gates T034 and T035
2. **Post-gates**: Merge beta branch to main
3. **Post-merge**: Implement non-blocking hardening (CI container test for timezone regression)

## Previous Focus (Archived)

Feature 340 (Shipment Delivered Terminal-State Sync Exclusion) and Feature 354 (Deep-Identification Run History & Wishlist-Eligible Coin of the Day) — implementation/review complete. Merged to beta pending Feature 355 final gates.
