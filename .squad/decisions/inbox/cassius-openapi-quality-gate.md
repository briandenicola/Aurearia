---
date: 2026-05-30
author: Cassius (Backend)
status: Proposed
run: "26656552925"
job: "78568056509"
---

# Decision: Resolve OpenAPI snapshot drift before merge

## Context
Quality Gate run `26656552925` failed in job `78568056509` at **Verify OpenAPI snapshot**. The CI log diff showed generated OpenAPI artifacts were stale.

## Root Cause
Swagger annotations in `src/api/handlers/webauthn.go` already include `@Failure 403` for:
- `POST /auth/webauthn/login/finish`
- `POST /auth/webauthn/register/finish`

But generated artifacts were not regenerated/committed, so CI regenerated files and found drift in:
- `src/api/docs/docs.go`
- `src/api/docs/swagger.json`
- `src/api/docs/swagger.yaml`
- `docs/openapi.json`

## Decision
Regenerate and commit OpenAPI artifacts whenever handler annotations change, using the same command path as CI (`swag init ...` + sync `docs/openapi.json`; equivalent to `task openapi`).

## Verification
- `go build ./...` ✅
- `go vet ./...` ✅
- `go test ./...` ✅
- OpenAPI snapshot verify command path rerun after regeneration ✅
