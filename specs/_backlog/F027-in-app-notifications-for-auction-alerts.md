---
id: F027
title: "Wire auction price alerts, bid reminders, and ending alerts into the in-app inbox"
status: backlog
priority: P1
effort: M
value: 4
risk: 2
owner: unassigned
created: 2026-07-19
updated: 2026-07-19
---

# F027 — Wire auction price alerts, bid reminders, and ending alerts into the in-app inbox

## Summary

`docs/features/auction-tracking.md` states bid reminders' "Notifications appear
in in-app inbox." They don't. `auction_alert_service.go` (price alerts, bid
reminders) and `auction_ending_scheduler.go` (ending-soon alerts) call
`pushoverSvc.SendNotification` directly and only that — there is no call
anywhere into the existing `NotificationService` / `Notification` model that
already backs the in-app inbox for wishlist availability, social events,
coin-of-the-day, AI jobs, valuation runs, and API key rotation. A user without
Pushover configured gets no notification at all for any auction event, contrary
to the docs, and this was identified as a likely root cause of "alerts
unreliable/missing" during the issue #482 audit.

No external credentials are needed to fix this — it's wiring an existing
in-app system into three call sites; testable by asserting a `Notification`
row is created, no Pushover call required for that part.

## Acceptance criteria

- [x] Price alerts, bid reminders, and ending-soon alerts each create an
      in-app `Notification` record (new `type` values, e.g.
      `auction_price_alert`, `auction_bid_reminder`, `auction_ending_soon`)
      in addition to (not instead of) the existing Pushover send.
- [x] In-app notification fires regardless of whether Pushover is configured —
      Pushover failure/absence must not suppress the in-app one.
- [x] `NotificationService` gets the equivalent of `NotifyWishlistUnavailable`
      etc. for these three auction event types, following the same pattern.
- [x] Existing Pushover-only behavior for users who have it configured is
      unchanged (they get both).
- [x] Docs (`auction-tracking.md`) claim matches actual behavior once this
      lands (already true in wording — just needs the implementation to catch
      up).

## Constitution alignment

- Principle I (Clear Layered Architecture) — new `NotificationService` methods
  follow the existing pattern exactly; callers in
  `auction_alert_service.go`/`auction_ending_scheduler.go` gain one additional
  call each, no restructuring.
- Principle III (Strict Types and Explicit Contracts) — new `Notification.Type`
  values are explicit constants, not ad hoc strings.
- Principle VI (Consistent User Experience) — auction notifications should
  behave like every other in-app notification type already in the inbox.

## Open questions

- [x] Should in-app notification failure (e.g. DB error) affect the run's
      success/failure reporting the way Pushover failures currently do in
      `auction_alert_service.go`'s `notifyPriceAlert`/`notifyBidReminder`?
      — No. The alert/reminder is claimed (`MarkTriggeredIfPending`/
      `MarkNotifiedIfPending`) *before* notifying; a DB error creating the
      `Notification` row is logged only, matching every other
      `NotifyXxx` method in `NotificationService` (none of them propagate
      DB errors to the caller). The run only fails now on a genuine
      repository error while claiming, not on notification delivery.

## Notes

Implementation ended up going one step further than "add a call in addition
to the existing Pushover send": Pushover delivery for price alerts and bid
reminders was *moved inside* the new `NotificationService.NotifyAuctionPriceAlert`
/ `NotifyAuctionBidReminder` methods (fire-and-forget `go s.sendPushover(...)`,
matching the pattern every other `NotifyXxx` method already used) rather than
kept as a separate direct call in `auction_alert_service.go`. This collapses
two call sites into one and means Pushover failure can never affect
claiming/idempotency, which is the actual fix for "alerts unreliable/missing":
previously a Pushover error left the alert/reminder unclaimed and it would
never fire again even after the price re-crossed, or would double-notify on
retry. `auction_ending_scheduler.go`'s ending-soon path keeps its own
synchronous Pushover send (it manages its own per-day dedup state that the
in-app side doesn't need to duplicate) but now also calls
`NotificationService.NotifyAuctionEndingSoon` for the in-app record.

`AuctionAlertEvaluator` and `AuctionEndingScheduler` both gained a
`*NotificationService` constructor dependency (`main.go` updated). Existing
tests asserting the old "Pushover failure leaves the alert retryable /
fails the run" behavior were intentionally rewritten to assert the opposite,
since that was the bug this card fixes, not a behavior to preserve — see
`TestAuctionAlertEvaluatorClaimsAlertsEvenWithoutPushover` and
`TestAuctionAlertSchedulerSucceedsWithoutPushover` in
`auction_alert_service_test.go`. New `notification_service_test.go` covers
`NotifyAuctionPriceAlert`/`NotifyAuctionBidReminder`/`NotifyAuctionEndingSoon`
directly. Note: no test in this codebase (before or after this change)
deterministically asserts the fire-and-forget Pushover goroutine's payload for
these methods — that path isn't testable without adding synchronization
machinery not used elsewhere in the codebase, so coverage instead targets the
synchronous, reliable in-app `Notification` record.

## History

- 2026-07-19: created (status: backlog) — one of two Pushover-coupling issues
  found during the issue #482 audit; split from F026 (sync eligibility) since
  they're independently fixable and independently valuable.
- 2026-07-19: implemented and unit-tested (`NotificationService` methods,
  `AuctionAlertEvaluator` and `AuctionEndingScheduler` wiring, `main.go`
  constructor updates); all acceptance criteria and the open question
  resolved. Status left at `backlog` pending Lead triage per
  `_backlog/README.md` — implementation does not self-advance status.
