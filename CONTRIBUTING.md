# Contributing

Guidelines for contributing to the Aurearia application.

Read the [constitution](.specify/memory/constitution.md) first: Principles I–VI
are the engineering rules, and its Working Rules cover how to deliver a change.

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

Work on `beta`. Each change needs tests for what it changes (a failing test
first for a bug fix) and a short PR description using the PR template. Specs
under `specs/` are optional planning aids. ADRs are only for hard-to-reverse
decisions.

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

Use [docs/testing.md §6](docs/testing.md#6-running-tests-locally-vs-ci) for
commands. From the root, run `task check:go`, `task check:web` or
`task check:agent` for the areas you changed (`task check` runs them all), and
`task check:openapi` after changing Go handlers. OpenAPI generation writes
files. Dependency setup uses the separate `setup:*` targets.

Scoped agent instructions live in `.github/instructions/` and skills in
`.github/skills/`. Start a fresh agent session after changing them.

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

Stage only the files your change needs and leave unrelated work alone.
