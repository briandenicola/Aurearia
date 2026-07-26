# Implementation Plan: Era & Category Consistency Remediation

**Branch**: `339-era-category-remediation` | **Date**: 2026-07-26 | **Spec**: `specs/339-era-category-remediation/spec.md`
**Input**: Feature specification from `specs/339-era-category-remediation/spec.md`

## Summary

Close the gap between the admin-configurable `CoinCategories`/`CoinEras`
settings and the ~20 code locations that still hardcode the legacy default
values (Roman/Greek/Byzantine/Modern/Other, ancient/medieval/modern). Add
backend category validation (currently nonexistent), route the Quick Capture
and AI-chat paths through the same allow-list era/category validation as the
main coin API instead of their own stricter or laxer checks, source every
frontend picker/filter from the existing `useCoinOptions` composable instead
of local hardcoded arrays, replace duplicated per-name color switches with
one shared deterministic-color utility, and give `Category` a built-in
baseline allow-list mirroring the existing `builtInCoinEras` mechanism so
the "Roman" value Emperor Tracker depends on can never be invalidated by an
admin edit to `CoinCategories`.

## Technical Context

**Language/Version**: Go (Gin, GORM) backend; Vue 3 + TypeScript (Vite) frontend
**Primary Dependencies**: existing `SettingsService`/`AppSetting`, existing `useCoinOptions` composable, existing `CatalogRegistryRepository`
**Storage**: No schema change — this is validation/consistency logic only
**Testing**: Go unit tests alongside `coin_service.go`/`quick_capture_service.go`/`set_criteria.go`; frontend component/unit tests for the converted pickers
**Target Platform**: Existing web app (no infra change)
**Project Type**: Web application (backend `src/api` + frontend `src/web`)
**Constraints**: Must not break existing coins whose category/era value predates a since-edited admin list (preserve legacy display, per spec edge case)
**Scale/Scope**: ~9 backend locations, ~13 frontend locations identified by audit

## Constitution Check

- No new architectural boundary — all changes are within existing services/
  handlers (backend) and existing composables/components (frontend).
- No schema/migration required.
- FR-008's built-in Category baseline is a direct structural mirror of the
  existing `builtInCoinEras` mechanism, not a new pattern — no new
  "reserved values" subsystem or settings-update guard is introduced.

## Audit Findings (reference)

### Backend (`src/api/`)

| # | Location | Issue | Fix |
|---|---|---|---|
| B1 | `services/coin_service.go:357-384` `validateCoinEra` | Correct today — reference implementation to mirror for category. | Extract into a shared, reusable allow-list checker parameterized by setting key + built-ins, used by both era and category validation, and by Quick Capture (B4). |
| B2 | `models/coin.go:46` `Category` field | No `binding` / no service-level validation at all — pure free string. | Add `validateCoinCategory` in `coin_service.go`, called from `prepareCoinForCreate` and the update path, same shape as `validateCoinEra`. |
| B3 | `services/set_criteria.go:89-104` `GetSuggestedCriteria` | Hardcodes `Roman/Greek/Byzantine` as starter smart-set suggestions. | Read the admin's current `CoinCategories` (top N or all) to build suggestions instead of literals. |
| B4 | `services/quick_capture_service.go:546-560` `ValidateCoinMinimumForPromotion` | Own hardcoded `switch` on era, stricter than and inconsistent with the main validator — rejects valid custom eras. | Replace with a call into the shared era-validation helper from B1. |
| B5 | `repository/scopes.go:26-31` `ActiveRomanCollection`; `handlers/coin_requests.go:87-88,262-263` | Both literal-compare `category == "Roman"` for Emperor Tracker coupling. | Leave the comparisons as-is — B6 makes the literal permanently valid regardless of setting edits, so the dependency can no longer silently go stale. |
| B6 | `services/coin_service.go` (new `builtInCoinCategories` constant) | No protection today against removing/renaming the "Roman" value in `CoinCategories` — Category has no built-in fallback at all (unlike Era's `builtInCoinEras`). | Add `builtInCoinCategories = []string{"Roman","Greek","Byzantine","Modern","Other"}`, structurally identical to `builtInCoinEras`, consumed by the shared allow-list validator from B1/B2. Admins remain free to edit `CoinCategories` (it only controls what's *offered* in pickers); "Roman" (and the other four defaults) stay *acceptable* no matter what, exactly mirroring how era's built-ins already behave. No settings-update guard/rejection needed — this replaces that approach entirely. |
| B7 | `services/collection_tools_service.go:116-144` `SearchMyCollection` | Category substring-matches only `roman/greek/byzantine/modern`; era only handles `ancient/medieval` (no `modern` branch at all). | Match against the live `CoinCategories`/`CoinEras` lists (case-insensitive substring/equality against each configured value) instead of a fixed literal set; fixes the missing "modern" era branch as a side effect. |
| B8 | `services/coin_lookup_service.go:429-443` `mergeNGCLabelFields` | Hardcoded category+era inference from NGC slab label text. | Out of scope (spec edge case) — heuristic inference, not a validation/persistence bug. No change. |
| B9 | `docs/docs.go` generated Swagger enums for `Category`/`Era` | Static enum listing overstates enforcement (Category) / understates dynamic allow-list (Era). | Regenerate docs after B1/B2 land; update doc comments on `models.Coin` to describe both fields as "validated against admin-configurable settings" rather than a fixed enum. |

### Frontend (`src/web/src/`)

| # | Location | Issue | Fix |
|---|---|---|---|
| F1 | `composables/useCoinOptions.ts`, `types/index.ts:678,681` | Correct pattern + intended fallback defaults — reference implementation. | No change to the composable itself; ensure every consumer below imports from it instead of `types/index.ts` constants directly. |
| F2 | `components/CategoryFilter.vue:11,15-23,33` | Imports `CATEGORIES` directly; hardcoded per-name color class. | Source chip list from `useCoinOptions().categoryOptions`; replace per-name color switch with the shared color utility (F-shared, below). |
| F3 | `components/sets/SetSmartRuleBuilder.vue:23-30,57-81,97-103` | Category value `<select>` hardcoded; era has no dedicated picker at all (falls to free text). | Wire both category and era value pickers to `useCoinOptions()`. |
| F4 | `components/admin/AdminCatalogsSection.vue:49-53,119-124` | Era chip color and era `<select>` hardcoded to ancient/medieval/modern only — admins can't assign a custom era to a Catalog Registry entry. | Source the era `<select>` from `useCoinOptions().eraOptions`; route color through the shared utility. |
| F5 | `components/ui/BaseBadge.vue:16-33` | Hardcoded per-category color switch (component currently has no call sites). | Replace switch with the shared color utility; leave wiring as-is since it's unused today. |
| F6 | `components/AuctionLotCard.vue:14-19,44-53` | Hardcoded per-category text-color classes. | Replace with shared color utility. |
| F7 | `components/stats/StatsBarChart.vue:20-33`, `pages/StatsPage.vue` (fill/badge class generation) | Hardcoded `fill-<name>`/`badge-<name>` CSS classes only defined for the 4 defaults. | Replace class-name-per-literal approach with the shared color utility returning an actual color value (not a class name keyed to a fixed CSS list), applied via inline style or a CSS variable. |
| F8 | `components/stats/StatsCoinFlowChart.vue:189-206` `PERIOD_PALETTE`/`ERA_COLORS` | Hardcoded positional/keyed palettes with neutral fallback for unknown values. | Replace with the shared color utility for both category and era coloring. |
| F9 | `composables/useCoinSearchChat.ts:27,29,46-59,83,100` | `VALID_CATEGORIES`/`VALID_ERAS` force-coerce AI-suggested category to "Other" / blank out non-default era before building the wishlist payload. | Validate against the live `useCoinOptions()` lists; pass through any admin-valid custom value unchanged instead of coercing/dropping it. |
| F10 | `utils/coinLookupDraft.ts:262-266` `normalizedEra` | Only accepts the 3 default eras, else `undefined` — drops custom era text when saving a Quick Capture draft. | Accept any non-empty era string; validation of admin-list membership happens server-side (B1/B4) at promotion time, consistent with how free-text fields are handled elsewhere. |
| F11 | `components/HelpSection.vue:30,33,107-336` | Static help copy listing the 5/3 legacy values, including a CSV example. | Lower priority: reword copy to say "your admin-configured categories/eras (defaults shown below)" rather than dynamically rendering live settings inside static help text; avoids overstating exact values without requiring a data-fetching help component. |
| F12 | `components/chat/ChatIntroPanel.vue:11-29` | Static example prompt copy mentioning Roman/Byzantine/Greek. | Cosmetic only — no change required; these are illustrative examples, not a functional list. |
| F13 | `components/quick-capture/QuickCaptureForm.vue:20`, `pages/QuickCaptureDraftPage.vue:99` | `placeholder="ancient"` on an already-unconstrained free-text input. | Cosmetic only — leave as an example placeholder; the field itself is not restricted so there's no functional bug here. |

**New shared utility**: `src/web/src/utils/categoryColor.ts` (or similar) —
one function `colorForLabel(label: string): string` that hashes the label
into a fixed, accessible palette, used by F2/F4/F5/F6/F7/F8 in place of their
individual switch statements. Removes five duplicated hardcoded mappings down
to one.

## Rollout Phases

1. **Phase 1 — Backend validation parity** (B1, B2, B4, B6): shared
   allow-list validator (built-ins ∪ AppSetting ∪ Catalog Registry, keyed by
   field), category validation added with its own `builtInCoinCategories`
   baseline mirroring `builtInCoinEras`, Quick Capture promotion uses the
   shared validator instead of its own hardcoded switch.
2. **Phase 2 — Backend consistency fixes** (B3, B7): suggested smart-set
   criteria and AI collection-search filters read live admin lists.
3. **Phase 3 — Frontend sourcing** (F2, F3, F4, F9, F10): every picker,
   filter, and data-path that touches category/era switches to
   `useCoinOptions` / passes through custom values instead of hardcoding or
   coercing.
4. **Phase 4 — Shared color utility** (F5, F6, F7, F8): introduce
   `colorForLabel`, migrate all five hardcoded color switches to it.
5. **Phase 5 — Docs & cosmetic cleanup** (B9, F11): regenerate Swagger docs,
   soften Help Section copy. F12/F13 left as-is (confirmed cosmetic-only,
   no functional bug).

## Complexity Tracking

No constitution violations — this is a consistency/bug-fix pass reusing
existing settings, composables, and validation patterns; no new services,
schema, or third-party dependency introduced.
