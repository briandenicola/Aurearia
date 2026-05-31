# Quickstart: AI Intake Draft + Confirm Coin Creation (#216)

## Prerequisites

- Go API, Vue web app, and Python agent service are running (`task up-all` from repo root).
- Authenticated test user with add-coin access.
- At least one coin image set suitable for OCR/visual attribution.

## Scenario 1: Generate intake draft

1. Open Add Coin page.
2. Upload obverse/reverse images and provide optional intake prompt.
3. Trigger AI draft generation.
4. Verify response contains:
   - structured draft payload (coin candidate fields),
   - confidence summary,
   - evidence items,
   - unresolved fields list when applicable.

## Scenario 2: Review and edit draft

1. Open intake review panel from generated draft.
2. Inspect field confidence and evidence.
3. Edit at least three draft fields (e.g., ruler, denomination, mint).
4. Verify staged payload reflects user edits, not original AI values.

## Scenario 3: Confirm commit creates coin + journal

1. Submit explicit confirm action with staged draft/overrides.
2. Verify API returns created coin payload and commit success metadata.
3. Verify new coin appears in collection and opens in Coin Detail page.
4. Verify coin activity journal includes AI intake source-tagged creation entry.

## Negative Scenarios

1. Attempt commit with missing/invalid `draftId` -> verify deterministic validation error.
2. Attempt commit on expired/discarded draft -> verify rejection and no coin created.
3. Attempt duplicate commit for already-confirmed draft -> verify no duplicate coin write.
4. Attempt cross-user draft commit (different auth context) -> verify unauthorized/not-found behavior.

## Compatibility Check

1. Create a coin manually (without intake) using existing Add Coin flow.
2. Verify manual flow remains unchanged and successful.
