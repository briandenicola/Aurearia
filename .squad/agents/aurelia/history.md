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

- **2026-06-01:** Admin table layout overflow fix pattern: when action buttons overflow on narrow viewports, stack related data vertically in earlier columns rather than letting the table stretch horizontally. In `AdminCatalogsSection.vue`, moved the era pill below the catalog code in the same cell (flex column with `gap: 0.35rem` and `align-items: flex-start`) to free up horizontal space. Action buttons use `display: flex` with `flex-shrink: 0` and `justify-content: flex-end` to ensure they stay right-aligned and never overflow the boundary. This pattern keeps tables responsive without sacrificing action button visibility.
- **2026-06-01:** Free-text Rarity/RIC UI removed in favor of the structured Catalog References section. Removed the Details metadata row from `src/web/src/composables/useCoinDetailMetadataRows.ts`, the legacy info-grid card from `src/web/src/components/coin/CoinInfoGrid.vue`, and the Rarity Rating (RIC) input from `src/web/src/components/CoinForm.vue`; data plumbing remains intact.
- **2026-06-01:** Storage Location frontend integration completed. Added `StorageLocation` types and API client CRUD methods (`getStorageLocations`, `createStorageLocation`, `updateStorageLocation`, `deleteStorageLocation`) in `src/web/src/api/client.ts`; `sanitizeCoin()` now normalizes `storageLocationId` and strips read-only `storageLocation`. Settings → Data now shows a two-column lookup manager with Tags and Storage Locations side by side in `SettingsDataSection.vue`; storage-location delete surfaces backend 409 conflict messages so users know to reassign coins first. `CoinForm.vue` loads `/storage-locations` and binds a single-select “Storage Location” dropdown with a “None” option; `useCoinDetailMetadataRows.ts` displays the chosen location as a Details row with `coin.storageLocation?.name ?? '—'`. Build and lint pass; full `npm test` remains blocked by pre-existing design-token budget failures unchanged from HEAD.
- **2026-06-01:** Settings reorganization completed. Added `src/web/src/components/settings/SettingsBackupsSection.vue` for collection export/PDF/import backups plus API key generation/revoke flows; moved `loadApiKeys()` exposure there. Settings now has tab id `backups` labeled “Backups & Keys” with the Archive icon, and the Data tab now contains only Tags + Storage Locations metadata management.

- **2026-06-01:** Backend storage-location migration convention: nullable `Coin` lookup FKs may exist without physical SQLite constraints (`constraint:-`) to avoid destructive rebuilds; frontend should continue treating `storageLocationId` as nullable and rely on API validation/errors.

- **2026-06-01:** Legacy catalog reference migration UI added to Settings → Data. New bordered section with Database and RefreshCw icons from lucide-vue-next, explanatory text (non-destructive, keeps originals, records outcomes in journal), trigger button with loading state, and result counts grid showing Succeeded (gold accent), Skipped, Failed (amber). Client function `migrateLegacyReferences()` calls `POST /references/migrate-legacy` and returns `LegacyMigrationResult { succeeded, skipped, failed, message? }` type. Results display uses design tokens (`--accent-gold`, `--text-muted`, `--bg-input`, `--border-subtle`, `--radius-sm`) and mobile-responsive stacked layout. Build and lint pass (no new warnings).

- **2026-06-01:** Coin detail back navigation bug fixed. Root cause: EditCoinPage used `router.replace('/coin/:id')` after save, which Vue Router treated as a new Detail entry, leaving the stack as [Gallery, Detail_old, Detail_new]. Changed to `router.back()` which properly pops the Edit entry and returns to the original Detail, maintaining the correct Gallery → Detail → Back → Gallery flow. The pattern: when a child form/edit view saves and should return to parent, prefer `router.back()` over `router.replace()` to avoid polluting the history stack with duplicate parent entries.

- **2026-06-01:** Coin detail "Back" button changed to absolute gallery navigation. Renamed from "Back" to "Back to Gallery" and changed from `router.back()` to `router.push('/')` in `CoinDetailHeaderActions.vue`. This prevents history pollution when users navigate from Coin Details to subpages (journal, health, analysis, etc.), click "Back to Overview" (which pushes back to Detail), then click the Detail page's back button. Without absolute navigation, `router.back()` would incorrectly pop to the subpage instead of the gallery. Parent pages with multiple child subpages should use absolute nav to their list view, not `router.back()`.

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
