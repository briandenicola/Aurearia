# P3 Shared Validation and Deterministic Governance

Date: 2026-09-21
ID: GOV-004
Scope: Owner-authorized delivery plan D08-D10
Status: Independently reviewed, verified and owner-accepted via PR #737
Authority: Constitution Principles IV/VII/VIII/IX; sections 17-19/21; ADR 0019

Taskfile completion targets are the shared local/CI route. Dependency setup
uses separate explicit targets; missing lint/tools are failures, not optional
success. Keep runner-only security, race, browser and compatibility checks.

The offline Node checker validates only explicit active files and structural
claims: links, principle identifiers, ADR index/status, current-work references
and required native-skill metadata. It does not infer approval, independence,
release, discovery or correctness of prose. Archives and historical bodies stay
unchanged; context size and Proposed lifecycle questions remain warnings.

The owner separately approved renaming 33 colon-containing historical log
filenames for Windows portability. The
[path map](../../artifacts/windows-log-path-mapping-2026-09-21.json) records
old/new names and identical blob IDs; contents and meaning are unchanged.
Both hosted delivery jobs now use full checkout with NTFS protection enabled.

Current-work metadata supports an existing process plan, a feature spec/tasks
pair, or a project issue URL. Small issue work does not require an invented spec.
URL existence/approval is not checked offline. P4 owns skill migration and
instruction reduction; P5 owns live protection configuration.

The existing Swagger/version mismatch is corrected only in the annotation and
four generated metadata snapshots. Swag pin verification uses module build
information because the pinned module's CLI banner reports an older version.

This record is reflected in active decisions GOV-004. No historical reviewer
block is cleared and no application repair, installation or deployment is
authorized by these delivery controls.

Owner merge `c36387275ce6b34ed012a29c2a861a2b5dd17f02` has the same tree as
reviewed candidate `d555f132dc56b130e1ab8f22ec633876b99323b1`. All 20 PR checks
passed. See the [final handoff](../../log/2026-09-21-delivery-validation-closeout.md)
for exact evidence and remaining boundaries.
