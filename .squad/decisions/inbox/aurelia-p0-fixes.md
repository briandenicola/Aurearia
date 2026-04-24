# P0 Fixes — Aurelia (2025-07-22)

## Decision: Admin Route Guard

**What:** Added `requiresAdmin: true` meta to the `/admin` route and a `beforeEach` guard that checks `auth.isAdmin`. Non-admin users are redirected to `/` (collection page).

**Why:** The admin page was only hidden via UI (conditional nav link in `App.vue`), but the route itself was accessible to any authenticated user by navigating directly to `/admin`.

**Impact:** `src/web/src/router/index.ts` — 2 small additions (meta flag + guard check). No breaking changes; non-admin users simply get redirected.

## Observation: v-html XSS Already Mitigated

All 4 `v-html` bindings in the frontend are already wrapped with `DOMPurify.sanitize()`:
- `CoinSearchChat.vue` → `formatMessage()` in `useCoinSearchChat.ts` (strict tag/attr allowlist)
- `CoinDetailPage.vue` → computed properties using `DOMPurify.sanitize(md.render(...))`

DOMPurify `^3.4.1` and `@types/dompurify` `^3.2.0` are installed. No additional work was required. This item from the code review backlog can be closed.
