# Project Context

- **Owner:** Brian
- **Project:** Ancient Coins QA/testing across Go API, Vue frontend, and Python agent
- **Architecture:** Tests enforce constitution-backed layer boundaries, auth/ownership guarantees, and feature acceptance criteria.

## Core Context

- Brutus owns testing and review. Durable test patterns: in-memory SQLite + httptest/Gin for Go handlers/repos/services, Vitest with axios/localStorage mocks for frontend, Ruff/pytest for Python, and architecture tests for import boundaries.
- Testing baseline expanded from minimal coverage to auth/security, API client, auth store, settings, parser, and component tests. `docs/testing.md` is canonical test strategy; known gaps include browser E2E, Go cross-process integration, and Python static type checking.
- Strict Lockout applies: when a reviewer marks BLOCK, the blocked implementer does not revise until the block is cleared by reviewer or delegated agent.
- Durable QA rules: verify actual code and data paths, not just claims; classify pre-existing failures separately; document non-blocking nits without blocking otherwise passing features.

## Recent Updates

- **2026-06-01:** QA contract note: coins now support nullable `storageLocationId` and optional `storageLocation`; storage-location CRUD is under protected `/api/storage-locations`; deleting a location in use returns 409 with a coin count. Settings Data now covers Tags + Storage Locations, while backups/imports/API keys moved to `Backups & Keys`.

- **2026-05-31:** Feature #219 QA approved: 12/12 functional requirements satisfied, route wiring/auth guards verified, vue type-check/build clean, no regressions except unrelated pre-existing test issues.
- **2026-05-31:** #216 Principle V token remediation executed under Strict Lockout: added design tokens and replaced flagged hardcoded colors; lint/build clean; Maximus later approved.
- **2026-06-01:** #218 polish validation approved: capability middleware tests added, Go build/vet/test clean, frontend build/lint clean, quickstart scenarios A/B/C and negative scenarios N1-N6 traced to code.
- **2026-06-01:** #218 BLOCK resolution applied: all Gin context type assertions in external tool handlers now use comma-ok guards returning 401/403 instead of risking panic; Go build/vet/test clean.
