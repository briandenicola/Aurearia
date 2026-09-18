# Feature 362 Verification Evidence

## Release A compatibility prerequisite

- ADR 0017 is accepted at commit
  `11bf09d60df95a81d0f4af9b51f6706caecdc977`.
- Release A must contain the closed Deep job-source compatibility guard before
  any Feature 362 schema, source value, callback, or row-producing work.
- `CoinCopilotAttributionEnabled` exists and defaults to `false`.
- Minimum compatibility-guard commit:
  `4119693ba1a3a2652f1e5253df2fe19154494a3d`.

## Local guard evidence

Date: 2026-09-18

- Red test-first run failed because the unknown-source error/validator and
  default-off attribution setting did not exist.
- Focused guard suite:
  `go test ./handlers ./repository ./services -run "UnknownSource|UnknownSources|CoinCopilotSettingsDefaultsAndIndependentFallbacks|AdminOCRE|GetLatestProviderStatus"`
  passed.
- The immutable Windows guard binary built from
  `4119693ba1a3a2652f1e5253df2fe19154494a3d` passed the guard-only rollback
  harness with SHA-256
  `1e19fe4be0521c3980f4ce35086512b697a89e6e36c9eb282a18fd9f5f636c66`.
- Tamper test temporarily recognized `copilot_draft` in the compatibility
  binary. The handler, repository, creation, and pipeline guards failed as
  required. The tamper was reverted.
- No `copilot_draft` source constant, `source_draft_id` column, handoff table,
  callback, migration, or Feature 362 row was added.

## Hosted artifact evidence

- Workflow run: pending.
- Guard artifact digest: pending.
- Guard artifact commit: pending.

## External/manual Release A checkpoint

Status: **BLOCKED - NOT DEPLOYED**

No deployment approval has been requested or granted. CI success, artifact
publication, or image publication does not satisfy T009. Feature 362 schema
and row-producing work remain blocked until the verified Release A artifact is
deployed to the intended environment, all guarded paths are manually verified,
the evidence is recorded here, and separate user approval is obtained.
