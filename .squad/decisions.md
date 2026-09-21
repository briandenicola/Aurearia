# Active Squad Decisions

Curated 2026-09-21 under Constitution 4.0.0 section 18.3 and owner-authorized
D05/P2. This is a current view, not a replacement constitution or task ledger.
Start with [current work](identity/now.md); follow the selected specification.

## Authority and preservation

Constitution > PRD > applicable accepted ADRs > active spec > plan > tasks >
backlog > active decisions > agent judgment. Framework instructions and historical
role notes cannot override that order.

All previous content is preserved unchanged in the
[decision snapshot](decisions-archive.md#curation-2026-09-21-decisions).
The pre-existing archive is unchanged; oversized role histories have adjacent
`history-archive.md` snapshots. The [inventory](artifacts/context-curation-2026-09-21.json)
records the baseline commit, SHA-256 hashes, byte ranges, and every heading/section.
Its `decisions-L...` identifiers use the ORIGINAL ledger line, not the appended
archive line. The old current-work pointer is also preserved, but superseded.

Archiving does not revoke a decision, clear a rejection, or prove a defect still
exists. Domain-specific decisions remain applicable unless higher authority or
an explicit superseding record says otherwise. Search the inventory for the
selected domain before retrieving its historical evidence; do not load the whole
archive routinely.

## Current cross-cutting decisions

| ID | Scope / status | Decision and authoritative source |
|---|---|---|
| GOV-001 | Delivery policy / Accepted | Constitution 4.0.0 and [ADR 0019](../docs/adr/0019-evidence-based-agentic-delivery.md) took effect through owner merge of [#734](https://github.com/briandenicola/Aurearia/pull/734); [#735](https://github.com/briandenicola/Aurearia/pull/735) recorded acceptance. Historical proposal wording is not a pending activation gate. |
| GOV-002 | Execution / Active | Use one implementation owner, the smallest sufficient lane, explicitly bounded delegation, and independent review where required. Evidence distinguishes implemented, verified, accepted, and released. Constitution sections 17-22 supersede older mandatory fan-out and author-never-repairs guidance for NEW reviews only. Existing restrictions below retain their terms. |
| GOV-003 | Context / Active | Current decisions <=20 KiB, role history <=12 KiB, now.md <=100 lines are warning budgets, never deletion authority. Preserve originals before curation. Current work belongs only in now.md with links to authoritative tasks, not another status ledger. Constitution section 18.3. |
| GOV-004 | P3 validation / Implemented, review and hosted evidence pending | The Taskfile `check:*` targets are shared with CI; setup stays separate and missing tools/lint fail. Offline checks validate active structural references, not historical approval or prose meaning. Small work may point to a project issue rather than an invented spec. See [bounded execution record](decisions/inbox/copilot-p3-validation.md) and [testing guide](../docs/testing.md#6-running-tests-locally-vs-ci). No historical block is cleared; live protections and native skill migration remain P5/P4 work. |
| ENG-001 | Architecture / Active | Preserve Go-owned authentication, durable state, data access and typed contracts; Python stays stateless. Follow Constitution Principles I-III and applicable ADRs. A framework or role memory is not permission to cross a service/data boundary. |
| ENG-002 | Validation / Active | Prove the exact changed workflow and sibling paths, including meaningful negative/guard cases. Apply the section 17 scope matrix; do not infer verification from checked tasks or substitute local Windows evidence for a required Linux/browser gate. Older command transcripts describe their original artifacts, not current passing results. |
| UI-001 | Shared UI behavior / Active | Reuse local component/token/navigation patterns. Narrow tables retain reachable columns through contained horizontal scrolling. Swipe consumers share the common primitive rather than parallel gesture engines. [Table decisions][tables] and [shared-swipe review][swipe] preserve scope and outstanding device conditions. |
| SEC-001 | Background removal / Accepted design | Eval-requiring model code is isolated in the dedicated same-origin module worker; do not relax app-wide CSP. [ADR 0014](../docs/adr/0014-background-removal-worker-csp-isolation.md) and [Brutus's explicit worker clearance][worker-clear] govern; the earlier app-wide unsafe-eval suggestion is not current authority. Real-photo browser smoke guidance remains in that clearance. |
| AI-001 | Coin Copilot / Accepted MVP and subsequent scoped extensions | Go owns durable run state; Python executes bounded allowed tools. The original read-only MVP excludes writes and dollar-cost enforcement; later capabilities require their own authorized spec. Preserve legacy fallback, existing drawer entry, explicit confirmation boundaries and iteration/tool/time/concurrency/token/payload controls. [Original scope decision][copilot], [ADR 0016](../docs/adr/0016-go-owned-durable-coin-copilot-state.md), Features 359/361/362/363. |
| AI-002 | Identification / Accepted | Deep Analysis reuses separate collection-grade face analysis and optional notes before provider verification, retaining evidence, disagreements, confidence and narrative. Quick Lookup remains the fast combined-image NGC-first path. [ADR 0018](../docs/adr/0018-role-specific-deep-analysis.md) supersedes ONLY ADR 0012's single-vision-call constraint. |
| AI-003 | Reduced F015 / Owner-approved scope, implementation recorded | [Feature 363](../specs/363-collector-curator-watchlist-provenance/spec.md) authorizes private collector context, read-only curator guidance, existing UI-owned Add to Wishlist, and native listing-URL intake. It does not authorize an auction rewrite, wishlist-action platform, action/audit tables, watchlist ranking or provenance-risk workflow. The archived T011 pause is superseded by the later task/evidence records; do not restart completed work. |

## D05 lifecycle reconciliation

Evidence checked on 2026-09-21 against baseline
`e6ab8313346b971dce4b288804036222f7d4c95d`. Approval and implementation are
different claims; merged code alone is not reviewer clearance.

| Record | Reconciled state / evidence |
|---|---|
| ADR 0005 | Accepted through owner-merged [#248](https://github.com/briandenicola/Aurearia/pull/248), 2026-06-09, merge `00c7403fef85f6c420ee7dcce7cb6cf87a26bccb`. PR explicitly adds the ADR and constitution consolidation. |
| ADR 0011 | Accepted through owner-merged [#626](https://github.com/briandenicola/Aurearia/pull/626), 2026-08-16, merge `c7e11ac82e61ed43cb6ad5db3020480cc67301ed`. PR explicitly names the added ADR. |
| ADRs 0016 / 0018 | Accepted under the ADR index's merge-promotes-status rule: both ADR files were added in owner-merged [#732](https://github.com/briandenicola/Aurearia/pull/732), 2026-09-20, merge `b03ac3a2c146936952e11b69bfd8dae57c8f744e`. This accepts the documents, not unrecorded reviewer clearance or release-wide audit completion. |
| ADRs 0013 / 0017 | Already Accepted in their source headers; index corrected in #734. No new approval inferred by this batch. |
| Feature 359 | Implementation and validation recorded through [T070](../specs/359-coin-copilot-harness/tasks.md); included in #732. No longer ready to start implementation. Final release-wide evidence remains qualified by #732's open audit checklist. |
| Feature 362 / F014 | Implementation, compatibility evidence and feature-specific QC PASS recorded in [quickstart evidence](../specs/362-coin-copilot-attribution/quickstart-evidence.md#post-major-work-qc-audit--feature-362--f014); included in #732. This feature-specific audit does not substitute for the combined F014/F015 audit. |
| Feature 363 / F015 | [Tasks](../specs/363-collector-curator-watchlist-provenance/tasks.md) T001-T043 checked; [quality evidence](../specs/363-collector-curator-watchlist-provenance/quickstart-evidence.md) records implementation checks. T044 remains open and its referenced combined `qc-audit.md` is absent. Included in #732, not paused after T011 and not fully release-verified. |
| Feature 357 Quick Access | Frontend implementation exists, but explicit Brutus and Maximus rejection records remain without located clearance. T093 cannot certify acceptance; T095/T096 remain outstanding. Preserve the frozen backend boundary and reviewer sequence below. |
| Release state | #732 merged into main; its body still says final release-wide audit/acceptance is pending. No GitHub Release objects were returned by the release-list query. This is not evidence of deployment absence or success. Do not infer deployment or retroactively certify the missing audit. |

Header corrections only: accepted ADR bodies and landed spec requirement bodies
are unchanged. Current metadata and the Feature 357 acceptance checkbox are
reconciled; old evidence is not rewritten.

## Unresolved review and release evidence

These are unresolved RECORDS, not newly diagnosed application vulnerabilities.
Only the original reviewer, or an owner-appointed independent successor after
re-review, can clear a block unless that reviewer already specified an automatic
evidence condition. This curation clears none. Assignment alone is not clearance.
Older author restrictions are retained even though new reviews use ADR 0019.

| ID | Scope / recorded restriction | Owner and next evidence needed |
|---|---|---|
| R357-QA | Quick Access frontend: [Brutus REJECT and re-review][quick-access-qa]. Aurelia and Livia are excluded from the next revision; Brutus cannot author that revision. | Brutus explicit CLEAR/APPROVE after an eligible independent revision; no clearance located. Marcus was assigned in the architecture record, not declared successful. |
| R357-ARCH | Quick Access frontend: [Maximus final REJECT][quick-access-arch], including unproven task claims, lifecycle/image refresh and mobile coverage. | Brutus clearance first, then Maximus re-review. T093 unchecked; T095/T096 remain open. No application repair is authorized by D05/P2. |
| R361 | Specialist Python contracts T002/T003/T004/T006/T008: [Brutus REJECT][specialist-block]. Cassius excluded; Livia assigned independent revision. | Locate Brutus's explicit clearance or request authorized re-review. Later implementation/merge and an assignment are insufficient. |
| R352 | Structured results Phases 3/4: [independent revision notes][structured-block] call themselves cleared but explicitly request Brutus re-review. | Brutus clearance evidence not established in this reconciliation. Preserve original independent-revision restrictions; do not accept author self-clearance. |
| R353 | Availability-run spec/plan/tasks: [Cassius revision record][availability-block] reports three Brutus findings repaired under strict lockout. | Locate original Brutus approval, not merely the reviser's "approved" label. Until then preserve the original-author restriction for that rejected scope. |
| R-SWIPE | [Maximus conditional approval][swipe]: B2-B5 and round-one lockout cleared. B1 converts automatically only on the specified green ubuntu Vue job with the guard passing; owner iOS/Android PWA checks remain release conditions. | Link the exact qualifying runner result and device acceptance. Do NOT invent a continuing blanket lockout: Aurelia/Brutus become ineligible again only if the stated B1 failure condition fires; Livia owns that revision. |
| R225 | [Brutus Mint Map REJECT][mint-block]: lint and phone verification; next revision must be non-Aurelia. | Locate Brutus clearance and device evidence. The separate 50-coin-cap approval is not proof that this earlier rejection was cleared. |
| R320 | [Brutus combined #316/#320/#322 BLOCK][toolchain-block]: toolchain source mismatch; non-original reviser required. | Locate explicit clearance for that batch. Later toolchain versions do not themselves clear the historical record. |
| R337 | Wishlist Search Alerts, issue #357 (NOT Feature 357 Quick Access): [Maximus BLOCK and Scribe completion summary][wishlist-block]. | Scribe reports Maximus approval but the underlying reviewer record was not established here. Locate that evidence; do not promote a transcription into reviewer clearance. |
| R-RELEASE | Combined F014/F015 release audit T044 and #732 final acceptance checklist remain unclosed in the inspected records. | Owner-authorized, separately scoped release audit and explicit disposition. D05/P2 is not that application audit and does not authorize a new release. |

Other historical blocks remain preserved and searchable. This is not a global
adjudication of every historical review. When a previously unselected artifact is
touched, inspect its indexed review chain; absent clearance remains unresolved.

## Located clearance pairs (do not reopen from an old REJECT alone)

| Scope | Evidence and limitation |
|---|---|
| External tools #218 | Original Maximus block in archive, explicitly cleared by Maximus at [archive lines 5659 onward][external-clear]. |
| Private media #313 / outbound #310 / public hardening / security gates #323 | Brutus's explicit scoped approvals at [archive lines 9813-10119][security-clear]. #323's historical settings evidence is not a claim about today's live protection configuration; P5 must check it anew. |
| Feature 341 / ADR 0008 | [Maximus final release clearance][feature341-clear] resolves the earlier plan/closure restrictions. |
| Valuation Feature 356 | [Maximus B1-B4 clearance][valuation-clear] and [Brutus B1/B2 clearance][valuation-qa-clear]; do not reclassify the preceding revisions as open. |
| Background-removal worker | [Brutus re-review][worker-clear] explicitly clears `3a0d7b04`; retained manual smoke guidance is not a new blanket rejection. |
| Architecture #317 | Brutus history records explicit re-review approval in the [preserved history](agents/brutus/history-archive.md#curation-2026-09-21-brutus), original line 565, dated 2026-06-19T15:21:36Z. |

[tables]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L72-L119
[copilot]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L11289-L11343
[worker-clear]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L11463-L11569
[quick-access-qa]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L11729-L11886
[quick-access-arch]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L12037-L12154
[specialist-block]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/agents/brutus/history.md#L888-L918
[structured-block]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L6995-L7088
[availability-block]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L6893-L6930
[swipe]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L10912-L11035
[mint-block]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions-archive.md#L9711-L9738
[toolchain-block]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions-archive.md#L11160-L11185
[wishlist-block]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions-archive.md#L12261-L12304
[external-clear]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions-archive.md#L5659-L5680
[security-clear]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions-archive.md#L9813-L10119
[feature341-clear]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L4682-L4745
[valuation-clear]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L2260-L2300
[valuation-qa-clear]: https://github.com/briandenicola/Aurearia/blob/e6ab8313346b971dce4b288804036222f7d4c95d/.squad/decisions.md#L2483-L2545
