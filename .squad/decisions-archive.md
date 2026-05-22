# Squad Decisions Archive

Archived decisions from `decisions.md` older than 30 days.

## Archived Decisions

### 1. Full-System Architecture Document

**Author:** Maximus (Lead/Architect)  
**Date:** 2025-07-18  
**Status:** Implemented  

#### What
Rewrote `docs/ARCHITECTURE.md` from a Go-API-only document (~214 lines) to a comprehensive full-system architecture reference (~761 lines) covering all three services.

#### Why
The previous doc only covered the Go API layered architecture. Missing: frontend architecture, Python agent service, data flow diagrams, database schema, auth flow details, agent integration pattern, background schedulers, build pipeline, configuration reference, and design decision rationale.

#### Scope
- System overview and container topology diagram
- Go API: layers, rules, package map, DI wiring, route groups, scopes, arch tests
- Vue 3: structure, routing, Pinia stores, API client (401 refresh queue), composables, PWA config
- Python agent: endpoints, supervisor routing, 11 team pipelines, LLM provider abstraction, SSE streaming
- Data flow diagrams: standard request, agent chat SSE, auth flow, availability check
- Database schema: 26 models across 6 categories
- Authentication: JWT + API key + WebAuthn details
- Background schedulers: availability + valuation
- Docker multi-stage build for both containers
- Configuration reference (env vars + runtime settings)
- Key design decisions with rationale

#### Impact
All team members and AI agents now have a single reference for system architecture. No code changes — documentation only.

---
