# #784 slice 2 — typography scale and colour utilities

Date: 2026-09-29
Branch: `beta`
Baseline: `af2df567` (slice 1, reviewed PASS)
Issue: #784 "UI fit and finish: consistent typography, badges and controls"

## Scope, as chosen by the owner

- Typography: every arbitrary font size in the app, in one change (not
  page-by-page).
- Colour: the `text-[var(--…)]` classes migrated to theme utilities.
- The scale is **extended** with real display steps so hero numerals keep their
  size rather than being snapped down.
- Screenshots: the owner reviews the running app after it lands.

Out of scope: `bg-[…]` and `border-[…]` arbitrary values, uppercase/casing
rules, table density, and the remaining 139 colour literals.

## What changed

### 1. The scale is now complete and explicit

`src/web/src/assets/styles/main.css` grew five steps, so that every size in
the app is a named token:

| New step | Size | For |
|---|---|---|
| `--text-2xs` | `0.55rem` | dense data grids at phone widths |
| `--text-micro` | `0.65rem` | dense data grids, micro-chips |
| `--text-md` | `1.1rem` | card and modal titles |
| `--text-3xl` | `2.5rem` | hero numerals |
| `--text-4xl` | `3rem` | single headline numeral |

### 2. 170 arbitrary sizes became scale steps

All 170 `text-[Nrem]` uses across 39 files were mapped onto the scale,
responsive variants included. The source values were
`0.55 / 0.65 / 0.7 / 0.72 / 0.75 / 0.78 / 0.8 / 0.82 / 0.85 / 0.88 / 0.95 /
1 / 1.05 / 1.1 / 1.35 / 1.75 / 1.9 / 2.25 / 2.5 / 2.75 / 3` rem — eleven of
them within `0.05rem` of a neighbour, which is the accidental drift the issue
describes. `0.82rem` alone accounted for 68 uses.

Two arbitrary sizes remain and are intentional: `[&_code]:text-[0.88em]` in
`CoinSearchChat.vue` and `[&_code]:text-[0.85em]` in `FeaturedCoinModal.vue`.
Both size rendered markdown relative to its container, which is a ratio rather
than a scale step, so the guard permits `em` and forbids `rem`/`px`/`pt`.

### 3. 154 raw `var()` text classes became utilities

`text-[var(--color-negative)]` → `text-loss`, and so on for 17 distinct
tokens across 44 files. Five material tokens (`--mat-*`) and three showcase
felt tokens (`--felt-*-bright`) had no utility, so they were added to the
`@theme inline` block; the rest already existed. The rendered colour is
identical in every case — this is an auditability change, not a visual one.

### 4. Guards

Two new zero-tolerance guards in `design-tokens.test.ts`:

- `uses no arbitrary absolute text sizes in templates` — forbids
  `text-[Nrem|px|pt]`, permits `em`.
- `uses theme colour utilities rather than raw var() text classes` — forbids
  `text-[var(--…)]`.

Both were tamper-tested: injecting `text-[1.03rem]` and
`text-[var(--accent-gold)]` into `CoinCard.vue` each failed exactly one test
naming the file and token, and both injections were reverted.

### 5. Dead responsive variants removed

Eight admin schedule tables carried `text-[0.8rem] md:table-fixed
md:text-[0.82rem]` — a real, if tiny, responsive step at baseline. Both legs
mapped to `text-chip`, at which point `md:text-chip` became a no-op duplicate
of its own base. The dead variant was then removed from all eight, matching
how the now-redundant `md:text-2xl` was handled in `AdminHealthSection.vue`.
The removal itself is zero-change; the underlying `0.82 → 0.8` shift at `md`
is covered by the imperceptible bucket above.

### 6. Documentation

`docs/design-system.md` section 2 was rewritten around the 13-step scale and
gained a template-colour rule. `.github/instructions/web.instructions.md`
carries the same table.

## The `text-xs` trap, and why the step is called `text-micro`

The first attempt defined the dense-data step as `--text-xs: 0.65rem`. That
was wrong and was caught before commit by checking the built CSS. `text-xs` is
a **Tailwind built-in** resolving to `0.75rem`, and it already had **35 call
sites** in this app that were never part of this migration. Redefining the
token would have silently shrunk all 35 by 13 per cent — an undeclared
regression far outside the agreed scope.

The step was renamed `--text-micro`, and the five dense-grid call sites were
retargeted by baseline line number. Verification: `text-xs` occurrences are
**35 before and 35 after**, the built CSS still emits `--text-xs:.75rem`, and
`text-xs` appears nowhere outside Vue templates.

## Occurrence counts before the commit, for every redefined step name

`text-xs` showed that redefining a Tailwind-default token name silently
changes every existing call site. These are the baseline (`af2df567`) counts
under `src/web/src`, so the same trap can be ruled out by inspection rather
than by trust:

| Name | Baseline uses | Status in this slice |
|---|---|---|
| `text-3xl` | **0** | newly defined, 1.875rem → 2.5rem — no call site could be affected |
| `text-4xl` | **0** | newly defined, 2.25rem → 3rem — no call site could be affected |
| `text-5xl` | 0 | not defined; still Tailwind's default |
| `text-xs` | 35 | **left at Tailwind's 0.75rem**; the dense step is `--text-micro` |
| `text-sm` | 342 | redefined in an earlier slice, unchanged here |
| `text-base` | 286 | redefined in an earlier slice, unchanged here |
| `text-lg` | 118 | redefined in an earlier slice, unchanged here |
| `text-xl` | 47 | redefined in an earlier slice, unchanged here |
| `text-2xl` | 9 | redefined in an earlier slice, unchanged here |

Only `text-3xl` and `text-4xl` are newly defined by this slice, and both had
zero prior uses, so neither could shift existing markup. The five names
redefined by the pre-existing scale are untouched here.

This leaves a real, recorded gap: `text-xs` is a duplicate of `text-sm` at
`0.75rem` and is not a step on the documented scale. Folding it into
`text-sm` is a follow-up, noted in `docs/design-system.md` section 2.

## Intended visual deltas

Screenshots were waived; the owner reviews the running app. Nothing below is
visually confirmed.

**No change (exact matches).** `0.55 → text-2xs`, `0.65 → text-micro`,
`0.7 → text-label`, `0.75 → text-sm`, `0.8 → text-chip`, `0.85 → text-body`,
`0.9 → text-base`, `1.1 → text-md`, `2.5 → text-3xl`, `3 → text-4xl`.

**Imperceptible, under 3 per cent — 111 uses.** `0.82 → 0.8` (68 uses),
`0.88 → 0.9` (23), `0.78 → 0.8` (17), `0.72 → 0.7` (3). At these sizes the
shift is well under one pixel.

**Small, about 5 per cent — 17 uses.** `0.95 → 0.9` (15 uses, secondary body
text shrinks slightly), `1.05 → 1.1` (1), `1.9 → 2` (1, the stats
value-change numeral).

**Noticeable, 9 to 14 per cent — 6 uses, listed individually.**

1. `CoinCard.vue` coin title `1rem → text-md` (1.1rem), **+10 per cent**.
   This is the Collection page card title. It is line-clamped to two lines,
   so longer names may truncate marginally sooner. Changed for consistency
   with every other card title in the app, which are all `1.1rem`.
2. `SetDashboardCard.vue` set name at `min-[561px]`, `1.35 → text-xl`
   (1.5rem), **+11 per cent**.
3. `SetDashboardCard.vue` coin count, **both legs of the responsive pair**:
   `2.25 → text-3xl` (2.5rem) below 561px, **+11 per cent**, and
   `2.75 → text-4xl` (3rem) at 561px and above, **+9 per cent**. The numeral
   therefore grows at every width. An earlier version of this record declared
   only the upper leg and misattributed the lower one.
4. `CollectionHealthScorecard.vue` score: **no size change.** `3rem → text-4xl`
   and `max-md:2.5rem → max-md:text-3xl` are both exact matches. Its only
   delta is the line-height noted below. An earlier version of this record
   wrongly attributed the +11 per cent from item 3 to this component, which
   would have pointed a reviewer at the wrong screen.
5. and 6. `AdminHealthSection.vue`, both stat numerals, `1.75 → text-2xl`
   (2rem), **+14 per cent at phone widths only**. These already jumped to
   `2rem` at `md`, so the now-redundant `md:text-2xl` was removed and the
   numeral is 2rem at every width. Growing rather than shrinking was chosen
   so no hero numeral gets smaller; the values are short (`87`, `12.5%`) in
   a `p-6` card.

**Line-height changes on 37 elements, including 8 form controls.** This was
first recorded as affecting a single element. That was wrong twice over — the
count and the component — and the corrected set is below.

Tailwind-default step names emit a line-height as well as a font size, because
`@theme` overrides only the `--text-N` size key and leaves
`--text-N--line-height` standing from Tailwind's own theme
(`node_modules/tailwindcss/theme.css`). The custom names (`2xs`, `micro`,
`label`, `chip`, `body`, `md`) have no such pairing and emit font-size only.
So any element that moved onto a default-named step **and does not set
`leading-*` itself** gains a line-height it previously inherited from
`body { line-height: 1.6 }`.

45 of the 170 conversions landed on a default-named step; 8 set `leading-*`
and are unaffected; **37 change**. The counting convention: a conversion is
excluded when the target token was already present on that line at baseline —
this drops `AdminHealthSection.vue`'s two stat numerals, which already carried
`md:text-2xl`. Counting them in gives 47 / 10 / 37; the consequential number
is 37 either way.

| File | Elements | Conversion | Line-height |
|---|---|---|---|
| `auction/AuctionLotDetailModal.vue` | 20 | `0.88rem → text-base` | 1.6 → 1.5 |
| `coin/CoinInfoGrid.vue` | 9 | `0.95rem → text-base` | 1.6 → 1.5 |
| `pages/FollowersPage.vue` | 3 | `0.95rem → text-base` | 1.6 → 1.5 |
| `stats/CollectionHealthScorecard.vue` | 2 | `3rem → text-4xl`, `2.5rem → text-3xl` | 1.6 → 1.1111 / 1.2 |
| `settings/SettingsToolsSection.vue` | 1 | `0.95rem → text-base` | 1.6 → 1.5 |
| `AuctionLotCard.vue` | 1 | `0.75rem → text-sm` | 1.6 → 1.4286 |
| `sets/SetDashboardCard.vue` | 1 | `1.35rem → text-xl` (`min-[561px]` only) | 1.6 → 1.4 |

**8 of the 20 in `AuctionLotDetailModal.vue` are `<input class="form-input …">`
controls, so this is a form-layout change, not only a text change.**
`.form-input` in `main.css` sets padding and `font-size: 0.9rem` but **no**
`line-height`, so the control inherited `1.6`. It now computes `1.5`, which
shrinks each control's content box by about `0.09rem` (roughly 1.4px) and
therefore its rendered height. The two `<textarea>` elements in the same modal
set `leading-[1.45]` and are unaffected. **Check the lot edit form first when
reviewing the running app.**

One entry in the "No change (exact matches)" list above needs this caveat:
`0.75 → text-sm` is exact *in font size*, but the single `AuctionLotCard.vue`
element carrying it has no `leading-*`, so its line box still tightens.

The effect throughout is tighter line spacing, most visible in multi-line
text. It is a consistency gain rather than a new inconsistency: `text-base`
already had 286 call sites at baseline, all rendering at 1.5, so the 33
converted `text-base` elements now match the majority instead of differing
from it. No test in the suite covers computed line-height, so none of this is
verified by the gates.

**No colour deltas.** Section 3 above renders identically.

## Verification (candidate: the commit that adds this log)

| Gate | Result |
|---|---|
| `npm run lint` (`eslint . --ext .vue,.ts,.tsx --max-warnings 0`) | PASS |
| `npm run type-check` (`vue-tsc --build`) | PASS |
| `npm run test` | PASS — 1754 passed, 1 skipped (208 files) |
| `npm run build` | PASS — built in 2.70s, PWA precache 198 entries |

Three pre-existing tests asserted the old class strings and were updated to
assert the new ones: two in `StatsValueOverTime.test.ts`
(`text-[var(--color-positive|negative)]` → `text-gain` / `text-loss`) and one
in `ui-patterns.test.ts` (`min-[561px]:text-[2.75rem]` → `min-[561px]:text-4xl`).
These are assertions about class names, not about behaviour.

Beyond the suite, the built CSS was inspected directly to prove the tokens are
real rather than silently dropped: every new step emits
(`--text-2xs:.55rem` … `--text-4xl:3rem`) and every new colour utility emits
(`.text-loss{color:var(--color-negative)}`, `.text-mat-gold`, and so on).

Counts in non-test `.vue` templates: arbitrary sizes **170 → 0**, raw `var()`
text classes **154 → 0**. Colour literals are unchanged at 139, still at the
slice 1 budget.

Go, Python and delivery gates were not run: no file outside `src/web` and
`docs/` changed.

## Independent review

Reviewer `aurearia-reviewer` (read-only) reviewed `58cbb684` and returned
**BLOCK** on three findings: the line-height set was declared as 1 element when
it is 37 (8 of them `form-input` controls that change height); deltas 3 and 4
were attributed to the wrong components; and the "`text-3xl`/`text-4xl` had no
prior call sites" claim was unevidenced. All three were repaired, along with
two non-binding findings, in `f265ff87`.

Re-review bound to **`f265ff87`**: **PASS**, block lifted, findings 1–3
cleared. Three further non-binding record-hygiene notes were raised (a
misquoted baseline class string, a stale "Next action", and the counting
convention behind the 45/8/37 split); all three are corrected in this log.

A PASS is a review verdict only — not owner acceptance, not merge permission,
and not release authorization.

## Next action

Owner review of the running app. Check the auction lot edit form first: eight
`form-input` controls there change height because they moved from an inherited
`1.6` line-height to `text-base`'s `1.5`. Then the individually listed size
changes — note that the health scorecard has **no** size change, only a
line-height change. Then agree whether #784 continues into casing/table
density or #766 comes next. #766 still needs its own spec. No deployment or
release is authorized.
