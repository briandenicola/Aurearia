# Phase 1 Data Model: OCRE Automated Deep Analysis Provider

**No database migration.** Every entity below persists in an existing structure.
This document defines the in-memory/wire shapes (Go + Python) and the exact
mapping onto the shipped Feature 344 persistence.

---

## 1. Persistence reuse (authoritative — no new tables)

| Concept | Stored in | Notes |
|---|---|---|
| Per-job OCRE run outcome | `DeepIdentificationProviderRun` (existing) | `Provider="ocre"`; one row per job. `Status`, `Automatable`, `Confidence`, `CallCount`, `LatencyMS`, `ErrorKind`, `ClaimsJSON`. |
| OCRE candidate claims | `DeepIdentificationProviderRun.ClaimsJSON` (existing `text`) | JSON array of the bounded claim shape (§3). |
| OCRE evidence in report/proposal | Job `ProposalJSON` via `buildDeepProposalDocumentJSON` (existing) | Same path as numista/nomisma claims. |
| Enablement flag | Setting `DeepIdentificationOCREEnabled` (existing, default `false`) | Read into `DeepIdentificationSettings.OCREEnabled` already. |
| OCRE call budget | Setting `DeepIdentificationOCRECallBudget` (**new key**, default `"3"`) | Additive key in `settingDefaults` + snapshot; no schema change. |

**Reused status enum** (`models/deep_identification_provider_run.go`, unchanged):
`contributed | no_match | failed | timed_out | skipped | not_automated | unavailable`.
Spec `invalid_response` → `Status=failed` + `ErrorKind="invalid_response"`
(that `ErrorKind` value already exists).

---

## 2. Bound SPARQL query parameters (`OCREQueryParams`) — Go, transient

The normalized, escaped/bound input set; also the cache-key source.

| Field | Type | Binding rule |
|---|---|---|
| `RulerSlug` | `string` | Nomisma authority id slug or `""`. Validated `^[a-z0-9]([a-z0-9_.-]*[a-z0-9])?$`. |
| `DenominationSlug` | `string` | Nomisma denomination id slug or `""`. Same validation. |
| `MintSlug` | `string` | Nomisma mint id slug or `""`. Same validation. |
| `MaterialSlug` | `string` | Nomisma material id slug or `""`. Same validation. |
| `LegendTokens` | `[]string` | Case-folded tokens; **scoring only, never in SPARQL**. |
| `OCREIDSlug` | `string` | Known OCRE id (Template K) or `""`. Extra shape check `^ric\.[0-9a-z_.()]+$`. |
| `Limit` | `int` | `resultCap + margin`, clamped. |

**Invariants:**
- Any slug failing validation is **dropped** (treated as absent), never
  interpolated (SC-010/FR-006).
- At least one of `{RulerSlug, DenominationSlug, MintSlug, OCREIDSlug}` must be
  present for a query to run; otherwise the node returns `no_match`/`skipped`
  with **no** SPARQL call (US1-AC5, "unsupported evidence" edge case).
- Presence of `OCREIDSlug` selects **Template K**; otherwise **Template E**.

**Cache key** = `SHA-256(join("\x1f", RulerSlug, DenominationSlug, MintSlug,
MaterialSlug, sort(LegendTokens), OCREIDSlug, Limit, flagGeneration))`.
`flagGeneration` derives from the enabled flag so a toggle never reuses a stale
entry.

---

## 3. OCRE candidate (`OCRECandidate`) — Go, transient → JSON

One per distinct surviving OCRE type after de-dup, scoring, and cap.

| Field | Type | Source |
|---|---|---|
| `TypeURI` | `string` | `?type` — canonical `https://numismatics.org/ocre/id/<slug>` (host re-validated). |
| `Label` | `string` | `?label` (`skos:prefLabel@en`), e.g. `"RIC II Hadrian 39b"`. |
| `MatchedFields` | `[]string` | e.g. `["ruler:hadrian","denomination:denarius","mint:rome"]`. |
| `Confidence` | `float64` | Deterministic score in `[0,1]` (research R3). |
| `Explanation` | `string` | Bounded human-readable matched-field summary (≤ 500). |

**Ordering (deterministic, ties fully broken):** `(-Confidence, -len(MatchedFields),
TypeURI asc)`. De-dup by `TypeURI` before ranking; cap on distinct types after.

---

## 4. Internal-tool wire shapes (`ocre_search`)

**Request** `POST /api/internal/tools/ocre_search` (job-token auth):

```jsonc
{
  "ruler": "hadrian",          // optional normalized slugs (Go re-validates)
  "denomination": "denarius",
  "mint": "rome",
  "material": "",
  "legend_tokens": ["cos", "iii"],
  "ocre_id": "",               // optional known OCRE id slug
  "limit": 5
}
```

**Response** (always HTTP 200 — never 5xx):

```jsonc
{
  "status": "ok",              // ok|empty|invalid_response|unavailable|timeout|quota_limited|cancelled
  "candidates": [
    { "type_uri": "https://numismatics.org/ocre/id/ric.2.hdn.39b",
      "label": "RIC II Hadrian 39b",
      "matched_fields": ["ruler:hadrian","denomination:denarius"],
      "confidence": 0.86,
      "explanation": "Matched ruler Hadrian and denomination denarius." }
  ],
  "attribution": "Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic Society — ODbL 1.0."
}
```

Status mapping (Go): `ok`=≥1 candidate; `empty`=0 rows (→ node `no_match`);
`invalid_response`=malformed/oversize (→ node `failed`+`error_kind=invalid_response`);
`unavailable`/`timeout`/`cancelled`=transport (→ node `failed`/`timed_out`, not
cached); `quota_limited`=budget exhausted (→ node degrades to `no_match`/`failed`).

---

## 5. `ProviderEvidence` mapping (Python, existing contract — UNCHANGED)

The node maps the tool response onto the existing `ProviderEvidence` /
`ProviderClaim` models (`app/models/responses.py`) — **no new field**:

```jsonc
{
  "provider": "ocre",
  "status": "contributed",       // contributed|no_match|failed|timed_out|not_automated
  "automatable": true,
  "confidence": 0.86,            // max candidate confidence (row-level)
  "call_count": 1,
  "error_kind": null,            // or "invalid_response"|"timeout"|"upstream"
  "link_out": "",
  "attribution": "Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic Society — ODbL 1.0.",
  "claims": [
    { "field": "coin_type",
      "value": "RIC II Hadrian 39b",
      "confidence": 0.86,
      "citation": "https://numismatics.org/ocre/id/ric.2.hdn.39b",
      "excerpt": "Matched ruler Hadrian and denomination denarius." }
  ]
}
```

- `field="coin_type"` must be in Go's coin-field allowlist (additive constant if
  missing) so `DeepSynthesis.proposed_fields` isn't dropped on ingest.
- Every claim citation host = `numismatics.org` → passes
  `merge.validate_citations("ocre", ...)` and the Go re-check (FR-011/SC-001).
- Multiple claims on `coin_type` = preserved ambiguity (FR-013); the
  evaluator/synthesizer already surface multi-claim fields as alternatives.

---

## 6. Validation & state rules (consolidated)

- **Injection:** dynamic values only enter SPARQL as validated slugs inside
  `<...>` URI brackets; query skeleton constant (SC-010).
- **Bounds:** per-job budget, 8 s timeout, 1 MiB response cap, result cap — all
  enforced Go-side (FR-008/SC-006).
- **De-dup + cap:** distinct `TypeURI`, then cap (FR-014/FR-012).
- **Determinism:** fixed weights + tie-break by canonical URI (SC-005).
- **Failure isolation:** every transport/parse failure → typed status, never a
  job failure or an unbounded retry (FR-015/SC-003).
- **Flag-off:** catalog `automatable=false` → `not_automated`, zero calls
  (FR-004/FR-016/SC-004).
- **Attribution:** present iff ≥1 OCRE claim; distinct from other providers
  (FR-019/FR-020/SC-002).
- **No image / no corpus:** no image field is ever populated or fetched; only
  bounded per-job rows persist (FR-021/SC-009).
