# Brutus QA Audit — Cross-System Reliability Decisions Needed (2026-05-29)

## Context
Cross-system QA audit across `src/web`, `src/api`, `src/agent`, and `docs/threat-model.md` found integration/timing risks that need coordinated owner decisions (frontend + api + agent).

## Proposed team-level decisions

1. **Define a single streaming resilience contract (web↔api↔agent).**  
   Require: token refresh support for streaming endpoints, client-side abort/timeout handling, and guaranteed terminal SSE semantics (`done` or explicit `error`) so UI cannot remain indefinitely loading.

2. **Define scheduler concurrency policy for manual vs scheduled runs.**  
   Require: explicit single-flight behavior (lock or DB guard) per scheduler type so overlapping triggers cannot create duplicate notifications or duplicate run records.

3. **Enforce cross-service payload caps at both boundaries.**  
   For availability checks, chunk Go→agent requests to respect agent `MAX_AVAILABILITY_ITEMS` and add tests proving behavior when wishlist URLs exceed one payload.

4. **Promote mitigated security controls to tested invariants.**  
   For threat-model findings marked Mitigated (notably DOMPurify render paths and auth rate-limit behavior), require at least one automated regression assertion per control.

## Why this needs team decision
These changes touch contracts and behavior across multiple services, not a single isolated module. Implementing piecemeal risks breaking compatibility or creating contradictory timeout/retry behavior.
