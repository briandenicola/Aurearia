# Features Index

Aurearia provides a comprehensive set of features for managing a personal coin collection, with deep support for ancient and historical coins. This directory contains detailed documentation for each major feature area.

## Visual Assets

The repository currently includes app icons in `src/web/public/` but does not include captured product screenshots or workflow GIFs. Recommended future captures are: collection dashboard, coin entry form, coin detail image upload, Coin Sets dashboard, wishlist availability checks, statistics/health dashboards, and PWA mobile gallery.

## Core Collection Features

- **[Collection Management](collection-management.md)** — Create, browse, filter, search, and organize coins with rich metadata
- **[Coin Details](coin-details.md)** — Store comprehensive numismatic data, images, references, activity journals, and notes
- **[Coin of the Day](coin-of-the-day.md)** — Daily featured coin notifications to help rediscover your collection

## Discovery & Acquisition

- **[Coin Lookup](coin-lookup.md)** — Photograph a coin or slab at a show, extract NGC Ancients certs, verify with NGC, and save to wish list or collection
- **[Wish List](wish-list.md)** — Track coins you'd like to acquire with AI-powered search and availability checking
- **[Auction Tracking](auction-tracking.md)** — Monitor NumisBids and CNG Auctions lots through bidding lifecycle with price alerts and reminders
- **[Sold Coins](sold-coins.md)** — Track coins you've sold with profit/loss analysis

## AI Features

- **[AI Coin Analysis](ai-analysis.md)** — Vision-model analysis of obverse/reverse photos using Anthropic Claude or Ollama
- **[AI Coin Search Agent](ai-search-agent.md)** — Chat with an AI agent to find coins matching your description across dealer sites
- **[AI Grading Assistant](ai-grading.md)** — Estimate coin grades from photos with reasoning and confidence scores
- **[Price Trend Analysis](price-trends.md)** — Analyze historical auction data to identify market trends
- **[Collection Gap Analysis](gap-analysis.md)** — Get AI-powered suggestions for coins missing from your collection
- **[Coin Photography Guide](photography-guide.md)** — Receive critiques and tips for improving your coin photography
- **[Similar Lot Finder](similar-lots.md)** — Find active auction listings similar to coins in your collection

## Organization & Analytics

- **[Coin Sets](coin-sets.md)** — Organize coins into open, defined, goal, and smart sets with trend tracking
- **[Custom Tags](custom-tags.md)** — Create flexible custom categories for organizing your collection
- **[Collection Statistics](statistics.md)** — View analytics including portfolio value, distributions, trends, and rankings
- **[Collection Showcase](collection-showcase.md)** — Create and share curated public coin subsets with shareable URLs

## Social & Community

- **[Social Features](social-features.md)** — Follow collectors, leave comments, and rate coins in shared collections
- **[User Profiles](user-profiles.md)** — Customize your public profile with avatar, bio, and privacy settings

## Administration & Configuration

- **[Authentication](../authentication.md)** — JWT tokens, WebAuthn passkeys, and API keys
- **[Admin Settings](admin-settings.md)** — User management, AI provider configuration, logging, and scheduled tasks
- **[External Tool Server](../external-tool-server.md)** — Expose your collection to external AI clients via OpenAPI
- **[Numista Catalog Lookup](numista-integration.md)** — Direct integration with Numista coin catalog

## Mobile & Offline

- **[PWA Features](pwa-features.md)** — Progressive Web App with installable UI, offline read access, and mobile-optimized views
- **[Camera Capture](camera-capture.md)** — Take coin photos directly from your device camera in PWA mode

## Advanced Features

- **[Image Operations](image-operations.md)** — Background removal, text extraction (OCR), and circle clipping
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
| Coin Lookup | Shipped | v2.0 |
| Auction Tracking | Shipped | v1.0 |
| Social Features | Shipped | v1.0 |
| External Tool Server | Shipped | v1.0 |
| Coin Sets with Trend Tracking | Shipped | v2.0 |
| PWA & Mobile Capture | Shipped | v1.0 |

For quick feature lookup by use case, see the [README Features Matrix](../../README.md#-feature-matrix).
