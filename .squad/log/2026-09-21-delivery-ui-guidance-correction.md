# P4 UI Guidance Contract Correction

Date: 2026-09-21. Implementation owner: Copilot CLI.
Authority: Owner-approved P4 follow-up; Constitution Principles IV/IX,
sections 17-18/21; ADR 0019. Branch: `fix/delivery-ui-guidance-contract`.
Base: owner-merged #739, `e08c4fd52181476654de02bbb3b33d6872cfef52`.

The original P4 move preserved the UI recipes but missed a test consumer.
[Beta Vue Web](https://github.com/briandenicola/Aurearia/actions/runs/35630621035/job/106435660360)
failed at `ui-patterns.test.ts:17` and `:84`: both read the old universal file.
Windows/Linux delivery and other PR checks passed. The merge and original
independent PASS did not prove the missing frontend contract.

The only executable correction changes the shared path constant and its two
read sites to `.github/instructions/web.instructions.md`. Every assertion is
unchanged. No production behavior, instruction body, task/package script,
dependency manifest/lock, installed tool version or historical block changed.
Active state records the true merge/failure; the original handoff is untouched.

Test-only source tree `27ab6d2462d2d286ce4f6bdf2bf6052958481d83`:
12 targeted tests passed. Deliberately restoring the old path produced exactly
2 failures/10 passes, exit 1; the canonical path was then restored.
Full `task check:web` passed: zero-warning lint, strict type checking,
8 Node asset tests, 1671 Vitest tests passed/1 skipped, and production/PWA build.
Full `task check:delivery` passed: 46 Node tests, PowerShell negative regression,
zero governance errors. The context-budget exception and sparse-link warnings remain.

The isolated worktree initially lacked dependencies. The owner explicitly
approved locked `task setup:web` restoration; npm reported zero audit
vulnerabilities and no manifest/lock changes. Local Node 24.14.0 produced engine
warnings; these successful local runs are not claimed as supported-version CI
parity. Hosted follow-up evidence remains required.

Independent reviewer `4f05f96c-2545-45b0-978c-0c046934c60e` re-reviewed candidate
tree `3f7913cbbc9be5cb9f0f99d153a235628b227681` and returned PASS, no findings.
It used 3/4 approved reads within the five-minute lease, inspecting the complete
diff, evidence and test source with only view/parallel capabilities. Git/test
identities were caller-supplied. It allowed this factual handoff/status update
without changing reviewed behavior and required the delivery gate rerun.

Next: publish the follow-up PR into beta, attach exact-head hosted results and
obtain owner acceptance. These are pending at this record's creation. No merge,
release, deployment, P5/P6 or application-restriction clearance is authorized.
The implementation owner persisted this handoff; no delegated write is assumed.
