# Implementation Plan: [FEATURE]

**Work ID / spec directory**: `[###-feature-name]`
**Working branch**: `beta` unless explicitly approved otherwise
**Date**: [DATE] | **Spec**: [approved path]
**Lane**: [feature / high risk]
**Owner**: [implementation owner] | **Reviewer**: [independent reviewer]

Subordinate to the constitution. ADR 0019 consumer changes remain Proposed until
section 22 acceptance. Small repairs can use a bounded issue rather than this
full template. Use `.github/agents/speckit.plan.agent.md` for planning guidance.

## Summary and Scope

[Approved outcome, acceptance criterion IDs, non-goals, and stop conditions.]

## Technical Context

[Read actual manifests and existing workflows. Record only relevant languages,
dependencies, storage, target environment, constraints, and unresolved questions.
Do not infer new platforms, cloud deployment, dependencies, or configuration.]

## Constitution Check

*Before research and after design:*

- [Applicable Principles I-IX and sections 0, 17-22]
- [Authorizing artifact, applicable accepted ADRs, and any required amendment]
- [Exact affected user workflows, contracts, configuration, and sibling paths]
- [Validation tiers and authorized machine/GitHub operations]
- [Review restrictions, independent reviewer, and owner approval boundaries]

Unapproved violations block work. Recording a justification does not waive policy.

### Complexity Tracking

| Required exception | Why necessary / simpler option rejected | Amendment and approval evidence |
|--------------------|------------------------------------------|---------------------------------|
| [Rule or N/A] | [Reason] | [Accepted authority or blocked pending approval] |

## Existing Structure and Reuse

[Verified paths to code, tests, and existing helpers; smallest complete change.
Do not generate new project scaffolding or copy an unrelated example tree.]

## Implementation Slices

[Bounded, usable slices with dependencies and criterion-level evidence.
Default to one implementation owner. Any delegation names paths, lease,
checkpoint, required result, and stop condition.]

## Verification and Recovery

[Exact-path regressions, failure cases, producer/consumer and sibling workflows,
applicable `docs/testing.md` §6 commands, CI-only evidence, and manual exceptions.
High-risk changes include compatibility, representative-data upgrade and recovery
evidence as applicable. Missing tools/permission mean pending, not passed.]

## Supporting Artifacts

[Create research.md, data-model.md, contracts/, or quickstart.md only when each
answers an actual need; otherwise record N/A with a reason here. Do not generate
a document set just to satisfy a template.]

## Lifecycle Evidence

[Implemented / verified / accepted / released status, exact commit/tree,
reviewer disposition, and required owner approvals. No automatic publication.]
