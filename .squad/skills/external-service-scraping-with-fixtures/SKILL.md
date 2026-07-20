# External Service Scraping with Fixtures & Credentials

## Pattern Overview

**Problem:** Scrape authenticated external auction sites (NumisBids, CNG) without committing credentials or creating test fragility from live HTTP calls.

**Solution:** Two-phase testing strategy:
1. **Spike Phase (Unit Tests):** Fixture-based (committed sanitized HTML snapshots), httptest stubs, no live calls, no credentials
2. **Implementation Phase (Integration Tests):** Environment-variable credentials, live testing deferred to CI/CD with temporary accounts

**Applies to:** NumisBids (existing), CNG Auctions (spike), and any future auction/marketplace integration.

---

## Architecture

```
┌─────────────────────────────────────────┐
│        External Service (e.g., CNG)     │
│    https://auctions.cngcoins.com        │
└──────────────▲──────────────────────────┘
               │
         ┌─────┴──────────────────────────────────┐
         │                                        │
    ┌────▼─────────────────────┐    ┌───────────▼──────────────┐
    │ Production Scraper        │    │ Test Stubs (httptest)    │
    │ (Phase 2, Live)           │    │ (Phase 1, Spike)         │
    │                           │    │                          │
    │ • Real HTTP client        │    │ • httptest.NewServer     │
    │ • Env var credentials     │    │ • Fixed responses        │
    │ • Session/cookie jar      │    │ • Error scenarios        │
    └────┬─────────────────────┘    └───────────┬──────────────┘
         │                                       │
    ┌────▼────────────────────────────────────────▼─────────┐
    │              Scraper Service Layer                     │
    │  (AuctionService, Parser, Auth handlers)              │
    │  • HTTP-agnostic business logic                       │
    │  • Error handling & retry policy                      │
    │  • Sentinel errors (auth required, rate-limited)      │
    └────┬──────────────────────────────────────────────────┘
         │
    ┌────▼──────────────┐       ┌────────────────────────┐
    │   Repository      │       │  Test Fixtures         │
    │  (AuctionLot)     │       │  (*.html snapshots)    │
    │                   │       │                        │
    │ • Stores parsed   │       │ • Committed to VCS     │
    │   data            │       │ • Immutable references │
    └───────────────────┘       │ • Updated via PR       │
                                └────────────────────────┘
```

---

## Phase 1: Spike (Unit Tests)

### Fixture Snapshots

**Goal:** Capture real external HTML once; reuse in all spike tests.

**Implementation:**

1. Manually fetch one real lot page from external site (e.g., `curl https://auctions.cngcoins.com/...`)
2. Save to VCS:
   ```
   src/api/services/testdata/
   ├── cng_public_lot.html          # Unauthenticated lot page
   ├── cng_authenticated_watchlist.html  # Logged-in watchlist (if different)
   ├── cng_sold_lot.html            # Edge case: sold/unavailable
   ├── cng_error_404.html           # 404 not found
   └── numisbids_watchlist.html     # Existing pattern (reference)
   ```

3. Load in tests:
   ```go
   func TestParseCNGLotPageExtractsFields(t *testing.T) {
       fixture := loadFixture(t, "cng_public_lot.html")
       // Parse fixture (no HTTP), validate fields
       svc := NewCNGAuctionService(logger)
       details, err := parseLotHTML(fixture)  // parse-only function
       require.NoError(t, err)
       require.NotEmpty(t, details.ImageURL)
   }
   ```

### Test Stub Pattern

**Goal:** Replace live HTTP with predictable responses.

**Implementation (existing pattern in `numisbids_service_test.go`):**

```go
// Stub login endpoint
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/registration/login.php" {
        http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "test"})
        w.Write([]byte(`{"status":"success"}`))
        return
    }
    if r.URL.Path == "/watchlist" {
        w.Write([]byte(`<a href="/sale/10/lot/5">Lot 5</a>`))
        return
    }
}))
defer server.Close()

// Use stub URL in test
svc := NewCNGAuctionService(logger)
client, err := svc.Login("user@test.com", "password")
require.NoError(t, err)
```

### Fixture Loader Helper

**Goal:** Consistent fixture loading across test files.

**Implementation:**

```go
// src/api/testutil/fixtures.go
package testutil

import (
    "os"
    "path/filepath"
    "testing"
)

// LoadFixture loads a test fixture HTML file.
func LoadFixture(t *testing.T, name string) string {
    t.Helper()
    path := filepath.Join("testutil", "fixtures", name)
    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("Failed to load fixture %q: %v", name, err)
    }
    return string(data)
}

// Usage in test
fixture := testutil.LoadFixture(t, "cng_public_lot.html")
```

### Error Scenario Testing

**Goal:** Test parser robustness to malformed/missing data.

**Implementation:**

```go
func TestCNGParser_MissingFields(t *testing.T) {
    tests := []struct {
        name    string
        html    string
        wantErr bool
    }{
        {
            name:    "Missing image",
            html:    `<div><h1>Lot Title</h1></div>`,
            wantErr: false, // Image is optional
        },
        {
            name:    "Malformed estimate",
            html:    `<div>Estimate: not_a_number USD</div>`,
            wantErr: false, // Estimate parse fails gracefully
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ParseCNGLotHTML(tt.html)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseCNGLotHTML error = %v, want %v", err, tt.wantErr)
            }
        })
    }
}
```

### Spike Acceptance Checklist

- [ ] `src/api/services/testdata/cng_*.html` committed with real-world HTML samples
- [ ] `TestParseCNG*` functions pass 100% (parser handles all fixtures)
- [ ] Auth stub tests pass (login, session, error scenarios)
- [ ] Regression: NumisBids parser tests still passing (`go test -v ./services -run TestNumisBids`)
- [ ] Architecture test passes: no new import violations (`go test ./... -run TestArchitecture`)
- [ ] `go vet ./...` and `go test ./...` pass from `src/api/`

---

## Phase 2: Implementation (Integration Tests)

### Environment Variable Credentials

**Goal:** Load credentials from environment; never hardcode.

**Implementation:**

```go
// src/api/services/cng_auction_service.go
package services

import "os"

func (s *CNGAuctionService) GetCredentials() (username, password string, err error) {
    username = os.Getenv("CNG_AUTH_USER")
    password = os.Getenv("CNG_AUTH_PASS")
    if username == "" || password == "" {
        return "", "", ErrCNGCredentialsMissing
    }
    return username, password, nil
}
```

### Integration Test File Convention

**Goal:** Separate integration tests (live calls) from unit tests.

**Implementation:**

```go
// src/api/services/cng_auction_service_integration_test.go
// +build integration

package services

import "testing"

func TestCNGAuctionService_FetchWatchlist_Live(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping live CNG test in short mode")
    }
    
    user, pass, err := getTestCredentials()
    if err != nil {
        t.Skip("CNG_AUTH_USER/CNG_AUTH_PASS not set")
    }
    
    svc := NewCNGAuctionService(logger)
    client, err := svc.Login(user, pass)
    require.NoError(t, err)
    // ... continue live test
}
```

### Run Integration Tests

```bash
# Unit tests only (fast, no live calls)
go test -v ./services

# Integration tests only (slow, live calls, requires credentials)
CNG_AUTH_USER=... CNG_AUTH_PASS=... go test -v ./services -tags integration

# All tests
go test -v ./...
```

### VCS Safety

**Goal:** Prevent credentials or live-session artifacts from leaking into repository.

**Implementation:**

```gitignore
# .gitignore (already present)
.env
.env.local
.env.*.local
secrets/
**/testdata/*_credentials.json
*.har
```

Browser HAR exports from NumisBids, CNG, or future providers are sensitive and must stay outside version control. If a live capture is needed to understand provider markup, extract the minimum HTML/JSON needed, remove cookies, account identifiers, bid tokens, and personal data, then commit only the sanitized fixture under `src/api/services/testdata/`.

**Pre-commit hook:**
```bash
# .githooks/pre-commit
if git diff --cached | grep -i "password\|secret\|token\|credential"; then
    echo "ERROR: Credentials detected in staged changes"
    exit 1
fi
```

### Post-Spike Credential Rotation

**Process:**
1. After spike PR merges, rotate CNG credentials
2. Remove temporary account from CNG
3. Document rotation procedure for future audits
4. CI/CD uses vault-injected credentials (GitHub Secrets / HashiCorp Vault)

---

## Common Patterns

### Sentinel Errors

**Purpose:** Explicit error types for caller to handle gracefully.

**Pattern (existing in codebase):**

```go
var (
    ErrNumisBidsAuthenticationRequired = errors.New("numisbids authentication required")
    ErrCNGAuthenticationRequired       = errors.New("cng authentication required")
    ErrRateLimitExceeded              = errors.New("rate limit exceeded")
)

// Usage
func (s *CNGAuctionService) FetchWatchlist(client *http.Client) (string, error) {
    // ...
    if isLoginPrompt(body) {
        return "", ErrCNGAuthenticationRequired  // Caller knows to re-auth
    }
}
```

### Timeout & Backoff Policy

**Pattern:**

```go
const (
    scrapeTimeout    = 10 * time.Second
    scrapeRetries    = 3
    initialBackoff   = 100 * time.Millisecond
    maxBackoff       = 5 * time.Second
)

func (s *CNGAuctionService) FetchWithRetry(url string) (string, error) {
    var lastErr error
    backoff := initialBackoff
    
    for attempt := 0; attempt < scrapeRetries; attempt++ {
        ctx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
        defer cancel()
        
        body, err := s.fetch(ctx, url)
        if err == nil {
            return body, nil
        }
        lastErr = err
        
        if attempt < scrapeRetries-1 {
            time.Sleep(backoff)
            backoff = min(backoff*2, maxBackoff)
        }
    }
    return "", lastErr
}
```

### HTTP User-Agent Header

**Pattern (consistent across services):**

```go
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
    "AppleWebKit/537.36 (KHTML, like Gecko) " +
    "Chrome/131.0.0.0 Safari/537.36"

req.Header.Set("User-Agent", userAgent)
```

### Keyword Detection (HTML 200 with "sold" keywords)

**Pattern (from `availability_service.go`):**

```go
var soldHTMLPattern = regexp.MustCompile(`(?i)>\s*sold\s*<`)

func CheckKeywords(body string) string {
    if soldHTMLPattern.MatchString(body) {
        return "unavailable"
    }
    // Check more patterns...
    return "unknown"
}
```

---

## Test Infrastructure

### Fixture Format

**Store as plain HTML (no compression/encoding):**

```html
<!-- src/api/testutil/fixtures/cng_public_lot.html -->
<!DOCTYPE html>
<html>
<head>
    <meta property="og:image" content="https://example.com/lot.jpg">
</head>
<body>
    <div class="auction">
        <h1>Lot 123 - Roman Denarius</h1>
        <p>Estimate: 150 USD</p>
        <p>Current Bid: 120 USD</p>
    </div>
</body>
</html>
```

### Fixture Updates

**When external site schema changes:**

1. Fetch new HTML
2. Update fixture file in a PR
3. Reviewer compares old vs new (git diff)
4. Tests auto-pass with new fixture
5. Document schema change in PR description

### Regression Test Matrix

**Maintain a table in `docs/testing.md`:**

| Auction Source | Public Lot | Watchlist | Sold/Unavail | Rate Limit | Timeout |
|---|---|---|---|---|---|
| NumisBids | ✅ fixture | ✅ fixture | ✅ fixture | ❌ skipped | ❌ skipped |
| CNG (Spike) | ✅ fixture | TBD (Phase 2) | ✅ fixture | ✅ stub | ✅ stub |

---

## Lessons Learned

1. **Never commit production credentials.** Use environment variables or vault injection.
2. **Fixture snapshots are immutable test references.** Treat as golden data; update only via PR with changelog. Commit sanitized HTML/JSON fixtures, never raw HAR captures.
3. **Regex parsers are fragile.** Unit-test each field extraction independently; log intermediate parsing steps for debugging.
4. **Separate unit (fixtures/stubs) from integration (live) tests.** CI runs unit tests fast (milliseconds); integration tests run on-demand or nightly.
5. **Sentinel errors enable graceful degradation.** Caller can distinguish "auth failed" from "network timeout" and respond appropriately.
6. **HTTP stubs (httptest) scale better than live calls.** Test all error scenarios (404, 429, 5xx) without rate-limiting real servers.

---

## Related Skills

- `testing-gorm-many-to-many-custom-timestamps` — Test pattern for complex data relationships
- `contextual-share-cards` — Public/shareable data exposure (similar credential/auth concerns)
- `webauthn-contract-tests` — Contract-based auth testing pattern

---

**Last Updated:** 2026-06-30  
**Owner:** Brutus (QA/Tester)  
**Applies To:** NumisBids (existing), CNG Auctions (spike), future auction sources
