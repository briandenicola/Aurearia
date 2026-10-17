# Validate once, promote the verified candidate

## Authority and scope

The owner requested removal of repeated beta/PR/main validation on 2026-10-16
and approved code changes, targeted checks, one independent review and GitHub
required-check changes. The owner separately confirmed the managed-to-repository
CodeQL switch, retaining four languages and weekly scans. This is high-risk delivery maintenance under
Constitution Principles III-V and sections 17-22; ADR 0022 records the proposal.
It does not authorize a main merge, release approval, deployment, installation
on the owner's machine or a change to application contracts.

## Delivery contract

1. A push to beta runs the existing complete Go, web, locked Python, race,
   delivery, security, container, compatibility and CodeQL checks.
2. Beta image publication waits for those trusted results on that exact SHA,
   builds the two beta images, and records their immutable digests as artifacts.
3. A beta-to-main PR runs one lightweight `Verified candidate` gate. It compares
   the actual proposed merge tree with the tested beta tree, verifies every
   original check against its GitHub App and workflow/run identity, and verifies
   successful beta image receipts. No application suite is repeated.
4. A push to main repeats that lightweight identity/evidence check for the merged
   tree, not the language suites. Main publishing retains protected owner
   approval, current-main-tip checking and a final read-only evidence check.
   It promotes the recorded image digests without rebuilding either image.

Different trees are never treated as equivalent. If main introduces content not
present in the tested beta candidate, reconcile that content into beta and run
full beta validation before merging. Missing, failed, ambiguous, expired or
untrusted evidence blocks reuse; it never becomes a successful skipped check.
Ordinary PRs targeting beta retain full checks. Scheduled security/CodeQL scans
remain enabled; this change removes event duplication, not coverage.
Optional runtime acceptance also permits its supported manual beta trigger;
all mandatory source checks still require push/beta provenance.

## Source of truth and rollout

The original check/App pairs and expected source workflows live in
`scripts/delivery/validation-policy.json`. Main requires the single GitHub
Actions `Verified candidate` check; its implementation verifies that full list.
`publish-policy.json` retains the release environment and exact-main check.
Managed default CodeQL must be replaced by the pinned repository workflow to
avoid independent duplicate scans.

Beta publishing uses a beta push plus an explicit validation dependency, rather
than a default-branch `workflow_run` definition. This permits a safe bootstrap:
the first candidate can generate its receipts before these changes reach main.
The legacy default-branch beta publisher may also run once during this cutover;
neither run is cancelled or treated as deployment authorization.

Activate new required-check settings only after the new beta checks and
independent review are successful. Preserve strict mode, PR requirement, admin
enforcement, force-push/delete settings and the release environment. Record the
before/after readback. No accepted ADR body or historical receipt is rewritten.

Rollback is a separate owner-authorized operation: hold main promotions, restore
the prior workflow/verifier policy through a reviewed revert, and restore the
captured twenty check/App pairs with strict mode unchanged once those jobs are
available again. Disable the repository CodeQL trigger before restoring managed
default setup with the captured languages/query suite/weekly schedule. Preserve
scan history, every other branch-protection field and the release environment.
Do not bypass a failing guard or cancel an unrelated run as a rollback shortcut.

## Acceptance and verification

- Prove successful beta/PR/main paths and equality of the complete Git tree.
- Reject altered content, foreign PRs, wrong providers/workflows/SHAs, failed or
  missing checks, stale/failed publisher runs, mismatched or expired receipts.
- Preserve all owner/environment/current-main/partial-rerun publication guards.
- Prove promotion uses exact digests and contains no build command.
- Run `task check:delivery` and load-bearing negative controls; run one restricted
  independent review with a ten-minute/40-read-call lease and no nested work.
- Bind implementation, local results, review and live settings to exact source
  identities. CI and first protected publication remain separate runtime proof.

## Evidence and state

Implemented locally; reviewer B1 repaired and explicitly cleared by the same
reviewer on staged tree `b42b9adc6051270ebfb069555a648284320db0e8`.
The bounded pre-push verdict is PASS, not live activation or release acceptance.
Exact-head beta CI/live cutover remain pending. The share-card work is completed in PR #825; its historical
evidence is not repurposed as proof of this delivery change.

| Command | Result |
|---|---|
| `task check:delivery` | PASS: 128 tests, including retained publication guards, source provenance, merge-tree mismatch, digest-only promotion, receipt CLI and deliberate guard/workflow mutations; SpecKit negative controls and governance also passed. Proposed ADR 0021/0022 lifecycle warnings remain visible. |
| `task check:delivery` after reviewer B1 repair | PASS: 131 tests plus SpecKit/governance. Successful optional manual runtime evidence is accepted; failed/foreign/untrusted optional results and manual replacement of mandatory push checks are rejected. Guard mutation remains load-bearing. |
| Existing agent venv Python with `yaml.safe_load` over the seven changed workflows | PASS: all seven parse. Node YAML parser packages were unavailable; no dependencies were installed. Python was used only for YAML syntax, not application acceptance. |
| `git diff --check` | PASS before recording these receipts. |

Within the delivery-only delta, no application code or dependency changed. Repeating local
application suites would not exercise this delivery fix; its runner proof is
the next beta validation and PR candidate gate, not another local app build.

The [software QC audit](audits/2026-10-17.md) records all eight reviewed domains,
local evidence and remaining independent/runtime approval boundaries.

The owner then selected a small share-card title/ruler ordering follow-up.
Its separate delta/evidence is recorded in
`.squad/log/2026-10-17-share-title-first.md`; no live settings or push occurred
before that redirect. The delivery review applies to its unchanged delivery
implementation, not to the new application delta or a future combined release.

On 2026-10-17 the owner authorized pushing **both changes together**, then
opening one beta-to-main PR after green gates. Delivery executable files remain
identical to the independently reviewed tree; the separate four-file share
delta changes only title/ruler ordering and its tests/docs. Its 28 focused
tests and two actual desktop/mobile PNG-download tests passed. Complete web and
other configured runner gates will bind the combined pushed commit. Beta was
fast-forwarded to merged main `6e11a6c9` after proving its tree identical to
baseline `8fd3cb69`; no content or reviewed delivery source changed during sync.
The PR completion record will hold exact commit/tree, CI, settings and review
receipts, without another beta receipt commit triggering repeated validation.
