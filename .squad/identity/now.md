---
updated_at: 2026-08-18T05:31:21-05:00
focus_area: Feature 353 production migration hotfix — AutoMigrate ordering and FK constraint gap resolved; PR #629 pending merge
active_issues: []
handoff_commit: 1df5a99
---

# What We're Focused On

**Feature 353 Production Migration Hotfix — PR #629 Pending Merge**

## Incident Response

Production startup failed post-Feature-353-release due to AutoMigrate parent/child ordering incompatibility plus latent pre-existing FK constraint.

**Root causes:**
1. `AvailabilityCycle` registered after `AvailabilityRun` in AutoMigrate slice → SQLite temp-table rebuild of `availability_runs` fails trying to add FK to non-existent parent table
2. Pre-existing `AvailabilityRun.User` FK incompatible with legacy `UserID=0` admin-run sentinel → latent `FOREIGN KEY constraint failed` on any rebuild

**Fixes committed (1df5a99):**
- Moved `AvailabilityCycle` immediately before `AvailabilityRun` in AutoMigrate (parent-before-child)
- Added `constraint:-` to `AvailabilityRun.User` FK tag (skip DDL; enforce ownership in services/repositories)

**Recovery:** All failed production restarts rolled back atomically; no manual DB cleanup required.

**Validation:**
- Brutus: Reproduced exact error; confirmed both bugs; validated fix with legacy rows. ✓ Approved
- Maximus: Reviewed parent-before-child precedent, `constraint:-` pattern, recovery atomicity, ownership enforcement. ✓ Approved

## Status

- ✓ Backend hotfix complete and validated
- ✓ Orchestration logs written (Cassius, Brutus, Maximus)
- ✓ Decision merged into decisions.md
- ✓ PR #629 opened
- ⧖ Pending: merge (auto-merge unavailable; normal merge process, awaiting beta branch sync)

## Next Phase

Release coordination and production deployment.
