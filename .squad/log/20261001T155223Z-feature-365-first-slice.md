# Feature 365 First Slice Handoff

## Scope

Implemented the shared desktop context header and migrated four representative
pages: Wishlist, Followers, Identify Coin, and Stats. Mobile/PWA local headers
remain unchanged. No API, route, database, Go, Python, dependency, deployment,
or release change was made.

## Implementation

- Added stable desktop title/action targets in `App.vue`.
- Added lifecycle-bound deferred Teleports in `DesktopPageContext.vue`.
- Extended `AppIconButton` with typed router-link support.
- Kept page callbacks, loading state, modal state, and route ownership local.
- Preserved collection-specific and global shell actions.
- Added title truncation and an action-group divider.

## Evidence

- Focused command:
  `cd src/web && npm run test -- --run src/components/__tests__/DesktopPageContext.test.ts src/__tests__/AppNavigation.test.ts src/pages/__tests__/WishlistPage.test.ts src/pages/__tests__/StatsPage.test.ts src/pages/__tests__/CoinLookupPage.test.ts`
  - Passed: 5 files, 67 tests.
- Completion command: `task check:web`
  - Passed lint, strict Vue type-check, full tests, and production build.
- Delivery command: `task check:delivery`
  - Passed 91 delivery tests, SpecKit selection/negative controls, and governance
    validation with zero errors and zero warnings.
- Guard proof:
  - Removed `defer` from both Teleports.
  - The shell integration test failed because both targets were unavailable at
    child mount.
  - Restored `defer`; the same focused test passed.

## Remaining

- Actual desktop screenshots and owner review are pending.
- Remaining page headers have not been migrated.
- Narrow-desktop overflow behavior must be validated before deciding whether a
  More actions menu is necessary.
- Independent review, final browser/mobile evidence, acceptance, merge, and
  deployment remain pending.
