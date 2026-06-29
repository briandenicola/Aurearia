# Quickstart: Quick Capture

## Prerequisites

- Work from repository root: `C:\Users\brian.denicolafamily\Code\AncientCoins`
- Do not switch branches for this plan; implementation follows `specs/336-quick-capture/`.
- Keep Python agent changes out of scope unless a test proves an existing contract was accidentally affected.

## Implementation order

1. **Backend model and migration**
   - Add quick-capture draft, draft image, and lifecycle event models under `src/api/models/`.
   - Add them to `database.AutoMigrate`.

2. **Backend repository/service/handler**
   - Add owner-scoped repository methods for list/get/create/update/discard/image operations.
   - Add service methods for partial-save validation, image orchestration, promotion readiness, transactional promotion, and idempotent repeated promotion.
   - Add protected routes in `src/api/main.go`.
   - Add Swagger annotations for every public handler.

3. **Image reuse**
   - Extract reusable validation/file-save helpers from existing coin image upload code.
   - Extend authenticated media lookup so draft images can be rendered by `AuthenticatedImage`.

4. **Frontend**
   - Add TypeScript types and API client functions.
   - Add `/quick-capture`, `/quick-capture/drafts`, and `/quick-capture/drafts/:id`.
   - Add PWA/mobile navigation entry and functional desktop route/sidebar entry.
   - Use existing buttons, chips, tokens, icons, camera/file fallback, and `AuthenticatedImage`.

5. **Tests**
   - Backend: repository/service/handler tests for owner scope, validation, image rejection, create/update/discard, promotion success, and repeated promotion.
   - Backend regression: drafts do not change `/coins?wishlist=false&sold=false`, `/stats`, wishlist totals, sold totals, or health eligible rows; promotion changes active count exactly once.
   - Frontend: API client payload tests, draft list/resume component tests, mobile/PWA capture flow tests, promotion error/success tests.
   - Existing add/edit/image tests must continue to pass.

## Manual verification path

Related artifacts:

- API contract: `specs/336-quick-capture/contracts/quick-capture-api.md`
- MVP task checklist: `specs/336-quick-capture/tasks.md`
- Requirements checklist: `specs/336-quick-capture/checklists/requirements.md`

1. Start app services using the repository's normal development command.
2. Log in as a collector.
3. In a 375px-wide/PWA-like viewport:
   - Open Quick Capture from navigation.
   - Add obverse/reverse photos or upload files.
   - Enter a working title or note, plus optional date/era/source/price.
   - Save.
4. Verify:
   - Draft appears in Quick Capture Drafts with preview, title/context, incomplete label, and updated timestamp.
   - Main collection count, wishlist count, sold count, and health summary are unchanged.
5. Reopen the draft:
   - Edit fields/images.
   - Save.
   - Confirm updated data persists.
6. Attempt promotion with missing required normal-coin fields:
   - Verify field-specific guidance and no new coin.
7. Complete required fields and promote:
   - Verify exactly one normal coin is created.
   - Verify draft is no longer listed as active and links to the promoted coin.
   - Verify collection count increases exactly once.
8. Repeat the promote action/request:
   - Verify no duplicate coin is created and the existing promoted coin is returned or clearly messaged.

## Automated verification notes

- US2 frontend draft-card/list/resume behavior is covered by targeted component/source tests for authenticated preview media, incomplete/context labels, updated timestamps, empty image fallback, active draft loading, edit persistence wiring, validation errors, and discard confirmation.
- US3/US4 backend regressions cover exact-once promotion count behavior, wishlist/sold count preservation, owner-scoped draft media, normal coin list exclusion before promotion, promoted coin list/edit compatibility, and health eligible-row changes only after promotion.
- US4 frontend preservation is represented by targeted tests for collection/wishlist/sold/stats/edit/navigation contracts rather than broad page tests where existing harnesses are brittle.
- Manual 375px PWA verification remains recommended before release; it was not executed in this non-interactive pass.

## Quality gate commands

From `src/api/`:

```bash
go vet ./...
go test ./...
```

From `src/web/`:

```bash
npm run build
```

If implementation unexpectedly touches `src/agent/`:

```bash
ruff check app/ tests/
pytest tests/ -v
```
