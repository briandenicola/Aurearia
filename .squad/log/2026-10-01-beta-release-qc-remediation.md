# Beta release QC remediation (2026-10-01)

- Release base: remote `main` at `e053e6c6`.
- Audited beta candidate: `f8a788b3`.
- Independent beta-wide audit verdict: **BLOCK**.
- Owner authorization: repair the blocker, run local validation, and appoint
  the independent read-only reviewer as successor for R-RELEASE disposition.

## Repair

The audit found that dictation could start after the form became disabled if
the browser microphone permission prompt was still pending. The disabled watcher
now aborts every non-idle dictation session, which advances the generation
counter and prevents the completed permission request from starting recognition.

- Implementation and regression test: `a09f5db1`.
- The regression holds permission open, disables dictation, grants permission,
  and proves recognition never starts.
- Tamper proof: restoring the old `disabled && recognition` condition failed
  exactly the new regression test; restoring the fix returned 18/18 focused
  tests to green.

## Exact-candidate evidence

The validation tree was identical to committed candidate `a09f5db1`.

- `task check:go`: passed build, vet, gofmt, and all Go package tests.
- `task check:web`: passed zero-warning lint, strict `vue-tsc --build`,
  210 test files plus one skipped file, 1,799 tests plus one skipped test, and
  production/PWA build.
- `task check:delivery`: passed 91/91 delivery tests, SpecKit negative controls,
  and governance with zero errors or warnings after correcting the current-work
  pointer.
- `npm run test:browser -- e2e/workflows/voice-identification.spec.ts`: 1/1
  passed.
- `npm run test:browser -- e2e/workflows/desktop-context-header.spec.ts`: 3/3
  passed.
- `git diff --check`: passed.

## Owner browser evidence

On `a09f5db1`, the owner verified Chrome and Edge over HTTPS:

- permission grant;
- permission denial with typed fallback;
- natural silence end;
- explicit Stop;
- navigation cleanup;
- editable transcript;
- Quick Identify notes submission; and
- Deep Analysis notes submission.

Firefox remains browser-neutral: when the recognition API is unavailable, the
dictation control is absent and typed notes remain available. The unsupported
browser regression test covers that fallback.

## Independent re-review and owner disposition

The owner-appointed independent successor reviewed receipt candidate `105240c3`
and explicitly cleared B2, B3, and B4. The reviewer found the Features 364/365
code technically sound and release-quality. The only remaining release block was
historical R-RELEASE because Feature 363 T044's separately tracked combined
Feature 362/363 audit was never recorded.

The repository owner granted a scoped exception for candidate `105240c3` and
its beta-to-main PR. The exception is recorded in
`.squad/decisions/inbox/release-exception-2026-10-01.md`. It does not mark T044
complete, claim the missing audit occurred, or authorize merge, publishing,
deployment, or release.
