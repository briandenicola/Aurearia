# Implementation Plan: External Tool Server Adapter With Read/Write Parity

**Branch**: `218-external-tool-server-adapter` | **Date**: 2026-06-01 | **Spec**: `/specs/218-external-tool-server-adapter/spec.md`
**Input**: Feature specification from `/specs/218-external-tool-server-adapter/spec.md`

## Summary

Implement issue #218 (Epic F012, Card 4) by adding a public, versioned
`/api/v1/tools/*` HTTP surface that re-exposes the issue #217 collection tool layer
to external clients with full read/write parity. Authentication reuses the existing
per-user API keys, extended with read/write capability scopes (default read-only).
Writes keep the two-phase `propose_update` / `commit_update` token flow and the same
field allowlist as in-app chat. The whole surface is gated by an admin kill switch,
per-key rate limited, journaled with source `external_tool_server`, and described by
a served, scoped OpenAPI document for client auto-import. MCP compatibility is
documentation-only via `mcpo`.

## Technical Context

**Language/Version**: Go 1.26.x, TypeScript (Vue 3)
**Primary Dependencies**: Gin, GORM, SQLite, Vue 3 + Pinia, swaggo (OpenAPI generation)
**Storage**: SQLite (extend `api_keys` with capability column; reuse existing
`collection_update_proposals`, `coins`, `coin_journals`)
**Testing**: `go test ./...`, `go vet ./...`, `npm run build`, `npm run lint`
**Target Platform**: Linux-hosted web app + PWA client; external HTTP/OpenAPI clients
**Project Type**: Web application (Go API + Vue frontend; Python agent unchanged)
**Performance Goals**: External read/proposal/commit latency comparable to the
existing internal tool endpoints; per-key rate limiting bounds external load
**Constraints**: Go API is the only writer (ADR 0002); server-side user scoping via
API key only; capability-gated operations; two-phase token-gated writes; same field
allowlist as in-app; no duplicated query/update logic (reuse `CollectionToolsService`);
external surface disabled by default
**Scale/Scope**: Single feature spanning Go API (new public route group, middleware,
model migration, served OpenAPI) and Vue frontend (API-key scope selection UI), plus docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Layered Architecture) | PASS | New thin handlers call the existing `CollectionToolsService`; capability check in middleware; no logic duplication. |
| Principle III (Service Boundary Separation) | PASS | Go remains the sole writer and auth boundary; no Python/DB changes; external clients act only through the API. |
| Principle VII (Schema-Driven Contracts) | PASS | A dedicated scoped OpenAPI document is served and committed; Swagger annotations on handlers. |
| Principle XI (Security Hardening) | PASS | API-key auth, capability scopes, admin kill switch, per-key rate limiting, allowlisted writes, output scoping. |
| Principle XII (Auth & Token Policy) | PASS | Reuses hashed API keys + revocation; least-privilege default (read-only); proposal tokens unchanged. |
| Principle XIII (PWA/Mobile Rules) | PASS | Only additive Settings UI for key scope; no service-worker or offline-boundary change. |
| §17 Quality Gate | PASS | Validation spans Go (build/vet/test, architecture test) and web (build/lint); docs updated. |
| §21 Definition of Done | PASS | Threat model + docs (features.md, api-reference, new external-tool-server doc) included in tasks. |

## Project Structure

### Documentation (this feature)

```text
specs/218-external-tool-server-adapter/
├── plan.md
├── spec.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── external-tool-server.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
src/api/
├── models/
│   └── api_key.go                         # extend: Capabilities/Scope field
├── repository/
│   └── api_key_repository.go              # capability persistence + lookups
├── middleware/
│   ├── auth.go                            # X-API-Key sets userId (reuse)
│   ├── capability.go                      # new: require read/write capability
│   └── ratelimit.go                       # new per-key external limiter
├── services/
│   ├── collection_tools_service.go       # reuse; thread journal source for writes
│   └── settings_service.go               # add ExternalToolServerEnabled (default off)
├── handlers/
│   ├── external_tools.go                  # new: /v1/tools/* handlers (thin)
│   ├── external_tools_openapi.go          # new: serve scoped OpenAPI doc
│   └── api_keys.go                         # accept scope at creation; expose in list
├── database/database.go                   # AutoMigrate updated ApiKey
└── main.go                                # wire /api/v1/tools group + kill-switch gate

src/web/src/
├── api/client.ts                          # api-key create payload includes scope
├── types/index.ts                         # ApiKey type gains capabilities
└── components/settings/                    # API-key management UI: scope selector

docs/
├── external-tool-server.md                # new: setup + OpenWebUI/LibreChat/n8n + mcpo
├── features.md                            # mention external tool server
├── api-reference.md                       # /v1/tools/* + scoped OpenAPI URL
└── threat-model.md                        # external surface threat update
```

**Structure Decision**: Add the external surface as a sibling public route group in
the same Gin process, delegating to the already-shared `CollectionToolsService`. This
fulfills the epic's "one tool layer, many adapters" principle: the internal Python
adapter (`/api/internal/tools/*`, HMAC token) and the external adapter
(`/api/v1/tools/*`, API key + capability) are two thin transports over identical core
operations.

## Phase 0 Research

See `research.md`. Resolves: external auth model (API key + capability scopes),
external write confirm model (two-phase parity), journal source threading, kill-switch
placement, per-key rate-limiting strategy, served scoped OpenAPI approach, and the
docs-only MCP compatibility path (`mcpo`).

## Phase 1 Design Outputs

1. `data-model.md` — `ApiKey` capability extension + migration default, journal
   source threading, and the reused proposal lifecycle for external writes.
2. `contracts/external-tool-server.openapi.yaml` — the `/v1/tools/*` operations,
   `X-API-Key` security scheme, capability/kill-switch error responses, and the
   served-spec endpoint.
3. `quickstart.md` — end-to-end external read + two-phase write validation and the
   negative safety scenarios (cross-user denial, read-only write denial, kill switch,
   rate limit, token replay).
4. Agent context updated via `.specify/scripts/powershell/update-agent-context.ps1
   -AgentType copilot` (run when a feature branch is checked out).

## Post-Design Constitution Check

| Gate | Status | Notes |
|------|--------|-------|
| Principle I (Layered Architecture) | PASS | Handlers stay thin; capability enforced in middleware; service layer unchanged in responsibility. |
| Principle III (Service Boundary Separation) | PASS | Single shared writer; no new persistence logic duplicated for the external transport. |
| Principle VII (Schema-Driven Contracts) | PASS | Scoped OpenAPI doc is the external contract and is served + committed. |
| Principle XI/XII (Security/Auth) | PASS | Least-privilege default, capability gating, kill switch, rate limiting, journaled writes. |
| Principle XIII (PWA/Mobile Rules) | PASS | Additive Settings UI only. |

## Complexity Tracking

No constitution violations or waivers identified at planning time. The only schema
change is an additive `ApiKey` capability column with a least-privilege default;
all collection read/write logic is reused from issue #217.
