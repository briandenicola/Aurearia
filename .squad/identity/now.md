---
updated_at: 2026-09-29
focus_area: Open issue batch on beta (issues #766-#784)
owner: Copilot CLI implementation owner; repository owner accepts results
work_branch: beta
baseline_commit: a363e4aa566f621ed4361de17088bf6282d1016c
work_artifact: https://github.com/briandenicola/Aurearia/issues/768
tasks_artifact: https://github.com/briandenicola/Aurearia/issues/768
---

# Current Work

The owner is working through the open issues on `beta` before any merge to
`main`. Completed and pushed: typography fixes, #770/#772/#769 (dealer listing
status, rate-limit back-off, direct dealer search). #768 (scheduler start time
and time zone) is implemented and verified in the commit that updates this file.

## #768 result

- All schedulers share `services/schedule_timing.go`: settings are re-read at
  least once a minute, start times are read in the new `ScheduleTimezone`
  setting (Coin of the Day may override), and interval jobs follow the start-time
  slot grid instead of "last run + interval".
- Admin > Schedules has a shared time-zone picker and a run summary per job.
- Evidence: `go build`/`go vet`/`go test ./...` on Go 1.26.6 and
  `task check:web` passed. `task check:go` could not start because the local
  Go is 1.26.1 with `GOTOOLCHAIN=local`; updating it needs owner approval.

## Next Action

Continue the issue groups: C (#776, #775, #777), F (#780, #781, #783),
E (#774, #771, #779), D (#766, needs a spec) and #784. Pending owner approval:
`task setup:agent` (ruff 0.16.8 lock mismatch) and the local Go toolchain update.
No deployment or release is authorized.