---
updated_at: 2026-09-21
focus_area: P4 corrective UI-guidance contract verification
owner: Copilot CLI implementation owner; repository owner approves merge
work_branch: fix/delivery-ui-guidance-contract
baseline_commit: e08c4fd52181476654de02bbb3b33d6872cfef52
work_artifact: docs/agentic-delivery-improvement-plan.md
tasks_artifact: docs/agentic-delivery-improvement-plan.md
---

# Current Work

The owner merged P4 through #739, then approved a narrow follow-up for two UI
documentation assertions still reading the relocated guidance's old path.
Authorization includes web/delivery gates, bounded re-review and locked frontend
dependency restoration in this isolated worktree. No tool-version upgrade,
merge, deployment, application behavior change or P5/P6 work is authorized.
Preserve the original dirty beta worktree. #738's app-container job failed
downloading Syft with HTTP 504; do not misreport that run as fully green.

## Authoritative pointers

- [Action plan](../../docs/agentic-delivery-improvement-plan.md): D11/D12/D13 are
  the task ledger; this pointer deliberately does not duplicate their checklist.
- [Constitution](../../.specify/memory/constitution.md), sections 17-22, and
  [accepted ADR 0019](../../docs/adr/0019-evidence-based-agentic-delivery.md).
- [Active decisions](../decisions.md): lifecycle evidence and unresolved review
  records, including exact reviewer ownership and grandfathered restrictions.
- [Preservation inventory](../artifacts/context-curation-2026-09-21.json):
  original content identities and archive locations.
- [P3 handoff](../log/2026-09-21-delivery-validation-closeout.md):
  review, validation, owner acceptance and filename-preservation evidence.

## Product / Release State

Features 359, 362 and reduced 363 were included in owner-merged
[#732](https://github.com/briandenicola/Aurearia/pull/732).
Do not resume F015 at T012: the old pointer was stale. Consult each feature's
tasks/evidence before selecting work.

Feature 363 T044 (combined F014/F015 audit) and #732's final release checklist
remain unclosed in the inspected evidence. No deployment claim is made.
Historical review records R357-QA/R357-ARCH, R361, R352/R353, R225/R320, R337
and the conditional R-SWIPE evidence are linked in active decisions; this
governance batch does not repair or clear them.

## Next Action

The [P4 integration record](../../docs/agentic-native-integration.md) records
native discovery, explicit instruction loading, the restricted launcher and
the original probe reviewer's clearance. SpecKit adoption is deferred.
Original P4 candidate review passed. The
[P4 handoff](../log/2026-09-21-delivery-native-integration.md) records the exact
reviewed source and original verification; it is not retroactively rewritten.
The corrective contract's independent re-review passed; the
[corrective handoff](../log/2026-09-21-delivery-ui-guidance-correction.md) records
local gates and remaining conditions. Obtain hosted follow-up results and owner
acceptance for the PR into beta. Do not merge or begin P5 without authorization.
No application review restriction is cleared by delivery-tooling work.
