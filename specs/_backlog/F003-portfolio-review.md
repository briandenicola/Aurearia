---
title: Portfolio Review
id: F003
status: promoted
priority: P2
effort: L
value: 4
risk: 3
owner: Maximus
created: 2026-05-28
updated: 2026-05-28
---

## Summary

AI-driven portfolio analysis: read all held coins, calculate valuations, analyze trends and gaps. Go API retrieves holdings summary, proxies to Python agent service (`portfolio_review.py` team), returns structured analysis with market insights.

## Acceptance Criteria

- When user requests portfolio review, Go API fetches all non-sold, non-wishlist coins via repository
- When holdings summary is sent to Python agent, agent returns structured analysis (total value, distribution, grade breakdown, era concentration, gaps)
- When user's AIProvider is Anthropic, agent uses Claude models; when Ollama, agent uses self-hosted models
- When Python agent completes, response streams via SSE to frontend with progress indicators
- When analysis is rendered, markdown is safe-escaped (no HTML injection)

## Constitution Alignment

**Principle II (Multi-Agent Architecture):** Stateless FastAPI service; Go API proxies byte stream via `services/agent_proxy.go`.
**Principle XVIII (Agent Operating Rules):** Supervisor enforces max iteration count; all worker agent outputs conform to Pydantic schema.
**Principle IX (AI Provider Configuration):** Agent respects user's selected provider (Anthropic or Ollama).

## Implementation Notes

- Portfolio summary endpoint: `GET /portfolio/summary` (returns coins, total value, grade distribution, era/region)
- Agent proxy: `services/agent_proxy.go` streams Python agent response via SSE
- Agent team: `app/portfolio_review.py` uses LangGraph StateGraph
- Vision model analysis (obverse/reverse separate) calls per coin if no cached AIAnalysis
- Valuation runs configurable: interval (default 7 days), start time (default 03:00), max coins per run (default 50)

## Open Questions

None — feature shipped.

## Notes

Retroactive card created 2026-05-28 for governance traceability under Constitution §0 (Hierarchy) Phase 2. Run history persists in `ValuationRun` model; manual trigger/cancel available in admin UI.
