# P3 Delivery Validation: Verification and Acceptance

Date: 2026-09-21
Implementation owner: Copilot CLI
Authority: Constitution Principles IV, VII, VIII, IX; sections 17-19/21; ADR 0019
Scope: Delivery improvement plan D08-D10; owner-approved filename portability

## Accepted candidate

- Baseline: `af2ac2488cf38cd4ce82a33cdd7de986d3fe8045`.
- Final source commit: `6920914484bb6d81ee69778fa47042d84a52162f`.
- Reviewed and tested tree: `d555f132dc56b130e1ab8f22ec633876b99323b1`.
- Owner-merged [PR #737](https://github.com/briandenicola/Aurearia/pull/737):
  `c36387275ce6b34ed012a29c2a861a2b5dd17f02`, 2026-09-21T15:09:11Z.
- GitHub's merge tree exactly equals the reviewed/tested tree.

D08-D10 are implemented, verified and accepted. The owner separately approved
a documentation-only closeout PR. This record does not certify checks on a
later commit or claim a release/deployment.

## Review and repairs

Independent reviewer `261350ad-c4c6-45ad-936c-549fb509bb15` reviewed P3 and its
bounded repairs. A scoped BLOCK demonstrated that inherited environment values
override Task 3.44.0 `env` defaults. Command-local assignments now enforce the
Go/Python/race settings. The original reviewer independently reproduced the
repair and negative controls, explicitly cleared its BLOCK, and carried that
clearance forward to the final tree above.

Hosted Windows checkout exposed colon-containing historical filenames. Sparse
exclusions did not bypass Git's path validation; local Git's pre-existing
disabled NTFS protection had masked this. The owner approved a filename-only
fix. All 33 renamed logs preserve exact original Git blobs and modes; the
[mapping](../artifacts/windows-log-path-mapping-2026-09-21.json) records old/new
paths. The reviewer verified preservation, no collisions, and no old-basename
references in baseline tracked text. Historical contents/restrictions and local
Git settings were not changed. Both delivery jobs use full protected checkout.

## Evidence

All 20 PR checks passed on the final source candidate:

- [Quality Gate](https://github.com/briandenicola/Aurearia/actions/runs/35616516147):
  full Windows/Ubuntu delivery checks; Go build/vet/tests and OpenAPI consistency;
  separate race detector; full frontend lint/type-check/package tests/build;
  locked Python environment verification, Ruff and pytest.
- [Security Scan](https://github.com/briandenicola/Aurearia/actions/runs/35616516150):
  Gitleaks, Govulncheck, npm/pip audits, container scans and agent runtime pip check.
- [Compatibility](https://github.com/briandenicola/Aurearia/actions/runs/35616516176):
  Release A guard and Coin Copilot browser acceptance.
- [CodeQL](https://github.com/briandenicola/Aurearia/actions/runs/35616511001):
  actions, Go, JavaScript/TypeScript and Python analysis.

Local evidence includes 35 Node tests, the PowerShell SpecKit Boolean/stream
regression, diagnostic-ablation and runtime negative controls, actual Go and
OpenAPI gates using existing tools, and offline governance with zero errors.
A disposable full Windows checkout failed on the old baseline and succeeded on
the final candidate with `core.protectNTFS=true` and no sparse omissions; the
delivery suite then passed inside that checkout. The reviewer independently
checked preservation/configuration but did not repeat this checkout experiment.

## Remaining boundaries

Automatic instruction length and initial-context size remain two explicit
maintenance warnings for P4, not permission to truncate authority. P4-P6 have
not started. Live branch protections are unchanged. Historical application
review restrictions and Feature 363 T044 remain unresolved by this batch.
No tool installation, assistant merge, deployment or application repair occurred.
The separate beta post-merge workflows are not substituted for the exact PR
candidate evidence recorded above.
