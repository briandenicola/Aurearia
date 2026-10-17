# Validate once; promote verified content

Date: 2026-10-16
Status: Owner-authorized implementation; review and live activation pending

The owner requested eliminating repeated beta/PR/main CI and approved code,
targeted checks, one independent review and GitHub required-check changes.
The bounded [protocol](../../../docs/ci-candidate-reuse.md) and proposed ADR 0022
retain all source validation, exact complete-tree comparison, trusted App/run
provenance, protected owner release approval and immutable image digests.

This supersedes only GOV-006's repeated per-SHA check execution/configuration
once the recorded rollout is activated. Original P5 acceptance and approval
history remain preserved; no historical reviewer block or application-release
condition is cleared. No main merge, protected release or deployment is
authorized by this process-maintenance request.
