---
applyTo: "src/web/**,Dockerfile"
---

# web guidance

Apply the [constitution](../../.specify/memory/constitution.md) and selected approved work. These scoped instructions implement, not replace, that authority.

### TypeScript / Vue
- `<script setup lang="ts">` with Composition API
- **Docker builds use stricter TS checking than local `vue-tsc`.** Always use optional chaining (`?.`) and nullish coalescing (`??`) on array index access. When passing nullable props (`string | null | undefined`) to a child component that expects non-nullable types (`string`), use `?? ''` (strings) or `?? 0` (numbers) at the call site. Local `vue-tsc --noEmit` may pass but Docker's `vue-tsc --build` will reject the mismatch.
- All API calls go through `src/web/src/api/client.ts` (Axios with JWT interceptor and 401 refresh queue)
- Agent chat streaming uses `fetch` + manual SSE parsing, not Axios
- `sanitizeCoin()` in the API client normalizes `''`/`undefined` → `null` before sending
- CSS variables: `--accent-gold`, `--bg-card`, `--border-subtle`, `--text-primary`
- Icons: `lucide-vue-next`

### UI / UX
- No emojis in UI text, prompts, or AI responses
- Dark theme is default
- PWA-compatible — test on mobile viewports

### Design System

All CSS values **must** use design tokens from `variables.css` and global classes from `main.css`. Never hardcode raw values when a token exists.

#### Design Tokens (variables.css)

| Token | Value | Use for |
|---|---|---|
| `--radius-sm` | `8px` | Cards, inputs, buttons |
| `--radius-md` | `12px` | Larger containers, modals |
| `--radius-lg` | `16px` | Hero sections |
| `--radius-full` | `9999px` | Pills, chips, badges |
| `--border-subtle` | gold 15% | Default borders |
| `--border-accent` | gold 40% | Hover/active borders |
| `--accent-gold` | `#c9a84c` | Primary accent, active states, links |
| `--accent-bronze` | `#b08d57` | Secondary accent |
| `--accent-gold-dim` | gold 30% | Active chip/pill backgrounds |
| `--accent-gold-glow` | gold 15% | Focus rings, subtle backgrounds |
| `--bg-card` | `#16213e` | Card backgrounds |
| `--bg-card-hover` | `#1a2747` | Card hover state |
| `--bg-input` | `#1e2a4a` | Input/textarea backgrounds |
| `--text-primary` | `#e8e0d0` | Body text |
| `--text-secondary` | `#a09880` | Secondary text, descriptions |
| `--text-muted` | `#706858` | Labels, hints, placeholders |
| `--text-heading` | `#d4b96a` | Headings (h1–h4) |
| `--cat-roman` | `#9b59b6` | Roman category |
| `--cat-greek` | `#6b8e23` | Greek category |
| `--cat-byzantine` | `#c0392b` | Byzantine category |
| `--cat-modern` | `#4682b4` | Modern category |
| `--mat-gold/silver/bronze` | metal colors | Material indicators |
| `--shadow-card` | box-shadow | Card elevation |
| `--shadow-glow` | gold glow | Hover/focus glow effect |
| `--transition-fast` | `0.2s ease` | Hover, focus |
| `--transition-med` | `0.3s ease` | Layout changes |

#### Typography Scale

| Element | Font | Size | Weight |
|---|---|---|---|
| h1 | Cinzel | `2rem` | 600 |
| h2 | Cinzel | `1.5rem` | 500 |
| h3 | Cinzel | `1.2rem` | 500 |
| h4 | Cinzel | `0.9rem` | 500 |
| Body | Inter | `0.9rem` | 400 |
| Secondary | Inter | `0.85rem` | 400 |
| Small | Inter | `0.8rem` | 400 |
| Tiny | Inter | `0.75rem` | 500 |

#### Uppercase Labels

All uppercase labels (section headers, info-card labels, sub-headings) use:
```css
font-size: 0.7rem;
font-weight: 600;
text-transform: uppercase;
letter-spacing: 0.08em;
color: var(--text-muted);
```
Use the global `.section-label` class or `.info-label` in detail grids.

#### Chip / Pill Hierarchy (global classes in main.css)

| Class | Use | Size | Padding |
|---|---|---|---|
| `.chip` | Interactive filter pills (face, category) | `0.8rem` | `0.35rem 0.85rem` |
| `.chip-sm` | Static tag/label pills | `0.75rem` | `0.15rem 0.5rem` |
| `.badge` | Category badges (Roman, Greek, etc.) | `0.75rem` | `0.2rem 0.7rem` |

All chips use `border-radius: var(--radius-full)`. Active state: `background: var(--accent-gold-dim); border-color: var(--accent-gold); color: var(--accent-gold)`.

#### Button Hierarchy (global classes in main.css)

| Class | Use | Padding | Font size |
|---|---|---|---|
| `.btn` | Standard button | `0.6rem 1.2rem` | `0.9rem` |
| `.btn-sm` | Compact button | `0.4rem 0.8rem` | `0.8rem` |
| `.btn-xs` | Inline/tiny actions | `0.25rem 0.6rem` | `0.75rem` |
| `.btn-primary` | Gold gradient CTA | — | — |
| `.btn-secondary` | Bordered neutral | — | — |
| `.btn-ghost` | Transparent, subtle border | — | — |
| `.btn-danger` | Red destructive | — | — |

#### Spacing Rhythm

- Section gaps: `1.5rem` between major sections (inscriptions, tags, info-grid, descriptions, notes)
- Sub-item gaps: `0.75rem` within sections
- Chip/tag gaps: `0.35rem`
- Card internal padding: `0.75rem` (info cards), `1rem` (feature cards), `1.5rem` (page cards)

#### UI Pattern Recipes

Before implementing UI, identify the closest existing page or component pattern and reuse it unless the user explicitly approves a new pattern. Do not fall back to generic layouts when a local pattern exists.

| Pattern | Use | Required shape |
|---|---|---|
| Page header | Standalone feature pages | `header.page-header` row with the title on the left and any back action as a compact icon/link on the right. Avoid long intro copy unless the page truly needs explanation. |
| Sidebar submenus | Related alternate views under one domain | Keep the parent item expandable/collapsible and start collapsed. Put related views under the parent as subitems instead of adding extra top-level menu items. |
| Summary metric row | A page has one primary count/value | Use one compact horizontal label/value row, e.g. `Mapped Coins:` left and value right. Do not create a large card or stacked label/value block for a single metric. |
| Pagination controls | Previous/next navigation | Keep controls in one row: `< Previous` then current label then `Next >`. Do not stack Previous and Next on mobile unless the viewport is too narrow to preserve usable tap targets. |
| Stats subviews | Health, value trends, timeline, map | Each subview is its own route/page under the Stats submenu; the Stats landing page stays summary-card focused. |
| Collection subviews | Gallery and Tray | Keep Gallery and Tray under the Collection submenu; the Collection parent starts collapsed like Stats. |
| Immersive PWA capture | Camera-first flows in an installed PWA (Add Coin, Identify Coin) | Use `PwaCaptureShell`: fixed full-bleed shell claiming `useImmersiveShellClaim()` so App.vue drops the nav bar and agent button. Close / Add-Identify segments / Quick Capture on top, a three-segment progress rail, one rounded stage with the guide ring, then Library + shutter + Manual (intake) or Deep (identify). Takes all color from the active theme's shared tokens (`--bg-primary`, `--bg-card`, `--bg-input`, `--accent-gold`, `--text-*`, `--border-subtle`) - no private palette and no literal colors. |

#### Rules for New UI Components

1. **Never hardcode** `border-radius`, colors, or spacing — always use tokens
2. **Never duplicate** chip/button CSS — use the global classes
3. **Never invent** a new font-size — pick from the typography scale
4. **All interactive pills** use `.chip` or extend it
5. **All static tags** use `.chip-sm` sizing (`0.75rem`, `0.15rem 0.5rem`)
6. **All uppercase labels** use `letter-spacing: 0.08em` — no other value
7. **Gold (`--accent-gold`)** is reserved for: active states, values/prices, links, section accents
8. **Cards** use `var(--radius-sm)` for small cards, `var(--radius-md)` for containers

## Completion

task check:web includes zero-warning lint, strict type checking, full package tests and production build; affected browser/mobile checks remain additional. See [testing](../../docs/testing.md#6-running-tests-locally-vs-ci).
Setup/installations require separate authorization; missing execution is incomplete.
