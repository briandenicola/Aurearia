# Auction Tracking

> Monitor NumisBids and CNG Auctions lots with provider-aware status tracking, price alerts, bid reminders, calendar links, and collection conversion.

## Overview

Track auction lots from NumisBids and CNG Auctions with status updates, price monitoring, and conversion to your collection when won. The two providers do not expose the same data: CNG Auctions supports richer hosted-auction sync and outcome automation where the provider reports the necessary signals, while NumisBids currently supports watchlist/import tracking only.

## Provider Capability Model

| Capability | CNG Auctions | NumisBids |
|---|---|---|
| Manual lot import from a lot URL | Supported | Supported |
| Credential validation | Supported | Supported |
| Watched-lot/watchlist sync | Supported | Supported |
| Current bid and hosted bid metadata | Synced where CNG exposes it | Best-effort listing data only |
| Max-bid tracking | Synced where CNG exposes absentee bid data | Manual entry required unless future NumisBids data exposes it |
| Won/lost/final outcome automation | Auto-detected where CNG reports closed-lot winner data | Manual status update required |
| Needs-attention flag after close | Supported as a reminder to verify unresolved lots | Supported and expected for unresolved lots |

NumisBids lots should be treated as tracked watchlist/import records. After the sale closes, check the provider page and update the lot status manually to **Won**, **Lost**, or **Passed**. If you won the lot, enter the winning bid before converting it to a collection coin.

## Key Features

- **Manual Lot Entry** — Paste NumisBids or CNG lot URLs to add lots
- **Watchlist Sync** — Auto-import configured NumisBids and/or CNG watched lots with one click
- **Status Workflow** — Watching → Bidding → Won/Lost/Passed, with CNG outcomes auto-detected where available and NumisBids outcomes updated manually
- **Price Alerts** — Notify when bidding crosses your threshold
- **Bid Reminders** — Get reminded X minutes before lot closes
- **Won → Collection** — Convert lots marked **Won** into collection coins
- **AI Auction Search** — Ask the agent to find similar lots
- **Filtered Views** — Filter by status and source with badge counts
- **Credential Validation** — Verify NumisBids or CNG login before saving
- **Lot Calendar** — Visual calendar of lot end dates

## Configuration

### Auction Provider Credentials
- Store NumisBids and/or CNG username/password per-user in Settings → Account
- Validated against the selected provider before saving
- Provider passwords are encrypted at rest with `AUCTION_CREDENTIAL_ENCRYPTION_KEY`; legacy plaintext values migrate lazily on next save or sync
- Status indicators: connected, error, validating
- CNG credentials enable richer watched-lot sync and outcome detection where CNG exposes the data
- NumisBids credentials enable watchlist/import tracking only; final outcome and max-bid fields remain manual today

### Provider Sources
- Existing NumisBids lots use `source = numisbids`
- CNG lots use `source = cng`
- Lot URLs are stored in provider-aware source URL fields while preserving legacy NumisBids URL compatibility
- `statusSource` records whether a terminal won/lost status came from sync (`sync`) or an explicit user override (`manual`)

### Auction Calendar
- View auction lots on monthly calendar
- Add custom events with title, date, optional URL
- Filter by date range

### Price Alerts
- Set target price and direction (above/below)
- Auto-notify when threshold crossed
- Triggered status prevents duplicate notifications

### Bid Reminders
- Configurable lead time (e.g., 15 minutes before close)
- Notifications appear in in-app inbox

## API Endpoints

Full list and all details are in the main [features.md](../features.md#auction-tracking).

See also: [Wish List](wish-list.md), [Collection Management](collection-management.md)
