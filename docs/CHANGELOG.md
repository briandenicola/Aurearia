# Changelog

All notable changes to the Aurearia project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed

- **Auction price alert layout** — The price alert now leads with the lot's title
  and lot number, names the sale on its own line, and states the current high
  bid against the target it crossed (`475.00 USD (275.00 over target)`) instead
  of the stuttering `Current bid: current high bid 475.00 USD`. The Pushover
  push is rich HTML with the lot number linked straight to the lot on the
  auction site, matching the watch-bid digest. In-app notification cards now
  preserve the line breaks their messages are composed with, so every
  multi-line notification reads as lines rather than one run-on paragraph. See
  `specs/_backlog/F033-auction-price-alert-lot-block.md`.

### Added

- **Watch bid digest bid comparison and lot links** — Every lot in the Auction
  Watch Bid Digest now says how its bid moved since the previous digest — `up
  from 75.00`, `down from 95.00`, or `no change` — instead of restating the
  current bid with no context. A lot reported for the first time shows its bid
  alone. The digest is now a rich-HTML Pushover push: lots are grouped under
  their sale, each lot's title is bold, and its lot number links straight to
  that lot on the auction site. Long provider catalog titles are shortened to
  their identifying clause. The comparison baseline only moves when a digest is
  actually delivered, so a failed push or a lot trimmed for length never loses a
  change. See `specs/_backlog/F032-watch-bid-digest-change-comparison.md`.

- **New auction lot notifications** — The background watchlist sync now sends one
  batched notification when it starts tracking lots you watched or bid on at
  NumisBids or CNG. The Pushover push is rich HTML listing each lot's coin name,
  auction house and sale, lot number, and whether it is being watched or bid on,
  with a fully-qualified link to that lot in the app (`/auctions?lot=<id>`, a new
  deep link on the Auctions page). Requires **Public App URL** in Admin →
  Settings for the links; users without Pushover still get the in-app
  notification. See `specs/_backlog/F031-auction-sync-new-lot-notifications.md`.
- **Now bidding alerts** — The same sync sends a separate batched notification
  when a lot you were only watching moves to bidding (the provider now reports a
  bid of yours on it), carrying the current high bid and your max bid alongside
  the lot's app link. CNG only in practice, since NumisBids exposes no bid data.

## [4.0.0] — 2026-08-15

### Added

- **Deep Analysis** — Optional persisted background identification for new intake
  and saved coins with obverse/reverse image analysis, ephemeral hints, automatic
  provider routing, bounded fan-out, replayable SSE progress, cancel/retry, cited
  synthesis, and editable confirm-gated proposals.
- **OCRE provider** — Roman Imperial coin-type evidence through fixed-template
  Nomisma SPARQL with canonical OCRE links and visible ODbL 1.0 / American
  Numismatic Society attribution. Disabled by default.
- **Nomisma mint authority linking** — Admin-confirmed links from global mint
  locations to Nomisma concepts with CC BY 4.0 attribution.
- **Improved Numista lookup** — Editable generated queries, deterministic
  relevance scoring, bounded enrichment, caching, telemetry, and selected
  reference persistence.

- **Health endpoints** — `GET /health` and `GET /healthz` for container orchestration probes
- **API rate limiting** — General 120 req/min limit on all protected routes; 30 req/min on expensive write operations (image uploads, AI analysis, agent chat, imports)
- **ESLint config** — Flat config with `eslint-plugin-vue` and `typescript-eslint` for the Vue/TS frontend
- **golangci-lint config** — Go linter with errcheck, gocritic, misspell, bodyclose, staticcheck
- **Comprehensive API reference** — 19 missing endpoints documented; field name corrections applied
- **Project constitution** — 16 principles (v1.1.0) governing all code changes, referenced by Squad agent charters
- **Constitution enforcement** — Automated design token tests (border-radius, hex colors, font-size budgets)
- **Layer READMEs** — New README.md files for `src/api/`, `src/agent/`, replaced default Vite template in `src/web/`
- **CNG Auctions support** — Import CNG lots, sync CNG watched lots, and filter auction tracking by provider alongside NumisBids
- **Encrypted auction credentials** — Stored NumisBids and CNG provider passwords are encrypted at rest with lazy plaintext migration
- **Agentic set proposals** — Agentic set creation now submits a proposal request, runs asynchronously through the Python agent service, notifies the user when ready, and requires human review before any set is created
- **Coin Agent note saving** — Completed Coin Agent answers can be reviewed as markdown and saved into Notes from the chat surface
- **OCRE Deep Analysis provider (beta)** — Online Coins of the Roman Empire (OCRE, via the Nomisma.org SPARQL endpoint) is now a first-class, deterministic automated Deep Analysis provider for Roman Imperial coin-type identification. Off by default behind the admin-gated `DeepIdentificationOCREEnabled` flag (rollback = flag off; zero OCRE calls when disabled). Closes Feature 344 gate G-OCRE / T155. Go owns the fixed-template, injection-proof SPARQL boundary (only validated Nomisma id slugs are interpolated as bracketed URIs), bounded per-job call budget, cache, scoring, and the authenticated internal `ocre_search` tool; Python stays stateless. Typed partial outcomes (timeout/unavailable/no_match/invalid) never fail the overall job. Visible attribution on every surface where OCRE contributes: "Online Coins of the Roman Empire (OCRE), American Numismatic Society — ODbL 1.0." See [ADR 0010](adr/0010-ocre-odbl-provider.md).

### Changed

- **Coin identification choices** — Quick Identify remains the default fast path;
  Deep Analysis is an explicit opt-in for richer evidence and continues in the
  background across UI disconnects.
- **Provider boundaries** — NGC remains official-link-out only. Automated RPC
  integration is paused because no supported API or downloadable corpus is
  available; RPC images and data are not ingested.
- **Go toolchain** — All intentional pins now use Go 1.26.6.
- **CoinDetailPage decomposition** — Reduced from 1130 to ~360 lines; extracted CoinTagsSection, CoinInfoGrid, CoinActionsPanel, CoinAIAnalysis, CoinListingStatus sub-components
- **Desktop layout** — Sticky image sidebar with 2-column dashboard; sticky action bar at top: 61px
- **Mobile/PWA** — Removed sticky positioning leak; single-column layout preserved
- **formatCurrency** — Shared utility in `@/utils/format.ts` adopted across all components (replaced 6 local copies)
- **Documentation overhaul** — Updated ARCHITECTURE.md, SDD.md, features.md, social-feature.md, security-principles.md, threat-model.md, incident-response.md, references.md, authentication.md, deployment.md, getting-started.md, copilot-instructions.md
- **Coin Sets terminology** — Set types are now Standard, Goal, Smart, and Agentic; legacy Open/Defined terminology is retired in user-facing docs
- **Museum Tray display** — Tray cards now show purchase dates in `YYYY-MM-DD`; wishlist placeholders are more transparent and display `TBD`

### Fixed

- Store link rendering — Clickable link with "View Listing" fallback
- Docker TS build — Nullable props use `?? ''` coalescing for strict `vue-tsc --build`
- Actions + AI Analysis — Stacked full-width instead of squeezed side-by-side
- PWA mobile regression — Sticky CSS scoped to desktop-only `@media (min-width: 769px)`
- Sticky action bar gap — Aligned `top: 61px` with actual navbar height (60px content + 1px border)
- Removed stray `console.log` statements from `useCoinSearchChat.ts`

### Security

- **Deep Analysis isolation** — Go owns authenticated job-scoped provider tools,
  persistence, artifact cleanup, citation allowlists, and confirm-gated writes;
  the Python agent remains stateless and receives no database access.

---

## [1.0.0] — 2026-04-26

### Added

- **Core coin CRUD** — Create, read, update, delete coins with full metadata (ruler, denomination, material, grade, era, provenance, purchase/sale details)
- **Image management** — Multi-image upload (file + base64), obverse/reverse/detail types, primary image selection, proxy and scrape helpers
- **AI analysis** — Vision model coin analysis, text extraction, multi-provider support (Anthropic + Ollama)
- **Multi-agent service** — Python FastAPI/LangGraph service with 5 team pipelines (Coin Search, Coin Shows, Coin Analysis, Portfolio Review, Availability Check)
- **Auction tracking** — NumisBids and CNG Auctions integration, import/sync watchlists, lot-to-coin conversion, calendar event linking
- **Social features** — Follow/accept/block users, view follower coins, comments, ratings, public profiles
- **Showcases** — Curated public galleries with drag-and-drop ordering, shareable slugs
- **Collection tools** — Tags, bulk operations, journal entries, value history tracking, suggestions autocomplete
- **Statistics dashboard** — Collection stats, category/era distribution, portfolio value history charts
- **Calendar** — Auction events, price alerts, bid reminders
- **Notifications** — In-app notifications for wishlist status changes and new follower coins
- **User management** — JWT + API key auth, WebAuthn/passkey biometric login, user profiles, avatars
- **Admin panel** — User management, app settings, availability/valuation runs, connection testing, log viewer
- **Export/Import** — JSON collection export/import, PDF catalog generation
- **PWA** — Installable progressive web app with offline support, dark theme default
- **Architecture enforcement** — `architecture_test.go` enforces layered import rules via AST parsing
- **Scheduled jobs** — Automated valuation runs and availability checks with configurable intervals
