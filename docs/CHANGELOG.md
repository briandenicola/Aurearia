# Changelog

All notable changes to the Ancient Coins project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- **Health endpoints** — `GET /health` and `GET /healthz` for container orchestration probes
- **API rate limiting** — General 120 req/min limit on all protected routes; 30 req/min on expensive write operations (image uploads, AI analysis, agent chat, imports)
- **ESLint config** — Flat config with `eslint-plugin-vue` and `typescript-eslint` for the Vue/TS frontend
- **golangci-lint config** — Go linter with errcheck, gocritic, misspell, bodyclose, staticcheck
- **Comprehensive API reference** — 19 missing endpoints documented; field name corrections applied
- **Project constitution** — 16 principles (v1.1.0) governing all code changes, referenced by Squad agent charters
- **Constitution enforcement** — Automated design token tests (border-radius, hex colors, font-size budgets)
- **Layer READMEs** — New README.md files for `src/api/`, `src/agent/`, replaced default Vite template in `src/web/`

### Changed

- **CoinDetailPage decomposition** — Reduced from 1130 to ~360 lines; extracted CoinTagsSection, CoinInfoGrid, CoinActionsPanel, CoinAIAnalysis, CoinListingStatus sub-components
- **Desktop layout** — Sticky image sidebar with 2-column dashboard; sticky action bar at top: 61px
- **Mobile/PWA** — Removed sticky positioning leak; single-column layout preserved
- **formatCurrency** — Shared utility in `@/utils/format.ts` adopted across all components (replaced 6 local copies)
- **Documentation overhaul** — Updated ARCHITECTURE.md, SDD.md, features.md, social-feature.md, security-analysis.md, authentication.md, deployment.md, getting-started.md, copilot-instructions.md

### Fixed

- Store link rendering — Clickable link with "View Listing" fallback
- Docker TS build — Nullable props use `?? ''` coalescing for strict `vue-tsc --build`
- Actions + AI Analysis — Stacked full-width instead of squeezed side-by-side
- PWA mobile regression — Sticky CSS scoped to desktop-only `@media (min-width: 769px)`
- Sticky action bar gap — Aligned `top: 61px` with actual navbar height (60px content + 1px border)
- Removed stray `console.log` statements from `useCoinSearchChat.ts`

---

## [1.0.0] — 2026-04-26

### Added

- **Core coin CRUD** — Create, read, update, delete coins with full metadata (ruler, denomination, material, grade, era, provenance, purchase/sale details)
- **Image management** — Multi-image upload (file + base64), obverse/reverse/detail types, primary image selection, proxy and scrape helpers
- **AI analysis** — Vision model coin analysis, text extraction, multi-provider support (Anthropic + Ollama)
- **Multi-agent service** — Python FastAPI/LangGraph service with 5 team pipelines (Coin Search, Coin Shows, Coin Analysis, Portfolio Review, Availability Check)
- **Auction tracking** — NumisBids integration, import/sync watchlists, lot-to-coin conversion, calendar event linking
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
