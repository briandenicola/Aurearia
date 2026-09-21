# Work Routing

How to choose an optional specialist. Subject to constitution sections 0 and 18.
ADR 0019 is Accepted via PR #734; constitution 4.0.0 governs.

## Routing Table

| Work Type | Route To | Examples |
|-----------|----------|----------|
| Go API, backend, models, repos, services | Cassius | New endpoints, DB migrations, middleware, agent proxy |
| Vue frontend, UI, components, stores | Aurelia | Components, views, Pinia stores, PWA, CSS |
| Architecture, design, cross-cutting | Maximus | System design, import rules, API contracts, scope decisions |
| Code review | Maximus | Review PRs, check quality, approve/reject |
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
