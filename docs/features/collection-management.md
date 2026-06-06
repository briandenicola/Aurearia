# Collection Management

> Organize, filter, search, and browse your ancient coin collection with flexible viewing and sorting options.

## Overview

Collection Management is the heart of Ancient Coins. It provides a responsive gallery interface for browsing your coins with powerful filtering, full-text search, category organization, and flexible sorting.

## Features

### Card Gallery
- Responsive grid layout with coin cards showing primary image, name, ruler, denomination, and category
- Category-based color coding: purple (Roman), olive (Greek), red (Byzantine), steel blue (Modern)
- Click any card to open the coin's detail page

### Filtering & Organization
- **Category Filter** — Filter by Roman, Greek, Byzantine, Modern, or Other
- **Tag Filter** — Show only coins with specific custom tags
- **Status Filter** — Filter by collection coins, wishlist, sold coins, or auction lots
- **Full-Text Search** — Search across name, inscription, ruler, denomination, and notes

### Viewing Modes

#### Swipe Gallery (Mobile/PWA)
- Touch-based card carousel for browsing on mobile devices
- Swipe left/right to navigate coins
- Position persists across page navigation
- Pull-to-refresh to reload collection
- Face toggle to switch between obverse and reverse images on cards

#### Grid View (Desktop)
- Traditional responsive card grid
- Adaptive columns based on screen size
- Same filtering and search options as swipe mode

### Sorting Options
- **Date Added** — Newest or oldest first (default)
- **Last Updated** — Recently modified coins
- **Current Value** — Highest or lowest value coins
- **Random** — Deterministic shuffle with configurable seed for reproducible browsing sessions

### Pagination
- Large collections load coins in pages for better performance
- Seamless infinite scroll or explicit pagination controls (configurable)

### Bulk Operations
- Enter multi-select mode to choose multiple coins
- Batch tagging across selected coins
- Batch status changes (move to wishlist, mark as sold, etc.)
- Batch export to CSV or JSON

## How to Access

1. After login, you'll land on the Collection page by default
2. Use the header to access filters, search, and sort controls
3. Switch between **Swipe** and **Grid** views in Settings → Appearance
4. Click any coin card to view full details

## Configuration

### User Preferences (Settings → Appearance)
- **Default Gallery View** — Choose swipe or grid as your default
- **Default Sort Order** — Select your preferred sort option
- **Time Zone** — Affects timestamps throughout the app

### Random Seed Persistence
When using random sort, a seed is stored in `sessionStorage` under `coins:randomSeed` for reproducible browsing within a session. Navigate away and come back to maintain the same shuffle.

## API Reference

### Collection Endpoints

```
GET  /api/coins                    # List all collection coins
POST /api/coins                    # Create a new coin
GET  /api/coins/:id               # Get coin details
PUT  /api/coins/:id               # Update a coin
DELETE /api/coins/:id             # Delete a coin
POST /api/coins/bulk              # Bulk operations
```

### Query Parameters
- `?search=...` — Full-text search
- `?category=...` — Filter by category
- `?status=...` — Filter by status (collection, wishlist, sold, auction)
- `?tags=...` — Filter by tag IDs
- `?sort=...` — Sort field (date_added, updated_at, current_value, random)
- `?order=...` — Order direction (asc, desc)
- `?seed=...` — Seed for random sort (validates as integer)
- `?page=...` — Page number for pagination
- `?limit=...` — Items per page

## Mobile PWA Considerations

- In PWA mode, a **Select** button appears in the header to enter bulk-select mode
- Camera capture button appears on the detail page for adding photos directly
- Pull-to-refresh gesture works on both iOS and Android
- Gallery view preference syncs across devices when signed in

## See Also

- [Coin Details](coin-details.md) — Store comprehensive metadata per coin
- [Coin Sets](coin-sets.md) — Organize coins into themed collections
- [Custom Tags](custom-tags.md) — Create flexible organizational categories
- [Collection Statistics](statistics.md) — Analyze your collection
