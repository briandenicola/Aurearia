---
name: route-openapi-drift-test
description: "Keep registered Go routes, public handler annotations, and generated OpenAPI snapshots aligned."
---

# Skill: Route vs OpenAPI Drift Testing

**Category:** Backend Development
**Applies to:** Ancient Coins Go API
**Last Updated:** 2026-06-19

## When to Use This Skill

Use this recipe when adding, moving, or reviewing Go API routes in `src/api/main.go` so Swagger/OpenAPI stays aligned with the registered Gin surface.

## Pattern

1. Add or update Swagger annotations on the public handler method with the route path relative to the API `@BasePath /api`.
2. Regenerate artifacts from the repository root with `task openapi`.
3. Run `cd src/api; go test -v -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI .`.
4. If a route is deliberately not public, add a narrow documented exemption in `src/api/route_openapi_drift_test.go` rather than leaving it as unexplained drift.

Generation writes files and requires authorization plus a prepared pinned
generator; do not install implicitly. Finish with applicable `task check:go`
and `task check:openapi`. Only add an exemption if the approved contract and
existing exemption policy justify it, never merely to silence the drift test.

## Route Normalization Rules

- Gin `:id` becomes OpenAPI `{id}`.
- Gin `*filepath` becomes OpenAPI `{filepath}`.
- Registered `/api/...` routes are compared against OpenAPI paths with `/api` stripped because `main.go` declares `@BasePath /api`.

## Intentional Exemptions

Keep exemptions limited to non-contract routes such as root health checks, Swagger UI assets, root media aliases, and `/api/internal/tools/*` internal-token callback routes.

## Example

See `src/api/route_openapi_drift_test.go` and the #316 annotation additions in handlers for tags, social, showcase, calendar, alerts, agent, health, and admin routes.
