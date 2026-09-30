# #784 slice 3 — casing rules and a shared data-table recipe

- **Branch:** `beta`
- **Base:** `13ab8ad4` (slice 2 reviewed PASS at `f265ff87`, record corrected)
- **Issue:** #784, UI fit and finish: consistent typography, badges and controls
- **Status:** implemented, awaiting independent review

## Scope

The two remaining structural items from the issue: mixed casing conventions on
labels and table headers, and the absence of a data-table header/cell style.

Not in scope, carried to slice 4: the 139 hardcoded colour literals, the
light-theme status-contrast gap, and folding `text-xs` into `text-sm`.

## What changed

### 1. One data-table recipe replaces 27 inlined ones

The header recipe had drifted into five near-identical variants spread over 27
tables — three different sizes of letter-spacing and two different font sizes,
all supposedly "the standard".

`main.css` gains a `.data-table` class carrying the typographic and border
parts of the recipe: `text-label`, weight 600, `uppercase`, `0.08em` tracking,
`--text-muted`, `line-height: 1.4`, left alignment and the bottom borders on
both `th` and `td`.

Two deliberate design choices:

- **Padding stays on the element.** Density genuinely varies between the dense
  admin schedule tables (`py-2`, `px-[0.35rem]`) and the roomier security and
  help tables (`py-3`, `px-[0.65rem]`). Folding padding into the class would
  have moved every table's box. It does not; only type changes.
- **The class is declared inside `@layer components`.** Tailwind utilities live
  in the later `utilities` layer, so per-cell overrides — `<th class="text-right">`,
  `<td class="align-top">`, `<td class="whitespace-nowrap">` — still win exactly
  as they did when the recipe was inlined. Declaring it unlayered, as the rest
  of `main.css` is, would have silently broken every such override. The
  mechanism was checked before writing the rule; the counts of affected cells
  first given here (5 headers, 4 cells) were never substantiated and are
  withdrawn. The concrete case is `CoinDetailValuationPage.vue`, whose three
  right-aligned headers survive the migration only because of the layer.

The selector is `table.data-table, .data-table table`. The second half exists
because the 11 help tables are rendered from markdown and cannot carry a class;
their container does instead.

| Location | Tables |
|---|---|
| `components/HelpSection.vue` | 11 (via wrapper) |
| `components/admin/schedules/*` | 11 |
| `components/admin/AdminSecuritySection.vue` | 2 |
| `pages/WishlistAvailabilityHistoryPage.vue` | 2 |
| Other admin | 1 |

### 2. 38 recipes for uppercase labels became 1

An uppercase element carrying a small size token is a label. All of them now
read `text-label font-semibold uppercase tracking-[0.08em]`; only the colour
and the alignment vary by role.

42 elements across 20 files changed. Distinct recipes went from **38 to 17**,
and 87 of the 92 in-scope elements are now the standard plus a colour and an
optional alignment.

Five uppercase elements are deliberately untouched, because they carry no
small size token and are not labels:

| Element | Why excluded |
|---|---|
| `AdminSystemSection.vue:423` `<td>` | A data cell whose value is uppercase |
| `CollectorProfileSection.vue:23` `<input>` | Uppercases the currency code the user types |
| `CollectionHealthScorecard.vue:9` grade badge | A badge at `text-base`; section 3 governs badges |
| 2 inline `<span>` role words | Inherit their size from the parent; tracking normalised to `0.08em`, size left alone |

`.badge` also sets `text-transform: uppercase` in CSS, so a template count
undercounts what actually renders uppercase. Those are governed by
`BaseStatusBadge` from slice 1. Noted in `docs/design-system.md`.

### 3. Guards

Two zero-tolerance guards in `design-tokens.test.ts`:

- `styles every uppercase label with the one documented recipe` — for any
  template element that is `uppercase` **and** carries one of `text-label`,
  `text-sm`, `text-xs`, `text-chip`, `text-micro`, `text-2xs`, requires
  `text-label` + `tracking-[0.08em]` + `font-semibold`.
- `styles table headers through .data-table rather than inline recipes` —
  forbids `[&_th]:uppercase`, `[&_th]:tracking-[…]`, `[&_th]:font-semibold`
  and `[&_th]:text-text-muted` in templates.

**Both were tamper-tested.** Changing one `text-label uppercase tracking-[0.08em]`
to `text-sm uppercase tracking-[0.05em]` in `AdminUsersSection.vue` failed the
first guard with the expected message (`text-sm (want text-label), missing
tracking-[0.08em]`). Adding `[&_th]:uppercase` back to a table in
`AdminSecuritySection.vue` failed the second. Both injections were reverted and
the file re-verified byte-identical.

The first tamper attempt did **not** apply — the token order in the file was
not the order I assumed — and the guard passed. That pass was a false
negative from a failed injection, not evidence; the tamper was redone against
the real string before the guard was accepted.

These guards inherit the same coverage holes as the slice 2 pair: they regex
the `<template>` block of `.vue` files only. See "What the guards do and do not
catch" in `docs/design-system.md`.

### 4. Documentation

`docs/design-system.md` gains an "Uppercase labels" subsection and a rewritten
section 6 for tables, including the layer rationale and the markdown-wrapper
case. The section 8 rules now name the label recipe and the `data-table` class.
`.github/instructions/web.instructions.md` mirrors both.

## Intended visual deltas

Nothing here moves a box except where stated. The changes are type only.

### Table headers — 27 tables

| Before | Tables | After | Effect |
|---|---|---|---|
| `[&_th]:text-sm` (0.75rem) | 20 | `0.7rem` | **−6.7%**, visible |
| `[&_th]:text-label` (0.7rem) | 7 | `0.7rem` | No size change |
| `tracking-[0.03em]` | 11 | `0.08em` | Wider, visible |
| `tracking-[0.05em]` | 13 | `0.08em` | Wider, visible |
| `tracking-[0.08em]` | 3 | `0.08em` | No change |

Line-height on headers is now an explicit `1.4` for all 27. The 20 that were
`text-sm` were on Tailwind's `1.4286` — imperceptible. The 7 that were
`text-label` inherited `body`'s `1.6` and now tighten to `1.4`, which shortens
each header row slightly. Those 7 are the 2 security tables, 2 valuation and
availability tables, and 3 dense schedule tables.

**The 11 help tables and the 9 admin schedule tables are where the header
change is most visible**: smaller and more widely spaced at the same time.

### Labels — 42 elements in 20 files

| Change | Elements | Effect |
|---|---|---|
| `text-sm` → `text-label` | 27 | 0.75 → 0.7rem, **−6.7%** |
| `text-xs` → `text-label` | 7 | 0.75 → 0.7rem, **−6.7%** |
| `text-chip` → `text-label` | 4 | 0.8 → 0.7rem, **−12.5%** |
| tracking 0.03/0.04/0.05em → 0.08em | 14 | Wider |
| weight 400 → 600 | 15 | Bolder |

Several elements take more than one of these at once; the largest single change
is a `text-chip` label that also gains tracking and weight.

### Line-height on the 34 converted labels

Same mechanism as slice 2's blocking finding, and it applies again here.
`text-sm` and `text-xs` are Tailwind-default step names, so they emit a
`line-height` as well as a font size. `text-label` is a custom name and emits
font size only, so a converted element falls back to `body { line-height: 1.6 }`.

| Conversion | Elements | Line-height |
|---|---|---|
| `text-sm` → `text-label` | 27 | 1.4286 → 1.6, **+12%** |
| `text-xs` → `text-label` | 7 | 1.3333 → 1.6, **+20%** |
| `text-chip` → `text-label` | 4 | 1.6 → 1.6, no change |

These 34 labels get a taller line box — about 2px at this size. In a flex row
with `items-center` that is invisible; in a tight grid it can nudge a row.

**I chose not to set `--text-label--line-height`.** Doing so would have pinned
these 34 at 1.4 but changed the other 46 `text-label` elements already in the
app (80 total, 79 without an explicit `leading-*`) from 1.6 to 1.4. Trading a
34-element change for a 79-element one, in a slice whose purpose is
consistency, is the wrong direction. All 80 now render at 1.6.

No test in the suite covers computed line-height. This is declared, not tested.

## Verification (candidate: the commit that adds this log)

| Gate | Result |
|---|---|
| `npm run lint` (`eslint . --ext .vue,.ts,.tsx --max-warnings 0`) | PASS |
| `npm run type-check` (`vue-tsc --build`) | PASS |
| `npm run test` | PASS — 1756 passed, 1 skipped (208 files) |
| `npm run build` | PASS — PWA precache 198 entries |

The built CSS was inspected directly to confirm the class is real and correctly
layered, rather than trusting the suite:

```
@layer components{ … }
table.data-table,.data-table table{border-collapse:collapse;width:100%}
.data-table th{font-size:var(--text-label);text-transform:uppercase;letter-spacing:.08em;color:var(--text-muted);text-align:left;border-bottom:1px solid var(--border-subtle);font-weight:600;line-height:1.4}
.data-table td{text-align:left;border-bottom:1px solid var(--border-subtle)}
```

Counts in non-test `.vue` templates: inlined table header recipes **27 → 0**,
distinct uppercase label recipes **38 → 17**, non-`0.08em` tracking values
**26 → 0**.

**Correction (repair commit).** The 27 above counted only tables that inlined
the recipe as `[&_th]:` variants on the `<table>` element. Five further tables
spelled the same recipe out on each individual `th`, so the codemod's scan and
the first version of guard 2 both missed them, and the claim "every data table
carries `data-table`" was false at the commit that introduced the rule. The
five — `AdminUsersSection.vue`, `AdminCatalogsSection.vue`,
`SettingsShipmentsSection.vue`, `CoinDetailValuationPage.vue` and
`AdminSystemSection.vue` — are migrated in the repair. The true figure is
**32 → 0**. Guard 2 now also rejects the per-`th` spelling and any `<table>` in
a template that never mentions `data-table`; both halves were tamper-tested,
with the injection verified by `grep` before the run this time.

Go, Python and delivery gates were not run: no file outside `src/web`,
`docs/` and `.github/instructions/` changed.

## Next action

Independent review, then owner review of the running app. Check the admin
schedule panels and the help content first — that is where the header change is
largest. Then slice 4: the 139 colour literals, the light-theme
status-contrast gap, and `text-xs`.

The reviewer flagged in advance that folding `text-xs` into `text-sm` is **not**
a no-op despite both resolving to `0.75rem`, because their Tailwind
line-heights differ (1.333 vs 1.4286). Slice 4 must handle that deliberately.

No deployment or release is authorized.

## Repair (independent review returned BLOCK)

The review blocked on B1 and raised six non-binding findings. All are addressed
here, in the same slice, by the author. Only the blocking reviewer can clear B1.

### B1 — the rule was false at the commit that introduced it

Five tables inlined the header recipe on each `th` instead of as `[&_th]:`
variants on the `<table>`. The codemod never saw them, guard 2 never matched
them, and guard 1 *passed* them, because per-`th` they carry a legitimate
`text-label` + `font-semibold` + `tracking-[0.08em]`. Worse, the honest caveat
in `docs/design-system.md` §6 — that some tables still used drifted headers and
would be migrated page by page — was deleted in the same commit, converting a
known remainder into a misstatement.

All five are migrated rather than excepted. New visible deltas:

| Table | Delta |
|---|---|
| `AdminUsersSection.vue` | Header line-height 1.6 → 1.4. Row rules move from `td` to the shared rule; unchanged visually. |
| `AdminCatalogsSection.vue` | Same. `align-top` on cells preserved as a `[&_td]:` utility. |
| `SettingsShipmentsSection.vue` | Same, plus the row rule moves from `tr` `border-t` to `td` `border-b`: identical between rows, but the last row now has a rule under it where it had none. |
| `CoinDetailValuationPage.vue` | Headers were inheriting `1.4286` from the table's `text-sm`; they now take the explicit `1.4`. The three `text-right` headers keep their alignment — this is the case `@layer components` exists for. The header row's `border-b` moves to the `th`, removing what would have been a doubled rule. |
| `AdminSystemSection.vue` | **The largest change in the slice.** Its headers were never on the standard at all: `p-2`, `--text-muted` on the `thead`, no uppercase, no tracking, `text-sm`. They now become `0.7rem`, uppercase, `0.08em`, weight 600, with a bottom border. Row rules move from `tr` `border-t` to `td` `border-b`, again adding a rule under the last row. Look at this one first. |

`docs/design-system.md` §6, `.github/instructions/web.instructions.md` and the
guard now say the same thing, including what the guard does *not* catch: its
containment check is file-scoped, so a file mixing a migrated and an unmigrated
table would still pass.

### N1 / N2 — BaseBadge and `.badge`

`BaseBadge.vue` is a shared primitive that was counted as one element. Its real
reach is **2 call sites**, both status badges (`DeepAnalysisPage.vue:28`,
`DeepAnalysisHistoryPage.vue:39`) — far smaller than the review feared, but it
should have been stated, not left implied.

The related divergence was real: `BaseBadge` moved to `0.7rem` / `0.08em` while
the unlayered `.badge` class in `main.css`, used at **12 sites**, stayed at
`0.75rem` / `0.05em`. Since #784 names badges explicitly, `.badge` is converged
onto the same recipe rather than logged as debt. Delta: those 12 badges shrink
`0.75rem` → `0.7rem` (−6.7%), tracking widens `0.05em` → `0.08em`, and they gain
an explicit `line-height: 1.4` where they previously inherited `1.6`.

### N3 — the line-height delta table overstates some rows

Line-height inherits. An element converted to `text-label` inside an ancestor
that sets an explicit `text-sm` lands at `1.4286`, not at `body`'s `1.6`. The
"+12%" and "+20%" rows above assume no such ancestor and are an upper bound,
not a universal result. The clearest instance was `CoinDetailValuationPage`'s
table, and the repair now pins those headers at `1.4` explicitly.

### N4 — two elements gained tracking from `normal`

Two converted elements had no `tracking-*` at all and so were at `normal`,
rather than moving between two explicit values. `CalendarPage.vue:196` is one
of them and appeared in no delta row. Going `normal` → `0.08em` on an uppercase
label is the intended direction, but it is a visible change and belonged in the
table.

### N5 — the counts do not reproduce

Declared: 42 elements across 20 files. The reviewer counted 43 across 22. A
count of the commit's own diff finds 36 single-line conversions plus an
unknown number spread over multi-line `class` attributes, and 33 `.vue` files
touched in total including table-only changes. None of the three agree. The
figure is approximate and is recorded as approximate; the guard, not the
count, is what holds the invariant.

### Verification

`npm run type-check`, `npm run lint`, `npm run test` (207 files, 1756 passed,
1 skipped) and `npm run build` all pass on the repaired tree. Guard 2 was
tamper-tested in both of its new halves — removing `data-table` from a table,
and re-inlining the recipe on a `th` — each verified with `grep` to have
actually applied before the run, each failing as intended, each reverted.
