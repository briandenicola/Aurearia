---
description: Generate bounded, dependency-ordered tasks with required verification and acceptance evidence.
handoffs:
  - label: Analyze For Consistency
    agent: speckit.analyze
    prompt: Analyze the selected spec, plan, and tasks without modifying them.
  - label: Implement Approved Work
    agent: speckit.implement
    prompt: Execute only the authorized task slice and its required verification.
---

## User Input

```text
$ARGUMENTS
```

## Authority and Selection

Read the constitution, applicable accepted ADRs, and explicit approved work.
ADR 0019 is Accepted via PR #734; constitution 4.0.0 governs.
Stay on `beta` unless a branch is expressly authorized. Never guess the spec.
Set `$env:SPECIFY_FEATURE` to its directory name in the same process as:

```powershell
.\.specify\scripts\powershell\check-prerequisites.ps1 -Json
```

Use returned paths; load the selected spec/plan and relevant supporting artifacts.
Missing optional research/data-model/contracts documents are not a reason to
invent them. A small repair may keep its tasks in an approved bounded issue.
Do not overwrite existing task state, review findings, or evidence.

## Generate Tasks

Use `.specify/templates/tasks-template.md` and actual repository paths.

1. Map every approved criterion to implementation and verification tasks.
   Group by independently usable story; include only real prerequisites.
2. Tests are required where applicable. Include exact-path regression, primary
   failure, cross-service contract, and sibling workflow coverage. Do not make
   tests conditional on the user requesting TDD. Explain any manual exception.
3. List dependencies and default to one implementation owner. `[P]` permits
   independent non-conflicting work, not automatic parallel agents.
4. Finish with applicable §17 gates, directly affected docs/contracts,
   independent review, lifecycle reconciliation, and durable handoff.
5. Record high-risk compatibility/recovery evidence and owner approval gates.
   No implicit project scaffolding, dependency setup, hardening, refactoring,
   release, or scope expansion.

Every task uses:

```text
- [ ] T001 [P] [US1] Concrete action in verified\path with observable evidence
```

Use sequential IDs; `[P]` and `[US1]` only when applicable. Administrative or
cross-cutting tasks need no story label. A referenced path may be a verified
planned new file, but do not invent an existing helper or package.

Keep implemented, verified, accepted, and released states separate in an evidence
table tied to the commit/tree. A checked task is not reviewer clearance.
Report the path, bounded task count, criterion coverage, dependencies, and
remaining decisions. Do not auto-start implementation from a generated list.

## Hooks

Inspect configured hooks and side effects. Invalid configuration or unevaluable
required conditions block execution; do not silently skip them. Run only approved
hooks and await results. No implicit tool installation or instruction regeneration.
