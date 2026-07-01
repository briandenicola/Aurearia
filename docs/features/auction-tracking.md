# Auction Tracking

> Monitor NumisBids and CNG Auctions lots through the complete bidding lifecycle with status tracking, price alerts, bid reminders, calendar links, and collection conversion.

## Overview

Track auction lots from NumisBids and CNG Auctions with status updates, price monitoring, and conversion to your collection when won. Existing NumisBids behavior remains supported while CNG is treated as a second auction provider.

## Key Features

- **Manual Lot Entry** — Paste NumisBids or CNG lot URLs to add lots
- **Watchlist Sync** — Auto-import configured NumisBids and/or CNG watched lots with one click
- **Status Workflow** — Watching → Bidding → Won/Lost/Passed
- **Price Alerts** — Notify when bidding crosses your threshold
- **Bid Reminders** — Get reminded X minutes before lot closes
- **Won → Collection** — Auto-convert won lots to collection coins
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

### Provider Sources
- Existing NumisBids lots use `source = numisbids`
- CNG lots use `source = cng`
- Lot URLs are stored in provider-aware source URL fields while preserving legacy NumisBids URL compatibility

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
