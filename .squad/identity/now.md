---
updated_at: 2026-08-21T12:09:54Z
focus_area: PWA swipe navigation account-wide preference (pwaSwipeNavEnabled, default false, Settings → Account, beta-only, mounted call-site integration tests satisfied, on-device evaluation remains main-gate blocker)
active_issues: []
handoff_commit: pending
---

# What We're Focused On

**PWA Swipe Navigation Account-Wide Preference (Approved for Beta, Mounted Gate Satisfied)**

## Status Summary

Account-wide `pwaSwipeNavEnabled` preference implemented and approved for beta release. Single boolean column on users table (default false); synchronized across all devices via server-side persistence. Settings → Account toggle (always visible; PWA-only scope in description). Confirmed save model. Listeners attach/detach reactively in `useCoinDetailSwipeNav` composable; gates at feature level, not call sites. Mounted call-site integration tests gate satisfied. Backend, frontend, OpenAPI, and docs all cleared (Maximus APPROVE, Brutus APPROVE). Ready for beta-only push. Main merge pending recorded installed-PWA device evaluation (on-device gesture feel, iOS/Android).

## Experimental PWA Coin-Detail Swipe Navigation Status (Released to Beta)

8-stop canonical menu swipe navigation (Overview → Shipment → Journal → Health → Notes → Actions → Analysis → Valuation) shipped to beta with this preference gating feature enabled. Sell/Copy excluded. All gates green; no blocks from earlier swipe experiment batch.

## PWA Account Preference Details

### Design & Contract

**Storage:** Single `pwa_swipe_nav_enabled` column on `users` table. Explicit GORM column name to pin acronym handling. Default false; AutoMigrate adds `NOT NULL DEFAULT 0` on existing rows (no backfill code).

**API:** Reuse existing profile endpoints.
- `GET /auth/me` — includes `pwaSwipeNavEnabled` in response
- `PUT /user/profile` — accepts `pwaSwipeNavEnabled` in request body (pointer semantics: omitted = unchanged)
- `writeAuthResponse` funnel — login, register, refresh, WebAuthn, OIDC all carry field
- Ownership: authenticated context only; request body never carries user id

**Settings UI:** Settings → Account, using existing toggle pattern.
- Visibility: Always shown (browser and installed PWA) — PWA-only constraint in description
- Save model: Confirmed (not optimistic)
- Failure behavior: auth store and localStorage unchanged; retry available

**Runtime Gate:** Inside `useCoinDetailSwipeNav` composable.
- Condition: `isPwa && auth.user?.pwaSwipeNavEnabled === true` (fail closed)
- Lifecycle: attach/detach driven by onMounted, onUnmounted, and reactive watch
- Live toggle: no remount; listeners update reactively
- Logout: detaches immediately (user.value = null)
- Account switch: applies new account's value immediately
- Modal gate (enabled option): independent; unchanged

**Documentation:** PWA Guide and Features documents clarified.
- Account tab table added to PWA Guide (was blocking on table rendering, now fixed)
- Features/PWA Features reorganized to Account heading (was under Appearance)
- `features.md` Settings section updated

### Implementation Details

**Backend (Cassius):**
- `PWASwipeNavEnabled` field + explicit `column:pwa_swipe_nav_enabled` tag
- Response types: `GetMe`, `UpdateProfile`, auth payloads
- 6 new targeted tests: migration, defaults, persistence, ownership, auth response
- `task openapi` regenerated 4 artifact files

**Frontend (Aurelia):**
- Types: `User`, `UserInfo`, `updateProfile` endpoint
- Composable: `useSettingsProfile` (ref, save payload, response sync, localStorage)
- UI: `SettingsAccountSection` toggle row (after Emperor Tracker, before Save)
- Startup: `App.vue` getMe() sync copies `pwaSwipeNavEnabled`
- Gate: `useCoinDetailSwipeNav` refactored (attach/detach lifecycle, preference watch, no-op guard)
- 17 new frontend tests + 2 mounted call-site integration test files

**Docs Revision (Maximus cycle):**
- `docs/pwa-guide.md` B1 → B1a: fixed Account Tab table (header, delimiter, rendering)
- `docs/features/pwa-features.md` B1: moved Swipe Navigation to Account heading
- `docs/features.md`: Settings section updated

### QA Results (Brutus)

✅ Go vet: PASS
✅ Go test (all 12 packages): PASS
✅ OpenAPI drift: PASS
✅ Frontend targeted (23 new tests): PASS
✅ Frontend full suite (1149 tests): PASS
✅ Type-check: PASS
✅ Production build: PASS

**All 40+ acceptance criteria verified:** defaults, persistence, ownership, API completeness, UI visibility/save model, reactive lifecycle, logout/switch, fail-closed, mounted call-site binding (highest-risk miss), App.vue hydration, gesture regression.

### Review Status

**Maximus:** APPROVED for beta only. All code and docs cleared. Mounted call-site integration tests satisfied (refs bound in components, tests assert HTMLElement at mount). Recorded installed-PWA device evaluation (on-device gesture feel, iOS/Android, real PWA install) remains sole outstanding main-gate blocker.

**Brutus:** APPROVE. Zero blocks; all gates green.

### Release Gates

✅ Backend and frontend implementation
✅ OpenAPI/docs/test coverage
✅ 1149 frontend tests + 12 Go packages
✅ Mounted call-site binding tests (integration test gate satisfied)
✅ Maximus APPROVE for beta
✅ Brutus APPROVE

⏳ Beta: Ready to push
⏳ Main: Blocked on recorded installed-PWA device evaluation only

## Previous Focus (Archived)

**Experimental PWA Coin-Detail Swipe Navigation (Released to Beta):** 8-stop menu (Overview → Shipment → Journal → Health → Notes → Actions → Analysis → Valuation), Sell/Copy excluded. Gestures: 64 px threshold, 2:1 axis dominance, 10 px axis-lock, 24 px edge guard, no wrap, passive listeners. 68 targeted tests + 1122 full suite green. Maximus APPROVE for beta only. Device evaluation awaited before main merge.

**Feature 356 Value History Remediation (Merged to Beta):** Journal bloat eliminated (scheduled AI estimates removed from activity log), tag suggestions restored (ruler-weight gate fixed, medium-confidence thematic tags surface), silent-failure bug fixed (error states distinct from empty results). All reviewer blocks cleared.

**Security Remediation (PYSEC-2026-3721):** Pip 26.2.1 lockfile update (Aquila), system pip removal from runtime image (Cassius), regression guard CI job with `/health` smoke test (Brutus). All blocks cleared; release-ready pending fresh beta gates.
