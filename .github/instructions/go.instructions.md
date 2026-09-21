---
applyTo: "src/api/**,Dockerfile"
---

# go guidance

Apply the [constitution](../../.specify/memory/constitution.md) and selected approved work. These scoped instructions implement, not replace, that authority.

### Go API — Layered Architecture

```
Handler → Service → Repository → Database
```

The full rule set lives in constitution **Principle I (Clear Layered Architecture)** and is enforced by `architecture_test.go` per **Principle IX**. Quick reference table below for the import rules.

**Package import rules:**

| Package | May import |
|---|---|
| `handlers/` | `services/`, `repository/`, `models/` |
| `services/` | `repository/`, `models/` |
| `repository/` | `models/`, `gorm.io/gorm` |
| `models/` | Standard library only |
| `middleware/` | `models/`, `gorm.io/gorm` |

**DI wiring in `main.go`:** `config.Load()` → `database.Connect()` → construct repos → construct services → construct handlers → register routes. Three route groups: `api` (public auth), `protected` (JWT required), `admin` (JWT + admin role).

### Go
- Constructor injection for all dependencies (`NewXxxHandler(repo, service)` pattern)
- Sentinel errors in services (e.g., `ErrNotFound`, `ErrInvalidCredentials`)
- Use GORM scopes from `repository/scopes.go` (`OwnedBy`, `OwnedByID`, `ActiveCollection`, `PublicCoins`, `ByCoinID`) instead of repeating `.Where()` clauses
- Swagger annotations on all public handler methods
- Settings use key-value `AppSetting` model; constants and defaults live in `services/settings_service.go`

### Adding a New API Feature

1. Model in `src/api/models/` → add to `AutoMigrate` in `database/database.go`
2. Repository in `src/api/repository/*_repository.go`
3. Service (if business logic needed) in `src/api/services/*_service.go`
4. Thin handler in `src/api/handlers/` with `NewXxxHandler()` constructor
5. Wire in `src/api/main.go` (create repo → service → handler, register routes under correct group)
6. Run `go test ./...` to verify architecture rules pass

## Completion

task check:go; task check:openapi when API metadata changes; runner-only race/security checks remain additional. See [testing](../../docs/testing.md#6-running-tests-locally-vs-ci).
Setup/installations require separate authorization; missing execution is incomplete.
