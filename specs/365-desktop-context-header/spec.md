# Feature Specification: Desktop Context Header

**Work ID / spec directory**: `365-desktop-context-header`
**Working branch**: `365-desktop-context-header`
**Lane**: Feature
**Created**: 2026-10-01
**Status**: Owner-approved implementation in progress

## Approved Outcome

On authenticated desktop layouts, move each page's title into the shared
application bar as `Aurearia | Page Title` and place page-owned actions there as
accessible icon buttons before the existing global actions. Preserve the
existing local page header and action behavior in mobile/PWA layouts.

## Scope

- Add stable desktop title and action hosts to `App.vue`.
- Keep page callbacks, permissions, loading state, routes, and modal state owned
  by the page that renders the action.
- Use lifecycle-bound Vue Teleports rather than module-level reactive state.
- Reuse `AppIconButton` for 44-pixel icon actions with labels and tooltips.
- Truncate long titles before they can overlap the action area.
- Migrate pages incrementally, starting with Wishlist, Followers, Identify Coin,
  and Stats before applying the pattern to the remaining standalone pages.

## Non-Goals

- No mobile/PWA redesign.
- No route, API, authentication, database, or service-contract changes.
- No global page-action registry or module-level current-page state.
- No removal of explanatory content that remains useful after a title moves.
- No automatic release, deployment, or merge to `beta`.

## Acceptance Criteria

1. Desktop displays `Aurearia | Page Title` in the shared bar for migrated pages.
2. Page actions render before a divider and the existing global actions.
3. Every action retains its existing callback, route, disabled/loading state,
   accessible name, tooltip, and usable target size.
4. Navigating away removes the prior page's title and actions without stale
   state; returning renders a fresh page context.
5. PWA/mobile continues to render the existing local title and actions.
6. Long desktop titles truncate rather than overlap actions.
7. The complete migrated web surface passes `task check:web` and affected
   browser/mobile workflows.

## First Usable Slice

The first slice covers the shared shell and four representative page shapes:
Wishlist (multiple stateful actions), Followers (one modal action), Identify
Coin (route action with immersive PWA behavior), and Stats (title with retained
supporting copy). Visual review of this slice gates broad migration.
