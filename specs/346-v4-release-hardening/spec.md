# Feature Specification: v4 Release Hardening

**Feature Branch**: `346-v4-release-hardening`
**Created**: 2026-08-15
**Status**: In Progress
**Input**: User description: "Perform a deep software and documentation analysis, fix verified release-readiness issues, and leave beta ready for the final v4 merge to main."

## User Scenarios & Testing

### User Story 1 - Publish an internally consistent v4 release candidate (Priority: P1)

As the application owner, I can review beta and find one consistent v4 identity
across the version file, README, changelog, product documentation, architecture
documentation, feature index, and implemented feature specifications.

**Independent Test**: Search all canonical release surfaces and confirm they
describe v4, Deep Analysis, Nomisma authority linking, automated OCRE evidence,
and the paused RPC integration without contradictory status or provider claims.

**Acceptance Scenarios**:

1. **Given** Features 343-345 are merged into beta, **When** canonical documents
   are reviewed, **Then** they identify those features as implemented and describe
   their production boundaries accurately.
2. **Given** RPC automation is paused, **When** provider documentation is reviewed,
   **Then** RPC is described as unavailable for automated analysis rather than
   shipped or silently omitted.
3. **Given** v4 is not yet merged to main, **When** release metadata is reviewed,
   **Then** beta is identified as a v4 release candidate and no release tag is
   created by this feature.

---

### User Story 2 - Resolve verified maintainability debt safely (Priority: P2)

As a maintainer, I can work in the highest-risk oversized modules without one
component or function owning unrelated responsibilities, while preserving all
existing behavior.

**Independent Test**: Compare public interfaces and run the existing focused and
full quality gates after behavior-preserving extractions.

**Acceptance Scenarios**:

1. **Given** an oversized module has independently testable responsibilities,
   **When** it is refactored, **Then** responsibilities move behind existing
   interfaces without changing routes, API contracts, or user behavior.
2. **Given** a large file is cohesive or generated, **When** the audit is applied,
   **Then** it is documented rather than split solely to reduce line count.

## Requirements

### Functional Requirements

- **FR-001**: Canonical release metadata MUST consistently identify version 4.0.
- **FR-002**: Canonical documentation MUST describe Deep Analysis as an optional,
  persisted, replayable background workflow while preserving Quick Identify.
- **FR-003**: Documentation MUST distinguish Nomisma authority linking from OCRE
  coin-type evidence and MUST include their separate license attributions.
- **FR-004**: Documentation MUST state that automated RPC integration is paused.
- **FR-005**: Implemented feature specifications 343-345 MUST no longer be marked
  Draft.
- **FR-006**: Any structural refactor MUST preserve behavior and existing public
  contracts and MUST be protected by existing or targeted tests.
- **FR-007**: The branch MUST pass the repository Quality Gate before merge.
- **FR-008**: This feature MUST merge only to beta; merging beta to main remains
  an explicit owner action.

## Success Criteria

- **SC-001**: No canonical release document advertises v1, v2, or v3 as current.
- **SC-002**: Deep Analysis, Nomisma, OCRE, and deferred RPC are discoverable from
  the README and feature index.
- **SC-003**: Full Go, Python, and Vue quality gates pass.
- **SC-004**: Independent post-change review reports no v4 release blocker.

## Assumptions

- Version 4.0 is the intended next release identity.
- The OCRE and Deep Analysis implementations currently on beta are the release
  baseline; this feature does not expand provider behavior.
- Generated API artifacts are updated only through the repository's existing
  generation command when their source metadata changes.

