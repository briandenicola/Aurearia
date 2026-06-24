# Data Model: OIDC Login for Entra ID and Pocket ID

## OIDCProvider

Represents an admin-configured OIDC provider.

| Field | Type | Validation / Notes |
|---|---|---|
| ID | uint | Primary key |
| Name | string | Stable slug or generated key; unique |
| DisplayName | string | Required; shown on login buttons |
| ProviderType | string | Required enum: `entra`, `pocket_id`, `generic` |
| Enabled | bool | Disabled providers cannot start login/link flows |
| IssuerURL | string | Required; HTTPS except localhost/local development; unique with ClientID |
| ClientID | string | Required |
| ClientSecret | string | Write-only in API responses; not logged |
| Scopes | string/list | Defaults to `openid profile email`; must include `openid` |
| CallbackPath | string | Required fixed relative path, e.g. `/api/auth/oidc/callback/{providerId}` |
| RequireVerifiedEmail | bool | Default true for account matching decisions |
| CreatedAt | time.Time | GORM managed |
| UpdatedAt | time.Time | GORM managed |
| LastTestedAt | *time.Time | Optional status metadata |
| LastTestStatus | string | `unknown`, `ok`, `failed` |
| LastTestMessage | string | Non-secret diagnostic summary |

### Constraints

- Provider read DTOs MUST return `clientSecretConfigured: boolean` and MUST NOT return `ClientSecret`.
- Deleting a provider should be blocked if linked identities exist unless implementation defines an explicit safe cascade/unlink behavior.
- Provider discovery must validate issuer consistency.

## ExternalIdentity

Links a local user to one OIDC subject.

| Field | Type | Validation / Notes |
|---|---|---|
| ID | uint | Primary key |
| UserID | uint | Required FK to `User` |
| ProviderID | uint | Required FK to `OIDCProvider` |
| Issuer | string | Required; normalized from verified token issuer |
| Subject | string | Required OIDC `sub`; never empty |
| Email | string | Optional claim snapshot |
| EmailVerified | bool | From provider claim |
| DisplayName | string | Optional claim snapshot |
| LastLoginAt | *time.Time | Updated after successful OIDC login |
| CreatedAt | time.Time | GORM managed |
| UpdatedAt | time.Time | GORM managed |

### Constraints

- Unique `(provider_id, issuer, subject)`.
- Unique `(user_id, provider_id, issuer, subject)`.
- Email is informational and MUST NOT be used to silently merge users.

## OIDCAuthState

Short-lived login or linking transaction.

| Field | Type | Validation / Notes |
|---|---|---|
| ID | uint | Primary key |
| StateHash | string | Unique hash of opaque browser state |
| ProviderID | uint | Required |
| FlowType | string | `login` or `link` |
| UserID | *uint | Required for `link`, nil for `login` |
| PKCEVerifier | string | Raw PKCE verifier required for token exchange; treat as secret and never log |
| NonceHash | string | Hash of nonce sent in auth request |
| RedirectPath | string | Relative app path only; reject absolute external URLs |
| ExpiresAt | time.Time | Short TTL, e.g. 10 minutes |
| ConsumedAt | *time.Time | Set atomically on callback |
| CreatedAt | time.Time | GORM managed |

### State transitions

1. `created`: start endpoint persists state with expiry.
2. `consumed`: callback atomically validates and marks state consumed.
3. `expired`: callback or cleanup ignores states past expiry.

Callbacks for consumed or expired states fail with a validation error and security event.

## User

Existing local account model.

### Relevant existing fields

- `ID`, `Username`, `Email`, `PasswordHash`, `Role`, `LockedUntil`, profile fields.

### Planned local-auth semantics

- A user has usable local credentials when `PasswordHash` contains a valid local password hash and local login is not disabled for the account.
- If implementation adds `LocalAuthDisabled bool`, final-local-admin checks must treat disabled local auth as unusable.
- If implementation chooses not to support disabling local login in v1, OIDC-only conversion tasks should be scoped to future-proof service methods and tests for any code paths that clear or disable password credentials.

## SecurityEvent

Existing model extended with event type constants.

### New event types

- `oidc_login_success`
- `oidc_login_failure`
- `oidc_link_success`
- `oidc_link_failure`
- `oidc_unlink_success`
- `oidc_unlink_failure`
- `oidc_provider_config_changed`
- `oidc_provider_test_failure`
- `final_local_admin_blocked`

### Logging constraints

Messages can include provider display name, provider ID, username, and high-level reason. Messages MUST NOT include client secrets, authorization codes, ID/access tokens, refresh tokens, PKCE verifiers, or raw claims payloads.
