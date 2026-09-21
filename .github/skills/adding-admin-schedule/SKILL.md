---
name: adding-admin-schedule
description: "Add or change an admin scheduler panel, manual trigger, and run history using existing Vue patterns."
---

# Admin schedule panels

Read the selected scope and existing `AdminSchedulesSection.vue`,
`useAdminConfig.ts`, `AdminPage.vue`, API client and backend settings contract.
Do not copy stale template prop names or invent settings the backend ignores.

1. Confirm exact setting keys, defaults, interval units/ranges and time semantics.
   The common `{Feature}CheckEnabled/CheckStartTime/CheckInterval` names are an
   example, not authority to create a new contract. Match all API validation paths.
2. Reuse the nearest panel's global form/button/toggle classes and theme tokens.
   Keep enable, daily anchor, interval, save state and errors understandable.
3. Wire settings state/defaults and save feedback through the existing composable
   and parent. Clear obsolete feedback and dispose timers on unmount.
4. If approved, add manual-trigger and history API client functions with typed
   responses. Keep run history in the existing component pattern, with loading,
   empty, failure, pagination and expanded-result states.
5. Surface load/trigger/detail failures visibly. A failed request is not an empty
   successful run history. Prevent duplicate submits and stale asynchronous results.
6. Preserve compact mobile pagination and usable interactive table controls.
   Read-only rows must not impersonate buttons.
7. Test panel-to-parent wiring, success/failure/loading states, accepted input
   boundaries and manual-trigger/history behavior against the backend contract.

Run authorized `task check:web` plus affected browser checks. Cross-layer setting
changes also require applicable Go validation. Do not install tools implicitly.
The `adding-scheduled-job` native skill covers backend lifecycle/run logging.
