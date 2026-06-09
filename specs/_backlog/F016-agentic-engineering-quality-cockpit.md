---
id: F016
title: "Create agentic engineering quality cockpit"
status: backlog
priority: P2
effort: L
value: 4
risk: 3
owner: unassigned
created: 2026-06-09
updated: 2026-06-09
---

# F016 — Create agentic engineering quality cockpit

## Summary

Create a repository health surface that helps humans and agents keep the app
simple, tested, and reliable. It should summarize workflow coverage, flaky
tests, complexity hotspots, large files, untyped maps/casts, stale specs/ADRs,
and whether each PR satisfied the Simple Complete Changes check.

## Acceptance criteria

- [ ] A repeatable command produces a repository health report with critical
      workflow coverage, test failures/flakes, complexity hotspots, and stale
      governance artifacts.
- [ ] PR template or automation records whether the real workflow was tested,
      sibling paths were checked, and the change stayed proportional.
- [ ] Report highlights files/components that exceed agreed complexity or size
      thresholds and links them to backlog/refactor candidates.
- [ ] Agent review roles are mapped to checks: architecture, backend, frontend,
      tests, security, and runtime/CI health.
- [ ] Dashboard/report output is deterministic enough to compare over time.

## Constitution alignment

- Principle IV (Simple Complete Changes) — tracks proportionality and workflow
  proof.
- Principle VII (CI, Supply Chain, and Release Integrity) — integrates with CI
  health without hiding failures.
- Principle VIII (Documented Decisions) — reports stale or conflicting specs,
  ADRs, and decisions.
- Principle IX (Automated Enforcement Over Manual Memory) — automates repeatable
  quality checks.
- §17 Quality Gate, §21 Definition of Done.

## Open questions

- [ ] Should the first version be a Markdown report, GitHub Actions summary,
      local web page, or admin-only app page?
- [ ] Which complexity thresholds should be advisory versus blocking?
- [ ] Should this use existing tools only at first, or add a dedicated analyzer?

## Notes

This is not a vanity dashboard. It should make regressions, drift, and
over-complexity visible early enough that agents can act on them before users
hit bugs.

## History

- 2026-06-09: created (status: backlog).
