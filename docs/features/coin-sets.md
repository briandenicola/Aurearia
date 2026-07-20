# Coin Sets

> Organize coins into open, defined, goal, and smart sets with automatic aggregation, completion tracking, and value trend monitoring.

## Overview

Coin Sets provides flexible organization beyond tags, allowing collectors to create thematic collections, track series completion, monitor portfolio segments, and analyze trends over time. Built on an evolved tagging system while adding set-specific metadata and analytics.

## Set Types

### Open Sets
- **Purpose** — Flexible, manually-managed collections (e.g., "Favorite Denarii", "Recently Acquired", "Investment Portfolio")
- **Membership** — Manual add/remove
- **Aggregates** — Coin count, total value, average value, ROI
- **Use Case** — General organization, curated collections

### Defined Sets
- **Purpose** — Track completion toward a target list (e.g., "US Morgan Dollars 1878-1921", "Roman Denarii of Augustus")
- **Membership** — Manual + completion matching against targets
- **Targets** — Import custom CSV or select from built-in templates
- **Completion** — Show percentage complete and list missing items
- **Built-in Templates** — US Morgan Dollars, US State Quarters, Roman Imperial series, Greek city-states, Byzantine gold coins

### Goal Sets
- **Purpose** — Track progress toward a collecting goal (e.g., "Complete Type Set of Marcus Aurelius")
- **Target Date** — Optional deadline for completion
- **Milestones** — Value-based checkpoints that trigger notifications when crossed
- **Aggregates** — Current count vs. target, value vs. goal

### Smart Sets
- **Purpose** — Rule-based automatic membership (e.g., "All silver coins over $1000")
- **Rules** — Define criteria such as:
  - Material (gold, silver, bronze, etc.)
  - Era / Category (Roman, Greek, Byzantine, Modern)
  - Value range (min/max current value)
  - Grade range
  - Acquisition date range
  - Mint location
  - Wishlist / Sold status
  - AND/OR logic for complex rules
- **Automatic Updates** — Membership recalculates when coins change

## Set Dashboard

A dedicated dashboard shows:

- **Set Cards** — Quick view of each set with:
  - Set name and icon
  - Coin count
  - Total current value
  - Average value per coin
  - Completion percentage (for defined/goal sets)
  - ROI if cost data available

- **Trending Sets** — Sets with significant value changes
- **Recently Updated Sets** — Recently modified memberships or snapshots
- **Create New Set** — Quick-access wizard

## Collection Integration

- **Multi-select apply** — Select coins from the collection grid and apply either a legacy tag or a collection set in one bulk action
- **Collection filters** — Use tag-like set chips to filter collection views by set membership
- **Coin detail chips** — Coin Detail shows both legacy tags and collection set memberships in the same Tags & Sets area

## Tray View

The Collection menu includes **Gallery** and **Tray** subviews. Tray view renders the collection with the shared museum-tray presentation and the user-selected felt color from **Settings → Appearance**.

Set detail pages use the same tray presentation for member coins. Embedded tray controls preserve row and spacing adjustments, and compact row controls keep reorder/remove actions available while viewing a set.

## Set Detail View

### Overview Tab
- Set name, description, type, icon, color
- Coin count, total value, average value
- Cost basis and ROI if available
- Completion percentage and missing items (defined sets)
- Edit and delete options

### Members Tab (Open/Defined/Goal Sets)
- Scrollable list of member coins
- Click to view coin details
- Add membership with a coin picker instead of manual ID entry
- Remove membership with compact design-system actions
- Sort and filter members

### Smart Rules Tab (Smart Sets)
- View active rules
- Edit criteria
- Preview matching coins
- Test rule changes before saving

### Snapshots Tab
- Historical valuation snapshots
- Date, total value, coin count, completion %
- Manual snapshot button
- Delete historical snapshots

### Trends Tab
- Interactive chart showing value over time
- Coin count over time
- Completion % over time (for defined/goal sets)
- Compare with other sets
- Export trend data

### Targets Tab (Defined Sets)
- List of target coins
- Match owned coins to targets
- Show owned vs. missing
- Import custom targets from CSV
- Download current targets

## Completion Tracking

### For Defined Sets
- Define target coins (by catalog reference, ruler, denomination, era)
- System matches owned coins to targets
- Display completion: "42 of 100 (42%)"
- List missing items with suggestions
- Visual checklist showing completed/incomplete targets

### For Goal Sets
- Set target coin count or value
- Track progress toward goal
- Show coins needed to complete
- Milestone notifications

## Trend Monitoring

### Automatic Snapshots
- Daily automatic snapshots at configured time (default: 3 AM)
- Captures: total value, total invested, coin count, completion %, average value, highest-value coin
- Skips empty sets (records zero state if previously had coins)

### Manual Snapshots
- Click "Capture Snapshot" to manually record current state
- Useful before/after major collection changes

### Value Milestones
- Set thresholds (e.g., "$10,000", "$50,000")
- Generate notification once when threshold is crossed
- Track milestone crossings in notification inbox

### Trend Analysis
- Compare set performance over time ranges (1 week, 1 month, 1 year, custom)
- Show value change and percentage change
- Compare multiple sets side-by-side
- Identify best/worst performers

## Set Comparison

Compare up to 3 sets:

- **Value Performance** — Current value, change since snapshot, percentage change
- **Completion** — Completion %, missing coins, targets matched
- **Metrics** — Coin count, average value, ROI
- **Export** — Download comparison as CSV

## Legacy Tag Migration

If upgrading from tags:
- **Automatic** — All existing tags become open sets
- **Preserved** — Names, colors, and coin memberships remain unchanged
- **New Features** — Add descriptions, set types, completion tracking, and snapshots

## Set Privacy

- **Public/Private Toggle** — Control visibility in collection showcase
- **API Scoping** — All set endpoints require authentication; no cross-user access

## API Endpoints

```
GET    /api/sets                    # List all sets
GET    /api/sets/templates          # List built-in set templates
POST   /api/sets/import-csv         # Create a set from CSV
POST   /api/sets/compare            # Compare multiple sets
POST   /api/sets/preview-smart      # Preview smart-set criteria
POST   /api/sets                    # Create a set
GET    /api/sets/:id               # Get set details
PUT    /api/sets/:id               # Update set metadata
DELETE /api/sets/:id               # Delete set

GET    /api/sets/:id/coins         # List member coins
POST   /api/sets/:id/coins         # Add coin to set
DELETE /api/sets/:id/coins/:coinId # Remove coin from set

GET    /api/sets/:id/completion    # Get completion details
POST   /api/sets/:id/snapshot      # Create manual snapshot
GET    /api/sets/:id/trends        # Get trend data
GET    /api/sets/:id/analytics     # Get set analytics
```

## Configuration (Admin)

- **Set Snapshot Schedule** — Interval and time for automatic snapshots (default: daily at 3 AM)
- **Milestone Notification Thresholds** — Value milestones that trigger notifications
- **Smart Set Max Rules** — Maximum number of criteria per smart set (default: 10)

## Related Features

- [Collection Management](collection-management.md) — Browse coins in your collection
- [Collection Statistics](statistics.md) — Aggregate portfolio analytics
- [Custom Tags](custom-tags.md) — Simpler alternative for basic organization
- [Notifications](notifications.md) — Milestone alerts and trend notifications

## See Also

- [Spec: Coin Sets with Trend Tracking](../../specs/main/spec.md)
