# Feature Specification: Era & Category Consistency Remediation

**Feature Branch**: `339-era-category-remediation`
**Created**: 2026-07-26
**Status**: Draft
**Input**: User description: "We have the custom eras and categories. For the most
part they work great but there are some parts of the app that still use legacy
ancient/modern/medieval hardcoded value. Can you research the code and detect
all usage of categories and era and sure they all are pulling from the same
definitions then develop a remediation plan."

## Background

`CoinCategories` and `CoinEras` are admin-editable `AppSetting` values (default
seed: categories `Roman/Greek/Byzantine/Modern/Other`, eras
`ancient/medieval/modern`), intended as the single source of truth for every
part of the app that offers or checks a coin's category/era. An audit found
that while the main coin form (`CoinForm.vue`) and the backend era validator
correctly consult these settings, roughly twenty other locations across the
backend and frontend independently hardcode the default five categories
and/or three eras — in some cases silently discarding a legitimate
admin-defined custom value, in others just degrading cosmetically.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Custom category is enforced everywhere, not just on the form (Priority: P1)

An admin adds a custom category (e.g. "Celtic"). A user should be able to
save a coin with that category through every entry point in the app, not
just the manual coin form.

**Why this priority**: This is the core trust issue — customizing the list
today gives a false sense that the whole app respects it, when several paths
silently reject or coerce the value.

**Independent Test**: Add a custom category via admin settings. Create a coin
with that category via (a) the coin form, (b) promoting a Quick Capture
draft, (c) accepting an AI chat wishlist suggestion tagged with that category.
Confirm all three persist the custom value unchanged.

**Acceptance Scenarios**:

1. **Given** a custom category exists in `CoinCategories`, **When** a coin is
   created or updated with that category via any API path, **Then** the
   backend accepts it (mirroring how era validation already works).
2. **Given** a Quick Capture draft with a custom era, **When** the user
   promotes it to a full coin, **Then** promotion succeeds instead of being
   rejected by a stricter, separate hardcoded check.
3. **Given** an AI chat wishlist suggestion carrying a custom category or
   era, **When** the user adds it to their wishlist, **Then** the custom
   value is preserved instead of being coerced to "Other" or dropped blank.

---

### User Story 2 - Every UI surface offers the current admin-defined list (Priority: P1)

Any dropdown, filter, or picker that lets a user choose a category or era
shows the live admin-defined list, not a fixed legacy set.

**Why this priority**: A custom category that can't even be selected/filtered
in half the app is effectively second-class.

**Independent Test**: Add a custom category and era via admin settings. Open
the collection category filter chips, the smart-set rule builder's
category/era pickers, and the Catalog Registry admin era picker. Confirm the
custom values appear in all of them.

**Acceptance Scenarios**:

1. **Given** a custom category, **When** the user opens the collection
   category filter, **Then** a chip for that category is available.
2. **Given** a custom category and era, **When** the user builds a smart-set
   rule on the `category` or `era` field, **Then** both offer the live
   admin-defined values instead of a hardcoded list (or a missing picker, in
   era's case today).
3. **Given** a custom era, **When** an admin edits a Catalog Registry entry,
   **Then** they can assign that custom era to it.

---

### User Story 3 - Custom values degrade gracefully in visuals, not silently (Priority: P2)

Charts, badges, and filter chips should render a distinct, stable
appearance for a custom category/era rather than a flat, generic fallback.

**Why this priority**: Cosmetic, but a custom category that's indistinguishable
from every other "unknown" value in stats/badges undermines the feature.

**Independent Test**: Add two custom categories. View the Stats page bar
chart, coin-flow chart, and any category badges. Confirm each custom category
gets its own consistent color, not all a single fallback color.

**Acceptance Scenarios**:

1. **Given** two custom categories, **When** viewing any chart or badge that
   colors by category, **Then** each gets a distinct, stable color derived
   automatically from its name (no per-admin color configuration required).

---

### Edge Cases

- What happens if an admin tries to rename or delete the "Roman" category,
  which the Emperor Tracker feature depends on structurally? → Backend MUST
  reject the rename/delete of that specific reserved value with a clear
  error, rather than allowing it and silently breaking Emperor Tracker.
- What happens to a coin whose category/era was set before this remediation
  and no longer matches anything in the admin list (e.g. list was edited
  since)? → Existing behavior is preserved: the value is displayed as-is
  (not blanked out), consistent with how `CoinForm.vue` already handles
  legacy out-of-list values today.
- What happens when the AI coin-lookup/NGC-label inference logic encounters
  a slab label implying a category/era outside the default three mappings?
  → Out of scope for this pass (heuristic inference, not a validation or
  data-loss bug); noted as a follow-up.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Backend MUST validate `Coin.Category` on create/update against
  the admin-defined `CoinCategories` list, mirroring the existing
  `validateCoinEra` mechanism (built-in defaults ∪ `CoinCategories` AppSetting
  ∪ Catalog Registry).
- **FR-002**: The Quick Capture draft-promotion era check MUST use the same
  validation logic/allow-list as the main coin create/update path, rather
  than an independently hardcoded switch statement.
- **FR-003**: The AI-chat wishlist-suggestion payload builder MUST validate
  category/era against the live admin-defined lists rather than a static
  five-value/three-value array, and MUST preserve a valid custom value
  instead of coercing it to "Other" or blanking it.
- **FR-004**: The Quick Capture / coin-lookup draft era normalization MUST
  accept any admin-valid era value, not only the three legacy defaults.
- **FR-005**: The collection category filter chips, the smart-set rule
  builder's category and era value pickers, and the Catalog Registry admin
  era picker MUST all source their options from the same admin-defined
  lists used by the coin form (i.e., the existing `useCoinOptions`
  composable), instead of hardcoded option arrays.
- **FR-006**: The "suggested smart-set criteria" starter templates MUST be
  derived from the admin's current category list rather than a fixed
  Roman/Greek/Byzantine set.
- **FR-007**: The AI "search my collection" natural-language filter MUST
  recognize any admin-defined category/era, not only the legacy defaults,
  and MUST apply an era filter for every admin-defined era (today "modern"
  era has no matching branch at all).
- **FR-008**: Backend MUST prevent the "Roman" category value specifically
  from being renamed or deleted via admin settings, since Emperor Tracker's
  matching scope and coin-request validation depend on that literal value.
- **FR-009**: Every UI location that colors a category or era (badges,
  filter chips, stats charts) MUST derive its color from one shared utility
  that produces a stable, distinct color for any category/era name —
  including custom ones — rather than duplicating a per-name hardcoded
  switch in each component.
- **FR-010**: Generated API documentation for `Coin.Category` MUST no longer
  describe it as a fixed enum, since it is validated dynamically against
  admin-configurable settings (mirroring how `Coin.Era`'s documentation
  should already reflect its dynamic allow-list).

### Key Entities

- **AppSetting `CoinCategories` / `CoinEras`**: existing newline-delimited
  admin-defined lists (no schema change required).
- **Category "Roman" (reserved value)**: gains a backend-enforced protection
  against rename/delete, tied to its use by Emperor Tracker.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A coin with a custom category can be created via every
  create/update path in the app (form, Quick Capture promotion, AI wishlist
  add) without rejection or silent coercion.
- **SC-002**: A custom era or category, once added by an admin, appears in
  every picker/filter in the app that offers category or era choices, with
  no exceptions.
- **SC-003**: Zero component in the frontend independently hardcodes the
  default category/era list for validation or option-population purposes —
  all route through `useCoinOptions` (or its data) or the shared color
  utility.
- **SC-004**: Attempting to rename or delete the "Roman" category via admin
  settings is rejected with a clear error instead of succeeding and breaking
  Emperor Tracker.

## Assumptions

- No new AppSetting or schema change is required — this is a consistency/
  enforcement fix on top of existing `CoinCategories`/`CoinEras` settings.
- Reserved-category protection is scoped to the single value "Roman" for
  now (the only category with a structural code dependency today); no
  broader "reserved values" framework is being introduced.
- Color assignment for custom categories/eras is fully automatic
  (hash-derived from the name); no admin UI for picking colors is added in
  this pass.
- AI coin-lookup/NGC-label-based category/era inference (heuristic slab-text
  parsing) is explicitly out of scope — it's a best-effort suggestion, not a
  validation or persistence path.
- Cosmetic/documentation-only occurrences (Help Section copy, AI chat intro
  example prompts, a placeholder string on a free-text input) are addressed
  as a low-priority cleanup pass, not blocking the core remediation.
