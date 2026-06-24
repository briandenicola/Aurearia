# Implementation Plan: OIDC Login for Entra ID and Pocket ID

**Branch**: `335-oidc-login` | **Date**: 2026-06-23 | **Spec**: `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\335-oidc-login\spec.md`  
**Input**: Feature specification from `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\335-oidc-login\spec.md`

## Summary

Add OIDC as an additive authentication strategy beside local password and WebAuthn login. The Go API owns provider configuration, OIDC authorization-code + PKCE validation, external identity linking, final-local-admin recovery enforcement, and security audit events. The Vue app adds login buttons, account linking UI, and admin provider management/status controls while preserving existing auth flows.

## Technical Context

**Language/Version**: Go 1.26.4 API, Vue 3 + TypeScript frontend  
**Primary Dependencies**: Gin, GORM, SQLite, existing JWT/refresh-token auth, existing security event service; add `github.com/coreos/go-oidc/v3/oidc` and `golang.org/x/oauth2` unless implementation discovers a better maintained OIDC verifier compatible with Go 1.26.4  
**Storage**: SQLite via GORM AutoMigrate; new provider, external identity, and short-lived auth state tables  
**Testing**: Go `go test -v ./...`; Vue `npm run build`, `npm test`, `npm run lint` when frontend changes are implemented  
**Target Platform**: Self-hosted web app/API running in existing Docker/development environments  
**Project Type**: Full-stack web application with Go API backend and Vue SPA frontend  
**Performance Goals**: OIDC callback validation should complete within normal auth request latency excluding provider network time; discovery/JWKS lookups should be cached per provider where the library supports it  
**Constraints**: Preserve local login/WebAuthn behavior; do not expose secrets; do not log tokens/codes/secrets; protect against open redirect/CSRF; always retain one local admin recovery account; follow Handler -> Service -> Repository architecture  
**Scale/Scope**: Two initial provider types (Entra ID and Pocket ID), multiple configured providers, existing single-tenant personal collection deployment

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I - Clear Layered Architecture**: PASS. Plan introduces `handlers/oidc.go`, `services/oidc_service.go`, and `repository/oidc_repository.go`; OIDC business rules and final-local-admin safety live in services, DB access in repositories, and handlers stay thin.
- **Principle II - Service Boundary Separation**: PASS. OIDC is API/auth work only; no Python agent coupling.
- **Principle III - Strict Types and Explicit Contracts**: PASS. Contracts are documented in `contracts/oidc-api.md`; Go and TypeScript types are planned.
- **Principle IV - Simple Complete Changes**: PASS. Uses a vetted OIDC verifier and existing auth/session/security patterns instead of replacing auth.
- **Principle V - Security, Auth, and Privacy by Default**: PASS with required implementation gates for PKCE/state/nonce, issuer/audience/signature/expiry validation, secret redaction, no token logging, and recovery admin enforcement.
- **Principle VI - Consistent User Experience**: PASS. Reuses existing LoginPage, Settings Account, and Admin Settings section patterns.
- **Principle IX - Automated Quality Enforcement**: PASS. Tasks include exact-path regression coverage for local auth preservation, OIDC validation failures, account linking, secret redaction, and final-local-admin protection.
- **§17 Quality Gate + §21 Definition of Done**: PASS. Plan includes exact failing-path tests and workflow-contract blast-radius checks for all auth/admin/account surfaces.

## Project Structure

### Documentation (this feature)

```text
specs/335-oidc-login/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── oidc-api.md
└── tasks.md
```

### Source Code (repository root)

```text
src/api/
├── database/database.go                 # AutoMigrate new models
├── handlers/
│   ├── auth.go                          # preserve existing token issuance shape
│   ├── oidc.go                          # public/protected OIDC endpoints
│   ├── oidc_test.go
│   ├── admin.go                         # final-local-admin guard for existing user ops
│   └── admin_test.go
├── models/
│   ├── oidc_provider.go
│   ├── external_identity.go
│   ├── oidc_auth_state.go
│   ├── security_event.go                # OIDC event constants
│   └── user.go                          # local-auth usability fields if needed
├── repository/
│   ├── oidc_repository.go
│   ├── auth_repository.go               # user lookup helpers by email/ID as needed
│   ├── admin_repository.go              # recovery-account count/transaction helpers
│   └── security_repository.go
├── services/
│   ├── oidc_service.go
│   ├── oidc_service_test.go
│   ├── auth_service.go                  # shared token issuance/local auth semantics if needed
│   ├── admin_recovery_service.go
│   ├── admin_recovery_service_test.go
│   └── security_service.go
└── main.go                              # dependency wiring/routes

src/web/src/
├── api/client.ts                        # OIDC/admin provider endpoints
├── stores/auth.ts                       # consume OIDC callback AuthResponse
├── types/index.ts                       # OIDC DTOs
├── pages/LoginPage.vue                  # provider buttons/callback errors
├── components/settings/SettingsAccountSection.vue
├── components/admin/AdminOIDCSection.vue
└── components/admin/__tests__/AdminOIDCSection.test.ts

docs/
└── oidc-setup.md                        # Entra ID and Pocket ID setup
```

**Structure Decision**: Use existing Go layered packages and Vue admin/settings/login surfaces. Add OIDC-specific model/repository/service/handler files to keep auth logic isolated while reusing the current JWT response shape and security event system.

## Complexity Tracking

No constitution violations are planned.

## Post-Design Constitution Check

- **Principle I** remains PASS because data model and contracts assign persistence to repositories, validation/business rules to services, and route parsing to handlers.
- **Principle V** remains PASS because contracts explicitly redact secrets, require state/nonce/PKCE, use fixed callback endpoints, and include final-local-admin blocking responses.
- **§17/§21** remain PASS because tasks require exact-path regression tests before implementation for OIDC validation, linking conflicts, local auth preservation, and admin recovery safety.
