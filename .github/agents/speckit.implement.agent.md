---
description: Execute an authorized task slice with evidence-backed verification and independent acceptance.
---

## User Input

```text
$ARGUMENTS
```

## Before Work

1. Read the constitution, explicit issue/spec/process plan, relevant decisions,
   and applicable accepted ADRs. ADR 0019 is Accepted via PR #734; constitution
   4.0.0 governs. Do not infer the work from `beta` or the newest spec.
2. For feature tasks, set `$env:SPECIFY_FEATURE` to the exact directory name in
   the same process as:

   ```powershell
   .\.specify\scripts\powershell\check-prerequisites.ps1 -Json -RequireTasks -IncludeTasks
   ```

   Load returned spec/plan/tasks and relevant supporting documents. For bounded
   non-feature work, use its approved issue/process plan instead.
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
actual evidence, exceptions, and unresolved restrictions. Persist an authorized
handoff and await required recording. No implicit commit, push, merge, deploy,
or new task pickup. Release needs owner authorization for the candidate commit.

## Hooks

If extensions exist, inspect relevant hooks and side effects. Invalid YAML or
unevaluable required conditions block execution; never silently skip them.
A "mandatory" flag does not authorize execution. Run only approved hooks and
await their results before claiming completion.
