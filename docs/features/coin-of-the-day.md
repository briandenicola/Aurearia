# Coin of the Day

> Daily featured coin with automated scheduling, in-app and push notifications, and idempotent cross-restart selection.

## Overview

Coin of the Day surfaces a featured coin each day, helping you rediscover coins you might have forgotten about. It uses intelligent selection to ensure each coin is featured only once before cycling through again.

## Features

### Daily Selection
- **Automated Scheduler** — Picks one coin per day from your collection
- **Configurable Time** — Set when daily selection occurs (default: morning)
- **Cycling Algorithm** — Cycles through every owned, non-wishlist, non-sold coin once before repeating
- **Idempotent** — Safe across process restarts on same day

### Notifications

#### In-App Notification
- Badge in notification inbox
- Appears after scheduled time
- Click to view coin in modal

#### Pushover Integration (Optional)
- Desktop/mobile notifications if Pushover configured
- Same message as in-app notification

### Featured Coin Modal

**Click notification to open:**

- **Coin Images** — Gallery of obverse, reverse, edge photos
- **Metadata** — Name, ruler, denomination, category, material, weight, diameter
- **AI Summary** — Cached analysis (obverse + reverse combined)
  - Fallback chain: AI Analysis → Obverse + Reverse → Structured fields → Bare name
- **Pricing** — Purchase price, current value, cost basis
- **Links** — View full coin detail page

### User Opt-In

**Per-User Control:**
- **User Setting** — `Settings → Account → Coin of the Day`
- **Toggle** — Enable/disable Coin of the Day notifications
- **Default** — Enabled for new users
- **Private Setting** — Not shared with followers

## Selection Algorithm

The system picks coins using a priority-based ORDER BY:

```sql
LEFT JOIN featured_coins ON coin.id = featured_coins.coin_id
WHERE user_id = ?
  AND status = 'collection'  -- not wishlist or sold
ORDER BY
  (last_shown IS NULL) DESC,  -- un-featured coins first
  last_shown ASC,              -- then oldest-featured coins
  coin.id ASC                  -- break ties deterministically
LIMIT 1
```

### Results
- **First month** — Each coin featured once before any repeats
- **Next month** — Cycle repeats from the beginning
- **Idempotent** — Re-running on same day picks same coin

### Dual Idempotency

- **In-Memory Cache** — Process remembers picks for current day
- **DB Check** — HasBeenFeaturedToday query prevents duplicates
- **Result** — Safe across restarts within a 24h window

## Admin Configuration

**Admin → Coin of the Day:**

- **Enable/Disable** — Toggle feature on/off globally
- **Daily Execution Time** — When to run scheduler (default: 06:00 AM)
- **Manual Trigger** — Button to run immediately
- **Run History** — View past runs:
  - Timestamp
  - User ID
  - Featured Coin ID
  - Status (success/skipped/error)
  - Per-user pick history with drill-down

## Configuration

### Settings Defaults

| Setting | Default | Description |
|---------|---------|---|
| `CoinOfDayEnabled` | true | Feature enabled/disabled |
| `CoinOfDayStartTime` | "06:00" | Daily execution time (24h format) |

### Environment Variables

None required; uses database settings.

### Per-User Toggle

Users can opt-out individually in Settings → Account. Disabling prevents notification generation; coin still participates in selection algorithm but no notification is sent.

## API Endpoints

```
GET    /api/featured-coins/latest                # Get today's featured coin
GET    /api/featured-coins/:id                   # Get specific featured coin (preloads coin.images)
POST   /admin/coin-of-day/run                    # Admin: trigger manual run
GET    /admin/coin-of-day-runs                   # Admin: list run history
GET    /admin/coin-of-day-runs/:id               # Admin: get run status/counts
GET    /api/auth/me                              # Includes coinOfDayEnabled
PUT    /api/user/profile                         # Update coinOfDayEnabled
```

## Data Model

### Featured Coins Table

```
featured_coin {
  id:              UUID
  user_id:         UUID (FK → users)
  coin_id:         UUID (FK → coins)
  displayed_at:    timestamp
  summary:         text (cached markdown)
  ai_analysis:     text (obverse + reverse combined)
  created_at:      timestamp
}
```

### Indexes
- `(user_id, displayed_at DESC)` — Quick lookup of recent features
- `(user_id, coin_id)` — Prevent duplicates per day

## Notifications

**Notification Type**: `coin_of_day`

**Payload**:
- `referenceId` — The `FeaturedCoin.ID` (NOT the coin ID)
- `title` — "Featured Coin: {coin name}"
- `message` — "{ruler}, {denomination}, {era}"
- `action_url` — None; click opens modal instead

## Summary Caching

To prevent repeated AI calls, the summary is computed at pick time:

**Fallback chain**:
1. Existing `AIAnalysis` (if both obverse & reverse present)
2. Concatenate `Obverse + Reverse` markdown
3. Format structured fields (name, ruler, denomination, etc.)
4. Bare `Coin.Name` as last resort

**Cached in** `featured_coins.summary` for instant modal loading.

## Related Features

- [Collection Management](collection-management.md) — Browse all coins
- [AI Coin Analysis](ai-analysis.md) — Generate coin summaries
- [Notifications](notifications.md) — Notification management

## See Also

- [Architecture: Background Schedulers](../ARCHITECTURE.md#background-schedulers)
