# Implementation Plan: v4 Release Hardening

**Branch**: `346-v4-release-hardening` | **Date**: 2026-08-15 | **Spec**: [spec.md](spec.md)

## Summary

Reconcile v4 release metadata and canonical documentation with Features 343-345,
then apply only evidence-backed, behavior-preserving maintainability fixes. Keep
RPC paused, preserve all external contracts, and complete the full release gate
before merging this branch to beta.

## Technical Context

**Languages/Versions**: Go 1.26.6, TypeScript/Vue 3, Python 3.12  
**Storage**: SQLite; no schema changes  
**Testing**: Go test/vet/build, pytest/ruff, Vitest/ESLint/vue-tsc/Vite  
**Target Platform**: Self-hosted Docker/PWA  
**Project Type**: Three-service full-stack web application  
**Constraints**: No feature expansion, no RPC automation, no main merge, no
contract or data-model changes

## Constitution Check

- **Principle I**: Structural Go changes preserve Handler -> Service ->
  Repository -> Database and composition-root ownership.
- **Principle II**: Python remains stateless; Go remains the authenticated data
  and provider boundary.
- **Principle III**: Existing typed contracts and strict builds remain unchanged.
- **Principle IV**: Refactors require measured responsibility seams; line count
  alone is insufficient.
- **Principles V-VII**: No auth/default/release-gate weakening.
- **§19-§21**: Canonical documentation, per-release threat-model review, and all
  Definition of Done gates are in scope.

No constitutional deviation is required.

## Work Batches

1. Align version, changelog, README, and implemented spec statuses.
2. Align PRD, architecture, SDD, feature, deployment, and threat-model docs.
3. Remove verified dead imports and add the missing provider-tool contract test.
4. Refactor only the verified highest-risk oversized modules with preserved
   interfaces and focused regression coverage.
5. Run full quality gates, independent QC, open a PR to beta, and merge only
   after required checks pass.

## Complexity Tracking

No violations.

