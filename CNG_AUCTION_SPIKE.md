# CNG Auctions Integration Spike Report

**Date:** 2026-06-30  
**Owner:** Maximus (Lead/Architect)  
**Objective:** Assess architecture and implementation effort to add CNG Auctions (https://auctions.cngcoins.com/) alongside existing NumisBids auction tracking.  
**Scope:** Spike plus first provider-aware implementation on `spike/cng-auctions`.  
**Credentials:** Temporary credentials were used only through local environment/HAR-assisted verification and were not committed.

---

## Executive Summary

Ancient Coins currently tracks auction lots exclusively from NumisBids. The spike analyzes adding CNG Auctions as a second auction source using the same feature boundaries. **Recommended approach: treat CNG as a second "auction provider" within the existing AuctionLot schema**, reusing the current UI/service layers with provider-aware scrapers and credential storage.

**Key Finding:** The existing NumisBids architecture is provider-specific (hardcoded URLs, login flow, HTML scraping). CNG will require parallel scraper, separate credential storage, and provider-agnostic UI toggles — but reuses the entire data model and service layer.

**Implementation Status:** Provider-aware CNG manual import and watched-lot sync are implemented. Remaining rollout risk is CNG site structure stability, credential storage parity with existing NumisBids behavior, and beta feedback.

---

## Current NumisBids Architecture

### Data Model
- **AuctionLot** table: single `NumisBidsURL` field; no `provider` or `auction_source` column
- Lot statuses: `watching`, `bidding`, `won`, `lost`, `passed`
- User stores credentials in `User.NumisBidsUsername` and `User.NumisBidsPassword` (plaintext in DB; considered acceptable per design)
- **AuctionEvent** for calendar linking (separate from NumisBids; supports both sources)

### Service Layer
- **NumisBidsService** (`src/api/services/numisbids_service.go`): 
  - Single-provider scraper with hardcoded URLs, regex parsers, and HTML extraction
  - Methods: `Login()`, `FetchWatchlist()`, `ParseWatchlist()`, `ScrapeLotPage()`, `ScrapeLotImage()`
  - Session cookie-based auth; validates login by confirming watchlist page loads without login form

### Handler Layer
- **AuctionLotHandler** (`src/api/handlers/auction_lots.go`):
  - `SyncWatchlist()`: reads stored NumisBids credentials, calls service, upserts lots
  - `ImportFromURL()`: manual import by pasting a NumisBids lot URL
  - `ValidateNumisBids()`: credential validation endpoint
  - Status transitions, conversion to coins, calendar linking — **all provider-agnostic**

### Frontend
- **ImportLotModal**: single "Add from NumisBids" form (URL paste + preview scrape)
- **SettingsAccountSection**: NumisBids username/password fields; validation toggle
- `syncNumisBidsWatchlist()` API call; no provider abstraction

### Public Site Research (CNG Auctions)
- **URL:** https://auctions.cngcoins.com/
- **Visible Structure:** 
  - Upcoming/live/past auctions with dates, locations
  - Auction types: "Live" and "Timed"
  - Individual lot pages (inferred from lot URLs in watchlist HTML)
  - AngularJS-based frontend with server-side HTML rendering
- **Authentication:** Unknown from public interface; likely session cookie-based (similar to NumisBids)
- **Watchlist:** Cannot verify without login
- **Lot Detail Pages:** Unknown structure; inferred to follow NumisBids pattern (image, estimate, current bid, description)

---

## Recommended Architecture Approach

### Option 1: Provider-Agnostic Factory (Recommended)

Create an **AuctionProvider interface** with implementations for NumisBids and CNG:

```
AuctionProvider (interface)
├── NumisBidsProvider
└── CNGProvider

AuctionLot (schema change: add provider field)
├── provider: "numisbids" | "cng"  // new field
├── sourceUrl: URL
└── (existing fields work for both)

User (schema change)
├── NumisBidsUsername, NumisBidsPassword
├── CngUsername, CngPassword  // new fields
└── (etc.)

Handler layer (minimal change)
├── SyncWatchlist(provider) → reuse existing sync logic
├── ImportFromURL(url, provider) → detect provider or infer
└── ValidateCredentials(provider, username, password)
```

**Pros:**
- Reuses service logic entirely; separates scraping concern
- Can add third auction source (Bonhams, Sotheby's) later without handler rewrites
- Clear provider strategy in settings UI

**Cons:**
- Requires schema migration (add `provider` column)
- Need to backfill existing NumisBids lots with `provider = 'numisbids'`
- Frontend must support provider picker in import modal

### Option 2: Parallel Table (Not Recommended)

Create separate `CngAuctionLot` table; reuse `AuctionLot` for NumisBids only.

**Cons:**
- Duplicates all business logic (status transitions, conversion, calendar linking)
- Violates DRY; UI must handle two separate queries
- Adding a third source becomes worse

---

## Scope & Feature Boundaries

### Exact Surfaces to Replicate

| Surface | NumisBids | CNG | Notes |
|---------|-----------|-----|-------|
| Import | Paste lot URL → preview scrape | Same UI, detect CNG domain | Need to identify CNG lot URL structure |
| Sync Watchlist | Login + fetch watchlist HTML + parse links | Same flow if CNG has watchlist | Requires authenticated access research |
| Lot Details | Scrape lot page for: image, auction house, sale name, current bid, estimate, description, lot number | Need CNG HTML structure | Public site doesn't expose; requires credentials |
| Status Lifecycle | watching → bidding → won/lost/passed | Same semantics | Both use date-based "sale passed" logic |
| Calendar Linking | Link lots to AuctionEvent by event ID | Same endpoint | Provider-agnostic already |
| Convert to Coin | "Won" lot → new Coin with source reference | Same | Works for both |
| Settings | Username/password fields | Add CNG fields | Separate sections in SettingsAccountSection |

### No Replication Needed
- Bulk actions (already provider-agnostic)
- Coin conversion logic
- Calendar event linking
- Lot status UI/filter/count endpoints

---

## Implementation Phases

### Current Status Summary

| Phase | Status | Notes |
|---|---|---|
| Phase 1: Research & Auth | Complete | CNG login, public lot pages, and authenticated `/watched-lots` were verified with normal Go HTTP; no headless browser required. |
| Phase 2: Schema & Service Layer | Complete | `AuctionLot` now has source/source URL/source IDs; `CNGAuctionService` parses `viewVars` and handles paginated watchlists. |
| Phase 3: Handler & Settings | Complete | Existing auction endpoints accept optional `source`; profile settings store CNG credentials and validation supports CNG. |
| Phase 4: Frontend & UX | Complete for MVP | Settings, import, sync, cards, detail modal, and calendar links are provider-aware and use existing UI patterns. |
| Phase 5: QA Hardening | Complete for MVP | Fixture-backed CNG service tests, provider-aware repository tests, OpenAPI regeneration, backend tests, vet, type-check, and frontend build pass. |
| Phase 6: Rollout | Ready for beta | Merge to beta after review; rotate the temporary CNG password after validation. |
| Phase 7: Future Hardening | Backlog | Credential encryption/secrets refactor and optional CNG AJAX/API refresh can be handled separately because NumisBids has the same credential-storage baseline. |

### Phase 1: Research & Auth (3–5 days)
**Goal:** Verify CNG site structure and authentication model.

**Deliverables:**
1. Sample CNG watchlist URL (requires temporary login from user)
2. Sample CNG lot page URL
3. Confirm login method (form-based? cookies? API token?)
4. Extract HTML structure: lot links pattern, lot detail page structure (image, current bid, estimate, etc.)
5. Identify if CNG requires user-agent spoofing or JavaScript execution (will affect Go scraper feasibility)

**Blocked On:** User-provided temporary credentials (email + password) to CNG for limited time.

**Critical Decision:** If CNG requires JavaScript rendering (dynamic content), fallback to:
- Headless browser service (Chromium) as subprocess — **increases Go deployment complexity significantly**
- Node.js wrapper for HTML rendering — adds another language to stack
- Accept reduced field extraction (some HTML unavailable without JS)

### Phase 2: Schema & Service Layer (3–5 days)
**Goal:** Extend data model and implement CNG scraper.

**Changes:**
1. **Schema:**
   - Add `provider` column to `AuctionLot` (default: 'numisbids' for existing rows)
   - Add `CngUsername`, `CngPassword` to `User` model
   - Add `provider` index to support queries like `GetByProviderAndURL()`

2. **Services:**
   - Create `CNGAuctionService` (parallel to NumisBidsService)
   - Implement: `Login()`, `FetchWatchlist()`, `ParseWatchlist()`, `ScrapeLotPage()`, `ScrapeLotImage()`
   - Create `AuctionProvider` interface; implement for both NumisBids and CNG
   - Update `AuctionLotService` to accept provider parameter in sync/import flows

3. **Repository:**
   - Add `GetByProviderAndURL()` (upsert key includes provider)
   - Update `Upsert()` to handle provider-keyed uniqueness
   - Support filtering by provider in list/count queries

4. **Tests:**
   - Unit tests for CNGAuctionService (mocked HTTP responses)
   - Unit tests for AuctionProvider factory
   - Integration tests for multi-provider sync

### Phase 3: Handler & Settings (2–3 days)
**Goal:** Extend API endpoints and user settings.

**Changes:**
1. **Handlers:**
   - Extend `ImportFromURL()` to detect provider from URL (domain parsing)
   - Extend `SyncWatchlist()` to accept optional `?provider=` query param (default: ask user if ambiguous)
   - Extend `ValidateCredentials()` to accept provider
   - Update `Get()`, `List()`, `Counts()` to optionally filter by provider

2. **User Settings (API & Frontend):**
   - Add `GetUserAuctionProviders()` endpoint (returns enabled providers + status)
   - Extend `UpdateUserSettings()` to save CNG username/password

3. **Tests:**
   - Handler tests for multi-provider sync
   - Credential validation for CNG

### Phase 4: Frontend & UX (2–3 days)
**Goal:** Update UI for provider selection and dual tracking.

**Changes:**
1. **ImportLotModal:**
   - Keep "Add from NumisBids" section
   - Add "Add from CNG Auctions" section (or tab)
   - Auto-detect provider if URL pasted (domain check)
   - Show provider icon/label in preview

2. **SettingsAccountSection:**
   - Keep existing NumisBids section
   - Add new "CNG Auctions Integration" section (parallel structure)
   - Reuse validation flow

3. **AuctionsPage:**
   - Add optional provider filter chip (NumisBids / CNG / All)
   - Show provider badge on lot cards (small label: "NumisBids" / "CNG")
   - Update sync button label: "Sync Watchlists" or dual-action menu

4. **Empty State:**
   - Update messaging: "Import lots from NumisBids or CNG Auctions"

**Tests:**
- Provider detection in ImportLotModal
- Settings form for CNG credentials
- Multi-provider filtering

---

## Security & Privacy Considerations

### Credential Handling
- **Current:** NumisBids creds stored plaintext in `User` table (`NumisBidsPassword` VARCHAR(100))
- **Recommendation for CNG:** **Do not introduce new plaintext storage**
  - **Option A (Recommended):** Migrate NumisBids to encrypted storage using existing vault/key management (if available)
  - **Option B (Quick):** Continue plaintext for CNG; document as known limitation; add admin warning
  - **Option C (Future):** Implement credential rotation endpoint (user enters password once per session, encrypted at rest, key refreshed on logout)

### HTTP Client Behavior
- Both NumisBids and CNG scrapers use cookie-based sessions
- Current: `http.Client` with `cookiejar.Jar`
- Risk: CSRF attacks if scraper requests come from same session (mitigated by server-side credential storage, not user-initiated)
- Mitigation: **Ensure CNG scraper uses dedicated client per user session** (already done in NumisBidsService)

### User-Agent Spoofing
- Current: `numisbidsUserAgent` hardcoded as Chrome user-agent
- Both sites may reject requests without valid user-agent
- **Recommendation:** Use realistic browser string; acceptable for scraping public content

### Data Retention
- Synced lot data is user-scoped; no cross-user visibility
- **New requirement:** If CNG lot URLs are shorter/common, ensure no collision with NumisBids URL uniqueness constraint
  - Add `(user_id, provider, source_url)` compound unique index

---

## Open Questions (Research Phase 1)

1. **Does CNG offer a watchlist feature similar to NumisBids?**
   - If not, fall back to manual import only (acceptable but less convenience)

2. **What is the CNG lot URL structure?**
   - NumisBids: `/sale/SALEID/lot/LOTNUMBER`
   - CNG: Unknown (sample needed)
   - Important: Must be parseable to extract `saleId` and `lotNumber` for deduplication

3. **Does CNG require JavaScript to render watchlist or lot detail pages?**
   - If yes, Go HTTP client cannot scrape; requires headless browser
   - If no, regex + HTML parser works

4. **What fields does CNG provide for each lot?**
   - Minimum needed: title, image, estimate, current bid, sale date, lot number, auction house
   - Optional: description, currency, hammer price

5. **Does CNG's login credential format differ?** (email vs username; different password encoding?)

6. **Are there any rate limits, bot detection, or anti-scraping measures?**
   - May need request throttling, retry logic, or proxy rotation
   - Risk: If detected, site could block the Ancient Coins app

7. **Does CNG have a public API or RSS feed?**
   - If available, reduces scraping risk entirely
   - Worth checking developer docs before committing to HTML scraping

8. **How frequently do CNG lot pages update?**
   - Affects sync interval recommendations
   - If real-time, consider polling strategy

---

## Risk Assessment

| Risk | Impact | Likelihood | Mitigation |
|------|--------|-----------|-----------|
| CNG site structure changes frequently | Scrapers break; sync fails | Medium | Implement robust error handling; version scrapers; monitor error rates |
| JavaScript rendering required | Go scraper cannot work; requires new infra | Medium | Research Phase 1; decide on headless browser if needed |
| Login credentials fail silently | User doesn't notice; lots don't sync | Medium | Add credential validation endpoint; periodic test login; error messages in UI |
| Rate limiting / bot detection | App gets IP-blocked | Low | Add exponential backoff; respect robots.txt; limit sync frequency |
| CNG watchlist not available | Feature limited to manual import | Low | Acceptable; document in settings |
| Plaintext credential storage | Compromise of DB exposes both NumisBids + CNG logins | Medium | Encrypt at rest (Phase 0 prerequisite) |
| Lot URL collision | Deduplication fails; user tracks same lot twice | Very Low | Add provider column to uniqueness constraint |

---

## Credential Handling Recommendation

**Do not ask user to paste credentials into chat or any web form without secure mechanism.**

Instead:
1. **Phase 1 Research:** User provides temporary CNG login credentials **via secure local mechanism only:**
   - Encrypted file shared locally (e.g., `cng-research-creds.json` in `.gitignore`)
   - Or: User provides credentials in a local environment variable during spike testing
   - Or: Use temporary test account created by CNG support

2. **Production (Phase 2+):** Add CNG username/password fields to Settings UI (same pattern as NumisBids)
   - User enters credentials in the app, encrypted on server, used by scheduler only
   - Never transmitted to Maximus or team

3. **Audit:** Ensure `User.CngPassword` is never serialized to API response (json tag `"-"`)

---

## Spike Deliverables

1. **Research Report** (completed by Phase 1):
   - CNG site structure analysis
   - Authentication method documentation
   - HTML scraping feasibility assessment
   - Risk report: JavaScript rendering, rate limiting, anti-scraping

2. **Proof-of-Concept (optional, depends on research findings):**
   - Sample CNG watchlist URL structure + lot URL structure
   - Sample lot detail page HTML
   - Go regex patterns for CNG field extraction

3. **Architecture Decision Record (ADR):**
   - Provider-agnostic vs. parallel-table decision
   - Credential storage encryption decision
   - Headless browser decision (if needed)

4. **Effort Estimate:**
   - Phase 1: 3–5 days (research)
   - Phase 2: 3–5 days (schema + service)
   - Phase 3: 2–3 days (handler + settings API)
   - Phase 4: 2–3 days (frontend)
   - **Total: 10–16 days (2–3 weeks)**

5. **Implementation Plan Draft** (Phase 1 output):
   - Detailed task breakdown
   - Dependency ordering
   - Test strategy
   - Rollout plan (flag for gradual enable?)

---

## Implementation Notes (if Proceeding)

### Principle Compliance

- **Principle I (Clear Layered Architecture):** Provider abstraction via interface; handlers remain thin
- **Principle II (Dependency Injection):** Both providers instantiated in `main.go`; passed to handler via constructor
- **Principle V (Security):** Credential storage requires encryption decision ADR
- **Principle IX (Architecture Tests):** Add test enforcing no direct CNG HTTP calls outside service layer

### Code Organization

```
src/api/
├── services/
│   ├── auction_lot_service.go     (unchanged)
│   ├── numisbids_service.go       (unchanged)
│   ├── cng_auction_service.go     (new)
│   └── auction_provider.go        (new: interface + factory)
├── repository/
│   └── auction_lot_repository.go  (extend: add provider filters)
├── handlers/
│   └── auction_lots.go            (extend: detect provider)
├── models/
│   ├── auction_lot.go             (add: provider field)
│   └── user.go                    (add: CNG creds)
```

### Testing Strategy

- Unit tests for CNGAuctionService (mocked HTTP)
- Integration tests for multi-provider sync flow
- Handler tests for provider detection
- E2E tests for settings UI (credential save/validate)
- Regression tests for existing NumisBids flow

### Feature Flags

Consider gradual rollout:
- `ENABLE_CNG_AUCTIONS` admin setting
- Show/hide CNG import section based on flag
- Allows testing in production before GA

---

## Conclusion

Adding CNG Auctions is feasible within the existing architecture. The work is clean, moderate-effort, and follows established patterns. **Critical path: Phase 1 research** to confirm CNG site structure and authentication method. **Prerequisite: Encrypt credential storage** to avoid plaintext proliferation.

**Recommended Next Step:** Spike Phase 1 (research) with temporary secure credential access from user.
