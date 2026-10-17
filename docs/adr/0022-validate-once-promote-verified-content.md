# ADR 0022: Validate Once and Promote Verified Content

Date: 2026-10-16
Status: Proposed

## Context

The same full application, security and compatibility suites currently execute
on beta push, beta-to-main PR and main merge. Images are also rebuilt for main
after beta publication. The owner explicitly requested removing this duplication
and approved implementation, targeted verification, one independent review and
required-check configuration changes.

## Decision

Run complete validation on beta. Require one lightweight candidate gate on
beta-to-main PRs and main pushes. Reuse evidence only after comparing complete
Git tree identities, checking the original trusted check/App and workflow/run
identities on the source SHA, and checking immutable beta image receipts.

Do not reuse evidence for changed content: reconcile and fully validate beta
again. Missing or unsuccessful proof fails closed. Replace managed CodeQL
triggers with a pinned repository workflow covering the same four languages on
beta and on the retained scheduled scan.

Retain actual owner release approval, the protected main-only environment,
current-main-tip verification and evidence rechecking in both publishing jobs.
Promote receipt-bound image digests instead of rebuilding. Promotion records
the tested beta commit separately from the main merge commit; labels are not
rewritten to pretend the image was built from a different commit.

## Consequences

The expensive checks run once per beta candidate. PR/main checks establish
content identity and provenance rather than repeating language tests. A
different main merge SHA is allowed only when its entire tree equals the tested
source tree. Changed content and expired artifacts require new beta evidence.

Main's required context changes from twenty individual contexts to one verifier
that still requires the original validation coverage. Existing historical
acceptance/publication receipts remain unchanged. This is not authorization to
merge, publish or deploy an application candidate.

## Related

- [Implementation contract and owner-authorized scope](../ci-candidate-reuse.md)
- [ADR 0019](0019-evidence-based-agentic-delivery.md)
- Constitution sections 17-22
