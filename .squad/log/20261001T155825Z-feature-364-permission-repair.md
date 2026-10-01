# Feature 364 Microphone Permission Repair

## Report

Edge desktop over HTTPS displayed
`Microphone permission was denied. You can continue by typing your notes.`
without first showing a microphone permission prompt.

## Cause

The browser speech-recognition API was expected to own the permission prompt.
That behavior is not reliable across Chromium browser/PWA permission states, and
`not-allowed` does not prove that a visible prompt was shown.

## Repair

- Request `getUserMedia({ audio: true })` from the same explicit dictation click.
- Stop every temporary stream track immediately after permission is granted.
- Start speech recognition only after the permission request succeeds.
- Show a requesting state while the browser prompt is pending.
- If permission is blocked, direct the collector to browser site permissions.
- Keep standard/prefixed recognition support and hide the control when
  recognition is unavailable.

## Verification

- Focused affected command passed 6 files and 82 tests.
- `task check:web` passed lint, strict type-check, full tests, and production
  build.
- Tamper proof: replacing the browser permission requester with `null` failed
  the permission/track-release test; restoring the requester passed it.

## Remaining

Real Chrome and Edge behavior over HTTPS must be recorded on the exact repair
candidate. Firefox behavior depends on whether that browser build exposes speech
recognition; typed notes remain the complete fallback when it does not.
