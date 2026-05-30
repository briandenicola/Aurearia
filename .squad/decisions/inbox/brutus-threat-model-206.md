---
date: 2026-05-28
author: Brutus (QA)
status: Proposed
issue: "#206"
---

# Decision: Threat Model Issue-Link Mechanism

## Context

Issue #206 requires that **all OPEN threat-model findings have GitHub issue links for execution tracking**. Audit of `docs/threat-model.md` revealed:
- **15 OPEN findings** (after audit corrections)
- **0 issue links** currently in document
- No mechanism or template for linking findings to tracking issues

## Problem

Without explicit issue links:
1. Open findings have no accountability — no way to know if they're being tracked or who owns them
2. Finding → issue mapping is implicit and manual, prone to loss during backlog churn
3. PR workflow has no way to validate that a finding is addressed in code without externally searching issues

## Solution

Add a **Findings Tracker** column to each finding table entry that:
1. **Format:** Add issue link as `#NNNN` in the Description or Status column (requires decision on UX)
2. **Policy:** Every OPEN finding must have a corresponding open GitHub issue with label `security-finding` and reference in threat-model.md
3. **CI Gate:** Linter (or manual PR checklist item) verifies no OPEN status without issue link
4. **Lifecycle:** When finding is MITIGATED, issue is closed with reference to the PR that fixed it

## Alternative (Rejected)

Keep finding descriptions generic and maintain a separate mapping document (`docs/security-findings-backlog.md`) — rejected because it decouples source of truth and creates duplicate work.

## Acceptance Criteria

1. ✗ Create 15 tracking issues for existing OPEN findings (separate effort, outside #206 scope)
2. ✓ Update threat-model.md template (§ How to add a new threat finding) to require issue link for Open status
3. ✗ Add PR template checklist item (if not already present in `.github/pull_request_template.md`)

## Timeline

- Issue link creation: tracked in **new issue #XXX** (TBD by Coordinator)
- Template update: included in **#206 PR**
- CI automation: **phase 3c backlog** (SECURITY.md enforcement)

## Team Input Needed

- **Maximus (arch):** Should issue link live in the Description cell or a separate column?
- **Scribe:** Which issue labels to use for security findings backlog?
- **Ralph (CI):** Can we add a linter check for threat-model.md format in pre-commit?

---

**Co-authored-by:** Brutus <brutus@squad.test>
