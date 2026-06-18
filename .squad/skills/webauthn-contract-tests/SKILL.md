---
name: "webauthn-contract-tests"
description: "How to lock WebAuthn begin-ceremony API response shapes against browser/client challenge mismatches"
domain: "api-design"
confidence: "high"
source: "earned"
---

## Context

Use this when changing WebAuthn/passkey registration or login endpoints. Browser APIs are strict about where `challenge`, `rpId`, and credential IDs live, and schema wrapper mismatches can appear as missing challenge data on Safari/PWA clients.

## Patterns

- Test the exact JSON shape the client consumes, not only status codes.
- For login begin, assert `options.challenge` is non-empty when the API contract says `options` is passed to `navigator.credentials.get({ publicKey: options })`.
- Assert the response challenge matches the server-side WebAuthn session challenge so the begin response and finish session cannot drift.
- Include `rpId` and at least one `allowCredentials` entry in the regression to catch credential loading or base64url encoding regressions.
- Keep missing-session and expired-session tests separate from begin-contract tests.

## Examples

See `src/api/handlers/webauthn_test.go`:
- `TestWebAuthnHandlerLoginBeginReturnsRequestOptionsWithChallenge`
- `TestWebAuthnHandlerLoginFinishExpiredSession`
- `TestWebAuthnHandlerLoginFinishMissingSession`

## Anti-Patterns

- Do not only assert that `BeginLogin` returns HTTP 200; that can still hide a missing or incorrectly nested challenge.
- Do not return go-webauthn's `CredentialAssertion` wrapper under another `options` key unless the frontend explicitly expects `options.publicKey.challenge`.
- Do not treat credential listing success as proof login begin is usable; registered credentials and active challenge sessions are separate contracts.
