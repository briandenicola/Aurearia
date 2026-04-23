# API Reference

Complete REST API reference for Ancient Coins. All endpoints are served under the `/api` prefix (e.g., `http://localhost:8080/api/coins`). Interactive documentation is also available via Swagger UI at `/swagger/index.html`.

## Authentication

Most endpoints require authentication via one of two methods:

| Method | Header | Example |
| ------ | ------ | ------- |
| JWT Bearer Token | `Authorization: Bearer <token>` | Obtained from `/api/auth/login` |
| API Key | `X-API-Key: <key>` | Generated via `/api/auth/api-keys` |

To obtain a JWT token, call the [login](#post-apiauthlogin) endpoint. Tokens can be refreshed via the [refresh](#post-apiauthrefresh) endpoint. API keys are managed through the [API Keys](#api-keys) endpoints.

---

## Public Endpoints

These endpoints do not require authentication.

### GET /api/auth/setup

Check whether any users exist. Used by the frontend to determine if the app needs first-time setup.

**Response:**

```json
{ "needsSetup": true }
```

### POST /api/auth/register

Create a new user account. The **first user** to register is automatically assigned the **admin** role.

**Request Body:**

```json
{
  "username": "collector",
  "password": "s3cur3P@ss",
  "email": "collector@example.com"
}
```

| Field | Type | Required | Validation |
| ----- | ---- | -------- | ---------- |
| `username` | string | Yes | 3–50 characters |
| `password` | string | Yes | Minimum 8 characters |
| `email` | string | Yes | Must be a valid email address |

**Response:**

```json
{
  "token": "eyJhbGciOi...",
  "refreshToken": "dGhpcyBpcyBh...",
  "user": {
    "id": 1,
    "username": "collector",
    "role": "admin"
  }
}
```

### POST /api/auth/login

Authenticate and receive a JWT token.

**Request Body:**

```json
{
  "username": "collector",
  "password": "s3cur3P@ss"
}
```

**Response:**

```json
{
  "token": "eyJhbGciOi...",
  "refreshToken": "dGhpcyBpcyBh...",
  "user": {
    "id": 1,
    "username": "collector",
    "role": "admin"
  }
}
```

**Example:**

```sh
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "collector", "password": "s3cur3P@ss"}'
```

### POST /api/auth/refresh

Refresh an expired JWT using a refresh token.

**Request Body:**

```json
{ "refreshToken": "dGhpcyBpcyBh..." }
```

**Response:** Same shape as the login response (`token`, `refreshToken`, `user`).

### POST /api/auth/webauthn/login/begin

Begin a WebAuthn (biometric/passkey) login ceremony.

**Request Body:**

```json
{ "username": "collector" }
```

### POST /api/auth/webauthn/login/finish

Complete a WebAuthn login ceremony. The request body contains the authenticator assertion response from the browser WebAuthn API.

**Query Parameters:**

| Param | Description |
| ----- | ----------- |
| `username` | The username that initiated the ceremony |

**Response:** Same shape as the login response (`token`, `refreshToken`, `user`).

### GET /api/auth/webauthn/check

Check whether a user has registered biometric/passkey credentials.

**Query Parameters:**

| Param | Description |
| ----- | ----------- |
| `username` | The username to check |

**Response:**

```json
{ "hasCredentials": true }
```

---

## Protected Endpoints

All endpoints below require a valid `Authorization: Bearer <token>` or `X-API-Key` header.

---

### Coins

#### GET /api/coins

List coins in the collection with filtering, sorting, and pagination.

**Query Parameters:**

| Param | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `category` | string | — | Filter by category (`Roman`, `Greek`, `Byzantine`, `Modern`, `Other`) |
| `search` | string | — | Full-text search across name, ruler, denomination, and other fields |
| `wishlist` | string | — | `true` for wishlist only, `false` for collection only |
| `sort` | string | `createdAt` | Field name to sort by (e.g., `name`, `purchasePrice`, `currentValue`, `createdAt`) |
| `order` | string | `desc` | Sort direction: `asc` or `desc` |
| `page` | int | `1` | Page number (1-indexed) |
| `limit` | int | `50` | Results per page |

**Response:**

```json
{
  "coins": [ { "id": 1, "name": "Augustus Denarius", "..." : "..." } ],
  "total": 142,
  "page": 1,
  "limit": 50
}
```

**Example — search Roman coins sorted by value:**

```sh
curl "http://localhost:8080/api/coins?category=Roman&sort=currentValue&order=desc&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

#### GET /api/coins/:id

Get a single coin with all fields and associated images.

**Example:**

```sh
curl http://localhost:8080/api/coins/42 \
  -H "Authorization: Bearer $TOKEN"
```

#### POST /api/coins

Create a new coin. Only `name` is required; all other fields are optional. See the [field reference](getting-started.md#field-reference-for-import) for the full list.

**Request Body:**

```json
{
  "name": "Nero Sestertius - Port of Ostia",
  "category": "Roman",
  "material": "Bronze",
  "denomination": "Sestertius",
  "ruler": "Nero",
  "purchasePrice": 1200.00,
  "isWishlist": false
}
```

**Example:**

```sh
curl -X POST http://localhost:8080/api/coins \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Nero Sestertius", "category": "Roman", "material": "Bronze"}'
```

#### PUT /api/coins/:id

Update an existing coin. Send any fields you want to change.

**Request Body:** Same schema as create — only include the fields you want to update.

#### POST /api/coins/:id/purchase

Move a wishlist coin into the collection by setting `isWishlist` to `false`. Use this when you acquire a coin you were tracking on your wishlist.

#### POST /api/coins/:id/sell

Mark a coin as sold.

**Request Body:**

```json
{
  "salePrice": 750.00,
  "buyerName": "John Smith"
}
```

#### DELETE /api/coins/:id

Delete a coin and all of its associated images (both database records and uploaded files).

---

### Images

#### POST /api/coins/:id/images

Upload an image file via multipart form data.

**Form Fields:**

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `file` | file | Yes | The image file |
| `imageType` | string | No | One of: `obverse`, `reverse`, `detail`, `other` |
| `isPrimary` | string | No | `true` to set as the primary/cover image |

**Example:**

```sh
curl -X POST http://localhost:8080/api/coins/42/images \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@coin-obverse.jpg" \
  -F "imageType=obverse" \
  -F "isPrimary=true"
```

#### POST /api/coins/:id/images/base64

Upload an image as a base64-encoded string. Useful for programmatic uploads or when working with image data directly.

**Supported extensions:** `jpg`, `jpeg`, `png`, `gif`, `webp`, `bmp`, `tiff`

**Request Body:**

```json
{
  "data": "/9j/4AAQSkZJRg...",
  "fileExtension": "jpg",
  "imageType": "obverse",
  "isPrimary": true
}
```

#### DELETE /api/coins/:id/images/:imageId

Delete an image by ID. Removes both the database record and the uploaded file.

#### GET /api/proxy-image

Proxy an external image URL through the server. Useful to avoid CORS issues when displaying images from third-party sites.

**Query Parameters:**

| Param | Description |
| ----- | ----------- |
| `url` | The external image URL to proxy |

**Example:**

```sh
curl "http://localhost:8080/api/proxy-image?url=https://example.com/coin.jpg" \
  -H "Authorization: Bearer $TOKEN" \
  --output coin.jpg
```

#### GET /api/scrape-image

Scrape the `og:image` meta tag from a URL. Returns the image URL found in the page's OpenGraph metadata.

**Query Parameters:**

| Param | Description |
| ----- | ----------- |
| `url` | The page URL to scrape |

---

### Journal

Each coin can have journal entries for tracking research notes, provenance details, or condition observations over time.

#### GET /api/coins/:id/journal

List all journal entries for a coin.

#### POST /api/coins/:id/journal

Add a journal entry to a coin.

**Request Body:**

```json
{ "entry": "Identified mint mark as Rome based on comparison with RIC plates." }
```

**Example:**

```sh
curl -X POST http://localhost:8080/api/coins/42/journal \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"entry": "Cleaned with distilled water. No change to surfaces."}'
```

#### DELETE /api/coins/:id/journal/:entryId

Delete a journal entry by ID.

---

### AI Analysis

AI-powered coin analysis using Ollama vision models. Requires Ollama to be running and configured in admin settings.

#### POST /api/coins/:id/analyze

Run AI analysis on a coin's images. Analyzes either the obverse or reverse side.

**Query Parameters:**

| Param | Values | Description |
| ----- | ------ | ----------- |
| `side` | `obverse`, `reverse` | Which side to analyze |

**Example:**

```sh
curl -X POST "http://localhost:8080/api/coins/42/analyze?side=obverse" \
  -H "Authorization: Bearer $TOKEN"
```

#### DELETE /api/coins/:id/analyze

Delete a stored AI analysis for a coin.

**Query Parameters:**

| Param | Values | Description |
| ----- | ------ | ----------- |
| `side` | `obverse`, `reverse` | Which analysis to delete |

#### POST /api/extract-text

Extract text from an image using OCR. Accepts a multipart form upload.

**Form Fields:**

| Field | Type | Description |
| ----- | ---- | ----------- |
| `file` | file | The image file to extract text from |

**Example:**

```sh
curl -X POST http://localhost:8080/api/extract-text \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@inscription.jpg"
```

#### GET /api/ollama-status

Check Ollama connectivity and whether the configured vision model is available.

**Response:**

```json
{
  "connected": true,
  "model": "llava",
  "modelAvailable": true
}
```

---

### Statistics

#### GET /api/stats

Get collection statistics including totals, category breakdowns, material breakdowns, and grade distributions.

**Example:**

```sh
curl http://localhost:8080/api/stats \
  -H "Authorization: Bearer $TOKEN"
```

#### GET /api/value-history

Get portfolio value snapshots over time, useful for charting collection value trends.

#### GET /api/suggestions

Get autocomplete suggestions for coin fields (e.g., rulers, mints, denominations). Used by the frontend to power form autocomplete.

---

### Numista

#### GET /api/numista/search

Search the [Numista](https://en.numista.com/) coin catalog.

**Query Parameters:**

| Param | Description |
| ----- | ----------- |
| `q` | Search terms (e.g., `Augustus denarius`) |

**Example:**

```sh
curl "http://localhost:8080/api/numista/search?q=Augustus+denarius" \
  -H "Authorization: Bearer $TOKEN"
```

---

### AI Agent

Chat with an AI-powered coin search agent backed by Anthropic models. The agent can search your collection, answer numismatic questions, and provide research assistance.

#### POST /api/agent/chat

Send a message to the AI agent. The response is streamed via **Server-Sent Events (SSE)**.

**Request Body:**

```json
{
  "messages": [
    { "role": "user", "content": "What Roman denarii do I have from the Julio-Claudian dynasty?" }
  ],
  "conversationId": "optional-conversation-id"
}
```

**SSE Stream Format:**

The response uses `Content-Type: text/event-stream`. Each event is a JSON object:

```
data: {"type": "text", "content": "Based on your collection..."}

data: {"type": "text", "content": " I found 3 denarii..."}

data: {"type": "done"}
```

**Example:**

```sh
curl -N -X POST http://localhost:8080/api/agent/chat \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"messages": [{"role": "user", "content": "Show me my most valuable coins"}]}'
```

> **Note:** Use `curl -N` (no-buffer) to see streamed events in real time.

#### GET /api/agent/models

List available Anthropic models that can be used with the agent.

#### GET /api/agent/prompt

Get the current system prompt used by the AI agent.

---

### Conversations

Save and manage AI agent conversation history.

#### GET /api/agent/conversations

List all saved conversations.

#### GET /api/agent/conversations/:id

Get a saved conversation by ID, including the full message history.

#### POST /api/agent/conversations

Save a conversation.

**Request Body:**

```json
{
  "title": "Julio-Claudian Research",
  "messages": [
    { "role": "user", "content": "What coins do I have from Augustus?" },
    { "role": "assistant", "content": "You have 5 coins from Augustus..." }
  ]
}
```

#### DELETE /api/agent/conversations/:id

Delete a saved conversation.

---

### User

#### GET /api/auth/me

Get the current authenticated user's information.

**Response:**

```json
{
  "id": 1,
  "username": "collector",
  "role": "admin",
  "email": "user@example.com",
  "avatarPath": "avatars/user-1.jpg",
  "isPublic": false,
  "bio": "Ancient coin enthusiast",
  "emailMissing": false,
  "createdAt": "2024-01-01T00:00:00Z"
}
```

| Field | Type | Description |
| ----- | ---- | ----------- |
| `id` | int | User ID |
| `username` | string | Username |
| `role` | string | `user` or `admin` |
| `email` | string | Email address |
| `avatarPath` | string | Relative path to avatar image |
| `isPublic` | bool | Whether the user's profile is publicly visible |
| `bio` | string | User bio / description |
| `emailMissing` | bool | `true` if the account has no email set (legacy accounts) |
| `createdAt` | string | ISO 8601 creation timestamp |

#### POST /api/auth/change-password

Change the current user's password.

**Request Body:**

```json
{
  "currentPassword": "oldP@ss",
  "newPassword": "n3wS3cur3P@ss"
}
```

#### GET /api/user/export

Export the entire collection as a JSON file download. The response has `Content-Disposition: attachment` headers.

**Example:**

```sh
curl http://localhost:8080/api/user/export \
  -H "Authorization: Bearer $TOKEN" \
  --output my-collection.json
```

#### POST /api/user/import

Import coins from a JSON array. See the [Getting Started guide](getting-started.md#import--export) for the full field reference and import behavior.

**Request Body:** A JSON array of coin objects.

```sh
curl -X POST http://localhost:8080/api/user/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @my-coins.json
```

**Response:**

```json
{ "message": "Import complete", "imported": 42 }
```

---

### API Keys

Manage API keys for programmatic access. API keys are an alternative to JWT tokens and are useful for scripts, integrations, and automation.

#### POST /api/auth/api-keys

Generate a new API key. The full key is returned **only once** in the response — store it securely.

**Request Body:**

```json
{ "name": "My Integration" }
```

The `name` field is optional and helps you identify the key later.

**Response:**

```json
{
  "id": 1,
  "name": "My Integration",
  "key": "ak_abc123def456...",
  "prefix": "ak_abc1",
  "createdAt": "2024-03-15T10:30:00Z"
}
```

**Example:**

```sh
# Generate a key
curl -X POST http://localhost:8080/api/auth/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Backup Script"}'

# Use the key
curl http://localhost:8080/api/coins \
  -H "X-API-Key: ak_abc123def456..."
```

#### GET /api/auth/api-keys

List all API keys for the current user. Only the key prefix is shown (not the full key).

#### DELETE /api/auth/api-keys/:id

Revoke an API key. The key will immediately stop working.

---

### WebAuthn (Credential Registration)

Manage WebAuthn/passkey credentials for biometric login. Login ceremonies use the [public endpoints](#public-endpoints) above.

#### POST /api/auth/webauthn/register/begin

Begin a WebAuthn credential registration ceremony. Returns challenge data for the browser WebAuthn API.

#### POST /api/auth/webauthn/register/finish

Complete credential registration. The request body contains the authenticator attestation response from the browser WebAuthn API.

#### GET /api/auth/webauthn/credentials

List all registered WebAuthn credentials for the current user.

#### DELETE /api/auth/webauthn/credentials/:id

Delete a registered WebAuthn credential.

---

### User Profile

#### PUT /api/user/profile

Update the current user's profile. All fields are optional. **Note:** changing `isPublic` from `true` to `false` deletes all existing followers.

**Request Body:**

```json
{
  "email": "newemail@example.com",
  "bio": "Specializing in Roman Republican coins",
  "isPublic": true
}
```

**Response:** The updated user profile (same shape as [GET /api/auth/me](#get-apiauthme)).

#### POST /api/user/avatar

Upload a profile avatar via multipart form data.

**Form Fields:**

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `avatar` | file | Yes | Image file (`.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`) |

**Response:**

```json
{ "avatarPath": "avatars/user-1.jpg" }
```

#### DELETE /api/user/avatar

Delete the current user's avatar image.

**Response:**

```json
{ "message": "Avatar deleted" }
```

---

## Social

Social features for following other collectors, browsing their public coins, and leaving comments and ratings.

All social endpoints require authentication.

---

### Follow Management

#### POST /api/social/follow/:userId

Send a follow request to another user. The target user must have a public profile.

**Response (201):**

```json
{ "message": "Follow request sent" }
```

**Errors:**

| Status | Condition |
| ------ | --------- |
| `400` | Attempting to follow yourself |
| `403` | Target user is not public or has blocked you |
| `409` | Already following or a pending request exists |

#### DELETE /api/social/follow/:userId

Unfollow a user. Removes pending or accepted follow records (does not affect blocks).

**Response:**

```json
{ "message": "Unfollowed user" }
```

#### PUT /api/social/followers/:userId/accept

Accept a pending follow request from another user.

**Response:**

```json
{ "message": "Follower accepted" }
```

#### PUT /api/social/followers/:userId/block

Block a user. Creates or updates the follow record to blocked status.

**Response:**

```json
{ "message": "User blocked" }
```

#### DELETE /api/social/followers/:userId/block

Unblock a previously blocked user. Removes the blocked follow record.

**Response:**

```json
{ "message": "User unblocked" }
```

#### GET /api/social/followers

List users who follow you (both pending and accepted).

**Response:**

```json
{
  "followers": [
    {
      "id": 2,
      "username": "numismatist",
      "avatarPath": "avatars/user-2.jpg",
      "isPublic": true,
      "bio": "Greek coin specialist",
      "status": "accepted"
    }
  ]
}
```

| Field | Type | Description |
| ----- | ---- | ----------- |
| `status` | string | `"pending"` or `"accepted"` |

#### GET /api/social/following

List users you are following (accepted only).

**Response:**

```json
{
  "following": [
    {
      "id": 3,
      "username": "romanfan",
      "avatarPath": "avatars/user-3.jpg",
      "isPublic": true,
      "bio": "Imperial Rome enthusiast",
      "isFollowing": true,
      "coinCount": 87
    }
  ]
}
```

#### GET /api/social/blocked

List users you have blocked.

**Response:**

```json
{
  "blocked": [
    {
      "id": 5,
      "username": "spammer",
      "avatarPath": ""
    }
  ]
}
```

---

### User Discovery

#### GET /api/users/search

Search for public users by username prefix.

**Query Parameters:**

| Param | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `q` | string | Yes | Search query (minimum 2 characters) |

**Response:**

```json
{
  "users": [
    {
      "id": 3,
      "username": "romanfan",
      "avatarPath": "avatars/user-3.jpg",
      "isPublic": true,
      "bio": "Imperial Rome enthusiast",
      "isFollowing": true,
      "followStatus": "accepted",
      "coinCount": 87
    }
  ]
}
```

| Field | Type | Description |
| ----- | ---- | ----------- |
| `isFollowing` | bool | Whether you currently follow this user |
| `followStatus` | string | `""`, `"pending"`, `"accepted"`, or `"blocked"` |
| `coinCount` | int | Number of coins in the user's collection |

#### GET /api/users/:username

Get a user's public profile by username.

**Response:**

```json
{
  "id": 3,
  "username": "romanfan",
  "avatarPath": "avatars/user-3.jpg",
  "isPublic": true,
  "bio": "Imperial Rome enthusiast",
  "isFollowing": true,
  "followStatus": "accepted",
  "coinCount": 87,
  "followerCount": 12,
  "followingCount": 5
}
```

---

### Follower Gallery

#### GET /api/social/following/:userId/coins

Get a followed user's public coin collection. Requires an accepted follow relationship and the user must have a public profile.

**Response:**

```json
{
  "coins": [
    {
      "id": 10,
      "name": "Augustus Denarius",
      "category": "Roman",
      "denomination": "Denarius",
      "ruler": "Augustus",
      "era": "27 BC - 14 AD",
      "material": "Silver",
      "grade": "VF",
      "images": []
    }
  ],
  "username": "romanfan"
}
```

> **Note:** Only a limited set of coin fields is returned (no purchase price, sale info, or private notes).

#### GET /api/social/following/:userId/coins/:coinId

Get a single coin's details from a followed user's collection, including comments and ratings.

**Response:** Same limited coin fields as the list endpoint, plus:

```json
{
  "comments": [
    {
      "id": 1,
      "userId": 2,
      "username": "numismatist",
      "avatarPath": "avatars/user-2.jpg",
      "comment": "Beautiful example!",
      "rating": 5,
      "createdAt": "2024-06-01T12:00:00Z"
    }
  ],
  "rating": {
    "average": 4.5,
    "count": 2,
    "userRating": 5
  }
}
```

---

### Comments & Ratings

#### POST /api/social/coins/:coinId/comments

Add a comment to a coin. You must have an accepted follow relationship with the coin's owner, or be the owner yourself.

**Request Body:**

```json
{
  "comment": "Excellent patina on this piece!",
  "rating": 5
}
```

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `comment` | string | Yes | Comment text |
| `rating` | int | No | Star rating, 0–5 (0 means no rating) |

**Response (201):** The created comment enriched with user info:

```json
{
  "id": 1,
  "userId": 2,
  "username": "numismatist",
  "avatarPath": "avatars/user-2.jpg",
  "comment": "Excellent patina on this piece!",
  "rating": 5,
  "createdAt": "2024-06-01T12:00:00Z"
}
```

#### GET /api/social/coins/:coinId/comments

Get all comments on a coin.

**Response:**

```json
{
  "comments": [
    {
      "id": 1,
      "userId": 2,
      "username": "numismatist",
      "avatarPath": "avatars/user-2.jpg",
      "comment": "Excellent patina on this piece!",
      "rating": 5,
      "createdAt": "2024-06-01T12:00:00Z"
    }
  ]
}
```

#### DELETE /api/social/coins/:coinId/comments/:commentId

Delete a comment. Only the comment author or the coin owner may delete a comment.

#### PUT /api/social/coins/:coinId/rating

Create or update your star rating for a coin.

**Request Body:**

```json
{ "rating": 4 }
```

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `rating` | int | Yes | Star rating, 1–5 |

**Response:**

```json
{
  "average": 4.5,
  "count": 3,
  "userRating": 4
}
```

#### GET /api/social/coins/:coinId/rating

Get the aggregate rating for a coin.

**Response:**

```json
{
  "average": 4.5,
  "count": 3,
  "userRating": 4
}
```

| Field | Type | Description |
| ----- | ---- | ----------- |
| `average` | float | Average star rating across all users |
| `count` | int | Total number of ratings |
| `userRating` | int | Your rating (0 if you haven't rated) |

---

## Admin Endpoints

These endpoints require the **admin** role. Regular users will receive a `403 Forbidden` response.

### GET /api/admin/users

List all registered users.

### DELETE /api/admin/users/:id

Delete a user account.

### POST /api/admin/users/:id/reset-password

Reset a user's password (admin override).

**Request Body:**

```json
{ "newPassword": "t3mpP@ss" }
```

### GET /api/admin/settings

Get all application settings.

### GET /api/admin/settings/defaults

Get the default values for all settings.

### PUT /api/admin/settings

Update one or more settings.

**Request Body:**

```json
[
  { "key": "ollama_url", "value": "http://localhost:11434" },
  { "key": "ollama_model", "value": "llava" },
  { "key": "log_level", "value": "info" }
]
```

**Example:**

```sh
curl -X PUT http://localhost:8080/api/admin/settings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '[{"key": "ollama_url", "value": "http://ollama:11434"}]'
```

### GET /api/admin/logs

Get application logs with optional filtering.

**Query Parameters:**

| Param | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `level` | string | — | Filter by log level (`trace`, `debug`, `info`, `warn`, `error`) |
| `limit` | int | `100` | Number of log entries to return |

**Example:**

```sh
curl "http://localhost:8080/api/admin/logs?level=error&limit=50" \
  -H "Authorization: Bearer $TOKEN"
```

### GET /api/admin/availability-runs

List availability check run history (paginated).

**Query Parameters:**

| Param | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `page` | int | `1` | Page number |
| `limit` | int | `20` | Results per page |

**Response:**

```json
{
  "runs": [
    {
      "id": 1,
      "userId": 1,
      "triggerType": "manual",
      "coinsChecked": 5,
      "available": 3,
      "unavailable": 1,
      "unknown": 1,
      "errors": 0,
      "durationMs": 4200,
      "startedAt": "2026-04-13T02:00:00Z",
      "completedAt": "2026-04-13T02:00:04Z"
    }
  ],
  "total": 1
}
```

### GET /api/admin/availability-runs/:id

Get details for a single availability check run, including per-coin results.

**Response:** Same as above with an additional `results` array containing per-coin outcomes (coinId, coinName, url, status, reason, httpStatus, agentUsed).

---

### POST /api/wishlist/check-availability

Trigger a manual availability check for the authenticated user's wishlist coins. Checks each coin's reference URL via HTTP + keyword heuristics, escalating ambiguous results to the AI agent.

**Response:**

```json
{
  "runId": 42,
  "coinsChecked": 5,
  "available": 3,
  "unavailable": 1,
  "unknown": 1,
  "durationMs": 4200
}
```

### PUT /api/coins/:id/listing-status

Update a coin's listing status (e.g., to dismiss an unavailable flag).

**Request Body:**

```json
{ "status": "" }
```

---

## Static Resources

| Path | Description |
| ---- | ----------- |
| `/swagger/index.html` | Interactive Swagger UI API documentation |
| `/uploads/*` | Uploaded coin images (served as static files) |

---

## Common Patterns

### Pagination

List endpoints support pagination via `page` and `limit` query parameters. The default page size is **50**.

```sh
# Get page 3 with 20 results per page
curl "http://localhost:8080/api/coins?page=3&limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

The response includes `total`, `page`, and `limit` fields so clients can calculate the total number of pages.

### Sorting

List endpoints support sorting via `sort` and `order` query parameters.

```sh
# Sort by purchase price, highest first
curl "http://localhost:8080/api/coins?sort=purchasePrice&order=desc" \
  -H "Authorization: Bearer $TOKEN"
```

### Error Responses

Errors return an appropriate HTTP status code with a JSON body:

```json
{ "error": "coin not found" }
```

| Status | Meaning |
| ------ | ------- |
| `400` | Bad request (invalid input) |
| `401` | Unauthorized (missing or invalid token/key) |
| `403` | Forbidden (insufficient permissions) |
| `404` | Resource not found |
| `500` | Internal server error |

---

## Auction Lots

All auction lot endpoints require authentication. Lots are scoped to the authenticated user.

### GET /api/auctions

List auction lots with optional filtering.

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `status` | string | Filter by status: `watching`, `bidding`, `won`, `lost`, `passed` |
| `search` | string | Full-text search across title, description, auction house |
| `sort` | string | Sort field (default: `createdAt`) |
| `order` | string | `asc` or `desc` (default: `desc`) |
| `page` | int | Page number (default: 1) |
| `limit` | int | Results per page (default: 50) |

### GET /api/auctions/:id

Get a single auction lot by ID.

### POST /api/auctions

Create a new auction lot.

### PUT /api/auctions/:id

Update an auction lot's fields.

### PUT /api/auctions/:id/status

Update a lot's status. Validates allowed transitions (e.g., only Bidding can become Won).

**Body:** `{ "status": "won" }`

### POST /api/auctions/:id/convert

Convert a won lot into a coin in the user's collection. Only works when status is `won`. Returns the newly created coin.

### DELETE /api/auctions/:id

Delete an auction lot.

### POST /api/auctions/import

Import a lot from a NumisBids URL. Accepts scraped data from the frontend.

**Body:** `{ "url": "https://www.numisbids.com/sale/123/lot/45", "title": "...", "imageUrl": "...", ... }`

### POST /api/auctions/sync

Sync the user's NumisBids watchlist. Requires NumisBids credentials configured in user settings. Logs into NumisBids, fetches the watchlist page, parses lots, and upserts them. Returns the number synced and the lot objects.

**Response:** `{ "synced": 5, "lots": [...] }`

### POST /api/auctions/validate-credentials

Validate NumisBids credentials by attempting a login. Does not save anything.

**Body:** `{ "username": "user@example.com", "password": "..." }`

**Response:** `{ "valid": true }` or `{ "valid": false, "error": "Login failed. Check your credentials." }`

---

## Tags

Manage user-defined tags for organizing coins. All tag endpoints require authentication.

### GET /api/tags

List all tags for the authenticated user.

**Response:**

```json
{
  "tags": [
    { "id": 1, "name": "Favorites", "color": "#e5a00d" },
    { "id": 2, "name": "For Sale", "color": "#3b82f6" }
  ]
}
```

### POST /api/tags

Create a new tag.

**Request Body:**

```json
{ "name": "Favorites", "color": "#e5a00d" }
```

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `name` | string | Yes | Tag display name |
| `color` | string | No | Hex color code for the tag badge |

**Response (201):** The created tag object.

### PUT /api/tags/:id

Update a tag's name or color.

**Request Body:**

```json
{ "name": "Top Picks", "color": "#f59e0b" }
```

### DELETE /api/tags/:id

Delete a tag. Removes the tag from all coins it was attached to.

### POST /api/coins/:id/tags

Attach a tag to a coin.

**Request Body:**

```json
{ "tagId": 1 }
```

### DELETE /api/coins/:id/tags/:tagId

Detach a tag from a coin.

---

## Bulk Operations

### POST /api/coins/bulk

Perform a bulk action on a set of coins. Requires authentication.

**Request Body:**

```json
{
  "action": "tag",
  "coinIds": [1, 2, 3],
  "tagId": 1
}
```

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `action` | string | Yes | One of `delete`, `tag`, `untag`, `status` |
| `coinIds` | int[] | Yes | Array of coin IDs to act on |
| `tagId` | int | Conditional | Required when `action` is `tag` or `untag` |

**Response:**

```json
{ "message": "Bulk action completed", "affected": 3 }
```

---

## Notifications

All notification endpoints require authentication and are scoped to the authenticated user.

### GET /api/notifications

List notifications for the authenticated user.

**Response:**

```json
{
  "notifications": [
    {
      "id": 1,
      "type": "follow_request",
      "message": "numismatist wants to follow you",
      "read": false,
      "createdAt": "2024-06-01T12:00:00Z"
    }
  ]
}
```

### GET /api/notifications/unread-count

Get the number of unread notifications.

**Response:**

```json
{ "count": 3 }
```

### PUT /api/notifications/:id/read

Mark a single notification as read.

**Response:**

```json
{ "message": "Notification marked as read" }
```

### PUT /api/notifications/read-all

Mark all notifications as read for the authenticated user.

**Response:**

```json
{ "message": "All notifications marked as read" }
```

### DELETE /api/notifications/:id

Delete a notification.

---

## Showcases

Curated collections of coins that can be published with a shareable URL. All endpoints except the public slug endpoint require authentication.

### GET /api/showcases

List showcases for the authenticated user.

**Response:**

```json
{
  "showcases": [
    {
      "id": 1,
      "name": "Roman Imperial Highlights",
      "description": "My best imperial coins",
      "slug": "roman-imperial-highlights",
      "isPublished": true,
      "coinCount": 8,
      "createdAt": "2024-06-01T12:00:00Z"
    }
  ]
}
```

### GET /api/showcases/:id

Get a showcase with its coins.

**Response:**

```json
{
  "id": 1,
  "name": "Roman Imperial Highlights",
  "description": "My best imperial coins",
  "slug": "roman-imperial-highlights",
  "isPublished": true,
  "coins": [ { "id": 10, "name": "Augustus Denarius", "..." : "..." } ],
  "createdAt": "2024-06-01T12:00:00Z"
}
```

### POST /api/showcases

Create a new showcase.

**Request Body:**

```json
{
  "name": "Roman Imperial Highlights",
  "description": "My best imperial coins"
}
```

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `name` | string | Yes | Showcase title |
| `description` | string | No | Short description |

**Response (201):** The created showcase object.

### PUT /api/showcases/:id

Update a showcase's metadata or publish status.

**Request Body:**

```json
{
  "name": "Updated Title",
  "description": "Updated description",
  "isPublished": true
}
```

### DELETE /api/showcases/:id

Delete a showcase. Does not delete the coins themselves.

### PUT /api/showcases/:id/coins

Set the coins included in a showcase. Replaces the existing coin list.

**Request Body:**

```json
{ "coinIds": [10, 22, 35] }
```

**Response:**

```json
{ "message": "Showcase coins updated" }
```

---

### Public Showcase

#### GET /api/showcase/:slug

Get a published showcase by its URL slug. **No authentication required.**

Returns `404` if the showcase does not exist or is not published.

**Response:** Same shape as [GET /api/showcases/:id](#get-apishowcasesid) (without private fields).

---

## Calendar

View auction lot dates and manage custom calendar events. All endpoints require authentication.

### GET /api/calendar

Get calendar data for a date range. Returns auction lots and custom events that fall within the range.

**Query Parameters:**

| Param | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `start` | string | Yes | Range start (ISO 8601 date, e.g., `2024-06-01`) |
| `end` | string | Yes | Range end (ISO 8601 date, e.g., `2024-06-30`) |

**Response:**

```json
{
  "lots": [
    { "id": 1, "title": "Augustus Aureus", "date": "2024-06-15T14:00:00Z", "auctionHouse": "CNG" }
  ],
  "events": [
    { "id": 1, "title": "Coin Show - NYC", "description": "Annual NYINC show", "date": "2024-06-20T09:00:00Z", "url": "https://example.com/show" }
  ]
}
```

### POST /api/calendar/events

Create a custom calendar event.

**Request Body:**

```json
{
  "title": "Coin Show - NYC",
  "description": "Annual NYINC show",
  "date": "2024-06-20T09:00:00Z",
  "url": "https://example.com/show"
}
```

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `title` | string | Yes | Event title |
| `description` | string | No | Event description |
| `date` | string | Yes | ISO 8601 date/time |
| `url` | string | No | Related URL |

**Response (201):** The created event object.

### PUT /api/calendar/events/:id

Update a custom calendar event.

**Request Body:** Same schema as create — only include the fields you want to update.

### DELETE /api/calendar/events/:id

Delete a custom calendar event.

---

## Price Alerts

Set alerts for auction lots to be notified when prices reach a target. All endpoints require authentication.

### GET /api/alerts

List price alerts for the authenticated user.

**Response:**

```json
{
  "alerts": [
    {
      "id": 1,
      "lotId": 42,
      "targetPrice": 500.00,
      "direction": "below",
      "createdAt": "2024-06-01T12:00:00Z"
    }
  ]
}
```

### POST /api/alerts

Create a price alert.

**Request Body:**

```json
{
  "lotId": 42,
  "targetPrice": 500.00,
  "direction": "below"
}
```

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `lotId` | int | Yes | The auction lot to monitor |
| `targetPrice` | float | Yes | Target price threshold |
| `direction` | string | Yes | `above` or `below` |

**Response (201):** The created alert object.

### DELETE /api/alerts/:id

Delete a price alert.

---

## Bid Reminders

Set reminders to be notified before auction lot bidding closes. All endpoints require authentication.

### GET /api/reminders

List bid reminders for the authenticated user.

**Response:**

```json
{
  "reminders": [
    {
      "id": 1,
      "lotId": 42,
      "minutesBefore": 30,
      "createdAt": "2024-06-01T12:00:00Z"
    }
  ]
}
```

### POST /api/reminders

Create a bid reminder.

**Request Body:**

```json
{
  "lotId": 42,
  "minutesBefore": 30
}
```

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `lotId` | int | Yes | The auction lot to set a reminder for |
| `minutesBefore` | int | Yes | Minutes before lot closes to trigger the reminder |

**Response (201):** The created reminder object.

### DELETE /api/reminders/:id

Delete a bid reminder.

---

## PDF Export

### GET /api/user/export/catalog

Download a PDF catalog of the authenticated user's coin collection. The PDF includes photos, grades, provenance, and valuations for each coin.

**Response:** `Content-Type: application/pdf` with `Content-Disposition: attachment` headers.

**Example:**

```sh
curl http://localhost:8080/api/user/export/catalog \
  -H "Authorization: Bearer $TOKEN" \
  --output my-catalog.pdf
```
