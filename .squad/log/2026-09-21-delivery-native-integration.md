# P4 Native Integration: Review and Publication Handoff

Date: 2026-09-21
Implementation owner: Copilot CLI
Authority: Owner-authorized delivery plan D11-D13; Constitution sections 17-18/21;
accepted ADR 0019. Worktree branch: `docs/delivery-native-integration`.
Base: `7df826fc2504ce1154c05bc8b5d11a818ef7014b`.
Independently reviewed source tree: `de19c2dbec1bc708767396ae9837e34dcbc12911`.

## Implemented and verified

Universal instructions reduced from 328 to 86 lines. Five native scoped files
retain domain rules. Nineteen native skill packages replace twenty legacy entry
points; wrappers link canonical packages and originals. Every inventory blob and
path was checked against the accepted baseline. The personal audit skill is
unchanged. Recipe corrections and the pinned, source-only SpecKit v1.0.9
evaluation are recorded in [the integration guide](../../docs/agentic-native-integration.md).
No installation, generated upgrade preview or application change was performed.

Single-owner delivery, optional Squad, bounded assignments, independent review,
empty-result handling and verified persistence are wired through routing,
ceremonies and the coordinator. `task review:read-only` adds explicit SQL/skill
exclusions to the native profile; plain `--agent` is not sufficient on this client.

`task check:delivery` passed on the exact reviewed source tree after review:
46 Node tests, PowerShell selection/Boolean-stream regression plus negative
control, and zero governance errors. New metadata/capability guards are
load-bearing: disabling their diagnostics breaks fixtures. Bad pointers,
duplicate recipes, invalid scopes and expanded tools reject. `git diff --check`
passed. All prior delivery guards remain active.

The 6,000-word conservative context warning remains (7,237 words on the reviewed
source), plus two links to tracked sparse-excluded historical logs in this
worktree. The context exception is recorded for owner acceptance, not hidden
by deleting binding policy or reviewer restrictions.

## Runtime failure and clearance

Installed CLI 1.0.87-0 discovered all nineteen enabled project skills separately
from the owner's personal audit. Probe reviewer
`4afde450-0635-4a31-958f-de462f95996c` initially returned BLOCK after six read
calls: profile selection also exposed session SQL/skill capabilities; instruction
metadata was native-discovered but matching bodies were not automatically injected.

The owner approved the explicit-exclusion launcher and explicit instruction
loads, renewing the probe to eight calls/five minutes. Relaunched through the
actual Task target, the SAME reviewer reported only `view` plus its parallel
wrapper, explicitly loaded four matching scoped bodies, refused the conflicting
write/execute/delegation request, and returned PASS with both findings explicitly
CLEARED. Seven read calls; before/after changed/untracked file hashes equal.
This proves the documented workaround, not plain-profile sufficiency or automatic
body injection. Raw local transcripts remain outside the repository because they
contain personal/global context; no raw debug logs are published.

## Independent review

Fresh reviewer `4f05f96c-2545-45b0-978c-0c046934c60e` used the restricted Task
launcher and all twelve approved read calls within ten minutes. It inspected the
complete supplied diff, authoritative policy/ADR/plan, instructions, skills,
provenance, guards, routing and evidence. Verdict: **PASS**, no findings.
It independently observed only file-reading/parallel tools. Git/tree identities,
test runs and earlier probe results were caller-supplied, not independently
executed by this read-only reviewer.

The reviewer required full validation on the exact final candidate before
publication and hosted evidence afterward. It allowed a closeout delta that only
accurately records its verdict/pending evidence without a new bounded review.
After its reviewed source tree, changes are limited to this handoff and factual
task/decision/current-pointer status. The implementation owner will run the full
delivery target again on that final tree and bind its commit/results in the PR.

## Pending conditions and exact next action

Publish the bounded PR into beta, record hosted Windows/Linux and other configured
checks against its head commit, and obtain owner acceptance. At this handoff's
creation, those hosted results and owner merge are pending; do not infer them
from checked tasks. The inherited #738 container-scan failure was a Syft-download
HTTP 504, not a newly diagnosed product vulnerability.

The original dirty beta worktree was preserved. No historical application
review block is cleared; no product repair, tool installation/upgrade, live
protection change, merge, release or deployment is authorized. P5/P6 have not
started. The implementation owner wrote and verified this canonical handoff;
Scribe/further fan-out was unnecessary.
