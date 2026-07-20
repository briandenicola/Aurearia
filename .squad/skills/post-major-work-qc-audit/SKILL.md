---
name: "post-major-work-qc-audit"
description: "Deep quality-control audit to run after major features, migrations, or integrations land — covers engineering best practices, security, docs, architecture, test coverage, supply chain, UX, and operational readiness"
domain: "quality-assurance"
confidence: "low"
source: "manual"
tools:
  - name: "grep / glob / view"
    description: "Navigate changed files, read specs, ADRs, CI config, and dependency manifests"
    when: "All investigation phases — always prefer direct reading over inference"
  - name: "git diff / git log"
    description: "Identify the exact changeset and authorship boundary for the audit"
    when: "Opening step to bound the scope of the audit"
---

## Context

Run this skill when a significant batch of work has landed: a large feature, a security-sensitive change, a cross-cutting refactor, a new integration, or a migration that touches schema/API/contracts. Normal CI and per-PR review catch local defects; this skill catches systemic gaps — spec drift, missing regression coverage, deployment blind spots, and cross-cutting concerns that only surface when the whole change is viewed together.

**Trigger conditions (any one suffices):**
- Feature spans 3+ files across multiple layers or packages
- New external dependency, API contract, schema migration, or auth surface introduced
- Security-sensitive paths modified (auth, permissions, secrets, user data)
- AI agent or background scheduler behaviour changed
- PWA/mobile interaction or service-worker boundary touched
- Public deployment readiness being evaluated

**Portability note:** Sections marked `[REPO-HOOK]` contain hooks specific to repositories that use a constitution/Quality Gate/Squad governance model. Skip or adapt those hooks when running in other repos.

---

## Invocation Pattern

```
Run a post-major-work QC audit on [feature/PR/branch description].
Scope: [list of changed files, or "git diff main..branch"].
Spec/design docs: [path(s) to spec.md, plan.md, ADRs, PRD sections].
Prior decisions: [path to decisions.md or ADR index, if present].
```

Before writing a single finding, complete the full Investigation Protocol below. Do not skip phases to save time.

---

## Investigation Protocol

Perform every phase in order. Record what you read, not what you assume.

### Phase 0 — Bound the Changeset
1. Run `git diff --name-only <base>..<head>` (or equivalent) to enumerate every changed file.
2. Identify the primary domains touched: backend, frontend, agent/ML, infra/config, docs, tests, migrations.
3. Confirm the spec, PRD section, or design doc that authorized this work.  
   `[REPO-HOOK]` Check `.specify/memory/constitution.md` §0 hierarchy: Constitution → PRD → spec → plan → tasks → decisions.

### Phase 1 — Diff Reading
4. Read every changed non-test source file. Note: new exports, removed exports, signature changes, error-path changes, side effects.
5. Read every changed test file. Record: what scenarios are covered, what is explicitly absent.
6. Read every changed config, migration, Dockerfile, workflow, or lock file.
7. Read every changed documentation file.

### Phase 2 — Reference Reading
8. Read the authorizing spec/plan fully, including acceptance criteria and non-goals.
9. Read any ADRs cited or created by this work.
10. Read CI workflow files to understand what gates already run.  
    `[REPO-HOOK]` Read `.github/pull_request_template.md` DoD checklist to understand what the author self-certified.

### Phase 3 — Cross-Cutting Checks (see Audit Domains below)
11. Work through each Audit Domain systematically. For each finding, record the file path, line reference, and the specific evidence from Phase 1–2 that supports the finding — not general principles.

---

## Audit Domains

### 1. Software Engineering Best Practices

Check against the actual diff:
- **Error handling completeness:** Every error path that can reach a user or a caller either returns a typed error, logs structured context, or both. Look for silent `if err != nil { return }` without log/wrap.
- **Abstraction leakage:** Implementation details (DB models, internal structs, ORM types) not exposed through public API boundaries without an explicit mapping layer.
- **Layering violations:** In layered architectures, confirm that the import direction did not change — handlers do not call repositories directly, services do not embed HTTP concerns, etc.  
  `[REPO-HOOK]` Verify Go package import rules: handlers→services→repository→models. Run `go test -run TestNoDirectDatabase ./...` if applicable.
- **Duplication introduced:** New code that duplicates existing helpers, scopes, or services. If so, cite both the new code and the existing equivalent.
- **Complexity delta:** Cyclomatic complexity jumps (long switch/if chains, deeply nested callbacks) introduced without justification.
- **Mutation and side effects:** Functions that modify caller state as a hidden side effect without documentation.

### 2. Security and Threat Model

For each changed auth, permission, secret-handling, or data-access path:
- **New attack surface:** List every new HTTP endpoint, WebSocket event, or IPC channel created. Confirm each has authentication and authorization checks.
- **Input validation:** Every externally supplied input (query params, body fields, URL segments, headers) is validated before use. Note exact fields that are not validated.
- **Secret exposure:** No credentials, tokens, API keys, or internal reference IDs appear in: logs, error messages returned to clients, URL parameters, SSE/WebSocket streams, or response bodies unless they are the intended deliverable.
- **Injection risks:** SQL queries use parameterized statements; HTML rendering escapes user content; shell commands do not interpolate user input.
- **Auth boundary changes:** Any new route group, middleware bypass, or role check addition/removal must be called out explicitly.
- **Data ownership:** Multi-tenant data access always scopes queries to the authenticated user's ID. Look for missing `.Where("user_id = ?", userID)` or equivalent.  
  `[REPO-HOOK]` Verify GORM scopes (`OwnedBy`, `OwnedByID`) are used instead of raw `.Where()`.
- **Token/session hygiene:** Refresh tokens, PKCE verifiers, OIDC state/nonce are single-use, stored safely, and not logged.
- **Supply chain:** Any new direct dependency — check it is pinned to a reviewed version (exact tag or digest, not `latest`).

### 3. Documentation and Alignment Pass

- **Spec coverage:** Every acceptance criterion in the authorizing spec has a corresponding implementation. List criteria that have no evident code path.
- **Spec drift:** Any behaviour in the diff that contradicts or extends the spec without an ADR or noted amendment.  
  `[REPO-HOOK]` Flag violations of the constitution hierarchy (§0). A lower-ranked artefact that contradicts a higher-ranked one is a blocker.
- **README / docs currency:** Files in `docs/`, `README.md`, or equivalent that describe components changed by this work but were not updated in the diff.
- **In-code comments:** Any comment that is now wrong or misleading given the new behaviour.
- **Changelog / decision log:** If the project tracks decisions (ADR, decisions.md), confirm a record was written for material architectural choices made in this work.

### 4. Architecture and Contract Alignment

- **API contract drift:** Compare new/modified handler signatures, response shapes, and status codes against the OpenAPI spec or equivalent contract file. List any field added, removed, renamed, or type-changed without a corresponding spec update.  
  `[REPO-HOOK]` Run `go test -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI ./...` if applicable.
- **Schema migration safety:** Every new column is nullable or has a default safe for existing rows. No column rename or drop without a migration step. No SQLite `ALTER TABLE` that would cause a destructive rebuild.
- **Pydantic / TypeScript interface alignment:** Python Pydantic models, Go response structs, and TypeScript interfaces that represent the same resource must agree on field names and optionality.  
  `[REPO-HOOK]` Run `npm run type-check` and `npx vue-tsc --noEmit`.
- **Backward compatibility:** If an existing API contract or DB schema changed, identify consumers and confirm they handle the old and new shape, or confirm a coordinated cutover.
- **Architecture test regressions:** Confirm that automated architecture/contract tests pass if they exist.

### 5. Test Coverage and Regression Quality

- **Coverage of new paths:** For every new function/method/handler in the diff, list the test that exercises the happy path and the primary failure path. Flag missing tests.
- **Edge cases:** Null/zero/empty inputs, off-by-one boundaries, concurrent access (if goroutines or shared state were changed), and idempotency under retry.
- **Regression anchors:** Changes that removed or weakened existing test assertions — note what was asserted before and what is asserted now.
- **Test isolation:** Tests that rely on shared mutable state, real clocks, network, or filesystem without mocking — flag as fragile.
- **Test fidelity:** Tests that mock so much that they cannot catch the real failure mode being guarded against.
- **Integration coverage gap:** New cross-layer flows (handler → service → repo → DB) that have only unit-level mocks with no integration-level coverage.

### 6. Dependency, Supply Chain, Config, and Deployment

- **New dependencies:** For every entry added to `go.mod`, `package.json`, `pyproject.toml`, or equivalent:
  - Is it from a reputable source?
  - Is it pinned to an exact reviewed version?
  - Does the lock file (`go.sum`, `package-lock.json`, `uv.lock`) reflect the pin?
- **Existing dependency version bumps:** Confirm the bump was intentional and the lock file regenerated deterministically.
- **Toolchain version changes:** Any change to Go version, Node version, Python version, or Docker base image must update all matching pins (CI matrix, Dockerfiles, version files, Taskfile targets).
- **Config / environment changes:** New environment variables or settings keys added — confirm defaults are safe for existing deployments, documented, and validated on startup.
- **Dockerfile / container changes:** Non-root user preserved, port exposure unchanged, healthcheck present, multi-stage build not broken.
- **CI workflow changes:** No step removed that previously gated merges; no secret leaked into workflow logs; added steps run on the correct triggers.
- **Migration run order:** DB migrations must be additive or explicitly sequenced; destructive migrations require a rollback plan.

### 7. UX, Accessibility, and Localization

*(Skip or reduce scope if the changeset contains no frontend changes.)*

- **Mobile / PWA viewport:** New UI elements function at 375px width; touch targets ≥ 44×44px; service-worker cache invalidated where needed.
- **Design token compliance:** No hardcoded colors, radii, or spacing values — only tokens from `variables.css` (or equivalent).  
  `[REPO-HOOK]` Enforce global CSS classes (`.chip`, `.btn`, `.badge`) and typography scale from the project constitution.
- **Keyboard and screen-reader access:** Interactive elements reachable by Tab; form fields have labels; ARIA roles not misused.
- **No emoji in UI text or AI responses.**
- **String externalization:** User-facing strings not embedded in component logic in ways that would block future localization.
- **Error messaging:** Validation errors and server errors surface a user-legible message — not a raw stack trace or internal code.

### 8. Operational Readiness, Observability, and Backward Compatibility

- **Structured logging:** New code paths emit structured log entries (key=value or JSON) at appropriate levels (Info for lifecycle, Error for failures, Debug for diagnostic). No `fmt.Println` or `print()` in production paths.
- **Error codes and observability:** HTTP errors return consistent status codes. Background jobs record run metadata (started_at, duration, item count, status). Failures are observable without requiring a debugger.
- **Graceful degradation:** External service failures (AI provider, third-party APIs) produce a user-legible fallback, not a 500 with a raw error.
- **Backward-compatible defaults:** Feature flags, new settings, and config keys default to the behaviour that existing deployments already experience — opt-in, not opt-out.
- **Rollback safety:** If the change ships but needs to be reverted, what happens to data written by the new code? Confirm the answer is "safe" or document the rollback steps.
- **Health and readiness signals:** If a new external dependency was introduced, confirm it is reflected in the health check endpoint.

---

## Report Format

Produce a single structured report. Every finding **must** reference a specific file path (and line number where possible) and quote or describe the exact evidence from the diff or referenced document. Do not generalize.

```
## Post-Major-Work QC Audit — [Feature / PR / Branch]
**Audited by:** [Agent name]
**Date:** [ISO 8601]
**Changeset:** [git diff base..head, or PR number]
**Spec / design doc:** [path or URL]

### Scope Summary
[2–4 sentences: what changed, which layers, which domains are in scope]

### Artifact Checklist
- [ ] Diff read (all N changed files)
- [ ] Authorizing spec / PRD section read
- [ ] Relevant ADRs read
- [ ] CI workflow read
- [ ] [REPO-HOOK] Constitution and DoD checklist reviewed
- [ ] All 8 Audit Domains traversed

### Blockers  ← Must be resolved before merge / deploy
| ID | Domain | File : Line | Finding | Required Remediation |
|----|--------|-------------|---------|----------------------|
| B1 | Security | src/api/handlers/foo.go:42 | Missing userID scope on DB query | Add `.Where("user_id = ?", uid)` before `.Find()` |

### Follow-Ups  ← Recommended but not merge-blocking
| ID | Domain | File : Line | Finding | Suggested Remediation |
|----|--------|-------------|---------|----------------------|
| F1 | Test Coverage | src/api/services/bar.go:88 | No test for nil-repo error path | Add unit test case with mock returning `ErrNotFound` |

### Positive Observations
[Optional: note design decisions or test patterns that are worth reusing]

### Confidence Notes
[Note any area where evidence was ambiguous, files were inaccessible, or a domain was partially skipped and why]
```

**Severity rules:**
- **Blocker:** Security vulnerability with exploitable path, data-loss risk, spec requirement unimplemented, contract broken without backward compat, Quality Gate failure, constitution violation.  
  `[REPO-HOOK]` Any finding that would fail §17 Quality Gate or violate a constitution Principle is automatically a Blocker.
- **Follow-Up:** Gaps in test coverage, documentation currency, minor observability omissions, style/token violations without functional impact.

---

## Anti-Patterns

**Do not do these:**

1. **Checklist-from-memory audit** — Listing general best practices ("ensure proper error handling") without citing a specific file and line from the diff. Every finding must be grounded in observed evidence.

2. **Inventing facts** — Claiming a test is missing without first reading the test files; claiming a dependency is unpinned without reading the lock file; claiming an endpoint is unauthenticated without reading the middleware chain.

3. **Skipping the diff** — Starting with domain checklists before completing the Investigation Protocol. Phase 0–2 are not optional.

4. **Escalating style to blocker** — Naming a follow-up a blocker because it violates a preference rather than a security, correctness, or contract requirement.

5. **Collapsing all findings into prose** — Narrative descriptions of many issues in a paragraph. Every issue must appear as a row in the Blockers or Follow-Ups table with a file path.

6. **Re-auditing what CI already gates** — If `go vet`, `ruff`, or `vue-tsc` already runs in CI and passes, do not re-raise issues those tools would have caught. Focus on what CI cannot see: cross-cutting spec alignment, deployment implications, threat model, and emergent interactions.

7. **Scope creep into unrelated files** — The audit covers the changeset plus directly referenced dependencies. Do not audit the entire codebase.

8. **Hedging every finding** — "This might be an issue depending on context." If confidence is insufficient to make a clear call, note it in the Confidence Notes section and exclude it from the tables.

---

## Examples

### Invocation (this repo)

```
Run a post-major-work QC audit on the OIDC Phase 1–5 implementation.
Scope: git diff main..oidc-mvp
Spec: specs/335-oidc-login/spec.md
Prior decisions: .squad/decisions.md (OIDC sections)
Constitution: .specify/memory/constitution.md
```

### Invocation (generic repo)

```
Run a post-major-work QC audit on the payment-gateway integration.
Scope: PR #412 (files: src/payments/, src/api/routes/checkout.ts, migrations/20260720_add_payment_methods.sql)
Design doc: docs/design/payment-gateway.md
No governance model — skip REPO-HOOK steps.
```

### Finding that IS a Blocker

> **B1 | Security | src/api/handlers/coins.go:134**  
> `GetCoinByID` calls `coinRepo.FindByID(id)` without scoping to the authenticated user's ID. Any authenticated user can fetch any coin by guessing an integer ID.  
> **Remediation:** Replace with `coinRepo.FindByIDForUser(id, userID)` using the `OwnedByID` scope, consistent with `UpdateCoin` at line 198.

### Finding that IS NOT a Blocker (Follow-Up)

> **F1 | Test Coverage | src/api/services/payment_service.go:88**  
> `ChargeCard` has no test case for the `ErrProviderTimeout` path. Happy-path and `ErrCardDeclined` paths are covered.  
> **Suggested:** Add a unit test with a mock that returns `ErrProviderTimeout`; confirm the service wraps it as a 503 to the caller.
