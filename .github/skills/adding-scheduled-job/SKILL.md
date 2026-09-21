---
name: adding-scheduled-job
description: "Add or change a Go background scheduler with validated settings, lifecycle, idempotency, and run evidence."
---

# Scheduled Go work

Read the approved scope, existing scheduler/service/repository patterns and
`settings_service.go`. Reuse actual constructors, loggers and interfaces; old
examples and line numbers are not a current API contract.

1. Keep business logic HTTP-agnostic in services, queries in repositories and
   standard-library-only models. Inject dependencies through constructors.
2. Put validated enable, anchor time and cadence settings/defaults in the existing
   settings mechanism. Confirm units and time-zone/DST behavior; avoid silent
   invalid-input defaults, division by zero and hardcoded operational addresses.
3. Support cancellation while waiting and working, and idempotent stop (for
   channel shutdown, follow the existing `sync.Once` pattern).
4. Wire dependencies and routes before starting workers in the composition root;
   integrate shutdown rather than merely launching an unowned goroutine.
5. Define retry, overlap and restart behavior. In-memory deduplication resets
   on restart; use durable state where the approved contract needs persistent
   idempotency. Group notifications by owner and preserve ownership boundaries.
6. Observe start, finish, counts, duration and failures without leaking secrets.
   Check repository, finalization and notification errors; never discard them.
7. For approved manual triggers/history, use existing authenticated admin routes,
   typed handler contracts, pagination validation and Swagger annotations.
   Clarify whether manual runs honor the enable flag. Do not invent a new queue.
8. If adding run models/migrations, prove representative upgrade/recovery safety.
   Preserve custom timestamps and transactions; retention is an explicit policy.
9. Test disabled jobs, schedule boundaries, cancellation/double-stop, overlap,
   restart/retry idempotency, ownership, manual triggers and failed persistence.
   Use deterministic clock/HTTP seams rather than live external services.

Update scheduler/config documentation and use authorized `task check:go`,
affected OpenAPI checks and runner race evidence. Actual scheduler execution or
external notifications require separate applicable authorization.
Use the `adding-admin-schedule` skill for UI wiring and accepted input parity.
