# Quickstart: Wishlist Search Alerts Planning

This quickstart describes the expected implementation verification path. It does not implement code.

## Prerequisites

- Repository root: `C:\Users\brian.denicolafamily\Code\AncientCoins`
- Feature directory: `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\337-wishlist-search-alerts`
- Branch must remain `feature/357-wishlist-search-alerts`
- Go API, Vue web app, and Python agent use the existing local/Docker setup.

## Implementation outline

1. Add Go models for alerts, runs, candidates, provenance, and review actions.
2. Add repositories that use `repository.OwnedBy` / `OwnedByID` for every owner-scoped query.
3. Add `WishlistSearchAlertService` for validation, CRUD, Run Now, duplicate suppression, review actions, and conversion transactions.
4. Add a typed `AgentProxy` discovery method and matching Python request/response models/route that reuse existing search capability.
5. Add thin Go handlers and protected routes under `/api/wishlist/search-alerts`.
6. Add frontend types/API client functions and a separate wishlist discovery page/review queue.
7. Add tests proving alert discovery does not mutate availability run/result history or listing status.

## Manual verification scenarios

### 1. Create and manage an alert

1. Sign in as a collector.
2. Create an alert with ruler/issuer, coin type, date range, mint, material, grade/condition, price range, source filters, cadence, and keywords.
3. Verify the alert appears in the alert list with active state and criteria summary.
4. Disable and re-enable the alert.
5. Edit criteria and verify future runs use the new criteria while old run detail still shows the old criteria snapshot.

Expected:
- No wishlist item is created.
- Existing wishlist availability settings/history are unchanged.

### 2. Manual Run Now

1. Select Run Now for an active alert.
2. Verify a run is recorded with `manual` trigger, started/completed timestamps, status, criteria snapshot, counts, and non-secret warnings/errors.
3. Verify candidates include source URL, title, reason for match, last seen, lifecycle state, and provenance status.

Expected:
- Duplicate candidates in repeated runs are updated/suppressed rather than creating noisy active rows.
- Broad-result alerts are capped and explain truncation.
- Agent failures produce `failed` or `partial` run status.

### 3. Candidate review

1. Dismiss a candidate with a reason.
2. Restore it.
3. Convert a verified candidate to a wishlist item.
4. Attempt conversion of a partial candidate and verify missing/uncertain fields require review or input.

Expected:
- Conversion is explicit.
- Created coin has `isWishlist=true` and source-backed `referenceUrl`.
- Candidate is marked `converted` and links to the created coin.

### 4. Existing availability separation

1. Start with a wishlist item that has `referenceUrl`.
2. Run existing `POST /api/wishlist/check-availability`.
3. Run a separate search alert.
4. Inspect availability history and alert run history.

Expected:
- Availability run checks saved wishlist URLs only.
- Alert run creates candidates only.
- Search alert run does not update `Coin.listingStatus`, `listingCheckedAt`, or `listingCheckReason`.
- Converted candidate becomes a normal wishlist item and only then participates in later availability checks.

## Test commands for implementation

From `src/api/`:

```powershell
go test -v ./...
go vet ./...
go build ./...
```

From `src/web/`:

```powershell
npm run build
```

From `src/agent/` if agent code changes:

```powershell
ruff check app/ tests/
pytest tests/ -v
```

From repository root:

```powershell
task build
task test
```

## Regression tests to include during implementation

- Alert CRUD rejects invalid ranges, unsupported cadence, empty criteria, and malformed source filters.
- Owner A cannot list, run, dismiss, convert, or delete Owner B alerts/candidates.
- Manual alert run stores criteria snapshot and candidate provenance.
- Duplicate suppression updates `lastSeenAt`/suppresses duplicates using canonical URL + normalized title + observed price + alert ID.
- Candidate conversion creates a wishlist `Coin` in a transaction and marks candidate converted.
- Conversion warns on matching existing wishlist item or previously converted candidate.
- Alert run does not create `AvailabilityRun` / `AvailabilityResult`.
- Alert run does not mutate existing coin listing status.
- Existing availability tests continue to pass.
- Agent discovery response schema rejects invented/missing required source fields and preserves partial/unverified statuses.
