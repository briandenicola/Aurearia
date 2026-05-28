# Ancient Coins — Product Requirements Document

**Status:** v1 draft (2026-05-28) | **Owner:** Brian (briandenicola)
**Constitution alignment:** Authored under §19 (Documentation Requirements). This document is item #2 in §0 Hierarchy of Authority — second only to the constitution.

---

## 1. Vision

Ancient Coins is a **personal-scale, self-hosted Progressive Web App** for managing a private ancient coin collection end-to-end. It's built for a single primary user (Brian) plus invited friends, enabling him to catalog coins with provenance, valuation, AI-assisted discovery, and social engagement — all stored on his own server, not in the cloud. The app emphasizes **depth over scale**: Roman/Greek/Byzantine/Modern taxonomy, dealer-enriched search results, vision-model analysis of obverse/reverse photos, and curator-approved statistics. It is decidedly not a marketplace, not a multi-tenant SaaS, and not a replacement for numismatic reference databases — it is a personal museum.

---

## 2. Personas

### Primary — Brian (Collector-Owner)
**Goal:** Maintain a comprehensive, beautifully organized record of his ancient coin collection with rich provenance, purchasing history, current valuations, and AI-assisted discovery of new coins.  
**Frustrations:** Spreadsheets lose context; commercial apps don't serve ancient numismatics; no app lets him analyze his own photos with vision models.  
**Success measures:** Add a coin (with images) in <60 seconds on mobile; search by ruler/inscription in <2 seconds; AI analysis returns novel insights; Coin of the Day surfaces forgotten coins.

### Secondary — Invited Friend (Read-Mostly Follower)
**Goal:** Browse Brian's public collection, leave comments, and discover collectors to follow.  
**Frustrations:** Can't see pricing or AI analysis; can't manage their own collection without an invite.  
**Success measures:** View galleries without friction; follow/unfollow friends; add star ratings.

### Tertiary — Public Visitor (Gallery Viewer)
**Goal:** See coins explicitly marked public (e.g., in a showcase link).  
**Frustrations:** No authentication, limited to published subsets only.  
**Success measures:** Load coin images and descriptions; click through to external references.

---

## 3. Goals (the product must)

1. **Catalog coins end-to-end** — CRUD operations on coins with images, attribution (ruler, mint, era), measurements (weight, diameter), category (Roman/Greek/Byzantine/Modern/Other), material, valuation, provenance notes, and free-text notes.
2. **Provide frictionless authentication** — JWT + refresh tokens, WebAuthn passkey login, and API keys for programmatic access. First registered user becomes admin.
3. **Surface coins via discovery** — List, search (by inscription, ruler, denomination), filter (by category, material, status), sort (by date added, value, random with deterministic seed), and paginate efficiently.
4. **Deliver five AI-assisted experiences** — Coin search (dealer discovery), coin shows (upcoming auctions), coin analysis (vision-model inspection of uploaded photos), portfolio review (collection valuation and gap analysis), and availability checking (monitor wishlist URL status).
5. **Surface one coin daily** — Coin of the Day scheduler picks an un-shown coin each morning, caches a summary, and dispatches in-app + Pushover notifications. Idempotent across restarts.
6. **Track wishlist & auctions** — Add wishlist coins with AI search; check availability; track NumisBids lots through bidding lifecycle; convert won lots to collection coins.
7. **Enable social engagement** — Follow other collectors, accept/block followers, leave comments and star ratings on their coins, upload avatars, control privacy (public/private profiles and per-coin privacy).
8. **Install as PWA** — Installable on iOS, Android, and desktop. Service-worker-cached offline read-only view of collection. Swipe carousel on mobile, grid on desktop.
9. **Admin-controlled AI provider selection** — Anthropic (Claude + web search) or Ollama (self-hosted models). Admins choose, configure keys, and customize analysis prompts.
10. **Provide rich statistics** — Total portfolio value, category/material/grade distributions, price trends, era/region heat maps, top coins by value, and valuation history snapshots.
11. **Support social showcases** — Create curated public-read-only coin subsets with unique shareable URLs (e.g., `/s/favorite-denarii`).
12. **Supply export/import** — JSON export of full collection; JSON import with per-coin validation; PDF catalog export with photos, grades, and provenance.

---

## 4. Non-Goals (the product will NOT)

- **Run a marketplace or process payments.** No buy-sell transactions, no PCI compliance, no currency conversion.
- **Be a multi-tenant SaaS.** Per-user collections are isolated; there is no cross-user billing, no multi-workspace support.
- **Index or compete with numismatic reference catalogs** (e.g., Numista, RIC, ACSearch). The app links to them but does not replicate their data.
- **Provide forensic-grade authentication or provenance verification.** The app is a personal record, not an expert appraisal tool.
- **Offer native iOS/Android apps.** The PWA covers mobile installation and offline read access; native apps are out of scope.
- **Support offline writes or conflict resolution.** Offline reads are cached; writes require network.
- **Bulk import from third-party catalogs.** Import is JSON-only; users must manually curate or use APIs.

---

## 5. Functional Areas

### 5.1 Collection Management (Core CRUD)
**Capability:** Create, read, update, and delete coins with structured fields (name, denomination, ruler, era, mint, material, weight, diameter, grade, inscriptions, images, notes, pricing, purchase date, store/dealer).  
**Cross-linked specs:** F001 (Coin of the Day), and shared model in `specs/001-foundation/spec.md` (Foundation v1.0).  
**Status:** Shipped (v1.0).  
**Out of scope:** Bulk edit without explicit per-coin save.

### 5.2 Discovery & Search
**Capability:** Browse collection via list or swipe gallery; filter by category, material, status; full-text search across name/inscription/ruler; sort by date added, value, or deterministic random; paginate efficiently; view sold-coin history.  
**Cross-linked specs:** `specs/001-foundation/spec.md` (User Story 2: Browse, search, discover).  
**Status:** Shipped (v1.0).  
**Out of scope:** Advanced faceted search or tag-based discovery (tags exist but not primary filter UI).

### 5.3 AI-Assisted Discovery (Five Team Pipelines)
**Capability:** Chat with AI agent to search dealers (Coin Search team), find upcoming auctions (Coin Shows team), analyze uploaded photos (Coin Analysis team), review portfolio gaps (Portfolio Review team), and check wishlist URL availability (Availability Check team). Streaming SSE responses with real-time status indicators.  
**Cross-linked specs:** F003 (Portfolio Review), F004 (Coin Analysis Vision); `specs/001-foundation/spec.md` (User Story 3: AI-assisted analysis).  
**Status:** Shipped (v1.0) with Anthropic + Ollama provider support; additional teams (Grading, Gap Analysis, Photo Guide, Price Trends, Similar Lots) added post-v1.0.  
**Out of scope:** Offline agent execution; multi-provider concurrent calls.

### 5.4 Wishlist & Auction Tracking
**Capability:** Mark coins as wishlist; AI-search for listings; track NumisBids lots through Watching → Bidding → Won/Lost workflow; verify availability of URLs on demand or via scheduled checks; auto-convert won lots to collection coins.  
**Cross-linked specs:** F002 (Wishlist & Availability).  
**Status:** Shipped (v1.0).  
**Out of scope:** Other auction houses besides NumisBids; live price scraping.

### 5.5 Social & Profiles
**Capability:** Send follow requests (pending → accepted/blocked); browse follower galleries (read-only, pricing/AI hidden); leave comments and 1–5 star ratings; manage privacy (public/private profile, per-coin private flag); search for collectors by username; upload avatars; add bio.  
**Cross-linked specs:** F005 (Social Following).  
**Status:** Shipped (v1.0).  
**Out of scope:** Public social network beyond gallery browsing; activity feeds; likes.

### 5.6 Authentication & Sessions
**Capability:** Register with username + password + email; JWT access (15 min) + refresh tokens (30 days, rolling); WebAuthn/FIDO2 passkey registration and login; API keys for programmatic access; first user auto-assigned admin.  
**Cross-linked specs:** F007 (WebAuthn Biometric Login); `docs/authentication.md`.  
**Status:** Shipped (v1.0).  
**Out of scope:** OAuth/SSO; two-factor SMS; hardware security keys (FIDO2 is platform authenticators only).

### 5.7 PWA & Offline
**Capability:** Install on home screen (iOS Safari → Add to Home Screen, Chrome → Install). Service-worker-cached static assets and previously-fetched coin lists/details. Swipe carousel on mobile; grid on desktop. Pull-to-refresh on gallery. Camera capture on image upload. Hamburger menu with filters/sort.  
**Cross-linked specs:** F006 (PWA Offline).  
**Status:** Shipped (v1.0) with offline reads; offline writes deferred.  
**Out of scope:** Offline write queuing; background sync; offline updates to coin data.

### 5.8 Admin & Settings
**Capability:** Admin page for users (list, delete, reset password), AI configuration (provider selection, API keys, model choice, custom prompts), system settings (log level, Numista API key), logs viewer, availability check scheduling, and valuation run scheduling.  
**Status:** Shipped (v1.0).  
**Out of scope:** Per-user admin roles; audit logging.

### 5.9 Notifications & Coin of the Day
**Capability:** In-app notification inbox; Pushover integration; daily scheduler picks one un-shown coin per user with dual idempotency (in-memory map + DB check); cached summary prevents extra AI calls; notification type `coin_of_day` with reference to `FeaturedCoin.ID`.  
**Cross-linked specs:** F001 (Coin of the Day).  
**Status:** Shipped (v1.0).  
**Out of scope:** Custom notification frequency; digest emails.

### 5.10 Statistics & Insights
**Capability:** Portfolio summary (total coins, total value, average value); category/material/grade distributions; value-over-time line chart from automatic snapshots; era/region heat map; top coins by value; Numista catalog lookup from coin detail page.  
**Status:** Shipped (v1.0).  
**Out of scope:** Machine-learning price prediction; tax reporting.

### 5.11 Showcases & Export
**Capability:** Create curated coin subsets (title, description, slug); publish/draft toggle; public read-only galleries at `/s/:slug` with no auth required; JSON collection export; JSON import with validation; PDF catalog export with photos.  
**Status:** Shipped (v1.0).  
**Out of scope:** CSV/Excel export; scheduled exports.

---

## 6. Constraints

- **Single primary user scale** — Designed for one primary collector (Brian) plus a small number of invited friends (< 10 concurrent). No multi-workspace isolation.
- **Self-hosted, single-node deployment** — SQLite database is acceptable at this scale. No clustering, no distributed transactions, no read replicas.
- **Docker containerization** — Two-container deployment (Go API + Vue SPA in one image; Python LangGraph agent in another). No Kubernetes, no load balancing.
- **Optional AI provider** — Anthropic or Ollama required for AI features; the app remains functional without them (AI actions disabled).
- **No PII outside owner's account** — Users can only manage their own data; no PII about other users is persisted beyond profiles and follow relationships.
- **No payment processing** — No stripe, no in-app purchases, no currency handling.
- **Constitution-bound** — All code changes must comply with the 16 Principles and 8 operational sections in `.specify/memory/constitution.md`. Deviations require an Amendment (§22).

---

## 7. Success Metrics

| Category | Metric | Target |
|----------|--------|--------|
| **Functional** | % of coins with full provenance fields populated | 90%+ |
| **Performance** | List view first paint (50-coin collection, local network) | < 1.5s |
| **Performance** | AI analysis stream start time | < 5s |
| **Reliability** | Architecture tests pass rate | 100% |
| **Reliability** | Data loss events in v1.0 | 0 |
| **Adoption** | Daily active sessions | ≥ 1 (Brian) |
| **PWA** | Lighthouse PWA checks pass | Yes |
| **UX** | New coin add time (mobile, with 2 images) | < 60s |

---

## 8. Open Product Questions

1. **Public link sharing:** Do we want read-only gallery links without authentication (already implemented in showcases) to expand to ad-hoc coin sharing?
2. **Portfolio valuation tracking:** Should we maintain a monthly price snapshot of the entire collection to track appreciation/depreciation over time? (Value-over-time is implemented; trend analysis could follow.)
3. **Multi-user collections:** Do we ever want multiple collectors to share one account's coin ownership (e.g., husband + wife co-curated collection)? If yes, this requires schema changes.
4. **Export formats beyond JSON/PDF:** Do we need CSV for spreadsheet import into Excel, or BIBTEX for academic reference?
5. **Coin age verification:** Should coins marked "sold" ever be moved back to "active" for re-acquisition, or are sold coins immutable history?
6. **Dealer/Source tracking:** Should we maintain a searchable database of dealers/auction houses to categorize purchase locations, or keep it as free-text fields?

---

## 9. Out of Scope Reference

**Technical constraints** are documented in the Constitution and are not restated here:

- **Principle I (Layered Architecture)** — Go API enforces Handler → Service → Repository → Database with strict import rules.
- **Principle II (Dependency Injection)** — All dependencies injected via constructors; only `main.go` imports the database package.
- **Principle XI (Security Hardening)** — Input validation, secret handling, output encoding.
- **Principle XII (Authentication & Token Policy)** — JWT issuance, refresh, revocation, storage.
- **Principle XIII (PWA / Mobile Rules)** — CSP, service worker scope, offline boundaries.

See `.specify/memory/constitution.md` §0–22 for the full governance model and quality gates.

---

## 10. Revision History

| Version | Date | Author | Notes |
|---------|------|--------|-------|
| 1.0 | 2026-05-28 | Scribe (drafted), Maximus (review TBD), Brian (approval pending) | Initial PRD extracted from README + features.md + auth/social docs. Cross-linked to specs/_backlog/F00N cards and Constitution §0 Hierarchy. Reflects v1.0 shipped surface plus post-v1.0 teams and capabilities. |
