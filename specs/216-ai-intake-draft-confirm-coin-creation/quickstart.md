# Quickstart: AI Intake Draft + Confirm Coin Creation (#216)

## Prerequisites

- Go API, Vue web app, and Python agent service are running (`task up-all` from repo root).
- Authenticated test user with add-coin access.
- At least one coin image set suitable for OCR/visual attribution.
- Optional: one coin-card image containing dealer/attribution details.
- PWA test context available (standalone display mode or installed app) with controllable camera permissions.

## Scenario 0: PWA default entry behavior

1. Open Add Coin in PWA mode.
2. If camera permission is granted, verify agentic intake opens by default with camera-ready capture.
3. Verify image upload remains available in the same intake surface.
4. Verify a link labeled `Use Manual Mode instead` appears below camera view.
5. Click the link and verify existing manual Add Coin form opens with no intake commit call.

## Scenario 1: Generate intake draft

1. Open Add Coin page.
2. Upload obverse/reverse images and optionally upload coin-card image.
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
5. Submit invalid coin-card upload format -> verify deterministic validation error response.
6. Deny camera permission in PWA mode -> verify upload-based intake and manual-mode link remain usable.

## Compatibility Check (Manual Bypass)

1. In desktop viewport, create a coin manually (without intake) using existing Add Coin flow.
2. Verify manual flow remains unchanged and successful.
3. Verify user can switch to manual mode immediately without generating intake draft first.
