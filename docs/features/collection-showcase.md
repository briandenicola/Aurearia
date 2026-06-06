# Collection Showcase

> Create and share curated subsets of your collection with unique shareable URLs.

## Overview

Collection Showcase lets you create themed, public-read-only subsets of your collection and share them with unique URLs. Perfect for sharing favorite coins, investment portfolios, or themed collections.

## Features

- **Create Showcases** — From your collection, create new showcases with title, description, slug
- **Two-Column Coin Picker** — Side-by-side interface to add/remove coins
- **Draft/Publish Toggle** — Control visibility; only published showcases are public
- **Unique URLs** — Each showcase gets `/s/:slug` (e.g., `/s/favorite-denarii`)
- **Public Read-Only** — Visitors see coin images and metadata; no pricing or AI analysis
- **No Auth Required** — Public URLs work without login

## Workflow

1. Go to **Collection → Create Showcase**
2. Enter title (e.g., "My Favorite Denarii")
3. Enter slug (URL-friendly name)
4. Add description (optional)
5. Use two-column picker to add coins
6. Click **Save as Draft**
7. Preview the showcase
8. Click **Publish** when ready
9. Share the public URL

## Public Showcase

**Visitors see:**
- Coin images (obverse/reverse gallery)
- Coin name, ruler, denomination, era, category
- Material, weight, diameter
- Free-text notes

**Visitors don't see:**
- Purchase price or current value
- AI analysis or ratings
- Private coins (if marked private)
- Comments or follower information

## Managing Showcases

**From Showcases Page:**
- View all your showcases
- See publication status (Draft/Published)
- Edit title, description, slug
- Toggle between draft and published
- Delete showcases
- View public URL for published showcases

## Coin Privacy in Showcases

- If a coin is marked **Private**, it cannot be added to showcases
- Toggling a coin to private removes it from existing showcases
- Removed coins disappear from the public view immediately

## API Endpoints

```
GET    /api/showcases                # List user's showcases
GET    /api/showcases/:id            # Get showcase
POST   /api/showcases                # Create showcase
PUT    /api/showcases/:id            # Update showcase
DELETE /api/showcases/:id            # Delete showcase
PUT    /api/showcases/:id/coins      # Manage coins in showcase

GET    /api/showcase/:slug           # Get public showcase (no auth)
```

## Sharing

- Share the public URL: `https://yourdomain.com/s/favorite-denarii`
- Showcase URL works forever (or until deleted)
- No password protection; use discreet slug names if privacy matters

## Related Features

- [Collection Management](collection-management.md) — Browse collection
- [Social Features](social-features.md) — Follow collectors
- [Coin Details](coin-details.md) — Full coin information

See also: [Getting Started](../getting-started.md#sharing)
