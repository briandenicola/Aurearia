# ADR 0021: Stateless MCP Adapter in the Go API

Date: 2026-10-04
Status: Proposed

## Context

Aurearia already has a default-off, API-key-authenticated external tool server
and a Go-owned durable Coin Copilot harness. External agentic harnesses
increasingly use Model Context Protocol rather than importing OpenAPI
operations directly. Adding MCP changes the authenticated external boundary and
introduces the official MCP Go SDK, so the decision requires an ADR.

A separate MCP process would either need database access, violating service
ownership, or would become a second proxy/authentication layer. Stateful MCP
sessions would also need strict binding to an API-key owner to prevent a stolen
session id from crossing tenant boundaries.

## Decision

Implement MCP as a protocol adapter in the existing Go API using the official,
pinned `github.com/modelcontextprotocol/go-sdk`.

The adapter uses stateless Streamable HTTP at `POST /api/mcp`. Each request is
authenticated by existing middleware, then receives a fresh MCP server whose
tool closures are bound to the authenticated owner. The endpoint reuses the
default-off external-tool setting and per-key rate limiter.

Read tools delegate to an HTTP-agnostic service backed by existing collection
and auction repositories. Coin Copilot lifecycle tools delegate to the existing
durable service and are registered only when the key has an exact `copilot`
capability. `write` does not imply `copilot`, and Copilot does not imply
collection mutation access.

MCP prompts, resources, roots, sampling, stateful sessions, direct Python
calls, arbitrary HTTP, and mutation tools are not exposed.

## Consequences

### Positive

- Agentic harnesses use a standard protocol without a fourth service.
- Authentication, owner scoping, feature flags, rate limits, and durable
  orchestration remain Go-owned.
- Stateless requests avoid cross-key session binding and cleanup complexity.
- Existing REST/OpenAPI integrations remain available.

### Negative

- The API gains a pinned third-party protocol dependency.
- Clients poll durable Copilot runs instead of receiving MCP server-initiated
  notifications.
- API-key management gains another explicit capability combination.

## Rollback

Set `ExternalToolServerEnabled=false` to disable MCP and the existing external
tool surface immediately. Removing the `/api/mcp` route and SDK dependency does
not affect persisted collection, auction, API-key, or Coin Copilot data.

## Related

- [Feature 365](../../specs/365-mcp-agentic-harness/spec.md)
- [ADR 0016](0016-go-owned-durable-coin-copilot-state.md)
- [External Tool Server](../external-tool-server.md)
