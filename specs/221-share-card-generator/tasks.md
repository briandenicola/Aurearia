# Tasks: Share Card Generator

**Input**: `specs/221-share-card-generator/spec.md`, `specs/221-share-card-generator/plan.md`  
**Branch**: `221-share-card-generator-tasks`  
**Target merge**: `beta`

## Phase 1: Test Fixtures and Privacy Contract

**Purpose**: Establish the exact privacy-safe output contract before implementation.

- [x] T001 [P] Add share-card test fixture helpers in `src/web/src/utils/__tests__/coinShareCard.test.ts` using `buildCoin()` from `src/web/src/test/fixtures/coins.ts`.
- [x] T002 [P] Add a metadata allowlist test in `src/web/src/utils/__tests__/coinShareCard.test.ts` proving generated metadata includes only name, ruler, denomination, era, mint, material, grade, and category.
- [x] T003 [P] Add privacy exclusion tests in `src/web/src/utils/__tests__/coinShareCard.test.ts` proving purchase price, current value, purchase location, notes, AI analysis, listing status, user id, tags, sets, and private flag are not returned by the metadata helper.
- [x] T004 [P] Add preferred-image selection tests in `src/web/src/utils/__tests__/coinShareCard.test.ts` for obverse-first, primary fallback, first-image fallback, and no-image cases.

**Checkpoint**: Tests describe the card's data contract and fail because the helper does not exist yet.

## Phase 2: Rendering Helper

**Purpose**: Create deterministic client-side card generation with safe metadata and image fallback.

- [x] T005 Create `src/web/src/utils/coinShareCard.ts` with exported `CoinShareCardInput`, `CoinShareCardMetadata`, and `CoinShareCardRenderOptions` types.
- [x] T006 Implement `getShareCardMetadata(coin: Coin): CoinShareCardMetadata` in `src/web/src/utils/coinShareCard.ts` using an explicit allowlist and trimming empty fields.
- [x] T007 Implement `getPreferredShareImage(coin: Coin): string | null` in `src/web/src/utils/coinShareCard.ts`, returning `/uploads/{filePath}` for obverse, primary, first image, or `null`.
- [x] T008 Implement safe filename generation in `src/web/src/utils/coinShareCard.ts`, e.g. `getShareCardFilename(coin): string`, with filesystem-unsafe characters removed.
- [x] T009 Add canvas image-loading helper in `src/web/src/utils/coinShareCard.ts` that resolves `HTMLImageElement` and rejects on load failure instead of silently continuing.
- [x] T010 Implement `renderCoinShareCard(input): Promise<Blob>` in `src/web/src/utils/coinShareCard.ts` with a fixed v1 template, branded title, image frame, metadata block, and missing-image placeholder.
- [x] T011 Add a render test in `src/web/src/utils/__tests__/coinShareCard.test.ts` with mocked canvas APIs proving `canvas.toBlob()` is called and returns a PNG blob.
- [x] T012 Add a missing-image render test in `src/web/src/utils/__tests__/coinShareCard.test.ts` proving a branded placeholder renders without throwing.

**Checkpoint**: Helper tests pass independently without Vue component wiring.

## Phase 3: Share and Download Composable

**Purpose**: Add browser capability detection, native Web Share integration, and download fallback.

- [x] T013 Add `src/web/src/composables/useCoinShareCard.ts` exposing `sharing`, `shareCoinCard(coin: Coin): Promise<CoinShareResult>`, and any needed readonly state.
- [x] T014 In `useCoinShareCard.ts`, call `renderCoinShareCard()`, wrap the blob in a `File`, and use `navigator.canShare?.({ files: [file] })` before native sharing.
- [x] T015 In `useCoinShareCard.ts`, implement native share path with `navigator.share({ files, title, text })`.
- [x] T016 In `useCoinShareCard.ts`, implement download fallback via object URL, temporary anchor, safe filename, and URL revocation.
- [x] T017 In `useCoinShareCard.ts`, surface generation/share/download failures through `useDialog().showAlert()` and rethrow or return an explicit failure result; do not silently swallow errors.
- [x] T018 Add `src/web/src/composables/__tests__/useCoinShareCard.test.ts` for supported native share path with mocked `navigator.share` and `navigator.canShare`.
- [x] T019 Add fallback download test in `src/web/src/composables/__tests__/useCoinShareCard.test.ts` proving object URLs are revoked and the generated anchor uses the safe filename.
- [x] T020 Add error-path test in `src/web/src/composables/__tests__/useCoinShareCard.test.ts` proving render/share failures display an alert.

**Checkpoint**: The composable can be tested without `CoinDetailPage.vue`.

## Phase 4: Coin Detail UI Integration

**Purpose**: Add the Share action to the coin detail workflow without changing existing Sell/Edit/Delete behavior.

- [x] T021 Update `src/web/src/components/coin/CoinDetailHeaderActions.vue` to import `Share2` from `lucide-vue-next`.
- [x] T022 Update `CoinDetailHeaderActions.vue` props to accept `sharing?: boolean` and emit a `share` event.
- [x] T023 Add a Share button in `CoinDetailHeaderActions.vue` using existing `btn btn-secondary btn-xs` classes; disable it while `sharing` is true and label it `Sharing...` while active.
- [x] T024 Add or update `src/web/src/components/coin/__tests__/CoinDetailHeaderActions.test.ts` proving Share emits, Sell/Edit/Delete remain available, and the Share button disables while sharing.
- [x] T025 Update `src/web/src/pages/CoinDetailPage.vue` to import and instantiate `useCoinShareCard()`.
- [x] T026 Wire `@share="handleShare"` and `:sharing="sharing"` on `CoinDetailHeaderActions` in `CoinDetailPage.vue`.
- [x] T027 Add `handleShare()` in `CoinDetailPage.vue` that no-ops only when `coin.value` is absent and otherwise awaits `shareCoinCard(coin.value)`.
- [x] T028 Ensure the detail page keeps existing lightbox, wishlist purchase, sell, delete, and refresh behavior unchanged.

**Checkpoint**: Coin detail page exposes Share and existing actions still behave as before.

## Phase 5: End-to-End Component Coverage

**Purpose**: Prove the user-facing workflow and privacy behavior at the component boundary.

- [x] T029 Add or update `src/web/src/pages/__tests__/CoinDetailPage.test.ts` to mount the detail page with a loaded fixture coin and assert the Share action is visible.
- [x] T030 Mock `useCoinShareCard()` in `CoinDetailPage.test.ts` and assert clicking Share calls `shareCoinCard()` with the current coin.
- [x] T031 Cover the pricing/value privacy regression at the helper metadata boundary in `src/web/src/utils/__tests__/coinShareCard.test.ts`.
- [x] T032 Add an unsupported-browser workflow test at the composable or page level proving Share does not dead-end and triggers download fallback.

**Checkpoint**: The exact user path from coin detail Share click to native share/download fallback is covered.

## Phase 6: Documentation and Validation

**Purpose**: Finish the branch with clear validation and no unrelated behavior changes.

- [x] T033 Update `specs/221-share-card-generator/spec.md` acceptance criteria from unchecked to checked only for behavior completed in this branch.
- [x] T034 Update `specs/221-share-card-generator/plan.md` if implementation chooses the optional reverse thumbnail or changes the fallback from download-only.
- [x] T035 Run targeted tests from `src/web`: `npm.cmd test -- coinShareCard useCoinShareCard CoinDetailHeaderActions CoinDetailPage --run`.
- [x] T036 Run `npm.cmd run type-check` from `src/web`.
- [x] T037 Run `npm.cmd run build` from `src/web`.
- [x] T038 Confirm the final diff contains no API/backend changes and no generated `dist/` artifacts.

## Dependencies and Execution Order

1. Phase 1 must be written first because it defines the privacy contract.
2. Phase 2 depends on Phase 1 and blocks share/download behavior.
3. Phase 3 depends on Phase 2 and blocks UI integration.
4. Phase 4 depends on Phase 3 and must preserve existing detail actions.
5. Phase 5 depends on Phase 4 and proves the full user path.
6. Phase 6 is final validation.

## Parallel Opportunities

- T001-T004 can be authored together.
- T011-T012 can be added while T005-T010 are implemented.
- T018-T020 can run in parallel after the composable API is shaped.
- T024 and T029-T032 can be split between component and page tests after the UI contract is defined.

## Implementation Notes

- Keep v1 frontend-only. Do not add a Go endpoint or database model.
- Keep one fixed card template. Do not add user-editable layout controls.
- Use allowlists rather than deleting sensitive fields from a copied `Coin`.
- Prefer direct, typed helpers over broad casts for `navigator.share` and canvas APIs.
- If canvas image loading fails for a coin image, show an explicit alert instead of producing a success-shaped broken share.
