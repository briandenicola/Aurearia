End-of-session protocol. Per Constitution §18.5 (Session Handoff), Scribe owns
this ceremony. The canonical surfaces are `.squad/log/` and
`.squad/decisions.md` — **never** `SESSION-NOTES.md` or `.copilot-state.md`
(both forbidden by §18).

Delegate to **Scribe**. Steps:

1. **Reconcile tasks.md** — check off every task completed this session;
   confirm in-flight tasks are still marked accordingly.
2. **Commit or stash WIP** — no uncommitted edits may straddle a session
   boundary. Either:
   - Commit using Conventional Commits + the `Co-authored-by: Copilot
     <223556219+Copilot@users.noreply.github.com>` trailer (per §17 / Principle
     VIII), citing the relevant Principle / section in the message; or
   - `git stash push -m "handoff: <branch> <reason>"` if work is not in a
     committable state.
3. **Merge decisions inbox** — Scribe merges every file in
   `.squad/decisions/inbox/` into `.squad/decisions.md` (append, never
   rewrite), then deletes the inbox files. Conflicts escalate to Maximus.
4. **Write session log** — `.squad/log/{YYYY-MM-DD-HHMM}-handoff.md` covering:
   tasks completed, decisions landed, open blockers, exact next action for
   the next session (file + line + intent).
5. **Append per-agent history** — each agent that took meaningful action this
   session appends a dated bullet to `.squad/agents/<name>/history.md` under
   `## Learnings`.
6. **Quality Gate sanity** — confirm the §17 checklist would pass for any
   landed commits (build, lint, tests, no secrets); flag failures in the log.

Output to chat: one-line confirmation per step (✓ / ✗ with reason). No new
work after handoff.

References: Constitution §17 (Quality Gate), §18 (AI Agent Operating Rules —
Session Handoff), Principle VIII (Commit Convention). Squad agents: Scribe
(owner), Maximus (escalation).
