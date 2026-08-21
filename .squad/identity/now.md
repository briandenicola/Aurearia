---
updated_at: 2026-08-21T10:37:35Z
focus_area: Experimental PWA coin-detail swipe navigation (approved for beta only; 68 targeted + 1122 full frontend tests/type-check/build green; Maximus APPROVE; awaiting device evaluation)
active_issues: []
handoff_commit: pending
---

# What We're Focused On

**Experimental PWA Coin-Detail Swipe Navigation (Beta-Only, Approved, Release-Ready)**

## Status Summary

Swipe navigation experiment complete and approved for beta release. Canonical 8-stop menu (Overview → Shipment → Journal → Health → Notes → Actions → Analysis → Valuation; Sell/Copy structurally excluded) navigable via left/right touch swipe on PWA installs. Design review and QA clearance complete (Maximus APPROVE, Brutus APPROVE). Full frontend test suite green (68 targeted + 1122 total). Shipped to beta only per Brian's directive; no main PR opened. Ready for device evaluation on beta; main merge pending mounted integration tests + real-PWA device results.

## Feature 356 Status (Completed)

Value history journal remediation shipped to beta and merged:
- Journal bloat eliminated (scheduled AI estimates removed)
- Tag suggestions restored (ruler-weight gate fixed)
- Silent-failure bug fixed (error states distinct from empty results)

## PWA Swipe Experiment Details

### Design & Scope

**8-stop canonical order (no wrap):**
1. Overview (base)
2. Shipment Tracker
3. Activity Journal
4. Metadata Health
5. Notes
6. Actions
7. AI Analysis
8. Value Trend

**Gesture semantics:**
- 64 px minimum horizontal distance
- 2:1 axis dominance (≈27° cone)
- 10 px axis-lock slop (vertical lock abandons swipe)
- 24 px edge guard on both sides (iOS/Android system swipe protection)
- Touch-only; primary pointer; multi-touch cancels
- No wrap at boundaries
- Passive listeners; no preventDefault (native scroll/momentum preserved)
- Interactive descendants suppressed (button, input, a, select, textarea, role=button, contenteditable, [data-swipe-ignore])

### Implementation (Aurelia)

**Components modified:**
- `useCoinDetailSwipeNav.ts` — New composable (exported constants: SWIPE_THRESHOLD, AXIS_SLOP, EDGE_GUARD, AXIS_DOMINANCE)
- `CoinDetailPage.vue` — Composable wired; modal gates (sell/purchase/reminder)
- `CoinDetailSectionPageShell.vue` — Composable wired
- `CoinDetailValuationPage.vue` — `data-swipe-ignore` added to value-history table (overflow-x-auto)

**Test coverage:**
- `useCoinDetailSwipeNav.test.ts` — 68 dedicated tests (all design-review criteria)
- Integration tests in `CoinDetailPage.test.ts`, `CoinDetailValuationPage.test.ts`, `CoinDetailHeaderActions.test.ts`

### QA Clearance (Brutus)

✅ **68 targeted tests PASS** — All design-review criteria verified
✅ **1122 full frontend tests PASS** — No regressions
✅ **Type-check PASS** (vue-tsc --build)
✅ **Production build PASS** (vite 2309 modules, no warnings)
✅ **No blocks or findings** — Implementation matches approved contract

### Review Status

**Maximus:** APPROVE for beta only. Main re-entry criteria:
1. Mounted call-site integration tests (real router/coin context)
2. Recorded installed-PWA device evaluation (iOS/Android real-device swipe feel)
3. Optionally: cleanup items from QA findings (non-blocking)

**Notable:** Test commit `aea17127` may show red in isolation if Feature 356 not merged; not a blocker.

### Release Plan

✅ Beta: Shipped; awaiting device evaluation
⏳ Main: Pending mounted integration tests + real-PWA device results

## Non-Blocking Follow-Up Items

From Maximus design review:
- Dead code cleanup: `components/coin/CoinDetailSectionLinks.vue` (not imported; separate `chore:` change)
- Real-device smoke testing (iOS/Android; not blocking beta)

## Operational Gates (Before Main Merge)

1. Real-PWA device evaluation (iOS/Android)
2. Mounted call-site integration tests
3. Optional: cleanup items from NB findings
