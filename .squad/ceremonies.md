# Ceremonies

> Risk-based coordination under constitution sections 18 and 20.
> ADR 0019 consumer changes are Proposed, not activated by local preparation.

## Design Review

| Field | Value |
|-------|-------|
| **Trigger** | risk-based within approved work |
| **When** | before |
| **Condition** | material shared contract, architecture, migration, or delivery-control decision |
| **Facilitator** | lead |
| **Participants** | implementation owner and one independent reviewer; others only if needed |
| **Time budget** | explicit bounded lease; stop at decision or blocker |
| **Enabled** | ✅ yes |

**Agenda:**
1. Review the task and requirements
2. Agree on interfaces and contracts between components
3. Identify risks and edge cases
4. Assign action items

---

## Retrospective

| Field | Value |
|-------|-------|
| **Trigger** | repeated/systemic failure or owner request |
| **When** | after |
| **Condition** | recurring failure or incident requiring a process change; not every failed test |
| **Facilitator** | lead |
| **Participants** | smallest group needed for the concrete cause |
| **Time budget** | explicit bounded lease |
| **Enabled** | ✅ yes |

**Agenda:**
1. What happened? (facts only)
2. Root cause analysis
3. What should change?
4. Action items for next iteration

## Audits and closeout

- Software QC: major/high-risk work and release readiness, scoped to the candidate.
- Agentic delivery: after every owner-designated major release; track prior actions.
- Explicitly invoke and record the audit at release closeout. A skill is not a trigger.
- Audits are read-only unless report/issue writes are authorized.
- Preserve per-release SBOM/threat-model and scheduled product/dependency/restore reviews.
- Record corrective-action evidence; a retrospective alone does not close a finding.
- The implementation owner records the handoff or delegates to Scribe and waits.
- No ceremony authorizes scope expansion, commits, installs, deployments, or release.
