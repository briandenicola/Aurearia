# Work Routing

How to choose an optional specialist. Subject to constitution sections 0 and 18.
ADR 0019 is Accepted via PR #734; constitution 4.0.0 governs.

## Routing Table

| Work Type | Route To | Examples |
|-----------|----------|----------|
| Go API, backend, models, repos, services | Cassius | New endpoints, DB migrations, middleware, agent proxy |
| Vue frontend, UI, components, stores | Aurelia | Components, views, Pinia stores, PWA, CSS |
| Architecture, design, cross-cutting | Maximus | System design, import rules, API contracts, scope decisions |
| Independent review | [aurearia-reviewer](../.github/agents/aurearia-reviewer.agent.md) | Read/search-only review; supply diff and validation evidence |
| Testing, QA, edge cases | Brutus | Write tests, verify fixes, coverage analysis |
| Scope & priorities | Owner, advised by Maximus | Authorization, trade-offs, decisions |
| Python agent, LangGraph, AI features | Cassius | Agent pipelines; consult Maximus only for needed architecture review |
| Session logging | Implementation owner or Scribe | Explicit, bounded recording assignment; await persistence |

## Issue Routing

| Label | Action | Who |
|-------|--------|-----|
| `squad` | Triage: analyze issue, assign `squad:{member}` label | Lead |
| `squad:{name}` | Candidate ownership; confirm scope and execution authorization | Named member |

### How Issue Assignment Works

1. When a GitHub issue gets the `squad` label, the **Lead** triages it — analyzing content, assigning the right `squad:{member}` label, and commenting with triage notes.
2. A `squad:{member}` label identifies candidate ownership, not permission to install, edit, publish, or release.
3. Reassignment requires applicable owner/reviewer authorization; it never clears a review block.
4. The `squad` label is the "inbox" — untriaged issues waiting for Lead review.

## Rules

1. **One implementation owner by default.** Delegate only bounded work needing separate context.
2. **Scribe is optional after amendment acceptance.** The owner of the work remains responsible for durable handoff and waits for required recording.
3. **Simple work stays direct.** Do not spawn agents for a short lookup or single trace.
4. **Choose the primary domain.** Consult others only for a concrete need.
5. **No speculative fan-out.** "Team" does not remove scope, effort, or permission limits.
6. **Every assignment has a lease.** State allowed paths, evidence, checkpoint, and stop condition; no unapproved nesting.
7. **Independent review stays independent.** Preserve all prior rejection restrictions; no self-clearance.

## Delegation and return contract

Before launch, supply every field below. Missing authority or an unapproved lease
means stop, not an invitation to infer defaults.

| Assignment field | Required content |
|---|---|
| Objective | One bounded outcome, lane, acceptance criteria and explicit non-goals |
| Authority | Owner-approved issue/spec/process plan; applicable policy and historical blocks |
| Candidate | Base and exact commit/tree or reproducible dirty-file manifest; supplied diff |
| Access | Named read/write paths, tool capabilities and prohibited side effects |
| Evidence | Required checks and supplied outputs, clearly distinguished from independent observations |
| Lease | Owner-approved maximum tool calls and elapsed time, checkpoint, and stop condition |
| Persistence | Exact return format and named handoff destination; writer and caller responsible |

Return PASS/BLOCK/INCOMPLETE, inspected/changed paths, evidence identity and
provenance, unresolved work, lease use and stop reason. An empty result or changed
files alone is INCOMPLETE. The caller inspects the result, verifies any authorized
writes and persists the handoff before reporting completion. Do not infer a
persisted record from a subagent's intention to write it.

For required independent review use `task review:read-only` and the native
profile, not a write-capable specialist renamed "reviewer". The profile permits
documented `read` and `search` aliases; the launcher excludes the installed
client's extra session SQL and skill-loading tools. No shell, edit, delegation,
model override or MCP wildcard is permitted. Supply Git
diffs and test output because it cannot run commands. If the client cannot enforce
the profile, stop and obtain an approved alternative; a prose promise is not a
capability boundary. Maximus may advise on architecture, but historical reviewer
ownership does not transfer to the new profile. Existing blocks retain their
original clearance sequence.
