# Social Feature

## Overview

This feature adds social capabilities to the coin application. Users can follow other collectors, view their coin galleries, and leave comments and star ratings on coins. Privacy controls let users manage who can see their collection.

## Original Requirements

The original feature request called for:

- New Follower Page that shows all your followers.
- Link to the Follower's Gallery Page from Followers Gallery. This provides a read only view of the gallery.
- Detail views of a Coin is limited to the coin images and essential details.
- Pricing/Value and AI analysis are not shown
- A user can rate or leave a comment on users they follow.
- The owner of the coin can remove any unwanted comments
- Followers show up with their Gravatar
- Users have the ability to upload a Gravatar in their settings page
- A default Gravatar would be the Ed-Mar coin that is currently used by this application
- A user can add followers from a button on the Followers page
- Followers are found by their username.
- The user account needs to be extended to include an email address for future requirements
- In PWA view the friend page is linked from the hamburger menu icon
- In PWA, the logout button will be moved from the Application icon to the hamburger menu icon
- In PWA, the Home Icon will be removed and going home will be invoked by clicking on the Application Icon (Ed-Mar)

## Implementation Details

The following sections describe what was actually implemented, noting differences from the original requirements where applicable.

### Follow Model & Workflow

The Follow model tracks relationships between users with a **Status** field that has three possible values:

| Status     | Description |
| ---------- | ----------- |
| `pending`  | Follow request has been sent but not yet accepted |
| `accepted` | The followed user accepted the request; the follower can view their gallery |
| `blocked`  | The followed user blocked the follower |

**Workflow:**

1. A user sends a follow request — status is set to `pending`.
2. The followed user reviews the request and either **accepts** or **blocks** it.
3. Only `accepted` followers can view the followed user's gallery.
4. **Blocked** users cannot re-request a follow unless the blocking user explicitly **unblocks** them.

> **Difference from original spec:** The original roadmap listed "block or reject followers" as a future item. This was implemented as part of the initial release with full accept/block/unblock support.

### Privacy & Visibility

- **Public/private profiles** — Users have an `isPublic` flag on their profile.
  - Only public users (`isPublic=true`) appear in user search results.
  - Only public users can receive follow requests.
  - **Setting a profile to private permanently deletes ALL existing followers.** This is a destructive action — followers are not merely hidden, they are removed.
- **Private coins** — Individual coins can be marked `isPrivate` to hide them from followers, even accepted ones.

### Gallery Access

Only **accepted followers** of **public users** can view the followed user's gallery. The gallery is read-only and limited to coin images and essential details (pricing/value and AI analysis are not shown), consistent with the original requirements.

The Followers page has two tabs: **Following** (users you follow) and **Followers** (users who follow you). Pending/blocked states are surfaced in the UI with appropriate action buttons.

Individual coins marked `isPrivate` are hidden from the follower gallery entirely, even for accepted followers.

### Comments & Star Ratings

- Accepted followers can leave **comments** on coins belonging to users they follow.
- Accepted followers can give a **star rating** (1–5) on coins.
- The **coin owner** can also comment on their own coins.
- Both the **commenter** and the **coin owner** can delete comments.
- Comments and ratings are returned together on the follower coin detail endpoint.
- **Private coins** (`isPrivate=true`) are blocked from follower access — comments and ratings cannot be left on private coins.

> **Difference from original spec:** Star ratings (1–5) were added beyond the original requirement of comments only.

### Avatar

- Users upload a custom avatar image (stored in `uploads/avatars/`).
- The default avatar is the Ed-Mar coin logo used throughout the application.

> **Difference from original spec:** The original spec referenced "Gravatar" (the third-party service). The implementation uses locally uploaded avatars instead, giving users direct control over their image without depending on an external service.

### Email

- Email is **required** for new user registrations.
- Legacy users (registered before this feature) are prompted with a **dismissible modal** to add their email address.
- The `GET /auth/me` endpoint includes an `emailMissing` flag so the frontend knows when to show the prompt.

### PWA Changes

The following PWA layout changes from the original requirements were implemented as specified:

- The follower page is linked from the hamburger menu icon.
- The logout button was moved from the Application icon to the hamburger menu icon.
- The Home Icon was removed; navigating home is done by clicking the Application Icon (Ed-Mar).

## Roadmap

- Exchange or Request to Buy

> **Note:** The notification system listed in the original roadmap has been implemented — see `src/api/services/notification_service.go` and the notifications page in the frontend.