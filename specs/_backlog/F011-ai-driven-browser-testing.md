---
id: F011
title: "AI-driven exploratory browser testing for runtime UI bugs"
status: promoted           # backlog | triaged | promoted | completed | dropped
priority: P0
effort: L
value: 5
risk: 3
owner: unassigned
created: 2026-05-28
updated: 2026-09-18
promoted_to: specs/360-ai-driven-browser-testing/
---

# F011 — AI-driven exploratory browser testing for runtime UI bugs

**Promoted to**: [specs/360-ai-driven-browser-testing/](../360-ai-driven-browser-testing/)

## Summary

Feature 220/F013 now provides deterministic browser coverage for critical
workflows, but it deliberately does not explore beyond scripted paths or assess
nondeterministic runtime behavior. UI bugs and edge cases such as broken
navigation after login, PWA install flow failures, modal focus traps, mobile
viewport regressions, agent-stream rendering glitches, and dark-mode contrast
failures can still escape those checks. This card tracks an AI-driven
exploratory layer that drives the real SPA in an ephemeral full stack, observes
responses, and reports evidence-backed findings.

## Acceptance Criteria

- [ ] A repeatable test job (local + CI) that boots the full stack (API + Web + Agent) and runs an LLM-driven browser session against it
- [ ] Test session produces a structured report: navigated routes, screenshots, console errors, network failures, accessibility violations, and an LLM-generated triage with severity
- [ ] Findings produce a structured report by default; issue creation requires
      explicit opt-in and is proven with a seeded defect without opening a real
      issue during ordinary tests
- [ ] `task test-ui-explore` target documented in `docs/testing.md`
- [ ] Cost ceiling per run defined (token budget + max steps) so unattended runs can't burn unlimited budget

## Constitution Alignment

- **§17 Quality Gate** — exploratory results are advisory (don't block merge initially); promote to gating once stable
- **§19 Documentation Requirements** — `docs/testing.md` testing-pyramid section gains an "Exploratory" tier
- **Principle IV (Simple Complete Changes)** — browser tests exercise the real user workflow, not just implementation details
- **Principle VII (CI, Supply Chain, and Release Integrity)** — any new MCP server / browser driver pinned to a specific version
- **Principle IX (Automated Enforcement Over Manual Memory)** — high-value workflow checks become repeatable
- **Principle V (Security, Auth, and Privacy by Default)** — exploratory agent runs against a throwaway database; no production data

## Promotion Decisions

- The explorer is provider-agnostic and uses dedicated test credentials rather
  than production application settings.
- Runs are manual plus nightly, advisory, and non-blocking until the active
  specification's stability criteria are met and a separate decision promotes
  them.
- A structured report artifact is the default. GitHub issue creation is
  disabled unless an explicit run input enables it.
- Runs reuse F013 golden fixtures and its critical workflow inventory against
  an ephemeral full stack with no production data.
- Steps, wall time, model calls/tokens, browser actions, and issue creation are
  hard-bounded.
- Evidence includes routes, screenshots, console errors, network failures,
  accessibility findings, and model triage.
- A deterministic seeded-defect test proves detection and the optional
  issue-creation request without opening a real issue during ordinary tests.
- Secrets are excluded from prompts, screenshots, reports, traces, logs, and
  issue bodies.
- New actions and dependencies are pinned, and no merge-blocking gate is added
  before the active specification's stability threshold is met.

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
- 2026-09-18: Promoted to active SpecKit feature
  [`specs/360-ai-driven-browser-testing/`](../360-ai-driven-browser-testing/)
  on the existing `beta` branch; product-owner decisions resolved provider,
  cadence, reporting, isolation, budget, privacy, seeded-defect, pinning, and
  gating scope.
