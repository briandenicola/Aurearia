# Quickstart: Feature 363 Validation

Use controlled fixtures. Do not depend on live dealer or auction sources.
Feature implementation must not edit an auction subsystem file.

## Prerequisites

1. Create owner A, owner B, and an administrator.
2. Seed owner A with a small collection containing known strengths, gaps,
   missing metadata, and one contradictory field.
3. Prepare typed Coin Copilot specialist fixtures for:
   - verified/available `dealer_listing`;
   - verified/sold dealer listing;
   - verified/unknown dealer listing;
   - partial/available dealer listing;
   - malformed or missing required data;
   - verified auction lot from `auction_search`.
4. Keep provider calls mocked. Record baseline row counts for coins, drafts,
   auction tables, settings, and Coin Copilot run records.

## Scenario 1 — Private collector profile

1. As owner A, open the Collector Profile settings section with no saved row.
2. Verify neutral empty values are displayed and Coin Copilot remains usable.
3. Save a valid budget range, currency, preferred periods/categories,
   exclusions, dealers, and collecting goals.
4. Reload and verify exact round-trip values.
5. Edit values, save, then clear every optional value and verify subsequent GET
   returns the cleared state.
6. Submit reversed budgets, negative/non-finite values, malformed currency,
   duplicate normalized list values, oversized items, and too many items.
7. Verify each invalid request returns field-specific guidance and the previous
   row remains unchanged.
8. As owner B, a public visitor, invited friend, and administrator outside A's
   session, verify A's profile cannot be read or changed.
9. Verify `/api/auth/me`, public profile, follower, export, and log payloads do
   not accidentally disclose collector context except the owner's explicit
   private export if that existing export policy includes it.

## Scenario 2 — Read-only curator guidance

1. Request curator guidance as owner A with the saved profile.
2. Verify the existing Coin Copilot plan/tool progress uses:
   `collection_summary`, `portfolio_review`, and `gap_analysis`.
3. Verify the final answer separates:
   - observed collection facts;
   - strengths/themes;
   - gaps and possible areas to explore;
   - profile preferences/goals that influenced a suggestion;
   - limitations and contradictory/missing data.
4. Clear the profile and repeat. Verify no deleted preference or goal is
   inferred.
5. Repeat with sparse collection data and verify suggestions are not presented
   as observed facts.
6. View, replay, cancel, and dismiss guidance. Compare database state and
   verify no coin, wishlist coin, quick-capture draft, profile, app setting, or
   auction record changed. Only normal existing Coin Copilot run durability
   may change.
7. Verify the Python request contains bounded profile context but no owner ID,
   credential, action URL, confirmation flag, or write tool.

## Scenario 3 — Eligible dealer Add to Wishlist

1. Render a completed `market_search` result containing a typed
   `dealer_listing` with:
   `verificationState="verified"` and `availability="available"`.
2. Verify `CopilotRunProgress.vue` displays an accessible explicit
   **Add to Wishlist** button.
3. Do not click it. Verify no coin or draft is created.
4. Click it and verify the event flows through the existing
   `useCoinSearchChat.addToWishlist`.
5. When category or era needs confirmation, choose a valid current option and
   verify `buildWishlistCoinPayload` and `createCoin` are called once.
6. Verify the created coin belongs to owner A and has `isWishlist=true`.
7. Verify only supported fields are present: name, category, material,
   denomination, ruler, era, notes, source reference URL/text, supported
   references, and interpretable current value.
8. Verify purchase/acquisition, invoice/SKU, storage, sold, owned-collection,
   auction, and draft fields remain empty/default.
9. Repeat with partial verification, unknown availability, and both. Check the
   listing-specific uncertainty, cancel confirmation (zero creates), confirm
   (one create), repeat clicks while pending (one dialog/create), and change
   availability to sold while confirmation is open (zero creates).

## Scenario 4 — Eligibility exclusions

For each fixture below, verify no **Add to Wishlist** button is rendered and no
call can be triggered:

- `auction_search` or `kind="auction_lot"`;
- absent/unknown verification;
- `availability="sold"`;
- null/absent availability;
- malformed result;
- non-specialist prose or rendered `facts` that merely contain words such as
  “verified” or “available”;
- price-trend, sale-observation, or similar-lot result.

Also verify no Feature 363 change is made to an auction model, repository,
service, route, provider configuration, or UI component.

## Scenario 5 — Category/era cancellation and safe mapping

1. Use an eligible dealer result with an unknown category or era.
2. Verify the existing `matchCategoryEra` flow runs and the existing
   `CategoryEraConfirmModal` appears when needed.
3. Cancel the confirmation and verify `createCoin` is never called.
4. Repeat and choose a valid option; verify one wishlist coin is created.
5. Use unsupported material and missing price fixtures. Verify material uses
   the existing safe fallback and current value remains empty when price cannot
   be interpreted.
6. Verify long text is bounded by `buildWishlistCoinPayload` and unsupported
   catalog references are omitted.

## Scenario 6 — Duplicate and retry behavior

1. Double-click an eligible card while creation is pending. Verify `addingIdx`
   allows one request.
2. Click an already-successful card. Verify `addedSet` prevents another
   request.
3. Seed an owner-scoped wishlist coin with the same reference URL, then click
   an equivalent eligible card. Verify the transaction-scoped
   `FindWishlistByReferenceURL` check prevents insertion and the owner receives
   clear duplicate status.
4. Run two concurrent create requests for owner A and the same non-empty
   reference URL. Verify the canonical create transaction produces at most one
   wishlist coin.
5. Repeat for owner B and verify A's row does not block B; owner scoping
   remains intact.
6. Verify ordinary owned-collection creation and wishlist items without a
   reference URL retain existing behavior.

## Scenario 7 — Image attachment

1. For a safe reachable source image, verify the existing sequence performs
   source scrape, proxy, and one obverse upload after coin creation.
2. For scrape failure with a typed fallback image URL, verify the existing
   fallback is used.
3. For unsafe, unreachable, empty, or unsupported images, verify:
   - the wishlist coin remains saved;
   - no second coin is created;
   - no unrelated field is populated;
   - the UI reports or logs a non-fatal image warning.

## Focused automated tests

### Go

- `CollectorProfile` validation, owner uniqueness, absent defaults, atomic
  replacement, and cross-owner isolation;
- handler auth/body/Swagger/error behavior;
- Coin Copilot specialist public projection retains typed dealer fields;
- `CoinService.CreateCoin` reuses owner/reference duplicate lookup inside the
  transaction for wishlist records;
- ordinary collection create, wishlist create without URL, structured
  references, and value snapshots remain unchanged;
- architecture tests prove handler → service → repository → database.

Suggested focused commands for implementation:

```powershell
Push-Location src/api
go test ./models ./repository ./services ./handlers -run 'CollectorProfile|CopilotSpecialist|Wishlist|CreateCoin' -count=1
go test -run TestArchitecture ./...
Pop-Location
```

### Python

- optional collector context accepts valid bounds and rejects unknown fields;
- existing planner selects the three existing read-only capabilities for
  curator requests;
- empty context invents no preferences;
- prompt-like profile text remains inert;
- no create/save/wishlist action tool or callback is added.

```powershell
Push-Location src/agent
ruff check app/ tests/
pytest tests/test_coin_copilot_contract.py `
  tests/test_coin_copilot_harness.py `
  tests/test_coin_copilot_security.py `
  tests/test_coin_copilot_architecture.py `
  tests/test_portfolio_review.py -v
Pop-Location
```

### Vue

- collector profile form load/save/edit/clear and validation errors;
- profile values are not written to `localStorage`;
- `CopilotRunProgress` eligibility matrix;
- button emission occurs only on an explicit click;
- `CoinSearchChat` routes the event to the existing wishlist function;
- category/era cancellation, mapping, repeat-click state, duplicate feedback,
  image success, and image failure;
- keyboard, 44px touch target, dark theme, 320px viewport, and installed-PWA
  layout.

```powershell
Push-Location src/web
npx vitest run `
  src/components/settings/__tests__/CollectorProfileSection.test.ts `
  src/components/chat/__tests__/CopilotRunProgress.collector.test.ts `
  src/composables/__tests__/useCoinSearchChat.test.ts `
  src/components/__tests__/CoinSearchChat.copilot.test.ts
npm run type-check
Pop-Location
```

## Existing-workflow regression gate

Verify unchanged:

- normal collection coin creation and update;
- legacy Coin Search suggestion cards and their current wishlist button;
- wishlist page and availability checks;
- quick-capture drafts and Deep Analysis;
- Coin Copilot run/replay/cancel/fallback;
- all auction tracking and auction-result rendering.

## Full quality gate for implementation

```powershell
Push-Location src/api
go build ./...
go vet ./...
go test ./...
go test -run TestArchitecture ./...
Pop-Location

Push-Location src/agent
python -m pip install -e ".[dev]"
ruff check app/ tests/
pytest tests/ -v
Pop-Location

Push-Location src/web
npm ci
npm run lint
npm run type-check
npm run test
npm run build
Pop-Location

task openapi
```

## Expected result

Each owner may keep one small private context profile. Coin Copilot uses it
only to contextualize its existing read-only collection analysis. A wishlist
coin is created only when the owner explicitly clicks an eligible dealer card
under amended FR-012 and confirms partial/unknown uncertainty when present,
through the existing canonical wishlist flow. Auction
results never expose the action, and no action platform or additional
subsystem exists.
