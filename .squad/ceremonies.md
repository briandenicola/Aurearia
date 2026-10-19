# Ceremonies

> Risk-based coordination under constitution sections 18 and 20.
> ADR 0019 is Accepted via PR #734; constitution 4.0.0 governs.

> Prepared for Proposed ADR 0024 / 5.0.0. Until §22 approval/merge apply effective
> 4.0.0 from the constitution's linked baseline, including #827's audit.
> The following prospective triggers activate together.

## Design Review

| Field | Value |
|-------|-------|
| **Trigger** | risk-based within approved work |
| **When** | before |
| **Condition** | one scoped feature/high-risk review; combine design concerns without a duplicate ceremony |
| **Facilitator** | lead |
| **Participants** | implementation owner and one independent reviewer; others only if needed |
| **Time budget** | explicit bounded lease; stop at decision or blocker |
| **Enabled** | yes |

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
| **Enabled** | yes |

**Agenda:**
1. What happened? (facts only)
2. Root cause analysis
3. What should change?
4. Action items for next iteration

## Audits and closeout

- Software QC: major architecture or owner-designated major releases, not every
  beta-to-main PR. Keep existing audit obligations and applicable high-risk proof.
- Agentic delivery: after every owner-designated major release; track prior actions.
- Explicitly invoke and record the audit at release closeout. A skill is not a trigger.
- Audits are read-only unless report/issue writes are authorized.
- Preserve per-release SBOM/threat-model and scheduled product/dependency/restore reviews.
- Record corrective-action evidence; a retrospective alone does not close a finding.
- The implementation owner records evidence in canonical issue/PR; no mandatory
  separate handoff/inbox/history. Routine work needs no independent ceremony.
- No ceremony authorizes scope expansion, commits, installs, deployments, or release.
- Use the [assignment/return contract](routing.md#delegation-and-return-contract)
  for any delegated review or recording. A ceremony does not bypass the native
  reviewer's read/search-only capabilities or extend its approved lease.
