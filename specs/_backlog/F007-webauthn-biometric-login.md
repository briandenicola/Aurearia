---
title: WebAuthn and Biometric Login
id: F007
status: promoted
priority: P1
effort: M
value: 4
risk: 3
owner: Cassius+Aurelia
created: 2026-05-28
updated: 2026-05-28
---

## Summary

Passwordless FIDO2 authentication (Face ID, Touch ID, fingerprint). Users register passkeys in Settings → Account, then log in biometrically without password. Graceful fallback to password-based login; DOMException handling for user cancellation and platform errors.

## Acceptance Criteria

- When user clicks "Register Passkey" in Settings, WebAuthn registration ceremony is initiated
- When user completes biometric registration (Face ID/fingerprint), credential is stored server-side
- When user logs in with passkey, WebAuthn authentication ceremony verifies credential with biometric
- When user cancels biometric prompt, frontend catches NotAllowedError and returns to password form
- When platform lacks WebAuthn support, "Register Passkey" button is hidden

## Constitution Alignment

**Principle XII (Authentication & Token Policy):** WebAuthn credentials are FIDO2-compliant; server stores public key; session tokens remain unchanged (JWT + refresh).
**Principle XIII (PWA/Mobile Interaction Rules):** WebAuthn requires secure context (HTTPS in production, localhost for dev); CSP allows `webauthn` protocol.
**Principle XI (Security Hardening):** Challenge validation prevents replay attacks; credential attestation validated per FIDO2 spec.

## Implementation Notes

- Passkey registration: `POST /auth/register-passkey` (handles attestation challenge + verification)
- Passkey login: `POST /auth/authenticate-passkey` (handles assertion challenge + user verification)
- Credential storage: `UserCredential` model (userID, credentialID, publicKey, sign_count, created_at)
- Client error handling: catch `NotAllowedError` (user cancelled), `SecurityError` (bad context), `UnknownError` (platform issue)
- Fallback: if WebAuthn fails or unsupported, user can still log in with password

## Open Questions

None — feature shipped.

## Notes

Retroactive card created 2026-05-28 for governance traceability under Constitution §0 (Hierarchy) Phase 2. Uses Go `github.com/duo-labs/webauthn` library; frontend uses `@simplewebauthn/browser` for ceremony orchestration. Requires HTTPS in production; localhost dev allowed.
