# Custom Tags

> Create flexible custom categories to organize and filter your collection beyond built-in fields.

## Overview

Custom tags provide flexible organization for your coins, allowing you to create personal categories and use them for filtering and bulk operations.

## Features

- **Create & Manage** — Add, rename, and delete tags from **Settings → Tags**
- **Color Selection** — Choose a color for each tag for visual organization
- **Attach to Coins** — Tag coins during creation or from detail page
- **Multiple Tags** — Any coin can have multiple tags
- **Filter by Tag** — Filter collection view by tag membership
- **Bulk Tagging** — Apply tags to multiple coins at once in bulk select mode

## Workflow

1. Go to **Settings → Tags**
2. Click **+ Create Tag**
3. Enter tag name (e.g., "Investment Portfolio", "Recently Acquired")
4. Choose a color
5. Click Save
6. Tag is immediately available when editing coins

## Tag Usage

**On Coin Detail Page:**
- Add/remove tags from a chip group
- Tags display with chosen color
- Click tag to show related coins

**In Collection View:**
- Use hamburger menu to filter by tag
- Shows only coins with selected tag(s)
- Multiple tag filters use OR logic

## Bulk Operations

- Select multiple coins
- Click **Tag Selected**
- Choose tags to add/remove
- Apply to all selected coins at once

## Built-in Coins Cannot Have Tags

- Wishlist coins can be tagged
- Sold coins can be tagged
- Auction lots cannot be tagged

## API Endpoints

```
GET    /api/tags                     # List all tags
POST   /api/tags                     # Create tag
PUT    /api/tags/:id                 # Update tag
DELETE /api/tags/:id                 # Delete tag

POST   /api/coins/:id/tags           # Add tag to coin
DELETE /api/coins/:id/tags/:tagId    # Remove tag from coin
```

## Migration Note

Prior to Coin Sets feature, tags were the primary organization mechanism. Sets provide additional features (defined sets, smart sets, trend tracking) while tags remain for simple, flexible categorization.

See also: [Coin Sets](coin-sets.md), [Collection Management](collection-management.md)
