---
updated_at: 2026-10-18
focus_area: Private original attribution-card retention and opt-in Share attachments
owner: Copilot CLI implementation owner; repository owner accepts results
work_branch: beta
baseline_commit: 86f64bdda864c34948ae48dba8da04bdc67e5ece
work_artifact: docs/attribution-card-retention.md
tasks_artifact: docs/attribution-card-retention.md
---

# Current Work

Retain attribution cards privately with opt-in Share attachments. Sale recording
is unchanged. Prior delivery/share-title work merged in #826.

## Next Action

PR #827 is open. Repair its CodeQL extension-path alerts using fixed allowlisted
suffixes; owner approved one reviewer read, then commit/push beta.
See [repair](../log/2026-10-18-attribution-extension-codeql.md).
Software QC audit remains required before merge; no merge/deployment authorized.
No builds, installs, containers, main merge or deployment. See
[handoff](../log/2026-10-17-attribution-card-retention.md).
