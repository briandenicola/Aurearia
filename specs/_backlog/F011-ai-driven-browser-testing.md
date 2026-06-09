---
id: F011
title: "AI-driven exploratory browser testing for runtime UI bugs"
status: triaged            # backlog | triaged | promoted | dropped
priority: P0
effort: L
value: 5
risk: 3
owner: unassigned
created: 2026-05-28
updated: 2026-06-09
---

# F011 — AI-driven exploratory browser testing for runtime UI bugs

## Summary

We have unit tests (Go 118, Vue 61, Python 35) but **zero browser-level coverage**. Runtime UI bugs and edge cases (broken nav after login, PWA install flow, modal focus traps, mobile viewport regressions, agent-stream rendering glitches, dark-mode contrast failures) currently surface only when Brian uses the app. This card tracks introducing an LLM-driven exploratory testing layer that can drive the real SPA, observe responses, and report findings as triaged issues.

## Acceptance Criteria

- [ ] A repeatable test job (local + CI) that boots the full stack (API + Web + Agent) and runs an LLM-driven browser session against it
- [ ] Test session produces a structured report: navigated routes, screenshots, console errors, network failures, accessibility violations, and an LLM-generated triage with severity
- [ ] At least one finding (real or seeded) flows into a GitHub issue with reproduction steps
- [ ] `task test-ui-explore` target documented in `docs/testing.md`
- [ ] Cost ceiling per run defined (token budget + max steps) so unattended runs can't burn unlimited budget

## Constitution Alignment

- **§17 Quality Gate** — exploratory results are advisory (don't block merge initially); promote to gating once stable
- **§19 Documentation Requirements** — `docs/testing.md` testing-pyramid section gains an "Exploratory" tier
- **Principle IV (Simple Complete Changes)** — browser tests exercise the real user workflow, not just implementation details
- **Principle VII (CI, Supply Chain, and Release Integrity)** — any new MCP server / browser driver pinned to a specific version
- **Principle IX (Automated Enforcement Over Manual Memory)** — high-value workflow checks become repeatable
- **Principle V (Security, Auth, and Privacy by Default)** — exploratory agent runs against a throwaway database; no production data

## Approach Options (to be decided during spec phase)

1. **Playwright MCP + vision-capable LLM** (recommended) — model drives a real Chromium via MCP, sees screenshots, decides next action. Closest analog: Anthropic Computer Use, OpenAI Operator. Highest signal, highest cost.
2. **Playwright codegen + LLM test author** — LLM writes traditional Playwright specs from PRD/spec files; humans run them. Cheaper, deterministic, but misses emergent bugs.
3. **Static LLM review of Vue components** — model reads `.vue` files and flags likely runtime bugs (null prop access, missing loading states, focus traps). Already partially covered by Brutus on demand.
4. **Sentry + LLM triage on production errors** — capture real runtime errors and have an LLM cluster + summarize them. Requires Sentry (or equivalent) wiring first.

Coordinator recommendation captured 2026-05-28: pursue (1) as primary, keep (2) as a fallback if MCP integration proves flaky, layer (4) later if Brian wants production telemetry.

## Open Questions

- [ ] Which LLM provider for the exploratory driver — reuse the user's configured Anthropic key, or require a separate "tester" provider so test runs don't share rate limits with production agent traffic?
- [ ] Run cadence — pre-merge on PRs touching `src/web/`, nightly, manual `task` only, or some mix?
- [ ] How are findings deduplicated across runs so we don't open the same issue every night?
- [ ] Do we seed deterministic test accounts/data via fixtures, or let the agent create its own via the UI on each run?
- [ ] Token-budget enforcement: hard cap per run, soft cap with continuation prompt, or both?

## Roadmap role

This card is now part of the Agentic Excellence Roadmap. It should follow F013
so the exploratory browser agent has a golden fixture collection and critical
workflow list to exercise.

## Notes

- Brian asked 2026-05-28: "is there a way to use an LLM to find or check on runtime bugs or edge cases in the UI?" — coordinator presented 4 ranked options in Direct Mode; user confirmed interest but wants to wait for #163.
- Brutus originally filed a narrower "browser E2E smoke tests" proposal in Phase 3b history; this card supersedes/upgrades that scope.
- Playwright MCP server: https://github.com/microsoft/playwright-mcp (pin to SHA when adopted).
- Anthropic Computer Use docs to reference for action-space design.

## Dependencies

- **Depends on:** F013 defining the golden fixture collection and critical
  workflows.
- Coordinates with: `docs/testing.md` (Brutus owns; will gain new tier).

## References

- Issue #163 (audit findings will scope which UI flows are highest-priority)
- F013 — Harden critical collection workflows
- `docs/testing.md` — current testing pyramid; F011 adds the exploratory tier
- `.squad/decisions.md` entry #18 (this card's deferral and tracking decision)
- `docs/backlog/agentic-excellence-roadmap-2026-06-09.md`

## History

- 2026-05-28: Created. Status `deferred` pending #163 completion. Coordinator recommendation: AI-driven (Playwright MCP + vision model). Brutus to own spec drafting on promotion.
- 2026-06-09: Moved to `triaged` / `P0` as part of the Agentic Excellence Roadmap. F013 should define the golden workflows before this is promoted.
