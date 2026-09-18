# Feature 362 Verification Evidence

## Release A compatibility prerequisite

- ADR 0017 is accepted at commit
  `11bf09d60df95a81d0f4af9b51f6706caecdc977`.
- Release A must contain the closed Deep job-source compatibility guard before
  any Feature 362 schema, source value, callback, or row-producing work.
- `CoinCopilotAttributionEnabled` exists and defaults to `false`.
- Minimum compatibility-guard commit:
  `0910bc7b86e6d5a567c62f8b25000c64fca493a2`.

## Local guard evidence

Date: 2026-09-18

- Red test-first run failed because the unknown-source error/validator and
  default-off attribution setting did not exist.
- Focused guard suite:
  `go test ./handlers ./repository ./services -run "UnknownSource|UnknownSources|CoinCopilotSettingsDefaultsAndIndependentFallbacks|AdminOCRE|GetLatestProviderStatus"`
  passed.
- The immutable Windows guard binary built from
  `0910bc7b86e6d5a567c62f8b25000c64fca493a2` passed the guard-only rollback
  harness with SHA-256
  `c0de8ac10ba956373057d570b16acfb46f24aa0440c15719daeda93c003e4338`.
- Tamper test temporarily recognized `copilot_draft` in the compatibility
  binary. The handler, repository, creation, and pipeline guards failed as
  required. The tamper was reverted.
- No `copilot_draft` source constant, `source_draft_id` column, handoff table,
  callback, migration, or Feature 362 row was added.

## Hosted artifact evidence

- Workflow run:
  [Feature 362 Compatibility 35393314716](https://github.com/briandenicola/Aurearia/actions/runs/35393314716)
  passed, including immutable build, guard-only harness, and artifact upload.
- Guard artifact: `feature362-guard-35393314716-1` (artifact id
  `10567285694`, 30,009,704 bytes).
- Guard artifact digest:
  `sha256:9e7a9993eb5b8a5440006fa105b1b17810fbe43b09b29febe2c963a248c7141e`.
- Guard artifact commit:
  `0910bc7b86e6d5a567c62f8b25000c64fca493a2`.

## External/manual Release A checkpoint

Status: **BLOCKED - NOT DEPLOYED**

No deployment approval has been requested or granted. CI success, artifact
publication, or image publication does not satisfy T009. Feature 362 schema
and row-producing work remain blocked until the verified Release A artifact is
deployed to the intended environment, all guarded paths are manually verified,
the evidence is recorded here, and separate user approval is obtained.
