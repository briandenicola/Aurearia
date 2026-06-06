# Collection Statistics

> View comprehensive analytics and visualizations of your collection's composition, value trends, and performance.

## Overview

The Stats page provides deep insights into your collection through interactive charts, rankings, and distribution analysis. Track portfolio value over time, identify your most valuable coins, understand your collecting patterns, and spot gaps in your collection at a glance.

## Summary Cards

Quick-view metrics at the top:

- **Total Coins** — Count of coins in active collection (excludes sold and wishlist)
- **Total Value** — Current estimated total value of all coins
- **Average Value** — Mean value per coin
- **Unique Rulers** — Count of distinct rulers/authorities in collection

## Category Breakdown

Pie chart showing distribution across:
- Roman
- Greek
- Byzantine
- Modern
- Other

Hover to see:
- Count of coins in category
- Percentage of collection
- Total category value

## Material Distribution

Breakdown by material type:
- Gold
- Silver
- Bronze
- Copper
- Electrum
- Other

Shows:
- Coin count per material
- Percentage of collection
- Total material value
- Average value per material

## Grade Distribution

Bar chart showing coins by grade:
- VF (Very Fine)
- EF (Extremely Fine)
- AU (About Uncirculated)
- MS (Mint State)
- Other grades

Features:
- Blue gradient coloring
- Hover for exact counts
- Identify grade gaps in collection

## Value Over Time

Interactive line chart tracking:

- **Portfolio Value** — Total current value of all coins (blue line)
- **Total Invested** — Sum of all purchase prices (gold line)
- **ROI** — Return on investment (calculated from snapshots)

### Data Source
- Automatic snapshots recorded after every coin create/update/delete
- Historical snapshots visible when zoomed into date ranges
- Hover for exact values on specific dates

### Interactions
- **Zoom** — Click and drag to zoom into date range
- **Hover** — See exact values at any point
- **Export** — Download trend data as CSV

## Top Coins by Value

Ranked list of your most valuable coins:

| Rank | Coin Name | Category | Material | Current Value | Purchase Price | ROI |
|------|-----------|----------|----------|---|---|---|
| 1 | Julius Caesar Aureus | Roman | Gold | $2,850 | $1,200 | +137% |
| ... | ... | ... | ... | ... | ... | ... |

Features:
- Click any coin to open detail page
- Sort by value, ROI, or acquisition date
- Filter by category or material

## Era/Region Heat Map

SVG-based heat map showing distribution across:

**Time Periods (Rows)**:
- 500 BC - Ancient
- 1 AD - Ancient Roman
- 500 AD - Medieval
- 1000 AD - Medieval
- 1500 AD - Early Modern
- 1800 AD - Modern

**Geographic Regions (Columns)**:
- Mediterranean
- Near East
- Asia Minor
- Europe
- Other

**Heat Intensity** reflects:
- Concentration of coins in each era/region cell
- Bright = high concentration
- Dim = few or no coins
- Helps identify collection strengths and gaps

## Health Scorecard

Overall collection metrics:

| Metric | Score | Status |
|--------|-------|--------|
| **AI Coverage** | 85% | Coins with AI analysis |
| **Image Coverage** | 92% | Coins with images |
| **Reference Coverage** | 72% | Coins with catalog references |
| **Metadata Completeness** | 88% | Avg % of fields filled per coin |

Status indicators:
- 🟢 Green: >80%
- 🟡 Yellow: 50-80%
- 🔴 Red: <50%

## Collection Composition

Detailed breakdown:
- **By Era** — Coins per time period with percentages
- **By Region** — Geographic distribution
- **By Material** — Precious metals vs. base metals
- **By Grade** — Distribution across grades
- **Price Range** — Histogram of value distribution

## Sold Coin Analytics

Separate section for coins you've sold:

- **Total Sold** — Count of coins sold
- **Total Sale Value** — Revenue from sales
- **Total Cost Basis** — Original investment
- **Overall Profit/Loss** — (Revenue - Basis)
- **Annualized ROI** — Return based on holding period
- **Top Sales** — Most expensive coins sold

## Wishlist Analytics

Metrics for coins on wishlist:

- **Total Wishlist Coins** — Count in wishlist
- **Total Estimated Cost** — Sum of estimated prices
- **Average Estimated Price** — Cost per coin
- **Availability Status** — % available/unavailable/unknown
- **Priority Coins** — Most-watched items

## Filters & Date Ranges

### Date Range Selection
- Last 30 days
- Last 90 days
- Last year
- All time
- Custom range (date picker)

### Filters
- **By Category** — Focus on Roman, Greek, etc.
- **By Material** — Gold only, silver only, etc.
- **By Status** — Collection only, with sold, with wishlist
- **By Grade Range** — Narrow to specific grades

## Export Options

Collection-level export is available from **Settings -> Data** as JSON backup and PDF catalog generation. The Stats page focuses on interactive dashboards rather than separate chart export endpoints.

## Data & Accuracy

### Snapshot Frequency
- Automatic: After every coin create/update/delete
- Manual: Click "Capture Snapshot" to force a snapshot

### Value Sources (in priority order)
1. **Current Value** field — If set manually
2. **AI Analysis** — If coin has AI-generated valuation
3. **Purchase Price** — If no valuation available
4. **Estimated Range** — If no other data exists

### Recalculation
- Stats recalculate when:
  - New coin added/removed
  - Coin value updated
  - Coin status changed (collection → sold → wishlist)
  - Snapshot is generated

## Admin Configuration

No admin configuration needed; stats auto-calculate from your collection data.

No separate chart toggles are currently required.

## API Endpoints

```
GET    /api/stats                   # Summary cards, charts, rankings, value history
GET    /api/stats/distribution      # Era/region heat map
GET    /api/stats/health            # Collection health summary
GET    /api/coins/health            # Paginated per-coin health list
GET    /api/coins/:id/health        # Single coin health score
```

## Performance

- Stats calculations are computed from your collection data and refreshed when coin values, status, or metadata change
- Health metrics are computed from metadata, image, valuation freshness, and per-side AI analysis coverage
- Value history depends on snapshots recorded as collection values change

## Related Features

- [Collection Management](collection-management.md) — Browse collection
- [Coin Sets](coin-sets.md) — Set-specific trend tracking
- [Collection Showcase](collection-showcase.md) — Share statistics publicly
- [AI Analysis](ai-analysis.md) — AI coverage metrics
