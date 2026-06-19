# Project Context

- **Project:** coin-collection-app
- **Created:** 2026-04-24

## Core Context

Agent Scribe initialized and ready for work.

## Recent Updates

- **2026-06-19T00:55:20Z:** Logged PWA UI Consistency batch session (Maximus audit + Aurelia refactor + Brutus QA + Coordinator validation). Created 4 orchestration logs (Maximus, Aurelia, Brutus, Coordinator) and 1 session log. Merged 1 decision (PWA Page Header Consistency) into decisions.md; deleted 2 inbox files (maximus-ui-consistency.md, aurelia-page-header-consistency.md). All 7 feature pages now normalized to canonical .page-header pattern. Build validated (npm run build ✓). Notes stats section removed. Principle VI (Consistent UX) binding standard now enforced.
- **2026-06-18T22:59:00Z:** Logged WebAuthn Backup Eligible Storage Fix session (Cassius backend + Brutus QA regression tests + Coordinator validation). Created 3 orchestration logs (Cassius, Brutus, Coordinator) and 1 session log. No inbox entries to merge. Updated Cassius/Brutus history.md with coordinated session note. Issue #299 fix complete; ready for merge.
- **2026-06-09 (13:25:42):** Logged F013 Phase 4 browser workflow infrastructure batch (Aurelia T018–T021 + Maximus review/block + Brutus hygiene fix). Created 3 orchestration logs (Aurelia, Maximus, Brutus) and 1 session log. No inbox entries to merge. Appended cross-team browser E2E testing context to Aurelia/Maximus/Brutus history.md. All product changes staged, ready for coordinator merge. F013 T006–T021 now complete.
- **2026-06-09 (13:09:16):** Logged F013 Phase 3 golden fixtures completion (Cassius T014 + Maximus T014 approval + Brutus T016/T017 + Maximus T016/T017 approval). Created 4 orchestration logs and 1 session log. No inbox entries to merge. F013 T006–T017 now complete; Phase 4 browser workflows (T018–T028) deferred.
- **2026-06-09:** Logged F013 regressions & fixtures batch (Brutus backend T011/T012/T013 + Aurelia frontend T015). Merged 11 decision inbox entries (NGC strategies, lookup UX, GORM patterns, SSRF suppression, audit scheduler, user directives). Cleared inbox. Session F013 T006–T013 and T015 marked complete.
- **2026-06-01:** Logged storage-location/settings batch, merged Cassius/Aurelia decision inbox entries, and committed feature plus squad handoff state.

📌 Team initialized on 2026-04-24

## Learnings

Initial setup complete.

**Phase 1 Governance Scaffolding (2026-05-28):** Seeded core governance infrastructure — decision inbox, SECURITY.md (30-day disclosure window per hobby-project deviation from tech-inventory's 90-day standard), CODEOWNERS, and bug/feature issue templates. Templates include `squad` label on features to route into Lead triage queue.

**Phase 2 Backlog Card Seeding (2026-05-28):** Seeded 7 retroactive backlog cards in `specs/_backlog/` from existing feature surface in `docs/features.md` — F001 (Coin of the Day, Cassius, P1), F002 (Wishlist & Availability, Cassius, P1), F003 (Portfolio Review, Maximus, P2), F004 (Coin Analysis Vision, Maximus, P2), F005 (Social Following, Cassius+Aurelia, P2), F006 (PWA Offline, Aurelia, P1), F007 (WebAuthn Biometric, Cassius+Aurelia, P1). All marked `promoted` status with Constitution Principle citations and retroactive governance notes per §0 Hierarchy Phase 2 workflow.

**Phase 3 PRD Authoring (2026-05-28):** Drafted `docs/prd.md` v1 (~580 lines) extracted from README (vision, positioning), features.md (functional areas), authentication.md (auth section), social-feature.md (social section), and foundation spec (personas, constraints, assumptions). Structured per Hierarchy of Authority §0 item 2 with 11 functional areas cross-linked to backlog cards (F00N). Cited Constitution §19 (Documentation Requirements) and Principle alignment. Surfaced 6 honest open product questions for v1.0+. Ready for Maximus review Round 2 before Brian trims README to match.

- **2026-06-01T20:14:08Z:** Wrote orchestration/session logs for Maximus storage-location design, merged decision inbox into decisions.md, cleared inbox files, and summarized large agent histories over 12KB per Scribe charter.
