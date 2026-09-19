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

Status: **COMPLETE - SCHEMA WORK APPROVED**

Deployment date: 2026-09-18

- The owner explicitly approved deploying Release A to the disposable beta
  environment, then corrected the deployment target from Azure to the existing
  local server. No Azure deployment or resource change occurred.
- Environment: `https://coins-beta.denicolafamily.com`.
- Deployed app/agent commit:
  `1fd371a0adc42fdffd4e35136f37519f688fd9a0`.
- App OCI index digest:
  `sha256:6e280855644e55a79f2b574c890b69865e504d8e6d5686d7f7c43393eba253b0`.
- Running app platform digest:
  `sha256:f7d62405e6caedf5569521e7e5db95717d8be12431e6f349dccf10a9a54784eb`.
- `/health` returned HTTP 200 with `{"status":"ok"}` after deployment and
  again after the worker-adoption restart.
- A transactionally consistent pre-fixture SQLite backup was retained at
  `/home/brian/f014-release-a-20260918T210809Z.db`, size 1,712,128 bytes,
  SHA-256
  `8da4f9dd373888e7889698628de1e959a88f36f969846c53311cb5b78be3dbdf`.
- `DeepIdentificationEnabled` was `true`.
  `CoinCopilotAttributionEnabled` had zero database overrides and therefore
  resolved to its built-in default `false`.
- Raw fixture: owner-scoped job `13`, source `copilot_draft`, artifact `26`.
  Its canonical row/artifact/file preservation digest was
  `013b2fde46471e3b88eeec565c8bf2966db98ff6d94c2c9730f6f92924127964`.
- Guarded HTTP results:
  - list: HTTP 200, fixture absent;
  - get/status, stream, retry, cancel, proposal edit, and apply: HTTP 404;
  - no response disclosed source, report, proposal, notes, or artifact data.
- Worker-adoption verification restarted only the app container with Deep
  Analysis enabled. After startup and an eight-second adoption window, the
  fixture remained queued with attempt count 0, last sequence 0, zero events,
  zero provider runs, no matching logs, and the exact same preservation
  digest.
- Existing supported-source job `12` (`intake`, `completed`) remained readable
  through the deployed API with HTTP 200.
- Cleanup deleted exactly synthetic job `13`, artifact `26`, and its fixture
  file. All three were confirmed absent afterward. The backup remains retained.

No Feature 362 schema, `copilot_draft` model constant, callback, handoff row, or
source-draft binding had been added at the checkpoint. After reviewing the
completed deployment evidence, the owner explicitly directed: "Take F014 all
the way to completion." This separately authorizes the additive Feature 362
schema and row-producing work after the mandatory Phase 1-2 red tests.

## Local full mixed-binary rollback matrix

Date: 2026-09-18

- Command: `scripts/compat/feature362-rollback.ps1` with an immutable guard
  binary built from `0910bc7b86e6d5a567c62f8b25000c64fca493a2` and the
  current Feature 362 worktree binary.
- Guard SHA-256:
  `977b17b8a339631e8def8793a457211221cc9bb9bb73644173c6be9f4499abb4`.
- Feature SHA-256:
  `7bd01327f62404a08bb55c7657d68b8764926dd4f164edac7df363cdce4a4344`.
- Guard source-vocabulary tests passed in handlers, repository, and services.
- The feature binary, guard binary, and re-upgraded feature binary each
  returned HTTP 200 from `/health` against the same copied SQLite database.
- The copied database retained its supported `intake` row, `copilot_draft`
  row, source-draft binding, handoff row, default-off attribution setting,
  report payload, and artifact bytes across rollback and re-upgrade.
- The guard produced zero events and zero provider runs for the unknown
  `copilot_draft` row; its status remained queued with attempt count and
  sequence both zero.
- Final result: `feature362_compatibility=full-pass`.
- `.github/workflows/feature362-compatibility.yml` now runs this full matrix
  after the immutable guard-only prerequisite on pushes and pull requests to
  `beta` and `main`.
