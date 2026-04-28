# Authentication Guide

This guide covers all authentication mechanisms in Ancient Coins — JWT tokens, refresh tokens, WebAuthn biometrics, and API keys — along with code examples and production security recommendations.

## JWT Authentication (Primary)

JWT is the primary authentication method. Users log in with a username and password to receive a short-lived access token.

### Registration

The first user to register is automatically assigned the **admin** role. All subsequent users are regular users.

Registration requires a **username**, **password**, and **email** (validated as a proper email format).

```sh
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "password": "s3cur3Pa$$word", "email": "alice@example.com"}'
```

### Login

```sh
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "password": "s3cur3Pa$$word"}'
```

**Response:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "rt_a1b2c3d4e5f6...",
  "user": {
    "id": 1,
    "username": "alice",
    "role": "admin",
    "email": "alice@example.com",
    "avatarPath": "/uploads/avatars/alice.jpg",
    "isPublic": false,
    "bio": ""
  }
}
```

The registration endpoint returns the same response shape.

### Using the Access Token

Pass the token in the `Authorization` header on all authenticated requests:

```sh
curl http://localhost:8080/api/coins \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

### Current User

```sh
curl http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Response:**

```json
{
  "id": 1,
  "username": "alice",
  "role": "admin",
  "email": "alice@example.com",
  "avatarPath": "/uploads/avatars/alice.jpg",
  "isPublic": false,
  "bio": "",
  "emailMissing": false,
  "createdAt": "2024-01-15T08:30:00Z"
}
```

#### Email Migration for Legacy Users

Users who registered before email was required will have `emailMissing: true` in the response. The frontend displays a dismissible modal prompting them to add their email address. The prompt can be dismissed for 7 days (tracked via `localStorage`), after which it reappears.

### Token Details

| Property | Value |
| -------- | ----- |
| **Expiry** | 15 minutes |
| **Signing secret** | `JWT_SECRET` environment variable |
| **Header** | `Authorization: Bearer <token>` |
| **Middleware** | `src/api/middleware/auth.go` — validates the JWT and populates `userId` and `userRole` in the Gin context |

---

## Refresh Tokens

Refresh tokens allow the frontend to silently obtain new access tokens without requiring the user to log in again.

### How It Works

1. On login, the server returns both an **access token** (15 min) and a **refresh token** (30 days).
2. When the access token expires, the client sends the refresh token to get a new pair.
3. The old refresh token is **revoked** on each refresh — this is a rolling window. Each refresh token can only be used once.

### Refreshing Tokens

```sh
curl -X POST http://localhost:8080/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refreshToken": "rt_a1b2c3d4e5f6..."}'
```

**Response:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...(new access token)",
  "refreshToken": "rt_f6e5d4c3b2a1...(new refresh token)"
}
```

### Token Format & Storage

| Property | Value |
| -------- | ----- |
| **Format** | `rt_` prefix + 32 random hex bytes |
| **Server storage** | SHA-256 hash in the database (`RefreshToken` model) |
| **Client storage** | `localStorage` on the frontend |
| **Lifetime** | 30 days (rolling — resets on each refresh) |

### Frontend Behavior

The frontend uses an **axios interceptor** that automatically handles token refresh:

- When a request returns **401 Unauthorized**, the interceptor sends the refresh token to `/api/auth/refresh`.
- While the refresh is in flight, any **concurrent requests** are queued and replayed with the new token once the refresh completes. This avoids race conditions when multiple API calls fail at the same time.
- If the refresh itself fails (e.g., the refresh token has expired), the user is redirected to the login page.

---

## WebAuthn / FIDO2 (Biometric Login)

WebAuthn enables passwordless login using Face ID, Touch ID, fingerprint sensors, and other platform authenticators. This uses the `go-webauthn/webauthn` v0.16.1 library on the backend.

### Configuration

| Environment Variable | Description | Default |
| -------------------- | ----------- | ------- |
| `WEBAUTHN_RP_ID` | Relying Party ID — your domain name | `localhost` |
| `WEBAUTHN_ORIGIN` | Full origin URL (supports comma-separated list for multiple origins) | `http://localhost:8080` |

The server also supports **dynamic origin detection** — it auto-detects the request origin and adds it to the allowed origins list, which simplifies development setups.

### Authenticator Settings

| Setting | Value | Notes |
| ------- | ----- | ----- |
| `AuthenticatorAttachment` | `platform` | Only biometric authenticators (Face ID, Touch ID, fingerprint) — no USB security keys |
| `ResidentKey` | `preferred` | Enables discoverable credentials for iOS Passwords integration |

### Registering a Credential

Registration requires an existing JWT session — the user must be logged in first.

**Step 1 — Begin registration:**

```sh
curl -X POST http://localhost:8080/api/auth/webauthn/register/begin \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

Returns `PublicKeyCredentialCreationOptions` for the browser.

**Step 2 — Browser ceremony:**

The frontend calls `navigator.credentials.create()` with the options from Step 1. The user sees a Face ID / Touch ID / fingerprint prompt.

**Step 3 — Finish registration:**

```sh
curl -X POST http://localhost:8080/api/auth/webauthn/register/finish \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{ ... credential response from browser ... }'
```

The credential is stored in the database and linked to the user's account.

### Logging In with Biometrics

WebAuthn login is **public** — no existing JWT is required.

**Step 1 — Begin login:**

```sh
curl -X POST http://localhost:8080/api/auth/webauthn/login/begin \
  -H "Content-Type: application/json" \
  -d '{"username": "alice"}'
```

Returns a challenge for the browser.

**Step 2 — Browser ceremony:**

The frontend calls `navigator.credentials.get()` with the challenge. The user sees a biometric prompt.

**Step 3 — Finish login:**

```sh
curl -X POST "http://localhost:8080/api/auth/webauthn/login/finish?username=alice" \
  -H "Content-Type: application/json" \
  -d '{ ... assertion response from browser ... }'
```

**Response:** Returns a JWT access token and refresh token, same as a password login.

### Managing Credentials

**Check if a user has registered credentials:**

```sh
curl "http://localhost:8080/api/auth/webauthn/check?username=alice"
```

**List your credentials** (requires JWT):

```sh
curl http://localhost:8080/api/auth/webauthn/credentials \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Delete a credential** (requires JWT):

```sh
curl -X DELETE http://localhost:8080/api/auth/webauthn/credentials/CREDENTIAL_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Frontend Behavior

- The login page **remembers the last username** in `localStorage` and shows a biometric login button when that user has registered WebAuthn credentials.
- Session data for in-progress ceremonies is stored **in-memory** on the server (a Go `map`). If the server restarts mid-ceremony, the user will need to start the registration or login flow again.
- **Known limitation:** In-memory ceremony sessions have **no TTL** — abandoned registration/login flows accumulate indefinitely. This is tracked as finding B-7 in the [security analysis](security-analysis.md).

---

## API Keys (Programmatic Access)

API keys provide a simple authentication method for scripts, CI pipelines, and third-party integrations.

### Generating a Key

```sh
curl -X POST http://localhost:8080/api/auth/api-keys \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "My Import Script"}'
```

**Response:**

```json
{
  "id": 1,
  "name": "My Import Script",
  "key": "ak_a1b2c3d4e5f6...full key shown only once...",
  "keyPrefix": "d4e5f6a1"
}
```

> **Important:** The full API key is returned **only once** at creation. Copy it immediately — it cannot be retrieved later. The `keyPrefix` (last 8 characters) is stored for identification.

### Using an API Key

Pass the key in the `X-API-Key` header:

```sh
curl http://localhost:8080/api/coins \
  -H "X-API-Key: ak_a1b2c3d4e5f6..."
```

### Listing Keys

```sh
curl http://localhost:8080/api/auth/api-keys \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

Returns all keys for your account (prefix and name only — never the full key).

### Revoking a Key

```sh
curl -X DELETE http://localhost:8080/api/auth/api-keys/1 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

Revoked keys are soft-deleted and immediately rejected on future requests.

### Key Details

| Property | Value |
| -------- | ----- |
| **Format** | `ak_` prefix + 32 random hex bytes |
| **Server storage** | SHA-256 hash in the database (`ApiKey` model) |
| **Identification** | `KeyPrefix` — last 8 characters, stored for display |
| **Name** | Optional label for identifying the key's purpose |
| **Tracking** | `LastUsedAt` timestamp updated on each use |

---

## Auth Middleware Priority

The auth middleware in `src/api/middleware/auth.go` checks credentials in this order:

1. **API Key** — Check the `X-API-Key` header. If present, hash the key and look it up in the database. Reject if the key is revoked.
2. **JWT** — If no API key is provided, check the `Authorization: Bearer <token>` header. Validate the JWT signature and expiry.

Both methods populate `userId` and `userRole` in the Gin context, so downstream handlers don't need to know which auth method was used.

### JWT Query Parameter Fallback

In addition to the `Authorization` header, the middleware accepts a `?token=` query parameter (`src/api/middleware/auth.go`, lines 35–51). This is used for image proxy URLs where a header cannot be set (e.g., `<img>` tags). The query param is internally converted to a `Bearer` header and follows the same validation path.

### JWT Claim Fields

The access token contains these claims (set in `src/api/services/auth_service.go`, lines 82–93):

| Claim | Type | Description |
|-------|------|-------------|
| `userId` | number | Database user ID |
| `username` | string | Login username |
| `role` | string | `admin` or `user` |
| `exp` | number | Expiry (Unix timestamp, 15 minutes from issue) |
| `iat` | number | Issued-at (Unix timestamp) |

---

## Production Security Checklist

Before deploying to production, review these settings:

### JWT Secret

Change the `JWT_SECRET` environment variable to a strong, random value (minimum 32 characters):

```sh
export JWT_SECRET=$(openssl rand -base64 48)
```

### WebAuthn Domain

Set `WEBAUTHN_RP_ID` to your production domain and `WEBAUTHN_ORIGIN` to the full URL:

```sh
export WEBAUTHN_RP_ID=coins.example.com
export WEBAUTHN_ORIGIN=https://coins.example.com
```

> **Note:** The RP ID is bound to the domain — credentials registered on one domain cannot be used on another.

### API Key Handling

- Treat API keys like passwords — they grant full access to the user's account.
- Keys are shown **only once** at generation. Store them in a secrets manager or password vault.

### Refresh Token Storage

Refresh tokens are currently stored in `localStorage` on the frontend. This is acceptable for a PWA but be aware that `localStorage` is accessible to any JavaScript running on the same origin.

### Summary

| Setting | Environment Variable | Example Value |
| ------- | -------------------- | ------------- |
| JWT signing secret | `JWT_SECRET` | Output of `openssl rand -base64 48` |
| WebAuthn domain | `WEBAUTHN_RP_ID` | `coins.example.com` |
| WebAuthn origin | `WEBAUTHN_ORIGIN` | `https://coins.example.com` |
