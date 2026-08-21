---
updated_at: 2026-08-21T15:19:36Z
focus_area: Swipe reliability correction and shared primitive (unified Gallery/detail/Tray on useSwipeGesture; root-cause pan-y fix; Livia B1-B5 clearance; Maximus CONDITIONAL APPROVE; beta validation authorized; new PR held unmerged; merge gate = Brian on-device iOS/Android evaluation)
active_issues: []
handoff_commit: pending
---

# What We're Focused On

**Swipe Reliability & Shared Engine Correction (Livia B1-B5 Cleared, Maximus CONDITIONAL APPROVE, Beta Validation + On-Device Gate)**

## Status Summary

Swipe navigation reliability root cause (UA pointer claim on `touch-action: pan-x pan-y`) diagnosed and fixed. Detail swipe migrated to shared `useSwipeGesture` primitive with unified direction semantics (left-next/right-prev). Gallery and Tray also migrated; all three surfaces on one engine. Livia independently cleared all five B1-B5 blocks (Linux guard, localStorage, Tray scroll, target migration, docs) under Strict Lockout. Maximus CONDITIONAL APPROVE (B2-B5 clear; B1 clears on green ubuntu CI). Beta-only validation authorized. New PR from beta to main held unmerged pending Brian's installed-PWA iOS/Android evaluation (detail swipe feel + full 12-coin Tray scroll).

## Swipe Reliability Details

### Root Cause Diagnosis (Maximus)

**Problem:** Detail swipe intermittent ("works occasionally, hard to start"). Gallery is reliable. Tray has same latent bug.

**Root cause:** Detail page inherits `touch-action: pan-x pan-y` from body. Page is scrollable. After ~8 px, browser UA claims pan, fires `pointercancel`, stops delivering `pointermove/pointerup`. Composable cannot survive: needs those events to lock axis and commit. Cancel fires `reset()`, `activePointerId` nulls, later `pointerup` filtered, gesture throws away.

**Why Gallery works:** Uses `touch-action: none` on `.card-stack`, removing UA from negotiation.

**Contributing issues (all live, ranked):**
1. Release-time 2:1 axis dominance gate rejected natural thumb swipes (80 px right, 50 px down)
2. Exclusion selector too broad (button, a, [role=button]) — top band unswipeable
3. Document-wide text selection gate silenced feature after accidental long-press
4. Commit distance not problem (detail 64 < Gallery 100, already permissive)

### Design Solution

**Fix reliability:**
1. Attach `touch-action: pan-y` on container while mounted; restore on detach/toggle-off
2. Delete release 2:1 dominance gate (10 px axis lock sufficient)
3. Narrow exclusion: input, textarea, select, [contenteditable], [data-swipe-ignore] (not button/a)
4. Drop text-selection gate
5. Keep 24 px edge guard

**Unify direction:** left = next (iOS); right = previous. Gallery was backwards (reversed in Phase 2).

**Shared primitive:** `useSwipeGesture` (Gallery, detail, Tray). No new horizontal-swipe engines permitted.

**Setting relocation:** Account → Appearance/View Settings. Storage stays account-wide server-backed.

### Implementation Phases

**Phase 1 (Aurelia):**
- `useSwipeGesture.ts` new (target, threshold, touchAction, capture, lockSlop, pointerTypes, callbacks)
- `useCoinDetailSwipeNav.ts` rewritten as adapter (pan-y, capture: false, lockSlop: 10, threshold: 64)
- Settings: Account → Appearance; immediate-apply semantics
- Docs: PWA Guide, Features

**Phase 2 (Aurelia):**
- Gallery migrated (touchAction: none, capture: true, lockSlop: null, threshold: 101; direction reversed)
- Tray migrated (same; fixed T3/T4 defects)
- G4 (pointercancel committed): → spring-back
- T3 (pointercancel committed): → spring-back
- T4 (touch-pan-y blocked scroll): → pan-y

### Block Revision Cycle 1 (Maximus - BLOCK, 5 Concrete Defects)

**B1 — Linux guard test fails CI:** Backslash path construction on POSIX produces literal backslash in filename; readFileSync throws on ubuntu-latest.

**B2 — Duplicate save path:** Component inline save omitted localStorage. Composable `savePwaSwipeNav()` had correct full path but never called. Tests certified behavior UI never reached (false positives).

**B3 — Tray scroll blocked:** `touchAction: none` + no axis lock. Page vertical scroll prevented; diagonal drags committed drawer.

**B4 — Target A→B swap broken:** Watcher condition `el && !attachedElement` failed on direct A→B (B truthy, A still attached). Test pinned as "known limitation" + asserted defect.

**B5 — Docs errors:** Empty Account table post-move. Appearance table still claimed "stored in local storage" (false for Swipe).

### Block Revision Cycle 2 (Livia B1-B5, Independent Under Strict Lockout)

**Agent:** Livia (Temporary Vue 3/TypeScript/PWA Specialist, production QA approved)

**Lockout:** Aurelia and Brutus locked out of rejected edits; Maximus review-only. All five items cleared by Livia independently.

| Item | Resolution |
|---|---|
| B1 | Removed replace, broadened heuristic (pointerdown+move+up rule), added invariant. Guard 3/3. ✅ |
| B2 | Component calls composable path, inline deleted, localStorage test added. Settings 28/28. ✅ |
| B3 | Tray pan-y + lockSlop: 10 (proven detail profile); new structural/behavioral tests. Tray 24/24 (+2). ✅ |
| B4 | Watcher `el !== attachedElement`; test inverted to A→B regression. Primitive 42/42. ✅ |
| B5 | Table removed, storage sentence clarified. Docs. ✅ |

### Final Review (Maximus - CONDITIONAL APPROVE)

**Verdict: CONDITIONAL APPROVE — B2-B5 clear; B1 clears on green ubuntu `CI / Vue Web` after beta validation push**

**What passes:**
- Architecture (one shared primitive for Gallery/detail/Tray)
- Root-cause fix (touch-action: pan-y eliminates UA pointer claim)
- Convergence (3 engines → 1)
- Arced swipes (release 2:1 gate removed)
- Buttons/links (selector narrowed)
- Text selection (gate removed)
- Cancel handling (routes to spring-back, not commit)
- Tray scroll (pan-y + lockSlop)
- Target migration (A→B watcher fixed)
- Docs

**Release:**
- New PR from beta to main (PR #653 already merged)
- Held unmerged pending Brian's installed-PWA evaluation
- Beta validation authorized

**Merge gate:** Brian installed-PWA iOS/Android check:
1. Detail swipe feel (start/responsiveness; should match Gallery post-pan-y)
2. Full 12-coin Tray vertical scroll on phone (verify pan-y does not block page scroll)

**Residual risks needing CI/on-device confirmation:**
- B1 Linux CI: guard test on ubuntu-latest (code correct, environment verification)
- B3 on-device: Tray scroll with full drawer (jsdom cannot model touch-action)
- B4 A→B consumer: no production surface does direct swap yet (watcher-loop risk low; test guards)

### QA Clearance

**Test counts:**
- Livia full suite: 1264 PASS
- Brutus targeted basis: 218 new / 1264 full
- Maximus narrower basis: 216 (8-file scope)

**All suites green:**
- Guard: 3/3 ✅
- useSwipeGesture primitive: 42/42 ✅
- Tray characterization: 24/24 (+2 B3 tests) ✅
- useSettingsProfile: 8/8 ✅
- SettingsAppearanceSection: 10/10 ✅
- SettingsAccountSection: 10/10 ✅
- useCoinDetailSwipeNav: 88/88 (R1-R9 reliability) ✅
- SwipeGallery characterization: 31/31 (direction corrected) ✅
- Mounted call-site: 2/2 ✅
- **Full frontend: 1264/1264** ✅
- Type-check: 0 errors ✅
- Build: clean ✅

**Defects fixed:**
- G4 (pointercancel committed): → spring-back ✅
- T3 (pointercancel committed): → spring-back ✅
- T4 (scroll blocked): → pan-y ✅

**Brutus verdict:** APPROVE (production behavior verified)

### Release Status

✅ All B1-B5 blocks cleared (Livia)
✅ Full QA: 1264/1264 PASS
✅ Maximus CONDITIONAL APPROVE (B1 on CI, B3 on device)
✅ Brutus APPROVE
✅ Beta validation authorized
⏳ New PR held unmerged
⏳ Merge gate: Brian on-device iOS/Android (detail swipe feel + 12-coin Tray scroll)

## Previous Focus (Archived)

**PWA Account-Setting Preference (Released to Beta):** Account-wide `pwaSwipeNavEnabled` (default false), Settings → Account toggle, confirmed save, backend/frontend/docs complete, 1149 frontend tests green, Maximus APPROVE, mounted call-site gate satisfied, on-device evaluation remains main blocker.

**Experimental PWA Coin-Detail Swipe Navigation (Released to Beta):** 8-stop menu, Sell/Copy excluded, 68 targeted + 1122 full tests green, Maximus APPROVE beta-only, device evaluation awaited.

**Feature 356 Value History Remediation (Merged to Beta):** Journal bloat eliminated, tag suggestions restored, silent-failure bug fixed, all blocks cleared.

**Security Remediation (PYSEC-2026-3721):** Pip 26.2.1 lockfile, runtime system pip removed, CI regression guard, all blocks cleared.
