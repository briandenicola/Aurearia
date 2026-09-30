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
distinct from the rest of the app. `CollectionHealthScorecard.vue` holds 12,
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
