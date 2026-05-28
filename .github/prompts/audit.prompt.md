Periodic audit per Constitution §20 (Audit & Continuous Improvement). Cadence:
weekly on active features, plus a deeper review at every release boundary.

Owners: **Maximus** (architectural / constitutional drift) and **Brutus**
(security / quality drift). Findings recorded under `docs/audits/`.

Steps:

1. **Re-read the gates** — `.specify/memory/constitution.md` (full text,
   not just the active spec).
2. **Run `/speckit.analyze`** against the active `specs/NNN-*/` to surface
   spec ↔ plan ↔ tasks drift. Capture its raw output.
3. **Maximus review** — walk every Principle (I–XVI) and operational section
   (§0, §17–§22). For each, note: `PASS`, `DRIFT` (with file:line + the
   higher-authority document violated), or `N/A`.
4. **Brutus review** — security & quality lens:
   - OWASP API Top 10 (2023) per protected endpoint
   - JWT issuance / refresh / revocation per Principle XII
   - Input validation, output encoding, secret handling per Principle XI
   - GORM: parameterization, N+1, missing indexes
   - Container: non-root, read-only FS, no exposed secrets
   - Recent commits (last 7 days) for drift from spec or Conventional Commits
5. **Architecture tests** — `go test -v -run TestNo ./src/api/...` to confirm
   Principle I import rules still hold (per Principle X).
6. **Write findings** — append to `docs/audits/{YYYY-MM-DD}.md` (create the
   folder and file if missing). One row per finding:
   `| severity | principle/§ | file:line | summary | proposed fix |`.
   Severities: `Critical` / `High` / `Medium` / `Low` / `Info`.
7. **Open issues** — for every `High`/`Critical`, open a GitHub issue via
   `gh issue create` titled `audit: <summary>` and link the audit file.
8. **Log + decisions** — append a one-line summary to `.squad/log/` and, if
   the audit requires a binding decision (e.g., revoke a previous waiver),
   drop a card into `.squad/decisions/inbox/`.

Output to chat: a table of findings grouped by severity. Do NOT make code
changes from this prompt — audits surface, they do not fix. Fixes happen via
normal spec → plan → tasks flow.

References: Constitution §20 (Audit & Continuous Improvement), §0 (Hierarchy
of Authority), §22 (Amendment Process). Squad agents: Maximus + Brutus
(co-owners), Scribe (logs).
