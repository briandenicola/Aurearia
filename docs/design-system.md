# Design System

The single written standard for Aurearia's web UI. Every new or changed view in
`src/web` must follow it. The agent-facing summary lives in
[`.github/instructions/web.instructions.md`](../.github/instructions/web.instructions.md);
this document is the canonical version and adds the parts that summary omits
(status tones, shared primitives, tables).

Enforcement lives in `src/web/src/__tests__/design-tokens.test.ts`. Budgets in
that suite are ratchets: they may only decrease.

## 1. Tokens

All CSS values must come from `src/web/src/assets/styles/variables.css` and the
global classes in `main.css`. Never hardcode a raw value when a token exists.

| Token | Value | Use for |
|---|---|---|
| `--radius-sm` | `8px` | Cards, inputs, buttons |
| `--radius-md` | `12px` | Larger containers, modals |
| `--radius-lg` | `16px` | Hero sections |
| `--radius-full` | `9999px` | Pills, chips, badges |
| `--border-subtle` | gold 15% | Default borders |
| `--border-accent` | gold 40% | Hover / active borders |
| `--accent-gold` | `#c9a84c` | Primary accent, active states, links |
| `--accent-bronze` | `#b08d57` | Secondary accent |
| `--accent-gold-dim` | gold 30% | Active chip / pill / toggle backgrounds |
| `--accent-gold-glow` | gold 15% | Focus rings, subtle backgrounds |
| `--bg-card` | theme | Card backgrounds |
| `--bg-input` | theme | Input, textarea and toggle track backgrounds |
| `--text-primary` | theme | Body text |
| `--text-secondary` | theme | Secondary text, toggle knobs |
| `--text-muted` | theme | Labels, hints, placeholders |
| `--text-heading` | theme | Headings (h1–h4) |
| `--transition-fast` | `0.2s ease` | Hover, focus |
| `--transition-med` | `0.3s ease` | Layout changes |

Gold (`--accent-gold`) is reserved for active states, values and prices, links,
and section accents.

## 2. Typography

One scale. Every font size in the app is a step on it — `text-[0.82rem]` and
friends are a test failure. Steps are defined in the `@theme` block of
`src/web/src/assets/styles/main.css`.

| Utility | Size | Use for |
|---|---|---|
| `text-2xs` | `0.55rem` | Dense data grids at phone widths only |
| `text-micro` | `0.65rem` | Dense data grids, micro-chips |
| `text-label` | `0.7rem` | Section labels, table headers, uppercase tiny |
| `text-sm` | `0.75rem` | Badges, `.chip-sm` |
| `text-chip` | `0.8rem` | Standard chip text, field hints |
| `text-body` | `0.85rem` | Form labels, secondary body |
| `text-base` | `0.9rem` | Primary body, h4 |
| `text-md` | `1.1rem` | Card and modal titles |
| `text-lg` | `1.2rem` | h3 |
| `text-xl` | `1.5rem` | h2 |
| `text-2xl` | `2rem` | h1, stat numerals |
| `text-3xl` | `2.5rem` | Hero numerals |
| `text-4xl` | `3rem` | Single headline numeral |

Headings also carry a font: h1–h4 use Cinzel (`font-display`) at weights
600 / 500 / 500 / 500; body copy uses Inter (`font-sans`).

The only permitted arbitrary size is an `em` value inside rendered markdown
(`[&_code]:text-[0.88em]`), because that is a ratio to its container rather
than a scale step.

### What the guards do and do not catch

`design-tokens.test.ts` forbids `text-[Nrem|px|pt]` and `text-[var(--…)]`, but
it is a regex over the `<template>` block of `.vue` files. Treat it as a net
with known holes, not a proof:

- **Not scanned at all:** class strings built in `<script>` (this app does
  that, for example the `statusClass()` helpers in the admin schedule
  panels), any `.ts`/`.tsx` file, and any `<template>` tag that carries
  attributes such as `lang="pug"`.
- **Units not covered:** only `rem`, `px` and `pt` are matched. `text-[1ch]`,
  `text-[2vw]` and `text-[1cm]` slip through.
- **Expressions not covered:** `calc()`, `clamp()`, `text-[length:…]`, and
  anything built dynamically such as `` :class="`text-[${n}rem]`" ``.
- **Colour forms not covered:** `text-[color:var(--x)]` and a `var()` with a
  fallback.

`em` values are deliberately allowed, for rendered markdown only.

> Known gap: `text-xs` is **not** a step above. It is Tailwind's built-in
> utility, which resolves to `0.75rem` — the same size as `text-sm` — and it
> has 35 existing call sites. The dense-data step is deliberately named
> `text-micro` rather than redefining `--text-xs`, because that would have
> silently shrunk all 35. Folding `text-xs` into `text-sm` is a follow-up.

### Uppercase labels

Every uppercase element that carries a small size token is a label, and every
label reads:

```html
<span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">
```

Only the **colour** and the **alignment** vary by role — `text-gold` for an
accent label, `text-text-primary` for an emphasised one, `text-center` or
`text-right` to match a column. The size, the weight and the tracking do not
vary. The `.section-label` and `.info-label` global classes bake the same
recipe in, including `--text-muted`; use them where that colour is wanted and
the utilities where it is not.

Elements that are uppercase but carry **no** small size token are out of
scope — a grade badge at `text-base`, a data cell, or an `<input>` that
uppercases what the user types. A guard in `design-tokens.test.ts` enforces
the recipe on in-scope elements only.

> Note: `.badge` sets `text-transform: uppercase` in CSS, so counting
> `uppercase` classes in templates undercounts what renders uppercase.
> Badges are governed by `BaseStatusBadge` and section 3 instead.

### Colours in templates

Use the theme utility, never a raw `var()` class: `text-loss`, not
`text-[var(--color-negative)]`. The two render identically, but only the
utility is auditable and only it fails when the token is removed. Every colour
token in `main.css`'s `@theme inline` block has a matching utility; add one
there rather than reaching for `var()` at a call site.

## 3. Chips, badges and buttons

| Class | Use | Size | Padding |
|---|---|---|---|
| `.chip` | Interactive filter pills | `0.8rem` | `0.35rem 0.85rem` |
| `.chip-sm` | Static tag / label pills | `0.75rem` | `0.15rem 0.5rem` |
| `.badge` | Category badges (Roman, Greek, …) | `0.75rem` | `0.2rem 0.7rem` |

| Class | Use | Padding | Font size |
|---|---|---|---|
| `.btn` | Standard button | `0.6rem 1.2rem` | `0.9rem` |
| `.btn-sm` | Compact button | `0.4rem 0.8rem` | `0.8rem` |
| `.btn-xs` | Inline / tiny actions | `0.25rem 0.6rem` | `0.75rem` |
| `.btn-primary` | Gold gradient CTA | — | — |
| `.btn-secondary` | Bordered neutral | — | — |
| `.btn-ghost` | Transparent, subtle border | — | — |
| `.btn-danger` | Red destructive | — | — |

Never duplicate chip or button CSS in a component; use the global classes.

## 4. Status tones

Run status, availability and outcome pills use the status tokens, never raw
`rgba()` literals.

| Tone | Background token | Foreground token | Use for |
|---|---|---|---|
| `success` | `--status-success-bg` | `--status-success-fg` | success, completed, available |
| `error` | `--status-error-bg` | `--status-error-fg` | error, failed, unavailable |
| `warning` | `--status-warning-bg` | `--status-warning-fg` | pending, queued, partial, unknown |
| `info` | `--status-info-bg` | `--status-info-fg` | running, in progress |
| `neutral` | `--status-neutral-bg` | `--status-neutral-fg` | skipped, inactive |

Each tone also has a `--status-*-border` (the foreground at 0.3 alpha) for
outlined pills, and `success` and `error` have a `--status-*-tint` at 0.1 for
subtle row highlights.

The `:root` values are tuned for the dark themes. The light theme overrides all
eight status foregrounds, because the dark-theme values fail WCAG AA on a white
card — `--color-positive` measured 2.10:1 and `--text-warning` 1.63:1. Every
foreground clears 4.5:1 on **all five** surface tokens, not just the card and
the page.

A guard in `design-tokens.test.ts` computes the WCAG ratio from the token values
rather than asserting a number by hand, so it keeps holding if a value changes.
It covers **seven theme states** — the six `[data-theme]` blocks and bare
`:root`, which is the default palette — and resolves each token through `:root`
and through `var()` indirection, in the theme's own scope, because an alias like
`--status-warning-fg: var(--text-warning)` resolves differently per theme.

Critically, it measures badge text **composited over its own `--status-*-bg`
fill**, not against the bare surface. A badge renders its text on a ~0.15 alpha
fill, which shifts the surface toward the foreground's hue and always reduces
contrast. Measuring the bare surface certified pairs that were really 4.00:1.

The guard **derives its cases instead of listing them**, which is the part that
matters when you add UI:

- **Surfaces** come from the `--bg-*` tokens declared in `:root`, so all five
  are covered. `--bg-secondary` and `--bg-input` are the extreme surfaces in
  most themes; an earlier version checked only `--bg-card` and `--bg-primary`
  and missed a nested table on `--bg-secondary` where every tone failed.
- **Pairings** come from the templates. The guard reads each element's class
  list — and each branch of a `:class` binding separately — and pairs a
  `bg-status-*-bg` fill with whatever `text-*` colour sits on the same element.
  It also expands a component that builds its tokens dynamically, such as
  `BaseStatusBadge`, across its declared tone union.

So a status fill paired with a **non-token colour** (`text-red-400`, or an
arbitrary `bg-[rgba(...)]`) is a **failure**, not an omission: a colour outside
the token system cannot be themed and cannot be measured. Use the status tokens.

If you add a theme, a surface token, or a badge on a new surface, it is checked
automatically. If you retune a `--status-*-bg` fill, the composited ratios move
— that is intended.

### Overlays

Scrims and translucent panel fills use `--overlay-20` through `--overlay-60`,
exposed as the `bg-overlay-*` utilities. The names are numeric on purpose: the
call sites do not share a semantic hierarchy, and naming them `scrim` or `veil`
would invent one.

## 5. Shared primitives

Primitives live in `src/web/src/components/ui` and are re-exported from
`components/ui/index.ts`. Import them from `@/components/ui`.

| Component | Use | Notes |
|---|---|---|
| `BaseButton` | Buttons that need loading or icon state | Wraps the `.btn` hierarchy |
| `BaseChip` | Interactive filter pills | Wraps `.chip` |
| `BaseBadge` | Category badges | Colour comes from `utils/categoryColor` |
| `BaseStatusBadge` | Status / outcome pills | `tone` prop from section 4 |
| `BaseToggle` | Every on/off switch | `v-model`, `size`, `disabled`, `label` |
| `BaseSpinner` | Inline loading indicator | — |
| `BaseEmptyState` | Empty list / no-results state | — |

### Toggle specification

`BaseToggle` is the only switch in the app. Hand-rolling a `peer sr-only`
checkbox is a test failure.

| Size | Track | Knob | Use for |
|---|---|---|---|
| `md` (default) | 28 × 50 px | 22 px | Primary settings and admin rows |
| `sm` | 22 × 42 px | 16 px | Indented sub-options and compact table cells |

The track uses `--bg-input` with a `--border-subtle` border; the knob uses
`--text-secondary`. Checked state switches to `--accent-gold-dim` /
`--accent-gold`. Every toggle carries a focus ring, and `label` is a **required**
prop so no switch can ship without an accessible name.

`class` and `style` passed to `BaseToggle` land on the wrapper, where layout
utilities work; every other attribute (for example `id`) is forwarded to the
hidden `<input>`.

## 6. Tables

Every data table carries the `data-table` class. It supplies the header recipe
(`text-label`, weight 600, `uppercase`, `0.08em` tracking, `--text-muted`,
`line-height: 1.4`, left-aligned, bottom border) and the cell borders and
alignment, from one place in `main.css`.

```html
<table class="data-table [&_th]:px-2 [&_th]:py-3 [&_td]:px-2 [&_td]:py-3">
```

Padding stays on the element, because density is a per-table decision. The
class is declared inside `@layer components`, so a Tailwind utility on an
individual cell still wins:

```html
<th class="text-right">Value</th>   <!-- overrides the left alignment -->
<td class="align-top">…</td>
```

Where the table is rendered from markdown and cannot carry a class — the help
content in `HelpSection.vue` — put `data-table` on the container that wraps it.
`.data-table table` matches descendants for exactly this case.

Columns that are optional on small screens carry `hidden md:table-cell` on both
the `th` and the matching `td`.

A guard in `design-tokens.test.ts` fails the build if a template re-inlines the
header recipe, in either spelling: on the table as `[&_th]:uppercase`,
`[&_th]:tracking-[…]`, `[&_th]:font-semibold` or `[&_th]:text-text-muted`, or
repeated on the individual `th` elements. It also fails a `<table>` in a
template that never mentions `data-table`. That containment check is file-scoped
rather than per-table, so a file mixing a migrated and an unmigrated table would
pass; the guard is a ratchet, not a proof.

## 7. Spacing rhythm

- Major section gaps: `1.5rem`
- Sub-item gaps within a section: `0.75rem`
- Chip / tag gaps: `0.35rem`
- Card padding: `0.75rem` (info cards), `1rem` (feature cards), `1.5rem` (page cards)

## 8. Rules for new UI

1. Never hardcode border radius, colour or spacing — use tokens.
2. Never duplicate chip, button, badge or toggle markup — use the primitive.
3. Never invent a font size — pick from the typography scale.
4. All uppercase labels use `text-label font-semibold tracking-[0.08em]`; only
   the colour and the alignment vary by role.
5. Every data table uses the `data-table` class.
6. Before implementing UI, find the closest existing page pattern and reuse it.
7. No emoji in UI text.
