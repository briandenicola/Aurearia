# Private attribution-card retention

Owner selected retention and Share inclusion now, originals as separate files
alongside the generated card, and no sale-recording changes. Approved focused
local tests, one restricted independent review (ten minutes/40 reads), and beta
push when ready. No local builds/installations/containers or main/deploy actions.

Baseline: beta `2b7503b054590ccf6c4af65b9afa0c0e2c2df33a`, initially clean.
Authority: Constitution 4.0.0, Principles I/III/V/IX, sections 17-21, explicit
owner selections and `docs/attribution-card-retention.md`; ADR 0023 is Proposed.

Discovery confirmed the existing upload service writes source bytes unchanged,
with separate optional variants. Current manual Add/Edit and AI confirmation
discard the card photograph. Ordinary image queries and follower/showcase media
authorization must exclude the new private card type. Sale and duplication
remain unchanged; duplication already excludes image/card rows.

Status: implementation and verification pending. Next action: implement the
contract, prove positive/negative behavior, then supply exact-tree evidence to
the authorized independent reviewer. No acceptance or release inferred.

## 2026-10-18 implementation and focused evidence

Implemented non-primary, owner-only `card` uploads/listing with original-byte
retention, no variants/clipping, private/no-store authenticated media responses,
public/follower denial, and shared photo-query exclusion. Manual Add and intake
queue card uploads with retry progress; Edit retains before OCR and reports
extraction failure. Opt-in Share uses separate authenticated originals alongside
the generated PNG. Gallery, set, featured, analysis, tray, Quick Access, health and
visual PDF queries exclude cards; full collection ZIP backup preserves originals.
Sale recording and image-free duplication are unchanged.

Fast-forwarded beta to owner-merged main
`86f64bdda864c34948ae48dba8da04bdc67e5ece` only after comparing its entire tree
with baseline `2b7503b0` and proving byte-identical content. Preserved all feature
changes. No main write or deployment.

Approved focused execution, using the previously prepared Go 1.26.9 SDK with
`GOTOOLCHAIN=local GOPROXY=off`, no installation or application build:

- `go test . ./handlers ./repository ./services -run 'TestAttribution|TestImageHandler|TestResolveAuthorizedMediaPath|TestValidateImageData|TestCoinImagePathInActiveShowcase|TestUserRepository|TestArchitecture|TestRegisteredAPIRoutesAreDocumentedInOpenAPI' -count=1 -json`
  passed: four packages, eleven top-level tests / 43 including subtests.
- Node-direct Vitest for `useCoinShareCard`, `coinShareCard`,
  `AddCoinPage.save`, `AddCoinPage`, `EditCoinPage`, `FeaturedCoinModal`,
  `CoinDetailPage`: 66 passed across seven files after mutation restoration.
- Node-direct Playwright `test coin-share-card --project chromium --workers 1`:
  six passed at 1440px and 390px. Actual download files and native-share File
  payloads preserve both originals byte-for-byte; decline fetches no originals.
- Node-direct ESLint on all thirteen changed frontend files,
  `--max-warnings 0`: passed, zero warnings.
- `task openapi`: passed with existing pinned swag, no installation.
  Registered-route drift and architecture tests included above.
- `node scripts/delivery/check-governance.mjs`: zero errors; three visible
  Proposed-ADR lifecycle warnings (0021, 0022, new 0023).
- Fifteen isolated mutation probes each produced an expected assertion failure
  for owner listing, follower/public denial, primary/content restrictions,
  original bytes, photo/Quick Access/health/tray exclusion, manual/intake storage,
  Edit retry, explicit consent and partial-share failure. The harness restored
  every source file byte-for-byte and final focused tests passed afterward.

Detailed author-supplied outputs and mutation receipts are in the session's
`files/attribution-*` artifacts. Full Go/web completion and runner-only checks
remain pending beta CI by owner choice. Independent review pending; no PASS,
acceptance, release, main PR or deployment claimed.

## Independent review B1 and repair

Restricted reviewer session `4f68263e-d1ab-49e7-bb02-f244acc97c6e` reviewed staged
tree `369f7a1d14c155a4ba27a0b98cd81e555f7fc8be` and returned BLOCK B1 after 34/40
read calls within the ten-minute lease. B1: Edit repeated completed face
replacement/deletion on retry after a later card-upload failure; the stale
deleted ID could permanently prevent reaching the card upload.

Author repair is permitted. Edit now retains per-face upload and outstanding
superseded-deletion progress, remembers completed deletions, targets the actual
saved replacement if the user changes/removes a face, and preserves cleanup-
pending notification state across retries. No other executable paths changed.

Six new Edit cases cover both-face replacement followed by a card failure,
cleanup-pending preservation, deletion retry, completed explicit removal, and
changing/removing the saved replacement before retry. Reproduced B1 by running
the new combined regression against the original staged editor: expected
assertion failures. Seven isolated repair guard mutations also failed as
expected; restored final source exactly. Final focused frontend evidence:
72 passing tests across seven files; repaired source/tests lint with zero
warnings. Existing Go/OpenAPI/Share browser evidence remains applicable because
those source paths are unchanged.

B1 remains open until the original reviewer inspects the scoped repair,
updated exact tree and evidence within its remaining six-read lease. No beta
push or reviewer clearance is inferred from this author repair.

## Re-review B1 clearance and B2

The same reviewer inspected revised tree
`8551dfaa794ee0e0834671c35b5a7ad2642506ef`, explicitly cleared B1, and blocked B2:
when a previously absent face is newly uploaded before card failure, clearing
its preview leaves both the selected file and original removed-image ID empty;
the editor did not reconcile its saved upload in that state.

Reviewer exhausted the approved 40/40 aggregate reads. No more reviewer work is
authorized without an owner lease extension. Author repair is permitted.

Author repaired B2 by deriving removal IDs from saved upload progress before
falling back to original removed-image IDs. Both obverse/reverse regressions
start with no saved images and null original IDs; clearing the selection after
a failed card upload now deletes the actual saved face. All sixteen Edit tests
pass; source/tests lint with zero warnings. B2 remains blocked until the original
reviewer independently clears this repair. No beta push has occurred.

Next action: request owner approval for a five-minute/ten-read extension of the
same reviewer, finish exact-candidate negative/focused evidence, then obtain
clearance before the already-authorized beta push.

Owner declined the requested review extension. No further reviewer work was
launched. The B2 author repair is local, not independently cleared; its negative
proof and final combined frontend rerun remain incomplete. Beta publication
remains blocked; this denial does not waive the required review.

## Owner-authorized final sign-off

Owner subsequently requested completion and explicitly approved one final read
by the existing reviewer, then commit and push beta; no broader review.
Final focused frontend selection passes all 74 tests across seven files.
Both no-original-face regressions fail at the expected deletion assertion when
saved-face reconciliation is disabled, then pass after restoration. Final Edit
source/tests lint with zero warnings. Evidence: `b2-negative.txt` and
`b2-final-web.txt` in the existing session review packet.

Original reviewer explicitly returned PASS on staged tree
`fd86517acef4b361ce7eb004f626cdcd7816a692`, parent
`86f64bdda864c34948ae48dba8da04bdc67e5ece`: B2 cleared, B1 remains cleared,
no remaining pre-push blocker. Exactly one authorized final read was used.
Receipt: `attribution-review/b2-final-review.txt` in the session artifacts.
Subsequent changes are this receipt and factual current-work pointer only;
reviewed application/test content is unchanged. Commit/push beta is authorized.
Full completion is determined by exact-head beta CI, not this pre-push PASS.
No new main PR, main merge, deployment or release is authorized.
