# Recoverable image deletion

**Status:** Accepted for #729 with user approval. Constitution Principles I, IV, V and IX; sections 17 and 21.

Deleting image metadata and deleting files cannot share a transaction. Previously, filesystem errors and metadata write errors were ignored, and files could disappear while metadata remained.

The image repository now transactionally inserts a small `image_cleanups` row and deletes the authoritative image row. No file is touched until that transaction commits. The cleanup row contains the image, coin and owner IDs, stored relative path, and creation timestamp. It has no cascading foreign keys, so deleting an account or coin cannot discard outstanding file work.

Cleanup removes only the recorded original and its `_thumb.jpg` and `_medium.jpg` variants, under the configured upload root. It never enumerates or recursively deletes filesystem directories. Root-scoped filesystem operations reject traversal outside the upload root; directory targets are rejected. Missing files count as already cleaned.

The row is removed only after all three files are absent. A cleanup failure returns HTTP 202 with `cleanupPending: true` and an explicit retry message, and is logged; HTTP 200 means cleanup completed. The UI warns about pending cleanup without asking the user to repeat an already-saved replacement upload. Metadata failures still return HTTP 500. The same owner may retry the same DELETE while the cleanup row exists, without recreating image metadata. Startup scans pending rows in batches of 100 and retries them; failures remain recorded and are logged without falsely claiming cleanup succeeded. Once cleanup is fully complete, later DELETE requests return 404.

This is deliberately not a general outbox, background scheduler, upload-orphan sweep, or production repair. There is no periodic retry; recovery is by the same DELETE or the next API startup. Existing upload and coin-delete workflows are unchanged.
