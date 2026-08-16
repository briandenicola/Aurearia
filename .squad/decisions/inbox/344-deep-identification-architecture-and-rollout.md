# Feature 344: Deep Identification Architecture and Rollout

Date: 2026-08-16
Status: Proposed
Authority: ADR 0011; Feature 344 plan; Constitution Principles II, IV, V and §17

Deep Analysis is a sibling, Go-owned persisted job/event/artifact domain rather
than an extension of `AIJob`. Python remains stateless; all provider HTTP calls,
credentials, budgets, persistence, retention, and confirmed writes remain in
Go. Public SSE frames are translated and persisted before replay.

Provider selection is backend-eligible and transparent. Exact provider outcomes
and router rationale are owner-visible. NGC remains link-out only, OCRE follows
ADR 0010, and RPC automation remains paused.

Rollout remains dark by default through `DeepIdentificationEnabled=false`.
Enable for controlled validation only after the full §17/§21 release gate and
post-major-work audit pass. Roll back by disabling the setting; in-flight and
retained data continue through their bounded lifecycle without a schema
rollback.
