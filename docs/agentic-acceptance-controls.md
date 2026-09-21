# Acceptance and Release Controls

P5 D14-D15 under Constitution sections 17-18/21 and accepted ADR 0019.
**Status: repository controls accepted in #741; live activation incomplete.**
The owner subsequently approved the scoped main-protection and release-environment
changes during release-readiness remediation. Main's twenty check/App pairs,
strict mode, PR requirement and admin enforcement are applied and read back.
The owner-reviewed `release` environment and main-only branch rule exist.
On 2026-09-21 the owner explicitly chose to leave `can_admins_bypass: true`
and authorized verifier alignment while retaining actual owner approval and
all exact-candidate checks. This supersedes only the original no-bypass-setting
requirement; an administrative environment bypass does NOT authorize publishing.
See the
[activation receipt](audits/2026-09-21.md#live-control-receipt).
Beta and Ralph remain unchanged. No publishing, deployment or main promotion
was performed. The first real protected approval path remains separately pending.

## One completion record, not another tracker

Use the [PR template](../.github/pull_request_template.md) as the canonical record.
Before a PR exists, use the approved issue/process plan. When publishing a PR,
point the earlier record at it. Handoffs/checkpoints link that record and describe
only what changed since the previous checkpoint; link raw logs instead of
duplicating transcripts or maintaining competing histories.

The record contains the approved criteria/non-goals, affected and sibling
workflows, candidate commit/tree, result/evidence/test identity per criterion,
independent reviewer/verdict/source identity, changes since review, unresolved
blocks/exceptions and actual owner decisions. Include code/test consumers of
relocated documentation: #739's missed UI-reader contract is the concrete example.

| State | Evidence needed |
|---|---|
| Implemented | Approved scoped change exists; pending work/checks are explicit |
| Verified | Applicable gates and original acceptance/negative paths are evidenced for this candidate |
| Accepted | Applicable independent review is current, blocks are cleared by the right reviewer, and required owner approval exists |
| Released | The explicitly authorized candidate was published; record image identities/run, not an assumed deployment |

Missing evidence is pending/INCOMPLETE, never a passing default or unexplained N/A.
Changed implementation invalidates applicable old proof. For a factual
receipt-only delta, show the exact diff and why prior evidence still applies;
do not infer equivalence from a branch name, a similar commit or checked tasks.
An open block prevents an accepted/release-ready claim even when CI is green.
Existing application restrictions retain their original reviewer/author terms.

An owner decision is a real owner message or GitHub approval/merge record tied
to the candidate and scope. Agent-written checkboxes/comments cannot supply it.
Owner acceptance into beta is not main/release approval. For main publishing,
the owner must inspect this record and unresolved release conditions before
approving the protected GitHub job displaying the candidate SHA. Publication
does not authorize a deployment to the local server.

## D15 before/after proposal

Historical read-only baseline inspected 2026-09-21, before the later approved
activation attempt. No repository rulesets were returned;
classic branch protections apply. The live main/beta publisher blobs match the
baseline checked-in files. Only the unprotected `copilot` environment exists.

| Control | Observed before | Proposed after |
|---|---|---|
| Main required checks | Seven, strict/up-to-date | All twenty [policy checks](../scripts/delivery/publish-policy.json), strict/up-to-date, bound to observed GitHub App IDs |
| Main PR requirement | None | PR required, **zero additional human approving reviews** |
| Main admin enforcement | Off | On; no routine admin bypass |
| Main force push/delete | Both off | Unchanged |
| Main signatures/linear history/conversation resolution/locking | Off | Unchanged |
| Beta | Direct pushes; zero required checks; admin enforcement off; force push/delete allowed | Unchanged; preserve the integration workflow |
| Main publication | Automatic after successful Quality Gate push; exact checked SHA | Prepare automatically, pause for explicit owner approval, then verify nineteen push checks and approval for that candidate before either image job |
| Beta publication | Automatic checked-SHA `beta` images | Unchanged; validation images, not a main release |
| New `release` environment | Absent | Required reviewer: repository owner; self-review allowed; admin-bypass availability permitted by the 2026-09-21 owner amendment, but actual owner approval still mandatory to publish; only branch `main`, no tags/wildcards |
| Ralph | `disabled_manually` | Intentionally disabled; enabling needs separate approval |
| Other settings | Auto-merge off; default workflow token read; Actions cannot approve PRs | Unchanged |

The seven existing main checks are Go API, Vue Web, Python Agent, Gitleaks,
Govulncheck, npm audit and pip-audit (App ID 15368). Add race detection, both
delivery runners, both container scans, the agent-runtime pip check, both
compatibility/browser checks, four CodeQL language analyses (also App ID 15368),
and the CodeQL aggregate (App ID 57789). The JSON policy is the canonical list;
do not maintain a second mutable copy of context names.

All twenty are required for main PRs. Publishing requires the nineteen checks
emitted on pushes, including all four CodeQL analysis jobs. The CodeQL aggregate
is marked `required_for_publish: false`: #740's PR head has it, while its green
merge SHA and the inspected live main SHA do not. Requiring that absent summary
on a push would permanently block publication. If a trusted aggregate does
exist on the candidate SHA, it must still be successful; it is not ignored.

Zero second-human approvals avoids locking a solo maintainer out of their own
PR. It does **not** waive independent agent review or owner acceptance.
Administrative capability to edit repository settings still exists; this is
not a claim that administrators are technically unable to change policy.

## Main publishing boundary

The main workflow adds a protected `approve-release` job; both existing image
jobs depend on its success and still check out `workflow_run.head_sha`.
Each publishing job repeats the read-only check before any Docker step: GitHub's
"rerun failed jobs" can reuse a previously successful approval job, so `needs`
alone must not authorize a now-stale candidate.
Main publishing runs are serialized so a later candidate cannot finish before
an earlier running publisher and then be overwritten by it. A superseded queued
candidate is not entitled to publish merely because it once passed checks.
Beta workflow bytes are unchanged.

After GitHub's environment approval, the
[read-only verifier](../scripts/delivery/verify-main-publish.mjs) checks:

1. The event and independently fetched upstream run identify a successful
   `.github/workflows/ci.yml` push on main, in this repository, for a full SHA.
2. The candidate is still the main tip at the approval check.
3. The live environment has the single owner reviewer, permits solo self-review,
   reports a Boolean admin-bypass setting, and allows only the main branch.
4. Every required publishing check has exactly one matching latest result from its expected
   App, for the same SHA, completed successfully. Missing, failed, pending,
   neutral, skipped, stale or ambiguous results block publication.
5. GitHub's approval history for **this publishing run**, not the triggering CI
   run, contains the actual owner's approval for `release` and no pending or
   rejected release decision.

The verifier only performs GET requests. It cannot approve a job, change settings,
publish an image or deploy. It rejects missing permission, malformed responses
and incomplete pagination rather than substituting a default. A profile, PR
checkbox or unprotected automatically created environment cannot satisfy it.
Admin-bypass availability may be enabled or disabled. Bypassing the environment
without GitHub-recorded owner approval for this publishing run still fails step 5;
it does not waive a failed check, stale candidate or malformed evidence.
It targets the current GitHub.com, individual-owner repository; an organization
transfer or GitHub Enterprise move requires a reviewed policy change.

If other checks are still pending, wait before approving. If verification fails,
inspect the explicit failure and fix the evidence/configuration; do not weaken
the guard. A run containing a rejected decision needs a fresh owner-authorized
publishing run. Never rerun an old candidate to move `latest` backward.
The two image pushes are not a registry-wide atomic transaction: record both
successful image identities before treating publication as complete.

These controls enforce technical checks and a platform-recorded owner action,
not the truth of every manual criterion. The owner/reviewer must still inspect
the completion record and unresolved application/release blocks.

## Activation sequence (separate owner approval required)

1. Review/test the repository draft and accept its PR into beta. This does not
   authorize main promotion, live settings changes, publication or deployment.
2. Obtain explicit approval for the before/after table and rollback below.
   Re-read live settings first; if they changed since the snapshot, stop and
   reconcile instead of overwriting someone else's changes.
3. Record the fresh baseline. Apply only the proposed main protection fields
   and new release environment, including its main-only branch policy. Keep
   beta, other environments, secrets, token defaults and Ralph unchanged.
   The owner amended the original no-bypass requirement on 2026-09-21: leave
   **Allow administrators to bypass configured protection rules** enabled and
   record the actual Boolean readback. Do not guess undocumented write fields.
   Leave **Prevent self-review** unchecked and set
   **Selected branches and tags** to one Branch rule named `main`.
4. Read back every changed setting and compare exact check/App pairs. Record
   approval source, operator, timestamp, before/after values and restoration
   procedure in the existing PR completion record. Do not claim enforcement
   from intended configuration alone.
5. Keep main promotion on hold until the reviewed guard workflow will be
   included in that candidate. `workflow_run` publisher definitions are taken
   from the default branch: beta acceptance alone does not activate this guard.
   The old main workflow remains automatic until that separately authorized
   promotion; live environment creation alone does not retrofit the old workflow.
6. A future, separately authorized main candidate must clear application/release
   blocks, all checks and the protected owner approval. Record its first real
   environment/approval/API verification result; offline fixtures are not proof
   of a live approval gate. No real publishing/deployment is a P5 validation shortcut.

D15 is **not fully activated** while live changes/readback or runtime approval
evidence remain pending. Do not mark settings as applied merely because code
or this proposal merged.

## Exact restoration plan

Rollback is an owner-authorized operation, not an automatic response to failure.
First hold new main promotions and identify any queued/running main publishers.
Disabling a workflow does not cancel existing runs; cancel only explicitly
identified runs with owner approval before changing their approval boundary.

Restore main's seven original check/App pairs with strict mode still on; remove
the new PR requirement (`required_pull_request_reviews: null`), restore
`enforce_admins: false`, and leave all other baseline fields unchanged. This
restores weaker historical controls and therefore requires explicit approval.
Use a reviewed revert for the guard workflow/helper/policy if requested; do not
rewrite an accepted commit or silently reactivate automatic publication.

Delete the new `release` environment only after no active workflow references it
and the owner authorizes deletion. If it acquired secrets or unrelated settings
since creation, stop for a revised preservation plan. Never touch `copilot`.
Beta and Ralph need no restoration because this proposal changes neither.
Read back the restored values and record the exact result in the same completion
record; do not erase the original approval/failure history.

## Verification and references

`task check:delivery` runs offline positive, negative and mutation fixtures for
candidate identity, trusted checks, actual approval, environment configuration,
pagination and both publisher dependency paths. They make no GitHub mutations,
container builds, image pushes or deployments. Prompt behavior and independent
review are separate evidence; live settings and first-run proof remain explicit.

- [GitHub deployment protection rules](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments)
- [Environment API](https://docs.github.com/en/rest/deployments/environments#get-an-environment)
- [Environment settings UI](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments)
- [Deployment branch policy schema](https://docs.github.com/en/rest/deployments/branch-policies#list-deployment-branch-policies)
- [GitHub's OpenAPI environment schema](https://github.com/github/rest-api-description/blob/main/descriptions/api.github.com/api.github.com.json)
- [Workflow approval history API](https://docs.github.com/en/rest/actions/workflow-runs#get-the-review-history-for-a-workflow-run)
- [P5 authorizing plan](agentic-delivery-improvement-plan.md#p5-tie-acceptance-and-release-to-evidence)
