# `specs/_backlog/` — Queued Feature Cards

Lightweight intake for ideas that are **not yet scheduled**. A card captures
just enough context to triage; it is NOT a spec. When a card is ready to
become real work, it is **promoted** into `specs/NNN-slug/spec.md` via
`/speckit.specify`.

See `../README.md` for the full SpecKit lifecycle and how this directory fits
the Constitution's §0 Hierarchy of Authority (backlog cards rank #6).

## File naming

```text
F0NN-kebab-slug.md
```

- `F0NN` — next free number (`F001`, `F002`, …), assigned at create time and
  **immutable** thereafter.
- `kebab-slug` — short, descriptive, lowercase, hyphen-separated.

Examples: `F001-coin-of-the-day.md`, `F014-bulk-image-import.md`.

## Required fields (see `_TEMPLATE.md`)

Cards use YAML frontmatter for structured triage fields and prose body for the
human-readable content. The 15 required fields are:

| Frontmatter         | Body section                |
|---------------------|-----------------------------|
| `id` (F0NN)         | `## Summary`                |
| `title`             | `## Acceptance criteria`    |
| `status`            | `## Constitution alignment` |
| `priority` (P0–P3)  | `## Open questions`         |
| `effort` (XS–XL)    | `## Notes`                  |
| `value` (1–5)       |                             |
| `risk` (1–5)        |                             |
| `owner`             |                             |
| `created` (date)    |                             |
| `updated` (date)    |                             |

### `status` values

- `backlog` — captured, not yet triaged.
- `triaged` — Lead has set priority/effort/value/risk and acceptance criteria
  are clear enough to estimate.
- `promoted` — `NNN` assigned, `specs/NNN-slug/` folder created. Add a
  `**Promoted to**: specs/NNN-slug/` line at the top of the body.
- `dropped` — will not be built. Add a `## History` note explaining why; the
  file stays in place (number is retired, not reused).

## Promotion rule

A card may be promoted when **all** of the following are true:

1. `status: triaged`
2. Acceptance criteria are concrete (testable, not aspirational).
3. Constitution alignment section names at least one Principle and one
   operational section (e.g., "Principle XI + §17").
4. No `priority: P3` cards are promoted ahead of `P0`/`P1` without a written
   exception in `.squad/decisions/inbox/`.

To promote:

1. Pick the next free `NNN` (highest existing + 1, never reuse).
2. Run `/speckit.specify` with the card body as input → creates
   `specs/NNN-slug/spec.md`.
3. Update the card: set `status: promoted`, add `**Promoted to**:` link, bump
   `updated`.
4. Commit both files in the same PR with `feat:` prefix and the standard
   `Co-authored-by: Copilot` trailer.

## Triage cadence

- **Lead (Maximus)** reviews all `status: backlog` cards **weekly**.
- Aim to either advance to `triaged` or `dropped` within two review cycles —
  perpetually-`backlog` cards rot and lie about themselves.
- Triage notes that change a card's verdict go into `.squad/decisions/inbox/`
  per Constitution §18, not into the card itself.

## Don'ts

- **Don't** invent an `NNN` for a card — promotion is the only way to get one.
- **Don't** edit a `promoted` card's body except to add a `## History` entry;
  the source of truth has moved to `specs/NNN-slug/spec.md`.
- **Don't** delete a card. Set `status: dropped` and keep the file.
