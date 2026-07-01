# CNG Auctions Phase 1 Research

**Date:** 2026-07-01  
**Scope:** Credential-safe research for adding `https://auctions.cngcoins.com/` alongside NumisBids.  
**Credentials:** Temporary credentials were referenced only from local environment variables. Values were never printed, stored, or committed.

## Executive Summary

CNG Auctions is technically more favorable than the initial spike assumed for **manual lot import**: public auction and lot pages are delivered through normal HTTP and embed structured lot data in the `viewVars` JavaScript object. A Go HTTP scraper should be able to parse auction and lot detail pages without a headless browser for public import.

The remaining blocker is **authenticated watchlist sync**. The site exposes watched-lot routes and AJAX endpoints, but the attempted form login did not create an authenticated session. Before implementing watchlist sync, the credentials/login flow needs manual verification or a browser-captured request comparison.

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

All attempts returned the login page and `/ajax/refresh-me` continued returning `null`, so no authenticated session was established.

### Authenticated Route Behavior

Unauthenticated/requested after failed login:

| Route | Result |
|---|---|
| `/ajax/refresh-me` | `200`, body `null` |
| `/watched-lots` | `302` redirect |
| `/auctions/my-upcoming-bids` | `302` redirect |
| `/ajax/watching/` | `404` without authenticated/contextual route state |
| `/ajax/my-bids/` | `200` JSON-like response but not useful for watched lots while unauthenticated |

### Watchlist Sync Status

Watchlist sync is **not yet validated**. The site clearly has watched-lot concepts, including:

- `/watched-lots`
- `/ajax/watching/`
- `/ajax/watch-lot/`
- `/ajax/unwatch-lot/`
- lot-level `watch_url`

But the login/session issue must be resolved before implementing the sync path.

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

**Blocked pending login/session verification.**

Do not implement watchlist sync until one of these is true:

1. Local scripted login succeeds and `/ajax/refresh-me` returns a user object.
2. A browser-captured successful login request reveals the missing token/header/payload.
3. The feature scope is reduced to manual import only.

## Recommended Next Steps

1. Have Brian verify the same temporary credentials can log in manually at `https://auctions.cngcoins.com/login`.
2. If browser login works, capture the successful login request shape locally without exposing credentials:
   - request path
   - method
   - non-secret headers required
   - form field names
   - redirect status
   - whether reCAPTCHA or another challenge is present
3. If browser login does not work, rotate/reset the temporary CNG password and retry Phase 1 auth validation.
4. Begin implementation with manual CNG lot import first; keep watchlist sync behind a separate gate.
5. Use fixture-based tests from `.squad/skills/external-service-scraping-with-fixtures/SKILL.md`; committed fixtures should be sanitized public lot/auction HTML only unless an authenticated fixture is reviewed for personal data.

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
