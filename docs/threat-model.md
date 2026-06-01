# Threat Model

> The live inventory of known security threats and security-review findings for Ancient Coins. This document keeps the original **backend / frontend / supply-chain** split because those are the operational owners and remediation surfaces in this repo.

## How to add a new threat finding

1. Add the finding to the correct surface area below (`B-*`, `F-*`, or `SC-*`).
2. Record **severity**, **status**, **location**, **description**, and **recommended remediation** in the same edit.
3. Use `Open`, `Mitigated`, or `Accepted` as the status. If prioritization matters, append it in parentheses (for example `Open (P1)`).
4. Link any architecture or policy change needed for remediation to an ADR or to [security-principles.md](security-principles.md).
5. If the finding becomes an active incident, manage the operational response in [incident-response.md](incident-response.md) and keep this document focused on the durable risk record.

## Status summary

| Domain | Findings | Mitigated | Open | Accepted |
|---|---:|---:|---:|---:|
| Backend API | 10 | 9 | 1 | 0 |
| Frontend | 8 | 3 | 4 | 1 |
| Supply chain & infrastructure | 7 | 2 | 5 | 0 |
| **Total** | **25 enumerated** | **14** | **10** | **1** |

**Last reconciliation:** 2026-06-01. Recent mitigations include B-2 (SQL injection whitelist), B-6/B-7/B-8 (request size limits, WebAuthn TTL and origin validation), B-10 (external tool server capability scoping, two-phase writes, kill switch, per-key rate limiting, tenant isolation), F-1/F-2/F-4 (DOMPurify sanitization for XSS), SC-1/SC-2 (GitHub Actions SHA pins, Taskfile secret generation). Status table reflects current code state.

## Backend API findings

| ID | Severity | Status | Location | Description | Recommended remediation |
|---|---|---|---|---|---|
| B-1 | Critical | Mitigated | `src/api/main.go`, `src/api/config/config.go` | CORS previously reflected arbitrary origins while allowing credentials. | Keep `CORS_ORIGINS` allowlisted and review any new origin-handling logic against Principle XI. |
| B-2 | Critical | Mitigated | `src/api/handlers/analysis.go` | A user-controlled column name was previously concatenated into a query, but is now protected by an explicit whitelist map in `DeleteAnalysis()` (lines 229–238) and switch-based validation in `Analyze()` (lines 175–185). | Maintain the whitelist pattern for any future dynamic column handling. |
| B-3 | High | Mitigated | `src/api/config/config.go` | The application previously allowed a weak default JWT secret. | Preserve the startup fail-fast and minimum-length checks; never weaken them for convenience. |
| B-4 | High | Mitigated | `src/api/handlers/images.go` | Uploads now enforce an extension allowlist, but the original finding still has a residual MIME-validation gap. | Add magic-byte validation so uploads are checked by content, not extension alone. |
| B-5 | High | Mitigated | `src/api/main.go` | Auth endpoints previously allowed unlimited brute-force attempts. | Keep auth rate limiting in place and extend it to new auth-adjacent endpoints when added. |
| B-6 | Medium | Mitigated | `src/api/main.go` (line ~130: `r.MaxMultipartMemory = middleware.DefaultMultipartMemoryBytes`), middleware | Request bodies and multipart uploads now have explicit size caps enforced by Gin. | Maintain the configured memory limit; document enforcement in deployment documentation. See issue #201. |
| B-7 | Medium | Mitigated | `src/api/handlers/webauthn.go` (lines ~20–30: `const webauthnSessionTTL = 5 * time.Minute`, cleanup logic) | In-memory WebAuthn ceremony sessions now expire after 5 minutes and are automatically cleaned up. | Maintain periodic cleanup; adjust TTL if UX feedback warrants it. See issue #202. |
| B-8 | Medium | Mitigated | `src/api/handlers/webauthn.go` | WebAuthn origin validation now restricts to configured RP origins; dynamic trust from request headers removed. | Ensure RP origin configuration is correct at deployment time; document in security-principles.md. See issue #202. |
| B-9 | Low | Open (P3) | `src/api/handlers/numista.go` | Some error responses expose more internal detail than clients need. | Return generic client-facing errors and keep specifics in logs only. [#163](https://github.com/briandenicola/coin-collection-app/issues/163) |
| B-10 | High | Mitigated | `src/api/handlers/external_tools.go`, `src/api/middleware/external_tools_gate.go`, `src/api/middleware/capability.go`, `src/api/main.go` (lines 469–506), `src/api/models/api_key.go` | External tool server (`/api/v1/tools/*`) exposes write operations over a public HTTP surface, introducing risk of unauthorized or accidental writes. Mitigations: (1) Default-off admin kill switch (`ExternalToolServerEnabled`), (2) API key capability scopes (`read` default, `read,write` opt-in), (3) Two-phase proposal+confirm flow (no auto-writes), (4) Field allowlist (identity fields rejected), (5) Per-key rate limiting (50 req/min, stricter than in-app), (6) Journaled audit trail with source `external_tool_server` and API key id/name/capabilities, (7) Server-side tenant isolation (user identity derived from key, no cross-user access). See [external-tool-server.md](external-tool-server.md) for full security model. | Maintain the layered defenses (kill switch, least-privilege scopes, confirm gate, allowlist, rate limits, journaling). Monitor audit logs for unexpected external commits. Periodically review API key scopes and revoke unused keys. Issue #218. |

## Frontend findings

| ID | Severity | Status | Location | Description | Recommended remediation |
|---|---|---|---|---|---|
| F-1 | Critical | Mitigated | `src/web/src/components/coin/CoinAIAnalysis.vue` | AI analysis Markdown was rendered unsanitized, but is now passed through `DOMPurify.sanitize()` before injection (lines 33–35). | Keep DOMPurify pinned to a stable version and monitor upstream security advisories. |
| F-2 | Critical | Mitigated | `src/web/src/composables/useCoinSearchChat.ts` | Chat messages were transformed and injected without sanitization, but `formatMessage()` now sanitizes all HTML via `DOMPurify.sanitize()` before rendering in `CoinSearchChat.vue` line 31. | Ensure all HTML rendering paths go through the sanitization function. |
| F-3 | Critical | Open (P3) | `src/web/src/stores/auth.ts` | JWT and refresh tokens live in `localStorage`, which turns any XSS into token theft. | Short term: eliminate XSS sinks (F-1/F-2/F-4 mitigations complete the path). Long term: evaluate `HttpOnly` cookie transport with an ADR. [#163](https://github.com/briandenicola/coin-collection-app/issues/163) |
| F-4 | High | Mitigated | `src/web/package.json` (DOMPurify ^3.4.1), `src/web/src/composables/useCoinSearchChat.ts`, `src/web/src/components/coin/CoinAIAnalysis.vue`, `src/web/src/components/coin/FeaturedCoinModal.vue` | Rich-text rendering is now backed by DOMPurify (`@types/dompurify` ^3.2.0) and sanitized at all HTML injection points. | Keep DOMPurify pinned and monitor for updates; audit any new HTML rendering paths to confirm sanitization. See security-principles.md Principle XI. |
| F-5 | High | Open (P3) | `src/web/src/api/client.ts` | Refresh tokens travel in a JSON body rather than an `HttpOnly` cookie-based flow. | Revisit transport if auth is redesigned; document the trade-off in ADRs. [#163](https://github.com/briandenicola/coin-collection-app/issues/163) |
| F-6 | Medium | Open (P2) | frontend auth responses (Go API: `src/api/handlers/auth.go`; Vue client proxies responses) | Sensitive responses are not explicitly marked `Cache-Control: no-store`. | Set no-store headers on auth and token-bearing responses at the Go API level (handlers) and confirm Vue client respects them. [#163](https://github.com/briandenicola/coin-collection-app/issues/163) |
| F-7 | Medium | Open (P2) | `src/web/src/api/client.ts` (line 385: `/auth/webauthn/login/finish?username=${...}`), `src/api/handlers/webauthn.go` (line ~480: `username` param in URL) | WebAuthn login finish sends the username in the query string, leaking it into histories and logs. | Move username from query param to request body in both client (client.ts) and handler (webauthn.go). [#163](https://github.com/briandenicola/coin-collection-app/issues/163) |
| F-8 | Low | Accepted (platform limitation) | `src/web/src/pages/LoginPage.vue` | Password state cannot be meaningfully scrubbed from browser memory after use in a reliable, portable way. | Accept the platform limit; keep lifetime short in reactive state and avoid unnecessary copies. |

## Supply chain and infrastructure findings

| ID | Severity | Status | Location | Description | Recommended remediation |
|---|---|---|---|---|---|
| SC-1 | Critical | Mitigated | `.github/workflows/docker-publish.yml`, `.github/workflows/docker-publish-beta.yml` (all `uses:` pinned to commit SHAs, verified lines 20, 23, 26, 33, 46, 65, 68, 71, 78, 87) | GitHub Actions are now pinned to immutable commit SHAs, eliminating supply-chain drift risk from action maintainer compromise. | Establish a quarterly audit cadence to review for action updates (check action repos for security advisories). See issue #204 for implementation details. |
| SC-2 | Critical | Mitigated | `Taskfile.yml`, `src/api/config/config.go` | JWT secret was previously hardcoded, but is now injected via environment variable at runtime. The `gen-env` task generates a random secret and stores it in a `.env` file (Taskfile.yml lines 143–145). Config enforces minimum 32-character length and fails fast if unset (config.go lines 21–33). | Maintain the env-based pattern; document that production deployments must set `JWT_SECRET` in CI/CD secrets before container start. |
| SC-3 | Critical | Open (P2) | `src/web/package.json` (@imgly/background-removal dependency) | `@imgly/background-removal` downloads large runtime models from external CDNs without integrity verification. | Evaluate bundling, server-side processing, or a lower-risk alternative. [#163](https://github.com/briandenicola/coin-collection-app/issues/163) |
| SC-4 | High | Open (P1) | `src/api/go.mod` | `golang.org/x/*` dependencies were called out for lagging current versions in the original review. | Re-check with `govulncheck` / dependency review and upgrade deliberately. [#163](https://github.com/briandenicola/coin-collection-app/issues/163) |
| SC-5 | High | Open (P2) | GitHub repository settings | Branch protection is not documented as enforced for `main`/`beta`. | Require PR reviews, required checks, and restricted direct pushes in GitHub Settings → Branches. [#163](https://github.com/briandenicola/coin-collection-app/issues/163) |
| SC-6 | Medium | Open (P2) | `Dockerfile` (lines 2, 15, 23: base image tags) | Base images (`node:24-alpine`, `golang:1.26-alpine`, `alpine:3.21`) use pinned minor versions but not reproducible digests (full image SHA). | Pin each base image to a specific digest (e.g., `alpine:3.21@sha256:...`) for production builds. [#163](https://github.com/briandenicola/coin-collection-app/issues/163) |
| SC-7 | Medium | Open (P2) | `Dockerfile` (line 33: ENTRYPOINT runs as root, no USER directive) | Final application containers run as root by default, widening the attack surface if image is compromised. | Create a non-root build user (e.g., `RUN addgroup -g 1000 app && adduser -D -u 1000 -G app app`) and switch before ENTRYPOINT. [#163](https://github.com/briandenicola/coin-collection-app/issues/163) |

## Related documents

- [security-principles.md](security-principles.md)
- [incident-response.md](incident-response.md)
- [authentication.md](authentication.md)
- [SECURITY.md](../SECURITY.md)
