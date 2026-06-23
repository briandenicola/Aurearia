# Research: OIDC Login for Entra ID and Pocket ID

## Decision: Use Go API as the sole OIDC boundary

**Rationale**: Existing auth, JWT issuance, refresh token persistence, security events, and admin/user account data live in `src/api/`. Keeping OIDC in Go preserves the constitution's service boundary and avoids involving the Python agent.

**Alternatives considered**: Frontend-only OIDC validation was rejected because token validation, client secret handling, and account linking require server-side trust. Python agent integration was rejected because auth is outside agent responsibilities.

## Decision: Use a vetted OIDC verifier plus OAuth2 client

**Rationale**: The issue explicitly calls out avoiding hand-rolled token validation. `github.com/coreos/go-oidc/v3/oidc` with `golang.org/x/oauth2` is the standard Go approach for discovery, JWKS-backed ID token verification, issuer/audience validation, and OAuth2 code exchange. It integrates cleanly into service code and tests can use local mock HTTP servers.

**Alternatives considered**: Manual JWKS/token validation was rejected as risky. Delegating auth to a reverse proxy was rejected because the app needs account linking, audit events, provider administration, and local auth preservation.

## Decision: Dedicated provider and identity tables instead of only key-value settings

**Rationale**: Issue #335 requires one or more OIDC providers, provider-specific status/test controls, and external identity linkage. Dedicated `OIDCProvider`, `ExternalIdentity`, and `OIDCAuthState` tables give unique constraints and transaction boundaries that are awkward in the existing `AppSetting` key-value model.

**Alternatives considered**: Structured JSON in `AppSetting` was rejected because multi-provider secrets, redaction, and uniqueness constraints would be harder to validate and test.

## Decision: Use stateful short-lived OIDC auth transactions

**Rationale**: Authorization-code flow requires state, nonce, PKCE verifier, provider ID, flow type, redirect target, and optional linking user ID to survive provider redirects. A short-lived DB-backed `OIDCAuthState` supports app restarts and lets callbacks atomically mark state consumed to prevent replay.

**Alternatives considered**: Cookie-only state was rejected because it complicates mobile/PWA/browser compatibility and restart behavior. Stateless encrypted state was rejected for v1 because replay prevention and key rotation add complexity.

## Decision: Subject linkage is authoritative; email matching never auto-merges

**Rationale**: OIDC `sub` scoped by issuer/provider is the stable external identity. Email can change or be unverified. The issue explicitly forbids silent merges when OIDC email matches an existing local account.

**Alternatives considered**: Auto-link by verified email was rejected because it can join accounts without proof of current local account ownership.

## Decision: Enforce local admin recovery through a shared service

**Rationale**: Recovery safety spans delete user, demote admin, reset/disable password, and future OIDC-only conversion paths. A shared `AdminRecoveryService` prevents drift across handlers and keeps the rule testable.

**Alternatives considered**: Inline checks in each handler were rejected because this rule is security-critical and easy to miss as account operations grow.

## Decision: Redact secrets on every read and preserve existing secrets on redacted/empty updates

**Rationale**: Admin UIs often round-trip forms. A redacted `clientSecretConfigured` boolean plus write-only `clientSecret` avoids secret disclosure and prevents accidental secret erasure when a user edits non-secret fields.

**Alternatives considered**: Returning masked secret text was rejected because placeholders can be confused with real values and can accidentally overwrite stored secrets.
