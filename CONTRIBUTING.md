# Contributing

Guidelines for contributing to the Aurearia application.

Read the constitution and applicable accepted ADRs first. This consumer prepares
Proposed ADR 0024 / 5.0.0. Until §22 approval/merge, effective 4.0.0 (ADR 0019,
PR #734) remains governing from the constitution's linked baseline, including
existing review/audit obligations. No #827 gate or historical block is waived.
The following lane/recording changes activate only with that amendment.

## Getting Started

```bash
# Clone and setup
git clone <repo-url>
task init        # generates .env with JWT secret

# Run locally (two terminals, or use task)
task up          # starts both API and web dev servers
```

| Service | URL | Source |
|---|---|---|
| API | `http://localhost:8080/api/*` | `src/api/` |
| Web | `http://localhost:5173` | `src/web/` |
| Swagger | `http://localhost:8080/swagger/index.html` | auto-generated |

## Branch Strategy

| Branch | Purpose |
|---|---|
| `main` | Production-ready code |
| `beta` | Ordinary feature/bug development and integration |
| Other branches/worktrees | Only when explicitly authorized |

Select work by the owner-approved issue/spec/process plan, not the newest spec.
Use constitution §18.1: routine work needs brief scope, appropriate regression
and CI, with no mandatory independent review; features need short criteria,
workflow/error tests and one independent review. Changed security/privacy/data-
safety, destructive migration, service or delivery boundaries add applicable
security/compatibility/recovery proof. Merely touching existing contracts does
not create a new boundary. ADRs are for material architecture.
One issue/PR owns scope, criteria, evidence, review and outstanding work; specs
are optional planning aids. No mandatory inbox/log/history document set.

For SpecKit on `beta`, set `$env:SPECIFY_FEATURE` to the exact selected spec
directory name in the same process as each script invocation. Do not run the
branch-creating script implicitly. Follow `.github/agents/speckit.specify.agent.md`
for its existing dry-run/manual-file preparation path.

Implemented, verified, accepted, and released are distinct states. Preserve
review restrictions and record the tested/reviewed commit or tree. Green CI
does not authorize a beta-to-main merge or release.

## Code Architecture

Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) before making API changes. The key rules:

1. **Only the composition root (`main.go` + `deps.go`) imports the `database` package** -- everything else uses dependency injection
2. **Handlers are thin** -- parse request, call service/repo, return response
3. **Services hold business logic** -- HTTP-agnostic, orchestrate repos
4. **Repositories own all DB access** -- every GORM query lives here
5. **Multi-step writes use transactions**
6. **Never leak internal errors to clients**

These rules are enforced by `src/api/architecture_test.go`.

## Adding a New Feature

### New API Endpoint

1. **Model** -- Add/update structs in `src/api/models/`
2. **Repository** -- Add DB methods to the appropriate `src/api/repository/*_repository.go`
3. **Service** (if business logic needed) -- Add to `src/api/services/*_service.go`
4. **Handler** -- Add thin HTTP handler in `src/api/handlers/`
5. **Wire** -- Register in `src/api/main.go` (create repo/service, pass to handler constructor)
6. **Swagger** -- Add annotations, then run `task openapi` and `task check:openapi` with the pinned generator already prepared
7. **Test** -- Run `go test ./...` from `src/api/` to verify architecture rules pass

### New Vue Page/Component

1. Add component in `src/web/src/pages/` or `src/web/src/components/`
2. Add route in `src/web/src/router/index.ts` if it's a page
3. Run applicable frontend gates from `docs/testing.md` §6: zero-warning lint,
   strict type-check, full tests, and build, plus affected browser workflows.

## Code Style

### Go (API)
- Follow standard Go conventions (`gofmt`, `go vet`)
- Use constructor injection for dependencies
- Return typed errors from services (sentinel errors like `ErrNotFound`)
- Log internal errors server-side, return generic messages to clients

### TypeScript (Web)
- Use optional chaining (`?.`) and nullish coalescing (`??`) on array access
- Docker builds use stricter TS checking than local -- always verify with `vue-tsc`
- No emojis in UI, prompts, or AI responses

## Build and Test

Use [docs/testing.md §6](docs/testing.md#6-running-tests-locally-vs-ci) for the
shared Taskfile completion gates, locked environment, Windows invocation, and
CI-only evidence. From the root, run applicable `task check:go`,
`task check:web`, `task check:agent`, `task check:openapi` and
`task check:delivery`; `task check` runs all base gates. OpenAPI generation
writes files. Dependency setup uses separate authorized `setup:*` targets.
Obtain required authorization before setup, builds, or containers; missing
execution is incomplete rather than passed. Race, security, browser,
compatibility and release checks remain additional where applicable.

Native Copilot domain instructions and skills are documented in
[the integration guide](docs/agentic-native-integration.md). Start a fresh client
session after instruction changes; static fixtures do not prove runtime
discovery. Squad is optional. Required independent review uses the read/search-only
profile with supplied diffs/evidence and an owner-approved bounded lease.

Keep pure-document review proportional, but exercise executable policy/prompts
and workflows with relevant behavior/fixtures. Lane-required independent review
and actual owner release approval cannot be replaced by task checkboxes.
Focused local evidence plus applicable exact-candidate CI may reuse unchanged
proof. Broad audits are only for major architecture/owner-designated major
releases; existing audit obligations remain.

Use the PR template's canonical completion record to bind each criterion and
affected workflow to its result, evidence link and tested commit/tree. Handoffs
link it and record only their delta. Stale reviews and unresolved blocks cannot
support acceptance. See [acceptance and release controls](docs/agentic-acceptance-controls.md)
for the proposed main protections, exact-candidate approval and activation status.
Beta validation/publishing is not main release or local-server deployment approval.

## Commit Messages

Use conventional commits:

```
feat: add coin grading service
fix: handle null purchase price in stats
refactor: extract social access rules to service layer
```

Include the co-author trailer when using Copilot:
```
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
```

Cite the governing Principle/section in commits and PRs. Commit/push only when
authorized, stage only approved paths, and preserve unrelated work. A handoff
does not automatically commit or stash the worktree.
