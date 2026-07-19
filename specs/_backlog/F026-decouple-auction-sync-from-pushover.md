---
id: F026
title: "Auto-sync watchlists for all configured users, not just Pushover users"
status: backlog
priority: P0
effort: S
value: 5
risk: 2
owner: unassigned
created: 2026-07-19
updated: 2026-07-19
---

# F026 — Auto-sync watchlists for all configured users, not just Pushover users

## Summary

`UserRepository.ListAuctionWatchDigestEligible()` (`user_repository.go:86`) is
the query the alert scheduler and digest scheduler use to decide which users'
watchlists to refresh in the background. It requires
`pushover_enabled = true AND pushover_user_key <> ''` — i.e. background sync
eligibility is gated on notification preferences, not on whether the user has
auction credentials configured at all. A user without Pushover set up gets
zero automatic background refresh; their `CurrentBid`/status only update when
they manually click Sync. This was identified as a likely root cause of
"bid updates not reflected" during the issue #482 audit, independent of and in
addition to the CNG field-mapping bug that commit `0943a56` fixed.

No external credentials are needed to fix this — it's a query change, testable
with existing unit-test patterns (create users with/without Pushover
configured, assert both get synced).

## Acceptance criteria

- [x] Background watchlist sync runs for any user with NumisBids and/or CNG
      credentials configured, regardless of Pushover configuration.
- [x] Pushover configuration continues to gate only whether a *notification* is
      sent (price alerts, bid reminders, ending-soon, digest) — not whether
      the underlying sync happens.
- [x] Existing behavior for users who do have Pushover configured is
      unchanged.
- [x] Regression test asserts a non-Pushover user's lots still get resynced by
      the scheduled path.

## Constitution alignment

- Principle I (Clear Layered Architecture) — fix stays in
  `repository/user_repository.go` (query) and callers in
  `services/auction_alert_scheduler.go` /
  `services/auction_watch_bid_digest_scheduler.go` (naming/semantics only, the
  sync call itself doesn't change).
- Principle IV (Simple Complete Changes) — this is a narrow, well-scoped fix;
  don't bundle in F027 (notification architecture) even though they were found
  together.

## Open questions

- [x] Should the query be renamed away from `ListAuctionWatchDigestEligible`
      once it's no longer specifically about digest/Pushover eligibility (e.g.
      `ListUsersWithAuctionCredentials`)? Renaming touches call sites in both
      schedulers. — Yes, renamed to `ListUsersWithAuctionCredentials`.

## Notes

Implemented by dropping the `pushover_enabled`/`pushover_user_key` WHERE
clause from the repository query (renamed
`ListAuctionWatchDigestEligible` → `ListUsersWithAuctionCredentials`) so it
selects any user with NumisBids and/or CNG credentials configured. Both
`auction_alert_scheduler.go` and `auction_watch_bid_digest_scheduler.go` call
sites were updated to the sync service's renamed
`SyncAllConfiguredUsers()` (was `SyncDigestEligibleUsers()`), which makes the
new, broader semantics explicit at the call site. Pushover configuration is
untouched as a gate on notification delivery — see F027 for the
still-outstanding "no Pushover configured means no notification at all"
gap, now fixed there as well.

## History

- 2026-07-19: created (status: backlog) — one of two Pushover-coupling issues
  found during the issue #482 audit; split from F027 (in-app notifications)
  since they're independently fixable and independently valuable.
- 2026-07-19: implemented and unit-tested (repository, both scheduler call
  sites); all acceptance criteria met. Status left at `backlog` pending Lead
  triage per `_backlog/README.md` — implementation does not self-advance
  status.
