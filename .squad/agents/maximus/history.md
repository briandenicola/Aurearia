# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins — full-stack PWA for managing a personal ancient coin collection
- **Stack:** Go/Gin/GORM/SQLite API, Vue 3/TypeScript frontend, Python FastAPI/LangGraph agent
- **Architecture:** Layered Handler → Service → Repository → Database; governed by `.specify/memory/constitution.md`.

## Core Context

- Maximus owns architecture, governance, feature design, and cross-artifact consistency. Durable governance outcomes: Constitution v2.0.0 with §0 hierarchy, §17 Quality Gate, §18 AI operating rules, §21 DoD, §22 amendment process; `.squad/log/` + `.squad/decisions.md` are the handoff surface, not `SESSION-NOTES.md` or `.copilot-state.md`.
- Governance scaffolding established: `specs/` workflow, retroactive `specs/001-foundation`, backlog cards F001-F007, PRD as product source of truth, ADR practice in `docs/adr/`, trimmed README, security-doc split, references/gitleaks/pre-commit support.
- Architecture baselines: Go API uses strict layered architecture and DI via `main.go`; Python agent remains stateless; Vue must use design tokens/global classes. Material auth/security/service-boundary choices require ADRs per §22.
- Important prior designs/reviews: #208 health scorecard audit identified scoring/test blockers; #216 camera-first intake was functionally sound but blocked on Principle V token violations until remediated; #217/#218 collection tools pivoted from route-based intent to composable LangChain tools with Go-owned internal/external adapters; Foundry rewrite spike was NO-GO except as a future strategic reconsideration.
- Brian's governance preference: copy cross-repo discipline (hierarchy, DoD, Quality Gate, ADRs, specs) without importing stack-specific .NET/Svelte rules.

## Recent Updates

- **2026-05-31:** Feature #219 validation gates defined for dual-side media, metadata tables, dedicated section pages, PWA behavior, and design-token compliance. Brutus later approved implementation.
- **2026-05-31:** Designed #217/#218 shared collection tool layer: 6 operations (`search_my_collection`, `get_coin`, `collection_summary`, `top_coins_by_value`, `propose_update`, `commit_update`), Go service ownership, Python ReAct tools, external adapter deferred/then implemented.
- **2026-05-31:** Re-reviewed #216 token remediation and approved: all originally flagged hardcoded colors moved to tokens except explicitly accepted contrast-safe black/white uses.
- **2026-06-01:** Storage Location design investigation completed. Recommended per-user `StorageLocation` lookup table with nullable `Coin.StorageLocationID`, single-select semantics, settings-style management, rename updates shared lookup row, duplicates rejected case-insensitively per user, and delete-while-in-use blocked by default pending Brian's final decision.

- **2026-06-01:** SQLite nullable-FK convention: for new nullable `Coin` lookup associations added after launch, keep the scalar `*_id` and preload association but use `constraint:-` to avoid destructive SQLite table rebuilds; enforce validity in service/repository code unless an explicit safe rebuild migration exists.
