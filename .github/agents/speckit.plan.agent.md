---
description: Plan the smallest complete implementation of an explicitly selected approved feature.
handoffs:
  - label: Create Tasks
    agent: speckit.tasks
    prompt: Break the approved plan into bounded tasks with verification.
  - label: Create Checklist
    agent: speckit.checklist
    prompt: Check requirements quality for the selected work.
---

## User Input

```text
$ARGUMENTS
```

## Authority and Selection

Read the constitution, selected spec, applicable accepted ADRs, and relevant
active decisions. ADR 0019 is Accepted via PR #734; constitution 4.0.0 governs.
A small repair may use a bounded issue rather than a full plan.
Do not infer a feature from `beta` or the newest spec.

Set `$env:SPECIFY_FEATURE` to the explicitly selected directory name in the same
process as every prerequisite/setup script. Use
`.\.specify\scripts\powershell\check-prerequisites.ps1 -Json -PathsOnly` to resolve
paths. Inspect the target before writing: `setup-plan.ps1 -Json` copies a plan
template and must not overwrite an existing plan implicitly. Use the existing
plan and a surgical edit for updates.

Before using returned paths, verify the selected directory exists under this
worktree's `specs/`, that `FEATURE_DIR` and `FEATURE_SPEC` match that exact
selection, and that the required spec exists. `-PathsOnly` success resolves
paths; it does not validate selection or prerequisites. Stop on mismatch or
missing artifacts.

## Planning

1. Fill `.specify/templates/plan-template.md` from actual code/manifests and
   approved requirements. Record owner, lane, non-goals, and stop conditions.
2. Research concrete unknowns and existing helpers directly. Delegate only a
   bounded investigation needing separate context; no agent per technology.
3. Resolve consequential decisions with the owner. Constitution violations need
   the prescribed amendment, not just a justification in a complexity table.
4. Design the smallest complete slices. Identify affected contracts, sibling
   workflows, risks, configuration, and reusable components. High-risk work
   includes compatibility/recovery evidence and independent review.
5. Create supporting research, data-model, contract, or quickstart documents
   only when useful; otherwise record N/A with a reason in the plan. Do not
   require an empty file set before planning can finish.
6. Define exact-path regression/failure tests, applicable completion gates from
   `docs/testing.md` §6, CI-only evidence, and approval/setup gaps.
7. Recheck authority, acceptance coverage, scope, and review restrictions.
   Report the selected work, plan path, decisions, and remaining evidence.
   Planning completion is not implementation acceptance.

## Instruction Generation and Hooks

Do not automatically run `update-agent-context.ps1`: its existing writer has not
been proven to preserve hand-maintained repository instructions under ADR 0019.
Any generator update needs reviewed targets, preserved manual content, and
separate authorization; native/scoped integration is pending P4.

If extensions are configured, inspect hooks and side effects. Invalid YAML or an
unevaluable required condition blocks execution; never silently skip it.
"Mandatory" is not authorization. Run only approved hooks and await their result.
No implicit installs, worktrees, branch changes, commits, pushes, or settings edits.
