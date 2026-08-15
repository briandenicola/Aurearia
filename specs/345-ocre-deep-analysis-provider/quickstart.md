# Quickstart: OCRE Automated Deep Analysis Provider (Feature 345)

Reference for developers/reviewers/operators to enable, exercise, and verify the
OCRE provider. Assumes Feature 344 Deep Analysis is already deployed.

---

## 1. Enable OCRE (admin, default-off)

OCRE ships **disabled**. Enable it via the existing admin setting (admin-only,
FR-017):

- Setting key: `DeepIdentificationOCREEnabled` → set to `true`.
- Optional: `DeepIdentificationOCRECallBudget` (default `3`, range 1–20).

When enabled, the provider catalog entry becomes `{Provider:"ocre",
Automatable:true, CallBudget:<OCRECallBudget>}` for **subsequent** jobs
(`deepPipelineProviderCatalog`). In-flight jobs keep their decided catalog.

Disable = set the flag to `false`: OCRE reverts to `not_automated`, makes **zero**
SPARQL calls, and renders no attribution (SC-004).

---

## 2. Happy-path verification (US1/US2)

1. Enable the flag. Start Deep Analysis on a clearly Roman-Imperial coin
   (e.g. normalized evidence resolves ruler=Hadrian, denomination=denarius).
2. Expect: router selects `ocre`; one or more candidate claims appear with
   canonical `https://numismatics.org/ocre/id/...` citations, RIC-style labels
   ("RIC II Hadrian 39b"), matched-field explanations, and deterministic
   confidences.
3. Expect on the report, draft proposal, provider-status panel, and an export:
   the exact string
   `Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic
   Society — ODbL 1.0.` with a link to the canonical OCRE type and a link to
   `https://opendatacommons.org/licenses/odbl/1-0/`, visually distinct from
   Nomisma/Numista attribution (SC-002).
4. Re-run on identical evidence → identical candidate ordering + set (SC-005).

---

## 3. Robustness verification (US3)

- Force upstream timeout / HTTP 500 / malformed bindings (fake internal tool or
  fault-injected fixture): job still reaches completed/partial; OCRE row is
  `timed_out`/`failed` (`error_kind=invalid_response` for malformed); no
  non-`numismatics.org` citation surfaces (SC-003, FR-011).
- Non-Roman coin (e.g. Greek): OCRE not selected / `no_match`, no call (US1-AC5).
- Flag off + explicit override naming OCRE: OCRE stays `not_automated`, zero
  calls (flag wins over override, FR-004).

---

## 4. Test matrix (offline fixtures — CI)

| Area | Test | Asserts |
|---|---|---|
| Injection | `ocre_query_test.go` | Bound query byte-identical across adversarial ruler/mint/legend inputs (SC-010). |
| Slug validation | `ocre_query_test.go` | Invalid slugs dropped, never interpolated; `<...>` URI slots only. |
| Transport | `ocre_client_test.go` (httptest) | 200 parse, empty→no_match, 500→unavailable, malformed→invalid_response, oversize→invalid_response, slow→timeout, cancel→cancelled. |
| Scoring | `ocre_scoring_test.go` | Determinism across ≥2 runs, tie-break by URI, de-dup, cap, ambiguity preserved, no-match. |
| Handler | `internal_tools_test.go` | Job-token required, budget→quota_limited, allowlist re-check, never-5xx. |
| Node | `test_deep_identification_ocre.py` | contributed/no_match/timeout/malformed mapping, flag-off not_automated (no call), attribution present. |
| Router | `test_deep_identification_router.py` | OCRE selected for Roman-Imperial evidence; explicit override include/exclude; flag-off wins. |
| Attribution | Vue `__tests__` | OCRE attribution present iff OCRE claim; distinct from Nomisma; absent otherwise (SC-002). |
| No image / corpus | inspection tests | No image field populated/fetched; only bounded rows persisted (SC-009). |

---

## 5. Manual live smoke test (EXCLUDED from CI)

Guard behind a build tag / skip-by-default so CI never hits the network
(Testing & CI Constraints). Reproduces the planning-validated call:

```powershell
$q = 'PREFIX nmo: <http://nomisma.org/ontology#> PREFIX skos: <http://www.w3.org/2004/02/skos/core#> SELECT ?type ?label WHERE { ?type a nmo:TypeSeriesItem ; skos:prefLabel ?label ; nmo:hasAuthority <http://nomisma.org/id/hadrian> ; nmo:hasDenomination <http://nomisma.org/id/denarius> . FILTER(LANG(?label)="en") } LIMIT 3'
$headers = @{ "User-Agent"="AncientCoins/1.0 (smoke)"; "Accept"="application/sparql-results+json" }
$uri = "https://nomisma.org/query?query=" + [uri]::EscapeDataString($q)
$r = Invoke-WebRequest -Uri $uri -Method Get -Headers $headers -TimeoutSec 45 -UseBasicParsing
[System.Text.Encoding]::UTF8.GetString($r.Content)
```

Expect HTTP 200, `application/sparql-results+json`, bindings with
`http://numismatics.org/ocre/id/ric.2.hdn.*` URIs. **GET only — POST returns 403.**

---

## 6. Build & gate commands (Constitution §17/§21)

```bash
# Go API (src/api/)
go vet ./... && go build ./... && go test ./...
# Python agent (src/agent/)
ruff check app/ tests/ && pytest tests/ -v
# Vue (src/web/)
npm run build
```

Plus: ADR 0010 present; Swagger regenerated for the new internal handler;
PR self-check citing Principles I/II/III/IV and §17/§21.
