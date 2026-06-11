# ADR 0006: Workflow Contract Regression Gates

## Status

ACCEPTED

## Context

Recent regressions have repeatedly broken user workflows even when unit tests,
linting, CodeQL, and CI passed. The recurring failure mode is not a static code
quality issue; it is contract drift between shared surfaces such as Admin
settings, Vue forms, API DTOs, service validation, and persistence behavior.

The coin Era regression is the concrete trigger: Admin `CoinEras` configured the
Edit Coin dropdown, but the API update path validated against a different source
of truth. Each layer looked reasonable in isolation, yet the user workflow
failed.

## Decision

The constitution and PR template now require workflow-contract checks:

1. Bug fixes must include a targeted regression test for the exact failing user
   path, or explicitly document why automation is deferred.
2. PRs touching shared workflow surfaces must list the affected blast radius and
   sibling workflows checked.
3. User/admin-configured UI values must be accepted by every API path the UI can
   submit them to, or the UI must prevent the invalid submission with a clear
   message.
4. CI remains a necessary gate, but green CI is not considered proof that the
   user workflow is safe unless the relevant workflow contract is covered.

## Consequences

- Regression fixes become slightly more expensive up front but harder to repeat.
- PR descriptions must explain workflow contracts and blast radius instead of
  only listing build/test commands.
- Shared surfaces such as Add Coin, Edit Coin, Admin Settings, wishlist flags,
  collection counts, set membership, and AI intake require sibling workflow
  checks when touched.
- CodeQL and static checks remain useful, but they are explicitly not a
  substitute for workflow-contract coverage.
