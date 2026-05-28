Mid-session pause. Capture state without ending the session. Per Constitution
§18.5 (Session Handoff), the canonical surfaces are `.squad/log/` and
`.squad/decisions/inbox/` — **never** `SESSION-NOTES.md` or
`.copilot-state.md` (forbidden by §18).

Delegate to **Scribe** if available; otherwise do it yourself.

Steps:

1. Re-ground: quote verbatim from `specs/NNN-*/tasks.md` the current task and
   its acceptance criteria. If drift detected from the active spec, STOP and
   report — do not silently course-correct.
2. Summarize **work-in-progress** since the last log entry: files touched,
   tasks moved to done, tasks newly in flight.
3. For each material decision made since the last checkpoint, write a
   one-file-per-decision entry to `.squad/decisions/inbox/<agent>-<slug>.md`
   (do **not** dump decisions into the chat transcript — they will be lost).
4. List **open questions** and **next steps** in priority order.
5. Append the full checkpoint to `.squad/log/{YYYY-MM-DD-HHMM}-checkpoint.md`
   following the template in `.squad/templates/`.
6. Do NOT commit or stash — session continues. (Use `/handoff` for end of
   session.)

Output to chat: a 5-line summary (current task · WIP files · decisions
written · open questions · next step) so the human can sanity-check before
work continues.

References: Constitution §18 (AI Agent Operating Rules — Session Handoff),
§0 (Hierarchy of Authority). Squad agents: Scribe (owner), Maximus (reviewer
on architectural drift).
