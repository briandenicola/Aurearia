# CNG Auctions Phase 1 Research

**Date:** 2026-07-01  
**Scope:** Credential-safe research for adding `https://auctions.cngcoins.com/` alongside NumisBids.  
**Credentials:** Temporary credentials were referenced only from local environment variables. Values were never printed, stored, or committed.

## Executive Summary

CNG Auctions is technically more favorable than the initial spike assumed: public auction pages, public lot pages, and authenticated watched lots are delivered through normal HTTP and embed structured lot data in the `viewVars` JavaScript object. A Go HTTP scraper should be able to parse manual lot imports and watchlist sync without a headless browser.

The initial environment-variable login failed because `CNG_USERNAME` differed from the successful browser login captured in the HAR. Using the HAR's successful form field values in memory, the scripted login succeeded and `/watched-lots` returned the authenticated watched-lot list.

## Confirmed Public Site Structure

### Platform

- CNG uses an Auction Mobility web module.
- Brand identifier: `n4-classicalnumismaticgroup`.
- Public pages are AngularJS-rendered but include server-side `viewVars` data in the HTML response.
- Normal HTTP GETs are sufficient to retrieve public auction and lot detail data.

### Routes and Endpoints

Relevant routes exposed in `viewVars.endpoints`:

| Purpose | Route |
|---|---|
| Login page | `/login` |
| Watched lots page | `/watched-lots` |
| Watched lots AJAX | `/ajax/watching/` |
| Watch lot | `/ajax/watch-lot/` |
| Unwatch lot | `/ajax/unwatch-lot/` |
| Refresh current user/session | `/ajax/refresh-me` |
| My bids AJAX | `/ajax/my-bids/` |
| Auction lots route | `/auctions/` |
| Lot detail route | `/lots/view/{lotId}/{slug}` |
| Lot AJAX route | `/ajax/lot/` |
| Lots AJAX route | `/ajax/lots/` |

### Auction URL Pattern

Observed auction URLs:

- `/auctions/4-LO2GO8/electronic-auction-612`
- `/auctions/4-LRPKJ2/keystone-17-the-w-toliver-besson-collection`

The auction page route is `auction-lots-index-slug`.

### Lot URL Pattern

Observed lot detail URL:

- `/lots/view/4-LO4IAT/eastern-europe-imitations-of-philip-ii-of-macedon-2nd-century-bc-ar-tetradrachm-23mm-1432-g-6h-near-vf`

The lot detail route is `lot-detail-slug`.

## Structured Data Findings

### Auction Page

Public auction pages embed a paginated lot collection in:

- `viewVars.lots.result_page`
- `viewVars.lots.query_info`
- `viewVars.ajaxLotListParams`
- `viewVars.auction`

Observed page size:

- `viewVars.lots.result_page` contained 48 lot summaries.

Observed `ajaxLotListParams`:

```json
{
  "n": 48,
  "order_by": "auction_date lot_number",
  "order": "desc asc",
  "fieldset": "timed-auction absentee-bid highest-live-bid summary live-bid-timed-count highlight-header",
  "auctionId": "4-LO2GO8",
  "lotsRange": null,
  "paramsType": "server"
}
```

Useful lot summary fields:

| CNG Field | AuctionLot Mapping |
|---|---|
| `row_id` | `SourceLotID` |
| `lot_number` + `lot_number_extension` | `LotNumber` |
| `title` | `Title` |
| `estimate_low`, `estimate_high` | `Estimate` |
| `currency_code` | `Currency` |
| `starting_price` | candidate `CurrentBid` fallback |
| `_detail_url` | `SourceURL` |
| `cover_thumbnail` | `ImageURL` |
| `auction.row_id` | `SourceSaleID` |
| `auction.title` | `SaleName` |
| `auction.effective_end_time` | `AuctionEndTime` |
| `auction.currency_code` | fallback `Currency` |

### Lot Detail Page

Public lot detail pages embed the full lot in:

- `viewVars.lot`

Useful lot detail fields:

| CNG Field | AuctionLot Mapping |
|---|---|
| `row_id` | `SourceLotID` |
| `lot_number`, `lot_number_extension` | `LotNumber` |
| `title` | `Title` |
| `description` | `Description` |
| `estimate_low`, `estimate_high` | `Estimate` |
| `currency_code` | `Currency` |
| `starting_price` | candidate `CurrentBid` fallback |
| `sold_price` | hammer/sold value if present |
| `status` | status inference support |
| `_detail_url` | `SourceURL` |
| `cover_thumbnail` | `ImageURL` |
| `images[].detail_url` | richer image candidates |
| `auction.title` | `SaleName` |
| `auction.row_id` | `SourceSaleID` |
| `auction.effective_end_time` | `AuctionEndTime` |

Example public lot included two image URLs under `images[]`, a complete HTML description, estimates, starting price, currency, auction metadata, and source API URLs.

## API Observations

CNG embeds Auction Mobility API URLs such as:

- `https://production4-server.auctionmobility.com/v1/auction/{auctionId}/lots`
- `https://production4-server.auctionmobility.com/v1/auction-lot/{lotId}/`
- `https://production4-server.auctionmobility.com/v1/auction-lot/{lotId}/watch`

Unauthenticated direct API calls to the lots API returned `401 Unauthorized`, so the implementation should prefer parsing embedded `viewVars` from public pages unless an authenticated API token/session can be established later.

## Authentication and Watchlist Findings

### Login Form

The login route contains a standard server-side form:

- `POST /login`
- `username`
- `password`
- submit button rendered as `Login`

Attempts made:

1. `POST /login` with `username` and `password`
2. `POST /login` with `username`, `password`, and `Login=Login`
3. Browser-like `POST /login` with `Origin`, `Referer`, `Accept`, and URL-encoded body

The attempts using the current environment variables returned the login page and `/ajax/refresh-me` continued returning `null`. The supplied HAR showed a successful browser login with the same password but a different username value. Using the HAR's successful `username`, `password`, and `Login=Login` fields in memory:

- `POST /login` returned a final authenticated home page.
- `viewVars.me` was present.
- `/ajax/refresh-me` returned a logged-in user JSON payload.
- `/watched-lots` returned the authenticated watched-lot page.

### Authenticated Route Behavior

Unauthenticated/requested after failed login:

| Route | Result |
|---|---|
| `/ajax/refresh-me` | `200`, body `null` |
| `/watched-lots` | `302` redirect |
| `/auctions/my-upcoming-bids` | `302` redirect |
| `/ajax/watching/` | `404` without authenticated/contextual route state |
| `/ajax/my-bids/` | `200` JSON-like response but not useful for watched lots while unauthenticated |

Authenticated with the HAR login fields:

| Route | Result |
|---|---|
| `/ajax/refresh-me` | `200` JSON payload, logged-in user present |
| `/watched-lots` | `200` HTML with `viewVars`, route `watched-lots-index` |
| `/ajax/watching/` | `404` HTML with `viewVars`, not needed for the first sync implementation |
| `/auctions/my-upcoming-bids` | `200` HTML with `viewVars`, route `my-upcoming-bids` |
| `/ajax/my-bids/` | `200` JSON payload with bid-related lot signals |

### Watchlist Sync Status

Watchlist sync is **validated for the HTML route**. The implementation can fetch `/watched-lots` after login and parse watched lots from:

- `viewVars.lots.result_page`
- `viewVars.lots.query_info`

Observed authenticated watchlist:

- route: `watched-lots-index`
- watched lots on page: `7`
- total results: `7`
- page size: `48`

Each watched lot contained the same core fields as public auction lot summaries, plus `is_watched = true`.

The site also has watched-lot AJAX concepts:

- `/watched-lots`
- `/ajax/watching/`
- `/ajax/watch-lot/`
- `/ajax/unwatch-lot/`
- lot-level `watch_url`

For the first implementation, use the HTML `/watched-lots` route and treat `/ajax/watching/` as optional follow-up research.

## Go / No-Go Assessment

### Manual CNG Lot Import

**Go.**

Manual import by pasted CNG lot URL is feasible with normal Go HTTP:

1. Fetch `/lots/view/{lotId}/{slug}`.
2. Extract the balanced `viewVars = {...}` JSON object.
3. Parse with duplicate-case-tolerant JSON handling.
4. Map `viewVars.lot` into `AuctionLot`.

No headless browser is needed for the observed public lot detail path.

### CNG Auction Page Import / Search

**Likely Go.**

Auction pages expose `viewVars.lots.result_page` and pagination metadata. Full-auction import or search may be feasible by paging public routes or by reproducing the site's AJAX route. This is not required for NumisBids parity unless the product expands beyond watchlist/manual import.

### CNG Watchlist Sync

**Go.**

Authenticated watchlist sync is feasible with normal Go HTTP:

1. `GET /login` to establish initial cookies.
2. `POST /login` with form fields `username`, `password`, and `Login=Login`.
3. Verify login by checking that the redirected page or `/ajax/refresh-me` has a non-null user.
4. Fetch `/watched-lots`.
5. Extract `viewVars.lots.result_page`.
6. Map each watched lot into provider-aware `AuctionLot` records.

## Recommended Next Steps

1. Merge the provider-aware implementation branch to beta for real-account testing.
2. Use `/watched-lots` HTML parsing for watchlist sync; `/ajax/watching/` remains optional follow-up research.
3. Keep committed tests fixture-based and sanitized; do not commit HAR files or authenticated HTML containing personal data.
4. Rotate the temporary CNG password after validation is complete.

## Implementation Notes

- Parser should be based on `viewVars` extraction rather than fragile DOM selectors.
- Use source identifiers:
  - `Source = "cng"`
  - `SourceLotID = viewVars.lot.row_id`
  - `SourceSaleID = viewVars.lot.auction.row_id`
  - `SourceURL = https://auctions.cngcoins.com + viewVars.lot._detail_url`
- Public lot detail pages already provide rich image data via `images[].detail_url`.
- `description` contains HTML; implementation should sanitize or store consistently with existing NumisBids description behavior.
- `auction.effective_end_time` appears to be the best source for `AuctionEndTime` on timed sales.
- Direct Auction Mobility API calls currently return `401`; avoid relying on them unless authenticated access is solved.
- The root HAR file is sensitive and is locally excluded through `.git/info/exclude`; do not commit HAR files or credentials.

## Implementation Completion Notes

- `src/api/services/cng_auction_service.go` implements CNG login, auth verification, manual lot parsing, and paginated watched-lot sync from `viewVars.lots`.
- Existing auction endpoints remain the public API surface; CNG is selected through `source = "cng"` or inferred from `auctions.cngcoins.com` URLs.
- The frontend adds CNG account fields under Settings > Account, accepts CNG lot URLs in the auction import modal, syncs configured watchlists, and labels CNG lots in auction/card/detail/calendar views.
- Password storage intentionally matches the existing NumisBids baseline for MVP parity; encryption/secrets hardening should be handled as a shared auction-credential follow-up.
