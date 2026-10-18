# Private original attribution cards

## Authority and scope

Owner-authorized extension of the existing image and Share workflows under
Constitution Principles I, III, V and IX and sections 17-21. High-risk feature
lane: private-media/data semantics; see [ADR 0023](adr/0023-private-attribution-card-images.md).
The owner selected retention now, separate original attachments alongside the
generated card, and no change to sale recording.

Manual Add, Edit and confirmed AI intake retain uploaded card photos. Existing
coins cannot recover previously discarded photos. Cards remain private even
when a coin, profile or showcase is public. They are not normal coin photos,
primary images, gallery/tray thumbnails, analysis faces or generated-card faces.
Share asks before attaching originals; declining preserves generated-card-only
sharing. Coin of the Day context remains intact.

## Contract and short plan

1. Reuse `coin_images` and the existing upload, authenticated media, deletion
   and owner export paths with a non-primary `card` image type. No new table,
   schema migration, provider or Python persistence. Preserve uploaded bytes;
   existing camera/library image preparation remains unchanged.
2. Exclude cards from ordinary photo queries and public/follower media access.
   Expose a separately owner-authorized card list for explicit sharing.
3. Upload originals before OCR, preserve Add/intake retry progress, and make
   Edit extraction failures visible without discarding an uploaded original.
4. Reuse `useCoinShareCard`, existing dialogs and authenticated blob retrieval.
   Native sharing receives the full selected file list; download fallback
   downloads those files separately and explains multiple-download permission.

## Acceptance criteria and sibling workflows

- Manual Add, Edit and AI draft confirmation save a card without circular
  clipping or additional encoding. Successful uploads are not repeated on retry.
- OCR failure does not remove the original. Upload failures remain visible.
- Only the owner can list/fetch card originals. Accepted followers and active
  public showcases cannot fetch them, including size-variant requests.
- Collection, set, featured, follower/showcase, tray, Quick Access and health
  photo selection/counting do not treat cards as coin photos.
- Sharing no cards or declining inclusion sends only the generated PNG.
  Opting in sends that PNG plus every saved original, unchanged, as separate
  files. Retrieval failure cannot silently produce a partial share.
- Existing contextual sharing, busy/error state and object-URL cleanup remain.
- Sale recording and duplication are unchanged; duplication already excludes
  image/card rows. The full collection ZIP backup retains originals; visual PDF
  catalogs exclude them. Deleting image metadata uses
  existing durable cleanup, and coin deletion removes all image rows.

## Verification and execution boundary

Compatibility/recovery: existing photo types and schema are unchanged. Owner
collection ZIP backups include card records and source files through the existing image path.
After storing cards, do not downgrade to an API that predates the `card` privacy
filters: old photo/showcase code would treat originals as ordinary images.
Prefer a forward repair; a historical rollback requires a matching pre-feature
database/uploads backup or an owner-approved privacy-preserving recovery plan.
No recovery operation is authorized by this implementation.

Owner approved focused local tests, one independent restricted read-only review
with a ten-minute/40-read lease, and a beta push when ready. Full completion,
race/security/compatibility checks run on beta CI. No local application builds,
installations, containers, main merge or deployment are authorized.

Focused evidence covers storage bytes, privacy/negative cases, photo query
exclusion, Add/Edit/intake retries and Share inclusion/decline/fallback. Perform
negative mutation probes for load-bearing guards. Regenerate API artifacts only
with the existing pinned generator; do not install tools implicitly.

Implementation, exact-tree evidence, review and CI receipts are recorded in the
[handoff](../.squad/log/2026-10-17-attribution-card-retention.md). Until those
criteria pass, this feature is not verified or accepted.
