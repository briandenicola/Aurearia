---
updated_at: 2026-08-20T15:42:00-05:00
focus_area: Feature 355 — Wishlist Purchase Reminders (scope expanded; admin schedule UI pending — T037-T038)
active_issues: []
handoff_commit: pending
---

# What We're Focused On

**Feature 355 — Wishlist Purchase Reminders (Scope Expanded)**

## Status Summary

Spec, plan, and tasks generated for Feature 355 — Wishlist Purchase Reminders. Core backend + frontend implementation complete. Route BLOCK (B1) cleared; NB1 (wishlist badge) resolved. **Scope expanded** (2026-08-20): Admin Settings schedule UI for enable/disable + start-time now required (FR-015a, D12, T037-T038). Pushover integration already satisfies the dual-notification directive — no backend changes needed.

## Implementation Deliverables (Planned)

### Feature 355 Core Capability

**Wishlist Purchase Reminders**
- Date-based reminders on wishlist coins with IANA timezone snapshot
- Daily scheduler with catch-up for overdue reminders; durable idempotency via status gate
- In-app notification (type `purchase_reminder`) with deep-link to coin detail; Pushover best-effort
- Auto-cancel on any IsWishlist -> false transition within the same transaction
- Inline modal + badge on existing wishlist/detail surfaces; no new route
- One active reminder per (coin, user); upsert on re-POST
- Admin Settings schedule UI: enable/disable toggle + start-time input (T037-T038, pending)

## Ownership

- **Cassius**: Backend (model, repo, service, handler, scheduler) — 22 tasks
- **Aurelia**: Frontend (modal, badge, composable, API client, admin schedule UI) — 12 tasks
- **Brutus**: Regression QA — 1 task
- **Maximus**: Architecture review — 1 task

## Next Steps

1. **NEXT:** Aurelia implements T037-T038 (Admin Purchase Reminder Schedule component + tests).
2. After T037-T038, re-run full frontend validation (T034) and Maximus final re-review (T036).
3. Route BLOCK (B1) cleared. Overall APPROVE held pending T037-T038 completion.

## Previous Focus (Archived)

Feature 340 (Shipment Delivered Terminal-State Sync Exclusion) and Feature 354 (Deep-Identification Run History & Wishlist-Eligible Coin of the Day) — implementation/review complete. Beta push pending.
