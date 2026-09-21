Run the explicitly selected audit under constitution section 20.
ADR 0019 cadence changes remain Proposed until section 22 acceptance.

1. Confirm the audit type and boundary:
   - Software QC for major/high-risk work or release readiness: exact base/head,
     requirements, contracts, and affected workflows.
   - Agentic delivery after an owner-designated major release: governance,
     instructions, state, enforcement, coordination, and prior corrective actions.
2. Read applicable authority, the exact changes/evidence, and relevant current
   decisions. Check all applicable Principles I-IX, not obsolete I-XVI labels.
3. Load the matching audit skill if available. Do not claim a repository-native
   skill is installed before P4. `/speckit.analyze` is only cross-artifact
   spec/plan/tasks analysis, not a replacement for either audit.
4. Investigate directly unless a bounded specialist context is genuinely needed.
   No mandatory fan-out, weekly broad ceremony, builds, or installs by default.
5. Report confirmed blockers, verification gaps, and follow-ups separately with
   file/line or exact command/commit evidence. State PASS, BLOCKED, or INCOMPLETE
   for the agreed scope. Green CI does not rule out untested defects.
6. Save a report, create issues, or update handoff records only when authorized.
   Do not fix code, alter settings, clear existing reviewer blocks, or release.

Software checks use the actual package/workflow recipes in `docs/testing.md` §6,
including lint and strict types. Permission/setup gaps remain incomplete.
For delivery audits, distinguish written policy from observed live enforcement.
Record release-closeout invocation explicitly: a skill alone is not a trigger.
A finding closes only with corrective-action evidence.
