# Feature 364 Microphone Permission Repair

## Report

Edge desktop over HTTPS displayed
`Microphone permission was denied. You can continue by typing your notes.`
without first showing a microphone permission prompt.

## Cause

The browser speech-recognition API was expected to own the permission prompt.
That behavior is not reliable across Chromium browser/PWA permission states, and
`not-allowed` does not prove that a visible prompt was shown.

Follow-up inspection found the application-owned root cause:
`Permissions-Policy: camera=(self), microphone=(), geolocation=()` explicitly
disabled microphone access. The deployment guide's nginx example repeated the
same block, so either layer could prevent browser permission UI from succeeding.

## Repair

- Request `getUserMedia({ audio: true })` from the same explicit dictation click.
- Stop every temporary stream track immediately after permission is granted.
- Start speech recognition only after the permission request succeeds.
- Show a requesting state while the browser prompt is pending.
- If permission is blocked, direct the collector to browser site permissions.
- Keep standard/prefixed recognition support and hide the control when
  recognition is unavailable.
- Allow microphone access only to Aurearia's own origin with
  `microphone=(self)` in the Go security header and nginx example.
- Keep geolocation, payment, USB, serial, Bluetooth, and cross-origin microphone
  access disabled.

## Verification

- Focused affected command passed 6 files and 82 tests.
- `task check:web` passed lint, strict type-check, full tests, and production
  build.
- Focused middleware security-header tests and `task check:go` passed.
- `task check:delivery` passed 91 tests and governance validation with zero
  errors and zero warnings.
- Tamper proof: replacing the browser permission requester with `null` failed
  the permission/track-release test; restoring the requester passed it.
- Policy tamper proof: restoring `microphone=()` failed the exact middleware
  header test; restoring `microphone=(self)` passed it.
- Hosted Feature 362 Compatibility run
  [36890888087](https://github.com/briandenicola/Aurearia/actions/runs/36890888087)
  failed only because the deterministic voice fixture did not grant the newly
  required permission preflight before emitting transcript text.
- The fixture now returns a fake microphone stream, proves its track is stopped,
  and waits for the recognition control to enter its started state.
- The focused Playwright voice workflow passed 1/1 and the exact hosted selector
  passed 13/13 locally.
- Per owner direction, the voice workflow remains as focused regression coverage
  but is no longer included in the unrelated Feature 362 Compatibility selector.
- After the selector change, the retained voice workflow passed 1/1, the reduced
  Feature 362 selector passed 12/12, and `task check:web` passed when run
  sequentially. Parallel local Playwright runs were discarded because they
  share one dev server and `test-results` directory.

## Remaining

Real Chrome and Edge behavior over HTTPS must be recorded on the exact repair
candidate. Firefox behavior depends on whether that browser build exposes speech
recognition; typed notes remain the complete fallback when it does not. The
deployed nginx header must be updated with the repository change.
