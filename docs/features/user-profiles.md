# User Profiles

> Customize your public profile with avatar, bio, and privacy settings.

## Overview

User profiles represent your identity in the Ancient Coins community. Customize how you appear to other collectors and control what information they can see.

## Profile Information

### Avatar
- **Upload Custom Avatar** — Change from the default Ed-Mar coin logo
- **Stored** — In `uploads/avatars/`
- **Formats** — JPEG, PNG, WebP
- **Size** — Recommended 200x200px for best quality

### Bio
- **Personal Description** — Up to 500 characters
- **Optional** — Leave blank if preferred
- **Displayed** — Public profile page when profile is public

### Username
- **Unique Identifier** — Cannot be changed after creation
- **Used For** — Social follows, mentions, search
- **Display** — On public profile and in follower lists

### Profile Status
- **Public** — Appear in user search, receive follow requests
- **Private** — Hidden from search, cannot receive follow requests
- **Toggle** — `Settings → Account`
- **Side Effect of Private** — All existing followers removed; they must re-follow if you go public again

## Accessing Profiles

### Your Profile
1. Go to **Settings → Account**
2. Upload avatar, edit bio
3. Toggle privacy status

### Other User's Profile
1. Search for user by username (Collection → Search button)
2. View public profile preview
3. If public, optionally send follow request

## Privacy Model

| Action | Public | Private |
|--------|--------|---------|
| Appear in search | ✅ Yes | ❌ No |
| Receive follow requests | ✅ Yes | ❌ No |
| Can be followed by anyone | ✅ Yes | ❌ No |
| Followers see collection | ✅ Yes | ✅ Yes (only accepted followers) |

## Profile Deletion

- Deleting your account also deletes your profile
- Followers lose access to your collection
- Comments and ratings remain but show "[deleted user]"

## API Endpoints

```
GET    /api/auth/me                  # Get current user profile/session
PUT    /api/user/profile             # Update bio, privacy, zip code, Coin of the Day preference
POST   /api/user/avatar              # Upload avatar
DELETE /api/user/avatar              # Delete avatar
GET    /api/users/search             # Search public users
GET    /api/users/:username          # Get another user's public profile
```

See also: [Social Features](social-features.md), [Admin Settings](admin-settings.md)
