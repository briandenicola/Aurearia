# P5 Completion Evidence and Proposed Release Controls

Date: 2026-09-21
ID: GOV-006
Scope: Owner-authorized delivery plan D14-D15
Status: Repository draft independently reviewed; owner acceptance/live-control approval pending
Authority: Constitution Principles IV/VII/IX; sections 17-18/21; ADR 0019

The owner selected "Start P5" and "Draft the recommended controls". Implement
one canonical completion record in the existing PR/approved-work surface and
link it from checkpoint/handoff records. Do not create another status platform.
Bind criteria, sibling workflows, raw results and independent verdicts to the
tested/reviewed candidate; stale proof and unresolved blocks prevent acceptance.

The [control proposal](../../../docs/agentic-acceptance-controls.md) records
the read-only baseline, proposed main protection changes, solo-owner release
environment, activation sequence and exact restoration boundary. The checked-in
guard validates real GitHub approval history, trusted check results and the
candidate SHA; beta publishing and the disabled Ralph workflow stay unchanged.
The owner approved twenty main PR checks and nineteen push checks after live
evidence showed the CodeQL aggregate is PR-only. Four CodeQL analysis jobs
remain mandatory on the published SHA. The authorized sixteen-read/ten-minute
review returned PASS on tree `8e4187d920164f6cfc3bce72f7560bc82938b1a2`;
all five synthetic acceptance cases were correctly classified. The reviewer
used eleven reads/105 supervised seconds, without writes or delegated agents.
GitHub settings changes, main promotion, publication, deployment, tool upgrades
and P6 require their own explicit approval. Offline fixtures do not prove live
environment behavior or waive the manual acceptance/review requirements.

The [action plan](../../../docs/agentic-delivery-improvement-plan.md) owns task
criteria; its preparation receipt moves to the P5 PR for current evidence.
The [handoff](../../log/2026-09-21-delivery-acceptance-controls.md) preserves
the source/evidence identity. This decision does not clear
application review restrictions or modify immutable P4 handoff records.
