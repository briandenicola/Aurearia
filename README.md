# Ancient Coins

> A self-hosted Progressive Web App for cataloging, analyzing, and sharing a personal ancient coin collection.

> **Note:** This application is 100% vibe coded. It exists for the author to learn and experiment with GitHub Copilot CLI.

[![CI](https://github.com/briandenicola/coin-collection-app/actions/workflows/ci.yml/badge.svg)](https://github.com/briandenicola/coin-collection-app/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)
![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vuedotjs)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178c6?logo=typescript)
![Python](https://img.shields.io/badge/Python-3.12-3776ab?logo=python)
![SQLite](https://img.shields.io/badge/SQLite-self--hosted-003b57?logo=sqlite)
![License](https://img.shields.io/badge/license-MIT-blue)

## What is this?

**Ancient Coins** is a self-hosted, full-featured Progressive Web App for managing a personal ancient coin collection end-to-end. Catalog, value, analyze, organize, and share your coins with powerful AI-assisted tools, offline access, and community features — all on your own server.

**For collectors who want:**
- 🏛️ Detailed numismatic records with provenance, images, and structured references
- 🤖 AI-powered analysis of coin photos (Claude or self-hosted Ollama)
- 🔍 Smart discovery tools for finding, tracking, and monitoring coins
- 📊 Rich portfolio analytics with value trends and gap analysis
- 🤝 Social features to follow collectors and share curated collections
- 📱 Install as a PWA; offline read access on mobile and desktop
- 🔒 Complete control of data; runs entirely on your infrastructure

Built for a single collector plus invited friends. Not a marketplace, SaaS, or reference database replacement. If you expose it outside a trusted home network, follow the public-facing hardening checklist in [`docs/deployment.md`](docs/deployment.md): TLS, reverse-proxy security headers, trusted proxy IP handling, invite/closed registration, private agent networking, backups, and monitoring are required.

**Product vision, personas, and detailed requirements** in [`docs/prd.md`](docs/prd.md). This README highlights features; see documentation links below for deep dives.

---

## ✨ Feature Highlights

### 🏛️ Collection Management
Organize coins with rich metadata: denomination, ruler, material, weight, inscriptions, grades, provenance, images, and free-text notes. Flexible filtering, full-text search, random shuffle with deterministic seed, and swipe/grid gallery views. **[Learn more →](docs/features/collection-management.md)**

### 🤖 AI-Powered Analysis
**Obverse & Reverse Analysis** — Upload coin photos for AI inspection with condition assessment, grade estimates, historical context, and market insights. Supports Anthropic Claude or self-hosted Ollama vision models.

**Multi-Team Agent System** — Specialized teams for search, shows, portfolio review, availability checks, grading, price trends, gap analysis, photography guidance, and similar-lot discovery handle different research tasks with streaming real-time status. **[Learn more →](docs/features/ai-search-agent.md)**

### 🎯 Discovery & Acquisition
**AI Coin Search** — Chat with an agent to find real dealer listings matching your description. Imports structured results with images, metadata, prices, and candidate catalog references.

**Coin Lookup** — Take or upload photos at a show to identify a coin or NGC Ancients slab. The app extracts NGC certification numbers, links to official NGC verification, enriches non-NGC lookups with Numista matches, and can save results to your wish list or collection. **[Learn more →](docs/features/coin-lookup.md)**

**Wish List** — Track coins you want with automatic availability checking, AI search, price tracking, and one-click purchase-to-collection conversion.

**Auction Tracking** — Monitor NumisBids lots through bidding lifecycle with status workflow, price alerts, bid reminders, auto-conversion to collection when won. **[Learn more →](docs/features/wish-list.md)**

### 📊 Portfolio Intelligence
**Collection Statistics** — Dashboard with portfolio value trends, category/material/grade distributions, top coins by value, era/region heat maps, ROI tracking, and health scorecards.

**Coin Sets** — Organize coins into open (flexible), defined (series with completion %), goal (milestones), or smart (rule-based automatic) sets with trend tracking, snapshots, and comparison tools. **[Learn more →](docs/features/coin-sets.md)**

### 🤝 Social & Community
**Follow Collectors** — Send follow requests, view follower galleries, leave comments, rate coins 1-5 stars.

**Profiles** — Customize avatar, bio, and privacy settings. Toggle public/private profile with follower management.

**Showcases** — Create curated public subsets with shareable URLs (e.g., `/s/favorite-denarii`). **[Learn more →](docs/features/social-features.md)**

### 📱 Mobile & Offline
**PWA Installation** — Install on iOS, Android, desktop (Chrome/Edge/Brave). Standalone app window, no browser UI.

**Offline Read Access** — Service worker caches collection data and images for offline browsing. Writes require an active network connection.

**Mobile Gestures** — Swipe gallery, pull-to-refresh, camera capture, touch-optimized controls. **[Learn more →](docs/features/pwa-features.md)**

### 🔐 Security & Control
**Multiple Auth Methods** — JWT + refresh tokens, WebAuthn passkeys (FIDO2), API keys for programmatic access.

**External Tool Server** — Expose read-only collection to external AI clients (OpenWebUI, LibreChat) via OpenAPI with scoped API keys and two-phase commit for writes.

**Fine-Grained Privacy** — Public/private profiles, per-coin privacy toggle, role-based admin access. **[Learn more →](docs/authentication.md)**

### More Features
**Structured Catalog References** — Formally link coins to RIC, RPC, SNG, Numista catalogs.
**Activity Journals** — Timestamped logs per coin (cleaned, graded, displayed).
**PDF Export** — Generate insurance catalogs with photos, provenance, valuations.
**OCR Text Extraction** — Extract text from store cards and certificates.
**Daily Featured Coins** — Automated scheduler to rediscover forgotten coins.
**Image Operations** — Background removal, circle clipping, automatic extraction.
**Bulk Operations** — Multi-select for batch tagging, status changes, exports.
**Configurable Coin Properties** — Admin-defined Era and Category option lists for tailoring collection metadata.

---

## 🎯 Feature Matrix

| Feature | Collection | Wish List | Auctions | Social | Analytics | Admin | Mobile |
|---------|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| Create/Edit Coins | ✅ | ✅ | ✅ | — | — | ✅ | ✅ |
| Coin Lookup | ✅ | ✅ | — | — | — | ✅ | ✅ |
| AI Analysis | ✅ | ✅ | ✅ | — | — | — | ✅ |
| Search & Filter | ✅ | ✅ | ✅ | — | — | — | ✅ |
| Tags & Sets | ✅ | ✅ | ❌ | — | ✅ | — | ✅ |
| Valuations | ✅ | ✅ | ✅ | — | ✅ | ✅ | ✅ |
| Offer/Sold | ✅ | ✅ | ✅ | — | ✅ | — | ✅ |
| Comments/Ratings | — | — | — | ✅ | — | — | ✅ |
| Follow/Share | — | — | — | ✅ | — | — | ✅ |
| Offline Access | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ |
| PWA Install | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## 🆕 What's New

**v2.0 (Latest)**
- **Coin Lookup** — Photo-based show workflow for NGC Ancients cert extraction, official NGC verification links, Numista fallback matches, and saving lookups to wish list or collection.
- **Configurable Coin Properties** — Admin-managed Era and Category options used by coin forms and lookup saves.
- **Coin Sets** — Organize coins into themed collections with trend tracking and completion analysis. Open, defined, goal, and smart (rule-based) set types. Snapshot history and value milestones.
- **Health Scorecard** — Track AI coverage, image coverage, and metadata completeness.
- Enhanced AI agent teams (grading, price trends, gap analysis, photography guide, similar lots).

**v1.0 (Current)**
- ✅ Collection CRUD with rich metadata
- ✅ AI-powered coin analysis (Claude/Ollama)
- ✅ AI coin search agent with dealer discovery
- ✅ Wish list with availability checking
- ✅ Auction tracking (NumisBids)
- ✅ Sold coin tracking with profit/loss
- ✅ Social features (follow, comment, rate)
- ✅ Collection statistics and portfolios
- ✅ PWA with offline read access
- ✅ External tool server for OpenWebUI integration
- ✅ Multi-auth (JWT, WebAuthn, API keys)
- ✅ Daily featured coins scheduler

## Architecture

Three services, two containers in production:

```
Browser ──► Vue 3 SPA  ──► Go API (Gin, GORM, SQLite) ──► Python LangGraph Agent
            (PWA)          :8080                          :8081 (stateless, SSE)
```

| Layer    | Tech                                  | Path         |
| -------- | ------------------------------------- | ------------ |
| Backend  | Go 1.26 (Gin), GORM, pure-Go SQLite   | `src/api/`   |
| Frontend | Vue 3, TypeScript, Vite, Pinia (PWA)  | `src/web/`   |
| Agent    | Python 3.12, FastAPI, LangGraph       | `src/agent/` |

The Go API serves both REST (`/api/*`) and the compiled Vue SPA; it proxies AI requests (including SSE streams) to the Python agent. The Python service is stateless — all configuration (provider, keys, prompts, user context) is passed per-request.

Full details in **[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)**; the binding rationale is captured in **[ADR 0002 — Three-Service Architecture](docs/adr/0002-three-service-architecture.md)**.

## Quick Start

**Prerequisites:** [Go 1.26+](https://go.dev/dl/), [Node.js 20+](https://nodejs.org/), [Task](https://taskfile.dev/) (optional), [Docker](https://docs.docker.com/get-docker/) (optional), [Ollama](https://ollama.ai/) (optional — for local AI).

```sh
git clone <repo-url> && cd coin-collection-app
task init        # generates .env with a random JWT secret
task up          # API (:8080) + web (:5173)
task up-all      # API + web + Python agent (:8081)
```

The first registered user becomes the admin and can configure AI providers from the Admin page. For a guided walkthrough see **[`docs/getting-started.md`](docs/getting-started.md)**; for production deployment see **[`docs/deployment.md`](docs/deployment.md)**.

### Running tests

```sh
task test            # Go architecture + unit tests
task test-agent      # Python agent tests
( cd src/web && npm run type-check )
```

Run `task --list` to see all targets.

## 📚 Documentation

### Quick Navigation

| Purpose | Where to Look |
|---------|---|
| **Features overview** | [Feature Index](docs/features/INDEX.md) — detailed docs for each feature |
| **Getting started** | [`docs/getting-started.md`](docs/getting-started.md) — step-by-step walkthrough |
| **Installation** | [`docs/deployment.md`](docs/deployment.md) — local dev & production setup |
| **Architecture** | [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system design & patterns |
| **API reference** | [`docs/api-reference.md`](docs/api-reference.md) — all endpoints, schemas |
| **Product roadmap** | [`docs/prd.md`](docs/prd.md) — goals, personas, requirements |

### All Documentation

| Area | Path |
|------|------|
| **Features (per-area detail)** | [`docs/features/INDEX.md`](docs/features/INDEX.md) — Browse by feature area |
| Feature: Collection Management | [`docs/features/collection-management.md`](docs/features/collection-management.md) |
| Feature: Coin Lookup | [`docs/features/coin-lookup.md`](docs/features/coin-lookup.md) |
| Feature: Coin Sets | [`docs/features/coin-sets.md`](docs/features/coin-sets.md) |
| Feature: Wish List | [`docs/features/wish-list.md`](docs/features/wish-list.md) |
| Feature: AI Analysis | [`docs/features/ai-analysis.md`](docs/features/ai-analysis.md) |
| Feature: AI Search Agent | [`docs/features/ai-search-agent.md`](docs/features/ai-search-agent.md) |
| Feature: Statistics | [`docs/features/statistics.md`](docs/features/statistics.md) |
| **Product & Vision** | [`docs/prd.md`](docs/prd.md) — vision, personas, goals |
| **Architecture** | [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system design |
| **API Reference** | [`docs/api-reference.md`](docs/api-reference.md) — all endpoints |
| **Authentication** | [`docs/authentication.md`](docs/authentication.md) — JWT, WebAuthn, API keys |
| **Deployment** | [`docs/deployment.md`](docs/deployment.md) — local dev & production |
| **Getting Started** | [`docs/getting-started.md`](docs/getting-started.md) — step-by-step guide |
| **PWA Guide** | [`docs/pwa-guide.md`](docs/pwa-guide.md) — installation & offline access |
| **Social Features** | [`docs/social-feature.md`](docs/social-feature.md) — follow, comment, rate |
| **External Tool Server** | [`docs/external-tool-server.md`](docs/external-tool-server.md) — OpenWebUI integration |
| **Security** | [`docs/security-principles.md`](docs/security-principles.md) — threat model, hardening |
| **Incident Response** | [`docs/incident-response.md`](docs/incident-response.md) — breach procedures |
| **References** | [`docs/references.md`](docs/references.md) — tools, standards, citations |
| **Software Design** | [`docs/SDD.md`](docs/SDD.md) — technical specifications |
| **ADRs** | [`docs/adr/`](docs/adr/) — architecture decision records |
| **Specs** | [`specs/`](specs/) — in-flight features |
| **Changelog** | [`docs/CHANGELOG.md`](docs/CHANGELOG.md) — version history |

**Design System** — Tokens, typography, components documented in [Copilot Instructions](.github/copilot-instructions.md#design-system) and locked by [ADR 0004](docs/adr/0004-design-token-system.md).

**Product Authority** — Per [Constitution §0](.specify/memory/constitution.md), the hierarchy is: Constitution → PRD → Active Spec → Plan → Tasks → Backlog → Decisions → Agent Judgment.

## Governance

- **Constitution:** [`.specify/memory/constitution.md`](.specify/memory/constitution.md) (v2.0.0) — non-negotiable contract for how this project is built. §0 defines the Hierarchy of Authority.
- **Spec workflow:** features ship through `specs/NNN-*/{spec,plan,tasks}.md` per the templates in [`.specify/templates/`](.specify/templates/).
- **Decision records:** [`docs/adr/`](docs/adr/) (Michael Nygard format, established by **[ADR 0001](docs/adr/0001-record-architecture-decisions.md)**).
- **Project decisions ledger:** [`.squad/decisions.md`](.squad/decisions.md); proposals land in [`.squad/decisions/inbox/`](.squad/decisions/inbox/).
- **AI team (Squad):** [`.squad/team.md`](.squad/team.md) and per-agent charters under [`.squad/agents/`](.squad/agents/).
- **Quality Gate & Definition of Done:** Constitution §17 and §21 — enforced on every PR.
- **Contributing:** [`CONTRIBUTING.md`](CONTRIBUTING.md).
- **Security:** [`SECURITY.md`](SECURITY.md).

## License

[MIT](LICENSE).
