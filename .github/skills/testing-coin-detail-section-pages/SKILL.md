---
name: testing-coin-detail-section-pages
description: "Test coin detail section pages by keeping the context composable and shell scoped slot synchronized."
---

# Synchronized coin detail fixtures

Section pages can consume the coin from both `useCoinDetailContext()` and the
`CoinDetailSectionPageShell` scoped slot. Mocking only one can leave the script
and template observing different states.

1. Inspect the current page, composable and shell contracts and the nearest
   section-page test (for example `CoinDetailValuationPage.test.ts`).
2. Use a typed shared reactive coin fixture for both the composable mock and
   shell stub's reactive slot value. Respect Vitest mock hoisting; do not access
   uninitialized module bindings from an eagerly evaluated mock factory.
3. Reset the complete fixture and API mock behavior before each test. Do not use
   `as never` or broad casts to disguise an incomplete response contract.
4. Stub the real route parameter shape, and exercise collection/wishlist/sold,
   missing coin, loading, error and success states applicable to the page.
5. Use `nextTick` for Vue updates and `flushPromises` for asynchronous effects as
   needed; do not assume all mount hooks run as deferred microtasks.
6. Assert behavior and exact payloads rather than snapshots alone; a slot stub
   must still exercise the same data boundary as the real shell.

Run focused tests for feedback and authorized `task check:web` for completion.
