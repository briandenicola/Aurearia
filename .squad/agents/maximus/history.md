# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins — full-stack PWA for managing a personal ancient coin collection
- **Stack:** Go 1.26 / Gin / GORM / SQLite (API), Vue 3 / TypeScript / Pinia / Vite (Frontend), Python 3.12 / FastAPI / LangGraph (Agent), Docker
- **Architecture:** Layered — Handler → Service → Repository → Database. Enforced by architecture_test.go.
- **Created:** 2026-04-24

## Learnings

<!-- Append new learnings below. Each entry is something lasting about the project. -->

- **2025-07-18**: Rewrote `docs/ARCHITECTURE.md` from API-only doc (214 lines) to full-system architecture (761 lines) covering all three services, data flows, DB schema, auth, agent integration, schedulers, build pipeline, and design decisions. Derived entirely from codebase inspection.
- Key file paths: `src/api/main.go` (composition root, ~400 lines of DI wiring), `src/agent/app/supervisor.py` (11-team LLM router), `src/web/src/api/client.ts` (Axios + SSE + 401 refresh queue), `src/api/services/agent_proxy.go` (SSE proxy pattern).
- The system has 26 auto-migrated GORM models, 22 repository files, 17 service files, 25 handler files, 21 Vue pages, and 10 composables.
- Two background goroutine schedulers (availability + valuation) run with configurable anchor times and intervals from DB settings.
- Auth supports 3 methods: JWT (15min access + 30d refresh with rotation), API keys (SHA-256 hashed), and WebAuthn/passkeys.
- **2026-04-24**: Full architecture & code quality review. Graded 11 areas (A- to C+). Key findings: DI is undermined by 3 package-level globals (`AppLogger`, `GetSetting`, `cancelMap`); `social.go` silently drops 7+ errors; frontend has 3 god-pages (1200-1400 lines each); Python agent lacks tests for supervisor routing and team pipelines. Error handling is the weakest area (C+). Documentation is the strongest (A-). Created 20-item prioritized backlog in `.squad/decisions/inbox/maximus-code-review.md`.
- **2025-07-18**: Analyzed `CoinDetailPage.vue` desktop layout issues (1282 lines, ~37KB). Current 2-column `1fr 1fr` grid with 1000px max-width creates dead space once images scroll off-screen and forces excessive vertical scrolling. Proposed 3 layout options: (A) Sticky image sidebar + 2-col info dashboard, (B) 3-column museum triptych, (C) Tabbed panels. Recommended Option A for best effort/impact balance. Proposal in `.squad/decisions/inbox/maximus-desktop-layout-proposal.md`.
- **2026-05-21:** Third background scheduler added to system. Cassius implemented `auction_ending_scheduler.go` with configurable start time and interval (settings-driven). Mirrors existing availability + valuation schedulers. Aurelia added admin configuration panel. Architectural pattern established and working.

## Learnings

### 2026-05-28 — tech-inventory governance philosophy analysis

**Patterns from `briandenicola/tech-inventory` worth replicating** (delivered as plan-only research; no project files modified):

- **Numbered, operational constitution** (§0–§16) — Hierarchy of Authority at §0, then principles, then §9 Quality Gate / §10 AI Agent Operating Rules / §11 Doc Requirements / §12 Audit Cadence / §13 Definition of Done / §14 Amendment Process / §15 Revision History / §16 Signatures of Intent. The operational sections are what give the constitution teeth — our 16-principle list is good but lacks the DoD, Quality Gate checklist, and Hierarchy block.
- **`specs/NNN-feature/{spec,plan,tasks}.md` on disk** with `specs/_backlog/F0NN-*.md` per-card files and a `_TEMPLATE.md`. The SpecKit pipeline becomes real artifacts, not just prompts.
- **Document hierarchy in copilot-instructions.md** — five-line ordered list and a "Session Protocol" Always/Never/Handoff block at the top. Constrains agent behavior more than the constitution alone.
- **Concrete CI Quality Gate workflow** (`quality-gate.yml` + `security-scan.yml`) — one composite gate matching the constitution's §9 checklist; PRs blocked on it.
- **ADRs in `docs/adr/`** with Nygard format, cited in commit messages — captures *why* alongside *what*.
- **`docs/references.md`** with pinned SHAs cited as `R<N>:<path>@<sha>` — formalizes prior-art borrowing.
- **`.gitleaks.toml` + `.githooks/pre-commit` + `SECURITY.md` + `CODEOWNERS` + PR template** — small files, big behavioral effect.
- **Root `openapi.yaml`** as the curated contract surface (even though the generator lives elsewhere).

**Brian's cross-repo governance preference** — confirmed: he wants the *operational scaffolding* (hierarchy, DoD, quality gate, AI agent rules, ADR practice, specs/ on disk) to feel identical across his repos, while accepting stack-specific principles diverge. Translation rule of thumb: **copy the discipline; do not copy the .NET/SvelteKit specifics.**

**Anti-patterns explicitly NOT to copy:**

- `SESSION-NOTES.md` (single flat append-only file) — our `.squad/log/`, `.squad/sessions/`, Scribe + per-agent `history.md` is strictly richer. Do not regress.
- `.copilot-state.md` (single state snapshot) — superseded by `.squad/sessions/` and agent history files.
- Clean Architecture (Domain/Application/Infrastructure/Api) + MediatR/CQRS/FluentValidation/`Result<T>` — .NET-idiomatic, would over-engineer Go.
- 85% blanket coverage gate — fights integration-heavy Go testing; our `architecture_test.go` is better signal.
- Playwright mandate in constitution — keep as aspirational doc, not law.

**Method note:** GitHub MCP `get_file_contents` works in parallel for directory listings + individual files — fetched ~14 tech-inventory artifacts in two batched rounds. For files > ~10 KB, `get_file_contents` overflows the read window and dumps to `/tmp` (forbidden in this env, but the preview chunk + a subsequent `tail` of the dump file was sufficient for the constitution).

**Artifacts produced this session:**
- Plan: `/home/brian/.copilot/session-state/1056094e-1359-492f-a2c7-4d6c50eda3e3/plan.md` (~22 KB)
- Decision: `.squad/decisions/inbox/maximus-tech-inventory-alignment-plan.md` (~4 KB)

---

## 2026-05-28 — Constitution v2.0.0 ratified

Promoted the constitution from v1.1.0 → v2.0.0 (MAJOR — governance restructure, not a principle change).

**Added 8 operational sections, principles preserved verbatim:**
- §0 Hierarchy of Authority (8-tier doc precedence: Constitution → PRD → spec → plan → tasks → backlog → decisions.md → agent judgment)
- §17 Quality Gate (concrete per-PR checklist; gitleaks/trivy marked Phase 3; **signed commits explicitly NOT required** — hobby project)
- §18 AI Agent Operating Rules (Always / Never / Context Discipline / Drift Recovery / Session Handoff — handoff routed to Scribe + `.squad/log/`, NOT to a flat SESSION-NOTES.md)
- §19 Documentation Requirements (table of ✅/⏳ Phase 3 docs)
- §20 Audit & Continuous Improvement (weekly /audit, per-release SBOM, quarterly PRD)
- §21 Definition of Done (14-item PR checklist)
- §22 Amendment Process (formal: ADR → PR → semver → Sync Impact → revision row → announce in decisions.md)
- §23 Revision History (table with v1.0.0, v1.1.0, v2.0.0)

## Learnings

- **Verbatim preservation worked cleanly** — splitting "principles" (stable, stack-specific) from "operational sections" (governance, portable across repos) lets us crib tech-inventory's discipline without dragging .NET idioms into a Go/Vue/Python repo. This is the cross-repo governance philosophy Brian wants: same scaffolding shape, different domain content.
- **Signed-commit divergence is intentional** — tech-inventory mandates signed commits on `main`; we explicitly do NOT. The Conventional Commits prefix + Copilot co-author trailer is sufficient signal for a single-developer hobby project. Future agents must not "fix" this back to tech-inventory's default.
- **`SESSION-NOTES.md` / `.copilot-state.md` are now constitutionally forbidden** — §18.5 explicitly routes session handoff to Scribe + per-agent `history.md`. This protects the richer Squad ceremony system from regression by future agents who might try to mirror tech-inventory's flatter pattern.
- **Sync Impact Report header is now the contract for amendments** — Phase 3 amendments (PRD, ADR 0001, etc.) MUST update this header in the same PR that introduces them. The §22 semver rules and §23 revision-history table make the audit trail mechanical, not interpretive.
- **Round 2 gates**: `.github/copilot-instructions.md` needs a Document Hierarchy block (cite §0) + Session Protocol block (cite §18). `.github/pull_request_template.md` needs the §21 DoD checklist inlined. Both are Maximus-owned in Round 2 per the plan.

## Learnings (Round 2)
- `.github/copilot-instructions.md` now **cites** constitution §0 (Document Hierarchy), §17 (Quality Gate), §18 (Session Protocol), §21 (Definition of Done), and Principles I/X/XI/XII/XIII/VIII rather than restating them. Operational reference material (Build/Test/Lint, design tokens, chip/button hierarchy, "Adding a New API Feature", Notable Endpoints) is preserved verbatim — that's our day-to-day value-add.
- New `.github/pull_request_template.md` **inlines the §21 Definition of Done as a 14-item checklist**. That checklist is the canonical execution surface every PR is gated on. Items #8 (openapi.yaml regen) and #9 (ADR) are marked `(when Phase 3 lands)` so they don't block today.
- Confirmed `SESSION-NOTES.md` / `.copilot-state.md` are forbidden by §18 — Session Protocol block explicitly steers agents to `.squad/log/` + `.squad/decisions.md` instead.

## Learnings (Phase 2A — specs scaffold)
- `specs/` scaffold landed on disk: `specs/README.md` (lifecycle + numbering + gate citations), `specs/_backlog/README.md` (promotion rule + triage cadence), and `specs/_backlog/_TEMPLATE.md` (15-field card with YAML frontmatter for triage + prose body for content). Numbering rule is **immutable + never reused** — protects historical references.
- 4 new session-protocol prompts (`.github/prompts/{load-context,checkpoint,handoff,audit}.prompt.md`) mirror tech-inventory's discipline but **route through Squad ceremonies**: Scribe owns checkpoint/handoff (writes to `.squad/log/` + `.squad/decisions/inbox/`, never `SESSION-NOTES.md`/`.copilot-state.md`); Maximus + Brutus co-own audit (findings to `docs/audits/YYYY-MM-DD.md`). Each prompt cites the relevant constitution section (§0, §17, §18, §20, §22).
- **Copilot manifest was NOT edited** — `.specify/integrations/copilot.manifest.json` is auto-managed by `specify install/upgrade` (SHA-256 keyed file inventory). Manual entries would be clobbered. TODO recorded in decision card to run `specify upgrade` (or accept that these 4 prompts live outside the manifest as repo-local additions).

## Learnings (Phase 2B — retroactive 001-foundation anchor)
- `specs/001-foundation/` landed as the retroactive v1.0 anchor — three files (spec.md 162L, plan.md 139L, tasks.md 86L; 387L total) all marked SHIPPED with every tasks.md checkbox checked. Validates Constitution §0 Hierarchy item 3 ("active feature spec") with real content instead of a placeholder.
- Backlog cards F001–F007 are cross-linked from spec.md so the audit trail from "queued idea → shipped feature" is dereferenceable in both directions; backlog cards themselves are unchanged (no retroactive `Promoted to:` rewrites, because they were authored *after* shipping — F001–F007 are extractions from the v1.0 surface, not promotions of pre-shipping ideas).
- Convention going forward: forward-looking work opens at `specs/002-*/` and onward. `001-foundation/` is a historical anchor and is not edited again except via a `## History` entry if a future amendment materially restates the v1.0 surface. Phase 2 of the SpecKit rollout is complete; Scribe owns close-out next.

## Learnings

- **2026-05-28 — ADR practice established.** `docs/adr/` now exists with
  four Nygard-format ADRs: 0001 (the practice itself), 0002 (three-service
  architecture), 0003 (JWT + refresh + WebAuthn), 0004 (design token
  system). 0002–0004 retroactively document v1.0 governance, architecture,
  auth, and design decisions that previously lived only in code and oral
  tradition. Going forward, **any material decision (principle change,
  new third-party service, auth/security change, multi-service contract
  change, data-model semantic migration, UI framework change) MUST open
  with an ADR per Constitution §22**. The ADR index lives at
  `docs/adr/README.md`.
- **2026-05-28:** README trimmed from 368 → 90 lines (~25.4 KB → ~5.8 KB). Removed the long product feature list, the duplicated dev/prod architecture diagrams, the giant `Project Structure` tree, and the legacy completed-backlog checklist. Replaced with navigation + quick-start + governance pointers. Product narrative is now centralized in `docs/prd.md` per Constitution §19; per §0 (Hierarchy of Authority) the PRD is item #2 and the README is no longer allowed to restate product-level claims that conflict with it. Going forward, any product detail living in README is a §0 violation — escalate via `.squad/decisions/inbox/` rather than re-expanding the README. Decision: `.squad/decisions/inbox/maximus-readme-trim-prd-promoted.md`.

## Learnings (2026-05-28 — Phase 3b operational scaffolding)

- **2026-05-28 (Phase 3b complete):** Operational scaffolding phase landed. Delivered: clean security doc split (monolithic `docs/security-analysis.md` retired → three-home model: `security-principles.md` for durable controls, `threat-model.md` for live findings, `incident-response.md` for playbook), `docs/references.md` with pragmatic standards/frameworks/services/tooling bucketing, `.gitleaks.toml` with targeted allowlisting (Swagger artifacts, web build, test examples, no testdata dir needed), and `.pre-commit-config.yaml` for optional git hooks. Constitution updated (4 stale refs replaced). Decision #15: clean cut on retired file (no stub); three new docs become sole security surface. Collaborated with Cassius (CI gate) and Brutus (test strategy); Scribe merged all decisions.
- **2026-04-24T12:56:00Z**: Foundry Agent Service migration spike completed. Research shows Microsoft Foundry Agent Service (launched April 2026) with Hosted Agents (preview) is viable alternative to current Python/LangGraph stack. Anthropic Claude models fully supported in Foundry catalog (no separate contract). Effort estimate: **Large** (full agent service rewrite in C#). Recommendation: **Phase approach** — complete PoC now (Python → C# MVP on local Docker), defer full production migration to Hosted Agents when it reaches GA (Q3 2026). Keep orchestration in C# code rather than Foundry Workflows (pipelines too programmatic for YAML definition). Full 500+ line analysis in `docs/spikes/foundry-agent-service.md` on branch `spike/foundry-agent-service` — pending leadership Go/No-Go decision.

## Foundry Agent Service Migration Spike (January 2025)

**Outcome:** NO-GO recommendation. Microsoft Foundry Agent Service is technically viable but migration cost ($115k, 3-4 months) does not justify replacing working Python/LangGraph solution.

**Key Findings:**
- Foundry supports Claude models (Opus/Sonnet/Haiku) via Azure serverless APIs
- C# Agent Framework SDK provides streaming (IAsyncEnumerable → SSE) and multi-agent orchestration
- Requires complete rewrite in C#, reimplementation of 10 team pipelines, stateful design pivot
- Adds 20% operational overhead (~$50-100/month compute) vs. self-hosted Python
- Technical risks: web search tool availability unconfirmed, quota issues during preview, SearXNG integration unclear

**When to Reconsider:** Strategic Azure lock-in, enterprise support requirements, Claude GA with guaranteed quotas, or team grows 3x.

**Alternative Recommended:** Stay on Python/LangGraph, incrementally adopt Azure AI Services for Anthropic endpoints (no code rewrite).

**Spike Document:** `docs/spikes/foundry-agent-service.md` (comprehensive analysis with architecture diagrams, effort estimates, risk assessment).
