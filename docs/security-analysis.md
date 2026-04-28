# Security Analysis — Ancient Coins Application

**Date:** June 2025
**Scope:** Full-stack analysis covering Go/Gin API, Vue 3/TypeScript frontend, Docker infrastructure, CI/CD pipelines, and supply chain dependencies.

---

## Executive Summary

This analysis identified **25 findings** across three domains: backend API, frontend application, and supply chain/infrastructure. The most critical issues involve CORS misconfiguration, XSS via unsanitized HTML rendering, SQL injection risk, and supply chain concerns with unpinned CI/CD actions.

| Severity | Count | Fixed |
|----------|-------|-------|
| 🔴 CRITICAL | 8 | 1 |
| 🟠 HIGH | 8 | 3 |
| 🟡 MEDIUM | 7 | 0 |
| 🟢 LOW | 2 | 0 |

---

## Table of Contents

1. [Backend API Findings](#1-backend-api-findings)
2. [Frontend Findings](#2-frontend-findings)
3. [Supply Chain & Infrastructure](#3-supply-chain--infrastructure)
4. [Positive Security Practices](#4-positive-security-practices)
5. [Remediation Priority](#5-remediation-priority)

---

## 1. Backend API Findings

### ✅ B-1: CORS Accepts All Origins with Credentials — FIXED

**Severity:** CRITICAL → **FIXED**
**File:** `src/api/main.go` (lines 57–80), `src/api/config/config.go` (lines 48–65)

~~The CORS middleware reflects any `Origin` header back with `Access-Control-Allow-Credentials: true`.~~

**Fix:** CORS now uses `AllowedOrigins()` from config. The `CORS_ORIGINS` environment variable accepts a comma-separated allowlist; when unset, it falls back to WebAuthn origins plus `localhost:5173` and `localhost:8080`. Only listed origins receive `Access-Control-Allow-Origin`.

---

### 🔴 B-2: SQL Injection via Column Name Concatenation

**Severity:** CRITICAL (but limited exploitability)
**File:** `src/api/handlers/analysis.go` (line 176)

User-supplied column name is concatenated directly into a SQL query string without parameterization. While GORM provides some protection and the input comes from an authenticated user, this is a classic injection vector.

**Recommendation:** Validate column names against a whitelist of allowed columns before use.

---

### ✅ B-3: Weak Default JWT Secret — FIXED

**Severity:** HIGH → **FIXED**
**File:** `src/api/config/config.go` (lines 21–34)

~~The JWT secret has a weak default value.~~

**Fix:** In release mode (`GIN_MODE=release`), the app calls `log.Fatal` if `JWT_SECRET` is missing or set to the dev default. A minimum length of 32 characters is enforced unconditionally (dev and production). In dev mode a warning is logged when the default secret is used.

---

### ✅ B-4: No File Extension Validation on Uploads — FIXED

**Severity:** HIGH → **FIXED**
**File:** `src/api/handlers/images.go` (lines 67–73)

~~Image uploads accept any file type without validating the extension or MIME type.~~

**Fix:** Uploads are now validated against an allowlist of extensions: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`. Files with other extensions are rejected with a `400 Bad Request`.

**Remaining gap:** MIME-type validation via magic bytes is not yet implemented — the check is extension-only.

---

### ✅ B-5: No Rate Limiting on Auth Endpoints — FIXED

**Severity:** HIGH → **FIXED**
**File:** `src/api/main.go` (lines 122–134)

~~Login, registration, and token refresh endpoints have no rate limiting.~~

**Fix:** A `middleware.RateLimit(10, 1*time.Minute)` limiter is applied to registration, login, token refresh, and WebAuthn login begin/finish routes. This limits each IP to 10 requests per minute on auth endpoints.

---

### 🟡 B-6: No Request Body Size Limits

**Severity:** MEDIUM
**File:** `src/api/main.go`

No `MaxMultipartMemory` or body size limit is configured. An attacker could send extremely large payloads to exhaust server memory.

**Recommendation:** Set `r.MaxMultipartMemory = 10 << 20` (10 MB) and add middleware to limit JSON body sizes.

---

### 🟡 B-7: WebAuthn Sessions Never Expire (Memory Leak)

**Severity:** MEDIUM
**File:** `src/api/handlers/webauthn.go`

WebAuthn challenge sessions are stored in an in-memory map without TTL. Abandoned registration/login flows accumulate indefinitely, eventually exhausting memory.

**Recommendation:** Add a TTL (5 minutes) and periodic cleanup goroutine for the session store.

---

### 🟡 B-8: WebAuthn Origin Validation Too Permissive

**Severity:** MEDIUM
**File:** `src/api/handlers/webauthn.go` (lines 85–129)

The `getWebAuthnForRequest()` function dynamically adds the request's `Origin` header to the allowed origins list. This effectively bypasses origin validation — any origin that sends a request is automatically trusted.

**Recommendation:** Require all valid origins to be listed in `WEBAUTHN_ORIGIN` environment variable. Remove dynamic origin addition.

---

### 🟢 B-9: Information Disclosure via Error Messages

**Severity:** LOW
**File:** `src/api/handlers/numista.go` (line 74)

Error responses include internal details that could help an attacker understand the system architecture.

**Recommendation:** Return generic error messages to clients; log detailed errors server-side only.

---

## 2. Frontend Findings

### 🔴 F-1: XSS via Unsanitized v-html with Markdown

**Severity:** CRITICAL
**File:** `src/web/src/pages/CoinDetailPage.vue` (lines 243, 251, 255)

AI analysis content is rendered via `markdown-it` and injected with `v-html` without any HTML sanitization. `markdown-it` does **not** sanitize HTML by default — it passes through `<script>`, `<img onerror>`, and other dangerous elements.

```vue
<div class="ai-content" v-html="renderedObverse"></div>
```

**Risk:** If the AI backend or a compromised data source returns malicious content, arbitrary JavaScript executes in the user's browser with full access to `localStorage` (including JWT tokens).

**Recommendation:** Install `dompurify` and sanitize all `v-html` output:
```typescript
import DOMPurify from 'dompurify'
const rendered = DOMPurify.sanitize(md.render(content), {
  ALLOWED_TAGS: ['h1','h2','h3','p','ul','ol','li','strong','em','br','code']
})
```

---

### 🔴 F-2: XSS in Chat Component

**Severity:** CRITICAL
**File:** `src/web/src/components/CoinSearchChat.vue` (line 40)

Chat messages use `v-html` with a `formatMessage()` function that provides zero HTML sanitization — it only converts `**bold**` and `\n` to HTML tags.

```typescript
function formatMessage(text: string): string {
  return text
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\n/g, '<br>')
}
```

**Risk:** Identical to F-1. Injected script can steal tokens from `localStorage`.

**Recommendation:** Apply `DOMPurify.sanitize()` to the output of `formatMessage()`.

---

### 🔴 F-3: JWT Tokens Stored in localStorage

**Severity:** CRITICAL (in combination with F-1/F-2)
**File:** `src/web/src/stores/auth.ts` (lines 7, 16–18)

JWT and refresh tokens are stored in `localStorage`, which is accessible to any JavaScript running on the page. Combined with the XSS vulnerabilities above, an attacker can steal tokens and impersonate the user indefinitely.

**Risk:** Token theft via XSS → full account takeover.

**Recommendation (short-term):** Fix XSS vulnerabilities (F-1, F-2) to eliminate the primary attack vector.

**Recommendation (long-term):** Migrate to `HttpOnly, Secure, SameSite=Strict` cookies set by the backend. This makes tokens inaccessible to JavaScript entirely.

---

### 🟠 F-4: Missing DOMPurify Dependency

**Severity:** HIGH
**File:** `src/web/package.json`

No HTML sanitization library is installed. The application has multiple `v-html` injection points with no defense.

**Recommendation:** `npm install dompurify && npm install -D @types/dompurify`

---

### 🟠 F-5: Refresh Token Sent in Request Body

**Severity:** HIGH
**File:** `src/web/src/api/client.ts` (line 56)

The refresh token is sent as a JSON body parameter. While encrypted by HTTPS in transit, this pattern is less secure than using `HttpOnly` cookies, which would be sent automatically.

---

### 🟡 F-6: No Cache Control on Sensitive Responses

**Severity:** MEDIUM

The application does not set `Cache-Control: no-store` headers on authentication responses. Browsers may cache JWT tokens or API keys, exposing them on shared devices.

---

### 🟡 F-7: WebAuthn Username in Query String

**Severity:** MEDIUM
**File:** `src/web/src/api/client.ts` (line 335)

Username is passed as a URL query parameter in WebAuthn login requests. Query strings appear in browser history, proxy logs, and server access logs.

```typescript
return api.post(`/auth/webauthn/login/finish?username=${encodeURIComponent(username)}`, body)
```

**Recommendation:** Move username to the request body.

---

### 🟢 F-8: Password Not Cleared from Memory After Login

**Severity:** LOW
**File:** `src/web/src/pages/LoginPage.vue` (lines 75–87)

Password remains in Vue reactive state after login. This is a JavaScript platform limitation — secure memory clearing is not reliably possible.

---

## 3. Supply Chain & Infrastructure

### 🔴 SC-1: Unpinned GitHub Actions Versions

**Severity:** CRITICAL
**Files:** `.github/workflows/docker-publish.yml`, `.github/workflows/docker-publish-beta.yml`

All GitHub Actions use mutable version tags (`@v4`, `@v3`, `@v5`) instead of commit SHAs. If an action maintainer's account is compromised, malicious code could be injected into CI/CD pipelines.

```yaml
uses: actions/checkout@v4           # ❌ Mutable tag
uses: docker/build-push-action@v6   # ❌ Mutable tag
```

**Recommendation:** Pin to full commit SHAs:
```yaml
uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # v4.2.2
```

---

### 🔴 SC-2: Hardcoded Dev Secret in Taskfile.yml

**Severity:** CRITICAL
**File:** `Taskfile.yml` (line 37)

A JWT secret is hardcoded in a tracked file:
```yaml
JWT_SECRET: dev-secret-key-change-in-production-min32chars
```

This secret is visible to anyone with repo access and persists in git history forever.

**Recommendation:**
1. Remove the hardcoded secret from `Taskfile.yml`
2. Use `dotenv: ['.env.local']` (already gitignored)
3. Consider rewriting git history with `git filter-repo`

---

### 🔴 SC-3: @imgly/background-removal Runtime CDN Downloads

**Severity:** CRITICAL (supply chain)
**File:** `src/web/package.json` (line 14)

This library downloads ONNX ML models from external CDNs at runtime (50–300MB). These downloads have no integrity verification (no SRI hashes). A CDN compromise could serve malicious models.

**Recommendation:**
- Pin exact version (remove `^` caret)
- Consider bundling models at build time or using server-side processing
- Evaluate alternative libraries

---

### 🟠 SC-4: Outdated golang.org/x Packages

**Severity:** HIGH
**File:** `src/api/go.mod`

`golang.org/x/crypto` and `golang.org/x/net` are behind current versions and may contain known CVEs affecting cryptographic operations and network handling.

**Recommendation:** `go get -u golang.org/x/crypto golang.org/x/net && go mod tidy`

---

### 🟠 SC-5: No Branch Protection Rules

**Severity:** HIGH (process)

No evidence of required code reviews or status checks before merging to `main` or `beta`. Any contributor push triggers immediate Docker build and push to registry.

**Update:** A CI workflow (`.github/workflows/ci.yml`) now runs Go build, vet, architecture tests, and Vue type checks on every PR and push to `main`/`beta`. Dependabot (`.github/dependabot.yml`) automates dependency updates weekly for Go, npm, and GitHub Actions.

**Remaining action — configure in GitHub repository settings:**
- Require pull request reviews (>=1 reviewer)
- Require `CI / Go Build & Test` and `CI / Vue Type Check` status checks to pass before merging
- Restrict direct push to `main`

---

### 🟡 SC-6: Mutable Docker Base Image Tags

**Severity:** MEDIUM
**File:** `Dockerfile` (lines 2, 15)

```dockerfile
FROM node:24-alpine    # Mutable — resolves to different images over time
FROM golang:1.26-alpine
```

**Recommendation:** Pin to specific patch versions for reproducible builds:
```dockerfile
FROM node:24.12.1-alpine AS web-build
FROM golang:1.26.4-alpine AS api-build
```

---

### 🟡 SC-7: Container Runs as Root

**Severity:** MEDIUM
**File:** `Dockerfile` (line 23)

The final Alpine container runs processes as root by default.

**Recommendation:**
```dockerfile
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
USER appuser
```

---

## 4. Positive Security Practices

The application demonstrates several good security practices:

- ✅ **Multi-stage Docker builds** — build artifacts don't leak to final image
- ✅ **Static Go binary** — `CGO_ENABLED=0` eliminates C library attack surface
- ✅ **Minimal final image** — only `ca-certificates` installed in production container
- ✅ **Comprehensive .gitignore** — `.env`, database files, uploads all excluded
- ✅ **package-lock.json present** — prevents dependency version drift
- ✅ **No postinstall scripts** — no arbitrary code execution at install time
- ✅ **Least-privilege CI/CD permissions** — `contents: read` only
- ✅ **JWT + WebAuthn** — modern authentication standards
- ✅ **Bcrypt password hashing** — industry-standard work factor
- ✅ **Token refresh with rotation** — refresh tokens are rotated on use
- ✅ **CSRF-safe architecture** — JWT via Authorization header, not cookies
- ✅ **API keys stored as SHA-256 hashes** — plaintext never persisted
- ✅ **No secrets in CI/CD logs** — build args are non-sensitive

---

## 5. Remediation Priority

### Immediate (P0) — Fix Before Next Deployment

| ID | Finding | Effort |
|----|---------|--------|
| ~~B-1~~ | ~~CORS: restrict to specific origins~~ | ~~Small~~ ✅ FIXED |
| F-1 | XSS: add DOMPurify to markdown rendering | Small |
| F-2 | XSS: sanitize chat messages | Small |
| SC-2 | Remove hardcoded secret from Taskfile | Small |

### High Priority (P1) — Fix This Sprint

| ID | Finding | Effort |
|----|---------|--------|
| B-2 | SQL injection: whitelist column names | Small |
| ~~B-3~~ | ~~JWT: fail startup if secret not configured~~ | ~~Small~~ ✅ FIXED |
| ~~B-4~~ | ~~Uploads: validate file extensions/MIME~~ | ~~Medium~~ ✅ FIXED |
| ~~B-5~~ | ~~Add rate limiting to auth endpoints~~ | ~~Medium~~ ✅ FIXED |
| SC-1 | Pin GitHub Actions to commit SHAs | Small |
| SC-4 | Update golang.org/x dependencies | Small |

### Medium Priority (P2) — Plan for Next Release

| ID | Finding | Effort |
|----|---------|--------|
| B-6 | Request body size limits | Small |
| B-7 | WebAuthn session TTL | Medium |
| B-8 | WebAuthn origin validation | Small |
| SC-3 | Evaluate @imgly CDN risk | Medium |
| SC-5 | Branch protection rules | Small |
| SC-6 | Pin Docker base image versions | Small |
| SC-7 | Non-root container | Small |
| F-6 | Cache-Control headers | Small |
| F-7 | Move username from query string to body | Small |

### Low Priority (P3) — Best Practice Improvements

| ID | Finding | Effort |
|----|---------|--------|
| B-9 | Generic error messages | Small |
| F-3 | Migrate tokens to HttpOnly cookies | Large |
| F-5 | Refresh token via cookie | Large |
| F-8 | Password memory clearing | N/A (JS limitation) |

---

*This analysis covers the codebase as of the `feature/security-analysis` branch. Findings should be reassessed after remediation.*
