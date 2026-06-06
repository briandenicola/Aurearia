# Social Features

> Follow other collectors, engage with their collections, and build a community around ancient coin collecting.

## Overview

Social features enable collectors to follow each other, comment on coins, rate collections, and build relationships within the community while maintaining privacy controls.

## Core Features

### Follow / Unfollow
- **Send Requests** — Request to follow other public users
- **Pending Status** — Requests start as pending until accepted
- **Accept / Block** — Review incoming follow requests
- **Block Users** — Blocked users cannot re-request unless unblocked
- **Follower List** — See who follows you

### Follower Gallery
- **Read-Only Access** — View accepted follower's coins
- **Hidden Information** — Pricing, values, and AI analysis are hidden from followers
- **Public Metadata** — View coin name, images, ruler, denomination, era, category

### Comments & Ratings
- **Leave Comments** — Add text comments on coins in follower's collection
- **Star Ratings** — Rate coins 1-5 stars
- **Delete Comments** — Both commenter and coin owner can delete
- **Comment Threads** — View all comments on a coin

### User Search & Discovery
- **Username Search** — Find other collectors by username
- **Public Users Only** — Only users with public profiles appear in search
- **View Public Profile** — See public user info and coin count before following

### Privacy Controls
- **Public/Private Profile Toggle** — Control if you appear in search and receive follow requests
- **Private Coins** — Mark individual coins as private to hide from followers
- **Setting Profile Private** — Removes all existing followers permanently
- **Re-Enable Public** — Public status can be re-enabled anytime

## User Profiles

### Public Profile Information
- **Avatar** — Custom profile picture
- **Username** — Unique identifier
- **Bio** — Personal description (up to 500 chars)
- **Follower Count** — Number of followers
- **Collection Size** — Coin count (for public users)

### Profile Customization
1. Go to **Settings → Account**
2. Upload avatar or use default
3. Add/edit bio
4. Toggle public/private status

## Notifications

Social interactions generate notifications:
- Follow requests pending
- Follows accepted
- New comments on your coins
- New star ratings
- Follower requests (if you block someone)

See [Notifications](notifications.md) for full details.

## API Endpoints

```
POST   /api/social/follow/:userId    # Send follow request
PUT    /api/social/followers/:userId/accept # Accept follow request
PUT    /api/social/followers/:userId/block  # Block user
DELETE /api/social/followers/:userId/block  # Unblock user
DELETE /api/social/follow/:userId    # Unfollow user
GET    /api/social/followers         # List my followers
GET    /api/social/following         # List who I follow
GET    /api/social/blocked           # List blocked users

POST   /api/social/coins/:coinId/comments # Add comment
GET    /api/social/coins/:coinId/comments # List comments
DELETE /api/social/coins/:coinId/comments/:commentId # Delete comment
PUT    /api/social/coins/:coinId/rating   # Rate coin 1-5
GET    /api/social/coins/:coinId/rating   # Get my rating

GET    /api/users/search             # Search public users
GET    /api/users/:username          # Get public profile
```

## Related Features

- [User Profiles](user-profiles.md) — Profile customization
- [Collection Showcase](collection-showcase.md) — Share curated subsets
- [Notifications](notifications.md) — Social interaction alerts

## See Also

- [Social Feature Specification](../social-feature.md) — Full technical details
