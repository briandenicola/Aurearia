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
