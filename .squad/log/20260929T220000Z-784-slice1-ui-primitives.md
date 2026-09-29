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
     always renders a focus ring.
   - **All 23 hand-rolled `peer sr-only` toggles migrated** across 15 files
     (9 schedule panels, `SettingsAccountSection`, `SettingsAppearanceSection`,
     `AdminSchedulesSection`, `AdminOIDCSection`, `AdminCatalogsSection`,
     `CoinForm`). Zero remain outside the primitive.

2. **`BaseStatusBadge`** (`src/web/src/components/ui/BaseStatusBadge.vue`, new)
   - `tone` of `success | error | warning | info | neutral`, driven by new
     `--status-*-bg` / `--status-*-fg` tokens in `variables.css`.
   - 10 duplicated status pills migrated across 6 admin schedule panels.
   - Template `rgba()` literals dropped from 195 to 109.

3. **Written standard** — `docs/design-system.md` is now the canonical design
   system: tokens, typography, chips/buttons, **status tones**, **shared
   primitives incl. the toggle spec**, **table header/cell recipe**, spacing and
   rules. Linked from `README.md`; `.github/instructions/web.instructions.md`
   now points at it and adds a shared-primitives table instead of drifting.

4. **Guards** in `src/web/src/__tests__/design-tokens.test.ts`
   - Hand-rolled `peer sr-only` toggles: must be zero.
   - The raw status-pill class shape: must be zero.
   - Template `rgba()` literals: ratchet budget of 109, may only decrease.

## Intended visual delta (one)

Three toggle size clusters existed on `beta`, not two: 22x42 (10 uses), 24x44
(4 uses) and 28x50 (8 uses). The owner chose two sizes before that third cluster
was known. The 24x44 group was folded into `sm` because every one of its
instances is an **indented sub-option** row (for example "Include wishlist
items" under "Coin of the Day"), where the smaller size is semantically correct.
Two other small deltas: the `AdminValuationSchedule` "pending" pill moves from
`rgba(243,156,18,…)/#f39c12` to the shared `warning` tone, and the
`AdminSchedulesSection` ParcelApp toggle track moves from `--bg-surface` to the
standard `--bg-input`.

## Verification (this tree)

| Gate | Result |
|---|---|
| `npm run lint` (`eslint . --ext .vue,.ts,.tsx --max-warnings 0`) | PASS |
| `npm run type-check` (`vue-tsc --build`) | PASS |
| `npm run test` | PASS — 1749 passed, 1 skipped (208 files) |
| `task check:delivery` | PASS — 91 subtests, governance 0 errors |

`npm run build` was **not** run: `node_modules/@fontsource` is absent on this
machine, so the build fails to resolve `@fontsource/inter/300.css`. This is a
pre-existing environment gap unrelated to this change and needs an
owner-authorized `npm ci`. Browser/mobile checks remain additional.

### Tamper tests (each guard broken, failure observed, reverted)

| Guard | Tamper | Result |
|---|---|---|
| Hand-rolled toggle check | Added a `peer sr-only` input to `CoinForm.vue` | 1 test failed |
| Status-pill shape check | Added the raw pill class to `CoinForm.vue` | 1 test failed |
| Template `rgba()` ratchet | Added one `bg-[rgba(1,2,3,0.1)]` class | 1 test failed |
| `BaseStatusBadge` token colours | Hardcoded a tone background to `rgba(...)` | 2 tests failed |

## Not done in this slice

- Typography and uppercase-label migration page by page (slice 2+), which the
  owner said needs their screenshots.
- Table header drift (`text-sm`, `tracking-[0.05em]`) — documented as the
  standard, not yet migrated.
- The `WishlistPage` and `AuctionLotDetailModal` status pills use a different
  size/shape and were left for the page-by-page slice.

## Next action

Owner review of the one intended visual delta, then agree slice 2 scope for
#784 and whether #766 comes next. No deployment or release is authorized.
