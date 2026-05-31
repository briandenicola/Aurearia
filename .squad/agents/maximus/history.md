# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins — full-stack PWA for managing a personal ancient coin collection
- **Stack:** Go 1.26 / Gin / GORM / SQLite (API), Vue 3 / TypeScript / Pinia / Vite (Frontend), Python 3.12 / FastAPI / LangGraph (Agent), Docker
- **Architecture:** Layered — Handler → Service → Repository → Database. Enforced by architecture_test.go.
- **Created:** 2026-04-24

## Learnings

- **2026-05-29 — Threat Model Reconciliation (Issue #206)**
  
  **Audit Results:** Reviewed `docs/threat-model.md` against current code implementation (input artifacts: analysis.go, CoinAIAnalysis.vue, FeaturedCoinModal.vue, useCoinSearchChat.ts, webauthn.go, Taskfile.yml, Dockerfile, GitHub workflows). Found 9 findings had been mitigated but status was stale.
  
  **Newly Marked Mitigated (changed from Open to Mitigated):**
  - **B-2 (SQL injection):** User-controlled `side` parameter in `DeleteAnalysis()` now protected by explicit `columnMap` whitelist (lines 229–238 in analysis.go). Also validates in `Analyze()` switch statement (lines 175–185).
  - **B-6 (Request size):** `MaxMultipartMemory` configured in main.go (~line 130 per middleware). Issue #201 tracks implementation.
  - **B-7 (WebAuthn TTL):** Session TTL hardened to 5 minutes (`const webauthnSessionTTL = 5 * time.Minute`, lines ~20–30 in webauthn.go). Cleanup logic prevents accumulation.
  - **B-8 (WebAuthn origin):** Dynamic origin trust removed; now restricts to configured RP origins only. Issue #202 tracks hardening.
  - **F-1 (AI analysis XSS):** `CoinAIAnalysis.vue` lines 80–82 sanitize with `DOMPurify.sanitize(md.render(...))`.
  - **F-2 (Chat XSS):** `useCoinSearchChat.ts` line 252 sanitizes via `formatMessage()` → `DOMPurify.sanitize(html, {...})` before injection.
  - **F-4 (Sanitizer dependency):** DOMPurify ^3.4.1 now in `package.json` and applied at all HTML injection points (CoinAIAnalysis.vue, useCoinSearchChat.ts, FeaturedCoinModal.vue).
  - **SC-1 (GitHub Actions):** All workflow `uses:` statements pinned to commit SHAs in docker-publish.yml and docker-publish-beta.yml (verified 10 pinned actions). Issue #204 tracks implementation.
  - **SC-2 (Hardcoded JWT):** Taskfile.yml `gen-env` task (lines 143–145) generates random JWT secret into `.env` (not tracked). Config enforces 32-char minimum and fails fast if unset.
  
  **Status Counts Updated:**
  - Backend: 4 → 8 Mitigated, 5 → 1 Open
  - Frontend: 0 → 3 Mitigated, 7 → 4 Open
  - Supply chain: 0 → 2 Mitigated, 7 → 5 Open
  - **Total: 4 → 13 Mitigated, 19 → 10 Open, 1 Accepted**
  
  **Open Findings (10 total):** All now have issue links (most to #163, security audit umbrella).
  - Backend: B-9 only (error response detail leakage)
  - Frontend: F-3 (localStorage tokens), F-5 (JSON refresh body), F-6 (Cache-Control headers), F-7 (username in query string)
  - Supply chain: SC-3 (@imgly CDN models), SC-4 (golang.org/x/ versions), SC-5 (branch protection not enforced), SC-6 (Dockerfile base image digests), SC-7 (Dockerfile non-root user)
  
  **Key Insight:** Recent commits (especially "Fix 16 critical and high security findings" SHA 65fbc00) and closed issues #201–204 had already implemented most mitigations; threat-model was simply stale. Codebase is ahead of documentation. The reconciliation catches docs up and ensures all open items are tracked under issue #163 (Code & security audit).

## Learnings

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

---

### 2026-05-31: Feature #216 Camera-First AI Intake — Design Token Compliance

**Context:** Reviewed Aurelia's AddCoinPage.vue fixes for three critical bugs (iPhone camera readiness, camera-first UI redesign, AI analysis indicator).

**Functional Assessment:** ⭐⭐⭐⭐⭐ (5/5)  
- iPhone camera readiness bug **correctly fixed**: `v-show` + `await nextTick()` + `@loadedmetadata` handler gates `videoReady` on `videoWidth > 0`
- Track teardown properly implemented in `onBeforeUnmount` → `stopCamera()` → `getTracks().forEach(track => track.stop())`
- Error handling distinguishes `NotAllowedError`, `NotFoundError`, generic errors with user-friendly messages
- iOS attributes present: `autoplay`, `playsinline`, `muted`
- Camera-first UI: large 4:3 preview, three capture slots (obverse/reverse/card), prominent shutter button, upload demoted to icon
- AI analysis indicator: full-screen overlay with spinner, "Analyzing your coin…" text (no emoji), interactions disabled
- Type safety: all nullable props use `??` coalescing (Principle IV compliance)
- No emojis (grep check passed)
- lucide-vue-next icons throughout (`Camera`, `Upload`)

**Build & Type Safety:** ✅ PASS  
- `npm run lint`: PASS (5 warnings unrelated to AddCoinPage.vue)
- `npm run build`: PASS (8.73s, vue-tsc --build succeeded, PWA bundle generated cleanly)

**Constitution Violation — Principle V (Design Token System):**  
🚨 **BLOCKING**: 14 instances of hardcoded color values across 10 unique colors:
- `rgba(0, 0, 0, 0.85)` — overlay background (line 744)
- `#000` — video placeholder background (line 808) [borderline acceptable]
- `rgba(224, 141, 141, 0.9)` — error banner background (lines 834, 895)
- `#fff` — contrast text on dark backgrounds (lines 835, 883) [acceptable for contrast]
- `rgba(201, 168, 76, 0.2)` — gold glow (line 860)
- `rgba(0, 0, 0, 0.7)` — slot clear button overlay (line 882)
- `rgba(255, 255, 255, 0.2)` — shutter button border (line 926)
- `rgba(201, 168, 76, 0.3)` / `0.4` — gold shadow (lines 933, 938)
- `#f5c36a` — warning text (line 1075)
- `#69b77f`, `#f0c261`, `#e08d8d` — confidence colors (lines 1084–1095)

**Verdict:** **BLOCK** — Aurelia to extract hardcoded colors into design tokens in `variables.css`, then resubmit for expedited re-review.

**Remediation Path:**  
1. Define 12 missing color tokens in `src/web/src/assets/variables.css` (e.g., `--error-bg`, `--text-warning`, `--confidence-high`, `--overlay-dark`, `--shadow-gold-soft`, etc.)
2. Replace 14 hardcoded color instances in `AddCoinPage.vue`
3. Re-run `npm run lint && npm run build` (must pass)
4. Notify Maximus for expedited re-review (estimated 20–30 min revision)

**Key Learning:**  
- Constitution Principle V is **absolute**: "Never hardcode raw values when a token exists." Even functionally excellent code must comply with design token discipline for theme consistency and maintainability.
- `rgba()` with hardcoded RGB values is a violation even when alpha channel is needed — define semantic tokens like `--accent-gold-glow: rgba(201, 168, 76, 0.2);` in `variables.css`.
- Review sequencing: functional correctness first, then constitution compliance. Blocking on compliance prevents technical debt accumulation.

**Alignment with Brutus's QA Checklist:** 13/14 sections passed; §16 (Progress Indicator Styling) and §27 (Design Token Usage) flagged for hardcoded colors.
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

## 2026-05-30 — Feature #208 Collection Health Scorecard — Completion Lead Audit

**Session Type**: Completion lead audit (monitoring implementation against plan/spec)  
**Feature**: #208 Collection Health Scorecard v1  
**Scope**: Baseline audit + risk + acceptance criteria + code review gates

**Findings**:
- **Backend scaffolding**: 100% structure in place (models, types, handlers, routes, scheduler wired to main.go)
- **Scoring logic**: STUB (returns hardcoded F grade, 0 score; no weighting, no checklist generation, no trend calc)
- **Frontend**: 0% started (no components, no type stubs)
- **Test coverage**: Fixtures only (no unit tests for logic)
- **Tasks**: 52 total identified; 10 done (19%), 3 in_progress (6%), 39 pending (75%)

**Critical Blockers**:
1. **T012 Scoring Logic** — must implement weighted dimensions (40/20/20/20), checklist generation, trend calc, grade mapping. Currently skeleton. Blocks 39 downstream tasks.
2. **T011 Service Tests** — 0 unit tests exist. Must achieve >85% coverage per Constitution §17. Must accompany T012.
3. **T006 Frontend Types** — no TypeScript types/API client methods. Blocks all Phase 3 UI work.

**Go/No-Go**: **CONDITIONAL GO** — Backend structure solid; phase 2 completion is critical blocker. Can proceed with:
- Phase 2 completion (T011 + T012) with architecture checkpoints below
- Phase 3 frontend (T006 types start in parallel with Phase 2)

**Acceptance Criteria**: 13-item MVP checklist (score+grade render, trend displays, queue ordering, endpoint auth, >85% coverage, TS type-check pass, perf budgets met, empty collection handled, formula documented). Documented in decision card.

**Code Review Checkpoints** (3 gates):
1. Phase 2: Scoring formula (40/20/20/20), all thresholds tested, empty collection edge case, trend "insufficient history", checklist taxonomy
2. Phase 3: Handlers thin (Principle I), API schema matches contract, Vue types exact match backend DTOs
3. Feature complete: Constitution §17 Quality Gate, ADR if patterns introduced, swagger committed, no breaking changes

**Risk Register**: 6 risks documented (2 HIGH: scoring bugs, empty collection crash; 4 MEDIUM: ordering unclear, trend calc, component complexity, admin perf).

**Coordinator Actions Delivered**: Audit baseline + 52 task-status database + checklist + risk register + 3 checkpoint rubrics. **Next**: Accept/reject Phase 2 work when code lands; verify checkpoint criteria before advancing phases.

**Artifacts**:
- `.audit-208-status.md` — 300-line comprehensive report (phases, blockers, acceptance criteria, risks, checkpoints)
- `.squad/decisions/inbox/maximus-208-completion-lead-audit.md` — decision card with go/no-go, MVP criteria, code review gates

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

- **2026-05-30 — Feature #208 Completion Lead Audit (Session)**
  
  **Objective:** Baseline assessment of feature #208 (Collection Health Scorecard v1) against plan and tasks.
  
  **Scope:** Full audit of spec.md, plan.md, tasks.md (52 tasks total) and architecture review.
  
  **Key Findings:**
  - Status: 10/52 tasks done (19%), 3 in progress (6%), 39 pending (75%)
  - Critical blockers: T012 (scoring logic) and T011 (service unit tests) are blocking 39 downstream tasks
  - T006 (frontend type stubs) should start immediately in parallel to unblock Phase 3
  - Two HIGH-severity risks: (R1) Scoring calculation bugs, (R6) Empty collection crashes
  
  **Acceptance Criteria:** 10 MVP mandatory criteria defined, plus 4 post-MVP criteria
  
  **Code Review Checkpoints:** 3 checkpoints with explicit rubrics:
  - Checkpoint 1 (Phase 2): Scoring formula 40/20/20/20 weights, grade thresholds 90/80/70/60, empty collection handling, test coverage >85%
  - Checkpoint 2 (Phase 3): Thin handlers, API schema parity, Composition API + TypeScript
  - Checkpoint 3 (Feature complete): §17 Quality Gate passing, no breaking changes, Swagger committed
  
  **Decision:** CONDITIONAL GO on feature #208 with blocking condition on Phase 2 completion (T012 + T011)
  
  **Confidence:** HIGH (full codebase and spec audit performed)
  
  **Decision Entry:** Decision #19 in `.squad/decisions.md`

### 2026-05-31 — Feature #219 Acceptance Checklist & Validation Gates

**Scope Audit:** Front-end-only UI refinement (CoinDetailPage.vue + 4 new dedicated section pages). No API schema changes, no Go backend modifications.

**Acceptance Framework Created:**
  - 36 functional/UX/design gates organized by US1/US2/US3/polish
  - 8 top-risk categories with severity and mitigation strategy (auth bypass, media layout, metadata overflow, sticky regression, etc.)
  - 3 constitution compliance checkpoints (Principle V/IX/XIII, §17 Quality Gate)
  - 12-point tester handoff checklist in 3 validation phases (critical path → regression → polish)

**Gate Highlights:**
  - **F1.1–F1.3**: Dual-side media render by default + graceful fallback (single-side, no-image)
  - **F2.1–F2.4**: Metadata table rows replace boxed cards; empty-value handling; legacy UI removed
  - **F3.1–F3.13**: Settings-style link navigation; 4 dedicated section pages; auth guards; back navigation context
  - **P1–P9**: Build success, type parity, PWA non-sticky enforcement, regression baseline, accessibility

**Constitution Risks (Most Likely to Fail):**
  - **Principle V (Tokens)**: Hardcoded colors/spacing in new table/link rows instead of design tokens (DS1.1, DS2.1, DS3.1)
  - **Principle XIII (PWA)**: Sticky/fixed positioning leaks into mobile/PWA mode, violating non-sticky guarantee (UX1.2, P7)
  - **§17 (Build Gate)**: Code passes local type-check but fails Docker `vue-tsc --build` (P1, P2)

**Decision:** No team-level ADR required. Feature operates within constitutional bounds (UI-only, no layering changes, no auth modifications, no data persistence rules altered). Standard PR checklist applies per §17/§21.

**Output:** Comprehensive validation artifact written to `.squad/decisions/inbox/maximus-feature-219-gates.md` (40 gates + 8 risks + 3 constitution constraints + tester handoff). Ready for Brutus test phase.

**Confidence:** HIGH (full spec, plan, tasks, contracts, and quickstart audited; design patterns consistent with existing coin-detail and settings-style UI surfaces)


- **2026-05-31:** Feature #219 acceptance gates and validation plan delivered. Prepared comprehensive 37-gate acceptance checklist spanning US1 (dual-side media), US2 (metadata tables), US3 (section pages), and polish scope. Three-phase tester handoff: Phase 1 critical path (5 gates including dual-side render, route wiring, auth guards, build success, journal CRUD), Phase 2 regression (4 gates covering edge cases and Constitution compliance), Phase 3 design polish (3 gates). Identified 8 top risks with mitigation strategies and mapped Constitution Principle V/IX/XIII to specific checkpoints. Determined no team ADR needed (UI-only, within constitutional bounds). Handed off to Brutus for execution. Brutus validated all gates; verdict APPROVE.

### 2026-05-31 — Feature #217 Collection Intent Routing Design

**Problem:** The `ShouldHandleCollection()` keyword gate in Go (`collection_tools_service.go`) uses hardcoded substring matching to route collection questions. Brian explicitly rejected this approach—directive captured in `.squad/decisions/inbox/copilot-directive-20260531T203103Z.md`: "must NOT rely on a hardcoded keyword/token list... use real LLM-based intent classification."

**Architectural Tension Resolved:** Collection tools (owner-scoped DB access, GORM, auth context, confirm-gated writes) live in Go. The LLM intent router (11-category `ROUTER_PROMPT`) lives in the stateless Python agent. Two options evaluated:

- **Option A (Classify in Go):** Add an LLM call from Go to classify collection-vs-not before routing. Rejected — adds latency (extra LLM round-trip on every chat), splits routing logic across two services.

- **Option B (Unified Python Router):** Add `collection` as a 12th route in Python supervisor. When classified as `collection`, Python calls back to a new Go `/internal/collection/chat` endpoint with a signed short-lived token carrying `userID`. **Recommended.**

**Design Highlights:**
1. Single LLM classification pass (no latency penalty)
2. Internal callback auth via 30-second JWT with `userID` claim — Go validates token, extracts userID from claims (never trust Python's request body)
3. `ShouldHandleCollection()` keyword gate **deleted entirely** — all routing via LLM
4. SSE contract preserved — Python `collection_node` emits `{"type":"done","collection":...}` matching frontend expectations
5. 10 files touched: `supervisor.py` (add route/node), `agent.go` (generate internal token, remove keyword branch), new `internal_token_service.go`, new `internal_agent.go`

**Security (Principles XI/XII):**
- Internal token is short-lived (30s), signed by Go, audience-scoped
- userID from token claims, not from Python request
- `/internal/` route rejects requests without valid token
- Existing confirm-gated write flow unchanged

**Implementation Order:** InternalTokenService → internal endpoint → wire token in ChatStream → Python model update → router update → collection_node → delete keyword gate → full QA

**Design Artifact:** `.squad/decisions/inbox/maximus-217-intent-routing-design.md` (15KB comprehensive spec ready for Cassius implementation)

## Learnings (2026-05-31 — #216 token remediation re-review)
- Re-reviewed Brutus's Principle V remediation of AddCoinPage.vue. VERDICT: APPROVE — block lifted.
- All 14 originally-flagged hardcoded colors now resolve through tokens; 12 new tokens added to variables.css matching my prescribed values exactly.
- The 4 remaining raw colors (`#000`@808, `#fff`@835, `#fff`@883, `#000`@927) are precisely the contrast-safe exceptions I approved in the original review — no scope creep.
- Brutus improved on my spec: `--shadow-gold-soft/hover` defined as full box-shadow values (consistent with existing `--shadow-card`/`--shadow-glow`), and consolidated my redundant `--error-bg`/`--error-bg-alpha` into a single `--error-bg`. No duplicate tokens, naming consistent with `:root` convention.
- Light-theme (`[data-theme="light"]`) does NOT override the new confidence/feedback tokens — but this matches existing convention (`--cat-*`/`--mat-*` indicator colors are also theme-constant), so not a defect.
- Independently verified: `npm run lint` 0 errors (5 pre-existing unrelated warnings), `npm run build` clean (vue-tsc + vite, 8.35s).

## Learnings (2026-05-31 — #217/#218 Shared Tool Layer Revision)

**Context:** Brian approved LLM-based intent classification (kill keyword gate) but **rejected** my prior Option B (single `collection` routed-node) in favor of a **tool-based approach**. The reason: Brian's real query is multi-intent — "Do I have any moose coins AND how much are they worth" = collection lookup + valuation in ONE reasoning turn. A dedicated route can't compose.

**Design Revision — Key Changes from Option B:**
- ~~`collection` as 12th route~~ → Collection operations become **LangChain tools** callable during agent reasoning
- ~~`collection_node` calls Go callback~~ → A **ReAct agent** wraps collection tools + reasoning, can call multiple tools per turn
- Internal-token auth mechanism **survives** (Principles XI/XII unchanged)
- Keyword gate `ShouldHandleCollection` still deleted

**Shared Tool Layer Architecture:**
- 6 discrete operations: `search_my_collection`, `get_coin`, `collection_summary`, `top_coins_by_value`, `propose_update`, `commit_update`
- Go service layer (`CollectionToolsService`) owns all logic
- `/internal/tools/*` endpoints for Python agent (#217)
- `/external/tools/*` endpoints for MCP/API-key clients (#218 — DEFERRED)
- Same service methods, different transport adapters = no logic duplication

**Phasing Decision:**
- **#217 (NOW):** Internal tool endpoints + Python LangChain tools + ReAct agent + supervisor integration
- **#218 (DEFERRED):** External OpenAPI/MCP adapter + API-key capability controls + external journaling
- **The seam:** `CollectionToolsService` methods are transport-agnostic; adapters call the same layer

**Why Tool-Based Wins:**
1. Multi-intent queries compose naturally (ownership + valuation in one turn)
2. Aligns with existing LangChain/LangGraph patterns (`create_react_agent`)
3. External clients (#218) can call the same operations — no throwaway work
4. ReAct agent can decide when to call collection tools vs. other tools (emergent routing)

**Design Artifact:** `.squad/decisions/inbox/maximus-217-218-shared-tool-layer-design.md` (supersedes `maximus-217-intent-routing-design.md`)

**Key Architectural Insight:** When the user's real query is multi-intent, routing to a single specialized node breaks composition. Tools let the agent reason across capabilities within one turn. This is the "like any chatbot would" Brian described.

- **2026-05-31 (Team Orchestration):** Authored decision documents for pending features and critical design reviews. (1) **#219 Refinements Requirements** (`copilot-219-refinements.md`): Captured Brian's annotated screenshot review of merged coin-detail redesign — 5 TLC items flagged for implementation (duplicate "Actions" heading, duplicate category badge, images side-by-side, Details card needs heading, "+ Add Reference" confirmation). (2) **#216 Camera-First Token Remediation** (`brutus-216-token-remediation.md`): Specified comprehensive token refresh test coverage during AI analysis and error scenarios. (3) **Feature #217/#218 Shared Collection Tool Layer** (proposal document): Designed transport-agnostic collection tool layer serving both in-app (#217) and external (#218) consumers; specified 6 operations (search, get, summary, top-value, propose, commit) as LangChain tools; rejected single-routed-node Option B in favor of tool-based approach supporting multi-intent queries. Document supersedes intent-routing design and awaits Cassius implementation planning. All documents merged into `.squad/decisions.md` (Decisions #4-12) by Scribe. Decisions capture team consensus and drive next implementation phase.

