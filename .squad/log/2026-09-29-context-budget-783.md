# Issue #783 context budget handoff

Owner choices (2026-09-29): keep the 6000-word startup budget, archive the
unresolved-review table with a pointer, commit to `beta` after checks and review.

- `.squad/decisions.md` curated from 2019 to 692 words; binding clauses and the
  block-clearance rule restored after review round 1 (BLOCK B1-B3). The prior file is
  appended byte-for-byte to `decisions-archive.md#curation-2026-09-29-decisions`
  (SHA-256 `71e9350a93cf8f10864932f7208698c71f33a0d86d7abd093ebf9c90ed1c2d68`,
  equal to `git show 73d5750c:.squad/decisions.md`).
- `now.md` refreshed to the group F focus; the old pointer is appended at
  `decisions-archive.md#curation-2026-09-29-now`.
- Inventory with byte offsets, line numbers and per-heading SHA-256:
  `.squad/artifacts/context-curation-2026-09-29.json`.
- The budget warning now lists per-file word counts, largest first; new test in
  `governance.test.mjs`, tamper-checked (fails with the old message).
- Evidence: `task check:delivery` exit 0; governance 0 errors, 0 warnings.
- Next: #780 and #781 wait on owner approval of the Go 1.26.6 toolchain and
  `task setup:agent`.
- Review: aurearia-reviewer round 1 BLOCK (B1 clearance rule, B2 dropped
  clauses, B3 inventory); repaired by the author. Round 2 (fresh instance of the
  same read-only role; the first ran sync and could not take follow-ups) PASS on
  the uncommitted tree. Optional follow-ups not applied: GOV-006 record-placement
  wording, "inspect the indexed review chain" sentence, SEC-001 smoke-guidance mention.
