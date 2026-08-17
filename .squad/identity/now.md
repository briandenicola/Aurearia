---
updated_at: 2026-08-17T22:48:52-05:00
focus_area: Feature 353 wishlist availability run observability specification complete
active_issues: []
handoff_commit: 54dc3f6
---

# What We're Focused On

Feature 353 specification complete and committed at 54dc3f6.

**Final design:**
- `AvailabilityCycle` parent (20 global terminal) + `AvailabilityRun` children per-user (20 per owner)
- Additive-only schema: new table + nullable `CycleID` FK, no synthetic backfill
- Per-coin `NotifyWishlistUnavailable` alerts + per-run `wishlist_availability_run` notifications coexist
- Legacy `UserID=0` admin rows labeled "Legacy" in UI, left unmodified post-migration
- 51 tasks (T001–T051) ready for implementation delegation

**Spec Status:** Locked in. All three BLOCK findings from initial review resolved.
**Next Phase:** Implementation (Phases 1–8 per tasks.md)
