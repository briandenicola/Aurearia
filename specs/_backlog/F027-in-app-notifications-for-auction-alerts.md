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

- [ ] Price alerts, bid reminders, and ending-soon alerts each create an
      in-app `Notification` record (new `type` values, e.g.
      `auction_price_alert`, `auction_bid_reminder`, `auction_ending_soon`)
      in addition to (not instead of) the existing Pushover send.
- [ ] In-app notification fires regardless of whether Pushover is configured —
      Pushover failure/absence must not suppress the in-app one.
- [ ] `NotificationService` gets the equivalent of `NotifyWishlistUnavailable`
      etc. for these three auction event types, following the same pattern.
- [ ] Existing Pushover-only behavior for users who have it configured is
      unchanged (they get both).
- [ ] Docs (`auction-tracking.md`) claim matches actual behavior once this
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

- [ ] Should in-app notification failure (e.g. DB error) affect the run's
      success/failure reporting the way Pushover failures currently do in
      `auction_alert_service.go`'s `notifyPriceAlert`/`notifyBidReminder`?

## History

- 2026-07-19: created (status: backlog) — one of two Pushover-coupling issues
  found during the issue #482 audit; split from F026 (sync eligibility) since
  they're independently fixable and independently valuable.
