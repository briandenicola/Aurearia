# ADR 0023: Private Original Attribution-Card Images

**Status:** Proposed
**Date:** 2026-10-17

## Context

Card OCR currently retains extracted values, not the source photograph. The
owner wants to retain the original privately and choose whether to attach it
alongside the generated Share card. Ordinary coin photos may be visible to
accepted followers or through public showcases; an attribution card can contain
private dealer, pricing or handwritten information.

## Decision

Extend the existing Go-owned image store with image type `card`, never primary.
Save uploaded bytes unchanged through the existing upload endpoint, without
coin-face clipping. Cards are excluded from ordinary photo reads and public
media authorization, regardless of coin/profile visibility. A separate
owner-authorized endpoint lists them for explicit Share use.

Manual Add, Edit and confirmed AI intake upload originals before OCR-dependent
completion. The stateless Python service and draft persistence do not change.
Share reuses the shared renderer and authenticated media reader, asking before
attaching all originals as separate files. Browser fallback requests separate
downloads and explains the browser's multiple-download permission.

No new table or schema migration is needed: `coin_images.image_type` is already
a string. Owner export and existing image deletion/cleanup retain their current
semantics. Sale recording and image-free duplication are unchanged.

## Consequences

Existing discarded originals cannot be recovered. Original attachments can
contain private information, so inclusion is opt-in on each Share operation.
Existing camera/library preparation still precedes upload; stored upload bytes
are preserved, not regenerated. Owner-only listing adds one request to Share.
Full photo-query exclusion and owner/public-media negative coverage are required.

This proposed record documents the owner-authorized implementation; it does not
declare policy acceptance, release authorization or a constitutional amendment.

## Related

- [Feature contract](../attribution-card-retention.md)
- Constitution Principles I, III, V and IX; sections 17-21
