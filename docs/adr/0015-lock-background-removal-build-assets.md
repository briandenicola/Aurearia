# ADR 0015: Lock Background-Removal Build Assets

Date: 2026-08-31
Status: Accepted

## Context

The production Docker build runs
`src/web/scripts/download-background-removal-assets.mjs` before building the
Vue application. The script previously fetched `resources.json` and every
required JavaScript, WebAssembly, and model chunk directly from
`staticimgly.com` without an independently trusted integrity value.

That made the CDN response part of the trusted build input. A compromised
origin, cache, or upstream release could change executable browser assets,
redirect a chunk request, or supply a chunk name that escaped the intended
output directory. npm's lockfile did not protect these files because they are
not included in the `@imgly/background-removal` package tarball.

ADR 0014 limits the runtime privileges of the background-removal worker, but
it does not establish the provenance of the worker's separately downloaded
WASM and model assets.

## Decision

Commit the exact required `@imgly/background-removal` 1.7.0 resource metadata
to `src/web/scripts/background-removal-assets.lock.json` and treat that file,
not the live CDN manifest, as the build contract.

The downloader:

- requires the npm dependency to use an exact semantic version matching the
  lockfile;
- permits exactly the three resources used by the application's CPU /
  `isnet_quint8` configuration;
- accepts only content-addressed, lowercase SHA-256 chunk names whose declared
  hash matches the name;
- fixes the source to the reviewed HTTPS origin and versioned path, rejects
  redirects, and prevents absolute, nested, or traversal paths;
- reads each response with the locked byte count as a hard upper bound, then
  verifies the final size and SHA-256 digest;
- writes verified chunks to a fresh staging directory and replaces the
  generated output only after every chunk succeeds.

Any future `@imgly/background-removal` upgrade must update the package version
and regenerate and review the lockfile in the same change. A mismatch fails
the build rather than falling back to live metadata.

Continuing to trust `resources.json` was rejected because a hash supplied by
the same compromised response does not provide independent integrity.
Committing the roughly 56 MB generated asset set was also rejected as
unnecessary: the small reviewed lockfile gives immutable content identities
without storing generated binaries in Git.

## Consequences

Production builds remain network-dependent, but the CDN can now affect
availability only. It cannot silently change a shipped chunk without a
reviewed lockfile change.

Builds fail closed when the package version, required resource set, origin,
redirect behavior, byte size, digest, or output path differs from the
committed contract. Focused Node tests cover each guard, including a full
staged-publication path, and are part of `npm test`.

The lockfile requires deliberate maintenance on dependency upgrades. If IMG.LY
changes its resource naming or hosting scheme, that change must be reviewed
rather than being accepted automatically.

This decision implements Constitution Principle V (secure defaults),
Principle VII (supply-chain and release integrity), Principle VIII (documented
security decisions), Principle IX (automated enforcement), and the Quality
Gate in §17.
