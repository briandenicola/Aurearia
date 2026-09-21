---
description: Create or update a bounded, owner-authorized feature specification.
handoffs:
  - label: Build Technical Plan
    agent: speckit.plan
    prompt: Create a plan for the explicitly selected approved specification.
  - label: Clarify Spec Requirements
    agent: speckit.clarify
    prompt: Clarify the explicitly selected specification.
---

## User Input

```text
$ARGUMENTS
```

## Authority and Selection

Read the constitution and applicable accepted ADRs. ADR 0019 is Accepted via
PR #734; constitution 4.0.0 governs. Do not change a landed spec without
amendment authority. Clarify material scope, behavior, and approval ambiguity;
never invent product decisions or treat a proposal as approved.

Use the feature/high-risk lane only when a feature spec is warranted. A small
repair may use a bounded issue; do not create a new document set automatically.
Ordinary development stays on `beta`. Select an existing spec explicitly from
the request or validated current-work pointer, never the latest directory.

## Workflow

1. Confirm outcome, non-goals, acceptance criteria, authorization, and whether
   this is a new spec or an approved update. Inspect existing artifacts first.
2. For a new feature, choose a concise work ID using the existing directory
   inventory. The checked-in script supports `-DryRun`, but normal invocation
   creates/checks out a branch. On `beta`, use only the non-mutating preview:

   ```powershell
   .\.specify\scripts\powershell\create-new-feature.ps1 -DryRun -Json -ShortName "<slug>" "<description>"
   ```

   Inspect its `BRANCH_NAME` and `SPEC_FILE`, confirm the destination does not
   already exist, and create the approved spec from the template using a file
   edit, staying on `beta`. The preview does not create a directory or set an
   enduring feature selection. Run branch-creating mode only with explicit
   branch authorization. Do not invent a no-branch script flag.
3. Use `.specify/templates/spec-template.md`. Keep meaningful, independently
   testable user journeys, requirements with IDs, edge/failure cases, measurable
   outcomes, non-goals, and explicit unresolved decisions. Remove sample stories
   that do not apply; do not manufacture arbitrary metrics.
4. Review the draft against the request and existing workflows. Check that every
   requirement is testable, scope is bounded, and defaults do not alter auth,
   data retention, deployment, or UX policy without approval.
5. Ask targeted clarification questions for material ambiguity. Do not silently
   fill uncertain scope just to remove clarification markers. Record approved
   answers; do not mark the draft accepted because it is written.
6. Report the spec path, work ID, branch, remaining decisions, and readiness for
   planning. Creating tasks, implementing, committing, or publishing is a
   separate action, not a side effect.

For subsequent commands on `beta`, set `$env:SPECIFY_FEATURE` to the exact chosen
directory name in the same process as each script invocation. Reconfirm selection
each time; shell state may not persist.

## Hooks and Side Effects

If `.specify/extensions.yml` exists, inspect relevant hooks and report their
commands/side effects. Invalid configuration or unevaluable required conditions
block execution; do not silently skip them. A hook marked mandatory is not owner
authorization. Execute only explicitly approved hooks and await their result.
Do not install/enable hooks, tools, or settings through specification generation.
