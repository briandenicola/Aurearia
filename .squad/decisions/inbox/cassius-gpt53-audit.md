# Cassius Runtime Audit — Decision Request (2026-05-29)

## Context
Principal-engineer audit of Go API + Python agent surfaced cross-cutting runtime risks that need team-level direction because fixes affect auth contracts, outbound network policy, and scheduler behavior.

## Decision Requests

1. **Auth token transport hardening**
   - Adopt policy: JWTs are accepted only via `Authorization: Bearer` for protected API routes.
   - Keep query-param token support only for explicitly carved-out legacy endpoints (if any), with sunset date.

2. **One-time refresh rotation semantics**
   - Enforce single-use refresh token rotation with atomic DB revoke (conditional `revoked_at IS NULL`) + uniqueness-safe retry path.
   - Define expected client behavior for concurrent refresh attempts (one success, one 401).

3. **Unified outbound HTTP safety profile**
   - Require all user-influenced outbound calls (Go + Python) to share baseline controls: URL scheme allowlist, private-IP/localhost denylist, redirect revalidation, explicit timeout budget, and bounded response reads.
   - Apply first to availability checks and NumisBids ingestion paths.

4. **Scheduler idempotency persistence standard**
   - For user-facing alerts, require DB-backed idempotency keying (date/user/type) rather than process memory maps to survive restarts and multi-instance deployment.

5. **Operational reliability guardrails**
   - Add mandatory tests for: refresh race, repeated cancel calls, SSRF blocking, and scheduler restart duplicate suppression.

## Why team decision is needed
These changes cross service boundaries and alter externally observable behavior (auth refresh outcomes, accepted token transport, alert delivery semantics). Aligning now avoids piecemeal fixes and regressions.
