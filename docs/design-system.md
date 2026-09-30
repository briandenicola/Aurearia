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

> Known gap: `text-xs` is **not** a step above. It is Tailwind's built-in
> utility, which resolves to `0.75rem` — the same size as `text-sm` — and it
> has 35 existing call sites. The dense-data step is deliberately named
> `text-micro` rather than redefining `--text-xs`, because that would have
> silently shrunk all 35. Folding `text-xs` into `text-sm` is a follow-up.

Never invent a font size. Uppercase labels (section headers, info-card labels,
table headers) always use `text-label` / weight 600 / `tracking-[0.08em]` /
`--text-muted` — use the `.section-label` or `.info-label` global class, or the
`text-label` utility with `tracking-[0.08em]`.

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

> Known gap: the status foregrounds are defined once in `:root`, matching the
> existing `--color-positive` / `--color-negative` / `--text-warning` pattern.
> They are tuned for the dark theme and their contrast on light-theme cards is
> below WCAG AA. Fixing that needs per-theme values and a visual check, which
> is out of scope for the #784 foundation slice.

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

Table headers use the uppercase label recipe from section 2:

```html
<th class="border-b border-border-subtle px-2 py-3 text-left text-label font-semibold uppercase tracking-[0.08em] text-text-muted">
```

Cells use `border-b border-border-subtle px-2 py-3 align-top`. Columns that are
optional on small screens carry `hidden md:table-cell` on both the `th` and the
matching `td`.

> Some existing tables still use drifted header styles (`text-sm`,
> `tracking-[0.05em]`). They are migrated page by page; new tables must use the
> recipe above.

## 7. Spacing rhythm

- Major section gaps: `1.5rem`
- Sub-item gaps within a section: `0.75rem`
- Chip / tag gaps: `0.35rem`
- Card padding: `0.75rem` (info cards), `1rem` (feature cards), `1.5rem` (page cards)

## 8. Rules for new UI

1. Never hardcode border radius, colour or spacing — use tokens.
2. Never duplicate chip, button, badge or toggle markup — use the primitive.
3. Never invent a font size — pick from the typography scale.
4. All uppercase labels use `letter-spacing: 0.08em`.
5. Before implementing UI, find the closest existing page pattern and reuse it.
6. No emoji in UI text.
