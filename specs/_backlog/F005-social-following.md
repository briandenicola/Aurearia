---
title: Social Following and Activity Feed
id: F005
status: promoted
priority: P2
effort: L
value: 3
risk: 3
owner: Cassius+Aurelia
created: 2026-05-28
updated: 2026-05-28
---

## Summary

User follow/unfollow with follower request workflow (pending → accepted/blocked). Activity feed renders follower actions. Follow relationships scoped per-user; non-followers cannot browse collection. Comments and star ratings on followed coins.

## Acceptance Criteria

- When user searches for public users by username, only public (non-private) profiles appear
- When user sends follow request, relationship starts as pending; recipient can accept or block
- When request is accepted, follower can view recipient's coin collection (read-only; pricing/AI analysis hidden)
- When follower leaves comment or star rating, coin owner and commenter can each delete
- When user toggles profile to private, all existing followers are purged

## Constitution Alignment

**Principle V (Design System):** Public profiles use consistent card/chip styling; follower gallery uses read-only gallery view.
**Principle XI (Security Hardening):** Follow relationships enforce ownership boundaries; non-followers cannot access private collections.
**Principle VII (Error Handling):** Invalid follow transitions return descriptive HTTP 400 (e.g., "cannot block already-blocked user").

## Implementation Notes

- Follow relationship models: `Follower` (userID, followerID, status: pending/accepted/blocked)
- Endpoints: `POST /users/:id/follow`, `PUT /follows/:id/accept`, `DELETE /follows/:id`
- Activity feed: `GET /activity-feed` (paginated, scoped to followers)
- Comments table: `Coin.Comments` (coinID, userID, text, deletable by author or coin owner)
- Star ratings: `Coin.StarRatings` (coinID, userID, score 1-5, timestamp)

## Open Questions

None — feature shipped.

## Notes

Retroactive card created 2026-05-28 for governance traceability under Constitution §0 (Hierarchy) Phase 2. Follower gallery renders coin cards without value/AI; search indexes only public users via `isPublic` flag on User model.
