# `specs/` — SpecKit Workflow On Disk

This directory is the on-disk home of the SpecKit lifecycle referenced by the
Constitution's **§0 Hierarchy of Authority** (items 3–6: spec → plan → tasks →
backlog card). Every active feature lives here as a folder; every queued but
unscheduled idea lives as a card under `_backlog/`.

## Layout

```text
specs/
├── README.md                     # this file
├── NNN-feature-slug/             # ACTIVE feature (one per scheduled feature)
│   ├── spec.md                   # /speckit.specify output  (required)
│   ├── plan.md                   # /speckit.plan output     (required)
│   ├── tasks.md                  # /speckit.tasks output    (required)
│   ├── checklist.md              # /speckit.checklist       (optional)
│   ├── notes.md                  # ad-hoc research / context (optional)
│   ├── research.md               # /speckit.plan Phase 0    (optional)
│   ├── data-model.md             # /speckit.plan Phase 1    (optional)
│   ├── quickstart.md             # /speckit.plan Phase 1    (optional)
│   └── contracts/                # /speckit.plan Phase 1    (optional)
└── _backlog/
    ├── README.md                 # backlog rules (read me before adding cards)
    ├── _TEMPLATE.md              # copy this when creating a new card
    └── F0NN-feature-slug.md      # one card per queued idea
```

## Numbering

| Range  | Used for                | Assigned                          | Mutable? |
|--------|-------------------------|-----------------------------------|----------|
| `NNN`  | Active feature folders  | At promotion from backlog         | **No**   |
| `F0NN` | Backlog cards           | At card creation (next free slot) | **No**   |

`NNN` starts at `001` and increments monotonically (`001`, `002`, …). `F0NN`
starts at `F001`. Numbers are **never reused** — even if a card is dropped or a
feature is abandoned, its number is retired with it. This keeps git history,
PR titles, and commit messages permanently dereferenceable.

## Lifecycle

```text
idea
  │
  ▼
specs/_backlog/F0NN-slug.md   (status: backlog)
  │  Lead triage (weekly)
  ▼
status: triaged               (priority, effort, acceptance criteria set)
  │  Scheduled into a release
  ▼
status: promoted              (NNN assigned, folder created via /speckit.specify)
  │
  ▼
specs/NNN-slug/spec.md   →  plan.md  →  tasks.md  →  implementation
  │
  ▼
tasks complete                (spec archived in place — NOT deleted)
```

A promoted card keeps its file as a historical pointer; add a
`**Promoted to**: specs/NNN-slug/` line and set `status: promoted`. A dropped
card sets `status: dropped` with a `## History` entry explaining why.

## Templates & Prompts

Per-feature folders use the SpecKit templates at:

- `.specify/templates/spec-template.md`
- `.specify/templates/plan-template.md`
- `.specify/templates/tasks-template.md`

Generate / edit them via the slash prompts:

| Stage          | Prompt                         |
|----------------|--------------------------------|
| Spec           | `/speckit.specify`             |
| Clarify spec   | `/speckit.clarify`             |
| Plan           | `/speckit.plan`                |
| Tasks          | `/speckit.tasks`               |
| Checklist      | `/speckit.checklist`           |
| Cross-artifact | `/speckit.analyze`             |
| Implement      | `/speckit.implement`           |
| Issue export   | `/speckit.taskstoissues`       |

## Gates

Every spec / plan / tasks artifact must satisfy:

- **§0 Hierarchy of Authority** — when a spec contradicts the constitution or
  PRD, the higher document wins; the spec must be updated in the same PR.
- **§17 Quality Gate** — the per-PR checklist (build, lint, tests, Conventional
  Commit, Co-authored-by trailer, no secrets) gates every commit touching
  `specs/NNN-*/`.
- **§21 Definition of Done** — the 14-item checklist in the PR template is the
  final gate before a feature is considered complete.

Read backlog rules in `_backlog/README.md` before creating a card.
