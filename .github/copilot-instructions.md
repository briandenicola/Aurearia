# Aurearia: Universal Copilot Instructions

Aurearia is a Go API, Vue/TypeScript PWA and stateless Python agent service.
Read the [constitution](../.specify/memory/constitution.md), relevant accepted
ADRs/active decisions and the owner-selected work before editing.

Constitution 5.0.0 / Accepted ADR 0024 governs following owner-approved PR #828.
Existing audit obligations and historical reviewer restrictions remain intact.

Use the constitution for authority, lanes, quality, review and approval boundaries,
not a duplicate policy here. One issue/PR (or approved work order before one
exists) owns scope, criteria, evidence, review and outstanding work.
[Current work](../.squad/identity/now.md) only links to it.
Stay on `beta` unless otherwise authorized; preserve unrelated dirty files.

Load only matching [scoped instructions](instructions) and relevant
[native skills](skills/README.md); read matching files explicitly if the client
supplies only an index, and report discovery limitations. Load role
charters/history only when acting in that role. Verify actual paths and source.

Default to one implementation owner within the explicitly approved overall
budget. No speculative/nested delegation or silent model escalation.
Use the approved work order's named actions once; ordinary in-scope repairs do
not need renewed approval. No implicit setup/install/live settings, publication,
merge, release or deployment authority. Stop at budget, scope or authority limits.

Use [testing](../docs/testing.md#6-running-tests-locally-vs-ci) for commands and
Taskfile/CI as executable quality sources. Focused local checks plus applicable
exact-candidate CI may reuse unchanged valid evidence; unexecuted proof is pending.
Do not weaken tests, suppress warnings or substitute weaker types.

Where required by the lane, use ONE independent
[read-only review](agents/aurearia-reviewer.agent.md) through `task review:read-only`
with its native tool exclusions. Preserve original blocking-reviewer clearance
rights and historical author restrictions. Review repairs over the affected
delta unless broader evidence is invalidated; never self-clear.

Keep candidate-bound results and authorization in canonical work, not duplicate
inbox/log/history ledgers. Preserve historical evidence. Use the
[six-question PR template](pull_request_template.md); distinguish authored,
verified, accepted and released. Never auto-stage/stash/commit as a handoff.
When committing is authorized, follow Principle VII and stage only approved paths.
