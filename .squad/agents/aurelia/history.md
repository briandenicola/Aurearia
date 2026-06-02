# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins frontend — Vue 3 / TypeScript / Pinia / Vite PWA
- **Architecture:** All API calls through `src/web/src/api/client.ts`; UI follows design tokens from `variables.css` and global classes from `main.css`.

## Core Context

- Aurelia owns frontend implementation and UX polish. Durable frontend rules: `<script setup lang="ts">`, strict nullable handling with `?.`/`??`, no emojis, lucide icons, dark theme, PWA/mobile support, and design-token-only CSS when tokens exist.
- Established patterns: accessible modals follow `FeaturedCoinModal` structure; composables expose cleanup functions and pages call them on unmount; timer/resource cleanup is mandatory; auth store syncs through client refresh callback.
- Feature #219 coin detail patterns: overview uses dual-side media, metadata table rows, settings-style section links, and section pages. Detail sections use `<h3>` headings with section spacing; tags are a full section and use `.chip` sizing for interactive pills.
- Camera/intake patterns from #216: 3-column capture-slot grid, active tile state via tokenized gold glow, circular focus-guide overlay, and AddCoinPage camera controls aligned under slots with Camera/Images lucide icons.

## Recent Updates

- **2026-06-01:** Tags UI refinements completed for #219: Tags promoted to full section after Details, before Catalog References; `.chip` sizing used; type-check/build clean.
- **2026-06-01:** AddCoinPage camera actions changed to a 3-column grid matching capture slots; shutter centered under REVERSE and photo library button aligned under CARD; Upload icon replaced by Images.
- **2026-06-01:** Purchase metadata moved into the Details table as full-width rows. `CoinDetailMetadataRow` gained `fullWidth?: boolean` and later `url?: string | null`; purchase location renders as store-only with optional sanitized `SafeExternalLink`.
- **2026-06-01:** Store prefix label added for purchase location: `Store: ` is rendered only for `row.key === 'purchaseLocation'`, styled muted/italic; only the store name is clickable when a URL is present.

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

## 2026-06-01 — PWA Stuck Tap Bug — REAL Root Cause (pull-to-refresh) — CORRECTED

**⚠️ Correction of earlier entries in this session.** Two earlier theories (`showChat` overlay reset, and `bulkSelectActive` leak) did NOT fix the reported tap-blocking bug. The `showChat.value = false` line added to `App.vue.onMounted` was a no-op (App mounts once at boot; `showChat` is already `false`) and was reverted.

**Reported bug:** After heavy PWA use, "can't tap a searched coin, can't rotate the reverse image." Screen looks completely normal (NOT dimmed) — taps just do nothing. Brian confirmed "screen looks normal/bright" which ruled out every dimmed backdrop overlay (chat, sidebar).

**Real root cause:** `src/web/src/composables/usePullToRefresh.ts`. The handler set `pulling=true` on touchstart and only cleared it on `touchend`. When the OS/browser hijacks a gesture (notification, multitouch, system back-swipe — frequent in PWA mode) `touchcancel` fires instead of `touchend`, so `pulling` stuck `true`. Its non-passive `touchmove` then called `e.preventDefault()` on any downward drift at scroll-top — including the tiny drift of a normal tap — which **suppresses the synthesized click** on mobile. Result: coins untappable and the global image-rotation control dead, with nothing visible on screen.

**Fix (commit `9f906bf`):**
- Added a `touchcancel` handler that resets `pulling`/`engaged`/`pullDistance`.
- Added an `ENGAGE_SLOP` (10px) movement threshold so `preventDefault()` only fires once a real pull is underway — taps and small drifts are left fully clickable.
- Defensive state reset on touchstart.

**Files Modified:**
- `src/web/src/composables/usePullToRefresh.ts` — touchcancel handler + slop guard
- `src/web/src/pages/CollectionPage.vue` — KEPT the `bulkSelectActive` mount/unmount reset (fixes a separate, real FAB-hidden leak)
- `src/web/src/App.vue` — REVERTED the no-op `showChat.value = false`

**Pattern (durable):** A non-passive `touchmove` that calls `preventDefault()` MUST be paired with a `touchcancel` handler that resets gesture state, and must never `preventDefault()` on a stationary tap — otherwise stuck gesture state silently kills clicks app-wide.

**Verification:** type-check ✅, build ✅, lint ✅ (0 errors). 3 pre-existing design-token font-budget test failures unchanged from HEAD.



## 2026-06-01 — Legacy Catalog Reference Migration UI (SHIPPED)

Added a bordered section to Settings → Data for triggering the legacy RIC→CoinReference migration endpoint and displaying result counts (succeeded/skipped/failed).

**Implementation:**
- `src/web/src/types/index.ts` — `LegacyMigrationResult` interface with `succeeded`, `skipped`, `failed` counts and optional `message`
- `src/web/src/api/client.ts` — `migrateLegacyReferences()` function calling `POST /references/migrate-legacy`
- `src/web/src/components/settings/SettingsDataSection.vue` — new migration card with:
  - Database icon + "Catalog Reference Migration" heading
  - Explanatory text (non-destructive, keeps originals, records in journal)
  - Trigger button (RefreshCw icon, spinning during request)
  - 3-column result grid: Succeeded (gold), Skipped (muted), Failed (amber)
  - Error handling with `apiErrorText()` helper

**Design System Compliance:**
- ✅ All tokens: `--accent-gold`, `--text-muted`, `--bg-input`, `--border-subtle`, `--radius-sm`, `--text-secondary`
- ✅ Global `.btn` / `.btn-primary` classes
- ✅ Lucide-vue-next icons only (Database, RefreshCw)
- ✅ No emojis
- ✅ Mobile-responsive (result grid stacks on narrow viewports)

**Verification:** npm run build/lint pass; no new warnings; commit 978eb23.

**Related:** Cassius implemented parallel backend endpoint with per-coin journaling.

- **2026-06-01:** CoinDetailPage UI refinements completed: "Details" heading in sections-list renamed to "Additional Details" to disambiguate from metadata table; Tags section moved below Catalog References; Inscription and Description sections merged into a single "Inscription" block with dual-side layout (Obverse/Reverse subsections showing inscription + description prose for each side), wrapped in `.section-content-card` with mobile-responsive stacking. Final section order: Title → Inscription → Details (metadata table) → Catalog References → Tags → Listing Status → Additional Details (section links to Activity Journal, Health, Analysis, Notes). All CSS uses design tokens; type-check, build, and lint pass with no new warnings.

## 2026-06-01 — Free-Text Rarity/RIC UI Removal

Removed legacy free-text Rarity/RIC surface from coin detail metadata and coin form. The structured Catalog References section now serves as the canonical UI for numismatic references.

**Files modified:**
- `src/web/src/composables/useCoinDetailMetadataRows.ts` — removed the `Rarity / RIC` metadata row backed by `coin.rarityRating`
- `src/web/src/components/CoinForm.vue` — removed the `Rarity Rating (RIC)` input field from the coin add/edit form  
- `src/web/src/components/coin/CoinInfoGrid.vue` — removed legacy `Rarity / RIC` fallback info card

**Notes:**
- TypeScript types and API client sanitization remain intact for backward compatibility
- Backend free-text `coin.rarityRating` persists; structured `CoinReference` records are the future canonical storage
- Commit: be84843

**Related:**
- Cassius proposed a design for migrating legacy `rarityRating` values into `CoinReference` records (PROPOSED/PENDING Brian approval)

## 2026-06-01 — Catalog Registry Admin Frontend + CoinReference Certainty → InvoiceNumber

Completed frontend deployment of catalog registry feature in parallel with Cassius's backend work.

**Changes:**
- **Types:** Renamed `CoinReference.certainty` → `invoiceNumber` in `src/web/src/types/index.ts`. Added `CatalogRegistry` interface.
- **API Client:** Added `listCatalogs()`, `adminCreateCatalog()`, `adminUpdateCatalog()`, `adminDeleteCatalog()` in `src/web/src/api/client.ts`.
- **Coin References UI:** Converted free-text catalog input to `<select>` dropdown from `listCatalogs()` with legacy fallback option; replaced `certainty` input with `invoiceNumber` input (placeholder updated).
- **Agent Chat:** Removed `certainty` field from candidate reference mapping in `useCoinSearchChat.ts` (AI no longer provides this).
- **Admin UI:** New `AdminCatalogsSection.vue` CRUD interface following existing admin patterns (table with code/name/era/toggle, modal form, 409 conflict handling for in-use catalogs).
- **Admin Page:** Registered `catalogs` tab in `AdminPage.vue` (configuration group, BookMarked icon, positioned after System).
- **Help Text:** Updated `HelpSection.vue` reference field list "(catalog, volume, number, certainty, authority URI)" → "(catalog, volume, number, invoice number, authority URI)".

**Design decisions:**
- Dropdown ensures catalog consistency; legacy fallback prevents data loss when editing references with removed catalogs
- Invoice number semantics: field repurposed from unused "certainty" → manual purchase invoice tracking
- Admin placement: catalogs are configuration (not operational), grouped with Users/AI/System
- Delete 409 handling: friendly error message ("in use by X coins") rather than raw API error

**Verification:** npm run build ✅, npm run lint ✅ (no new warnings). Commit 0de29af.

**Backend integration:** Cassius implemented CRUD + field rename + AI concept removal. Commit d0d3db1.

**OpenAPI:** Coordinator regenerated. Commit 100087f. All three commits pushed to origin/main.

**Prior batch (unlogged):** Promoted CatalogArchives in HelpSection helpful resources + refined description (commit 1e0de0d).

## 2026-06-01 — Agent FAB Hidden Bug Fix (Module-Level State Leak)

> Note: this fix addressed the **agent FAB staying hidden** after bulk-select, NOT the reported tap-blocking bug (that was pull-to-refresh — see the corrected entry above). Kept because it resolves a real, separate state leak.

Fixed a state leak where the agent FAB stayed hidden after heavy PWA use / bulk-select navigation.

**Root Cause:**
Module-level `bulkSelectActive` ref in `useBulkSelect.ts` was never reset when CollectionPage unmounted. The ref persists across navigation (module scope), but the local `selectMode` ref is destroyed on unmount. When user:
1. Activates bulk select mode in CollectionPage (`selectMode = true`, `bulkSelectActive = true`)
2. Navigates away (search results, coin detail)
3. CollectionPage unmounts → `selectMode` destroyed, but `bulkSelectActive` remains `true`
4. Returns to gallery → fresh CollectionPage mounts with `selectMode = false`, but `bulkSelectActive` still `true`
5. UI appears normal but the agent FAB in App.vue stays hidden (reads stale `bulkSelectActive`)

This didn't directly break coin clicks (CoinCard uses passed `selectable` prop, not the global ref), but caused confusion and hidden UI bugs. The real issue was **state leak across navigation** due to module-level refs not respecting component lifecycle.

**Fix:**
- Added `onUnmounted()` hook to CollectionPage to reset `bulkSelectActive` when navigating away with select mode active
- Added `onMounted()` reset of `bulkSelectActive` to ensure clean state on every mount (defensive, handles edge cases)

**Files Modified:**
- `src/web/src/pages/CollectionPage.vue` — added `onUnmounted` import and cleanup hook

**Pattern Learned:**
Module-level refs (exported from composables) do NOT reset on component unmount. When a ref gates global UI state (like hiding the agent FAB), the owning component MUST clean up on unmount or the state will leak across navigation. Alternative solutions:
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
