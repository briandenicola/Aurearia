# ADR 0018: Role-Specific Deep Analysis

- **Status:** Accepted
- **Acceptance evidence:** This ADR was added in owner-merged [PR #732](https://github.com/briandenicola/Aurearia/pull/732), 2026-09-20, merge `b03ac3a2c146936952e11b69bfd8dae57c8f744e`; header reconciled 2026-09-21. Body unchanged. Document acceptance does not certify the PR's still-unclosed release audit.
- **Date:** 2026-09-19
- **Supersedes:** The single-vision-call constraint in ADR 0012
- **Related:** ADR 0011, ADR 0012, ADR 0017, Feature 351

## Context

ADR 0012 replaced Deep Analysis' unused free-form image prose with one typed
combined-image hypothesis call. That made routing and proposals deterministic,
but it did not reuse the collection analysis behavior users already rely on:
separate obverse and reverse examination, side-specific configured prompts, and
collector notes. In practice, difficult coins could lose important face-level
observations before provider research and final synthesis.

Quick Lookup has a different product goal. It is intentionally fast, sends both
images together, and prioritizes an NGC certification number when visible. Its
behavior is not changed by this decision.

## Decision

Deep Analysis will:

1. Run the shared Collection AI Analysis image-examination stage once for the
   obverse and once for the reverse, with the correct image, configured
   side-specific prompt, and supplied collector notes.
2. Keep those calls side-effect free. They do not create collection AI jobs or
   write saved coin fields, notes, `obverse_analysis`, or `reverse_analysis`.
3. Derive the existing typed `CoinHypothesis` from the two face narratives,
   Quick Lookup evidence, and collector notes in a bounded text-only structured
   step.
4. Continue using that hypothesis for provider routing, query construction,
   deterministic disagreement evaluation, proposals, and synthesis.
5. Persist both face narratives additively in the Deep report and provide them
   to final narrative synthesis. Existing evidence, coverage, confidence,
   disagreements, unresolved questions, proposed fields, and attributions
   remain intact.
6. Preserve backward compatibility for reports created before face narratives
   were added.

## Consequences

- Deep Analysis deliberately performs more LLM work than the ADR 0012 design:
  two role-specific vision calls plus one bounded text-only structuring call.
  This quality tradeoff is accepted for the explicitly deeper workflow.
- Quick Lookup remains the lower-latency combined-image path.
- Provider calls, bounds, timeouts, concurrency controls, and deterministic
  query precedence remain unchanged.
- The Python, Go, persisted-report, Coin Copilot handoff, and browser contracts
  must accept the additive face-analysis field.
- User approval of the corrected Deep Analysis behavior is required before
  Feature 015 resumes.
