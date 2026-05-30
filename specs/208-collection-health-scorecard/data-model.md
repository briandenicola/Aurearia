# Data Model — Collection Health Scorecard (v1)

## 1) CoinHealthScore (computed projection)

Represents health diagnostics for one coin.

### Fields

- `coinId` (uint, required)
- `score` (int, required, 0-100)
- `grade` (enum: `A|B|C|D|F`, required)
- `metadataScore` (int, 0-100, required)
- `imageScore` (int, 0-100, required)
- `valuationFreshnessScore` (int, 0-100, required)
- `aiCoverageScore` (int, 0-100, required)
- `missingItems` (array of `MissingChecklistItem`, required; may be empty)
- `quickActions` (array enum values: `edit_metadata`, `upload_images`, `run_valuation`, `run_ai_analysis`)
- `computedAt` (timestamp, required)

### Validation Rules

- `score` MUST equal weighted sum of dimension scores using fixed default weights (40/20/20/20).
- `grade` MUST map from `score` via fixed threshold table.
- `missingItems` MUST only include known checklist keys.

## 2) MissingChecklistItem (computed child projection)

Represents one missing requirement for a coin.

### Fields

- `key` (string, required; e.g. `metadata.denomination`, `images.reverse`, `valuation.freshness`, `ai.obverse_analysis`)
- `dimension` (enum: `metadata|images|valuation|ai`, required)
- `label` (string, required, UI-safe)
- `severity` (enum: `high|medium|low`, required)
- `actionHint` (enum: `edit_metadata|upload_images|run_valuation|run_ai_analysis`, required)

### Validation Rules

- `dimension` MUST match `key` prefix.
- `actionHint` MUST map to an existing user workflow.

## 3) CollectionHealthSummary (computed projection)

Represents one user’s collection-level scorecard.

### Fields

- `userId` (uint, required)
- `score` (int, required, 0-100)
- `grade` (enum `A|B|C|D|F`, required)
- `eligibleCoinCount` (int, required, >=0)
- `dimensionAverages` (object with metadata/image/valuation/ai averages, each int 0-100)
- `trend30dDelta` (int|null, nullable when insufficient history)
- `trendDirection` (enum: `up|flat|down|unavailable`)
- `computedAt` (timestamp, required)

### Validation Rules

- `eligibleCoinCount=0` MUST produce `trendDirection=unavailable`.
- `trend30dDelta` MUST be null when no suitable 30-day baseline snapshot exists.

## 4) CollectionHealthSnapshot (persisted table)

Persists daily collection summary for trend calculations.

### Fields

- `id` (uint, PK)
- `userId` (uint, indexed, required)
- `snapshotDate` (date, indexed, required)
- `score` (int, required, 0-100)
- `gradeA` (int, required, >=0)
- `gradeB` (int, required, >=0)
- `gradeC` (int, required, >=0)
- `gradeD` (int, required, >=0)
- `gradeF` (int, required, >=0)
- `eligibleCoinCount` (int, required, >=0)
- `createdAt` / `updatedAt` (timestamps)

### Constraints

- Unique index on (`user_id`, `snapshot_date`) to guarantee one snapshot/day/user.
- Snapshot transaction MUST persist atomically.

## 5) AdminHealthAggregate (computed projection)

Admin-only aggregate across all eligible coins.

### Fields

- `medianScore` (int, required, 0-100)
- `lowScorePercentage` (float, required, 0-100; low score means `<60`)
- `lowScoreCount` (int, required, >=0)
- `eligibleCoinCount` (int, required, >=0)
- `topMissingFields` (array of `MissingFieldStat`, required)
- `computedAt` (timestamp, required)

## 6) MissingFieldStat (admin aggregate child projection)

### Fields

- `key` (string, required)
- `count` (int, required, >=0)
- `percentage` (float, required, 0-100)

## Relationships

- `CoinHealthScore` is derived from `Coin` (+ `CoinImage`, `CoinValueHistory`, AI fields on `Coin`).
- `CollectionHealthSummary` aggregates many `CoinHealthScore` records for one `User`.
- `CollectionHealthSnapshot` stores historical states of `CollectionHealthSummary`.
- `AdminHealthAggregate` aggregates `CoinHealthScore` and missing checklist frequencies across users.

## State Transitions

### Coin health lifecycle

1. `NeedsScoring` (coin created/updated, image or valuation changed)
2. `Scored` (score + checklist computed on read or batch update)
3. `Improving` (score increase after quick action)
4. `AtRisk` (score < 60)

### Snapshot lifecycle

1. Scheduler trigger (daily)
2. Compute per-user summary
3. Upsert `CollectionHealthSnapshot` for date
4. Trend available after day 31+
