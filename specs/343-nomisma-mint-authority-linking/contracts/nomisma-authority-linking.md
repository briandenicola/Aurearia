# Contract: Nomisma Authority Linking (Admin API)

All routes below sit under the existing admin-only mint-location group in
`main.go` (same middleware chain as `POST/PUT/DELETE /admin/mint-locations`).
Non-admin users receive the existing 403 handling; there is no non-admin
variant of any of these three routes (FR-014). None of these routes are
reachable for a private (`UserID != nil`) mint location — they all resolve
the target mint via the same `FindByID` + global-only guard used by
`UpdateGlobal`/`DeleteGlobal` and return `404 Mint location not found`
otherwise (FR-006).

## 1. Search Nomisma candidates

```
GET /admin/mint-locations/{id}/nomisma/search?query={text}
```

- **Auth**: Bearer token, admin role required.
- **Path param**: `id` — the global `MintLocation` ID being curated. (Included
  so future rate-limiting/observability can scope by mint even though the
  query text itself is what's sent to Nomisma; the mint's own name/aliases
  are never auto-injected into the query without the admin typing it — the
  admin explicitly provides `query`, consistent with `Geocode`'s existing
  contract of "only the typed name is sent.")
- **Query param**: `query` (required, non-blank after trim; server-side
  validated before any Nomisma call — max length e.g. 200 chars).
- **Response 200**:
  ```json
  {
    "status": "ok",
    "candidates": [
      { "uri": "http://nomisma.org/id/roma", "label": "Roma", "score": 100, "match": true }
    ]
  }
  ```
  `status` is one of `ok`, `no_match`, `unavailable` (see data-model.md's
  `NomismaSearchOutcome`). `candidates` is always present (possibly empty)
  so the frontend never has to special-case a missing field.
- **Response 400**: `query` missing/blank/too long — `ErrorResponse` shape
  matching existing handlers (`{"error": "..."}`).
- **Response 404**: `id` does not resolve to a global mint location (either
  it doesn't exist, or it is a private mint — both return the same generic
  404, so the response never confirms whether a given ID belongs to another
  user's private mint).
- **Never returns 5xx for a Nomisma-side failure** — an upstream timeout or
  error is surfaced as `200 { "status": "unavailable", "candidates": [] }`,
  per FR-007 ("non-blocking, clearly surfaced 'lookup unavailable' state").
- **Swagger**: annotated on the new `MintLocationHandler.SearchNomisma`
  method, tag `Mint Locations`, consistent with existing handler doc
  comments in `mint_location.go`.

## 2. Confirm (link) a Nomisma candidate

```
POST /admin/mint-locations/{id}/nomisma
```

- **Auth**: Bearer token, admin role required.
- **Body**:
  ```json
  { "uri": "http://nomisma.org/id/roma", "label": "Roma" }
  ```
  Both fields required, non-blank, server-validated (`uri` must parse as an
  absolute `http`/`https` URL under the `nomisma.org` host — reject anything
  else so the persisted "durable concept URI" can't be an arbitrary string;
  `label` bounded to the same 256-char column limit as other display
  fields).
- **Behavior**: This is an explicit admin action — it is never invoked as a
  side effect of `Search`. It replaces any existing link if one is present
  (User Story 2, Scenario 1): sets `NomismaURI`, `NomismaLabel`,
  `NomismaLinkedAt = now()` in one `repo.Update` call touching only those
  three columns — `DisplayName`, `Lat`, `Lng`, `Region`, `Aliases` are never
  part of this update's column set (FR-005).
- **Response 200**: the updated `models.MintLocation` (same shape as the
  existing `PUT /admin/mint-locations/{id}` response), now including the
  three new fields.
- **Response 400**: invalid/missing `uri`/`label`.
- **Response 404**: same as above (unknown or private mint ID).
- **Response 500**: unexpected persistence failure (existing
  `handleMintLocationError` default case).

## 3. Remove a Nomisma link

```
DELETE /admin/mint-locations/{id}/nomisma
```

- **Auth**: Bearer token, admin role required.
- **Behavior**: Clears `NomismaURI`, `NomismaLabel`, `NomismaLinkedAt` to
  nil/empty in one `repo.Update` call, touching only those columns
  (User Story 2, Scenario 2). Idempotent — unlinking an already-unlinked
  mint location is a no-op success, not an error, so a slow admin
  double-click can't produce a confusing failure.
- **Response 200**: `MessageResponse{ "message": "Nomisma link removed" }`,
  matching the existing `Delete`/`DeletePrivate` response shape.
- **Response 404**: same as above.

---

## Frontend contract (`src/web/src/api/client.ts`)

```ts
export type NomismaCandidate = { uri: string; label: string; score: number; match: boolean }
export type NomismaSearchStatus = 'ok' | 'no_match' | 'unavailable'
export type NomismaSearchResponse = { status: NomismaSearchStatus; candidates: NomismaCandidate[] }

export const searchNomismaMintCandidates = (id: number, query: string) =>
  api.get<NomismaSearchResponse>(`/admin/mint-locations/${id}/nomisma/search`, { params: { query } })
export const linkNomismaMintLocation = (id: number, uri: string, label: string) =>
  api.post<MintLocation>(`/admin/mint-locations/${id}/nomisma`, { uri, label })
export const unlinkNomismaMintLocation = (id: number) =>
  api.delete<MessageResponse>(`/admin/mint-locations/${id}/nomisma`)
```

`MintLocation` (existing shared type in `src/web/src/types/index.ts`) gains:

```ts
nomismaUri?: string
nomismaLabel?: string
nomismaLinkedAt?: string // ISO timestamp, same convention as createdAt/updatedAt
```

## Attribution contract (visual, not an API contract)

Wherever `nomismaUri` is present on a rendered `MintLocation`/`MintReference`,
the shared `NomismaAttribution.vue` component renders exactly:

```
Source: Nomisma.org · CC BY 4.0
```

- `Nomisma.org` links to `nomismaUri` (the specific confirmed concept, not
  the Nomisma homepage).
- `CC BY 4.0` links to `https://creativecommons.org/licenses/by/4.0/`.

This exact string is asserted verbatim in `NomismaAttribution.test.ts` per
SC-002 ("100% of mint locations displaying Nomisma attribution show the
exact ... text").

## Non-goals reaffirmed by this contract

No route in this file accepts a private mint's ID and returns anything other
than 404; no route accepts a bulk/list payload; no route exposes SPARQL,
`getRdf`, or `getMints`; no route is reachable without the existing admin
middleware. These are intentional contract boundaries, not omissions.
