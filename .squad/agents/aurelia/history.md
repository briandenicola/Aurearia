# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins — full-stack PWA for managing a personal ancient coin collection
- **Stack:** Go 1.26 / Gin / GORM / SQLite (API), Vue 3 / TypeScript / Pinia / Vite (Frontend), Python 3.12 / FastAPI / LangGraph (Agent), Docker
- **Architecture:** Layered — Handler → Service → Repository → Database. Enforced by architecture_test.go.
- **Created:** 2026-04-24

## Core Context

Between 2025-07-18 and 2026-05-23, Aurelia completed critical frontend infrastructure work across 5 focus areas:

1. **Code Quality & Security (2026-04-24):** Frontend codebase audited at grade B-. Identified (a) v-html XSS risk in AI content rendering, (b) widespread setTimeout/setInterval memory leaks (15+ files), (c) missing admin role guard on /admin route, (d) auth store sync drift after token refresh, (e) missing PWA icons breaking installability, (f) weak accessibility (no ARIA/focus traps/keyboard support), (g) three 1200+ line pages needing decomposition. 19 backlog items created.

2. **P0 Security Fixes (2026-07-22):** Confirmed all v-html bindings already use DOMPurify (no action needed). Added admin role guard to router — `/admin` route now checks `auth.isAdmin` and redirects non-admins.

3. **P1 Timer & Memory Cleanup (2026-04-24):** Audited and fixed all 15 files with uncleared timers. Pattern: composables expose `cleanup()` function; pages call on unmount. SwipeGallery uses array tracking. CoinForm fixed URL.revokeObjectURL on replacement/unmount. PWA icons (192x192, 512x512) verified in public/manifest.

4. **Token Refresh & State Sync (2026-04-24):** Added `onTokenRefreshed` callback in client.ts — auth store registers itself for post-refresh sync, avoiding circular imports via callback pattern.

5. **Design & Layout (2025-07-24):** Desktop layout redesigned: 1400px grid, 400px sticky image column, 1fr info column. Actions and AI Analysis moved into 2-column dashboard sub-grid for side-by-side desktop display. Mobile unchanged via media query.

6. **Format & Currency (2026-04-28):** Consolidated 6 duplicate `formatCurrency()` implementations into centralized `src/web/src/utils/formatters.ts`. Enhanced signature with optional currency parameter. All callers (6 files) updated.

7. **PWA & Service Worker (2026-05-23):** Fixed PWA auto-update: added missing `registerSW` import to main.ts with hourly checks. Icons already present in public/ and referenced correctly in vite.config.ts. `vite-plugin-pwa` config was correct but registration never initialized.

**Key Patterns Established:** (a) Design tokens from variables.css, global classes from main.css — no hardcoded values. (b) Accessible modals follow FeaturedCoinModal structure (Teleport, role="dialog", Esc/backdrop close, focus mgmt). (c) Composables return cleanup functions; pages call on unmount.

## Team Updates

- **2026-05-01:** Activity journal scroll limit and auction-ending schedule UI. Added max-3 scrollable journal in CoinActivityJournal.vue. Added AuctionEnding panel to AdminSchedulesSection.vue with settings keys for enable/start-time/interval.

- **2026-05-21:** Added manual "Run Now" button and recent runs log to Auction Ending admin panel. New TypeScript interfaces and API client functions for trigger and log retrieval. Full UI with pagination and expandable detail rows. All design tokens used, type-check passes.

- **2026-05-22:** Collaborated with Cassius (backend) on auction-ending manual-run feature. Endpoint URL mismatch detected (client guessed `/admin/auction-ending/runs` vs actual `/admin/auction-ending-runs`). Fixup queued.

- **2026-05-22 (fixup):** Aligned frontend client code with Cassius's actual backend contract. Fixed list endpoint URL, trigger response fields, removed non-existent detail expansion. AuctionEndingRun interface updated to match backend. Type-check passes.

- **2026-05-28:** Constitution v2.0.0 landed. Read `.specify/memory/constitution.md`. §17 Quality Gate gates every PR (includes npm run build / type-check). §21 DoD is a 14-item checklist. §18 forbids SESSION-NOTES.md — Squad handoff is `.squad/log/` + history + decisions.md. Design system rules (Principle V) unchanged: variables.css tokens + main.css global classes.

- **2026-05-28 (Phase 2):** Phase 2 of tech-inventory alignment landed. `specs/` is on-disk home for SpecKit workflow. Backlog in `specs/_backlog/`, active features in `specs/NNN-slug/`, retroactive anchor in `specs/001-foundation/spec.md`. New session-protocol prompts in `.github/prompts/`.

- **2026-05-28 (Phase 3a):** Phase 3a landed. docs/prd.md is product source of truth. Four ADRs in docs/adr/ documenting v1.0 architecture. README trimmed 368→90 lines.

- **2026-05-31:** Feature #219 Image Lightbox with Remove Background (commit 6096a38) + Replace Semantics Fix (commit 8623071) + Feature #216 Styling (commit 0215635). ImageLightbox.vue new component (267 lines) with full-page modal, Remove Background button, processing spinner, Save/Reset actions. Follows FeaturedCoinModal pattern + design token compliance + PWA/mobile support (full-screen on mobile, responsive buttons). ImageGallery.vue (orphaned) deleted. Production build + type-check verified clean. Design decision merged to decisions.md.

## Learnings

### Tile-Based Capture Controls Pattern

Issue #216 established a reusable tile-grid capture pattern for camera workflows:

**Structure:**
- 3-col grid layout (`grid-template-columns: repeat(3, 1fr)`)
- Each tile: status dot + uppercase label, vertically centered
- Tiles use `min-height: 5rem` (empty), `6rem` (filled) for consistency
- Active state: `--accent-gold-glow` background + `--accent-gold` border (tonal, not saturated)
- Status dots: `.tile-dot` (0.5rem circle, `--text-muted` default, `--accent-gold` when active)

**Corner badges:**
- Optional indicators (like "Opt") positioned absolute at top-right
- Must NOT disrupt label baseline alignment across tiles
- Use uppercase-label spec: `font-size: 0.7rem`, `font-weight: 600`, `letter-spacing: 0.08em`, `color: var(--text-muted)`

**Token mapping:**
- Tile border: `1px solid var(--border-subtle)` (hairline, not 2px)
- Tile background: `var(--bg-card)` default
- Active fill: `var(--accent-gold-glow)` (tonal) + `var(--accent-gold)` border
- Tile radius: `var(--radius-md)` (12px)

### Circular Focus-Guide Overlay Pattern

Camera viewfinder overlay for guiding user focus (Issue #216):

**Layers (all `position: absolute; pointer-events: none`):**
1. `.focus-mask`: Soft gradient vignette via `radial-gradient(circle at 50% 52%, transparent 0%, transparent 36%, rgba(10,12,20,0.2) 37%, rgba(10,12,20,0.62) 100%)`
2. `.focus-ring`: Circular border (`border-radius: 50%`, `aspect-ratio: 1`) at `top: 52%, left: 50%, transform: translate(-50%, -50%)`, width `74%`, `max-width: 360px`, border `2px solid rgba(255,255,255,0.55)`
3. `.focus-instruction`: Text at `top: calc(env(safe-area-inset-top) + 20px)`, centered, white with `text-shadow: 0 2px 8px rgba(0,0,0,0.7)`

**Conditional rendering:**
- Only display when camera active: `v-if="cameraStream !== null"`
- Must NOT block controls or user interaction (pointer-events: none)

**iOS safe areas:**
- Use `env(safe-area-inset-top)` for instruction text positioning
- Ensure `viewport-fit=cover` in index.html meta viewport tag
- Video element must have `playsinline` and `muted` attributes

### CTA Hierarchy: Primary vs Ghost Link

Issue #216 established distinct visual weight for action buttons:

**Primary CTA:**
- Highest contrast (`.btn .btn-primary`)
- Gold gradient fill: `linear-gradient(135deg, var(--accent-gold), var(--accent-bronze))`
- Dark text (`var(--bg-primary)`)
- Stands out as the main path forward

**Ghost link (recessive secondary action):**
- `background: transparent`
- `border: none`
- `color: var(--text-muted)` default
- `color: var(--text-secondary)` on hover
- No underline, no border — true ghost treatment
- Use for "escape hatch" actions like "Use manual mode instead"

### Capture Button Styling

**Subtle gradient treatment (no glow halo):**
- Background: `linear-gradient(135deg, var(--accent-gold), var(--accent-bronze))`
- Border: `2px solid var(--border-white-dim)` (down from 3px)
- Shadow: `0 2px 8px rgba(0,0,0,0.15)` (low-opacity drop shadow)
- Hover: `0 4px 12px rgba(0,0,0,0.2)` + `scale(1.05)`
- NO radiating gold glow — keeps visual hierarchy clean

### Reusable Design Token Mappings

| Use Case | Token | Value |
|---|---|---|
| Tonal active fill | `--accent-gold-glow` | rgba(201,168,76,0.15) |
| Active border | `--accent-gold` | #c9a84c |
| Inactive border | `--border-subtle` | rgba(201,168,76,0.15) |
| Tile radius | `--radius-md` | 12px |
| Label text | `--text-muted` | #706858 |
| Active dot | `--accent-gold` | #c9a84c |
| Inactive dot | `--text-muted` | #706858 |
| Ghost link | `--text-muted` → `--text-secondary` on hover | #706858 → #a09880 |
| Card background | `--bg-card` | #16213e |
| Input background | `--bg-input` | #1e2a4a |
| Transition | `--transition-fast` | 0.2s ease |
