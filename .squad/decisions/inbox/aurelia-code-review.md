### 2026-04-24: Frontend Code Quality Review
**By:** Aurelia (Frontend Dev)

## Grades

| Area | Grade | Notes |
|------|-------|-------|
| Component Quality | B | Good Composition API usage across the board. Several components exceed 400+ lines and should be split. Props/emits are well-typed in most places. |
| TypeScript Usage | B+ | Very few `any` casts (only in `sanitizeCoin`). Good use of interfaces and type imports. Minor gaps: some `ref<HTMLElement>()` missing nullable init, ad-hoc `_retry` property on Axios config. |
| State Management | B- | Pinia stores are lean but too lean — `coins.ts` has no error state, `auth.ts` can drift from localStorage after token refresh. Mixed concerns in filter state. |
| API Integration | B+ | Token refresh queue pattern is solid. `sanitizeCoin` normalizer is a nice touch. Gap: refresh updates localStorage but not the Pinia auth store, causing stale UI state. |
| Accessibility | D+ | Minimal ARIA attributes (only 4 files use `aria-*` or `role=`). Clickable divs without keyboard support. No focus traps on modals/drawers. Autocomplete lacks listbox/option roles. Canvas crop has no keyboard alternative. |
| Responsive Design | B- | PWA-first design is good. Some page components are very dense (AdminPage, SettingsPage) and will be cramped on mobile. Tables and wide grids lack horizontal scroll handling. |
| PWA Quality | C+ | Plugin config is present with runtime caching. But: referenced icons (pwa-192x192.png, pwa-512x512.png) are missing from `public/`, reducing installability. No offline fallback UI. No update prompt UX. |
| UX Polish | B | Loading and empty states exist on most pages. Error feedback is generally present. Some setTimeout-based message clears lack cleanup. `v-html` used for AI content without visible sanitization. |

**Overall: B-** — Solid foundation with good TypeScript discipline. Main gaps are accessibility, PWA completeness, and memory leak hygiene.

---

## Issues Found

### 1. Security: `v-html` XSS risk
**Files:** `CoinSearchChat.vue:31`, `CoinDetailPage.vue:282,290,294`
AI-generated content is rendered with `v-html`. If `formatMessage()` or the markdown renderer don't sanitize, this is a direct XSS vector. Need to verify DOMPurify or equivalent is in the pipeline.

### 2. Memory Leaks: Uncleared timers
**Files with `setTimeout`/`setInterval` but no corresponding cleanup:**
- `AdminPage.vue` — 8 timer calls, only partial cleanup
- `SwipeGallery.vue` — `setTimeout` not tracked on unmount
- `ResetPasswordModal.vue` — `setTimeout(() => emit('close'), 1200)` not cleared
- `useCollectionFilters.ts` — debounce timer never cleared
- `useNotifications.ts` — global `pollTimer` with no auto-cleanup
- `useAdminConfig.ts` — `setTimeout` not cleared
- `useCoinSearchChat.ts` — `setTimeout` at line ~153 not cleared
- `useImageProcessor.ts` — `searchTimeout` not cleared
- `FollowersPage.vue` — debounced search timer, no `onUnmounted`
- `WishlistPage.vue` — `dismissTimer` not cleared on unmount

### 3. Memory Leaks: Object URL leaks
**Files:** `CoinForm.vue` — 3 `createObjectURL` calls, 0 `revokeObjectURL` calls. Preview blob URLs will accumulate until page navigation.

### 4. Auth Store Drift After Token Refresh
**Files:** `api/client.ts:58-60`, `stores/auth.ts`
Token refresh interceptor updates `localStorage` but never syncs the Pinia `auth` store. This means `auth.isAuthenticated` and `auth.user` can be stale after a silent refresh, potentially causing UI flicker or false logouts.

### 5. Router: No Admin Role Guard
**File:** `router/index.ts:138-143`
The `beforeEach` guard only checks `requiresAuth`. The `/admin` route has no role-based protection — any authenticated user can navigate directly to it. The admin link is UI-hidden in `App.vue` but the route is fully accessible.

### 6. Accessibility: Clickable Divs Without Keyboard Support
**Files:** `CoinCard.vue` (whole-card click), `SwipeGallery.vue` (drag-only nav), `CoinSearchChat.vue` (drawer)
Interactive elements use `@click` on `<div>` without `tabindex`, `role="button"`, or `@keydown.enter` handlers. Screen reader and keyboard-only users cannot interact.

### 7. Accessibility: Autocomplete Missing ARIA
**File:** `AutocompleteInput.vue:13-22`
Dropdown lacks `role="listbox"`, options lack `role="option"`, no `aria-activedescendant`. Keyboard navigation exists but is invisible to screen readers.

### 8. Accessibility: No Focus Traps on Modals
**Files:** `PurchaseModal.vue`, `SellModal.vue`, `ImportLotModal.vue`, `BulkTagPickerModal.vue`, `ResetPasswordModal.vue`, `CoinSearchChat.vue` (drawer)
Modals don't trap focus — Tab can escape to background content. No Escape key handler on most.

### 9. Large Components Need Splitting
| Component | Lines | Recommendation |
|-----------|-------|----------------|
| `AdminPage.vue` | 1378 | Split into section sub-components (users, system, logs, settings) |
| `SettingsPage.vue` | 1371 | Extract profile, security, appearance, and tools into sub-components |
| `CoinDetailPage.vue` | 1242 | Extract AI analysis panel, image gallery, activity journal sections |
| `StatsPage.vue` | 807 | Extract chart sections |
| `FollowerCoinDetailPage.vue` | 800 | Extract repeated detail sections |
| `ImageProcessor.vue` | 479 | Extract crop controls vs. processing logic |
| `CoinCard.vue` | 443 | Borderline — monitor |

### 10. PWA: Missing Icons
**Config:** `vite.config.ts` references `pwa-192x192.png` and `pwa-512x512.png` in the manifest, but these files don't exist in `public/`. This breaks PWA installability on mobile devices.

### 11. PWA: No Update Prompt
**Config:** `registerType: 'autoUpdate'` silently swaps the service worker. Users get no notification that new content is available, which can cause confusion if cached UI doesn't match API changes.

### 12. Coins Store: No Error State
**File:** `stores/coins.ts`
`fetchCoins`/`fetchCoin` have `try/finally` but no `catch` — errors bubble unhandled. No `error` ref in the store means pages can't reactively show error states from the store level.

### 13. Form Validation Gaps
**Files:** `RegisterPage.vue`, `SettingsPage.vue`, `ShowcasesPage.vue`, `CalendarPage.vue`
Password confirmation is checked in script only, not via HTML constraint. Most forms rely solely on `required` — no length/pattern validation for passwords, no duplicate-name checks.

### 14. Index-as-Key in Lists
**Files:** `CoinSearchChat.vue:26`, `CoinSuggestionGrid.vue:3,24`, `CoinShowResultsGrid.vue:3,17`, `AdminLogsSection.vue:33`
Using array index as `:key` in `v-for` causes incorrect DOM reuse when items are reordered or removed.

---

## Backlog Items (Frontend)

| # | Title | Priority | Effort | Description |
|---|-------|----------|--------|-------------|
| 1 | Sanitize v-html AI content | P0 | S | Add DOMPurify (or verify existing sanitization) to all `v-html` bindings in `CoinSearchChat.vue` and `CoinDetailPage.vue`. XSS risk from AI-generated content. |
| 2 | Add admin role guard to router | P0 | S | Add `requiresAdmin` meta to `/admin` route and enforce role check in `router.beforeEach`. Currently any authenticated user can access admin pages directly. |
| 3 | Fix auth store drift after token refresh | P1 | S | After successful token refresh in `api/client.ts`, update the Pinia auth store (not just localStorage) so `auth.isAuthenticated` and reactive user data stay current. |
| 4 | Clear all timers on unmount | P1 | M | Audit all `setTimeout`/`setInterval` calls (15+ files) and ensure each is tracked and cleared in `onUnmounted` or `onBeforeUnmount`. Priority files: `AdminPage`, `SwipeGallery`, `useNotifications`, `useCollectionFilters`, `FollowersPage`. |
| 5 | Revoke object URLs in CoinForm | P1 | S | Add `URL.revokeObjectURL()` cleanup for image previews in `CoinForm.vue` to prevent memory leaks during multi-image workflows. |
| 6 | Add PWA icons to public/ | P1 | S | Generate and add `pwa-192x192.png` and `pwa-512x512.png` to `public/` directory so the PWA manifest is valid and the app is installable on mobile. |
| 7 | Add focus traps to modals | P2 | M | Implement focus trapping (e.g., `@vueuse/integrations` `useFocusTrap`) on all modal/drawer components. Add Escape key to close. Affects 6+ modal components. |
| 8 | Add ARIA to AutocompleteInput | P2 | S | Add `role="listbox"`, `role="option"`, `aria-activedescendant`, and `aria-expanded` to the autocomplete dropdown for screen reader compatibility. |
| 9 | Make clickable divs keyboard-accessible | P2 | M | Add `tabindex="0"`, `role="button"`, and `@keydown.enter`/`@keydown.space` to clickable `<div>` elements in `CoinCard`, `SwipeGallery`, and `CoinSearchChat` drawer. |
| 10 | Split AdminPage.vue | P2 | L | Extract the 1378-line `AdminPage.vue` into focused sub-components: users management, system settings, AI config, log viewer, and scheduler controls. |
| 11 | Split SettingsPage.vue | P2 | L | Extract the 1371-line `SettingsPage.vue` into sub-components: profile, security/password, appearance, and tools sections. |
| 12 | Split CoinDetailPage.vue | P2 | M | Extract the 1242-line `CoinDetailPage.vue` — AI analysis panel, image gallery, and activity journal should be separate components. |
| 13 | Add error state to coins store | P2 | S | Add an `error` ref to `stores/coins.ts` and populate it in `catch` blocks so pages can reactively display store-level errors. |
| 14 | Add PWA update prompt | P2 | S | Replace `registerType: 'autoUpdate'` with `'prompt'` and add a toast/banner prompting users to reload when a new service worker is available. |
| 15 | Replace index-as-key in v-for loops | P3 | S | Use unique item identifiers as `:key` in `CoinSearchChat`, `CoinSuggestionGrid`, `CoinShowResultsGrid`, and `AdminLogsSection` to prevent DOM reuse bugs. |
| 16 | Improve form validation | P3 | M | Add client-side password strength rules, confirmation matching via HTML constraints, and name uniqueness checks to registration, settings, and showcase forms. |
| 17 | Add responsive handling for dense pages | P3 | M | Add horizontal scroll wrappers for tables and card grids on `AdminPage`, `AuctionsPage`, and `FollowersPage` for small viewports. Review header `nowrap` on `FollowersPage`. |
| 18 | Remove `any` casts in sanitizeCoin | P3 | S | Replace `(clean as any)[field]` in `api/client.ts:106-107` with a properly typed approach using `Record<string, unknown>` or keyof mapping. |
| 19 | Redirect authenticated users from login | P3 | S | In `router.beforeEach`, redirect already-authenticated users away from `/login` and `/register` to the collection page. |
