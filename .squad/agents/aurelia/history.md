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

### API Key Scope Management (Issue #218, T022/T023)

**Location:** API key management UI is in `SettingsDataSection.vue` (Data Management settings section).

**Scope control pattern:**
- Chip-based toggle selector using global `.chip` class
- Two options: "Read" (default) and "Read/Write"
- Positioned between name input and generate button
- State: `apiKeyScope = ref<'read' | 'read,write'>('read')`
- Resets to "read" after successful key generation

**Create payload contract:**
- `generateApiKey(name: string, scope?: 'read' | 'read,write')`
- Optional `scope` field passed to `POST /auth/api-keys`
- Backend defaults to "read" when omitted

**Capability display:**
- Small `.chip-sm` badge next to key name in list
- Two variants:  
  - Read: Blue accent (`rgba(59, 130, 246, 0.1)` bg, `#3b82f6` text)
  - Read/Write: Gold accent (`--accent-gold-glow` bg, `--accent-gold` text/border)
- Helper functions: `capabilityLabel()` → "Read" | "Read/Write", `capabilityClass()` → CSS class
- Badge uses design tokens and `.chip-sm` sizing (0.75rem font, 0.15rem 0.5rem padding)

### In-App External Tool Server Documentation (Issue #218, 2026-06-01)

**Location:** `src/web/src/components/HelpSection.vue` — new accordion titled "Connecting AI Tools (External Tool Server)".

**Structure:** Three-perspective documentation (Admin, User, Developer) in a single accordion:
- **For Admins:** How to enable the server via Admin Settings (`ExternalToolServerEnabled`), default-off security posture, what to tell users about scoped API keys and journaled writes
- **For Users:** Step-by-step guide to create scoped API keys (read vs read/write), import the OpenAPI URL into external clients (OpenWebUI, LibreChat, n8n), and understand the two-phase write confirmation flow
- **For Developers:** Base path `/api/v1/tools/*`, `X-API-Key` auth, six available tools (four read, two write), OpenAPI spec endpoint, mcpo wrapper for MCP compatibility, security model (tenant isolation, rate limiting, field allowlist)

**Content source:** `docs/external-tool-server.md` — authoritative technical reference.

**Styling:** Uses existing `.help-accordion`, `.help-content`, `.help-table`, `.help-code` classes. No emojis, no hardcoded values. Includes table of six tools with capability requirements.

**Placement:** Inserted immediately before "Helpful Resources" accordion — positioned as an app-setup topic rather than coin-collecting content.

**Validation:** `npm run build` (type-check passed), `npm run lint` (HelpSection.vue warnings fixed, exit 0).

### Multi-Container Deployment Note (2026-06-01)

Related to Issue #217 (Collection Chat) and #218 (External Tool Server), Cassius documented multi-container deployment requirements:

- **AGENT_INTERNAL_CALLBACK_URL** environment variable required for collection chat to work across containers (defaults to `localhost:8080`, unreachable in Docker networks)
- Set to API service name: `http://coins:8080` or `http://app:8080` depending on your docker-compose setup
- Documented in `docs/deployment.md` with startup warning in release mode
- Your external tool server work (#218) depends on this multi-container layer functioning correctly

### Coin Detail Section Organization (2026-06-01)

**Section ordering pattern on CoinDetailPage.vue:**
1. Title hierarchy (name, ruler, category/status badges)
2. Inscriptions (obverse/reverse)
3. Purchase metadata (date, store, reference URL) — italicized
4. Details (metadata table — material, denomination, mint, weight, etc.)
5. **Tags** (user tags with add/remove)
6. Catalog References (RIC, etc.)
7. Description (obverse/reverse descriptions)
8. Listing Status (availability check banner)
9. Detail section links (settings-style navigation)

**Tags as full section (not a sub-heading):**
- Changed from `<h4 class="section-label">Tags</h4>` to `<h3>Tags</h3>` to match sibling sections (Details, Description, Inscriptions)
- Section wrapper renamed from `.detail-tags-section` to `.tags-section` with `margin-bottom: 1.5rem` (full section spacing)
- Heading gets `margin-bottom: 0.75rem`, `font-size: 1rem` (consistent with other `<h3>` in page)
- This pattern applies to any future detail-page sections: use `<h3>` inside a `.xxx-section` wrapper with 1.5rem bottom margin

**Tag pill sizing — .chip vs .chip-sm:**
- `.chip` size (interactive filter pills): `font-size: 0.8rem; padding: 0.35rem 0.85rem`
- `.chip-sm` size (static tag/label): `font-size: 0.75rem; padding: 0.15rem 0.5rem`
- Tags on detail page are now `.chip`-sized (larger, more tappable) for better mobile interaction and visual consistency with other interactive elements
- Button controls (`.btn-tag-add`, `.tag-picker`) bumped to match: `0.8rem` font, `0.35rem 0.85rem` padding for `.btn-tag-add`, `gap: 0.5rem` in `.detail-tags` row, `margin-top: 0.75rem` for `.tag-picker`
- Use `.chip-sm` only for static badges (like "Wishlist", "Sold" chips in title area)

## Team Updates

- **2026-06-01:** Feature #219 Tags UI refinements completed. Promoted Tags to full section (h3 heading, 1.5rem spacing, .chip-sized pills), repositioned after Details and before Catalog Reference. Purchase metadata line italicized for visual distinction. Type-check + build pass clean. Ready for merge. Decision merged to decisions.md.

- **2026-06-01:** AddCoinPage camera controls layout updated. Changed `.camera-actions` to 3-column grid matching `.capture-slots` for precise column alignment. Shutter button (Camera icon) centered under REVERSE tile (column 2). Photo selection button (Images icon, replaced Upload) aligned right under CARD tile (column 3). Icon swap provides better semantic match for photo library access. Type-check + lint pass clean. Decision in inbox.

## Learnings

- **FeaturedCoinModal**: New modal component integrated into App.vue with EventBus listening for `openFeaturedCoinModal` events from NotificationPopup clicks. Modal fetches coin + featured record via `/featured-coins/:id` endpoint, displays cached summary + coin images. Supports "close-this-modal-only" pattern via direct `showFeaturedCoinModal.value = false` and "close-all-modals" via EventBus `closeAllModals` + Escape key. Daily featured coin notification type is `coin_of_day`, referenceId is FeaturedCoin.ID (not coin id). Modal uses design tokens, no emojis, lucide icons, dark theme. Positioned after NotificationPopup in App.vue `z-index` stack (below dialogs/prompts, above popups).

- **AddCoinPage Camera Controls Layout**: Camera action buttons now use a 3-column grid (`display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.5rem`) matching `.capture-slots` tile grid for precise vertical alignment. Shutter button (`.shutter-btn`) positioned in column 2 (`grid-column: 2; justify-self: center`) under the REVERSE tile. Photo selection button (`.upload-icon-btn`) positioned in column 3 (`grid-column: 3; justify-self: end`) aligned to the right edge of the CARD tile. Column 1 remains empty. Icon changed from `Upload` to `Images` (lucide `Images` icon for photo library access). This layout ensures camera controls visually align with their corresponding capture targets above.
