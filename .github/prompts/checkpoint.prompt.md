Capture an authorized mid-session checkpoint under constitution section 18.5.
ADR 0019 consumer edits remain Proposed until section 22 acceptance.

1. Re-ground in the selected issue/spec/process plan and acceptance criteria.
   Stop on unauthorized scope drift.
2. Record changed paths, implemented versus verified work, exact commit/tree,
   checks actually run, unavailable evidence, and unresolved reviewer restrictions.
3. Persist approved decisions in `.squad/decisions/inbox/` and a concise
   checkpoint in `.squad/log/`. Preserve historical records.
4. Point `.squad/identity/now.md` to the authoritative work state when authorized;
   do not duplicate a full task list. Record the exact next action.
5. After amendment acceptance, the implementation owner can record directly or
   delegate to Scribe with a bounded assignment. Wait for required persistence.

Do not commit, stash, push, clear a block, or introduce another flat state file.
Do not check off unverified acceptance tasks. Briefly report current work,
remaining evidence/blockers, and the next action.
