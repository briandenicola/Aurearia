Cold-start ritual for any AI agent (Copilot CLI, Coding Agent, Squad member).
Per Constitution §18 (Session Protocol — "Always"), read these before editing
code. Do them in parallel where possible.

Read, in order:

1. `.specify/memory/constitution.md` — full constitution, especially §0
   Hierarchy of Authority and §18 AI Agent Operating Rules.
2. `.squad/decisions.md` — landed project decisions.
3. `.squad/decisions/inbox/` — pending decisions not yet merged.
4. `.squad/log/` — most recent session log entry (single file, latest
   timestamp).
5. **Active spec** — `specs/NNN-*/spec.md` + `plan.md` + `tasks.md` for the
   feature implied by the current branch name, or named by the user. If
   ambiguous, ask.
6. **Agent charter** — `.squad/agents/<you>/charter.md` if running as a Squad
   member; plus `.squad/agents/<you>/history.md` for prior learnings.
7. **Relevant skills** — any `.squad/skills/*.md` referenced in the charter
   or active spec.
8. `.github/copilot-instructions.md` — repo-level conventions.

Then output a short status block:

- **Loaded**: bullet list of artifacts read (with version / last-updated where
  available).
- **Active spec**: `specs/NNN-slug/` and current task ID from `tasks.md`.
- **Changes since last log**: new decisions, completed tasks, open blockers.
- **Next action**: one sentence — what you propose to do first.

If a Squad coordinator (Maximus) is available, delegate the read fan-out to
it. Wait for explicit user confirmation before editing code.

References: Constitution §0 (Hierarchy of Authority), §18 (AI Agent Operating
Rules — "Always" and "Context Discipline").
