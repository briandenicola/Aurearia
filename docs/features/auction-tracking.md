# Auction Tracking

> Monitor NumisBids auction lots through the complete bidding lifecycle with status tracking, price alerts, and bid reminders.

## Overview

Track auction lots from NumisBids with real-time status updates, price monitoring, and conversion to your collection when won.

## Key Features

- **Manual Lot Entry** — Paste NumisBids URLs to add lots
- **NumisBids Watchlist Sync** — Auto-import your watchlist with one click
- **Status Workflow** — Watching → Bidding → Won/Lost/Passed
- **Price Alerts** — Notify when bidding crosses your threshold
- **Bid Reminders** — Get reminded X minutes before lot closes
- **Won → Collection** — Auto-convert won lots to collection coins
- **AI Auction Search** — Ask the agent to find similar lots
- **Filtered Views** — Filter by status with badge counts
- **Credential Validation** — Verify NumisBids login before saving
- **Lot Calendar** — Visual calendar of lot end dates

## Configuration

### NumisBids Credentials
- Store username/password per-user in Settings
- Validated against NumisBids before saving
- Status indicators: connected, error, validating

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
