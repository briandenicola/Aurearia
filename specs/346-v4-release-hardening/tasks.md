# Tasks: v4 Release Hardening

## Phase 1 - Release identity and governance

- [x] T001 Set the release identity to v4 consistently.
- [x] T002 Cut a v4 changelog section without creating a release tag.
- [x] T003 Mark implemented Features 343-345 as implemented.
- [x] T004 Correct stale constitution and toolchain references.

## Phase 2 - Canonical documentation

- [x] T005 Align the PRD with Deep Analysis, Nomisma, OCRE, and paused RPC.
- [x] T006 Align architecture and SDD service/provider flows.
- [x] T007 Align feature index and detailed feature documentation.
- [x] T008 Align deployment, security, and threat-model guidance.

## Phase 3 - Verified quality debt

- [x] T009 Remove high-confidence dead imports reported by lint.
- [x] T010 Add provider-tool boundary contract coverage.
- [x] T011 Refactor verified oversized UI responsibilities without behavior change.
- [x] T012 Refactor verified oversized composition/backend responsibilities where
  existing seams permit a proportional extraction.

## Phase 4 - Release gate

- [x] T013 Run full Go, Python, and Vue quality gates.
- [x] T014 Run security and generated-contract checks.
- [x] T015 Complete independent post-major-work QC.
- [x] T016 Commit and push coherent batches, open a PR to beta, and merge only
  after all required checks pass.
