# Feature Specification: v1.0 Foundation — Ancient Coins PWA

**Feature Branch**: `001-foundation`
**Created**: 2026-05-28 (retroactive)
**Status**: SHIPPED (retroactive)
**Backlog refs**: F001, F002, F003, F004, F005, F006, F007 (cross-linked — many were extracted from the v1.0 surface during backlog seeding)
**Constitution alignment**: Principle I (Layered Architecture), Principle II (Dependency Injection / Three-Service Architecture), Principle V (Design Token System), Principle X (Architecture Enforcement), Principle XII (Authentication & Token Policy — JWT + WebAuthn), Principle XIII (Security Baseline / PWA boundaries)
**Input**: User description: "Document the v1.0 feature surface that already shipped, retroactively, to anchor Constitution §0 Hierarchy item 3 (active feature spec) with real content."

---

## Problem

Brian wanted a personal Progressive Web App to manage his ancient coin collection end-to-end: provenance, valuation, search/discovery, AI-assisted analysis, and notifications, deployable as a small self-hosted Docker stack. No existing off-the-shelf product covered ancient numismatics with the desired depth (Roman/Greek/Byzantine taxonomy, dealer-page enrichment, vision analysis of obverse/reverse photos) while keeping the data on his own server.

v1.0 of Ancient Coins is the answer: a three-service application (Go API, Vue 3 PWA, Python LangGraph agent) backed by a single SQLite file, packaged as two Docker containers, that ships the complete catalog-management and AI-analysis surface for a single primary user plus optional invited friends.

## Users

- **Primary user** — Brian. Full CRUD over his collection, admin settings, AI features.
- **Invited friends** — admin-provisioned accounts (no self-signup). Can manage their own collection scope.
- **Public visitors** — read-only access to coins explicitly marked public (gallery view only; no auth).

## User Scenarios & Testing

### User Story 1 — Catalog a coin end-to-end (Priority: P1) 🎯 MVP

Brian acquires a new coin and wants it cataloged with images, attribution, provenance, and valuation in under five minutes from his phone.

**Why this priority**: Without coin CRUD, nothing else in the app has anything to operate on. This is the irreducible MVP.

**Independent Test**: Log in on a mobile viewport, tap "Add Coin", attach two images (obverse + reverse), fill the structured fields (denomination, ruler, mint, date, weight, diameter, category, material), save, and verify the coin appears in the list view and detail view with images rendered.

**Acceptance Scenarios**:
1. **Given** an authenticated user, **When** they POST a coin with images via the Add form, **Then** the coin persists with all fields and is queryable in the list within the same session.
2. **Given** a saved coin, **When** the user edits any field, **Then** changes survive a page refresh.
3. **Given** a sold coin, **When** the user marks it sold with a sale price and date, **Then** it is excluded from active-collection counts but remains in history.

### User Story 2 — Browse, search, and discover (Priority: P1)

Brian wants to browse his collection by category, search by inscription or ruler, sort by value/date/random, and get a fresh-feeling gallery on every visit.

**Why this priority**: A collection that cannot be browsed loses its value as the count grows past a few dozen.

**Independent Test**: With 20+ coins seeded, filter by `category=Roman`, sort `sort=random&seed=42`, paginate, and verify deterministic order within the session via `sessionStorage` seed persistence.

**Acceptance Scenarios**:
1. **Given** ≥20 coins, **When** the user opens the list with `?sort=random&seed=N`, **Then** the order is stable for the session and changes on a new session.
2. **Given** a search term matching an inscription substring, **When** the user submits search, **Then** matching coins appear in the result set.
3. **Given** category and face filters, **When** combined, **Then** results respect AND semantics.

### User Story 3 — AI-assisted analysis (Priority: P2)

Brian wants to ask the app to analyze a coin photo, search dealer sites for comparable listings, check upcoming coin shows, review his portfolio value, and verify availability of a listed URL — without leaving the app.

**Why this priority**: AI is the differentiator vs. a plain CRUD app, but the catalog must exist first.

**Independent Test**: With Anthropic or Ollama configured in Admin Settings, run each of the five team pipelines (search, shows, analysis, portfolio, availability) from the UI and confirm the SSE stream renders progressive output and a final structured result.

**Acceptance Scenarios**:
1. **Given** AI provider configured and reachable, **When** the user triggers Coin Analysis on a coin with images, **Then** the vision model returns a structured Pydantic-conformant result rendered in the analysis modal.
2. **Given** a Coin Search request, **When** the agent runs, **Then** every dealer URL in the result is one returned by a tool call (no invented URLs — Principle VI).
3. **Given** a Coin Shows request, **When** the verification node runs, **Then** every date in the output is in the future.

### User Story 4 — Daily Coin of the Day (Priority: P2)

Brian wants the app to surface one of his own coins each morning as a notification, with a cached AI-style summary, to keep him engaged with his catalog.

**Independent Test**: Trigger `POST /admin/coin-of-day/run`, verify exactly one un-shown coin per enrolled user is picked, an in-app notification is created with `type=coin_of_day` and `referenceId` equal to the `FeaturedCoin.ID`, and a Pushover notification is dispatched.

**Acceptance Scenarios**:
1. **Given** an enrolled user with N owned non-wishlist non-sold coins, **When** the scheduler runs N+1 times across N+1 days, **Then** every coin appears exactly once before any repeat.
2. **Given** the scheduler ran today, **When** triggered again, **Then** dual idempotency (in-memory map + `HasBeenFeaturedToday` DB check) prevents a duplicate.

### User Story 5 — Authenticate with passkey or password (Priority: P1)

Brian wants frictionless biometric login on mobile (WebAuthn passkey) with password fallback, and the session must survive token refresh transparently.

**Independent Test**: Register a passkey, log out, log back in via biometric prompt, observe access token auto-refresh on a 401 without user-visible disruption, and confirm DOMException paths (user-canceled prompt) surface a usable error message.

**Acceptance Scenarios**:
1. **Given** a registered passkey, **When** the user taps "Sign in with passkey", **Then** the WebAuthn ceremony completes and a JWT + refresh token are issued.
2. **Given** an expired access token, **When** the SPA makes an authenticated request, **Then** the Axios interceptor refreshes via the 401 queue and retries the original request without user action.

### User Story 6 — Install as PWA and read offline (Priority: P2)

Brian wants to install the app to his phone home screen and at least browse previously-viewed coins without network.

**Independent Test**: On a mobile browser, accept the install prompt, launch from home screen, disable network, and verify cached collection views and coin detail pages render from the service worker cache.

**Acceptance Scenarios**:
1. **Given** the app is installed, **When** launched offline, **Then** previously-fetched coin lists/details render.
2. **Given** offline, **When** the user attempts a write, **Then** the failure is surfaced gracefully (offline-write is explicitly out of scope for v1.0).

### Edge Cases

- Empty collection (new user) — list views must render a useful empty state.
- AI provider unreachable — `GET /ai-status` returns `available: false`; UI must disable AI actions and explain why.
- Image upload >server limit — return a typed error; UI must surface the field-level failure.
- Sold coin in `random` sort — must be excluded from active-collection gallery but reachable via the "Sold" filter.
- User cancels WebAuthn prompt — `DOMException` must be caught and not crash the login view.
- Coin-of-the-day with zero eligible coins — scheduler logs `skipped` and emits no notification.

## Requirements

### Functional Requirements

- **FR-001**: System MUST provide create/read/update/delete operations on coins with images, attribution (ruler, mint, date), measurements (weight, diameter), category (Roman/Greek/Byzantine/Modern), material, valuation, provenance notes, and tags.
- **FR-002**: System MUST authenticate users via JWT (access + refresh) and MUST support WebAuthn passkey registration and login as an alternative to password.
- **FR-003**: System MUST provide list, search, filter, and sort operations on the collection, including a deterministic `?sort=random&seed=N` gallery shuffle.
- **FR-004**: System MUST proxy all AI agent calls from the Vue SPA through the Go API to the Python agent service via SSE — the SPA MUST NOT call the agent directly (Principle III).
- **FR-005**: System MUST support five AI team pipelines: coin search, coin shows, coin analysis (vision), portfolio review, availability check.
- **FR-006**: System MUST support both Anthropic and Ollama as interchangeable AI providers, selectable via the `AIProvider` admin setting.
- **FR-007**: System MUST pick one Coin of the Day per enrolled user via a scheduler, with dual idempotency (in-memory + DB), and dispatch in-app + Pushover notifications.
- **FR-008**: System MUST be installable as a PWA and MUST serve a service-worker-cached read-only view when offline.
- **FR-009**: Admin users MUST be able to manage application-wide settings via a key-value `AppSetting` model surfaced in an Admin Settings page.
- **FR-010**: Public visitors MUST be able to view coins explicitly marked public without authentication.

### Key Entities

- **User** — authenticated principal; owns coins; opts in/out of Coin of the Day; may hold passkey credentials and an admin role.
- **Coin** — primary aggregate; has many Images; categorized; valued; flagged sold/wishlist/public.
- **FeaturedCoin** — historical record of a Coin of the Day pick; ties a User to a Coin on a date with a cached summary.
- **Notification** — in-app notification with `type` and `referenceId` (e.g., `coin_of_day` references a `FeaturedCoin.ID`).
- **AppSetting** — key-value admin configuration (AI provider, schedule windows, etc.).
- **PasskeyCredential** — WebAuthn public key + counter scoped to a User.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A new coin (form + 2 images) can be added in under 60 seconds on a mobile viewport.
- **SC-002**: List view first paint < 1.5s on a 50-coin catalog over local network.
- **SC-003**: AI analysis stream begins emitting tokens within 5s of request on a configured Anthropic provider.
- **SC-004**: Random gallery seed survives page refreshes within the same browser session 100% of the time.
- **SC-005**: Architecture tests (`go test ./...` including `architecture_test.go`) pass on every commit touching `src/api/`.
- **SC-006**: PWA install passes Lighthouse PWA checks on the production build.

## Assumptions

- Single primary user with a small number of invited friends — no multi-tenant isolation beyond per-user ownership.
- Self-hosted on a small VPS or home server; SQLite single-node is acceptable for this scale.
- Brian provides his own Anthropic API key or runs Ollama locally; the app never ships keys.
- Browser support targets modern evergreen browsers (Chrome/Safari/Firefox current); no IE/legacy fallbacks.
- Offline-write is intentionally deferred to a future spec (see F006).

## Out of Scope (v1.0)

- Marketplace, trading, or payment processing.
- Multi-user collections per account (each user owns exactly one logical collection).
- Public social network beyond optional follow (tracked in F005).
- Native iOS/Android apps — PWA only.
- Offline writes / conflict resolution.
- Bulk import from third-party catalogs.

## Open Questions

None — the v1.0 feature surface shipped. Forward-looking questions belong in `specs/00N-*` (N ≥ 2) or in `specs/_backlog/F0NN-*` cards.

## Notes

Retroactive spec authored 2026-05-28 to anchor Constitution §0 Hierarchy item 3 ("active feature spec") with real content and validate the SpecKit workflow end-to-end. All `tasks.md` checkboxes are checked because every item is already present in production code as of v1.0. Active forward-looking work should open `specs/002-*/` and onward; this folder remains as a historical anchor and SHOULD NOT be edited except to add a `## History` note if the v1.0 surface is materially restated by a future amendment.
