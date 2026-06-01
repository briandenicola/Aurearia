# Decision: Per-Coin Metadata Health Endpoint

**Agent:** Cassius (Backend Developer)  
**Date:** 2026-06-01  
**Commit:** 5bd36e9  
**Status:** Shipped

## Problem

The Metadata Health subpage on Coin Detail (`/coin/:id/health`) always showed "No health data available for this coin yet." even for existing coins with complete metadata. A screenshot from Brian showed a real coin (Alexios III Angelus Komneus) displaying the empty-state message instead of its health score and missing-items checklist.

Root cause: The frontend called `getCoinHealthList({ page: 1, limit: 1000 })` (a paginated endpoint) and then filtered client-side for `c.coinId === coinId`. If the collection had more than 1000 coins or the target coin wasn't on that page, the filter found nothing → "No health data available."

This approach was fundamentally fragile and inefficient (fetching ALL coins just to get one coin's health).

## Solution

Added a user-scoped single-coin health endpoint: `GET /coins/:id/health` (protected group, JWT required).

### Backend Implementation

**Repository (`repository/health_repository.go`):**
- `GetSingleEligibleCoin(coinID, userID uint) (*EligibleCoinRow, error)` — fetches one coin's health data using the `ActiveCollection(userID)` scope (non-wishlist, non-sold, user-owned), same SELECT clause as the list query (includes subqueries for `image_count`, `primary_image_count`).

**Service (`services/health_service.go`):**
- `GetCoinHealth(coinID, userID uint) (*CoinHealthItem, error)` — reuses ALL existing scoring logic:
  - `scoreCoinMetadata(row)` — 7 fields (denomination, ruler, era, mint, category, material, grade), 0-100
  - `scoreCoinImages(row)` — image_count: 0=0, 1=50, ≥2=100
  - `scoreCoinValuationFreshness(row)` — current_value + purchase_date age: ≤30d=100, ≤90d=80, ≤180d=60, ≤365d=35, >365d=0
  - `scoreCoinAICoverage(row)` — ai_analysis, obverse_analysis, reverse_analysis: 0=0, 1=33, 2=66, 3=100
  - `computeWeightedScore(metadata, image, valuation, ai)` — weighted average (metadata 40%, image 20%, valuation 20%, AI 20%)
  - `generateCoinChecklist(row)` — missing-items checklist (dimension, label, severity, actionHint)
  - `extractQuickActions(checklist)` — unique action hints for quick-fix buttons
- Returns the same `CoinHealthItem` shape the list endpoint uses (coinId, title, score, grade, dimensions, missingItems, quickActions).

**Handler (`handlers/health.go`):**
- `GetCoinHealth(c *gin.Context)` — thin handler:
  - Extracts `userID` from JWT context
  - Parses `coinID` from URL param (validates integer)
  - Calls `healthSvc.GetCoinHealth(coinID, userID)`
  - Returns 404 "Coin not found or not in active collection" if GORM returns `ErrRecordNotFound` (coin doesn't exist, is wishlist/sold, or isn't the user's)
  - Returns 200 with `CoinHealthItem` JSON
- Swagger annotation: `@Summary Get metadata health for a single coin`, `@Security BearerAuth`, `@Param id path int true "Coin ID"`, `@Success 200 {object} services.CoinHealthItem`

**Route Wiring (`main.go`):**
- `protected.GET("/coins/:id/health", healthHandler.GetCoinHealth)` — placed after `GET /coins/health` (list) to avoid route collision

### Frontend Implementation

**API Client (`src/web/src/api/client.ts`):**
- Added `getCoinHealth(coinId: number)` function: `api.get<CoinHealthItem>(\`/coins/${coinId}/health\`)`
- Added `CoinHealthItem` to the types import list (was exported from `@/types` but missing from the import)

**Coin Detail Health Page (`src/web/src/pages/CoinDetailHealthPage.vue`):**
- Replaced `getCoinHealthList({ page: 1, limit: 1000 })` + client-side filter with direct `getCoinHealth(coinId)` call
- Same loading/error/empty-state logic (only shows empty state when the API genuinely returns null, which for an existing owned coin should never happen since health is computed)
- No changes to `CoinHealthChecklist.vue` component (already expects `score`, `grade`, `missingItems` props)

## Architecture Compliance

- **Principle I (Layered Architecture):** Handler → Service → Repository → Database. Health computation logic stays in service layer, repository encapsulates GORM query.
- **Principle VII (Schema-Driven Contracts):** Swagger annotation on handler, OpenAPI artifacts regenerated.
- **Principle XI (Security Hardening):** User ownership validated via `ActiveCollection(userID)` scope; returns 404 (not 403) if coin isn't found/owned to avoid leaking existence.

## Key Insights

1. **Health is COMPUTED, not stored:** Every active collection coin has a score/grade/checklist (even if score=0). The data is derived from coin fields on-the-fly, so the endpoint never returns "no data" for an existing owned coin.
2. **Scope reuse:** `ActiveCollection(userID)` scope (`is_wishlist=false AND is_sold=false AND user_id=userID`) is the canonical filter for all health queries. Reusing it ensures consistent ownership validation.
3. **Scoring logic reuse:** The single-coin endpoint calls the exact same scoring functions (`scoreCoinMetadata`, `scoreCoinImages`, etc.) that the list endpoint uses. No logic duplication, no drift risk.
4. **Empty-state semantics:** The "No health data available" message should only show for wishlist/sold coins (which are explicitly excluded by the scope). For active collection coins, there is always a score.

## Verification

- Backend: `go build ./...`, `go vet ./...`, `go test ./...` — all pass including `architecture_test.go` (TestNoDirectDatabaseImports)
- Frontend: `npm run build` — type-check + vite build pass
- Pre-push hook: OpenAPI artifacts regenerated (`task openapi`), committed with `docs.go`, `swagger.json`, `swagger.yaml`, `docs/openapi.json`

## Related Work

- Aurelia is concurrently fixing a SEPARATE navigation bug touching `src/web/src/router/index.ts`, the Coin Detail page's back button, and the Coin Edit page. This fix deliberately avoided those files to prevent merge conflicts.
- If the health subpage still shows empty state after this fix, the coin is either wishlist/sold (intentional behavior) or there's a different bug (e.g., routing, component lifecycle). The API now reliably returns health data for all active collection coins.

## Future Considerations

- Consider adding per-coin health to the main coin detail response (preload health data when fetching `GET /coins/:id`) to avoid an extra round-trip. Current implementation is acceptable (one extra call per health subpage view) but could be optimized if the health subpage becomes a primary navigation target.
- If the collection grows to 10,000+ coins, the `getCoinHealthList` endpoint's pagination logic (page/limit) will be essential. The new per-coin endpoint bypasses that concern but doesn't replace the list endpoint (which powers the standalone Health List view).
