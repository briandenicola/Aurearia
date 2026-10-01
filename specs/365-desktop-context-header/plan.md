# Implementation Plan: Desktop Context Header

## Architecture

`App.vue` owns stable desktop-only DOM targets for a title and page actions.
`DesktopPageContext.vue` uses deferred Vue Teleports into those targets. The
component renders nothing in PWA mode, so pages keep their existing mobile
headers. Teleported content unmounts with its page, avoiding a global registry
and explicit cleanup state.

## Slices

1. Add the shell hosts, responsive truncation, action divider, and navigation
   cleanup tests.
2. Migrate Wishlist, Followers, Identify Coin, and Stats as representative page
   types and verify desktop/PWA behavior.
3. Review the first slice visually at representative desktop widths.
4. Inventory and migrate the remaining standalone page headers, including
   dynamic titles and stateful actions.
5. Add overflow handling when the migrated action inventory proves it is needed.
6. Run full web and affected browser/mobile verification, independent review,
   and owner acceptance before any merge or deployment.

## Risks and Controls

- **Stale page context**: component-owned Teleports unmount with the route;
  navigation-away tests prove cleanup.
- **Same-tree target timing**: use Vue 3.5 deferred Teleports and tamper-test the
  requirement.
- **PWA regression**: render no Teleports in PWA mode and keep local headers.
- **Action behavior drift**: page components retain their existing handlers and
  reactive state.
- **Crowded desktop bar**: left content is shrinkable/truncated; broad migration
  will determine whether a More actions menu is required.
