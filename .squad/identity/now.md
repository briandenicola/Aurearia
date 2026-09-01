---
updated_at: 2026-09-01T10:55:49Z
focus_area: Pinned Sets Implementation (Approved, Ready for Merge)
active_issues: []
handoff_commit: pending
---

# What We're Focused On

**Pinned Sets in the Sidebar Sets Submenu (Approved, Ready for Merge)**

## Status Summary

Design review completed and accepted by Maximus. Backend implementation (Cassius) and frontend implementation (Aurelia) complete. Acceptance verification (Brutus) passed all 15 criteria with APPROVE verdict. No blockers. Ready for merge to publication branch and CI.

**Key accomplishment:** Design-time GORM concern about camelCase update-map keys was independently investigated and disproven by Brutus through empirical testing on pristine HEAD. Existing keys (`setType`, `creationMode`, `isPublic`, `parentSetId`, `targetCompletionDate`) persist correctly. No false defect ticket filed; design record to be amended with finding.

## Pinned Sets Implementation

### Feature Summary

Users can pin up to 5 of their coin sets directly in the sidebar Sets submenu for quick access. Pinned sets appear below `My Sets` and `Emperors`, ordered by pin time, truncated with tooltips.

### Design Decisions Accepted

| Decision | Outcome |
|---|---|
| D1: Persistence | `PinnedAt *time.Time` column on `CoinSet`, derived pinned state, additive AutoMigrate |
| D2: API | Reuse `PUT /sets/:id` with `pinned?` field, snake_case `pinned_at` key for GORM |
| D3: Cap | 5-set limit enforced server-side in service |
| D4: UX | Single icon affordance on detail page header (PWA + desktop), `Pin`/`PinOff` icon, gold when pinned |
| D5: Sidebar | Pinned appended to existing `Sets.children`, order by pin-time then name, truncated with title |
| D6: Data Flow | Singleton `usePinnedSets.ts` composable, refresh on mount, no polling |
| D7: Risks | 8 mitigated: R1 guard-tested (GORM key round-trip), R2–R8 architected |

### Acceptance Criteria: 15/15 Pass

**Backend (A1–A7):**
- ✅ A1: `PinnedAt *time.Time`, additive AutoMigrate, no backfill
- ✅ A2: `{"pinned":true}` → 200 + `pinned:true` + non-null `pinnedAt`
- ✅ A3: `{"pinned":false}` → 200, `pinned:false`, null `pinnedAt`
- ✅ A4: Double-pin preserves `pinnedAt` byte-identical
- ✅ A5: 6th pin → 400 `you can pin up to 5 sets`, set stays unpinned
- ✅ A6: Foreign set → 404, no cross-user leakage
- ✅ A7: `GET /sets` includes `pinned` + `pinnedAt` on every summary

**Frontend (A8–A14):**
- ✅ A8: Sidebar shows pinned after static children, pin-time order, truncated + title
- ✅ A9: Zero pinned → submenu byte-identical to pre-feature rendering
- ✅ A10: Detail header icon: `Pin` unpinned, `PinOff` gold when pinned, `aria-pressed`
- ✅ A11: PWA tap navigates to `/sets/:id` and closes drawer
- ✅ A12: Sets parent collapsed on load with pins present
- ✅ A13: Unpinning last pin removes it reactively
- ✅ A14: Logout clears (in-tab login gap is pre-existing pattern)

**Quality Gates (A15):**
- ✅ A15: All gates green — `go vet/test`, `vue-tsc --build --force`, `npm run build`, `vitest run 1322/1322`; guard files byte-untouched

### Implementation Status

- **Backend files:** Model + Service + Repository + Handler + Tests + Docs — all complete and green
- **Frontend files:** Composable (NEW) + App sidebar merge + SetDetailPage affordance + Tests — all complete and green
- **Test coverage:** 9 backend + 16 frontend tests (mounted DOM assertions, full suite green)
- **Constitution:** ✅ Principle I, IV, V, VI; ✅ §17 Quality Gate, §21 DoD

### GORM camelCase Field Investigation (Design D2 Finding)

**Original concern:** Design cautiously flagged existing camelCase keys in `PUT /sets/:id` (`setType`, `creationMode`, `isPublic`, `parentSetId`, `targetCompletionDate`) as suspect, hypothesizing they might not persist via GORM's `Schema.LookUpField` resolution.

**Brutus's independent verification:** Drove real `SetRepository.Update` round-trips against pristine HEAD (gorm v1.31.2, glebarez/sqlite v1.11.0).

| Key | Tested Value | Result |
|---|---|---|
| `setType` | `"goal"` | ✅ PERSISTED |
| `creationMode` | `"manual"` | ✅ PERSISTED |
| `isPublic` | `true` | ✅ PERSISTED |
| `parentSetId` | `7` | ✅ PERSISTED |
| `targetCompletionDate` | `"2030-05-04T00:00:00Z"` | ✅ PERSISTED |
| **Control: `totallyBogusColumn`** | any | ❌ **LOST** (proves probe detects unresolvable keys) |

**Conclusion:** GORM resolves JSON-tag camelCase names to Go struct fields via case-insensitive matching; no defect exists. Cassius's use of snake_case `pinned_at` is correct and is guarded by `TestSetRepository_Update_PinnedAtRoundTrip` (R1). **No follow-up ticket filed** — the false premise was corrected before propagation. **Recommendation:** Amend §D2 in the design record to document this finding.

### Non-Blocking Observations

1. **A9 wording vs. authorized CSS:** Design added `min-w-0`/`truncate`/`:title` to submenu links — visually inert for short labels, spec-additive. Documented.
2. **In-tab login refresh gap:** `refreshPinnedSets()` runs only in `onMounted` auth block — in-tab re-login skips remount. Pre-existing pattern (notifications identical). Recommend separate ticket if desired.
3. **Non-boolean `pinned` silently ignored:** `{"pinned":"yes"}` returns 200 with no state change. Fail-closed, harmless.

## Release Path

✅ Design review complete (Maximus ACCEPTED)
✅ Backend complete (Cassius)
✅ Frontend complete (Aurelia)
✅ Acceptance verification complete (Brutus APPROVE, all A1–A15 pass)
✅ Constitution compliance verified
⏳ Coordinator merges decisions + orchestration logs
⏳ Feature branch merge to publication/CI (no additional gates)

**Status: APPROVED — Ready for Merge**

## Previous Focus (Archived)

**Swipe Reliability & Shared Engine Correction (Livia B1-B5 Cleared, Maximus CONDITIONAL APPROVE):** Detail swipe migrated to shared `useSwipeGesture` primitive with unified direction semantics. Livia independently cleared all five B1-B5 blocks under Strict Lockout. Maximus CONDITIONAL APPROVE (B2-B5 clear; B1 clears on green ubuntu CI). Beta-only validation authorized. Merge gate = Brian on-device iOS/Android evaluation (detail swipe feel + full 12-coin Tray scroll).

**PWA Account-Setting Preference & Experimental Swipe Navigation:** Released to beta. Maximus APPROVE, mounted call-site gate satisfied, on-device evaluation remains main blocker.

**Security Remediation (PYSEC-2026-3721):** Pip 26.2.1 lockfile, runtime system pip removed, CI regression guard, all blocks cleared.

**Feature 356 Value History Remediation:** Journal bloat eliminated, tag suggestions restored, silent-failure bug fixed, all blocks cleared.

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
