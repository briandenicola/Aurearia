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
3. **Given** an AI chat wishlist suggestion carrying a category/era that
   exactly or approximately matches an admin-defined value (e.g. "Roman
   Republic" vs. the admin's "Roman"), **When** the user adds it to their
   wishlist, **Then** it is mapped to the matching admin-defined value
   instead of being coerced to "Other" or dropped blank.
4. **Given** an AI chat wishlist suggestion carrying a category/era with no
   confident match to any admin-defined value, **When** the user adds it to
   their wishlist, **Then** they are shown the AI's raw suggestion and asked
   to either map it to an existing value or keep it as a new one — it is
   never silently coerced or silently accepted.

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

- What happens if an admin removes or renames "Roman" in the `CoinCategories`
  setting, which the Emperor Tracker feature depends on structurally? → The
  value remains valid for acceptance regardless of the setting's contents
  (see FR-008) — mirroring how `ancient`/`medieval`/`modern` already remain
  acceptable even if removed from `CoinEras` — so Emperor Tracker cannot be
  silently broken by an admin edit. The admin edit only affects what's
  *offered* in pickers, not what's *accepted*.
- What happens to a coin whose category/era was set before this remediation
  and no longer matches anything in the admin list (e.g. list was edited
  since)? → Existing behavior is preserved: the value is displayed as-is
  (not blanked out), consistent with how `CoinForm.vue` already handles
  legacy out-of-list values today.
- What happens when the AI coin-lookup/NGC-label inference logic encounters
  a slab label implying a category/era outside the default three mappings?
  → Out of scope for this pass (heuristic inference, not a validation or
  data-loss bug); noted as a follow-up.
- What happens when a direct/programmatic API call (import script, external
  integration — no interactive user present) submits a category/era that
  doesn't match any admin-defined or built-in value? → Rejected outright
  with a clear 4xx error naming the accepted values. Unlike the interactive
  AI-chat path, there's no user available to resolve ambiguity, so silent
  coercion or fuzzy-matching would be worse than a deterministic failure.
- What happens when an AI chat suggestion's category/era doesn't exactly
  match an admin-defined value but is a clear variant of one (e.g. differs
  only by case, punctuation, or a recognizable substring)? → Normalized/
  fuzzy matching (reusing the same normalization approach already used for
  mint-location matching) resolves it to the existing admin-defined value
  automatically, without asking the user, so near-miss spelling doesn't
  create unnecessary confirmation prompts.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Backend MUST validate `Coin.Category` on create/update against
  the admin-defined `CoinCategories` list, mirroring the existing
  `validateCoinEra` mechanism (built-in defaults ∪ `CoinCategories` AppSetting
  ∪ Catalog Registry). Any non-matching value submitted through the coin
  API (interactive or programmatic) MUST be rejected with a clear error
  naming the accepted values — never silently coerced.
- **FR-002**: The Quick Capture draft-promotion era check MUST use the same
  validation logic/allow-list as the main coin create/update path, rather
  than an independently hardcoded switch statement.
- **FR-003**: The AI-chat wishlist-suggestion payload builder MUST resolve
  an AI-suggested category/era against the live admin-defined lists using a
  three-tier match — (1) exact match, (2) normalized/fuzzy match against
  admin-defined values (reusing the mint-location-style normalization
  already in the codebase), (3) if no confident match, present the raw
  suggestion to the user for explicit confirmation (map to an existing
  value or keep as a new one) — rather than a static five-value/three-value
  array that silently coerces to "Other" or blanks the field.
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
  era has no matching branch at all). It MUST use the same normalized/fuzzy
  match as FR-003 (not a fixed literal list) since this path has no user
  present to confirm ambiguous matches — an unmatched term is simply not
  filtered on, rather than guessed at.
- **FR-008**: Backend MUST maintain a built-in baseline allow-list for
  `Category` (`Roman/Greek/Byzantine/Modern/Other`), exactly mirroring the
  existing `builtInCoinEras` mechanism for `Era` — these values remain valid
  for coin create/update regardless of the current contents of the
  `CoinCategories` AppSetting, so Emperor Tracker's dependency on the literal
  value "Roman" can never be broken by an admin edit to that setting.
- **FR-009**: Every UI location that colors a category or era (badges,
  filter chips, stats charts) MUST derive its color from one shared utility
  that produces a stable, distinct color for any category/era name —
  including custom ones — rather than duplicating a per-name hardcoded
  switch in each component.
- **FR-010**: Generated API documentation for `Coin.Category` MUST no longer
  describe it as a fixed enum, since it is validated dynamically against
  admin-configurable settings (mirroring how `Coin.Era`'s documentation
  should already reflect its dynamic allow-list).
- **FR-011**: A single shared normalized-match function (name/case/
  punctuation-insensitive, with substring/alias tolerance) MUST back every
  category/era matching decision that isn't a raw user pick from a
  dropdown — used by FR-003 (AI wishlist suggestions) and FR-007 (AI
  collection search) alike, rather than each path reimplementing its own
  matching logic.

### Key Entities

- **AppSetting `CoinCategories` / `CoinEras`**: existing newline-delimited
  admin-defined lists (no schema change required).
- **Built-in Category baseline** (`Roman/Greek/Byzantine/Modern/Other`): a
  new code-level constant, structurally identical to the existing
  `builtInCoinEras`, always accepted independent of the `CoinCategories`
  setting.

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
- **SC-004**: Removing or renaming "Roman" in the `CoinCategories` setting
  never breaks Emperor Tracker — coins can still be validated/saved with
  category "Roman" via the built-in baseline list, exactly as removing
  "ancient" from `CoinEras` today doesn't stop coins from being saved with
  that era.
- **SC-005**: A programmatic API call with an invalid category/era receives
  a clear rejection naming the accepted values, with zero silent coercion.
- **SC-006**: No AI-chat-suggested category/era is ever silently discarded,
  silently coerced to "Other"/blank, or silently accepted as a brand-new
  unconfirmed value — every unmatched suggestion surfaces an explicit
  user choice.

## Assumptions

- No new AppSetting or schema change is required — this is a consistency/
  enforcement fix on top of existing `CoinCategories`/`CoinEras` settings.
- The built-in baseline list is scoped to today's five default categories
  (matching the existing `builtInCoinEras` pattern for the three default
  eras); no broader "reserved values" framework or settings-side
  rename/delete blocking is introduced — the setting itself stays fully
  editable, the built-in list is a separate, code-level acceptance
  guarantee underneath it.
- Color assignment for custom categories/eras is fully automatic
  (hash-derived from the name); no admin UI for picking colors is added in
  this pass.
- AI coin-lookup/NGC-label-based category/era inference (heuristic slab-text
  parsing) is explicitly out of scope — it's a best-effort suggestion, not a
  validation or persistence path.
- Cosmetic/documentation-only occurrences (Help Section copy, AI chat intro
  example prompts, a placeholder string on a free-text input) are addressed
  as a low-priority cleanup pass, not blocking the core remediation.
