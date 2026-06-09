# 5. Streamline Constitution Principles

Date: 2026-06-09
Status: Proposed

## Context

The constitution had grown to 17 principles. Several were valuable but
overlapping: architecture with architecture enforcement, service boundaries
with AI isolation, strict typing with schema contracts, UI consistency with
design tokens and PWA rules, and security with auth/privacy/account lifecycle.

Recent coin edit regressions also exposed a separate gap: the constitution did
not plainly say that fixes must be simple, complete, and proportional.

## Decision

Consolidate the constitution into 9 principles:

1. Clear Layered Architecture
2. Service Boundary Separation
3. Strict Types and Explicit Contracts
4. Simple Complete Changes
5. Security, Auth, and Privacy by Default
6. Consistent User Experience
7. CI, Supply Chain, and Release Integrity
8. Documented Decisions
9. Automated Enforcement Over Manual Memory

Principle IV adds the simple-fix rule: choose the simplest complete
proportional change. This rejects two failure modes:

- **Too narrow:** fixing only the first observed stack trace.
- **Too clever:** adding broad abstractions or oversized rewrites for small bugs.

## Consequences

+ Agents get fewer top-level rules to remember.
+ Overlapping principles are consolidated without dropping binding intent.
+ The simple-fix standard is explicit: simple, complete, proportional.
+ PR review can reject hopeful narrow patches and clever oversized changes.
- Renumbering principles is a major constitution change and requires updating
  references in instructions, templates, PRs, and future reviews.
- Proportionality still requires human judgment.

## Related

- Constitution Principles I-IX
- Constitution Principle IV (Simple Complete Changes)
- Constitution §17 (Quality Gate)
- Constitution §21 (Definition of Done)
- Constitution §22 (Amendment Process)
- `.github/copilot-instructions.md`
- `.github/pull_request_template.md`
