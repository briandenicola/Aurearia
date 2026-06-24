# Feature Specification: OIDC Login for Entra ID and Pocket ID

**Feature Branch**: `335-oidc-login`  
**Created**: 2026-06-23  
**Status**: Draft  
**Input**: GitHub issue #335: "Add OIDC login support for Entra ID and Pocket ID"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Admin configures OIDC providers (Priority: P1)

An administrator can configure Microsoft Entra ID and Pocket ID as additional sign-in providers without exposing provider secrets back to the browser.

**Why this priority**: No OIDC sign-in can work until providers are configured, validated, and safely stored.

**Independent Test**: Configure disabled, invalid, and valid providers from Admin Settings; verify read responses redact secrets and provider status/test results clearly report success or misconfiguration.

**Acceptance Scenarios**:

1. **Given** an admin is signed in, **When** they save an Entra ID provider with tenant-specific issuer metadata, client ID, client secret, scopes, and callback path, **Then** the provider is persisted and shown on login only when enabled.
2. **Given** an admin is signed in, **When** they save a Pocket ID provider with its issuer discovery URL, **Then** discovery succeeds against the standard `/.well-known/openid-configuration` endpoint.
3. **Given** provider settings include a client secret, **When** the admin settings API returns provider details, **Then** the secret value is never returned and only a configured/not-configured flag is exposed.
4. **Given** provider settings are changed, **When** the save completes, **Then** a security event records the configuration change without tokens, authorization codes, or secrets.

---

### User Story 2 - User signs in with OIDC (Priority: P1)

A user can choose an enabled Entra ID or Pocket ID provider from the login page, complete the authorization-code flow, and receive the same app JWT/refresh-token session used by local login.

**Why this priority**: This is the primary user-facing value of the feature and must preserve local login behavior.

**Independent Test**: Use mocked provider discovery/JWKS/token/userinfo responses to complete login; verify valid flows issue app tokens and invalid issuer, audience, expiry, state, or nonce fail safely.

**Acceptance Scenarios**:

1. **Given** a provider is enabled, **When** an unauthenticated user opens the login page, **Then** the provider appears as an alternate sign-in button.
2. **Given** an OIDC identity is already linked to a user, **When** the provider callback validates successfully, **Then** the app issues the normal `AuthResponse` payload and redirects to the app.
3. **Given** a callback has an invalid state, nonce, issuer, audience, signature, expiry, missing subject, or unverified/missing email where required, **When** the callback is processed, **Then** login is denied with a non-secret error and a failure event is recorded.
4. **Given** a local username/password account exists, **When** OIDC is enabled, **Then** the existing local login, registration, password reset, WebAuthn/passkey, and admin flows continue to work.

---

### User Story 3 - Existing users link and manage OIDC identities (Priority: P2)

A signed-in local user can link an OIDC identity from Account Settings and later unlink it without unsafe silent account merges.

**Why this priority**: Existing collectors need a safe migration path from local credentials to external identities.

**Independent Test**: Sign in locally, start a linking flow, complete provider callback, list the linked identity, and unlink it while verifying email-match conflicts are not silently merged.

**Acceptance Scenarios**:

1. **Given** a local user is authenticated, **When** they start linking an enabled provider and complete callback validation, **Then** the external identity is associated with their current user.
2. **Given** an OIDC login email matches an existing local account but the external identity is not linked, **When** login completes validation, **Then** the app blocks automatic merge and explains that the user must link while signed in.
3. **Given** a user has linked identities, **When** they view Account Settings, **Then** each linked provider displays provider name, issuer, subject-safe identifier, verified email, and linked date.
4. **Given** unlinking would leave the user with no usable sign-in method, **When** unlink is requested, **Then** the app blocks the operation with a clear error.

---

### User Story 4 - System preserves local admin recovery (Priority: P1)

The application always keeps at least one admin who can recover the app through local credentials even if OIDC is unavailable or misconfigured.

**Why this priority**: OIDC is an external dependency; recovery must not depend on provider availability.

**Independent Test**: Attempt every operation that could remove the final local admin recovery path and verify it is blocked while equivalent operations on non-final admins are allowed.

**Acceptance Scenarios**:

1. **Given** there is exactly one admin with local credentials, **When** an admin tries to delete that user, remove their admin role, clear/disable their password, or convert them to OIDC-only, **Then** the operation is blocked and audited.
2. **Given** at least two admins have usable local credentials, **When** one admin is converted to OIDC-only or loses admin role, **Then** the operation can proceed if normal authorization rules pass.
3. **Given** OIDC admins exist, **When** evaluating recovery safety, **Then** OIDC-only admins do not count toward the required local admin.

---

### User Story 5 - Admin and user UX explains OIDC status/errors (Priority: P3)

The frontend distinguishes disabled providers, misconfiguration, denied access, validation failures, and account-link conflicts.

**Why this priority**: Clear messages reduce lockout risk and support burden after the secure backend behavior exists.

**Independent Test**: Trigger each error category through mocked API responses and verify the login, account settings, and admin settings surfaces render distinct, actionable messages.

**Acceptance Scenarios**:

1. **Given** provider discovery fails, **When** an admin tests the provider, **Then** the UI reports provider misconfiguration rather than a generic login failure.
2. **Given** a user denies consent or cancels at the provider, **When** control returns to the app, **Then** the login page shows a denied/cancelled message.
3. **Given** an unlinked matching email conflict occurs, **When** the callback returns, **Then** the login page directs the user to sign in locally and link from Account Settings.

### Edge Cases

- Provider issuer URL is malformed, non-HTTPS outside local development, unreachable, or serves invalid discovery metadata.
- Provider rotates signing keys; cached discovery/JWKS must refresh safely without accepting invalid signatures.
- Callback is replayed, expired, has mismatched state/nonce, wrong provider ID, or mismatched redirect URI.
- Email claim is absent or unverified; subject remains the stable identity key and email is not enough to link accounts.
- Two providers emit the same subject; identity uniqueness must include provider and issuer.
- Admin deletes, demotes, resets, disables, or converts accounts in combinations that could leave zero local admin recovery accounts.
- OIDC is misconfigured after a user disables local credentials; recovery rule must still protect at least one separate local admin.
- Frontend receives provider records with redacted secrets and must not treat placeholder secret text as an actual secret.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support multiple configured OIDC providers with enabled flag, display name, provider type, issuer URL, client ID, client secret, scopes, redirect/callback path, and timestamps.
- **FR-002**: System MUST support Microsoft Entra ID tenant-specific issuer metadata and Pocket ID standard OIDC discovery.
- **FR-003**: System MUST never expose OIDC client secrets through frontend/admin read APIs, logs, audit events, errors, or generated docs.
- **FR-004**: System MUST expose only enabled, non-secret public provider metadata to unauthenticated clients for login button rendering.
- **FR-005**: System MUST implement authorization-code flow with PKCE, state, and nonce for OIDC login and account linking.
- **FR-006**: System MUST validate issuer, audience/client ID, nonce/state, expiry, signature, subject, and email/verified-email policy before issuing app tokens.
- **FR-007**: System MUST store OIDC provider configuration and external identity linkage separately from local password and WebAuthn credentials.
- **FR-008**: System MUST issue the existing JWT access token and refresh token response shape after successful OIDC login.
- **FR-009**: System MUST preserve existing local username/password login, registration, password reset, WebAuthn/passkey, JWT refresh, and admin flows unless a specific safety rule blocks an unsafe operation.
- **FR-010**: System MUST allow authenticated users to link and unlink OIDC identities from Account Settings.
- **FR-011**: System MUST block silent merge when an OIDC email matches an existing local account unless the current authenticated user explicitly completes a linking flow.
- **FR-012**: System MUST always retain at least one admin with usable local credentials; OIDC-only admins MUST NOT satisfy this recovery requirement.
- **FR-013**: System MUST block deleting, demoting, disabling local auth for, or converting the final local admin recovery account to OIDC-only.
- **FR-014**: System MUST record security events for OIDC login success/failure, linking, unlinking, provider config changes, provider test failures, and blocked final-local-admin operations.
- **FR-015**: System MUST avoid logging tokens, authorization codes, client secrets, raw ID tokens, raw access tokens, and refresh tokens.
- **FR-016**: Login page MUST show enabled OIDC providers as alternate sign-in buttons while keeping local login and WebAuthn options available.
- **FR-017**: Account Settings MUST show linked OIDC identities and link/unlink actions.
- **FR-018**: Admin Settings MUST provide OIDC provider create/update/delete/list/test controls.
- **FR-019**: Error messages MUST distinguish provider misconfiguration, provider-denied access, OIDC validation failure, and account-linking conflicts.
- **FR-020**: Documentation MUST explain setup for Entra ID and Pocket ID, including redirect URI registration and recovery admin guidance.

### Key Entities

- **OIDCProvider**: Admin-managed provider configuration including provider type, display name, issuer, client ID, secret storage, scopes, callback path, enabled state, discovery status, and timestamps.
- **ExternalIdentity**: Linkage between a local user and an OIDC identity using provider ID/name, issuer, subject, verified email, display name, and timestamps.
- **OIDCAuthState**: Short-lived login/linking transaction state with provider, PKCE verifier/challenge, nonce, redirect target, optional authenticated user ID, expiry, and consumed timestamp.
- **User**: Existing local account; gains local-auth usability semantics while preserving username/password, WebAuthn, role, and profile behavior.
- **SecurityEvent**: Existing audit event model extended with OIDC-specific event types.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Local username/password login and WebAuthn login pass their existing regression tests unchanged after OIDC is added.
- **SC-002**: Mocked Entra ID and Pocket ID providers can each complete authorization-code login and issue the existing app `AuthResponse`.
- **SC-003**: Tests reject invalid issuer, audience, signature/JWKS, expiry, state, nonce, missing subject, and unlinked matching-email conflict cases.
- **SC-004**: Admin settings read APIs never include client secret values; tests verify saved secrets are redacted on read and not overwritten by empty/redacted placeholders.
- **SC-005**: Final-local-admin protection tests cover delete, demote, local-auth disable/conversion, and OIDC-only conversion attempts.
- **SC-006**: Account linking tests verify successful link, duplicate external identity conflict, email-match non-merge, unlink success, and unlink blocked when no usable sign-in method remains.
- **SC-007**: Documentation enables an admin to configure one Entra tenant and one Pocket ID issuer without reading source code.

## Assumptions

- OIDC support is implemented in the Go API only; the Python agent service is not involved.
- SQLite remains the only database and GORM AutoMigrate is used for additive schema changes.
- Client secrets are stored using the existing settings/database storage model initially; encryption-at-rest is out of scope unless an existing secret-encryption helper is discovered during implementation.
- `PublicAppURL` or equivalent deployment base URL is required to construct stable redirect URIs in production.
- Local development may allow `http://localhost` issuer/redirect URLs; non-local production issuers and redirect URLs should require HTTPS.
- One or more OIDC providers are supported from the start because the issue requires both Entra ID and Pocket ID.
