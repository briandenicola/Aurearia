End the session under constitution sections 17, 18.5, and 21.
ADR 0019 is Accepted via PR #734; constitution 4.0.0 governs.

1. Reconcile the selected issue/spec/process plan: distinguish implemented,
   verified, accepted, and released work. Checked tasks do not clear review blocks.
   Use the canonical completion record in the PR (or approved issue/plan before
   a PR exists), following `docs/agentic-acceptance-controls.md`. Link criteria,
   affected/sibling workflows, exact tested/reviewed identity, raw evidence,
   exceptions and actual owner decisions. Missing evidence stays pending.
2. Preserve the worktree and unrelated edits. Report changed paths; do not
   automatically commit or stash. A separately authorized commit stages only
   approved paths and uses Principle VII/section 17 conventions and trailers.
3. Persist authorized decisions and a concise `.squad/log/` handoff with
   evidence identity, actual check results, unavailable checks, owner approval
   boundaries, unresolved blocks, and the exact next action.
   Link the canonical completion record and add only the checkpoint delta;
   do not copy raw logs or create a parallel evidence/status ledger.
4. Update the authorized current-work pointer without duplicating task state.
   Preserve original evidence before any separately authorized curation.
5. After amendment acceptance, the implementation owner may do this directly;
   Scribe is optional help. Wait for required recording to finish.

Never declare the gate "would pass." Unexecuted checks remain incomplete.
An old PASS cannot certify changed implementation. Record the diff and obtain
applicable verification/re-review; do not infer equivalence from a branch name.
Open blocks or missing owner approval prevent accepted/release-ready claims.
Do not introduce `SESSION-NOTES.md` or `.copilot-state.md`.
Report the handoff and remaining blocker briefly; start no new work afterward.
