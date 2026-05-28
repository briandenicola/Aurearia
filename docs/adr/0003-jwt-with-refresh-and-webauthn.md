# 3. JWT Auth with Refresh Tokens and WebAuthn Passkeys

Date: 2026-05-28
Status: Accepted

## Context

A personal PWA needs authentication that is secure, modern, and
low-friction on mobile devices. Three alternatives were weighed:

- **Pure password auth with session cookies** — dated UX, awkward on
  mobile, and cookies pair poorly with the SSE-over-proxy pattern
  used for agent streaming.
- **Full OAuth/OIDC against a third-party IdP** (Google, GitHub,
  Auth0) — overkill for a single-collector tool, adds an external
  runtime dependency, and surrenders the user's identity surface to
  a vendor.
- **Magic-link email auth** — fine for low-stakes apps, but slow on
  mobile and dependent on a working mail relay.

None of these satisfy the joint constraints of: offline-capable PWA,
SSE-friendly transport, no third-party IdP, and Face ID / Touch ID
class UX on mobile.

## Decision

We will use a layered approach:

- **JWT access tokens** — short-lived, HS256-signed, carried in the
  `Authorization: Bearer <token>` header on every authenticated
  request. Stateless validation, no per-request DB lookup.
- **Refresh tokens** — longer-lived, server-side stored and
  revocable. Silent renewal is handled client-side by an Axios 401
  interceptor with a **refresh queue** that deduplicates concurrent
  refreshes (multiple in-flight 401s share a single refresh promise).
- **WebAuthn / FIDO2 passkeys** — the primary auth UX on supported
  platforms (Face ID, Touch ID, Windows Hello, Android biometrics).
  Password remains as fallback for unsupported environments and
  initial account creation.
- **Client error handling** — WebAuthn `DOMException`s
  (`NotAllowedError`, `InvalidStateError`, `SecurityError`,
  `NotSupportedError`) are mapped to user-friendly messages at the
  component boundary; raw browser errors never reach the UI.

## Consequences

+ Mobile-first: passkey unlock is one biometric gesture.
+ Stateless access tokens fit the SSE proxy pattern cleanly — no
  cookie / CORS gymnastics across the Vue → Go → Python hop.
+ No third-party IdP dependency; the system self-hosts identity.
+ Refresh-token revocation gives us a server-side kill switch
  without invalidating every active session.
− WebAuthn requires careful client-side error handling — mitigated
  by the centralised `DOMException` mapping documented in
  `docs/authentication.md` and the per-component patterns in
  `useWebAuthn`.
− Refresh-queue logic must be correct: a naive implementation
  double-refreshes under concurrent 401s, burning refresh tokens
  and racing the server's rotation policy. The Axios interceptor's
  promise-sharing pattern is the canonical fix.
− Passkey support is not universal; the password fallback path
  remains a maintained surface, not a deprecated one.

## Related

- Constitution Principle XII (Authentication & Token Policy)
- Constitution Principle XIII (PWA / Mobile Interaction Rules)
- Constitution Principle XI (Security Hardening)
- `docs/authentication.md` — full auth guide
- `specs/_backlog/F007-webauthn-biometric-login.md`
- `src/web/src/api/client.ts` — Axios interceptor + refresh queue
