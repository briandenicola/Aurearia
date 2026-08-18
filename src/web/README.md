# Ancient Coins — Web Frontend

Vue 3 + TypeScript + Pinia + Vite PWA for managing a personal ancient coin collection. Dark-themed, mobile-friendly single-page application that communicates with the Go API backend.

## Prerequisites

- **Node.js** `^20.19.0` or `>=22.12.0`

## Install & Run

```bash
npm install          # install dependencies
npm run dev          # start dev server (http://localhost:5173, proxies /api to :8080)
npm run build        # type-check + production build
npm run preview      # preview production build locally
```

## Testing

```bash
npm run test         # run tests with Vitest
npm run test:watch   # run tests in watch mode
npm run type-check   # vue-tsc type checking
```

## Key Dependencies

| Dependency | Purpose |
|---|---|
| `vue` 3.5+ | UI framework (Composition API, `<script setup>`) |
| `vue-router` 5 | Client-side routing |
| `pinia` 3 | State management |
| `axios` | HTTP client (JWT interceptor, 401 refresh queue) |
| `vite` 7 | Build tool and dev server |
| `vite-plugin-pwa` | Service worker and PWA manifest generation |
| `lucide-vue-next` | Icon library |
| `markdown-it` + `dompurify` | Render and sanitize markdown (AI chat) |
| `sortablejs` | Drag-and-drop reordering |
| `@imgly/background-removal` | Client-side image background removal |
| `vitest` | Unit testing framework |
| `vue-tsc` | TypeScript checking for `.vue` files |

## Directory Structure

```
src/
  pages/             # Route-level page components
  components/        # Reusable UI components
  stores/            # Pinia stores (auth, coins, settings, etc.)
  api/               # Axios client with JWT interceptor (client.ts)
  composables/       # Vue composables (shared reactive logic)
  utils/             # Pure utility functions
  types/             # TypeScript type definitions
  assets/styles/     # CSS variables and global styles
  router/            # Vue Router configuration
  __tests__/         # Vitest test files
```

## Routes

| Path | Page | Auth |
|---|---|---|
| `/login` | LoginPage | No |
| `/register` | RegisterPage | No |
| `/` | CollectionPage | Yes |
| `/coin/:id` | CoinDetailPage | Yes |
| `/coin/:id/journal` | CoinDetailJournalPage | Yes |
| `/coin/:id/health` | CoinDetailHealthPage | Yes |
| `/coin/:id/notes` | CoinDetailNotesPage | Yes |
| `/coin/:id/actions` | CoinDetailActionsPage | Yes |
| `/coin/:id/analysis` | CoinDetailAnalysisPage | Yes |
| `/coin/:id/valuation` | CoinDetailValuationPage | Yes |
| `/add` | AddCoinPage | Yes |
| `/quick-capture` | QuickCapturePage | Yes |
| `/quick-capture/drafts` | QuickCaptureDraftsPage | Yes |
| `/quick-capture/drafts/:id` | QuickCaptureDraftPage | Yes |
| `/lookup` | CoinLookupPage | Yes |
| `/edit/:id` | EditCoinPage | Yes |
| `/wishlist` | WishlistPage | Yes |
| `/wishlist/search-alerts` | WishlistAlertsPage | Yes |
| `/sold` | SoldPage | Yes |
| `/auctions` | AuctionsPage | Yes |
| `/stats` | StatsPage | Yes |
| `/stats/timeline` | TimelinePage | Yes |
| `/stats/mint-map` | MintMapPage | Yes |
| `/stats/health` | StatsHealthPage | Yes |
| `/stats/value-trends` | StatsValueTrendsPage | Yes |
| `/stats/investment-breakdown` | StatsInvestmentBreakdownPage | Yes |
| `/stats/distribution` | CollectionDistributionPage | Yes |
| `/stats/emperors` | Redirects to `/sets/emperors` | Yes + Emperor Tracker |
| `/mint-map` | Redirects to `/stats/mint-map` | Yes |
| `/timeline` | Redirects to `/stats/timeline` | Yes |
| `/notes` | NotesPage | Yes |
| `/settings` | SettingsPage | Yes |
| `/settings/oidc/link/callback/:providerId` | OIDCLinkCallbackPage | Yes |
| `/auth/oidc/callback/:providerId` | OIDCLoginCallbackPage | No |
| `/admin` | AdminPage | Yes + Admin |
| `/followers` | FollowersPage | Yes |
| `/followers/:username/gallery` | FollowerGalleryPage | Yes |
| `/followers/:username/coins/:coinId` | FollowerCoinDetailPage | Yes |
| `/notifications` | NotificationsPage | Yes |
| `/showcases` | ShowcasesPage | Yes |
| `/showcases/:id/edit` | ShowcaseEditPage | Yes |
| `/s/:slug` | PublicShowcasePage | No |
| `/calendar` | CalendarPage | Yes |
| `/tray` | TrayViewPage | Yes |
| `/sets` | SetsPage | Yes |
| `/sets/emperors` | EmperorTrackerPage | Yes + Emperor Tracker |
| `/sets/:id` | SetDetailPage | Yes |
| `/process-image` | Redirects to `/settings?tab=process` | — |

A global navigation guard redirects unauthenticated users to `/login`, non-admin users away from `/admin`, and users without Emperor Tracker enabled away from `/sets/emperors`.

## PWA

The app uses `vite-plugin-pwa` with `registerType: 'prompt'`:

- **Manifest**: `standalone` display mode, dark theme (`#1a1a2e` / `#0f0f1a`)
- **Precaching**: All JS, CSS, HTML, images, and fonts via Workbox glob patterns
- **Runtime caching**:
  - `GET /api/showcase*` — NetworkFirst (5 min cache, 50 entries)
  - `PUT/POST/DELETE /api/*` — NetworkOnly
  - `/uploads/*` — not runtime cached; legacy `coin-images` caches are cleared on logout/user switch
- **Navigation fallback**: Denies `/api`, `/uploads`, and `/sw.js`
- **Update notification**: New service workers wait until the user accepts the in-app update banner; update checks run on registration, hourly, and when the app returns to the foreground

## Beta Screenshot Tour

`npm run screenshots:beta` is a separate, deliberately-real-network tool for capturing
production-like desktop and mobile screenshots of a real deployment (e.g. Brian's beta
site) to show prospective users. It is **not** part of the deterministic `npm run
test:browser` suite (F013) and is excluded from it via `testIgnore` in
`playwright.config.ts` — it uses its own config at `e2e/screenshots/playwright.config.ts`.

### Required environment variables (PowerShell)

```powershell
npx playwright install chromium
$env:PLAYWRIGHT_BASE_URL = "https://coins-beta.denicolafamily.com"
$env:AUREARIA_SCREENSHOT_USERNAME = "<existing beta account username>"
$env:AUREARIA_SCREENSHOT_PASSWORD = "<existing beta account password>"
npm run screenshots:beta
```

- `npx playwright install chromium` — first-run only. Both the `desktop` and `mobile`
  projects in `e2e/screenshots/playwright.config.ts` use the Chromium engine (the
  `mobile` project emulates an iPhone 13 viewport/UA/touch input via Chromium's mobile
  emulation rather than launching WebKit), so installing only the `chromium` browser
  is sufficient — no WebKit download is required.
- `PLAYWRIGHT_BASE_URL` — target site. Defaults to `http://127.0.0.1:4173` (local Vite
  preview) if unset; the local `test:browser` `webServer` is only skipped when this is
  set to an external URL.
- `AUREARIA_SCREENSHOT_USERNAME` / `AUREARIA_SCREENSHOT_PASSWORD` — an existing beta
  account's credentials. **Never hardcode, log, or commit these.** The tool fails fast
  with a clear error listing exactly which variable is missing if either is unset.

### What it does

1. Logs in via the real `/api/auth/login` contract to seed/refresh three
   `[Screenshot]`-prefixed fixture coins (an ancient collection coin, a modern/slabbed
   coin, and a wishlist target) using sanitized, plausible public numismatic data —
   no personal prices, notes, or dealer URLs. Fixtures are matched by exact name and
   updated in place on rerun, so **rerunning never creates duplicates** and never
   touches any other coin in the account.
2. Logs in through the real login UI (not a mocked session) and captures:
   collection gallery (filtered to `[Screenshot]` via the real search box), a coin
   detail page, the wishlist (scoped to just the fixture's card, since the wishlist
   page has no search UI — this guarantees no other wishlist items are ever captured),
   the Stats overview, and the Deep Analysis entry surface on a coin's Actions page
   (entry point only — no AI job is started).
3. Disables animations/transitions/caret, waits for fonts and network to settle, and
   hides the account-specific notification unread-count badge before every capture.

### Output

Screenshots are written to `src/web/artifacts/screenshots/` as
`<desktop|mobile>-NN-<name>.png` (e.g. `desktop-01-collection-gallery.png`). This
directory is **git-ignored** (`src/web/artifacts/`) so Brian can review and select
images locally before publishing any of them elsewhere (e.g. copying a chosen file
into `docs/` deliberately).

### Privacy warning

Only run this against beta data you're comfortable capturing. The tool is scoped to
avoid unrelated/personal records where the UI allows it (search-filtered gallery,
card-scoped wishlist), but it does **not** attempt to sanitize any other pre-existing
account data. Do not run it against an account containing real purchase prices,
private notes, or PII you don't want in a screenshot, and never commit captured
images directly — review them first.

## Design System

All styling uses design tokens defined in `assets/styles/variables.css` and global utility classes from `main.css`. Key tokens include `--accent-gold`, `--bg-card`, `--border-subtle`, `--text-primary`, and `--radius-sm` through `--radius-full`. See the root `.copilot-instructions.md` for the full token reference and component class hierarchy (`.chip`, `.btn`, `.badge`, etc.).
