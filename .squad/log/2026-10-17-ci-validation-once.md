# Validate-once delivery maintenance

**Date:** 2026-10-17
**Owner:** Copilot CLI implementation owner
**Baseline:** beta `8fd3cb698ef3c34eb297d42912d3d30ddd8101ca`
**Authority:** Owner requested removal of repeated beta/PR/main CI; approved code,
targeted checks, one independent review and required-check changes; separately
approved moving managed CodeQL to the pinned beta/weekly repository workflow.
**Lane:** High-risk delivery controls; Principles III-V and sections 17-22.

The canonical [protocol, audit and local receipts](../../docs/ci-candidate-reuse.md)
record scope, non-goals, exact-tree reuse, source provenance, digest promotion,
bootstrap/rollback boundaries and pending runtime evidence. Local delivery
validation passed 128 tests plus SpecKit/governance; all seven changed workflows
parsed using the prepared Python YAML parser. No local app build, installation,
container, main merge, release approval or deployment was performed.

## Next action

Obtain the one approved restricted independent review against the staged tree.
After any required repairs and clearance, commit/push beta, verify exact-head
source validation, and apply/read back only the authorized CodeQL/check changes
in the documented safe order. Persist runtime/PR references in the canonical
completion record without manufacturing acceptance or reusing prior share proof.

## Review repair delta

Independent reviewer session `58508e6c-da12-41fc-954e-ecddc91a2c2e` blocked the
first staged tree on B1, successful manual optional runtime evidence being
rejected. The author added a bounded optional-event allowlist and explicit
positive/negative/mutation regressions; the shared delivery gate passed 131
tests. Same-reviewer clearance is still required before push/live mutation.

The same reviewer subsequently cleared B1 and returned bounded pre-push PASS
for `b42b9adc6051270ebfb069555a648284320db0e8`. The owner then selected the
share-title follow-up, recorded separately. Delivery source remains unchanged;
no delivery commit/push or live settings mutation has occurred. Resume the
authorized cutover only with exact combined-candidate evidence and an explicit
applicability record for the separate application delta.

Owner subsequently approved pushing both changes together and opening one PR
after green gates. Delivery executable source still matches reviewed tree
`b42b9adc6051270ebfb069555a648284320db0e8`. Share regression evidence is 28 unit/
entry-point tests plus two real desktop/mobile PNG downloads. Beta synchronized
to merged main only after full-tree equality with the reviewed baseline was
proved. Next: commit/push the approved combined scope and perform the already
authorized settings cutover, binding all runner proof in the PR.
