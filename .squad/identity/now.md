---
updated_at: 2026-09-28
focus_area: Frontend typography compliance
owner: Copilot CLI implementation owner; repository owner accepts visual result
work_branch: beta
baseline_commit: 15ff45ffb6c91d34b09c6360c514a0b0b1813ea2
work_artifact: docs/audits/2026-09-28-typography-compliance.md
tasks_artifact: docs/audits/2026-09-28-typography-compliance.md
---

# Current Work

The owner authorized a repository-wide font-family audit and full remediation
after Admin Schedules showed weak heading/body hierarchy in desktop and PWA
screenshots. The implementation is complete on `beta`: fonts are bundled locally,
family use is tokenized and enforced, and Schedule headings match the documented
typography scale.

## Authoritative pointers

- [Completion receipt](../../docs/audits/2026-09-28-typography-compliance.md)
- [Constitution](../../.specify/memory/constitution.md), Principles IV, VI and IX
- [Web design guidance](../../.github/instructions/web.instructions.md)
- [Active decisions](../decisions.md), UI-001 and ENG-002

## Next Action

Commit and push the exact verified candidate to `beta`. Physical installed-PWA
inspection is owner acceptance evidence; no deployment or release is authorized.
