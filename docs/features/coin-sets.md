# Coin Sets

> Organize coins into standard, goal, smart, and human-reviewed Agentic sets with tray presentation, completion tracking, and value trend monitoring.

## Overview

Coin Sets provides flexible organization beyond tags, allowing collectors to create thematic collections, track series completion, monitor portfolio segments, and analyze trends over time. Built on an evolved tagging system while adding set-specific metadata and analytics.

## Set Types

### Standard Sets
- **Purpose** — Flexible, manually-managed collections (e.g., "Favorite Denarii", "Recently Acquired", "Investment Portfolio")
- **Membership** — Manual add/remove
- **Aggregates** — Coin count, total value, average value, ROI
- **Use Case** — General organization, curated collections

### Goal Sets
- **Purpose** — Track progress toward a collecting goal (e.g., "Complete Type Set of Marcus Aurelius")
- **Membership** — Collection and wishlist coins both participate
- **Completion** — Collection items divided by collection plus wishlist items
- **Use Case** — Human-curated goals where wishlist items represent gaps still to fill

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

### Agentic Sets
- **Purpose** — Describe an open-ended set idea and let the agent propose the roster for human review
- **Submission** — Creating an Agentic set submits a proposal request; no set is created immediately
- **Agent workflow** — The Go API queues the request, proxies to the Python agent service, and stores a proposal with scope interpretation, slots, verification notes, and transcript summary
- **Human review** — The user opens the proposal from the notification, reviews scope and roster, edits slots if needed, regenerates with feedback, rejects, or approves
- **Approval** — Only approval creates the actual Agentic set and roster
- **Matching** — After approval, owned collection coins are automatically matched to proposal slots like the Roman Emperors tracker; users do not manually add or remove Agentic set members

## Set Dashboard

A dedicated dashboard shows:

- **Set Cards** — Quick view of each set with:
  - Set name and icon
  - Coin count
  - Total current value
  - Average value per coin
  - Completion percentage (for goal sets)
  - ROI if cost data available

- **Trending Sets** — Sets with significant value changes
- **Recently Updated Sets** — Recently modified memberships or snapshots
- **Create New Set** — Quick-access wizard for Standard, Goal, Smart, and Agentic sets

## Collection Integration

- **Multi-select apply** — Select coins from the collection grid and apply either a legacy tag or a collection set in one bulk action
- **Collection filters** — Use tag-like set chips to filter collection views by set membership
- **Coin detail chips** — Coin Detail shows both legacy tags and collection set memberships in the same Tags & Sets area

## Emperor Tracker

The Sets menu includes **Emperors** at `/sets/emperors` when Emperor Tracker is enabled in **Settings → Account**. This view tracks Roman emperor and figure coverage as a specialized completion set while continuing to use the existing `/api/stats/emperors` backend progress endpoint.

## Pinning Sets to the Sidebar

Any set can be pinned to the sidebar for one-tap access:

- **Where** — Open a set's detail page and use the pin/unpin icon button in the header actions (next to Back), on both the desktop and PWA layouts. The button shows a filled `Pin` icon when unpinned and an outlined `PinOff` icon in gold when pinned, with `aria-pressed` reflecting the current state.
- **Sidebar placement** — Pinned sets render as additional entries under **Sets → My Sets** (and **Emperors**, when enabled), ordered oldest-pinned-first, then alphabetically. Long names truncate with the full name available via a tooltip.
- **Cap** — Up to 5 sets can be pinned at a time, enforced by the server. Attempting to pin a 6th set surfaces an error toast and the pin button disables itself once the cap is reached; unpinning is never capped.
- **Empty state** — With no pinned sets, the Sets submenu is unchanged from its default two entries.
- **Session behavior** — Pinned sets refresh when the app loads for an authenticated user and clear on logout so the next signed-in user only sees their own pins.

## Tray View

The Collection menu includes **Gallery** and **Tray** subviews. Tray view renders the collection with the shared museum-tray presentation and the user-selected felt color from **Settings → Appearance**.

Set detail pages use the same tray presentation for member coins. Embedded tray controls preserve row and spacing adjustments, and compact row controls keep reorder/remove actions available while viewing a set.

## Set Detail View

### Overview Tab
- Set name, description, type, icon, color
- Coin count, total value, average value
- Cost basis and ROI if available
- Completion percentage and wishlist gap counts (goal sets)
- Edit and delete options

### Members Tab (Standard/Goal Sets)
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
- Completion % over time (for goal sets)
- Compare with other sets
- Export trend data

## Completion Tracking

### For Goal Sets
- Add owned coins and wishlist coins to the goal set
- Display completion as `collection / (collection + wishlist)`
- Wishlist coins represent targets still to acquire

### For Agentic Sets
- Review the generated proposal before creation
- Approve to create the roster
- Let automatic matching associate owned coins with proposal slots
- Return to the proposal review flow to regenerate instead of accepting a weak proposal

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
- **Automatic** — All existing tags become standard sets
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

POST   /api/set-builder/runs                 # Submit Agentic set proposal request
GET    /api/set-builder/proposals            # List human-reviewable proposals
GET    /api/set-builder/proposals/:id        # Review one proposal
PUT    /api/set-builder/proposals/:id        # Edit proposal metadata and slots
POST   /api/set-builder/proposals/:id/approve # Approve and create set
POST   /api/set-builder/proposals/:id/reject # Reject without creating a set
POST   /api/set-builder/proposals/:id/regenerate # Regenerate from feedback
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
