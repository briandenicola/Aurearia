# Private original attribution cards

Owner decisions: retain uploaded originals now; include them as separate files
alongside the generated share card only after a Share prompt; leave sale
recording unchanged. Manual Add, Edit and confirmed AI intake are in scope.

Implementation reuses the existing Go-owned image store with non-primary type
`card`. Ordinary photo queries, follower/public showcase media, visual catalogs,
tray/Quick Access and face-analysis inputs exclude it. The full owner collection
ZIP backup retains it. No new table, migration, Python state or provider is added.
Existing camera/library preparation remains; stored uploaded bytes are preserved.

See the [feature contract](../../../docs/attribution-card-retention.md) and
[Proposed ADR 0023](../../../docs/adr/0023-private-attribution-card-images.md).
The ADR remains Proposed; source preparation and beta push do not imply owner
acceptance, main merge, release or deployment. Downgrading to an API without the
new privacy filters after retaining cards requires a separate safe recovery plan.

Execution approval: focused local checks, one restricted independent reviewer
with a ten-minute/40-read lease, and beta push when ready. Full gates on beta CI.
No local application builds, installations or containers.
