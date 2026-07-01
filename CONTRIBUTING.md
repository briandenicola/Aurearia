# Contributing

Guidelines for contributing to the Aurearia application.

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
| `beta` | Pre-release testing |
| `feature/*` | Feature development |

## Code Architecture

Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) before making API changes. The key rules:

1. **Only `main.go` imports the `database` package** -- everything else uses dependency injection
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
6. **Swagger** -- Add annotations to handler, run `swag init` if using swag CLI
7. **Test** -- Run `go test ./...` from `src/api/` to verify architecture rules pass

### New Vue Page/Component

1. Add component in `src/web/src/pages/` or `src/web/src/components/`
2. Add route in `src/web/src/router/index.ts` if it's a page
3. Run type check: `cd src/web && npx vue-tsc --noEmit`

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

```bash
# Go API
cd src/api
go build ./...                    # compile
go vet ./...                      # lint
go test -v ./...                  # architecture tests

# Vue frontend
cd src/web
npm run build                     # production build
npx vue-tsc --noEmit              # type check

# Python agent
cd src/agent
ruff check app/ tests/            # lint
pytest tests/ -q                  # tests

# Docker
task docker-build                 # full container image
```

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
