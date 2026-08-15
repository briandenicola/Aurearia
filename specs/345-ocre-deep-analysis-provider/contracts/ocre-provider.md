# Contract: OCRE Provider (internal) — Feature 345

Extends the Feature 344 internal contract
(`specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md`)
with the OCRE automated provider. **The generic Go↔Python provider contract
(§3/§4/§5 there) is unchanged**; this document adds one internal tool, one
upstream SPARQL boundary, and the OCRE-specific candidate/attribution shapes.

---

## 1. New internal tool endpoint (Go)

`POST /api/internal/tools/ocre_search` — registered in the existing
`internalDeepProviderTools` group (`src/api/main.go` ~line 845,
`middleware.InternalJobTokenRequired`), alongside `numista_search`,
`numista_detail`, `nomisma_search`. Handler:
`handlers.DeepProviderToolsHandler.OCRESearch`.

**Auth:** `Authorization: Bearer <minted job token>` (job-scoped; same mechanism
as the sibling tools). `jobID` is derived from the token (`deepProviderJobID`).

**Request body** (`OCRESearchRequest`, all fields optional except that ≥1 signal
must decode):

```jsonc
{ "ruler": "hadrian", "denomination": "denarius", "mint": "rome",
  "material": "", "legend_tokens": ["cos","iii"], "ocre_id": "", "limit": 5 }
```

**Response** (always HTTP 200):

```jsonc
{ "status": "ok",
  "candidates": [
    { "type_uri": "https://numismatics.org/ocre/id/ric.2.hdn.39b",
      "label": "RIC II Hadrian 39b",
      "matched_fields": ["ruler:hadrian","denomination:denarius"],
      "confidence": 0.86,
      "explanation": "Matched ruler Hadrian and denomination denarius." } ],
  "attribution": "Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic Society — ODbL 1.0." }
```

`status ∈ {ok, empty, invalid_response, unavailable, timeout, quota_limited,
cancelled}`. **Never returns 4xx/5xx for an upstream problem** (only `401` for a
missing job binding, `400` for an unparseable body) — the never-5xx rule
(FR-015, ADR 0009 precedent). Budget is consumed via
`deepProviderBudgets.TryConsume(jobID, "ocre", settings.OCRECallBudget)`; over
budget → `{"status":"quota_limited","candidates":[],"attribution":...}`. Every
returned candidate's `type_uri` host is re-validated `== numismatics.org`
Go-side before emission (FR-011).

**Swagger:** the handler carries `@Router /internal/tools/ocre_search [post]`
annotations mirroring `NomismaSearch` (Constitution III / §21.10).

---

## 2. Go upstream SPARQL boundary (`OCREClient`)

`src/api/services/ocre_client.go` — the **only** OCRE/Nomisma-triplestore HTTP
boundary. Mirrors `nomisma_client.go`.

```go
type OCREErrorKind string
const (
    OCREErrorUnavailable     OCREErrorKind = "unavailable"
    OCREErrorNoMatch         OCREErrorKind = "no_match"
    OCREErrorInvalidResponse OCREErrorKind = "invalid_response"
    OCREErrorInvalidRequest  OCREErrorKind = "invalid_request"
    OCREErrorCancelled       OCREErrorKind = "cancelled"
)

type OCRECandidate struct {
    TypeURI       string   `json:"type_uri"`
    Label         string   `json:"label"`
    MatchedFields []string `json:"matched_fields"`
    Confidence    float64  `json:"confidence"`
    Explanation   string   `json:"explanation"`
}

type OCREClient interface {
    Search(ctx context.Context, params OCREQueryParams, limit int) ([]OCRECandidate, OCREErrorKind, error)
}
```

**Transport (validated live 2026-08-15 — see research.md R1):**
- Method **GET** to `https://nomisma.org/query?query=<url-encoded SPARQL>`.
- Headers: `Accept: application/sparql-results+json`, `User-Agent:` a fixed
  non-default identifier (a default/empty UA is rejected 403 by the CDN).
- **POST is NOT used** (returns 403 at the Cloudflare edge).
- Timeout 8 s (`http.Client{Timeout}`), response cap 1 MiB (`io.LimitReader`),
  context cancellation → `OCRECancelled`. Any non-200 → `OCREErrorUnavailable`;
  malformed/oversize JSON → `OCREErrorInvalidResponse`. Never leaks a raw error.
- `NewHTTPOCREClientForTest(baseURL)` constructor for httptest (never reaches the
  real host in CI), same pattern as `NewHTTPNomismaClientForTest`.

**Response parse:** standard SPARQL 1.1 JSON — `results.bindings[]`, each with
`type.value` (URI) and `label.value` (literal), plus optional `auth/den/mint`.

---

## 3. Fixed SPARQL templates & binding (`ocre_query.go`)

Query skeleton is **constant**; only validated slugs are interpolated inside
`<...>` URI brackets. (Full templates + verified samples in research.md R2.)

- **Template E** (evidence search): `?type a nmo:TypeSeriesItem ; skos:prefLabel
  ?label` + zero-or-more of `?type nmo:hasAuthority|hasDenomination|hasMint|
  hasMaterial <http://nomisma.org/id/{slug}> .` + `FILTER(LANG(?label)="en")`
  `LIMIT {cap+margin}`.
- **Template K** (known-id confirm): subject
  `<http://numismatics.org/ocre/id/{ocre_id_slug}>`, `skos:prefLabel ?label`
  (+ optional auth/den/mint), `LIMIT 1`. Non-resolving id → `unresolved`
  (reported, not fabricated).

**Binding rules (SC-010 / FR-006):**
1. Slug regex `^[a-z0-9]([a-z0-9_.-]*[a-z0-9])?$`, length-bounded; OCRE id also
   `^ric\.[0-9a-z_.()]+$`. Failing values are **dropped**, never interpolated.
2. No `FILTER`/`VALUES`/`regex` ever receives raw text; only the constant
   `LANG(?label)="en"` filter is used.
3. Legend/inscription free text is **never** placed in SPARQL — scoring-only.
4. The emitted query string is **byte-identical** for a given slug set regardless
   of adversarial input values → asserted by `ocre_query_test.go` fixtures.

---

## 4. Deterministic scoring (`ocre_scoring.go`)

Pure function `Score(params, rows) []OCRECandidate`:
- De-dup by `TypeURI`; weighted match score (authority > denomination > mint >
  material; bounded legend-token-in-label bonus); confidence = normalized clamp.
- Order `(-Confidence, -len(MatchedFields), TypeURI asc)`; cap on distinct types.
- Deterministic — identical `(params, rows)` ⇒ identical output (SC-005).

---

## 5. Python node + tool wrapper

- `app/tools/provider_tools.py` gains
  `async def ocre_search(self, *, ruler, denomination, mint, material,
  legend_tokens, ocre_id, limit=5) -> dict` (thin authenticated POST to
  `/api/internal/tools/ocre_search`, same pattern as `nomisma_search`; raises
  `ProviderToolError` on transport failure).
- `app/teams/deep_identification/providers/ocre.py` `run(entry, tools,
  quick_evidence, notes)` (automated when `entry.automatable`):
  1. Decode bound params from `quick_evidence` (`coin_fields` ruler/denomination/
     mint/material + legend tokens + optional OCRE id).
  2. If no Roman-Imperial signal decodes → return `no_match`/`skipped` **without**
     a tool call (US1-AC5).
  3. Else call `tools.ocre_search(...)`, map `{status,candidates}` → typed
     `ProviderEvidence` with `claims[field="coin_type"]`, fixed attribution.
  4. On `ProviderToolError` → typed `failed`/`timed_out` row (never propagates).
  5. When `entry.automatable is False` (flag off) → trivial `not_automated` row,
     no call.
- `graph.py`: `"ocre"` moves from `_TRIVIAL_PROVIDER_NODES` into the automated
  fan-out path (still short-circuiting to `not_automated` when non-automatable).
- `router.py`: unchanged (OCRE flows through normal automatable selection;
  `provider_override` honored verbatim).

---

## 6. Citation & attribution

- Canonical citation host: **`numismatics.org`** (OCRE allowlist entry in
  `merge.CITATION_HOST_ALLOWLIST["ocre"]` — unchanged; Go re-checks).
- Fixed attribution string (row-level, exact): `Coin type data: Online Coins of
  the Roman Empire (OCRE), American Numismatic Society — ODbL 1.0.`
- License link constant (UI): `https://opendatacommons.org/licenses/odbl/1-0/`.
- Rendered distinct from Nomisma/Numista/RPC (new `OCREAttribution.vue`,
  separate from `NomismaAttribution.vue`) — FR-019/FR-020.

---

## 7. Settings

- `DeepIdentificationOCREEnabled` — existing, default `false`; gates
  `Automatable` on the OCRE catalog entry in
  `deepPipelineProviderCatalog(settings)`.
- `DeepIdentificationOCRECallBudget` — **new** key, default `"3"`, range 1–20;
  added to `settingDefaults` + read into `DeepIdentificationSettings`. No schema
  change.

---

## 8. Non-regression guarantees

- Numista/Nomisma/NGC/RPC provider paths, citation allowlists, SSE privacy,
  confirm-gated writes, and quick Identify are **untouched** (FR-024/SC-008).
- No generic `ProviderEvidence`/`ProviderClaim`/`DeepSynthesis` field added.
- No new table; no OCRE image or corpus (FR-021/SC-009).
