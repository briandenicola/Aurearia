# Ancient Coins

> A self-hosted Progressive Web App for cataloging, analyzing, and sharing a personal ancient coin collection.

> **Note:** This application is 100% vibe coded. It exists for the author to learn and experiment with GitHub Copilot CLI.

## What is this?

Ancient Coins is a full-stack PWA for managing a private ancient coin collection end-to-end — catalog, valuation, AI-assisted discovery, social engagement, and offline-capable mobile access, all running on your own server. It is built for a single primary collector plus a small number of invited friends; it is **not** a marketplace, a multi-tenant SaaS, or a replacement for numismatic reference catalogs.

The product story — vision, personas, goals, non-goals, functional areas, constraints, success metrics, and open questions — lives in **[`docs/prd.md`](docs/prd.md)**. Per [Constitution §0](.specify/memory/constitution.md) the PRD is the product source of truth; this README is intentionally a thin navigation surface.

## Architecture

Three services, two containers in production:

```
Browser ──► Vue 3 SPA  ──► Go API (Gin, GORM, SQLite) ──► Python LangGraph Agent
            (PWA)          :8080                          :8081 (stateless, SSE)
```

| Layer    | Tech                                  | Path         |
| -------- | ------------------------------------- | ------------ |
| Backend  | Go 1.26 (Gin), GORM, pure-Go SQLite   | `src/api/`   |
| Frontend | Vue 3, TypeScript, Vite, Pinia (PWA)  | `src/web/`   |
| Agent    | Python 3.12, FastAPI, LangGraph       | `src/agent/` |

The Go API serves both REST (`/api/*`) and the compiled Vue SPA; it proxies AI requests (including SSE streams) to the Python agent. The Python service is stateless — all configuration (provider, keys, prompts, user context) is passed per-request.

Full details in **[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)**; the binding rationale is captured in **[ADR 0002 — Three-Service Architecture](docs/adr/0002-three-service-architecture.md)**.

## Quick Start

**Prerequisites:** [Go 1.26+](https://go.dev/dl/), [Node.js 20+](https://nodejs.org/), [Task](https://taskfile.dev/) (optional), [Docker](https://docs.docker.com/get-docker/) (optional), [Ollama](https://ollama.ai/) (optional — for local AI).

```sh
git clone <repo-url> && cd coin-collection-app
task init        # generates .env with a random JWT secret
task up          # API (:8080) + web (:5173)
task up-all      # API + web + Python agent (:8081)
```

The first registered user becomes the admin and can configure AI providers from the Admin page. For a guided walkthrough see **[`docs/getting-started.md`](docs/getting-started.md)**; for production deployment see **[`docs/deployment.md`](docs/deployment.md)**.

### Running tests

```sh
task test            # Go architecture + unit tests
task test-agent      # Python agent tests
( cd src/web && npm run type-check )
```

Run `task --list` to see all targets.

## Documentation

| Area                         | Path                                                       |
| ---------------------------- | ---------------------------------------------------------- |
| **Product (source of truth)**| [`docs/prd.md`](docs/prd.md)                               |
| Architecture                 | [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)             |
| API reference                | [`docs/api-reference.md`](docs/api-reference.md)           |
| Authentication               | [`docs/authentication.md`](docs/authentication.md)         |
| Features (per-area detail)   | [`docs/features.md`](docs/features.md)                     |
| Deployment                   | [`docs/deployment.md`](docs/deployment.md)                 |
| Getting started              | [`docs/getting-started.md`](docs/getting-started.md)       |
| PWA guide                    | [`docs/pwa-guide.md`](docs/pwa-guide.md)                   |
| Social feature spec          | [`docs/social-feature.md`](docs/social-feature.md)         |
| Security analysis            | [`docs/security-analysis.md`](docs/security-analysis.md)   |
| Software design doc          | [`docs/SDD.md`](docs/SDD.md)                               |
| Architecture decision records| [`docs/adr/`](docs/adr/)                                   |
| Feature specs (in-flight)    | [`specs/`](specs/)                                         |
| Feature backlog (F00N cards) | [`specs/_backlog/`](specs/_backlog/)                       |
| Changelog                    | [`docs/CHANGELOG.md`](docs/CHANGELOG.md)                   |

The design system (tokens, typography, chip/button hierarchy) is documented in the [Copilot Instructions](.github/copilot-instructions.md#design-system) and locked in by **[ADR 0004 — Design Token System](docs/adr/0004-design-token-system.md)**.

## Governance

- **Constitution:** [`.specify/memory/constitution.md`](.specify/memory/constitution.md) (v2.0.0) — non-negotiable contract for how this project is built. §0 defines the Hierarchy of Authority.
- **Spec workflow:** features ship through `specs/NNN-*/{spec,plan,tasks}.md` per the templates in [`.specify/templates/`](.specify/templates/).
- **Decision records:** [`docs/adr/`](docs/adr/) (Michael Nygard format, established by **[ADR 0001](docs/adr/0001-record-architecture-decisions.md)**).
- **Project decisions ledger:** [`.squad/decisions.md`](.squad/decisions.md); proposals land in [`.squad/decisions/inbox/`](.squad/decisions/inbox/).
- **AI team (Squad):** [`.squad/team.md`](.squad/team.md) and per-agent charters under [`.squad/agents/`](.squad/agents/).
- **Quality Gate & Definition of Done:** Constitution §17 and §21 — enforced on every PR.
- **Contributing:** [`CONTRIBUTING.md`](CONTRIBUTING.md).
- **Security:** [`SECURITY.md`](SECURITY.md).

## License

[MIT](LICENSE).
