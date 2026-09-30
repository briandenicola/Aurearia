# Beta to main release QC audit (2026-09-30)

- Candidate: `beta` @ `d4f77459` against `origin/main` @ `57f00398` (27 commits, 217 files, +7453/-1565).
- Owner authorization: owner chose "run the quality audit first, then open the PR".
- Gates: `task check` on `d4f77459`, EXIT=0 (delivery 91/91, Go 111 packages ok, web lint/type-check/207 files/build, agent ruff + 766 pytest, OpenAPI no drift).
- Independent auditor: aurearia-reviewer `d17c1f9f`, read-only, skill `aurearia-software-qc-audit`.
- Verdict: **PASS**, medium confidence, no blockers. Not merge or release authorization.
- Applied before PR: follow-up 8 (stale #784 status row in `docs/audits/2026-09-29-open-issue-plans.md`), within the auditor's carve-out.
- Open follow-ups (non-blocking):
  1. Invalid stored schedule zone logs a warning every evaluation; warn once.
  2. Invalid `CoinOfDayTimezone` should fall through to `ScheduleTimezone`.
  3. Interval jobs now run on a daily grid; confirm admin interval options avoid non-whole-day values above 24h.
  4. Coin Agent chat defaults to direct dealer search (#769); call out in release notes.
  5. Go should check http/https on comparable listing URLs.
  6. Price-range lookup can add up to 25s; check server write timeout and Deep Analysis budget.
  7. Confirm CI invokes `task check:go` (gofmt gate).
  9. Comparable listing with unsafe URL renders price without a title.
  10. #784 section G items (65 literals, `text-xs` fold, guard gaps).
- Unverified by auditor: parts of `41-web-vue.diff`, all of `42-web-css.diff`, tail of agent/test/docs diffs, `.squad/` files.
- Next action: owner reviews the beta to main PR; merge needs separate explicit owner approval.
