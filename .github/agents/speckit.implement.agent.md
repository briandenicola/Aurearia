---
description: Execute an authorized task slice with evidence-backed verification and independent acceptance.
---

## User Input

```text
$ARGUMENTS
```

## Before Work

> Prepared for Proposed ADR 0024 / 5.0.0; until §22 approval/merge use effective
> 4.0.0 from the constitution's linked baseline. Existing #827 audit and historical
> blocks remain. Prospective rules below do not authorize this session's checks.

1. Read the constitution, explicit issue/spec/process plan, relevant decisions,
   and applicable accepted ADRs. ADR 0019 is Accepted via PR #734; constitution
   4.0.0 governs until the recorded §22 activation; thereafter apply the accepted
   version. Do not infer the work from `beta` or the newest spec.
2. Only when the owner selected a SpecKit task set, set `$env:SPECIFY_FEATURE` to the exact directory name in
   the same process as:

   ```powershell
   .\.specify\scripts\powershell\check-prerequisites.ps1 -Json -RequireTasks -IncludeTasks
   ```

   Load returned spec/plan/tasks and relevant supporting documents. For bounded
   work without that task set, use its approved issue/PR/work order instead.
   Do not require a new spec/plan/tasks set for a feature whose canonical issue
   already contains sufficient criteria.
3. Inspect the worktree and existing helpers. Preserve unrelated edits.
   Confirm authorized task slice, paths, non-goals, owner, and stop conditions.
4. Read actual checklist items and review records, not just checked-item counts.
   Missing approval or evidence cannot be bypassed by "continue anyway."
   Preserve all existing reviewer blocks/author restrictions; escalate conflicts.
5. Check required tools and permissions. Do not install dependencies, initialize
   frameworks, generate ignore files, switch branches, or rewrite instructions
   just because technology detection suggests doing so.

## Implementation

- Default to one owner. Respect task dependencies; parallel markers do not mandate
  agents. Every permitted delegation has a path boundary, lease, and stop condition.
- Add applicable regression/failure/contract tests and prove behavior guards
  fail when broken. Reuse existing patterns and check directly related siblings.
- Implement the smallest complete approved change; stop before scope expansion.
- Run targeted feedback, then applicable completion checks from `docs/testing.md`
  §6 when authorized. Record exact commands/results and commit/dirty-tree identity.
- Surface errors. Missing tools, permission, or CI-only results remain incomplete.
- Update task checkboxes only for completed task-specific evidence; do not mark
  review/release tasks complete merely because code exists.

## Completion

Verify the actual acceptance criteria, contracts, failure paths, and applicable
§17 gates. Obtain required independent review of the exact tree. An author cannot
clear their own block; reassignment and green CI are not clearance.

Report implemented versus verified/accepted/released status, changed paths,
actual evidence, exceptions, and unresolved restrictions in canonical work.
Use one overall authorized budget and meaningful checkpoints, not per-read or
per-repair permission. Explicit work orders may authorize relevant tests/review/
repairs and beta commit/push/PR when green; execute only named actions.
Conditional merge needs an exact eligible candidate and no unmet gate.
Setup/live operations and deployment need applicable separate authority.

## Hooks

If extensions exist, inspect relevant hooks and side effects. Invalid YAML or
unevaluable required conditions block execution; never silently skip them.
A "mandatory" flag does not authorize execution. Run only approved hooks and
await their results before claiming completion.
