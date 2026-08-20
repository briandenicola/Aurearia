# Features Index

Aurearia provides a comprehensive set of features for managing a personal coin collection, with deep support for ancient and historical coins. This directory contains detailed documentation for each major feature area.

## Visual Assets

Approved product screenshots live in `docs/assets/screenshots/`. Generated captures first land in the
git-ignored `src/web/artifacts/screenshots/` (see `npm run screenshots:beta` in `src/web/README.md`) for
Brian's review; only approved images are copied/moved into `docs/assets/screenshots/` for use in docs.

| File | Description |
|---|---|
| `desktop-01-collection-gallery.png` | Desktop collection gallery tour capture |
| `desktop-02-coin-detail.png` | Desktop coin detail tour capture |
| `desktop-03-wishlist.png` | Desktop wishlist tour capture |
| `desktop-04-stats-overview.png` | Desktop stats overview tour capture |
| `desktop-05-deep-analysis-entry.png` | Desktop Deep Analysis entry-point tour capture |
| `mobile-01-collection-gallery.png` | Mobile collection gallery tour capture |
| `mobile-02-coin-detail.png` | Mobile coin detail tour capture |
| `mobile-03-wishlist.png` | Mobile wishlist tour capture |
| `mobile-04-stats-overview.png` | Mobile stats overview tour capture |
| `mobile-05-deep-analysis-entry.png` | Mobile Deep Analysis entry-point tour capture |
| `coin-detail-justinian.png` | Coin detail page — Justinian coin |
| `collection-tray.png` | Collection tray view |
| `wishlist-detail-maximinus.png` | Wishlist detail — Maximinus coin |
| `auctions-dashboard.png` | Auctions dashboard view |
| `rulers-set-completion.png` | Rulers set/completion tray view |

## Core Collection Features

- **[Collection Management](collection-management.md)** — Create, browse, filter, search, and organize coins with rich metadata
- **[Coin Details](coin-details.md)** — Store comprehensive numismatic data, images, references, activity journals, and notes
- **[Coin of the Day](coin-of-the-day.md)** — Daily featured coin notifications to help rediscover your collection

## Discovery & Acquisition

- **[Quick Capture](../quick-capture.md)** — Mobile-first intake drafts for show-floor photos and notes before promoting to collection or wishlist
- **[Coin Lookup](coin-lookup.md)** — Photograph a coin or slab at a show, extract NGC Ancients certs, verify with NGC, and save to wish list or collection
- **[Wish List](wish-list.md)** — Track coins you'd like to acquire with AI-powered search, availability checking, and saved search alerts
- **[Auction Tracking](auction-tracking.md)** — Monitor NumisBids and CNG Auctions lots with provider-aware sync, price alerts, and reminders
- **[Sold Coins](sold-coins.md)** — Track coins you've sold with profit/loss analysis

## AI Features

- **[AI Coin Analysis](ai-analysis.md)** — Vision-model analysis of obverse/reverse photos using Anthropic Claude or Ollama
- **[Deep Analysis](deep-analysis.md)** — Resumable multi-provider identification with cited evidence and confirm-gated proposals
- **[Coin Agent](ai-search-agent.md)** — Chat with an AI agent to find coins, answer collection questions, research shows, and save useful answers to Notes
- **[AI Grading Assistant](ai-grading.md)** — Estimate coin grades from photos with reasoning and confidence scores
- **[Price Trend Analysis](price-trends.md)** — Analyze historical auction data to identify market trends
- **[Collection Gap Analysis](gap-analysis.md)** — Get AI-powered suggestions for coins missing from your collection
- **[Coin Photography Guide](photography-guide.md)** — Receive critiques and tips for improving your coin photography
- **[Similar Lot Finder](similar-lots.md)** — Find active auction listings similar to coins in your collection

## Organization & Analytics

- **[Coin Sets](coin-sets.md)** — Organize coins into standard, goal, smart, and human-reviewed Agentic sets with trend tracking and tray presentation
- **[Custom Tags](custom-tags.md)** — Create flexible custom categories for organizing your collection
- **[Collection Statistics](statistics.md)** — View analytics including portfolio value, distributions, trends, health, maps, investment breakdown, and emperor tracking
- **[Collection Time Machine](time-machine.md)** — Replay the collection as it stood on any past date, reconstructed from purchase dates and recorded valuation history
- **[Collection Showcase](collection-showcase.md)** — Create and share curated public coin subsets with shareable URLs

## Social & Community

- **[Social Features](social-features.md)** — Follow collectors, leave comments, and rate coins in shared collections
- **[User Profiles](user-profiles.md)** — Customize your public profile with avatar, bio, and privacy settings

## Administration & Configuration

- **[Authentication](../authentication.md)** — JWT tokens, WebAuthn passkeys, and API keys
- **[OIDC Setup](../oidc-setup.md)** — Configure Entra ID, Pocket ID, or generic OIDC login and account linking
- **[Admin Settings](admin-settings.md)** — User management, AI provider configuration, OIDC, security, catalogs, logging, and scheduled tasks
- **[External Tool Server](../external-tool-server.md)** — Expose your collection to external AI clients via OpenAPI
- **[Numista Catalog Lookup](numista-integration.md)** — Direct integration with Numista coin catalog

## Mobile & Offline

- **[PWA Features](pwa-features.md)** — Progressive Web App with installable UI, offline read access, and mobile-optimized views
- **[Camera Capture](camera-capture.md)** — Take coin photos directly from your device camera in PWA mode

## Advanced Features

- **[Image Operations](image-operations.md)** — Background removal, text extraction, authenticated image serving, and responsive image variants
- **[PDF Export](pdf-export.md)** — Generate insurance/provenance catalogs with photos and structured data
- **[Bulk Operations](bulk-operations.md)** — Multi-select coins for batch actions
- **[Notifications](notifications.md)** — In-app notifications for social interactions, alerts, and reminders
- **[Auction Calendar](auction-calendar.md)** — Visual calendar of auction dates and custom events

## Integration & Import/Export

- **[Data Import/Export](import-export.md)** — Import coins from JSON, export full collection, backup/restore workflows

---

## Feature Timeline

| Feature | Status | Introduced |
|---------|--------|-----------|
| Collection CRUD | Shipped | v1.0 |
| AI Analysis (Ollama/Anthropic) | Shipped | v1.0 |
| Wish List with AI Search | Shipped | v1.0 |
| Quick Capture | Shipped | v2.0 |
| Wishlist Search Alerts | Shipped | v2.0 |
| Coin Lookup | Shipped | v2.0 |
| Auction Tracking | Shipped | v1.0 |
| Social Features | Shipped | v1.0 |
| External Tool Server | Shipped | v1.0 |
| Coin Sets with Trend Tracking | Shipped | v2.0 |
| Agentic Set Proposal Review | Shipped | v2.1 |
| Coin Agent Notes | Shipped | v2.1 |
| Improved Numista Lookup | Shipped | v4.0 |
| Nomisma Mint Authority Linking | Shipped | v4.0 |
| Deep Agentic Coin Identification | Shipped (default off) | v4.0 |
| OCRE Roman Imperial Evidence | Shipped (default off) | v4.0 |
| Automated RPC Integration | Paused | Post-v4 |
| Stats Health, Investment, Value, Map, and Emperor subviews | Shipped | v2.0 |
| PWA & Mobile Capture | Shipped | v1.0 |

For quick feature lookup by use case, see the [README Features Matrix](../../README.md#-feature-matrix).
