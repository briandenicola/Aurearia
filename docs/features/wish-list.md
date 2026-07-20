# Wish List

> Track coins you'd like to acquire, search for real listings via AI, verify availability, and convert purchases to your collection.

## Overview

The Wish List helps collectors systematically track and acquire coins. It integrates AI-powered dealer discovery, automatic availability checking, and seamless conversion to owned coins when acquired.

## Core Functionality

### Creating Wishlist Coins

#### Manual Entry
1. Create a new coin and toggle the **Wishlist** flag
2. Coin appears in the Wish List view instead of main collection
3. Add images, notes, and target price information

#### AI Discovery
1. Navigate to **Wish List → Find Coins**
2. Open the AI agent chat drawer
3. Describe coins you're looking for (e.g., "Roman silver denarii of Julius Caesar under $500")
4. Agent searches dealer sites and returns structured suggestions
5. Each result shows: image, metadata, estimated price, source link
6. Click **Add to Wishlist** to import with one click

#### Coin Lookup
1. Navigate to **Wish List → Lookup Coin**
2. Take or upload photos of a coin or certification slab
3. Review extracted details, NGC verification links, or Numista fallback matches
4. Click **Add to Wishlist** to create a wishlist coin with captured photos and structured references

### Viewing Your Wish List

- **Dedicated Gallery** — Separate view from main collection
- **Same Filters/Sort** — Full-text search, category filtering, sorting by price/date
- **Status Indicators** — Show if coin is actively listed, unavailable, or unknown
- **Price Tracking** — Note current asking prices and compare over time
- **Lookup Saves** — Coins saved from Coin Lookup include captured photos and generated NGC or Numista references

## Availability Checking

### Manual Availability Check

1. Click **Check Availability** on the Wish List page
2. System visits each coin's source URL
3. Uses HTTP status codes and keyword heuristics (e.g., "sold", "buy now" buttons)
4. Returns status per coin: Available, Unavailable, or Unknown

### Check Results

**Summary Banner** shows:
- Available count (green)
- Unavailable count (red)
- Unknown count (gray)

**Per-Card Indicators:**
- ✅ **Green dot** — Listing is active and available
- ❌ **Red "Unavailable"** — Listing is sold or removed
- ⚠️ **Amber dot** — Status unknown; AI agent may provide additional analysis

### Automatic Availability Checks

**Admin Configuration** (`Admin → Availability Checks`):
- **Enable/Disable** — Turn automatic checks on/off
- **Schedule** — Daily start time (e.g., 2:00 AM)
- **Repeat Interval** — Minutes between checks (e.g., every 120 minutes)
- **Run History** — View past checks with per-coin drill-down
  - URL checked
  - Status (Available/Unavailable/Unknown)
  - Reason (HTTP code, keyword match, AI analysis)
  - Whether AI agent was used for ambiguous results

### Dismissing Results

- Click **Dismiss** on unavailable coins to clear the status indicator
- Dismissed coins remain in wishlist but status is hidden

## Wishlist Search Alerts

Wishlist Search Alerts are saved acquisition searches for recurring discovery:

1. Open **Wish List → Search Alerts**.
2. Create an alert with a query, optional category/material filters, optional price range, optional source URLs, and a cadence.
3. Choose **manual** for alerts that should only run when you click **Run Now**, or choose daily/weekly/monthly for scheduled discovery.
4. Review each run's candidates, dismiss poor matches, and convert accepted candidates into wishlist coins.

Alert runs are persisted with status, trigger type, candidate counts, and run details. Run Now returns immediately and processes in the background, so the page polls run status instead of blocking while the agent searches. Candidate review filters out sold/unavailable listings and price-range mismatches before candidates are saved.

## Purchase Workflow

### Move to Collection

1. Find the coin you've acquired in your Wish List
2. Click **Purchase** or **Move to Collection**
3. A styled modal prompts for:
   - **Purchase Price** — Amount paid
   - **Purchase Date** — When acquired
   - Optional: **Dealer/Source**

4. Click **Confirm**
5. Coin moves to main collection with purchase data
6. Edit page opens to add images, detailed notes, and provenance

## AI Coin Search Agent

### Using the Agent

1. Click **Find Coins** from the Wish List page
2. Chat drawer opens with AI agent
3. Describe what you're looking for:
   - "Roman silver coins of Julius Caesar"
   - "Greek silver tetradrachms from Athens"
   - "Byzantine gold coins from the 6th century"

### Agent Workflow

The agent uses a multi-team pipeline:

1. **Search Team** — Searches dealer sites, auction listings, and catalogs
2. **Fetch Team** — Retrieves detailed information (images, specs, prices)
3. **Format Team** — Structures results with coin metadata
4. **Candidate References** — Includes structured catalog references (RIC, RPC, Numista IDs)

### Agent Responses

Each result includes:
- **Image** — Fetched from listing (or auto-extracted via og:image meta tag)
- **Name & Metadata** — Denomination, ruler, era, category
- **Material** — Gold, silver, bronze, etc.
- **Estimated Price** — Dealer asking price or market estimate
- **Source Links** — Direct link to the listing
- **Catalog References** — Structured references (catalog, volume, number, URI)
- **Add to Wishlist** — One-click import with all metadata

### Streaming Responses

- Responses stream in real-time via Server-Sent Events (SSE)
- Progressive text display with status indicators
- See which agent team is active (searching, fetching, formatting)

## Auto-Image Extraction

When importing a wishlist coin from search results:

1. System attempts to extract primary image using `og:image` meta tag
2. Falls back to agent-provided image URL if meta tag scraping fails
3. Image is downloaded and attached to the coin
4. Falls back to manual entry if auto-extraction fails

### Manual Image Entry

If auto-extraction misses an image:
1. Go to the coin detail page
2. Click **Paste Image URL** in the Images section
3. Provide the image URL
4. System downloads and attaches the image

## Saved Conversations

- **Save Chat** — Save search conversations for later reference
- **Access Saved Chats** — `Settings → Conversations`
- **Reopen Chats** — Click to continue previous conversation
- **Delete Chats** — Remove conversations you no longer need

## Wish List Statistics

- **Count** — Total coins in wish list
- **Total Estimated Cost** — Sum of estimated prices (if available)
- **Availability Status** — % available / unavailable / unknown
- **Most Expensive** — Highest-priced wishlist coin

## Configuration

### User Preferences
- None — Wish List uses same settings as main collection (view, sort, filters)

### Admin Settings (`Admin → AI Configuration`)
- **AI Provider** — Choose Anthropic Claude or Ollama
- **Web Search** — Anthropic has built-in web search; Ollama requires SearXNG
- **Search Agent Prompt** — Customize the agent's system prompt

## API Endpoints

```
GET    /api/coins?status=wishlist   # List wishlist coins
POST   /api/coins                   # Create wishlist coin with status
PUT    /api/coins/:id               # Update wishlist coin details
DELETE /api/coins/:id               # Remove wishlist coin
POST   /api/coins/:id/purchase      # Move to collection with purchase info

POST   /api/wishlist/check-availability # Trigger availability check
PUT    /api/coins/:id/listing-status    # Update/dismiss listing status
GET    /admin/availability-runs         # Admin: view check history
GET    /admin/availability-runs/:id     # Admin: view details of one run

GET    /api/wishlist/search-alerts      # List saved search alerts
POST   /api/wishlist/search-alerts      # Create a saved search alert
PUT    /api/wishlist/search-alerts/:id  # Update a saved search alert
POST   /api/wishlist/search-alerts/:id/run # Queue a manual alert run
GET    /api/wishlist/search-alerts/:id/runs # List alert run history
```

## Related Features

- [Collection Management](collection-management.md) — Browse owned coins
- [Coin Details](coin-details.md) — Full metadata for each coin
- [AI Coin Search Agent](ai-search-agent.md) — Deep dive on agent capabilities
- [Auction Tracking](auction-tracking.md) — Track NumisBids and CNG Auctions lots instead
- [Collection Statistics](statistics.md) — Portfolio analytics

## See Also

- [AI Coin Search Agent](ai-search-agent.md) — How the search agent works
- [Auction Tracking](auction-tracking.md) — Similar feature for auction lots
