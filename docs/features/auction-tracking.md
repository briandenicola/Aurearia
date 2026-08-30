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
- **New Lot Notifications** — One batched alert when the background sync starts tracking lots you watched or bid on at the provider
- **Watch Bid Digest** — A scheduled digest of active watched/bidding lots grouped by sale, each showing how its current high bid moved since the previous digest, with its lot number linked to the auction site
- **Now Bidding Alerts** — A separate batched alert when the sync sees a bid of yours appear on a lot you were only watching
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
- The notification names the lot and its linked lot number, the sale, your target, and the current high bid against that target
- Triggered status prevents duplicate notifications

### Bid Reminders
- Configurable lead time (e.g., 15 minutes before close)
- Notifications appear in in-app inbox

### New Lot Notifications
- Fired by the background watchlist sync when it starts tracking lots that were not tracked before — lots you added to a watchlist, or placed a bid on, at NumisBids or CNG
- All lots found in one sync run are batched into a single notification, never one per lot
- The Pushover push is rich HTML and names each lot's coin name, auction house and sale, lot number, whether it is being watched or bid on, and links straight to the lot in the app
- Deep links require **Public App URL** in Admin → Settings; without it the notification still lists the lots, just without links
- Lots that first appear already closed (passed/won/lost) do not notify, and re-syncing an unchanged watchlist stays silent
- The manual **Sync Watchlists** button does not notify — it reports its results on screen

### Now Bidding Alerts
- Fired when the background sync sees a lot you were only watching move to **Bidding** — i.e. the provider now reports a bid of yours on it, usually because you bid on their site
- Batched the same way: one alert per sync run covering every lot that moved, whichever provider it came from
- Each lot carries the current high bid and your max bid alongside the coin name, auction, lot number, and app link
- CNG only in practice: CNG exposes your absentee (max) bid, NumisBids exposes no bid data, so NumisBids lots never move to Bidding automatically
- A lot that stays in Bidding across later syncs does not re-alert

## API Endpoints

Full list and all details are in the main [features.md](../features.md#auction-tracking).

See also: [Wish List](wish-list.md), [Collection Management](collection-management.md)
