# Tasks: OIDC Login for Entra ID and Pocket ID

**Input**: Design documents from `C:\Users\brian.denicolafamily\Code\AncientCoins\specs\335-oidc-login\`  
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/oidc-api.md`

**Tests**: Required by issue #335 and constitution §17/§21 for exact-path regression coverage.

**Organization**: Tasks are grouped by independently testable user story.

## Phase 1: Setup and dependency baseline

- [x] T001 Add OIDC dependencies to `src/api/go.mod`: `github.com/coreos/go-oidc/v3/oidc` and `golang.org/x/oauth2`; update `src/api/architecture_test.go` service-layer external allowlist so Principle IX continues to enforce the dependency boundary explicitly.
- [x] T002 Run dependency tidy from `src/api` and commit updated `go.mod`/`go.sum`.
- [x] T003 [P] Add OIDC TypeScript DTOs to `src/web/src/types/index.ts` for public providers, admin providers, linked identities, and start-flow responses.
- [x] T004 [P] Add API client wrappers in `src/web/src/api/client.ts` for the endpoints documented in `specs/335-oidc-login/contracts/oidc-api.md`.

---

## Phase 2: Foundational API model, repository, and safety services

**Purpose**: Core data and security rules that block all OIDC user stories.

- [x] T005 Create `src/api/models/oidc_provider.go` with provider fields, validation-friendly types, and JSON tags that never expose client secret.
- [x] T006 Create `src/api/models/external_identity.go` with uniqueness constraints for provider/issuer/subject.
- [x] T007 Create `src/api/models/oidc_auth_state.go` for short-lived state, PKCE, nonce, redirect path, flow type, expiry, and consumed timestamp.
- [x] T008 Add the three new OIDC models to `src/api/database/database.go` AutoMigrate.
- [x] T009 Create `src/api/repository/oidc_repository.go` with provider CRUD, public enabled-provider listing, external identity CRUD, email lookup support, and atomic state consume.
- [x] T010 Extend `src/api/models/security_event.go` with OIDC login/link/config/final-admin event constants.
- [x] T011 Extend `src/api/services/security_service.go` with helper methods for OIDC event recording that redact sensitive values.
- [x] T012 Create `src/api/services/admin_recovery_service.go` to count admins with usable local credentials and guard delete, demote, local-auth disable, and OIDC-only conversion flows.
- [x] T013 [P] Add tests in `src/api/services/admin_recovery_service_test.go` for final-local-admin blocking and allowed non-final-admin operations.
- [x] T014 [P] Add SQLite-backed repository tests in `src/api/repository/oidc_repository_test.go` for unique external identity constraints and atomic state replay prevention.

**Checkpoint**: Schema, repository, security events, and recovery safety are ready.

---

## Phase 3: User Story 1 - Admin configures OIDC providers (Priority: P1)

**Goal**: Admins can manage Entra ID and Pocket ID provider configs safely.

**Independent Test**: Provider CRUD and test endpoints work against mocked discovery metadata, and secret reads are redacted.

### Tests for User Story 1

- [x] T015 [P] Add handler tests in `src/api/handlers/oidc_admin_test.go` verifying provider create/update/list redacts client secrets and preserves existing secret on empty/redacted updates.
- [x] T016 [P] Add service tests in `src/api/services/oidc_service_test.go` for Entra tenant issuer discovery and Pocket ID discovery using `httptest`.
- [x] T017 [P] Add security event tests verifying provider config changes and provider test failures are audited without secrets.

### Implementation for User Story 1

- [x] T018 Create admin provider DTOs and validation in `src/api/services/oidc_service.go`.
- [x] T019 Implement provider create/update/delete/list/test methods in `src/api/services/oidc_service.go`.
- [x] T020 Implement admin OIDC provider handlers in `src/api/handlers/oidc.go`.
- [x] T021 Wire `OIDCRepository`, `OIDCService`, and admin OIDC routes in `src/api/main.go`.
- [x] T022 Add `AdminOIDCSection.vue` under `src/web/src/components/admin/` using existing admin card/form patterns and design tokens.
- [x] T023 Add Admin Settings navigation/rendering for the OIDC section in the existing admin page structure.
- [x] T024 Add component tests in `src/web/src/components/admin/__tests__/AdminOIDCSection.test.ts` for redacted secret handling, provider test statuses, and save errors.

**Checkpoint**: Admin can safely configure and test providers; no login flow required yet.

---

## Phase 4: User Story 2 - User signs in with OIDC (Priority: P1)

**Goal**: Linked users can sign in with enabled OIDC providers and receive existing app tokens.

**Independent Test**: Mock provider login succeeds, invalid validation paths fail, and local/WebAuthn login regressions still pass.

### Tests for User Story 2

- [x] T025 [P] Add OIDC login start/callback tests in `src/api/handlers/oidc_test.go` for successful login with an existing external identity.
- [x] T026 [P] Add exact failure tests for invalid state, replayed state, invalid nonce, invalid issuer, invalid audience, expired token, bad signature/JWKS, missing subject, and unverified email policy.
- [x] T027 [P] Add local auth regression tests in `src/api/handlers/auth_handler_test.go` and/or `src/api/services/auth_service_test.go` confirming password login and refresh behavior are unchanged.
- [x] T028 [P] Add WebAuthn regression coverage in existing `src/api/handlers/webauthn_test.go` if OIDC wiring touches auth token issuance.
- [x] T029 [P] Add LoginPage tests in `src/web/src/pages/__tests__/LoginPage.test.ts` for OIDC provider buttons and distinct error categories.

### Implementation for User Story 2

- [x] T030 Implement public enabled-provider listing in `src/api/handlers/oidc.go`.
- [x] T031 Implement OIDC login start with PKCE/state/nonce generation, relative redirect validation, and persisted `OIDCAuthState`.
- [x] T032 Implement provider code exchange and ID token validation in `src/api/services/oidc_service.go`.
- [x] T033 Implement callback handling that finds linked external identity, updates `LastLoginAt`, records audit events, and issues existing `AuthResponse` tokens without placing tokens in URL query strings.
- [x] T034 Add account-conflict response for verified/matching email with no linked identity, without creating or merging accounts.
- [x] T035 Wire public OIDC routes in `src/api/main.go` under existing auth rate limits.
- [x] T036 Update `src/web/src/pages/LoginPage.vue` to load public providers, render alternate sign-in buttons, start OIDC login, and display callback errors.
- [x] T037 Update `src/web/src/stores/auth.ts` only if needed to consume a callback/session-exchange response while preserving existing token storage. Not needed for the implemented callback flow; existing token storage remains unchanged.

**Checkpoint**: OIDC login works for linked identities; local auth remains intact.

---

## Phase 5: User Story 4 - Preserve local admin recovery (Priority: P1)

**Goal**: Every account mutation that could remove the final local admin recovery path is blocked and audited.

**Independent Test**: Existing admin user operations return `409` for final-local-admin removal and continue working for non-final admins.

### Tests for User Story 4

- [x] T038 [P] Add admin delete-user tests in `src/api/handlers/admin_test.go` for blocking deletion of the final local admin and allowing deletion when another local admin exists.
- [x] T039 [P] Add admin role-update tests in `src/api/handlers/admin_test.go` for blocking demotion of the final local admin.
- [x] T040 [P] Add service tests for password clearing/local-auth disable/OIDC-only conversion guard paths even if UI for disabling local login is not delivered in v1.
- [x] T041 [P] Add security event tests for `final_local_admin_blocked` without sensitive payloads.

### Implementation for User Story 4

- [x] T042 Integrate `AdminRecoveryService` into `src/api/handlers/admin.go` delete and role update flows.
- [x] T043 Add repository transaction helpers in `src/api/repository/admin_repository.go` so recovery checks and mutations cannot race.
- [x] T044 Add guard hooks to any OIDC-only conversion or local-auth-disable service methods introduced for account migration.
- [x] T045 Ensure errors use `409 Conflict` with the contract message and record security events.

**Checkpoint**: Recovery safety is enforced across current and planned account-conversion flows.

---

## Phase 6: User Story 3 - Existing users link and manage OIDC identities (Priority: P2)

**Goal**: Authenticated users can link/unlink OIDC identities without unsafe account merges.

**Independent Test**: Local user links provider, linked identity appears in Account Settings, unlink works unless it would remove the user's last sign-in method.

### Tests for User Story 3

- [x] T046 [P] Add linking start/callback tests in `src/api/handlers/oidc_test.go` for successful authenticated link.
- [x] T047 [P] Add conflict tests for external identity already linked to another user and matching-email non-merge.
- [x] T048 [P] Add unlink tests for success, not-owned identity, and blocked no-usable-sign-in-method case.
- [x] T049 [P] Add Account Settings component tests for list/link/unlink states and conflict messages.

### Implementation for User Story 3

- [x] T050 Implement protected OIDC link start and callback flows in `src/api/handlers/oidc.go` and `src/api/services/oidc_service.go`.
- [x] T051 Implement `GET /user/oidc-identities` and `DELETE /user/oidc-identities/:identityId`.
- [x] T052 Add linked-identity API wrappers to `src/web/src/api/client.ts`.
- [x] T053 Update `src/web/src/components/settings/SettingsAccountSection.vue` to show linked OIDC identities and link/unlink actions after the existing profile/security sections.
- [x] T054 Record OIDC link/unlink success/failure security events.

**Checkpoint**: Existing local users can migrate safely by explicit linking.

---

## Phase 7: User Story 5 - UX clarity and documentation (Priority: P3)

**Goal**: Admin/user-facing messages distinguish status and failure categories; setup docs are complete.

**Independent Test**: Mock each error category and verify UI copy; follow quickstart with mocked or real provider config.

### Tests for User Story 5

- [x] T055 [P] Add frontend tests for misconfiguration, denied access, validation failure, and account-link conflict messages.
- [x] T056 [P] Add backend response tests that map service errors to distinct HTTP status/message categories.

### Implementation for User Story 5

- [x] T057 Normalize OIDC service errors into typed sentinel errors in `src/api/services/oidc_service.go`.
- [x] T058 Map OIDC error categories in `src/api/handlers/oidc.go` without leaking internal provider errors.
- [x] T059 Add `docs/oidc-setup.md` with Entra ID, Pocket ID, redirect URI, scopes, and recovery admin guidance.
- [x] T060 Link OIDC setup docs from the Admin OIDC section if this repo has an existing admin help/docs pattern.

**Checkpoint**: The feature is understandable to admins and users.

---

## Phase 8: Quality gate and blast-radius checks

**Timing:** After Phases 6-7 complete, user acceptance testing on `beta` finishes, and proportional feedback adjustments are applied. This is the final security and engineering review gate before main branch merge. See `.squad/decisions/inbox/maximus-oidc-phase8-before-main.md` for the delivery guardrail.

- [ ] T061 Run `go test -v ./...` from `src/api`.
- [ ] T062 Run `npm run build` from `src/web`.
- [ ] T063 Run `npm test` from `src/web`.
- [ ] T064 Run `npm run lint` from `src/web`.
- [ ] T065 Run targeted manual smoke test: local login, WebAuthn login if available, OIDC provider listing, OIDC mock login, account linking, admin final-local-admin block.
- [ ] T066 Verify no logs, security events, API responses, frontend state, docs, or tests contain real client secrets, auth codes, ID/access tokens, refresh tokens, or PKCE verifiers.
- [ ] T067 Run a security audit focused on OIDC threat paths: provider configuration, redirect handling, state/nonce/PKCE, token validation, account linking conflicts, admin recovery safety, secret redaction, logs, and audit events.
- [ ] T068 Run a software engineering best-practices review for layered architecture, transaction boundaries, type safety, error handling, test coverage, UI consistency, maintainability, and blast-radius containment before main merge.

---

## Dependencies & Execution Order

- Phase 1 must complete before buildable OIDC code is added.
- Phase 2 blocks every user story.
- Phases 3 and 4 can proceed after Phase 2 and should complete before real login rollout.
- Phase 5 can proceed after Phase 2 and should complete before any OIDC-only migration behavior ships.
- Phase 6 depends on provider config and callback validation from Phases 3-4.
- Phase 7 can run in parallel after API error categories stabilize.
- Phase 8 is the final quality gate.

## Scope Summary

Estimated scope is large: backend auth/security/schema work, frontend login/settings/admin UX, documentation, and extensive regression coverage. The safest MVP is Phases 1-5: provider config, linked-identity login, and final-local-admin protection. Account linking UX (Phase 6) is required by issue #335 acceptance but can be delivered immediately after the MVP foundation because it reuses the same callback machinery.
