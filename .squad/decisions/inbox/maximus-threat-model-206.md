# Decision: Threat-Model Reconciliation Complete (Issue #206)

**Date:** 2026-05-29  
**Author:** Maximus  
**Context:** Issue #206 requested audit of `docs/threat-model.md` against current code implementation.

## Summary

Completed full audit of all 24 threat findings (B-1..B-9, F-1..F-8, SC-1..SC-7). Found 9 findings had been mitigated in code but status was stale in documentation.

## Outcome

✅ **Updated threat-model.md with current state:**
- **13 findings now Mitigated** (was 8): B-2, B-6, B-7, B-8 + F-1, F-2, F-4 + SC-1, SC-2
- **10 findings remain Open** (was 15): B-9 + F-3, F-5, F-6, F-7 + SC-3, SC-4, SC-5, SC-6, SC-7
- **1 finding Accepted** (unchanged): F-8 (platform limitation)

**All open findings now have issue links** for execution tracking (mostly #163, security audit umbrella; specific remediations linked to #201, #202, #204).

## Key Mitigations Identified

### Backend (B-2, B-6, B-7, B-8)
- **B-2 SQL injection:** Explicit whitelist map in `DeleteAnalysis()` + switch validation in `Analyze()`
- **B-6 DoS:** `MaxMultipartMemory` configured in main.go
- **B-7 WebAuthn TTL:** 5-minute TTL, cleanup logic preventing session accumulation
- **B-8 WebAuthn origin:** Dynamic origin trust removed, now restricted to configured RP origins

### Frontend (F-1, F-2, F-4)
- **F-1/F-2 XSS:** DOMPurify.sanitize() applied in CoinAIAnalysis.vue, useCoinSearchChat.ts, FeaturedCoinModal.vue
- **F-4 Sanitizer:** DOMPurify ^3.4.1 and @types/dompurify ^3.2.0 pinned in package.json

### Supply Chain (SC-1, SC-2)
- **SC-1 GitHub Actions:** All `uses:` statements pinned to commit SHAs (10 actions verified)
- **SC-2 Hardcoded secret:** Taskfile.yml `gen-env` task generates random JWT secret; config enforces 32-char minimum

## Remaining Work

10 open findings remain in scope for future remediation:
- **B-9** (error response detail): Generic error handling
- **F-3, F-5** (auth): JWT in localStorage vs HttpOnly cookies (architectural decision)
- **F-6, F-7** (auth responses): Cache-Control headers, username in query string
- **SC-3, SC-4, SC-5, SC-6, SC-7** (supply chain): CDN integrity, dependency versions, branch protection, Dockerfile hardening

All tracked under issue #163 (Code & security audit).

## Evidence

- Commit: 434f159 (docs: reconcile threat-model with current code state)
- Audit artifacts: input files analyzed (analysis.go, CoinAIAnalysis.vue, webauthn.go, Taskfile.yml, Dockerfile, GitHub workflows)
- Verification: Manual inspection of mitigated code paths + GitHub issue references (#201–204 closed issues)

## Decisions

1. **Documentation follows code:** Threat-model reflects current implementation as the single source of truth for security status.
2. **All open findings tracked:** Issue #163 is the umbrella tracker; specific issues (#201–204) document closed remediations.
3. **No architectural changes required:** All mitigations fit within current design; no ADRs needed (per Constitution §22).

## Next Steps

→ Scribe: Merge this decision into `.squad/decisions.md` under **Security Governance**.  
→ Brian: Review issue #163 for prioritization of 10 remaining open findings.  
→ Maximus: Quarterly threat-model audits per Constitution §20 (Audit cadence).
