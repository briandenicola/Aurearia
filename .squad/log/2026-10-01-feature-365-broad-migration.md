# Feature 365 Broad Desktop Header Migration

## Completed

- Migrated the authenticated page inventory to `DesktopPageContext` across
  static, action-bearing, supporting-copy, and dynamic-title pages.
- Preserved local headers for installed PWA mode and ordinary browser widths up
  to 768 pixels.
- Kept page callbacks, routing, loading/disabled state, menus, and modal state
  owned by each page while moving desktop presentation into the application bar.
- Added shared test Teleport targets and updated affected page tests for the new
  shell contract.
- Left the specialized owned-coin detail shell unchanged because its action bar
  and content title are part of the section-page layout rather than a removable
  page header.

## Verification

- `task check:web` passed on the current uncommitted tree:
  - ESLint passed with zero warnings.
  - `vue-tsc --build` passed.
  - Vite build passed.
  - 1,799 Vitest tests ran: 1,798 passed and 1 skipped.
- `task check:delivery` passed:
  - 91 delivery tests passed.
  - Governance validation reported zero errors and zero warnings.
- The first independent review blocked on Add Coin's Teleported mode controls
  escaping their disabled fieldset. The controls now bind the shared form guard
  directly, and a real-Teleport regression test proves they remain disabled and
  non-operative during intake analysis.
- Contextual-header Playwright coverage passed 6 of 6 tests across the 768/769px
  boundary, normal desktop, installed PWA at desktop width, and the crowded
  four-action Wishlist header. The boundary test also found and drove a CSS
  specificity repair that now keeps shared title/actions hidden at 768 pixels.
- Independent re-review passed for repaired candidate patch SHA-256
  `a29dc341be1cca6b434eea1def84cd4ce1067805023484a0cd776619238eaef4`
  with no remaining high-confidence blocker.
- The repository owner visually accepted the 768px local header, 769px shared
  header, installed-PWA local header, and crowded Wishlist action screenshots
  on 2026-10-01 and authorized commit/push of the feature branch.

## Incomplete

- Commit and push remain pending. Merge into `beta` requires separate owner
  approval.

## Next Action

Commit and push the feature branch, then request separate approval before
merging it into `beta`.
