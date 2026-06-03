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

- **2026-06-01 (CORRECTED):** The `showChat` defensive-reset theory below was WRONG and the fix was reverted. `App.vue.onMounted` runs once at app boot when `showChat` is already `false`, so `showChat.value = false` there is a no-op and cannot fix an intermittent mid-session freeze. The reported tap-blocking bug ("can't tap searched coin, can't rotate image") was actually caused by the **pull-to-refresh touch handler** in `src/web/src/composables/usePullToRefresh.ts`: it set `pulling=true` on touchstart but only cleared it on `touchend`. When the OS/browser hijacks a gesture (notification, multitouch, system back-swipe — common in heavy PWA use) `touchcancel` fires instead of `touchend`, leaving `pulling=true`; every later tap at scroll-top then hit a non-passive `touchmove` that called `e.preventDefault()`, which suppresses the synthesized click on mobile — so taps did nothing while the screen looked completely normal. Real fix (commit `9f906bf`): add a `touchcancel` handler that resets state, plus an `ENGAGE_SLOP` so `preventDefault()` only fires on a real pull, never on taps. Lesson: a non-passive `touchmove` that calls `preventDefault()` MUST be paired with a `touchcancel` reset and must never `preventDefault()` on a stationary tap.
- **2026-06-01:** Module-level refs in composables do NOT reset on component unmount. When a module-level ref (exported from a composable like `useBulkSelect.ts`) gates global UI state or interaction behavior, the owning component MUST explicitly reset the ref in `onUnmounted()` or the state will leak across navigation. In CollectionPage, `bulkSelectActive` (module-level) stayed true after unmount while `selectMode` (local) was destroyed, causing the agent FAB in App.vue to stay hidden indefinitely. Fix: add `onUnmounted()` hook to reset module-level state, and defensive `onMounted()` reset to ensure clean state on every mount. Alternative patterns: move state to Pinia store with proper lifecycle, or avoid module-level refs entirely for interaction-gating state—pass via props/emits instead.
- **2026-06-01:** Admin table layout overflow fix pattern: when action buttons overflow on narrow viewports, stack related data vertically in earlier columns rather than letting the table stretch horizontally. In `AdminCatalogsSection.vue`, moved the era pill below the catalog code in the same cell (flex column with `gap: 0.35rem` and `align-items: flex-start`) to free up horizontal space. Action buttons use `display: flex` with `flex-shrink: 0` and `justify-content: flex-end` to ensure they stay right-aligned and never overflow the boundary. This pattern keeps tables responsive without sacrificing action button visibility.
- **2026-06-01:** Free-text Rarity/RIC UI removed in favor of the structured Catalog References section. Removed the Details metadata row from `src/web/src/composables/useCoinDetailMetadataRows.ts`, the legacy info-grid card from `src/web/src/components/coin/CoinInfoGrid.vue`, and the Rarity Rating (RIC) input from `src/web/src/components/CoinForm.vue`; data plumbing remains intact.
- **2026-06-01:** Storage Location frontend integration completed. Added `StorageLocation` types and API client CRUD methods (`getStorageLocations`, `createStorageLocation`, `updateStorageLocation`, `deleteStorageLocation`) in `src/web/src/api/client.ts`; `sanitizeCoin()` now normalizes `storageLocationId` and strips read-only `storageLocation`. Settings → Data now shows a two-column lookup manager with Tags and Storage Locations side by side in `SettingsDataSection.vue`; storage-location delete surfaces backend 409 conflict messages so users know to reassign coins first. `CoinForm.vue` loads `/storage-locations` and binds a single-select “Storage Location” dropdown with a “None” option; `useCoinDetailMetadataRows.ts` displays the chosen location as a Details row with `coin.storageLocation?.name ?? '—'`. Build and lint pass; full `npm test` remains blocked by pre-existing design-token budget failures unchanged from HEAD.
- **2026-06-01:** Settings reorganization completed. Added `src/web/src/components/settings/SettingsBackupsSection.vue` for collection export/PDF/import backups plus API key generation/revoke flows; moved `loadApiKeys()` exposure there. Settings now has tab id `backups` labeled “Backups & Keys” with the Archive icon, and the Data tab now contains only Tags + Storage Locations metadata management.
- **2026-06-01:** Bulk assign location UI completed. Created `BulkLocationPickerModal.vue` (mirroring `BulkTagPickerModal.vue`) with "No location" clear option that emits `null`. Extended `bulkAction()` client signature to accept `opts?: { tagId?: number; storageLocationId?: number | null }` instead of a single `tagId` parameter, maintaining backward compatibility with existing call sites. Updated `BulkActionBar.vue` to add "Assign Location" button with `MapPin` icon emitting `location` event. Wired up `CollectionPage.vue` to load storage locations on mount, handle `@location` event, render `BulkLocationPickerModal`, and call `bulkAssignLocation(locationId)` which posts `{ coinIds, action: 'assign-location', storageLocationId }` to `/coins/bulk`. Build, type-check, and lint all pass (no new warnings).

- **2026-06-01:** Backend storage-location migration convention: nullable `Coin` lookup FKs may exist without physical SQLite constraints (`constraint:-`) to avoid destructive rebuilds; frontend should continue treating `storageLocationId` as nullable and rely on API validation/errors.

- **2026-06-01:** Legacy catalog reference migration UI added to Settings → Data. New bordered section with Database and RefreshCw icons from lucide-vue-next, explanatory text (non-destructive, keeps originals, records outcomes in journal), trigger button with loading state, and result counts grid showing Succeeded (gold accent), Skipped, Failed (amber). Client function `migrateLegacyReferences()` calls `POST /references/migrate-legacy` and returns `LegacyMigrationResult { succeeded, skipped, failed, message? }` type. Results display uses design tokens (`--accent-gold`, `--text-muted`, `--bg-input`, `--border-subtle`, `--radius-sm`) and mobile-responsive stacked layout. Build and lint pass (no new warnings).

- **2026-06-01:** Coin detail back navigation bug fixed. Root cause: EditCoinPage used `router.replace('/coin/:id')` after save, which Vue Router treated as a new Detail entry, leaving the stack as [Gallery, Detail_old, Detail_new]. Changed to `router.back()` which properly pops the Edit entry and returns to the original Detail, maintaining the correct Gallery → Detail → Back → Gallery flow. The pattern: when a child form/edit view saves and should return to parent, prefer `router.back()` over `router.replace()` to avoid polluting the history stack with duplicate parent entries.

- **2026-06-01:** Coin detail "Back" button changed to absolute gallery navigation. Renamed from "Back" to "Back to Gallery" and changed from `router.back()` to `router.push('/')` in `CoinDetailHeaderActions.vue`. This prevents history pollution when users navigate from Coin Details to subpages (journal, health, analysis, etc.), click "Back to Overview" (which pushes back to Detail), then click the Detail page's back button. Without absolute navigation, `router.back()` would incorrectly pop to the subpage instead of the gallery. Parent pages with multiple child subpages should use absolute nav to their list view, not `router.back()`.

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
