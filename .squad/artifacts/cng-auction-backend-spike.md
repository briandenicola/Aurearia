# CNG Auctions Backend Integration Spike

**Date:** 2026-06-30  
**Engineer:** Cassius (Backend Developer)  
**Status:** Findings & Recommendations (No Code)  
**Scope:** Backend/API layer for adding https://auctions.cngcoins.com alongside NumisBids

---

## Executive Summary

Adding CNG Auctions to the Ancient Coins backend is **technically feasible** and follows an established pattern already in place for NumisBids. The integration will require HTML scraping (no public API available), credential storage with careful security handling, and minor data model changes. Estimated scope: **2–3 sprints** for full feature parity, phased as:

1. **Phase 1 (1 sprint):** Core scraping service, login flow, watchlist fetch
2. **Phase 2 (1 sprint):** Lot detail parsing, image extraction, URL normalization
3. **Phase 3 (0.5–1 sprint):** Settings, admin UI, scheduler integration, testing

The main risks are:
- **Scraping fragility** — CNG's platform is Auction Mobility–based, undocumented, subject to DOM changes
- **Credential storage** — User credentials must be encrypted or ephemeral (not persisted)
- **Rate limiting & detection** — Aggressive scraping may trigger anti-bot measures
- **Authentication complexity** — CNG may use different auth mechanisms than NumisBids

---

## Current NumisBids Architecture

### Data Model

**`AuctionLot` model** (`src/api/models/auction_lot.go`):
```
ID, NumisBidsURL, SaleID, LotNumber, AuctionHouse, SaleName, SaleDate, 
AuctionEndTime, Title, Description, Notes, Category, Estimate, CurrentBid, 
MaxBid, Currency, Status (enum), ImageURL, CoinID (FK), EventID (FK), UserID (FK)
```

**Status enum:**
- `watching`, `bidding`, `won`, `lost`, `passed`

**Key constraint:** `NumisBidsURL` (string, not null) — currently assumes source is always NumisBids.

### Services Layer

**`NumisBidsService`** (`src/api/services/numisbids_service.go`) — ~620 lines:

1. **Login** (`Login(username, password) → *http.Client`)
   - Form POST to `https://www.numisbids.com/registration/login.php`
   - Returns authenticated HTTP client with session cookie jar
   - Includes verification step (`verifyAuthentication`) to confirm login success

2. **Watchlist Fetch** (`FetchWatchlist(client) → string`)
   - GET `https://www.numisbids.com/watchlist`
   - Returns raw HTML; checks for login prompt to detect auth failure

3. **Watchlist Parse** (`ParseWatchlist(rawHTML) → []WatchlistLot`)
   - Regex-based extraction of lot links from watchlist HTML
   - Parses `href`, image URLs, estimates, currencies
   - Returns slice of `WatchlistLot` structs (source-agnostic struct)

4. **Lot Detail Scrape** (`ScrapeLotPage(lotURL) → *LotPageDetails`)
   - Fetches individual lot page HTML
   - Extracts og:image, auction house, sale name, current bid, description
   - Uses regex patterns for parsing; handles edge cases (ranges, multi-line)

5. **Date Parsing** (`ParseSaleDate(raw) → *time.Time`)
   - Handles NumisBids format: `"20-21 Apr 2026"` → last date of range
   - Fallback layouts for month abbreviations

### Handlers Layer

**`AuctionLotHandler`** (`src/api/handlers/auction_lots.go`):
- Depends on `NumisBidsService` injected at construction
- Routes:
  - `POST /auctions/validate-credentials` — test login
  - `POST /auctions/sync` — sync user's watchlist
  - `POST /auctions/import` — import single lot by URL

**Key handler patterns:**
- `ValidateNumisBids(c)` — calls `nbSvc.Login()`, returns 200 if successful
- `SyncWatchlist(c)` — calls `nbSvc.Login()`, `FetchWatchlist()`, `ParseWatchlist()`, then bulk-creates/updates `AuctionLot` records
- `ImportFromURL(c)` — calls `nbSvc.ScrapeLotPage()`, creates single record

### Repository Layer

**`AuctionLotRepository`** (`src/api/repository/auction_lot_repository.go`):
- Standard CRUD: `Create`, `GetByID`, `List`, `Update`, `Delete`, `UpdateFields`
- Filtering: by status, search (title/description/auction_house), pagination, sorting
- Multi-step writes use transactions

### Schedulers

**`AuctionEndingScheduler`** (`src/api/services/auction_ending_scheduler.go`):
- Periodic check (daily) for auctions ending within 24 hours
- Queries all `AuctionLot` records for current user, filters by `AuctionEndTime`
- Sends Pushover notifications to enrolled users

### Settings & Configuration

**AppSetting model** (key-value store):
- No explicit NumisBids-related settings yet; credentials are per-request only
- Future: could store encrypted credentials (risky), or just API keys

### Wiring (main.go)

```go
nbSvc := services.NewNumisBidsService(logger)
auctionUserRepo := repository.NewUserRepository(database.DB)
auctionLotHandler := handlers.NewAuctionLotHandler(auctionLotRepo, auctionLotSvc, auctionUserRepo, nbSvc, logger)
// Routes: GET /auctions, POST /auctions/sync, POST /auctions/validate-credentials, etc.
```

---

## CNG Auctions Assessment

### Public URL Structure

**CNG's auction platform:** `https://auctions.cngcoins.com/`  
**Platform:** Auction Mobility (third-party software — undocumented)  
**URL pattern for lots:**
- Primary: `https://www.cngcoins.com/Lot.aspx?LOT_ID=XXXXXX` (if same vendor)
- or: `https://auctions.cngcoins.com/<sale-id>/lot/<lot-id>` (needs verification with credentials)

**Architecture:** AngularJS-based SPA (based on public page inspect); real data fetching likely via AJAX/API or dynamically loaded.

### Key Differences from NumisBids

| Aspect | NumisBids | CNG Auctions |
|--------|-----------|--------------|
| **Platform** | Proprietary | Auction Mobility (3rd party) |
| **Public API** | None | None |
| **Auth Type** | Email + password, session cookies | Unknown (needs testing) |
| **Watchlist Access** | `/watchlist` (HTML page) | Unknown; may require AJAX token |
| **Rate Limiting** | Not documented | Unknown; possibly stricter |
| **DOM Stability** | Relatively stable | Likely to change (AngularJS templates) |
| **Image URLs** | `images.numisbids.com` | Unknown domain |
| **Lot Details** | Individual page scrape | Unknown; may be AJAX-loaded |

### Risks & Unknowns

1. **Authentication Flow**
   - Is it form-based like NumisBids, or OAuth, or 2FA-gated?
   - Does it use CSRF tokens?
   - Are session cookies stable?
   - **→ Requires credentialed testing**

2. **Watchlist Availability**
   - NumisBids exposes a public HTML watchlist page
   - CNG may not; watchlist could be AJAX-fetched or hidden behind different URL
   - **→ Requires credentialed access to confirm**

3. **Rate Limiting & Bot Detection**
   - Auction Mobility may have stricter anti-scraping measures
   - Could include IP bans, CAPTCHA, or request throttling
   - **→ Requires stress testing after implementation**

4. **DOM Fragility**
   - AngularJS frontends often use auto-generated class names
   - CNG's templates may lack stable anchors for regex parsing
   - **→ Test parse logic against live pages; monitor for breakage**

5. **Lot URL Stability**
   - Need to confirm primary URL pattern (`/Lot.aspx?LOT_ID=...` vs `auctions.cngcoins.com/...`)
   - Relative vs. absolute link handling differs
   - **→ Requires scraping sample pages**

---

## Proposed Backend Changes

### 1. Data Model Refactoring

**Problem:** `AuctionLot.NumisBidsURL` hardcodes the source.

**Solution:** Introduce an auction source field:

```go
// In models/auction_lot.go

type AuctionSource string

const (
    AuctionSourceNumisBids AuctionSource = "numisbids"
    AuctionSourceCNG       AuctionSource = "cng"
)

type AuctionLot struct {
    // ... existing fields ...
    Source             AuctionSource `gorm:"type:varchar(20);default:'numisbids'" json:"source"`
    SourceURL          string        `gorm:"not null" json:"sourceUrl"`       // generic URL field
    SourceLotID        string        `gorm:"index" json:"sourceLotID,omitempty"` // source-specific ID
    SourceSaleID       string        `json:"sourceSaleID,omitempty"`          // source-specific sale ID
    
    // Backward compat: alias NumisBidsURL to SourceURL for serialization
}
```

**Migration:**
- Add columns: `source`, `source_url`, `source_lot_id`, `source_sale_id`
- Backfill `source = 'numisbids'` and `source_url = numisbids_url` for existing rows
- Drop `numisbids_url` after verification (optional, keep for safety)

**Alternative (simpler, less breaking):**
- Keep `NumisBidsURL` as-is; add `SourceURL` (use it if non-null, else fall back to `NumisBidsURL`)
- No migration needed; minimal schema change

**Recommendation:** Use the simpler alternative for Phase 1; refactor in Phase 2 if needed.

### 2. Scraping Service Layer

**New: `CNGAuctionsService`** (`src/api/services/cng_auctions_service.go`):

Mirrors `NumisBidsService` structure:

```go
type CNGAuctionsService struct {
    logger *Logger
    baseURL string  // e.g., "https://www.cngcoins.com" or "https://auctions.cngcoins.com"
}

func NewCNGAuctionsService(logger *Logger) *CNGAuctionsService { ... }

// Login(username, password) (*http.Client, error)
// FetchWatchlist(client) (string, error)
// ScrapeLotPage(lotURL) (*LotPageDetails, error)
// ParseWatchlist(rawHTML) []WatchlistLot
// VerifyAuthentication(client) error
```

**Key implementation notes:**
- Use same `WatchlistLot` and `LotPageDetails` structs as NumisBids (agnostic)
- User-Agent header (identical to NumisBids pattern)
- Regex patterns tuned to CNG's HTML structure (requires sample pages)
- Date parsing adapted for CNG's format (needs validation)
- Error handling: return `ErrCNGAuthenticationRequired` analog

**File: ~500–600 lines** (similar complexity to NumisBids)

### 3. Handler Layer Changes

**Option A (Recommended):** Abstract scraping service behind an interface

```go
// In services/auction_source.go

type AuctionSourceService interface {
    Login(username, password string) (*http.Client, error)
    FetchWatchlist(client *http.Client) (string, error)
    ScrapeLotPage(lotURL string) (*LotPageDetails, error)
    ParseWatchlist(rawHTML string) []WatchlistLot
}

// NumisBidsService and CNGAuctionsService both implement AuctionSourceService
```

**Handler refactor:**

```go
type AuctionLotHandler struct {
    repo            *repository.AuctionLotRepository
    svc             *services.AuctionLotService
    userRepo        *repository.UserRepository
    auctionServices map[string]services.AuctionSourceService  // key: "numisbids", "cng"
    logger          *services.Logger
}

func (h *AuctionLotHandler) SyncWatchlist(c *gin.Context) {
    userID := c.GetUint("userId")
    var req struct {
        Source   string `json:"source"` // "numisbids" or "cng"
        Username string `json:"username"`
        Password string `json:"password"`
    }
    
    service, ok := h.auctionServices[req.Source]
    if !ok {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown auction source"})
        return
    }
    
    client, err := service.Login(req.Username, req.Password)
    // ... rest of logic (source-agnostic)
}
```

**Option B (Simpler):** Keep handler as-is, add new routes

```go
protected.POST("/auctions/sync/cng", writeRateLimit, auctionLotHandler.SyncWatchlistCNG)
protected.POST("/auctions/validate-credentials/cng", auctionLotHandler.ValidateCNG)
```

**Recommendation:** Use **Option A** (interface-based) — enables future sources, keeps logic DRY.

### 4. Settings & Credential Storage

**Design:** Do NOT persist credentials in the database.

**Rationale:**
- Encryption overhead and key rotation complexity
- User may not trust server with persistent credentials
- Credentials can be re-entered per session (acceptable UX for on-demand sync)

**Approach:**
- Accept credentials in request body (HTTPS-only; go through middleware validation)
- Create authenticated client for this request only
- Fetch/parse data; close connection
- Store results in `AuctionLot`, discard credentials

**Security checklist:**
- ✓ Credentials never logged or exposed in error messages
- ✓ HTTPS enforced (existing middleware: `SecurityHeaders()`)
- ✓ Rate limiting on credential validation endpoint
- ✓ No credential audit trail (or: log only "login attempted" without details)
- ✓ Consider: add "credential validation" as an admin setting (toggle per source)

**Optional (Phase 2):** Encrypted credential storage

If users request persistent sync (scheduled watchlist refresh), implement:
1. Encrypt credentials with user's password (derived key)
2. Store in DB; decrypt on scheduler run
3. Document the risks; make opt-in

### 5. Repository & Query Changes

**Minimal changes:**
- `List()` accepts new filter: `Source` (e.g., `"cng"`, `"numisbids"`, `""` for all)
- `Create()` accepts `Source` field
- Backward compat: default `Source = "numisbids"` if not provided

```go
type AuctionLotListFilters struct {
    Status    string
    Search    string
    Source    string  // NEW: "numisbids", "cng", or ""
    SortField string
    SortOrder string
    Page      int
    Limit     int
}

func (r *AuctionLotRepository) List(userID uint, filters AuctionLotListFilters) (...) {
    query := r.db.Model(&models.AuctionLot{}).Scopes(OwnedBy(userID))
    
    if filters.Source != "" {
        query = query.Where("source = ?", filters.Source)
    }
    // ... rest
}
```

### 6. Scheduler Changes

**`AuctionEndingScheduler`:** No changes needed

- Scheduler runs on all `AuctionLot` records regardless of `Source`
- Filters by `AuctionEndTime` (source-agnostic)
- Notifications are per-user, not per-source

**Optional Phase 2:** Per-source scheduler stats

- Track "last sync" per source per user (add to `AppSetting` or new table)
- Admin dashboard: see sync status across sources

---

## Phased Implementation Plan

### Phase 1: Core CNG Service (1 sprint)

**Deliverables:**
1. `CNGAuctionsService` (skeleton + login flow)
2. HTML sample parsing (regex patterns)
3. Unit tests for login/auth detection
4. Internal documentation of URL structure & DOM selectors

**Acceptance Criteria:**
- [ ] Login succeeds with valid credentials (credentialed testing)
- [ ] Login fails gracefully with invalid credentials
- [ ] Session verification works (analogue to NumisBids)
- [ ] Unit tests pass (mocked HTTP)

**Risks:**
- Authentication flow unknown (first blocker)
- Rate limiting during testing

**Blockers:**
- Temporary access to credentials (provided by user)
- Sample HTML pages for parsing

### Phase 2: Data Parsing & Handler Integration (1 sprint)

**Deliverables:**
1. Watchlist parse logic (Watchlist URL discovery + regex)
2. Lot detail scraping (LOT_ID extraction, detail page parsing)
3. Handler routes: `POST /auctions/sync/cng`, `POST /auctions/validate-credentials/cng`
4. Integration tests (end-to-end sync)
5. Data model migration (add `Source`, `SourceURL` fields)

**Acceptance Criteria:**
- [ ] Watchlist fetch returns parsed `[]WatchlistLot`
- [ ] Lot detail scrape populates image, estimate, currency, description
- [ ] Sync route creates/updates `AuctionLot` records in DB
- [ ] Backwards compatibility: existing NumisBids lots still work

**Risks:**
- CNG DOM fragility (changes break regex)
- Rate limiting triggers mid-testing

### Phase 3: Settings, Scheduler, Admin UI (0.5–1 sprint)

**Deliverables:**
1. Frontend: source filter in auction list, validation UI
2. Admin: source stats, last-sync timestamp tracking
3. Settings migration: add `CNG Auctions` toggle (if desired)
4. Optional: scheduled sync for CNG (if Phase 2 user feedback requests it)

**Acceptance Criteria:**
- [ ] User can filter by source (NumisBids vs. CNG)
- [ ] Admin dashboard shows source breakdown
- [ ] Scheduler integrates CNG auctions in ending-time checks

---

## URL Normalization Strategy

### Dual-Source Handling

**Problem:** NumisBids and CNG have different URL schemes. We need consistent, canonical URLs.

**Solution:** Store original URL + source, reconstruct canonically.

```go
type AuctionLot struct {
    Source        AuctionSource  // "numisbids" or "cng"
    SourceLotID   string         // e.g., NumisBids: "10003", CNG: "123456"
    SourceSaleID  string         // e.g., NumisBids: "10749", CNG: "N/A" or ""
    SourceURL     string         // original/full URL from watchlist
    
    // Canonical getters
    CanonicalURL() string {
        switch s.Source {
        case AuctionSourceNumisBids:
            if s.SourceSaleID != "" {
                return fmt.Sprintf("https://www.numisbids.com/sale/%s/lot/%s", s.SourceSaleID, s.SourceLotID)
            }
            return s.SourceURL
        case AuctionSourceCNG:
            return fmt.Sprintf("https://www.cngcoins.com/Lot.aspx?LOT_ID=%s", s.SourceLotID)
        }
        return s.SourceURL
    }
}
```

### Import Handler

```go
func (h *AuctionLotHandler) ImportFromURL(c *gin.Context) {
    var req struct {
        URL string `json:"url" binding:"required"`
    }
    
    userID := c.GetUint("userId")
    
    source, lotID, ok := parseAuctionURL(req.URL)
    if !ok {
        c.JSON(400, gin.H{"error": "Unrecognized auction URL"})
        return
    }
    
    service, ok := h.auctionServices[source]
    if !ok {
        c.JSON(400, gin.H{"error": "Auction source not supported"})
        return
    }
    
    details, err := service.ScrapeLotPage(req.URL)
    // ... create AuctionLot with Source, SourceLotID, SourceURL
}

func parseAuctionURL(urlStr string) (string, string, bool) {
    // Try NumisBids patterns
    if strings.Contains(urlStr, "numisbids.com") {
        // extract SaleID/LotID
        return "numisbids", ..., true
    }
    // Try CNG patterns
    if strings.Contains(urlStr, "cngcoins.com") {
        // extract LOT_ID
        return "cng", ..., true
    }
    return "", "", false
}
```

---

## Security & Credential Handling

### Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| **Credentials in logs** | Never log plaintext passwords; sanitize error messages |
| **Credentials in URLs** | Store in request body only (HTTPS enforced) |
| **Brute-force login** | Rate limit `/validate-credentials` endpoints |
| **Session hijacking** | Use HTTPS + secure cookies; HttpOnly flags |
| **Scraped data exposure** | User-scoped queries; respect `OwnedBy` scope in repository |
| **IP bans from aggressive scraping** | Implement backoff; add per-user rate limiting |

### Implementation Checklist

- [ ] Middleware: `SecurityHeaders()` enforces HTTPS (already in place)
- [ ] Endpoint: Rate limit on credential validation (add `writeRateLimit` or custom)
- [ ] Service: Sanitize error messages (never expose password/token details)
- [ ] Logging: Log only action (e.g., "login failed") + generic reason (not credentials)
- [ ] Storage: If credentials cached, encrypt; no plaintext in DB or logs
- [ ] Testing: Unit tests with mocked credentials; no real logins in CI

---

## Testing Strategy

### Unit Tests

**Existing patterns** (from `numisbids_service_test.go`):
- Test watchlist parsing against HTML fixtures
- Test lot detail extraction
- Test URL parsing (relative vs. absolute)
- Test currency/estimate parsing

**CNG tests (same pattern):**
```go
func TestCNGParseWatchlist(t *testing.T)
func TestCNGScrapeLotPage(t *testing.T)
func TestCNGParseAuctionURL(t *testing.T)
func TestCNGDateParsing(t *testing.T)
```

Fixtures: Sample CNG HTML pages (manually saved from credentialed testing).

### Integration Tests

- End-to-end sync with mocked HTTP (use httptest.Server)
- Database state before/after sync
- Verify `Source` and `SourceLotID` fields are populated

### Stress Testing

*Post-Phase 1, requires production-like environment:*
- Sync 1000+ CNG lots; measure response time
- Monitor for rate-limit errors (HTTP 429, etc.)
- Check for IP bans or CAPTCHA triggers

---

## Migration & Backward Compatibility

### No Breaking Changes

**Schema:**
- Add new fields; don't drop or rename `NumisBidsURL`
- Default `Source = 'numisbids'` for existing records
- Add DB index on `(UserID, Source)` for filtered queries

**API:**
- Existing endpoints remain unchanged
- New endpoints: `POST /auctions/sync/cng`, `POST /auctions/validate-credentials/cng`
- Optional: Add `source` filter to `GET /auctions` (backward compat: omit to get all)

**UI (Frontend Responsibility):**
- Add source selector in watchlist sync UI
- Display source badge on auction lot cards

---

## Open Questions (Require Credentialed Testing)

1. **Authentication Flow**
   - What is CNG's login endpoint?
   - Does it return JSON or HTML?
   - Are session cookies named predictably (e.g., `PHPSESSID`, `.AspNet.ApplicationCookie`)?
   - Is there a 2FA requirement?

2. **Watchlist Access**
   - Is there a public watchlist URL like NumisBids `/watchlist`?
   - Is watchlist AJAX-loaded or server-side rendered?
   - If AJAX, what endpoint & parameters?
   - What response format (HTML, JSON)?

3. **Lot Page Structure**
   - What is the base URL for lot pages?
   - Do lot IDs follow a predictable pattern?
   - What DOM selectors/classes identify key fields (title, image, estimate)?
   - Is the og:image meta tag present (for image extraction)?

4. **Rate Limiting**
   - Does CNG throttle repeated requests?
   - Are there IP-level or account-level limits?
   - What status codes indicate rate limiting (429, 403, 503)?

5. **Currency Handling**
   - What currencies does CNG use?
   - How are they formatted in HTML (e.g., "USD", "£", "€")?

6. **Seasonal/Archival Data**
   - Are completed auctions accessible (for historical tracking)?
   - URL pattern for archived lots?

---

## Cost Estimate

| Phase | Sprint | Effort | Risk |
|-------|--------|--------|------|
| **1. Core Service** | 1 | 5–8 pts | High (auth unknown) |
| **2. Parser + Handler** | 1 | 8–13 pts | Medium (DOM fragility) |
| **3. Settings + Admin** | 0.5–1 | 3–5 pts | Low |
| **Testing & Polish** | TBD | 3–5 pts | Low |
| **Stress Testing** | TBD | 2–3 pts | Medium (rate limits) |

**Total: 21–34 story points (3–4 sprints)**

---

## Recommendation

1. **Immediate (Pre-Phase 1):**
   - User provides temporary credentials
   - Brian logs into CNG, captures sample pages (watchlist + lot details)
   - Cassius analyzes HTML structure, identifies selectors

2. **Phase 1 (Next Sprint):**
   - Build `CNGAuctionsService` skeleton
   - Test login/auth against live CNG (within rate limits; ~10 logins max)
   - Document findings in `.squad/skills/cng-auction-scraping/`

3. **Phase 2–3:**
   - Scale parsing logic; integrate handler
   - Front-end UI changes (concurrent with backend)
   - Stress test before production rollout

4. **Risk Mitigation:**
   - Add circuit breaker for CNG service (fail gracefully if site changes)
   - Monitor scraping logs; alert on increased error rates
   - Document all regex patterns with rationale (for future maintenance)

---

## Files Affected (No Edits Required for Spike)

**New files to create:**
- `src/api/services/cng_auctions_service.go` (Phase 1)
- `src/api/services/cng_auctions_service_test.go` (Phase 1)
- `src/api/services/auction_source.go` (Phase 2, interface)
- `.squad/skills/cng-auction-scraping/SKILL.md` (Phase 1 findings)

**Files to modify (Phase 2+):**
- `src/api/models/auction_lot.go` (add Source field)
- `src/api/repository/auction_lot_repository.go` (add Source filter)
- `src/api/handlers/auction_lots.go` (refactor with interface)
- `src/api/main.go` (wire CNGAuctionsService)
- `src/api/database/database.go` (add migration)

**Frontend (separate from this spike):**
- `src/web/src/views/AuctionList.vue` (add source filter)
- `src/web/src/api/client.ts` (add CNG endpoints)

---

## Architecture Compliance

**Principle I (Clear Layered Architecture):** ✓
- CNGAuctionsService in `services/` layer
- Handler delegates to service (no business logic)
- Repository owns queries

**Principle II (Service Boundary Separation):** ✓
- All scraping in Go API (no Python agent involvement)
- Stateless service (no database access)

**Principle XI (Security Hardening):** ✓
- Input validation on URLs
- No credentials logged
- HTTPS enforced
- Rate limiting on credential endpoints

**Principle VII (Schema-Driven Contracts):** ✓
- Swagger annotations on all new handlers
- Response structs defined and typed

---

## Appendix: Reference URLs

- **NumisBids Base:** https://www.numisbids.com
- **CNG Auctions:** https://auctions.cngcoins.com/
- **CNG Legacy Lots:** https://www.cngcoins.com/Lot.aspx?LOT_ID={ID}
- **Auction Mobility (Platform):** https://www.auctionmobility.com/

