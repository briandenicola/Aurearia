---
updated_at: 2026-09-19T12:19:13Z
focus_area: F014 stabilization complete; F015 paused after private profile foundation
active_issues:
  - User retest pending for Coin Copilot execution frames and Deep Analysis notes
  - F015 T012-T034 intentionally paused for a new context
handoff_commit: f0f3e807
---

# What We're Focused On

**Stabilize F014, then resume the reduced F015 scope in a new context.**

## Current Status

- F014 execution-frame and Deep Analysis notes regressions were repaired and
  pushed in `8ac2b1e9`.
- Coin Copilot checkpoint compaction now emits the exact Go-compatible bounded
  result envelope.
- Deep Analysis handoff frames and checkpoints accept the same deterministic
  bounded fallback.
- Collector notes now reach vision hypothesis generation and final synthesis as
  untrusted evidence.
- F015 T001-T011 were completed and pushed in `f0f3e807`.
- F015 T012-T034 are intentionally paused.

## Verified Gates

- Agent: 632 tests passed; Ruff passed.
- Go: `go vet ./...` and `go test ./...` passed.
- Web: type-check, lint, production build, and full Vitest suite passed.
- OpenAPI was regenerated and its route contract gate passed.
- New frame and notes guards were tamper-tested.

## Binding Scope

Reduced F015 remains limited to:

1. Private lightweight collector profile/context.
2. Read-only curator guidance using existing collection tools.
3. Existing UI-owned Add to Wishlist for verified available dealer results.

Do not change the auction subsystem or add wishlist-action endpoints, action
tables, audit workflows, watchlist ranking, or provenance-risk workflows.

## Next Context

1. Let the user retest Coin Search and Deep Analysis with detailed attribution
   notes.
2. Re-read the constitution, Feature 363 spec/plan/tasks, and this handoff.
3. Resume at T012 only when the user explicitly restarts F015.
4. Run the combined F014/F015 engineering audit before any beta-to-main v4.2
   release PR.
