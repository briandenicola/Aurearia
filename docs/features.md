# Features

> **This page has been reorganized!** Detailed feature documentation has moved to individual files in the [`docs/features/`](features/) directory for better discoverability and maintainability.

**👉 [Go to Feature Index →](features/INDEX.md)** to browse features by category.

---

## Quick Feature Reference

### 🏛️ Core Collection
- [**Collection Management**](features/collection-management.md) — Browse, filter, search with swipe/grid views
- [**Coin Details**](features/coin-details.md) — Rich metadata, images, provenance, journals, references
- [**Coin of the Day**](features/coin-of-the-day.md) — Daily featured coin scheduler

### 🎯 Discovery & Acquisition
- [**Coin Lookup**](features/coin-lookup.md) — Photo-based lookup for NGC Ancients slabs and Numista matches
- [**Wish List**](features/wish-list.md) — Track coins with AI search and availability checking
- [**Auction Tracking**](features/auction-tracking.md) — Monitor NumisBids and CNG Auctions lots through bidding lifecycle
- [**Sold Coins**](features/sold-coins.md) — Track sales with profit/loss analysis

### 🤖 AI Features
- [**AI Coin Analysis**](features/ai-analysis.md) — Vision model analysis of obverse/reverse photos
- [**AI Coin Search Agent**](features/ai-search-agent.md) — Chat agent for dealer discovery
- [**AI Grading**](features/ai-grading.md) — Grade estimation from photos
- [**Price Trends**](features/price-trends.md) — Market trend analysis
- [**Gap Analysis**](features/gap-analysis.md) — Collection gap suggestions
- [**Photography Guide**](features/photography-guide.md) — Photo quality feedback
- [**Similar Lots**](features/similar-lots.md) — Find matching auction listings

### 📊 Organization & Analytics
- [**Coin Sets**](features/coin-sets.md) — Themed collections with trend tracking
- [**Custom Tags**](features/custom-tags.md) — Flexible categorization
- [**Collection Statistics**](features/statistics.md) — Portfolio analytics & charts
- [**Collection Showcase**](features/collection-showcase.md) — Share curated public subsets

### 🤝 Social & Community
- [**Social Features**](features/social-features.md) — Follow, comment, rate
- [**User Profiles**](features/user-profiles.md) — Avatars, bio, privacy controls

### 🔐 Admin & Configuration
- [**Admin Settings**](features/admin-settings.md) — User management, AI config, scheduling, configurable coin properties
- [**Authentication**](authentication.md) — JWT, WebAuthn, API keys
- [**External Tool Server**](external-tool-server.md) — OpenAPI for external clients

### 📱 Mobile & Offline
- [**PWA Features**](features/pwa-features.md) — Installable app, offline read access
- [**Camera Capture**](features/camera-capture.md) — Direct device camera integration

### 🔧 Advanced
- [**Image Operations**](features/image-operations.md) — Background removal, OCR, clipping
- [**PDF Export**](features/pdf-export.md) — Insurance/provenance catalogs
- [**Bulk Operations**](features/bulk-operations.md) — Multi-select batch actions
- [**Notifications**](features/notifications.md) — In-app alerts
- [**Numista Catalog**](features/numista-integration.md) — Catalog reference integration
- [**Auction Calendar**](features/auction-calendar.md) — Visual event calendar
- [**Data Import/Export**](features/import-export.md) — JSON backup/restore

---

## Detailed Feature Documentation

### Collection Management

The main collection page supports browsing your coins with filtering, search, and sorting:

- **Card Gallery** — Responsive card grid showing each coin's primary image with name, ruler, denomination, and category. Supports filtering by category (Roman, Greek, Byzantine, Modern), tag filter, and full-text search across all fields.
- **Face Toggle** — Switch between obverse and reverse images on coin cards (grid mode).
- **Sorting** — Sort by date added, date last updated, or current value (ascending or descending).
- **Swipe / Grid Toggle** — Switch between a swipeable card carousel (ideal for mobile/PWA) and a traditional grid layout. Default view preference is configurable in Settings.
- **Pagination** — Coins are loaded with pagination for large collections.
- **Category Colors** — Each category has a distinct color accent: purple for Roman, olive for Greek, red for Byzantine, and steel blue for Modern.
- **Bulk Select Mode** — Enter select mode to multi-select coins for batch operations (tagging, status changes, export). In PWA mode, a dedicated "Select" button appears in the header.

### Wish List

Track coins you'd like to acquire with an AI-powered search agent:

- **Add to Wish List** — When creating a coin, toggle the "Wishlist" flag. The coin appears in the dedicated Wish List view instead of the main collection.
- **AI Coin Search Agent** — Click "Find Coins" to open a chat drawer powered by the AI agent service (Anthropic Claude or Ollama, configurable in Admin). Describe the coins you're looking for (e.g., "Roman silver denarii of Julius Caesar under $500") and the agent searches the web for real listings and references. Results appear as cards with images, metadata, estimated prices, and source links — each with an "Add to Wishlist" button for one-click import.
- **Purchase** — Move a wishlist coin to your main collection when you acquire it. A styled purchase modal prompts for the purchase price and date, replacing the default browser confirm dialog.
- **Wish List Gallery** — A separate page showing only wishlist items with sorting support.
- **Coin Lookup Entry Point** — Launch Coin Lookup directly from the Wish List page to photograph a coin at a show and save the result as a wishlist item.
- **Availability Check** — Click "Check Availability" on the Wish List page to verify whether listed coins are still for sale. The system visits each coin's reference URL and uses HTTP status codes plus keyword heuristics (sold indicators, buy-now buttons) to determine listing status. Ambiguous results are escalated to the AI agent (Team 6) for deeper analysis. Results show as a summary banner (available / unavailable / unknown counts) and per-card status indicators (green dot, red "Unavailable" overlay, amber dot). Unavailable coins can be dismissed to clear the status.
- **Scheduled Checks** — Admins can enable automatic availability checks with a configurable start time and repeat interval (e.g., starting at 2:00 AM, repeating every 120 minutes). Run history with per-coin drill-down is available in the Admin Availability tab.

### Sold Coins

Track coins you've sold with profit/loss visibility:

- **Sell from Detail Page** — Click "Sell" on any collection coin's detail page. A styled modal prompts for the sale price and buyer name.
- **Profit / Loss Tracking** — Sold coins display the sale price, original cost basis, and calculated profit or loss (green for profit, red for loss).
- **Sold Gallery** — A dedicated gallery page showing all sold coins with their sale history, accessible from the navigation bar.
- **Stats Integration** — Sold coins are excluded from active collection totals while their sold values are tracked separately.

## Auction Tracking

Track auction lots from NumisBids and CNG Auctions through a complete bidding lifecycle:

- **Add Lots** — Manually add lots by pasting a NumisBids or CNG Auctions URL. The app detects the provider and scrapes the lot page for title, image, estimate, auction house, and sale name.
- **Watchlist Sync** — Connect your NumisBids and/or CNG account in Settings → Account and sync configured watchlists with one click. Stored provider passwords are encrypted at rest and legacy plaintext values migrate on next use.
- **Status Workflow** — Lots progress through statuses: Watching → Bidding → Won / Lost / Passed. Status transitions are validated (e.g., only Bidding lots can be marked Won).
- **Filtered Views** — Filter the auctions page by status and source. Counts for each status appear as badges on filter buttons.
- **Won → Collection** — When a lot is marked as Won, it is automatically converted into a coin in your collection (mapping title, category, auction house, sale date, bid price) and the edit page opens so you can add details. A manual "Add to Collection" button is also available.
- **AI Auction Search** — Ask the AI agent to search supported auction sources such as NumisBids and CNG Auctions for coins matching a description. The agent (Team 5) searches, fetches lot details, and formats results.
- **Credential Validation** — NumisBids and CNG credentials are validated against the live provider before being saved. The Settings page shows connected/error/validating status indicators.

## Coin Details

Each coin can store:

- **Core fields** — Name, denomination, ruler/authority, era (`ancient`, `medieval`, `modern`), category, material, weight, diameter
- **Numismatic details** — Mint mark, obverse/reverse inscriptions, grade, RIC number, rarity rating
- **Financial data** — Purchase price, current value, acquisition date, dealer/source
- **Images** — Multiple image uploads per coin (obverse, reverse, edge, detail, full) with a gallery viewer. Supports file upload, paste-from-URL (fetched via server proxy), and direct camera capture in PWA mode.
- **Camera Capture (PWA)** — In PWA/mobile mode, a "Photo" button appears on upload sections letting you take coin photos directly with the rear camera. Available on the coin detail page and the add/edit form.
- **AI Analysis** — Markdown-formatted analysis from Ollama, stored with the coin (obverse and reverse analyzed separately)
- **Structured Catalog References** — Add and manage normalized references per coin (`catalog`, `volume`, `number`, optional invoice number, optional authority URI) directly in coin detail.
- **Activity Journal** — Timestamped log entries per coin (e.g., "cleaned", "sent to NGC for grading", "displayed at coin show"). Add and delete entries directly from the detail page.
- **Numista Catalog Lookup** — Search the Numista coin catalog directly from a coin's detail page. Results show thumbnails, title, issuer, and year range with links to the full Numista catalog entry.
- **Notes** — Free-text notes field

## AI Coin Analysis

Upload photos of a coin and click **Analyze with AI** to get an AI-powered numismatic analysis via Ollama. The analysis covers identification, obverse/reverse descriptions, inscriptions, condition assessment, historical context, and estimated market value. Obverse and reverse sides are analyzed separately with dedicated prompts. If accepted, the analysis is saved to the coin's record.

To enable AI analysis:

1. Install [Ollama](https://ollama.ai/) and pull a vision model: `ollama pull llava`
2. Start Ollama: `ollama serve`
3. Configure the Ollama URL and model in **Admin → AI Configuration**

## AI Coin Search Agent

Chat with an AI agent that searches the web for coins matching your description. Powered by the multi-agent Python service (supports Anthropic Claude with built-in web search or Ollama with SearXNG), the agent returns structured coin suggestions with names, categories, materials, price estimates, source links, and candidate catalog references. Each suggestion can be added to your wishlist with one click.

Key features:

- **Streaming Responses** — Agent replies stream in real-time via Server-Sent Events (SSE) with a progressive text display and blinking cursor, so you see results as they arrive.
- **Real-time Status Indicators** — During agent processing, status indicators show which team and step is active (searching, fetching, formatting).
- **SSE Progress Events** — The agent streams structured progress events so the frontend can display step-by-step status updates.
- **Multi-team Architecture** — The agent uses specialized teams: coin search finds listings, coin shows finds upcoming events, portfolio review analyzes your collection, coin analysis examines uploaded images, coin grading estimates grades from photos, gap analysis identifies missing coins in your collection, photo guide critiques your coin photography, price trends analyzes market direction from auction history, and similar lots finds matching active auction listings.
- **Automatic Image Extraction** — When you add a coin to your wishlist, the system automatically extracts the listing's primary image using `og:image` meta tag scraping from the source page. Falls back to the agent-provided image URL if scraping finds nothing.
- **Candidate Catalog References** — Suggestions can include structured candidate references (`catalog`, `volume`, `number`, optional invoice number, optional authority `uri`) which are carried into wishlist coin creation.
- **Paste Image URL** — If automatic extraction misses an image, you can paste an image URL directly on the coin detail page to fetch and attach it.
- **Save Conversations** — Save search conversations for later reference. Saved chats appear in the Settings → Conversations tab where you can reopen or delete them.
- **Configurable Model & Prompt** — Admins can select the Claude model from a dropdown populated from the Anthropic API, and customize the agent's system prompt.

To enable the search agent:

1. Get an API key from [console.anthropic.com](https://console.anthropic.com/)
2. Configure it in **Admin → AI Configuration → Anthropic API Key**
3. Optionally select a different model or customize the agent prompt in Admin settings

## Coin Lookup

Use **Lookup Coin** from the main menu or Wish List page when evaluating a coin at a show. Capture or upload one or more photos of the coin or slab label. The app sends the images through the configured vision provider, extracts visible label text and coin fields, and looks for NGC Ancients certification numbers.

When an NGC cert is found, the result includes the normalized cert and an official NGC Ancients verification link in the form `https://www.ngccoin.com/certlookup/{compactCert}/NGCAncients/`. The lookup returns immediately after NGC extraction instead of waiting on catalog enrichment. When no NGC cert is found, the app uses configured Numista access to search for possible catalog matches and displays links to Numista entries.

Lookup results can be saved directly to the Wish List or Collection. The save flow creates the coin first, uploads the captured photos, then adds generated NGC or Numista structured references.

## Numista Catalog Integration

Look up coins in the [Numista](https://en.numista.com/) catalog directly from any coin's detail page or as a fallback from Coin Lookup when no NGC cert is detected. The search uses coin names, denominations, rulers, and extracted fields to find matching entries, displaying thumbnails and linking to full catalog pages.

To enable Numista lookup:

1. Get a free API key at [numista.com/api](https://en.numista.com/api/) (2,000 requests/month free)
2. Configure it in **Admin → System Settings → Numista API Key**

## Collection Statistics

The **Stats** page shows:

- **Summary cards** — Total coins, total value, average value, unique rulers
- **Category breakdown** — Distribution across Roman, Greek, Byzantine, Modern, and Other
- **Material distribution** — Breakdown by gold, silver, bronze, copper, and electrum
- **Grade distribution** — Bar chart showing coin counts by grade (VF, EF, AU, etc.) with a blue gradient
- **Value over time** — SVG line chart tracking total portfolio value and total invested over time, built from automatic snapshots recorded after every coin create, update, or delete
- **Top coins by value** — Ranked list of the most valuable coins in your collection
- **Era/Region Heat Map** — SVG-based heat map showing the distribution of your collection across time periods and geographic regions, highlighting concentrations and gaps at a glance. Available via the `/stats/distribution` endpoint.

## Image Text Extraction

Upload photos of store cards, certificates, or coin holder labels and extract text via OCR powered by Ollama. Useful for quickly capturing dealer information, grades, and reference numbers when adding coins.

## Social Features

Follow other collectors and interact with their coin collections:

- **Follow / Unfollow** — Send follow requests to other public users. Requests start as `pending` until the other user accepts.
- **Accept / Block** — Review incoming follow requests and accept or block them. Blocked users cannot re-request unless explicitly unblocked.
- **Follower Gallery** — View an accepted follower's coin collection in a read-only gallery (pricing/value and AI analysis are hidden).
- **Comments & Star Ratings** — Leave comments and 1–5 star ratings on coins belonging to users you follow. Both the commenter and the coin owner can delete comments.
- **User Search & Discovery** — Search for other collectors by username. Only public users appear in search results.
- **Privacy Controls** — Toggle your profile between public and private. Setting your profile to private permanently removes all existing followers. Mark individual coins as private to hide them from followers.

For the full social API reference, see the [API Reference](api-reference.md#social).

## User Profiles

- **Avatar** — Upload a custom avatar image (stored in `uploads/avatars/`). The default is the Ed-Mar coin logo.
- **Bio** — Add a personal bio to your profile.
- **Public/Private Toggle** — Control whether your profile appears in search results and can receive follow requests.
- **Email** — Required for new registrations. Legacy users are prompted with a dismissible modal to add their email.

## User Settings

All authenticated users can access **Settings**, organized in a tabbed layout:

- **Account** — Change password (requires current password), register WebAuthn/FIDO2 passkeys for passwordless login, manage avatar and profile settings.
- **Appearance** — Choose from five color themes: Dark (default), Light, British Museum (sandstone neutrals with bronze-green accents), Louvre (warm cream stone with Napoleon III gold), or Capitoline (terracotta warmth with aged copper-bronze). Set time zone, choose default gallery view (swipe or grid), and default sort order. Preferences persist across sessions.
- **Data** — Export your entire collection as JSON, import coins from a JSON file, download an insurance/provenance PDF catalog of your collection (with photos, grades, provenance, valuations, and structured references), and manage API keys for programmatic access. See the [Getting Started Guide](getting-started.md#import--export) for the full import file format.
- **Conversations** — View, reopen, or delete saved AI search agent conversations.
- **Tags** — Create, rename, and delete custom tags with color selection. Tags created here can be attached to any coin in your collection.
- **Help** — Beginner's guide to ancient coin collecting.

## Admin Settings

The first registered user is the admin. Admins can access **Admin** to manage:

- **Users** — View all registered users, delete accounts, and reset passwords.
- **AI Configuration** — Select your AI Provider: Anthropic (recommended) or Ollama. Configure Ollama (URL, vision model, timeout, analysis prompts), Anthropic (API key, model dropdown, editable agent prompt for the search agent), and SearXNG URL for Ollama web search.
- **System** — Set the application log level (trace, debug, info, warn, error) and configure the Numista API key for catalog lookups.
- **Coin Properties** — Configure newline-delimited Category and Era option lists used by coin forms and lookup saves.
- **Logs** — View real-time application logs with level filtering, auto-refresh, and log export.
- **Availability Checks** — Enable/disable automatic wishlist availability checking, configure the daily start time and repeat interval, and view paginated run history with per-coin drill-down results (URL, status, reason, HTTP code, whether the AI agent was used).
- **Valuation Runs** — Scheduled collection valuation using the AI agent. Configurable interval (default: 7 days), start time (default: 03:00), and max coins per run (default: 50). View run history with per-coin results. Trigger or cancel runs manually.

## Authentication

The application supports multiple authentication methods for flexibility and security:

- **JWT + Refresh Tokens** — 15-minute access tokens with 30-day rolling refresh tokens. The frontend silently refreshes expired tokens so users stay logged in.
- **WebAuthn / Passkeys** — FIDO2 biometric authentication (Face ID, Touch ID, fingerprint). Register a passkey in Settings → Account, then use it to log in without a password. Requires HTTPS in production.
- **API Keys** — Generate API keys in Settings → Data for programmatic access. Keys use the `X-API-Key` header and are checked before JWT.

For a detailed walkthrough of each auth method, see the [Authentication Guide](authentication.md).

## External Tool Server

The External Tool Server exposes your collection to external AI clients (OpenWebUI, LibreChat, n8n, MCP-compatible clients) over a versioned HTTP API. External clients can query your collection, analyze statistics, and optionally propose updates through a secure two-phase commit flow.

**Key Features:**

- **Default-Off** — Admin must enable in System Settings. Disabled by default for security.
- **Scoped API Keys** — Each key has `read` (default) or `read,write` capability. Write must be explicitly chosen at key creation.
- **Two-Phase Writes** — External writes require `propose_update` (returns preview + token) followed by `commit_update` (with explicit confirm). No auto-writes.
- **Journaling** — External commits write audit entries with source `external_tool_server`, API key name, and changed fields.
- **Tenant Isolation** — Every operation is scoped to the API key owner. No cross-user access.
- **Per-Key Rate Limiting** — Stricter rate limits (50 req/min) prevent abuse from external clients.
- **Field Allowlist** — External writes restricted to `grade`, `currentValue`, `notes`, `tags`, `referenceText`, `referenceUrl`, `references`. Identity fields are rejected.
- **OpenAPI-First** — Served OpenAPI document at `/api/v1/tools/openapi.json` for client auto-import.
- **MCP Compatible** — Wrap the OpenAPI spec with `mcpo` to expose tools to MCP clients like Claude Desktop.

**Available Tools:**

- Read: `search_my_collection`, `get_coin`, `collection_summary`, `top_coins_by_value`
- Write: `propose_update`, `commit_update`

For setup instructions, security model, and client integration guides, see the [External Tool Server Guide](external-tool-server.md). The guide is organized by audience: administrators (enabling/managing the server), users (creating API keys and connecting clients), and developers (API reference and error handling).

## PWA Features

The app is a Progressive Web App installable on iOS, Android, and desktop:

- **Swipe Gallery** — Touch-based card carousel for browsing coins on mobile, with position persistence across navigation.
- **Pull-to-Refresh** — Pull down from the top of the gallery to refresh the coin list.
- **Camera Capture** — Take coin photos directly with the device camera from upload sections.
- **Hamburger Menu** — Gallery controls (filters, sort, view) in a compact popover menu.
- **Background Removal** — Remove image backgrounds in-place on the coin detail page using client-side ML.

For installation instructions and a full feature guide, see the [PWA Guide](pwa-guide.md).

## AI Grading Assistant

Upload coin photos and ask the AI agent to estimate the grade (VF, EF, AU, etc.) with reasoning and a confidence score. The grading assistant is accessible through the AI agent chat — send photos and ask "Grade this coin."

- **Vision-Based Grading** — The agent examines surface wear, strike quality, and detail preservation to estimate a Sheldon-scale grade.
- **Reasoning and Confidence** — Each grade estimate includes an explanation of the factors considered and a confidence level.
- **Agent Team 7** — Powered by the `coin_grading.py` agent team in the Python agent service.

## Price Trend Analysis

Ask the AI agent about price trends for a specific coin type. The agent searches NumisBids and other auction sources for historical sale data, analyzes results over time, and reports market direction.

- **Historical Auction Data** — Searches for past auction results of similar coins to build a price history.
- **Market Direction** — Identifies whether prices are trending up, down, or stable for a given coin type.
- **Agent Team 8** — Powered by the `price_trends.py` agent team in the Python agent service.

## Collection Gap Analysis

The AI agent reviews your collection and suggests what is missing for completeness. It analyzes your holdings by category, era, ruler, and material to find gaps, then suggests specific coins to fill them with estimated prices.

- **Portfolio-Aware** — Uses the portfolio summary from the Go API to understand your current holdings.
- **Targeted Suggestions** — Recommends specific coins with estimated market prices to fill identified gaps.
- **Agent Team 9** — Powered by the `gap_analysis.py` agent team in the Python agent service.

## Coin Photography Guide

Upload coin photos and ask the AI agent for photography tips. The agent evaluates your photos and provides actionable advice on lighting, focus, background, and positioning.

- **Photo Quality Analysis** — The agent critiques uploaded images and identifies areas for improvement.
- **Practical Tips** — Suggestions cover lighting angles, background choices, camera settings, and coin positioning.
- **Agent Team 10** — Powered by the `photo_guide.py` agent team in the Python agent service.

## Similar Lot Finder

Find active auction lots similar to coins in your collection. The agent searches NumisBids for matching lots and ranks results by relevance.

- **Collection-Based Search** — Describe a coin or reference one in your collection to find similar active listings.
- **Relevance Ranking** — Results are sorted by how closely they match the target coin.
- **Agent Team 11** — Powered by the `similar_lots.py` agent team in the Python agent service.

## Collection Showcase

Create curated subsets of your collection to share publicly. Each showcase gets a unique shareable URL.

- **Create and Manage** — Create showcases from your collection, each with a title, description, and slug for the public URL (`/s/:slug`).
- **Two-Column Coin Picker** — Add or remove coins from a showcase using a side-by-side picker interface.
- **Published / Draft Toggle** — Control visibility of each showcase. Only published showcases are accessible via the public URL.
- **Public View** — The public showcase page is read-only and requires no authentication. It displays coin images and basic metadata.
- **API Endpoints** — `GET/POST /showcases`, `GET/PUT/DELETE /showcases/:id`, `PUT /showcases/:id/coins`. Public endpoint: `GET /api/showcase/:slug` (no auth required).

## Auction Calendar

A monthly calendar view showing auction lot end dates and custom events.

- **Calendar View** — Visual month-by-month calendar with lot indicators showing which days have scheduled auctions.
- **Date Range Filter** — Filter events by date range using query parameters.
- **Custom Events** — Add custom auction events with a title, description, date, and optional URL (e.g., for auction house registration links).
- **Lot Indicators** — Days with tracked auction lots display visual indicators on the calendar.
- **API Endpoints** — `GET /calendar`, `POST/PUT/DELETE /calendar/events`.

## Price Alerts

Set target prices on watched auction lots and get notified when bidding crosses your threshold.

- **Target Price** — Set a price threshold and direction (above or below) for any tracked auction lot.
- **Triggered Notifications** — When bidding crosses the threshold, a notification is generated in your notification inbox.
- **API Endpoints** — `GET/POST /alerts`, `DELETE /alerts/:id`. Each alert tracks the lot ID, target price, direction, and triggered status.

## Bid Sniping Reminders

Set reminders for a configurable number of minutes before a watched lot's auction session closes.

- **Configurable Lead Time** — Choose how many minutes before the auction close you want to be reminded.
- **Notification Integration** — Reminders appear in the in-app notification inbox when triggered.
- **API Endpoints** — `GET/POST /reminders`, `DELETE /reminders/:id`. Each reminder tracks the lot ID, remind-before minutes, and triggered status.

## Custom Tags and Labels

Create user-defined categories beyond the built-in fields. Tags provide flexible, personal organization for your collection.

- **Create and Manage** — Create, rename, and delete custom tags with color selection in **Settings → Tags**.
- **Attach to Coins** — Attach or detach tags on any coin in your collection. Tags display on coin cards and the detail page.
- **API Endpoints** — `GET/POST /tags`, `PUT/DELETE /tags/:id`, `POST /coins/:id/tags`, `DELETE /coins/:id/tags/:tagId`.

## Bulk Operations

Multi-select coins for batch operations across your collection.

- **Select Mode** — Enter select mode, tap coins to select them, then apply a bulk action.
- **Supported Actions** — Batch status changes, tagging, and export.
- **PWA Integration** — In PWA mode, a dedicated **Select** button appears in the header. The agent FAB is automatically hidden during bulk select mode.
- **API Endpoint** — `POST /coins/bulk`.

## Notifications

An in-app notification system with an unread badge count in the navigation bar.

- **Social Notifications** — Follow requests, comments, and star ratings generate notifications.
- **Alert and Reminder Triggers** — Price alert and bid sniping reminder triggers create notifications.
- **Mark as Read** — Mark individual notifications or all at once as read.
- **Delete** — Remove notifications you no longer need.
- **Real-Time Badge** — The unread count badge polls automatically using a shared composable.
- **API Endpoints** — `GET /notifications`, `GET /notifications/unread-count`, `PUT /notifications/:id/read`, `PUT /notifications/read-all`, `DELETE /notifications/:id`.

## Insurance / Provenance PDF Export

Generate a downloadable PDF catalog of your collection with photos, grades, provenance details, valuations, era, and structured catalog references. Useful for insurance documentation or sharing with dealers.

- **Download from Settings** — Available in **Settings → Data** as a one-click download.
- **API Endpoint** — `GET /user/export/catalog`.

## Configuration

### Environment Variables

| Variable          | Default                         | Description |
| ----------------- | ------------------------------- | ----------- |
| `JWT_SECRET`      | *(auto-generated for dev)*      | JWT signing key (`openssl rand -base64 48`) |
| `DB_PATH`         | `./ancientcoins.db`             | SQLite database file path |
| `PORT`            | `8080`                          | HTTP server port |
| `UPLOAD_DIR`      | `./uploads`                     | Directory for uploaded coin images |
| `WEBAUTHN_RP_ID`  | `localhost`                     | Relying Party ID for FIDO2/WebAuthn |
| `WEBAUTHN_ORIGIN` | `http://localhost:8080`         | Origin URL for WebAuthn |
| `AGENT_SERVICE_URL` | `http://agent:8081`           | Python agent service URL |
| `AGENT_INTERNAL_SERVICE_TOKEN` | —                 | Shared API → agent credential required by hardened agent endpoints |
| `AGENT_LOG_LEVEL` | `INFO`                          | Python agent log level |
| `CORS_ORIGINS`    | *(WebAuthn origin + localhost)* | Comma-separated allowed CORS origins |

### Admin-Managed Settings

These are configured in the Admin UI (not environment variables) and stored in the database:

| Setting              | Description |
| -------------------- | ----------- |
| `AIProvider`          | Explicit provider choice: `anthropic` or `ollama` (must be set before agent features work) |
| `AnthropicAPIKey`     | API key for Claude models |
| `AnthropicModel`      | Claude model to use (default: `claude-sonnet-4-20250514`) |
| `OllamaURL`          | Ollama server URL (default: `http://localhost:11434`) |
| `OllamaModel`        | Vision model name (default: `llava`) |
| `OllamaTimeout`      | AI request timeout in seconds (default: `300`) |
| `SearXNGURL`         | SearXNG search engine URL (required for Ollama web search) |
| `NumistaAPIKey`       | Numista catalog API key for coin lookups |
| `CoinSearchPrompt`   | System prompt for the AI coin search agent |
| `CoinShowsPrompt`    | System prompt for the coin shows agent |
| `ValuationPrompt`    | System prompt for the value estimator |
| `ObversePrompt`       | Custom prompt for obverse image analysis |
| `ReversePrompt`       | Custom prompt for reverse image analysis |
| `TextExtractionPrompt`| Custom prompt for OCR text extraction |
| `LogLevel`            | Application log level (trace/debug/info/warn/error) |
