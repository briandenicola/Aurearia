# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins frontend — Vue 3 / TypeScript / Pinia / Vite PWA
- **Architecture:** All API calls through `src/web/src/api/client.ts`; UI follows design tokens from `variables.css` and global classes from `main.css`.

## Core Context

**Durable patterns & learnings:**
- `<script setup lang="ts">` with strict nullable handling (`?.`, `??`); no emojis; lucide icons; dark theme; PWA/mobile support; design-token-only CSS
- Feature #219 coin detail: overview uses dual-side media, metadata table, section links; detail sections use `<h3>` with section spacing; tags use `.chip` sizing
- Camera/intake: 3-column capture slots, active-tile gold glow, circular focus overlay, camera controls aligned under slots (Camera/Images lucide icons)
- Modal structure from `FeaturedCoinModal`; composables expose cleanup functions; `onUnmounted()` cleanup mandatory
- **State management:** Module-level refs in composables do NOT reset on unmount — must explicitly reset in `onUnmounted()` or state leaks across navigation (learned from `bulkSelectActive` hiding agent FAB indefinitely)
- **Composition API:** Interaction-gating state must never be module-level refs; prefer Pinia store with lifecycle or pass via props/emits
- **Non-passive touchmove gotcha:** A non-passive `touchmove` handler that calls `e.preventDefault()` MUST have a `touchcancel` reset handler paired with it, else OS/browser gesture hijacks leave the state stuck (learned from pull-to-refresh touch handler freezing UI)
- **History stack:** Prefer `router.back()` on child form save (pops form, returns to parent naturally). Parent pages with multiple subpages use absolute nav to list (`router.push('/')`) to avoid history pollution and incorrect back routing
- **Responsive tables:** When action buttons overflow, stack related data vertically in earlier columns; use `flex-shrink: 0` and `justify-content: flex-end` on action containers
- **Backend nullable FKs:** SQLite FKs may use `constraint:-` notation (no physical constraints) to avoid destructive rebuilds; frontend treats lookup fields as always-nullable

**Recent batch outcomes (2026-06-02 — 2026-06-03):**
- Camera Capture Modal: extracted into reusable `CameraCaptureModal.vue` (live preview, circular focus, permission handling, leak-free cleanup); `CoinActionsPanel.vue` now uses modal instead of native `<input capture>`
- Camera Permissions API pre-check: progressive enhancement for persisted grants (iOS PWA over HTTPS), shows actionable error on denial, fallback to `getUserMedia` for unsupported browsers
- Per-coin Value Trend Subpage: created `CoinDetailValuationPage.vue` mirroring coin detail subpage pattern; new route `/coin/:id/valuation` (coin-detail-valuation); added `'valuation'` section after analysis in coinDetailSections.ts; chart migrated from deleted `StatsCoinValueTrend.vue` (SVG polyline + circles, y-axis/date labels, `formatCurrency`, `getCoinValueHistory(coinId)`, seed from coin metadata); collection-wide trend stays on Stats page; npm type-check/build/lint all passed

**Architecture compliance:** All recent work follows Principle IV (Strict Typing), Principle V (Design Token System), Principle IX (UI/UX Consistency), Principle XIII (PWA/Mobile)

## Recent Updates

- **2026-06-01:** Tags UI final, AddCoinPage camera grid, Purchase metadata moved to Details table, Store prefix label for purchase location, free-text Rarity UI removed, Storage Location frontend integration, Settings tab reorganization (backups/API keys), bulk assign location UI
- **2026-06-01 (Learnings):** Fixed PWA tap-blocking (pull-to-refresh touchcancel leak) and agent FAB hidden bug (module-level state leak); documented back navigation pattern and responsive table overflow handling
- **2026-06-02:** Coin detail UI reordering (Inscription consolidation, section renames, metadata hierarchy); Settings tab split (Backups ↔ API Keys as separate tabs); Camera modal extraction; Camera permissions pre-check; Per-coin value trend subpage

## Learnings

- **2026-06-19:** PWA Workbox runtime caching must never CacheFirst private `/uploads/*` media. The legacy `coin-images` CacheStorage bucket is now a cleanup-only compatibility concern cleared from `src/web/src/stores/auth.ts` on logout and user switch; uploaded media URLs remain unchanged until backend authenticated media routes land.
- **2026-06-18:** WebAuthn login begin responses from the Go API are shaped as `{ options: { publicKey: ... }, username }`, matching go-webauthn's browser options wrapper. The frontend biometric login path must unwrap `options.publicKey` before converting `challenge` and `allowCredentials` to `ArrayBuffer`; using `options.challenge` directly leaves iPhone PWA/Safari with missing challenge data before `navigator.credentials.get()` can run. `src/web/src/stores/auth.ts` now accepts both nested and legacy flat shapes, trims the ceremony username, uses the server-returned username for finish, and tests the missing-challenge guard in `src/web/src/stores/__tests__/auth.test.ts`.
- **2026-06-18:** Mint Map must not rely on the shared collection store page cache, because the store may contain only the current paginated page. `src/web/src/pages/MintMapPage.vue` now fetches active collection coins directly through `getCoins()` page-by-page (`wishlist:false`, `sold:false`) before grouping with `src/web/src/utils/mintMap.ts`; regression coverage lives in `src/web/src/pages/__tests__/MintMapPage.test.ts`.
- **2026-06-09:** F013 browser workflow smoke uses Playwright under `src/web/e2e/` with route-level API mocks and authenticated localStorage setup, so login, manual add, and edit-one-field coverage can run without a live backend or production data.
- **2026-06-09:** F013 frontend fixture data now lives in `src/web/src/test/fixtures/`. Coin builders return cloned `Coin` objects so component/browser tests can safely override nested images, references, tags, sets, and storage-location data without mutating shared golden fixtures.
- **2026-06-09:** F013 add/edit workflow inventory: manual add and intake commit use the safer allowlisted `buildCoinPayload()`, but `EditCoinPage` still sends the whole loaded `Partial<Coin>` through `updateCoin(form.id!, form)`. `sanitizeCoin()` normalizes nullable scalars, strips `storageLocation`, defaults `currentValue`, and formats date-only values, but does not strip read-side fields such as `images`, `tags`, `sets`, ids, owner/status fields, or timestamps. F013 browser coverage should assert both user-visible results and request payloads, with tags/sets treated as detail-page association workflows rather than edit-form fields.
- **2026-06-01 (CORRECTED):** The `showChat` defensive-reset theory below was WRONG and the fix was reverted. `App.vue.onMounted` runs once at app boot when `showChat` is already `false`, so `showChat.value = false` there is a no-op and cannot fix an intermittent mid-session freeze. The reported tap-blocking bug ("can't tap searched coin, can't rotate image") was actually caused by the **pull-to-refresh touch handler** in `src/web/src/composables/usePullToRefresh.ts`: it set `pulling=true` on touchstart but only cleared it on `touchend`. When the OS/browser hijacks a gesture (notification, multitouch, system back-swipe — common in heavy PWA use) `touchcancel` fires instead of `touchend`, leaving `pulling=true`; every later tap at scroll-top then hit a non-passive `touchmove` that called `e.preventDefault()`, which suppresses the synthesized click on mobile — so taps did nothing while the screen looked completely normal. Real fix (commit `9f906bf`): add a `touchcancel` handler that resets state, plus an `ENGAGE_SLOP` so `preventDefault()` only fires on a real pull, never on taps. Lesson: a non-passive `touchmove` that calls `preventDefault()` MUST be paired with a `touchcancel` reset and must never `preventDefault()` on a stationary tap.
- **2026-06-01:** Module-level refs in composables do NOT reset on component unmount. When a module-level ref (exported from a composable like `useBulkSelect.ts`) gates global UI state or interaction behavior, the owning component MUST explicitly reset the ref in `onUnmounted()` or the state will leak across navigation. In CollectionPage, `bulkSelectActive` (module-level) stayed true after unmount while `selectMode` (local) was destroyed, causing the agent FAB in App.vue to stay hidden indefinitely. Fix: add `onUnmounted()` hook to reset module-level state, and defensive `onMounted()` reset to ensure clean state on every mount. Alternative patterns: move state to Pinia store with proper lifecycle, or avoid module-level refs entirely for interaction-gating state—pass via props/emits instead.
- **2026-06-01:** Admin table layout overflow fix pattern: when action buttons overflow on narrow viewports, stack related data vertically in earlier columns rather than letting the table stretch horizontally. In `AdminCatalogsSection.vue`, moved the era pill below the catalog code in the same cell (flex column with `gap: 0.35rem` and `align-items: flex-start`) to free up horizontal space. Action buttons use `display: flex` with `flex-shrink: 0` and `justify-content: flex-end` to ensure they stay right-aligned and never overflow the boundary. This pattern keeps tables responsive without sacrificing action button visibility.
- **2026-06-01:** Free-text Rarity/RIC UI removed in favor of the structured Catalog References section. Removed the Details metadata row from `src/web/src/composables/useCoinDetailMetadataRows.ts`, the legacy info-grid card from `src/web/src/components/coin/CoinInfoGrid.vue`, and the Rarity Rating (RIC) input from `src/web/src/components/CoinForm.vue`; data plumbing remains intact.
- **2026-06-01:** Storage Location frontend integration completed. Added `StorageLocation` types and API client CRUD methods (`getStorageLocations`, `createStorageLocation`, `updateStorageLocation`, `deleteStorageLocation`) in `src/web/src/api/client.ts`; `sanitizeCoin()` now normalizes `storageLocationId` and strips read-only `storageLocation`. Settings → Data now shows a two-column lookup manager with Tags and Storage Locations side by side in `SettingsDataSection.vue`; storage-location delete surfaces backend 409 conflict messages so users know to reassign coins first. `CoinForm.vue` loads `/storage-locations` and binds a single-select “Storage Location” dropdown with a “None” option; `useCoinDetailMetadataRows.ts` displays the chosen location as a Details row with `coin.storageLocation?.name ?? '—'`. Build and lint pass; full `npm test` remains blocked by pre-existing design-token budget failures unchanged from HEAD.
- **2026-06-01:** Settings reorganization completed. Added `src/web/src/components/settings/SettingsBackupsSection.vue` for collection export/PDF/import backups plus API key generation/revoke flows; moved `loadApiKeys()` exposure there. Settings now has tab id `backups` labeled “Backups & Keys” with the Archive icon, and the Data tab now contains only Tags + Storage Locations metadata management.
- **2026-06-01:** Bulk assign location UI completed. Created `BulkLocationPickerModal.vue` (mirroring `BulkTagPickerModal.vue`) with "No location" clear option that emits `null`. Extended `bulkAction()` client signature to accept `opts?: { tagId?: number; storageLocationId?: number | null }` instead of a single `tagId` parameter, maintaining backward compatibility with existing call sites. Updated `BulkActionBar.vue` to add "Assign Location" button with `MapPin` icon emitting `location` event. Wired up `CollectionPage.vue` to load storage locations on mount, handle `@location` event, render `BulkLocationPickerModal`, and call `bulkAssignLocation(locationId)` which posts `{ coinIds, action: 'assign-location', storageLocationId }` to `/coins/bulk`. Build, type-check, and lint all pass (no new warnings).

- **2026-06-01:** Backend storage-location migration convention: nullable `Coin` lookup FKs may exist without physical SQLite constraints (`constraint:-`) to avoid destructive rebuilds; frontend should continue treating `storageLocationId` as nullable and rely on API validation/errors.

- **2026-06-09:** F013 Frontend Fixture Batch Completed
   - Implemented 10 golden coin fixtures covering all lifecycle states (Roman, Greek, Byzantine, wishlist, sold, private, tagged, set-member, storage-location, image-heavy, valued, references, custom era)
   - Fixtures live in `src/web/src/test/fixtures/coins.ts` with public export index
   - Task T015 marked complete
   - Orchestration log: `.squad/orchestration-log/2026-06-09T12-51-39Z-aurelia.md`

- **2026-06-10:** Coin of the Day Pushover Public URL Configuration Revision
   - Cassius initially implemented with relative `/coin/{coinID}` links; Brutus blocked (relative URLs not usable outside app context)
   - Added `PublicAppURL` admin setting in System tab (`AdminSystemSection.vue`) with validation and frontend type (`src/web/src/types/index.ts`)
   - Coin of the Day Pushover links now build as absolute URLs (trim trailing slashes, join host + path) when setting is configured; omit link entirely when blank/invalid
   - In-app notification behavior unchanged (in-app notification uses `ReferenceID = FeaturedCoin.ID` for modal)
   - Tests pass: `npm run type-check`, `npm run build` ✅
   - Brutus cleared BLOCK. Decision merged to `decisions.md`
   - Orchestration log: `.squad/orchestration-log/2026-06-10T20-31-52Z-aurelia.md`

- **2026-06-18:** Mint Map Frontend 50-Coin Limit Fix Completed
   - Fixed MintMapPage.vue by replacing store page-cache use with direct `getCoins()` pagination loop (`limit=100` until total covered)
   - Regression test added: loads 120 active Rome coins across two API pages, asserts mapped count equals 120
   - Targeted tests passed: `npm.cmd run test -- MintMapPage.test.ts` (6 tests) ✅
   - Frontend build passed: `npm.cmd run build` ✅
   - Cassius backend analysis confirmed: no backend changes needed; existing pagination contract is correct
   - Brutus QA approved the fix with regression coverage validation
   - Decision merged to `decisions.md` as: "Mint Map Frontend 50-Coin Limit — Pagination Loop Implementation"
   - Orchestration log: `.squad/orchestration-log/2026-06-18T21-14-02Z-aurelia.md`


- **2026-06-01:** Legacy catalog reference migration UI added to Settings → Data. New bordered section with Database and RefreshCw icons from lucide-vue-next, explanatory text (non-destructive, keeps originals, records outcomes in journal), trigger button with loading state, and result counts grid showing Succeeded (gold accent), Skipped, Failed (amber). Client function `migrateLegacyReferences()` calls `POST /references/migrate-legacy` and returns `LegacyMigrationResult { succeeded, skipped, failed, message? }` type. Results display uses design tokens (`--accent-gold`, `--text-muted`, `--bg-input`, `--border-subtle`, `--radius-sm`) and mobile-responsive stacked layout. Build and lint pass (no new warnings).

- **2026-06-01:** Coin detail back navigation bug fixed. Root cause: EditCoinPage used `router.replace('/coin/:id')` after save, which Vue Router treated as a new Detail entry, leaving the stack as [Gallery, Detail_old, Detail_new]. Changed to `router.back()` which properly pops the Edit entry and returns to the original Detail, maintaining the correct Gallery → Detail → Back → Gallery flow. The pattern: when a child form/edit view saves and should return to parent, prefer `router.back()` over `router.replace()` to avoid polluting the history stack with duplicate parent entries.

- **2026-06-01:** Coin detail "Back" button changed to absolute gallery navigation. Renamed from "Back" to "Back to Gallery" and changed from `router.back()` to `router.push('/')` in `CoinDetailHeaderActions.vue`. This prevents history pollution when users navigate from Coin Details to subpages (journal, health, analysis, etc.), click "Back to Overview" (which pushes back to Detail), then click the Detail page's back button. Without absolute navigation, `router.back()` would incorrectly pop to the subpage instead of the gallery. Parent pages with multiple child subpages should use absolute nav to their list view, not `router.back()`.

- **2026-06-07:** Category, Era, and Material dropdowns made configurable. Created `AdminCoinPropertiesSection.vue` component with textarea inputs (one value per line) for each property type; added "Coin Properties" tab to Admin page (`properties`, positioned after System before Catalogs, `Settings2` icon). Created `utils/options.ts` with `parseOptionList()` (trims, dedupes, falls back to hardcoded defaults) and `formatOptionList()`. Created `useCoinOptions()` composable that loads settings from `/app-settings` and parses into reactive arrays with fallback to `CATEGORIES`, `COIN_ERAS`, `MATERIALS`. Updated `CoinForm.vue` and `AddCoinPage.vue` to load options from composable instead of constants. Backend settings keys: `CoinCategoryOptions`, `CoinEraOptions`, `CoinMaterialOptions` (newline-delimited strings). `vue-tsc --build` passes; first-option defaults safely handled with `??` fallbacks. Era dropdown keeps blank "Unspecified"/"Unknown" option in both forms.

- **2026-06-02:** Camera architecture learned from AddCoinPage integration. AddCoinPage camera is implemented inline with: live `<video>` preview, circular `.focus-overlay` with `.focus-ring` and `.focus-mask` radial gradient, `startCamera()` using `navigator.mediaDevices.getUserMedia({ video: { facingMode: { ideal: 'environment' } }, audio: false })`, permission error handling (NotAllowedError/NotFoundError), `stopCamera()` releasing all tracks, `computeCoverCropRect()` to handle object-fit:cover source rectangle, and `captureFromCamera()` drawing the cover-cropped region to canvas then `canvas.toBlob('image/jpeg', 0.92)`. The circular clipping itself is SERVER-SIDE: `uploadImage()` accepts an optional `circleClip` flag that the Go backend (`src/api/handlers/images.go`) uses to clip obverse/reverse images to circular transparent PNGs; card images are never clipped by the backend even if the flag is set. `CameraCaptureModal.vue` is now the reusable in-app camera component. CoinActionsPanel "Photo" button now opens this modal and passes `circleClip=true` for obverse/reverse, `false` for other types. AddCoinPage intentionally left unchanged this iteration to avoid refactoring the working multi-slot guided flow.

- **2026-06-02:** Camera permission pre-check added to `CameraCaptureModal.vue` using the Permissions API (`navigator.permissions.query({ name: 'camera' as PermissionName })`). Progressive enhancement: when permission state is `'denied'`, the modal shows "Camera access is blocked. Please enable it in your browser or site settings." and skips the `getUserMedia` call (no point — it would reject immediately). When `'granted'`, the camera opens directly with no re-prompt (the "persisted allow" UX Brian wanted). When `'prompt'` or the API is unavailable, falls through to the existing `getUserMedia` flow unchanged. Added `status.onchange` listener for runtime permission changes (cleared in `stopCamera()` for leak-free cleanup). TypeScript cast `'camera' as PermissionName` required since `'camera'` isn't in the default union; ESLint no-undef disabled on type-only lines. **Platform reality documented:** the app cannot force grant persistence — that's browser/OS-controlled. Installed PWA over HTTPS (Brian's setup) gives the best chance; iOS Safari tabs never persist. The pre-check is a UX enhancement where the browser DOES remember, not a persistence mechanism itself.

## Archived Sessions (2026-06-01)

Prior session work consolidated here for reference. See `.squad/decisions.md` for detailed records of:
- PWA tap-blocking bug root cause (pull-to-refresh touchcancel handling) + fix
- Legacy RIC→CoinReference migration UI
- Free-text Rarity UI removal (deprecated in favor of structured Catalog References)
- Catalog Registry admin frontend + CoinReference field rename (certainty → invoiceNumber)
- Agent FAB hidden bug fix (module-level state leak in `bulkSelectActive`)
- Storage location bulk assignment feature
- Coin detail back navigation fix (router.back() vs router.replace())
- Coin detail section reordering (Details → Inscription consolidation)

Key learnings from 2026-06-01 work moved to "## Learnings" and "## Core Context" sections above.
1. Move state into Pinia store with proper reset logic
2. Scope state locally and pass explicitly via props/emits (avoid module-level shared refs for interaction-gating state)
3. If module-level is required, document cleanup contract and enforce via lifecycle hooks

**Verification:** npm run type-check ✅, npm run build ✅, npm run lint ✅ (no new warnings).

## 2026-06-02 — Coin Detail UI Refinements (CoinDetailPage reordering + Inscription consolidation)

Completed three UI refinements to `CoinDetailPage.vue`:
1. **Heading disambiguation:** "Details" → "Additional Details" (above Activity Journal) to clarify these section links lead to detail subpages, not the core metadata table
2. **Section reordering:** Catalog References now precedes Tags (aligns with metadata hierarchy: numismatic identifiers before user classification)
3. **Inscription consolidation:** Merged separate "Inscriptions" + "Description" sections into a single "Inscription" block positioned at page top with:
   - Dual-side layout (Obverse | Reverse subsections via `.inscription-grid`)
   - Each side conditionally shows "Inscription:" line + description prose
   - Mobile-responsive stacking (2-column on desktop, 1-column on mobile)
   - CSS: all design tokens (`--bg-card`, `--border-subtle`, `--radius-sm`, `--text-*`); dead CSS removed

**Final Section Order:** Title → Inscription → Details (metadata) → Catalog References → Tags → Listing Status → Additional Details

**Verification:** npm run type-check ✅, npm run build ✅, npm run lint ✅ (no new warnings)

- **2026-06-09:** F013 Phase 4 Storage Location & Tags/Sets Workflows APPROVED
   - T022: Storage Location edit workflow (Playwright E2E test with route-level mocks, fixture-backed)
     - Navigate to coin detail → update storage location → save → verify summary updated
     - Uses golden fixture data; no live backend required
   - T023: Tags & Sets edit workflow (Playwright E2E test with route-level mocks, fixture-backed)
     - Navigate to coin detail → add/remove tags and set membership → save → verify detail page reflects changes
     - Uses golden fixture data; deterministic test coverage
   - Coordinator pre-validation: npm type-check (99 tests ✅), npm test (99 tests ✅), npm run test:browser (6 Playwright tests ✅), git diff --check (✅ no whitespace issues)
   - Maximus review: ✅ APPROVED — workflows deterministic and fixture-backed, mock extensions proportional, isolated from live backend, scope boundaries strict (storage/tags/sets only; no creep into T024–T028), Principles IV & VI satisfied
   - Completion mark justified: all builds, tests, lints pass; no architecture violations; ready for merge
   - Orchestration log: `.squad/orchestration-log/2026-06-09T13-32-43Z-maximus-t022-t023-approved.md`
   - F013 Phase 4 progress: T018–T021 (infrastructure) + T022–T023 (storage/tags workflows) = 6 of 11 tasks complete
   - Remaining: T024–T028 (edit validation, image upload, search/filter, mobile viewport, Taskfile, docs)

**Status:** Code change UNCOMMITTED, awaiting Brian's approval. Decision merged to `.squad/decisions.md`.

## 2026-06-02 — Settings Tab Split: Backups & API Keys Now Separate

Split the monolithic "Backups & Keys" Settings tab into two focused tabs: **Backups** (export/PDF/import) and **API Keys** (key generation/revocation).

### Implementation

**Component split:**
- Kept `SettingsBackupsSection.vue` with backups-only logic (export ZIP, PDF catalog, CSV/JSON import with template + guide links)
- Created new `SettingsApiKeysSection.vue` with full API key lifecycle (generate w/ scope selector, reveal box, list w/ capability badges, revoke)
- Removed `loadApiKeys()` exposure from SettingsBackupsSection; added it to SettingsApiKeysSection

**SettingsPage.vue wiring:**
- Added `apikeys` tab to both `baseTabs` array and PWA-admin `tabs` computed (keeping them in sync per existing pattern)
- Changed backups tab label from `'Backups & Keys'` → `'Backups'`
- Added `KeyRound` icon to `tabIcons` map for `apikeys`; kept `Archive` for `backups`
- Imported `SettingsApiKeysSection` and rendered it conditionally (`v-if="activeTab === 'apikeys'"`)
- Added `apiKeysSection` ref; moved `loadApiKeys()` call in `handleRefresh` from `backupsSection` → `apiKeysSection`
- `validTabIds` auto-derives from `baseTabs`, so deep-linking (`?tab=apikeys`) works without extra code

**Key pattern learned:** Settings tab structure requires dual maintenance: `baseTabs` array (desktop + general cases) AND `tabs` computed (PWA with admin case). Both must stay in sync for consistent rendering. `tabIcons` map provides icon-per-tab. Refs call exposed methods (`loadApiKeys()`) on mount/refresh. `validTabIds` auto-derives from `baseTabs` for deep-link validation.

### Verification
- npm run type-check ✅
- npm run build ✅ (no new chunks, clean output)
- npm run lint ✅ (0 errors, 5 pre-existing warnings unchanged from HEAD)

**Status:** Code change uncommitted, awaiting Brian's approval. Decision logged to `.squad/decisions/inbox/`.


## 2026-06-02 — Camera Capture Modal Extraction (Complete + Shipped)

Unified camera capture UX: Coin Details "Photo" button now uses same in-app circular camera modal as Add Coin flow.

**Implementation:**
- Extracted camera logic from AddCoinPage into reusable `CameraCaptureModal.vue`
- Live preview + circular focus overlay, cover-crop capture to JPEG (0.92 quality)
- Permission handling with friendly errors; lifecycle cleanup prevents stream leaks
- `CoinActionsPanel.vue` replaced native `<input capture>` with modal trigger
- Type-driven clipping: `circleClip=true` for obverse/reverse, `false` for other types

**Verification:** npm run type-check/build/lint ✅

**Commit:** 7a0eb40

**Cross-agent note:** Cassius added `Coin.CurrentValueUpdatedAt` field (API response now includes `currentValueUpdatedAt`). Available for future UX showing valuation timestamps.

---

## 2026-06-02 12:31:33Z: Camera Permissions Pre-Check — Decision Merged

**Status:** Merged to decisions.md

Added navigator.permissions.query pre-check to CameraCaptureModal.vue. Persisted grants (iOS PWA over HTTPS) now open camera instantly without re-prompt. Denied state shows actionable error before getUserMedia. Fallback to getUserMedia for unsupported browsers. Leak-free cleanup.

**Files:** `src/web/src/components/CameraCaptureModal.vue`. No backend changes.

**Cross-agent:** Cassius (backend) completed AI-coverage health fix; no frontend impact on camera permissions.

**Commit:** 17f75b4

## Learnings (continued)

- **2026-06-02:** Per-coin value trend subpage pattern: Follow existing coin detail subpage structure (`CoinDetailHealthPage.vue` → `CoinDetailValuationPage.vue`). New route: `/coin/:id/valuation`, added as `coin-detail-valuation` in router. New section type: `'valuation'` in `coinDetailSections.ts` with title "Value Trend", description "Estimated value over time", and positioned after analysis in `SECTION_ORDER`. Chart logic migrated from `StatsCoinValueTrend.vue`: SVG polyline + circles, y-axis labels, date labels, `formatCurrency`, `getCoinValueHistory(coinId)`, seed from `purchasePrice`/`purchaseDate` then append `CoinValueHistory` entries sorted by date. Empty state handling: wishlist/sold coins show "only available for active coins" message; < 2 data points shows "Not enough data points to chart. Run an AI estimate to start tracking." The collection-wide value trend (`StatsValueOverTime.vue`) stays on Stats page. Per-coin trend component (`StatsCoinValueTrend.vue` with dropdown) deleted after moving logic to valuation subpage.

- **2026-06-07:** Era/Category Options Frontend + Coin Lookup UX Proposal
  - **Admin UI:** Created `AdminCoinPropertiesSection.vue` with three textarea inputs (newline-delimited format, one value per line). Styling follows existing admin section patterns; `SaveChangesButton` for persistence.
  - **Parsing Utility:** `src/web/src/utils/options.ts` with `parseOptionList()` — trims whitespace, deduplicates, drops blank lines, falls back to hardcoded defaults if invalid/empty. 22-test spec covering parse/format/roundtrip/edge cases. Part of existing `npm run test:unit` suite.
  - **Composable:** `useCoinOptions()` loads settings from API (`GET /admin/settings`), exposes reactive arrays for `CoinForm` and `AddCoinPage` to populate dropdowns dynamically.
  - **Integration:** Modified `CoinForm.vue` and `AddCoinPage.vue` to load options on mount via composable; Era dropdown retains blank "Unspecified" option (UI-only, not persisted).
  - **Type-Safety Blocker (QA finding):** `AdminPage.vue` lines 93–95 use `v-model` binding that conflicts with explicit prop binding (lines 96–98); both patterns cannot coexist in Vue. Fix: Remove `v-model` lines, keep nullish-coalesced props per Principle IV. Validation gate: `npm run type-check` must pass before merge.
  - **Coin Lookup UX Design:** New `/lookup` route (nav item between "Add Coin" and "Wishlist"). Single-page flow: Capture State (full-screen PWA camera with circular focus overlay) → Analyzing State (loading overlay) → Results State (read-only draft display + auto-triggered Numista search + quick actions). Quick actions: "Retake Photo", "Add to Collection" (navigate to `/add?draft=<id>`), "Add to Wishlist" (same with `?wishlist=true`). MVP scope: camera capture + file upload, AI draft generation (obverse only), auto-triggered Numista search. Defer: NGC, reverse capture, edit draft inline, lookup history, sharing. Backend dependency: Cassius to confirm `IntakeDraft` response includes persistent `id` for query param flow.

- **2026-06-09:** F013 nullable scalar update contract: Go DTO pointer fields cannot distinguish omitted vs JSON `null` without raw-body presence tracking. For update DTOs, pair typed binding with raw `json.RawMessage` presence detection so explicit `null` clears nullable scalar DB columns while omitted fields preserve values.

- **2026-06-09:** F013 Phase 4 Browser Workflow Infrastructure (T018–T021, APPROVED)
  - Implemented Playwright-based browser testing framework under `src/web/e2e/`
  - **Deliverables:** `playwright.config.ts`, `src/web/e2e/fixtures/workflow.ts` (reusable fixtures + mocked API routes), `src/web/e2e/workflows/auth.spec.ts` (login/logout), `src/web/e2e/workflows/coin-form.spec.ts` (add/edit coin), `test:browser` npm script
  - **Test Coverage:** Golden fixtures with consistent mock data, no live backend or production data, frontend validation verified, all 4 Playwright tests passing
  - **Hygiene:** Playwright-generated outputs (`src/web/test-results/`, `src/web/playwright-report/`) now properly ignored in `.gitignore`
  - **Review Status:** Maximus BLOCKED on hygiene (stale docs, missing `.gitignore` entries), Brutus independently remediated, Maximus approved revision
  - **Coordinator Validation:** `npm run type-check` ✅, `npm run test -- --run` 99 tests ✅, `npm run test:browser` 4 tests ✅, `git diff --check` ✅
  - **Next Tasks:** T022–T028 (edit workflows, image upload, search/filter, mobile viewport, Taskfile, docs)

- **2026-06-09:** F013 storage/tag browser workflows now extend the Playwright route mock with deterministic `/storage-locations`, `/tags`, `/sets`, tag attach/detach, and set add/remove handlers. T022 changes storage location A→B→None through `CoinForm`; T023 treats tags/sets as detail-page association edits and asserts captured mutation payloads plus visible chips.

- **2026-06-09:** F013 final browser workflows (T024-T026) completed upload/delete image, collection search/filter, and mobile viewport edit Playwright coverage using the shared `src/web/e2e/fixtures/workflow.ts` mocked API state. Workflow mocks now record image uploads/deletes and coin query params, filter fixture coins deterministically, and CoinForm image controls have accessible labels for stable production-safe hooks. Learning: form image edits save the scalar update first, then delete removed/replaced images, then upload replacements; tests should assert all three effects.

- **2026-06-09 (13:45:22):** F013 T024–T026 Image Upload, Search/Filter, Mobile Viewport Workflows APPROVED
   - **T024 (Image Upload):** Manual add upload obverse/reverse with preview → save → verify persisted; manual edit replace/delete images → save → verify detail page reflects changes. Route-level mocked API state for image upload/delete handlers; fixture-backed deterministic File objects.
   - **T025 (Search/Filter):** Navigate collection list → filter by category/era/material/name → verify results deterministically updated; route-level mocked filter handlers; fixture coins filtered by query params.
   - **T026 (Mobile Viewport):** Playwright mobile device profile (375px width) → edit one field on coin detail → submit → verify responsive form layout, no scroll/overflow issues. Fixture-backed, deterministic mobile browser context.
   - **Coordinator Pre-Validation:** npm type-check ✅ (99 tests), npm test ✅ (99 tests), npm run test:browser ✅ (9 Playwright tests), git diff-check ✅
   - **Maximus Review:** ✅ APPROVED — workflows deterministic and fixture-backed, mocks proportional, scope boundaries strict (no creep into T027–T028), Principle VI + Principle IX satisfied
   - **F013 Phase 4 Completion:** T018–T023 + T024–T026 = 9 of 11 Phase 4 tasks complete. T027–T028 (Taskfile + docs) pending.
   - **Key Learning:** Image form edits perform scalar update first, then image deletes, then uploads; tests must assert all three side effects separately.
   - **Orchestration Log:** `.squad/orchestration-log/2026-06-09T13-45-22Z-aurelia.md`

- **2026-06-10:** Collection Pagination Count Summary — Added "Showing X–Y of Z coins" range display to CollectionPagination.vue (grid mode) to clarify total collection size when pagination limits to 50 per page. Responsive layout: mobile shows range above page number (vertical stack), desktop shows range + page number inline with bullet separator. Computed properties rangeStart and rangeEnd calculate current page item range. Updated tests to verify range formatting for first/last/partial pages. Type-check + tests pass. Preserves existing active-collection filters (wishlist:false, sold:false). PWA swipe mode already shows "X / Total" counter; only grid mode needed explicit range summary.

- **2026-06-10:** Coin of the Day Pushover links need an explicit `PublicAppURL` admin setting because Pushover opens outside the PWA/app route context. When configured, backend builds absolute `http(s)://host/coin/{coinID}` links after trimming trailing slashes; when blank or invalid, keep HTML message formatting but omit both `url` and `<a>` so no broken relative `/coin/{id}` link ships.

- **2026-06-18:** Mint Map frontend now treats admin-managed `/mint-locations` as the runtime source for map grouping. `groupCoinsByMint(coins, mintLocations)` builds its lookup from passed backend locations, while `ancientMints.ts` remains only a seed/reference file. Admin → Coin Properties owns the Custom Locations card with local CRUD state, client-side coordinate validation, alias comma/newline parsing, and confirmation-based delete.
- **2026-06-18:** Custom Locations list rows should keep row actions inside the title/header flex row, with detail metadata below. This preserves right-aligned Edit/Delete controls on desktop while allowing the button group to wrap with the title on narrow screens.
- **2026-06-18:** Custom Locations list alignment: keep the name/details column explicitly `text-align: left` and width-filling, with only the Edit/Delete action group right-aligned via the row header layout.
- **2026-06-18:** Admin Schedules panels without run history should still share `useAdminConfig` schedule message refs, `.avail-settings` layout, and a `Run Now` client wrapper. Collection health snapshots use `CollectionHealthSnapshotsEnabled` plus `CollectionHealthSnapshotsStartTime` defaults (`false`, `04:30`) and call `POST /admin/collection-health-snapshots/run`.

- **2026-06-18:** Biometric login PWA fix (issue #299) completed. Frontend WebAuthn login flow now unwraps backend's nested `options.publicKey`, converts base64url challenge and allowCredentials IDs to ArrayBuffer, trims requested username, and uses backend-returned username in login finish. Maintains backward compatibility with legacy flat options shape. Replaced emoji with lucide `LockKeyhole` icon in biometric buttons per Principle VI. Frontend regression tests added for flat options, legacy nested shape, and missing-challenge guard. `npm run test -- auth.test.ts`, `npm run type-check`, and `npm run build` all pass.

---

## 2026-06-18 18:50:13 — Page Header Consistency Pass

**Task:** Standardize page headers across the PWA to match the elegant museum pattern from AuctionsPage.

**Pattern established:**
- Title aligned left (h1, no icons inside)
- Action buttons aligned right (icon + text on desktop, icon-only on PWA per usePwa composable)
- No verbose descriptions in headers
- Uses global .page-header class from main.css

**Pages updated:**
1. **NotesPage.vue** — Removed icon from h1, removed "Personal workspace" label, removed verbose intro. Removed the large stats summary card (3-column grid with note counts). Now shows clean title left, "New Note" button right. Grid layout simplified to just list + editor.

2. **FollowersPage.vue** — Removed Users icon from h1, standardized button sizing (btn-primary without btn-sm for consistency).

3. **CoinLookupPage.vue** — Removed btn-sm class from Back button for consistency.

4. **SetsPage.vue** — Complete rewrite for museum aesthetic:
   - Changed h1 from "My Sets" to "Sets"
   - Replaced .sets-page/.sets-header with standard .container/.page-header
   - Replaced centered loading/empty states with clean .loading-overlay and .empty-state following AuctionsPage pattern
   - Added Teleport for modal, X icon close button in modal-header
   - Modernized modal styling: modal-overlay, modal-content.card, modal-header with flex layout
   - Added spinner animation matching other pages
   - Removed all hardcoded dimensions, now uses design tokens throughout

5. **ShowcasesPage.vue** — Already correct, no changes needed.

6. **CalendarPage.vue** — Already correct (title left, "Add Event" button right).

**Design token compliance:**
- All border-radius uses var(--radius-sm), var(--radius-md), var(--radius-full)
- All colors use var(--accent-gold), var(--bg-card), var(--border-subtle), var(--text-primary), etc.
- All transitions use var(--transition-fast), var(--transition-med)
- All spacing uses rem units
- Global button classes (.btn, .btn-primary, .btn-ghost) from main.css
- Global .spinner animation pattern reused

**Validation:** vue-tsc --noEmit passed with exit code 0.

**Impact:**
- Consistent museum look and feel across all feature pages
- Users will now see a unified visual language: clean titles, elegant action buttons, no clutter
- Mobile PWA users will see icon-only action buttons where the usePwa composable is active
- Notes page feels more spacious without the redundant summary card
- Sets page now matches the refined aesthetic of Auctions/Calendar

- **2026-06-19:** Backend-authenticated `/uploads/*` cannot be rendered with plain `<img src="/uploads/...">` because browser image requests do not carry the localStorage JWT. Private frontend media now goes through `AuthenticatedImage` / `useAuthenticatedMedia` / `utils/media.ts`, which fetches `/api/uploads/*` with `Authorization: Bearer <token>`, `cache: 'no-store'`, converts to blob URLs, and revokes them on cleanup. Public showcase media must use `publicShowcaseMediaUrl(slug, filePath)` so shared links load `/api/showcase/:slug/uploads/*` without private auth.
