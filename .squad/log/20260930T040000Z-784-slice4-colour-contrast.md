# #784 slice 4 — colour tokens, overlays and the light-theme contrast gap

Issue: #784 UI fit and finish: consistent typography, badges and controls.
Branch: `beta`. Author: implementation owner. Slices 1-3 are reviewed PASS
(`af2df567`, `f265ff87`, `8dd0dbb7`).

## What this slice does

Three things, in order of risk.

### 1. The light theme was failing WCAG AA on every status colour

This is the only correctness defect in the slice; everything else is tidying.

`variables.css` defined the status foregrounds once in `:root`, tuned for the
dark themes, and `[data-theme="light"]` overrode **none** of them. Measured
against the light theme's `--bg-card` (`#ffffff`):

| Token | Dark-theme value | Ratio on white | AA 4.5:1 |
|---|---|---|---|
| `--color-positive` | `#2ecc71` | 2.10:1 | fail |
| `--text-warning` | `#f5c36a` | 1.63:1 | fail |
| `--status-info-fg` | `#3498db` | 3.15:1 | fail |

Eight foregrounds now have light-theme overrides. Each was chosen to clear
4.5:1 against **both** `--bg-card` (`#ffffff`) and `--bg-primary` (`#f5f0e8`),
because status text appears on both surfaces:

| Token | Light value | on `#ffffff` | on `#f5f0e8` |
|---|---|---|---|
| `--color-positive` | `#157347` | 5.87:1 | 5.18:1 |
| `--color-negative` | `#c0392b` | 5.44:1 | 4.79:1 |
| `--text-warning` | `#8a6100` | 5.54:1 | 4.88:1 |
| `--status-info-fg` | `#1f6a9e` | 5.81:1 | 5.12:1 |
| `--status-neutral-fg` | `#5d6d6e` | 5.42:1 | 4.77:1 |
| `--confidence-high` | `#2f7a4d` | ≥4.5:1 | ≥4.5:1 |
| `--confidence-medium` | `#8a6100` | ≥4.5:1 | ≥4.5:1 |
| `--confidence-low` | `#a32316` | ≥4.5:1 | ≥4.5:1 |

The guard **computes** the ratio from the token values rather than asserting
the numbers above by hand, so it keeps holding if someone retunes a value. It
was tamper-tested in both halves: substituting a low-contrast value fails, and
deleting an override fails. Both reverted.

Visible delta: **every status colour in the light theme changes.** The greens
darken noticeably. Nobody has looked at this; see "Next action".

#### Follow-up: the guard was the wrong shape, and the gap was wider

I scoped both the fix and the guard to `[data-theme="light"]`, because that is
where the failure was obvious. That was wrong. There are six themes, and the
guard should check the value each theme *resolves* to — its own override, or
`:root`'s if it has none — rather than assuming only the light theme can fail.

Re-measuring all six with the corrected shape found two more failures that the
light-only guard could never have caught:

| Theme | Token | On `--bg-card` |
|---|---|---|
| `louvre` | `--color-negative` `#e74c3c` | 4.10:1 |
| `modern-greek` | `--color-negative` `#e74c3c` | 4.48:1 |

`:root`'s `--color-negative` moves `#e74c3c` → `#f06a5a`, which clears 4.5:1 on
every dark theme's card and page (louvre 5.16/5.97, modern-greek 5.63/6.41) and
is still recognisably the same red. The light theme keeps its own `#c0392b`.
One token change fixes both themes; per-theme overrides would have left five
themes on one red and one on another.

Visible delta: **the error/loss red lightens slightly on all five dark themes**,
not just the two that failed.

The guard now iterates every `[data-theme]` block, resolves each token through
`:root`, and checks both surfaces. Tamper-tested twice: restoring `#e74c3c`
fails naming `louvre` and `modern-greek` specifically, and deleting the light
theme's `--color-negative` override fails because light then resolves to the
dark value at 3.04:1. Both reverted.

The lesson is worth recording: the defect was not the colour, it was scoping a
guard to the one case I had already found.

### 2. Overlay and status-variant tokens

Added `--overlay-20` … `--overlay-60`, `--status-*-border` (foreground at 0.3)
and `--status-{success,error}-tint` (at 0.1), with matching `@theme inline`
utilities. The overlay names are numeric deliberately — the call sites share no
semantic hierarchy, and naming them `scrim` or `veil` would invent one.

Template colour literals: **139 → 61**. 33 overlays, 30 status literals, 7
tints and 8 hardcoded `#c9a84c` golds converted.

Two literals that *look* convertible were deliberately left alone. `#3498db`
and `#9b59b6` appear in `CollectionHealthScorecard`'s grade gradients, not as
status colours; they match `--status-info-fg` and a purple by value only.
Value-identity is not grounds for conversion.

### 3. The colour budget is now per file

The slice-3 reviewer pointed out that a net total is evadable: removing a
literal in one component pays for adding one in another. `COLOR_BUDGET` is now
a per-file map, a file absent from it must have none, and the failure message
names the file. Tamper-tested with exactly the swap the old guard missed —
one literal removed from `AuctionLotDetailModal` and one added to
`NotesPage` — which nets zero and now fails on `NotesPage`. Reverted.

## What this slice does not do

**The `text-xs` fold is deferred.** `text-xs` and `text-sm` both resolve to
`0.75rem`, so folding 35 call sites looks free. It is not: their Tailwind
line-heights are 1.3333 and 1.4286, so every one of those 35 elements would
get 7% taller. That belongs in a typography slice with its own declaration,
not bundled into a colour slice.

**The remaining 61 literals need an owner decision, not more refactoring.**
They are not oversights; they are a genuine palette conflict that cannot be
resolved without seeing the app. The app carries near-duplicate values with no
single winner:

- greens: `#2ecc71`, `#27ae60`, `#229954`, `#4ade80` (5)
- reds: `#e74c3c` (4), `#f87171` (4), `#ef4444`, `#c0392b`, `#450a0a`
- ambers: `#f39c12` (2), `#f59e0b` (4), `#e67e22`, `#f97316`
- greys: `#6b7280` (6), `#7f8c8d`, `#999999`
- purples/blues: `#8b5cf6`, `#8e44ad`, `#6366f1`, `#3b82f6`, `#6496ff`, `#ec4899`

`AuctionLotDetailModal.vue` alone holds 16 and uses a Tailwind-ish palette
distinct from the rest of the app. `CollectionHealthScorecard.vue` holds 16,
most of them grade gradients. Picking which green wins is a design call.

## Verification

| Gate | Result |
|---|---|
| `npm run type-check` | clean |
| `npm run lint` | clean at `--max-warnings 0` |
| `npm run test` | 207 files, 1758 passed, 1 skipped |
| `npm run build` | succeeded |

Go, Python and delivery gates were not run: nothing outside `src/web`, `docs/`
and `.github/instructions/` changed.

Guards tamper-tested in this slice: the two WCAG contrast halves, and the
per-file colour budget. Each injection was verified present before the run.

## Also in this commit

Three follow-ups the slice-3 reviewer raised as non-blocking, closed here
because they are one-line changes to files this slice already touches:

- `SettingsShipmentsSection.vue` — the last row's new `border-b` sat directly
  on the container's own bottom border, drawing a doubled 2px line. Suppressed
  with `last:[&>td]:border-b-0`.
- `AdminUsersSection.vue`, `AdminCatalogsSection.vue` — removed leftover
  per-`td` `border-b`/padding/`align-top` classes that `.data-table` and the
  table-level `[&_td]:` utilities now supply. Rendering-identical.
- The reviewer's `border-collapse` question is **settled, not assumed**:
  `dist/assets/index-*.css` contains Tailwind preflight's
  `table{text-indent:0;border-color:inherit;border-collapse:collapse}`, so the
  two tables that gained `border-collapse` from `.data-table` were already
  collapsed. No layout shift.

## Next action

Independent review, then the owner's visual pass — which is now overdue across
four slices. **Switch to the light theme first.** Every status colour in it
changes in this slice, and no one has seen any of it; screenshots were waived
twice. After that, the 61-literal palette decision above.

No deployment or release is authorized.


## Second repair — independent review returned BLOCK

The review blocked at `bb866804`. The blocker was correct and the diagnosis was
sharper than my own: I had fixed the *scope* of the guard but not its *shape*.

### The blocker — the guard measured the wrong surface

`BaseStatusBadge.vue:15-18` pairs `color: var(--status-{tone}-fg)` with
`background: var(--status-{tone}-bg)`, and every `-bg` is a ~0.15 alpha fill.
The text therefore renders on **fill over surface**, not on the bare surface.
The fill shifts the surface toward the foreground's own hue, so it always
*reduces* contrast — measuring the bare surface was not conservative, it was
simply the wrong measurement. The guard was certifying pairs that were really
4.00:1.

Recomputing with the fill composited found failures the previous two versions
of this guard both passed:

| Theme | Pair | Composited |
|---|---|---|
| `light` | error on `--bg-primary` | 4.00:1 |
| `light` | success on `--bg-primary` | 4.16:1 |
| `light` | neutral on `--bg-primary` | 4.29:1 |
| `light` | info on `--bg-primary` | 4.44:1 |
| `light` | error on `--bg-card` | 4.49:1 |
| `default` | info on `--bg-card` | 4.04:1 |
| `louvre` | info on `--bg-card` | 4.05:1 |
| `louvre` | error on `--bg-card` | 4.35:1 |
| `modern-greek` | info on `--bg-card` | 4.36:1 |
| `british-museum` | info on `--bg-card` | 4.49:1 |

Note `louvre`'s error at 4.35:1. The previous commit raised its *bare* ratio
from 4.10 to 5.16 and I recorded that as fixed. The badge was still failing.

Five token values were retuned to clear 4.5:1 composited, on both surfaces, in
all seven theme states:

| Token | Was | Now |
|---|---|---|
| `:root --color-negative` | `#f06a5a` | `#fa6e5e` |
| `:root --status-info-fg` | `#3498db` | `#38a4ed` |
| light `--color-negative` | `#c0392b` | `#ad3327` |
| light `--status-info-fg` | `#1f6a9e` | `#1e6698` |
| light `--status-neutral-fg` | `#5d6d6e` | `#576667` |
| light `--text-warning` | `#8a6100` | `#875f00` |

The fills themselves are unchanged, so no badge background moves.

### The guard, third version

It now checks three things across **seven** theme states — the six
`[data-theme]` blocks *and* bare `:root`, which is the default palette and a
theme users actually see, and which the previous version used only as a
fallback scope and never iterated:

1. every token resolves to a parseable colour;
2. status text clears 4.5:1 as plain text on both surfaces;
3. status text clears 4.5:1 **composited over its own `-bg` fill**.

It also follows `var()` indirection, because `--status-error-fg` is declared as
`var(--color-negative)`. The previous version matched `#RRGGBB` only and treated
any non-hex override as no override at all, silently measuring `:root`'s value
instead. And it now lists the `--status-*-fg` tokens components actually
consume, not the source tokens they alias to.

Tamper-tested three ways, each injection verified by `grep` before the run and
reverted:
- restoring light's `#c0392b` fails at exactly the 4.49 / 4.00 the reviewer
  computed — the guard now catches what it previously certified;
- restoring `#3498db` fails naming `default` **and** `louvre`, proving the
  default palette is really iterated;
- aliasing a theme's `--status-info-fg` to `var(--bg-card)` fails at 1.00:1,
  proving indirection is resolved rather than skipped.

### Other findings

**Grade ramps reverted.** `CollectionHealthScorecard.vue` and
`NeedsAttentionQueue.vue` each render an A-F ramp. I had converted grades B and
F to status tokens because their values matched, leaving A, C and D as literals
— the exact value-identity conversion this log refuses two sections earlier,
and it half-bound a gradient to status semantics. Reverted. **The colour-literal
count therefore rises 61 to 69, and two budget entries go up.** That is the
right direction: the conversions were wrong, and a ratchet that has to be paid
for a correctness fix is worth paying.

**Undeclared delta, now declared.** The eight `#c9a84c` to `var(--accent-gold)`
conversions in `FollowerCoinDetailPage.vue` change the star colour in every
theme, not only the light one: `#9a7b2e` light, `#d8d4cc` british-museum,
`#977b3c` louvre, `#f0bc42` capitoline, `#ffbf00` byzantine, `#ffffff`
modern-greek. The default theme is unchanged at `#c9a84c`. This is almost
certainly the intended behaviour — a hardcoded gold ignoring the theme was the
bug — but it is a visible change in six theme states and belonged in the list.

**`-fg` and `-bg` bases have deliberately diverged.** `--color-negative` is now
`#fa6e5e` while `--status-error-bg` stays `rgba(231, 76, 60, …)`. Re-deriving
the fills from the new foregrounds would move every badge background in every
theme. The comment in `variables.css` now records the divergence as intentional
rather than describing a convention that no longer holds.

**Duplicate token removed.** `--color-overlay-full` was declared twice in the
same `@theme inline` block with the same value.

**Known gap, not closed.** The separate hex guard covering `<style scoped>`
blocks is still a single net total of 190, and a literal moved into `<script>`
is counted by neither guard. Both are recorded in the test file. Closing them
is a separate change, not a colour retune.

### Verification

| Gate | Result |
|---|---|
| `npm run type-check` | clean |
| `npm run lint` | clean at `--max-warnings 0` |
| `npm run test` | 207 files, 1759 passed, 1 skipped |
| `npm run build` | succeeded |

### The pattern worth recording

Three versions of this guard, three scoping errors, each found by someone other
than me: light-theme only, then bare-surface only, then default-theme excluded.
The colours were never the hard part. Each time I scoped the check to the case I
had already found rather than to the shape of the thing being checked.

## Third repair — the guard now derives its cases

The second repair was reviewed **BLOCK** again, correctly. The composite
arithmetic was right and the reviewer verified all six retunes by hand, but the
check still *enumerated* its cases: five tone pairs over two surfaces.

### Fourth instance of the same pattern

| # | Guard version | Scope error | Missed |
|---|---|---|---|
| 1 | light only | one theme | louvre, modern-greek |
| 2 | every theme, bare surface | wrong measurement | badges really at 4.00:1 |
| 3 | no bare `:root` | default palette | default info 4.04:1 |
| 4 | hand-listed pairs, 2 surfaces | enumerated, not derived | `--bg-secondary`, confidence pills |

`AdminValuationSchedule.vue:99` puts a nested table on `bg-surface-secondary`,
and that table renders both a `BaseStatusBadge` and three confidence pills that
pair `--confidence-*` with a *status* fill. Neither the surface nor those
pairings were in the guard's cross-product.

### The fix: derive, don't list

- **Surfaces** from the `--bg-*` tokens declared in `:root` — all five. No
  argument about which surfaces host a badge; the extreme ones are covered, so
  a future placement cannot land somewhere unmeasured.
- **Pairings** from the templates: each element's class list, each `:class`
  branch separately, plus dynamic token construction expanded across the tone
  union (`BaseStatusBadge` builds `var(--status-${props.tone}-fg)`).

A first cut of the extractor was too *loose* — its double-quoted alternative
swallowed whole `:class` ternaries and merged branches, manufacturing 84
failures from pairings that do not exist (`--text-muted` on a warning fill,
etc.). Bounding by element start-tag and per-branch literal brought it to the
**37 real failures**. A guard that is too loose is as useless as one too narrow;
it just fails differently.

### Twelve retunes

`:root` (covers every theme that does not override; solved against the worst
case, louvre on `--bg-input`): `--color-negative` `#fa6e5e`→`#fb8476`,
`--status-info-fg` `#38a4ed`→`#54b1f0`, `--status-neutral-fg` `#95a5a6`→`#a3b1b2`,
`--confidence-high` `#69b77f`→`#7abf8d`, `--confidence-low` `#e08d8d`→`#e19090`.

`light`: `--color-negative` `#ad3327`→`#a63125`, `--status-info-fg`
`#1e6698`→`#1d6292`, `--status-neutral-fg` `#576667`→`#546364`,
`--confidence-high` `#2f7a4d`→`#296b44`, `--confidence-medium` `#8a6100`→`#835c00`,
`--text-warning` `#875f00`→`#835c00`, `--color-positive` `#157347`→`#146e44`.

`--status-warning-fg` needed no change: it is `var(--text-warning)`, so fixing
light's `--text-warning` fixed it. That alias is also why `resolve()` must follow
indirection **in the theme's own scope** — an aliased token has a different
value per theme even when only `:root` declares it.

### Literals fell 69 → 65

`CollectionHealthTrendIndicator.vue` and `WishlistPage.vue` rendered status
badges with `text-red-400` / `text-green-400` and arbitrary `bg-[rgba(...)]`
fills. Binding them to status tokens removed four literals **and** brought those
badges under the guard — a colour outside the token system cannot be measured.
Both files came off the budget list entirely. Pairing a status fill with a
non-token colour is now an explicit failure.

### Tamper tests (four, each grep-verified then reverted)

1. Revert light `--color-positive` → fails on `--bg-secondary` (4.29) and
   `--bg-input` (4.49) **only** — precisely the surfaces the previous guard
   never checked — and names the three source files.
2. Restore `text-red-400` → fails the non-token pairing check by name.
3. Revert light `--confidence-high` → fails, reproducing the reviewer's exact
   **4.16:1** figure on `--bg-primary`, the row the previous failure table had
   misattributed to the success tone.
4. Append a duplicate `[data-theme="louvre"]` block → fails; a second block
   would otherwise silently replace the first and hide every token in it.

### Also closed

Comments are stripped before scoping (a token named in a comment was matchable).
`contrast()` composites a translucent foreground before measuring — no live
effect, all foregrounds are opaque hex, but it removes the assumption. The
`CollectionHealthScorecard` 12-vs-16 discrepancy between this log and the audit
is reconciled to **16**.

### Gates

Type-check clean; lint clean at `--max-warnings 0`; **1760 passed / 1 skipped**
across 208 files; build succeeded.

### Still open

The two budget gaps (the `<style scoped>` hex guard is still a single net total;
`<script>` literals are counted by neither) remain recorded, not closed. The
visual pass and the 65-literal palette choice remain owner decisions.
