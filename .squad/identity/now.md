---
updated_at: 2026-08-19T12:50:40Z
focus_area: Feature 354 — Deep-Identification Run History & Wishlist-Eligible Coin of the Day (IMPLEMENTATION/REVIEW COMPLETE; beta push pending)
active_issues: []
handoff_commit: pending
---

# What We're Focused On

**Feature 354 — Deep-Identification Run History & Wishlist-Eligible Coin of the Day (Implementation & Review Complete)**

## Status Summary

All team deliverables approved and ready for merge:
- ✓ Maximus: Spec/plan/tasks authored, design decisions frozen (D1–D13), architecture review complete
- ✓ Cassius: Go API backend (Phases 2–6) implemented, tests passing (10/10 packages)
- ✓ Brutus: Python agent (Phase 7) route implemented, tests passing (366 tests), independent QA regression suite complete (37 tests), APPROVE (no BLOCK)
- ✓ Aurelia: Vue frontend (Phases 8–9) implemented, types strict, tests passing (140 files / 906 tests), Auction grouping complete
- ✓ Quality Gate: All `go build`/`go vet`/`go test`, `pytest`, `npm run build` (Docker strict), `npx vue-tsc --noEmit` passing

## Implementation Deliverables

### Feature 354 Core Capabilities

1. **Persistent deep-identification run history**
   - Terminal-completed/partial runs retained indefinitely (nullable `expires_at` with sentinel `9999-12-31`)
   - Owner-invoked hard-delete (`DELETE /deep-identification/jobs/{id}`)
   - Per-(job,target,linked-coin-existence) re-apply idempotency
   - `DeepAnalysisHistoryPage.vue` with reverse-chrono list, cursor pagination, applied-linkage badge

2. **Wishlist-eligible Coin of the Day**
   - `FeaturedCoin.SourceType` (owned|wishlist)
   - `PickNextCoinID` includes wishlist when `coinOfDayIncludeWishlist` user flag enabled
   - Stateless Python agent `/collection/wishlist-featured-summary` route (≤500 chars, no invented facts)
   - Go proxy with 10s timeout, deterministic fallback on agent failure
   - `FeaturedCoinModal.vue` wishlist badge + "Move to Collection" CTA
   - Per-user Settings Account toggle (default-on per approval)

### Auction Watching/Bidding Grouping

- `AuctionsPage.vue` client-side grouping by `auctionHouse` → `saleName`
- Toggle chip to enable/disable (session-only state)
- Defaults **on** for watching/bidding statuses; other statuses unchanged
- No backend changes; uses existing `.section-label` and `h3` classes

## Quality Gate Validation

All §17 Quality Gate checks passing:
- **Go:** `go build ./...` clean, `go vet ./...` clean, `go test ./... -count=1` (10/10 packages) ✓
- **Python:** `pytest tests/ -q` (366 tests) ✓
- **Vue:** `npx vue-tsc --noEmit` clean, `npm run build` (Docker strict) ✓, Vitest (140 files / 906 tests) ✓
- **Architecture:** Principle I (layered), Principle IV (proportional), Principle V (security) all maintained
- **Contract:** All typed assumptions verified post-merge; zero corrective edits needed

## Next Steps

1. ✓ Orchestration logs (4 files): 2026-08-19T125040Z-{maximus,cassius,aurelia,brutus}.md
2. ✓ Session log: 2026-08-19T125040Z-scribe-feature354-auction-grouping.md
3. ✓ Decisions merged to `.squad/decisions.md`; inbox cleared
4. ✓ Cross-agent history updated
5. **PENDING:** Beta push per user directive (captured in decisions.md)
6. **PENDING:** Coordinator combined product/state commit after scope inspection

## Material Design Decisions (D1–D13)

- **D1:** Nullable `expires_at` + sentinel `9999-12-31` fallback; janitor skips `NULL`
- **D2:** Re-apply idempotency per (job, target, linked-coin-existence)
- **D3:** Hard-delete cascades job/runs/events/artifacts; never Coin
- **D4:** `appliedCoinExists` computed via correlated EXISTS
- **D5:** `FeaturedCoin.SourceType` (owned|wishlist)
- **D6:** `PickNextCoinID` includes wishlist when flag enabled
- **D7:** Stateless Python proxy route, bounded ≤500 chars
- **D8:** Fallback to `buildCoinSummary()` on agent failure; pick never dropped
- **D9:** Move-to-Collection reuses existing coin-update endpoint
- **D10:** Notification schema unchanged; `sourceType` on payload
- **D11:** `User.CoinOfDayIncludeWishlist` bool (DEFAULT true)
- **D12:** Hints ephemeral; only obverse/reverse retained
- **D13:** Failed/cancelled keep 90-day expiry; completed/partial indefinite

## Previous Focus (Archived)

Feature 353 hotfix (PR #629 / #630 merged at d625b08) is done. Production is on hotfix 2; recovery was atomic.
