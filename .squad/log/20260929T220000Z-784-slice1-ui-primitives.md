# Handoff: #784 slice 1 — shared toggle and status-badge primitives

- Date: 2026-09-29
- Branch: `beta`
- Issue: #784 (UI fit and finish: consistent typography, badges and controls)
- Scope authorized by the owner: **slice 1 (foundation) only**; screenshots
  waived for this slice; toggles consolidated to **two** sizes.

## Completed

1. **`BaseToggle`** (`src/web/src/components/ui/BaseToggle.vue`, new)
   - `v-model`, `size` (`md` default 28x50 / 22px knob, `sm` 22x42 / 16px knob),
     `disabled`, `label` (aria-label), `inheritAttrs: false` so `id` and other
     attributes land on the `<input>`.
   - Standardizes the track on `--bg-input`, the knob on `--text-secondary`, and
     always renders a **keyboard** focus ring (`peer-focus-visible`), so a
     mouse click leaves no persistent outline.
   - **All 23 hand-rolled `peer sr-only` toggles migrated** across 15 files
     (9 schedule panels, `SettingsAccountSection`, `SettingsAppearanceSection`,
     `AdminSchedulesSection`, `AdminOIDCSection`, `AdminCatalogsSection`,
     `CoinForm`). Zero remain outside the primitive.

2. **`BaseStatusBadge`** (`src/web/src/components/ui/BaseStatusBadge.vue`, new)
   - `tone` of `success | error | warning | info | neutral`, driven by new
     `--status-*-bg` / `--status-*-fg` tokens in `variables.css`.
   - 10 duplicated status pills migrated across 6 admin schedule panels.
   - Template `rgba()` literals dropped from 195 to 109 (the ratchet in
     section 4 counts hex and `rgb()` too, so its budget is 139).

3. **Written standard** — `docs/design-system.md` is now the canonical design
   system: tokens, typography, chips/buttons, **status tones**, **shared
   primitives incl. the toggle spec**, **table header/cell recipe**, spacing and
   rules. Linked from `README.md`; `.github/instructions/web.instructions.md`
   now points at it and adds a shared-primitives table instead of drifting.

4. **Guards** in `src/web/src/__tests__/design-tokens.test.ts`
   - Visually hidden checkboxes outside `BaseToggle`: must be zero, whatever
     the class order.
   - Status-driven pills carrying a hardcoded colour: must be zero.
   - Hardcoded template colour literals (`rgba()`, `rgb()`, hex): ratchet
     budget of 139, may only decrease.

## Intended visual deltas

Standardizing 23 drifted copies onto one primitive necessarily changes pixels.
The full list, derived from the `de0dd508..HEAD` diff (an earlier version of
this log understated it as a single delta):

1. **Three size clusters collapse to two.** Baseline had 22x42 (10 uses),
   24x44 (5 uses) and 28x50 (8 uses). The 24x44 cluster folds into `sm`. Four
   of its five members are indented sub-option rows, where `sm` is
   semantically right; the fifth, `CoinForm.vue` "Private Coin", is a
   top-level form row and does shrink. Its knob also goes 20px to 16px and its
   knob colour `--text-primary` to `--text-secondary`.
2. **Knob grows 20px to 22px** on the four `md` settings toggles that used
   `after:h-5`.
3. **Track background unified on `--bg-input`.** Baseline used `bg-surface`
   (schedule panels) and `bg-[var(--bg-primary)]` (settings) as well as
   `bg-input`.
4. **Focus ring model unified on `peer-focus-visible`.** Of the 23 baseline
   toggles, **18** already used `peer-focus-visible` (10 schedule panels, the 7
   in `SettingsAccountSection`, and `SettingsAppearanceSection`), **3** used
   `focus-within` on the wrapper, which also rings after a mouse click
   (`AdminCatalogsSection` form row and both `AdminOIDCSection` toggles), and
   **2** had no ring at all (`CoinForm`, and the `AdminCatalogsSection` table
   cell). Net effect: 18 unchanged, 3 stop ringing after a mouse click, 2 gain
   a keyboard ring.
5. **Knob colour unified on `--text-secondary`**; one copy used
   `--text-primary`.
6. **Checked border unified on `--accent-gold`**; one copy used
   `--border-accent`.
7. **`AdminCatalogsSection` read-only table-cell toggle 28x50 to 22x42**,
   applying the documented "compact table cells use `sm`" rule.
8. **`AdminValuationSchedule` "pending" pill** moves from
   `rgba(243,156,18,...)`/`#f39c12` to the shared `warning` tone.

Knob geometry is unchanged from baseline: a 2px inset on all four edges of the
track's padding box in both sizes. An earlier attempt to "centre" it was a
regression caught in review and reverted.

Screenshots were waived by the owner for this slice, so none of the above was
visually confirmed.

## Verification (candidate `c864cea0`)

All rows below were run on the final candidate `c864cea0` unless the row says
otherwise.

| Gate | Result |
|---|---|
| `npm run lint` (`eslint . --ext .vue,.ts,.tsx --max-warnings 0`) | PASS |
| `npm run type-check` (`vue-tsc --build`) | PASS |
| `npm run test` | PASS — 1752 passed, 1 skipped (208 files) |
| `npm run build` | PASS — built in 2.79s, PWA precache 198 entries |
| `task check:delivery` | PASS — 91 subtests, governance 0 errors. Run on `29aadfd7`; every later commit touches only `src/web` and documentation, so no delivery-scoped path changed and the result carries over. |

`npm run build` initially could not run because `node_modules/@fontsource` was
absent. The owner authorized `npm ci` in `src/web` (688 packages), after which
`npm run build` passed: built in 2.79s, PWA generateSW, 198 precache entries,
`dist/sw.js` generated. Browser/mobile checks remain additional.

### Tamper tests (each guard broken, failure observed, reverted)

| Guard | Tamper | Result |
|---|---|---|
| Hand-rolled toggle check | Added a `peer sr-only` input to `CoinForm.vue` | 1 test failed |
| Status-pill shape check | Added the raw pill class to `CoinForm.vue` | 1 test failed |
| Colour ratchet | Added one `bg-[rgba(1,2,3,0.1)]` class | 1 test failed |
| `BaseStatusBadge` token colours | Hardcoded a tone background to `rgba(...)` | 2 tests failed |
| Hardened toggle check | Reordered to `class="sr-only peer"` (evades the old regex) | 1 test failed |
| Hardened status-pill check | Status pill coloured with a hex literal | 2 tests failed |
| Hardened colour ratchet | Added one bare `text-[#123456]` | 1 test failed |
| Knob geometry | Changed `sm` inset to `after:bottom-[3px]` | 1 test failed |
| Keyboard-only focus ring | Swapped `peer-focus-visible:` for `focus-visible:` | 1 test failed |

The `COLOR_BUDGET = 139` figure was derived by counting
`rgba?\(|#[0-9a-fA-F]{3,8}\b` inside the `<template>` block of every
non-test `.vue` file under `src/web/src`; the same expression the guard uses.

## Independent review and repairs (commit `a87c73ee`)

The first review of `29aadfd7` returned INCOMPLETE with two blocking findings —
both evidence gaps, not defects: no diff was supplied to bind the candidate, and
the production build had not been run. Both were remediated (full diff supplied
from the session workspace; `npm ci` + `npm run build` run with owner
authorization). The reviewer did not require an independent reviser, so the
author repaired the non-binding findings:

| Finding | Repair |
|---|---|
| F-1 attribute fallthrough onto the hidden input | `class`/`style` now go to the wrapper, other attrs to the input; test added |
| F-2 optional `label` allowed a nameless toggle | `label` is now a required prop |
| F-3 table-cell toggle contradicted the documented `sm` rule | switched to `sm`; aria-label now names the row's catalog |
| F-4 untested `@change` fallthrough on Public Collection | regression test pins the revert-on-uncheck semantics |
| F-5 guards evadable by class reorder or hex notation | all three guards rewritten and re-tamper-tested with evasions the old regexes missed; ratchet now counts `rgba()`, `rgb()` and hex, budget 139 |
| F-6 light-theme status contrast | colours unchanged; recorded as a known gap in `docs/design-system.md` §4 |
| F-9 knob not vertically centred | `md` knob centred and travel made symmetric in both sizes |

Answer to the reviewer's factual question, corrected after the reviewer found a
counter-example to an earlier over-general claim. Searching every baseline
schedule panel (`git grep -n 'form-label" for=' de0dd508 --
'src/web/src/components/admin/schedules/*.vue'`), exactly **one** toggle had an
associated text label: `AdminPurchaseReminderSchedule.vue` L9,
`<label class="form-label" for="reminder-check-enabled">`. Its `id` is
forwarded to the `<input>` through `BaseToggle`'s attribute pass-through, so
that association survives. Every other toggle's sibling `form-label` was an
orphan label before the change and still is. Click-to-toggle on the switch
itself is preserved in all cases because `BaseToggle` wraps its own label.

Gates re-run on `a87c73ee`: lint PASS, type-check PASS, 1751 passed / 1 skipped,
`npm run build` PASS.

## Not done in this slice

- Typography and uppercase-label migration page by page (slice 2+), which the
  owner said needs their screenshots.
- Table header drift (`text-sm`, `tracking-[0.05em]`) — documented as the
  standard, not yet migrated.
- The `WishlistPage` and `AuctionLotDetailModal` status pills use a different
  size/shape and were left for the page-by-page slice.

## Next action

Owner review of the eight intended visual deltas listed above, then agree
slice 2 scope for #784 and whether #766 comes next. No deployment or release
is authorized.
