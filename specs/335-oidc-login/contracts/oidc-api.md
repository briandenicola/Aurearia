# API Contract: OIDC Login for Entra ID and Pocket ID

Base path: `/api`

## Public provider discovery

### `GET /auth/oidc/providers`

Returns enabled providers safe for unauthenticated login UI.

**Response 200**

```json
{
  "providers": [
    {
      "id": 1,
      "name": "entra-work",
      "displayName": "Microsoft",
      "providerType": "entra"
    }
  ]
}
```

Secrets and issuer/client details are intentionally omitted unless needed by UI; client secret is never returned.

## Start OIDC login

### `POST /auth/oidc/:providerId/start`

Starts login flow and returns provider authorization URL.

**Request**

```json
{
  "redirectPath": "/"
}
```

**Response 200**

```json
{
  "authorizationUrl": "https://provider.example/authorize?...",
  "expiresAt": "2026-06-23T16:00:00Z"
}
```

**Errors**

- `400` invalid redirect path or provider ID
- `404` provider not found
- `409` provider disabled
- `500` provider discovery/configuration failure

## OIDC callback

### `GET /auth/oidc/:providerId/callback?code=...&state=...`

Completes login flow. MVP backend implementation returns the existing `AuthResponse` JSON body directly from this callback after provider validation; it does not redirect with app JWTs or refresh tokens in URL query strings. A later frontend callback/session-exchange wrapper may replace this transport shape without changing token semantics.

**Successful app session exchange response shape**

```json
{
  "token": "jwt-access-token",
  "refreshToken": "rt_refresh-token",
  "user": {
    "id": 1,
    "username": "collector",
    "role": "user",
    "email": "collector@example.com",
    "avatarPath": "",
    "isPublic": false,
    "bio": "",
    "zipCode": ""
  }
}
```

**Errors**

- `400` denied/cancelled provider response, invalid state, invalid code, invalid nonce
- `401` token validation failure
- `409` matching local email exists but identity is not linked
- `500` provider configuration or token exchange failure

## Account linking

### `POST /auth/oidc/:providerId/link/start`

Protected. Starts a linking flow for the current user.

**Request**

```json
{
  "redirectPath": "/settings"
}
```

**Response 200**

```json
{
  "authorizationUrl": "https://provider.example/authorize?...",
  "expiresAt": "2026-06-23T16:00:00Z"
}
```

### `GET /auth/oidc/:providerId/link/callback?code=...&state=...`

Completes linking flow. Requires callback state created by an authenticated user. The callback state identifies the user; bearer token may not be present after provider redirect.

**Response 200**

```json
{
  "message": "OIDC identity linked",
  "identity": {
    "id": 10,
    "providerId": 1,
    "providerDisplayName": "Microsoft",
    "issuer": "https://login.microsoftonline.com/tenant/v2.0",
    "subjectPreview": "abc123...",
    "email": "collector@example.com",
    "emailVerified": true,
    "createdAt": "2026-06-23T15:59:00Z"
  }
}
```

**Errors**

- `400` invalid callback state or claims
- `409` external identity already linked to another user
- `500` provider configuration or token exchange failure

### `GET /user/oidc-identities`

Protected. Lists linked identities for current user.

**Response 200**

```json
{
  "identities": [
    {
      "id": 10,
      "providerId": 1,
      "providerDisplayName": "Microsoft",
      "issuer": "https://login.microsoftonline.com/tenant/v2.0",
      "subjectPreview": "abc123...",
      "email": "collector@example.com",
      "emailVerified": true,
      "createdAt": "2026-06-23T15:59:00Z",
      "lastLoginAt": null
    }
  ]
}
```

### `DELETE /user/oidc-identities/:identityId`

Protected. Unlinks an identity from current user.

**Response 200**

```json
{
  "message": "OIDC identity unlinked"
}
```

**Errors**

- `404` identity not found for current user
- `409` unlink would leave the account without a usable sign-in method

## Admin provider management

### `GET /admin/oidc/providers`

Admin only.

**Response 200**

```json
{
  "providers": [
    {
      "id": 1,
      "name": "entra-work",
      "displayName": "Microsoft",
      "providerType": "entra",
      "enabled": true,
      "issuerUrl": "https://login.microsoftonline.com/tenant/v2.0",
      "clientId": "client-id",
      "clientSecretConfigured": true,
      "scopes": ["openid", "profile", "email"],
      "callbackPath": "/api/auth/oidc/1/callback",
      "lastTestStatus": "ok",
      "lastTestMessage": "Discovery succeeded"
    }
  ]
}
```

### `POST /admin/oidc/providers`

Admin only. Creates provider.

### `PUT /admin/oidc/providers/:providerId`

Admin only. Updates provider. Omitted or empty `clientSecret` preserves the existing secret unless a separate explicit clear-secret field is implemented.

### `DELETE /admin/oidc/providers/:providerId`

Admin only. Deletes provider only when safe.

### `POST /admin/oidc/providers/:providerId/test`

Admin only. Runs discovery/metadata validation without exposing secrets.

**Response 200**

```json
{
  "available": true,
  "message": "Discovery succeeded",
  "issuer": "https://provider.example",
  "authorizationEndpoint": "https://provider.example/oauth2/authorize",
  "tokenEndpoint": "https://provider.example/oauth2/token"
}
```

## Final-local-admin protection

Existing admin/user endpoints that can remove a local admin recovery path return:

**Response 409**

```json
{
  "error": "At least one admin with usable local credentials is required for recovery"
}
```

Applicable flows include deleting the final local admin, demoting the final local admin, disabling local login for the final local admin, or converting the final local admin to OIDC-only.
