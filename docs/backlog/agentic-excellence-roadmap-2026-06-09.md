# Agentic Excellence Roadmap (2026-06-09)

## Goal

Make Ancient Coins the best-coded, best-tested, most useful agentically built
coin-collecting application: harden the strong agentic foundation that already
exists, then extend it into features that behave like a careful numismatic
assistant.

## Current baseline

The roadmap is not a greenfield plan. The app already has important agentic
building blocks:

- In-app agent chat routed through the Python supervisor.
- `collection_chat` with collection read tools and confirm-gated update
  proposals.
- A Go `CollectionToolsService` and internal/external tool endpoints.
- Agentic coin intake drafts from images and coin cards.
- Portfolio review, gap analysis, availability checking, price trends, similar
  lots, coin search, coin shows, grading, and vision analysis teams.

The work below is about **hardening, testing, and leveling up** those surfaces,
not duplicating them.

## Operating principles

- **Reliability before novelty**: critical collection workflows must be boringly
  dependable before new agentic surfaces expand.
- **Agent proposes, user commits**: AI may draft, explain, and recommend; users
  confirm changes before writes land.
- **Structured over prose**: agent outputs that drive UI or writes use typed
  schemas with confidence and evidence.
- **One tool layer, many adapters**: in-app chat, external tools, and future
  agents reuse the same server-side collection capabilities.
- **Simple Complete Changes**: every implementation slice stays direct,
  proportional, and covered by workflow tests.

## Phased implementation plan

Detailed implementation tasks live in
`docs/backlog/agentic-excellence-implementation-tasks-2026-06-09.md`.

| Phase | Focus | Backlog cards | Exit criteria |
|---|---|---|---|
| 0 | Core workflow hardening | F013 | Coin create/edit/update paths use typed contracts and regressions cover sibling update paths. |
| 1 | Testing foundation | F011, F013 | Golden collection fixtures and browser workflow tests cover critical add/edit/search/mobile flows. |
| 2 | Collection intelligence spine | F012, F014 | Existing collection tools/chat are hardened, expanded, and paired with attribution/reference drafts. |
| 3 | Collector-facing agents | F015 | Existing portfolio/gap/availability capabilities evolve into curator, watchlist, and provenance workflows with citations and confidence. |
| 4 | Agentic engineering loop | F016 | PRs and releases expose workflow coverage, complexity hotspots, flaky tests, and agent review status. |

## Promotion order

1. **F013 — Harden critical collection workflows.** This repairs trust and
   creates the typed/tested foundation for all future agentic writes.
2. **F011 — AI-driven browser testing.** Promote once F013 defines the golden
   workflows and fixture set the browser agent should exercise.
3. **F012 — Agentic collection access.** Harden and expand the existing shared
   collection tool layer, collection chat routing, and confirm-gated writes.
4. **F014 — Attribution and reference assistant.** Improve the existing coin
   intake/analysis foundation with source-backed references, confidence, and
   evidence without silent writes.
5. **F015 — Curator, watchlist, and provenance agents.** Level up existing
   portfolio/gap/availability features into dedicated collector workflows.
6. **F016 — Agentic engineering quality cockpit.** Add durable visibility into
   whether the repo is staying simple, tested, and healthy.

## Dependency map

```text
F013 Core workflow hardening
  ├─ enables F011 browser workflow testing
  ├─ de-risks F012 confirm-gated collection writes
  └─ provides golden fixture data for F014/F015

F012 Existing collection tool layer
  ├─ hardens in-app collection chat
  ├─ extends external tool adapters
  └─ provides safer read/write primitives for F015

F014 Attribution/reference assistant
  └─ improves data quality for F015 recommendations

F016 Quality cockpit
  └─ monitors all phases; can start after F013 defines core metrics
```

## Definition of success

- Critical collection workflows have browser-level regression coverage.
- Coin update/create contracts are typed, explicit, and hard to misuse.
- AI-generated coin data is evidence-backed, confidence-scored, and reviewable.
- Collector agents answer questions and recommend actions using owned collection
  data, not generic chatbot guesses.
- Agentic writes are confirm-gated, journaled, and scoped server-side.
- Pull requests show whether the real workflow was tested and whether the change
  stayed simple, complete, and proportional.
