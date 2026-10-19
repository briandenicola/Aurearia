Load the minimum context for the explicitly selected work under constitution
sections 0 and 18. ADR 0019 is Accepted via PR #734; constitution 4.0.0 governs.
Archival execution still requires authorization; historical blocks remain intact.

Prepared for Proposed ADR 0024 / 5.0.0: until §22 approval/merge, effective
4.0.0 governs from the constitution's linked baseline. After activation select
the canonical issue/PR (or approved work order), with optional specs rather than
a mandatory document set. Load scoped skills on demand; no legacy migration
preflight is required. The current-work pointer links sources only.

1. Read `.specify/memory/constitution.md` and repository instructions.
2. Identify the owner-named issue/spec/process plan. Check
   `.squad/identity/now.md` against that source and relevant review evidence.
   Ask if ambiguous; never choose the newest spec or infer one from `beta`.
3. Read the selected acceptance criteria, non-goals, plan/tasks as applicable,
   accepted ADRs, and relevant active decisions/inbox entries.
4. Read a charter/history only for a role you are acting in. Load applicable
   native skills on demand; legacy `.squad/skills/` entries are compatibility links.
5. Inspect the worktree and relevant prior handoff without changing anything.
   Do not fan out agents just to read context.

Report briefly: selected work/lane, authority, evidence loaded, unresolved blocks,
and the next permitted action. Contradictory state is a blocker, not permission
to restart completed work. Respect existing owner approval; request clarification
only where scope or machine/publication authority is missing.

For SpecKit on `beta`, use the explicit feature-directory name in
`$env:SPECIFY_FEATURE` in the same process as each prerequisite/setup command.
Do not infer or silently persist a selection. Non-feature work does not need a
synthetic spec.
