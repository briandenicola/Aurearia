# Contract: Go ↔ Python Deep Identification (internal)

**Direction 1 — inference stream**: Go → Python
`POST {AGENT_SERVICE_URL}/api/deep-identify/stream` → `text/event-stream`
**Direction 2 — provider data**: Python → Go
`POST {tools_base_url}/api/internal/tools/{numista_search|numista_detail|nomisma_search}`

Both directions are internal-only and never reachable from the browser.

---

## 1. Authentication

| Direction | Credential | Existing mechanism |
|---|---|---|
| Go → Python | header `X-Internal-Service-Token: <AGENT_INTERNAL_SERVICE_TOKEN>` | `services.AgentProxy.attachInternalCredential` ↔ `src/agent/app/security.py::InternalServiceAuthMiddleware` (`secrets.compare_digest`) |
| Python → Go | header `Authorization: Bearer <short-lived minted token>` | `services.InternalTokenService.Mint(userID)` ↔ `middleware.InternalTokenRequired` on the `/api/internal/tools` group (`src/api/main.go:804-811`) |

The minted token is issued per job run, carries the owner binding, and is
passed into the Python request body as `internal_token` exactly as
`AgentChatProxyRequest` does today for collection tools. Python holds **no**
database handle, **no** API keys, and **no** persistent state (Principle II,
FR-035).

---

## 2. Go → Python request (`DeepIdentifyRequest`)

Pydantic `StrictRequestModel` (`extra="forbid"`), defined in
`src/agent/app/models/requests.py`; Go mirror struct
`services.DeepIdentifyProxyRequest` in `src/api/services/agent_proxy.go`.

```jsonc
{
  "job_id": 88,                          // opaque correlation id (no DB meaning to Python)
  "schema_version": 1,
  "llm_config": { "provider": "anthropic", "api_key": "…", "model": "…" },   // existing LLMConfig
  "images": [
    { "role": "obverse", "data_uri": "data:image/jpeg;base64,…" },
    { "role": "reverse", "data_uri": "data:image/jpeg;base64,…" },
    { "role": "hint",    "data_uri": "data:image/jpeg;base64,…", "hint_kind": "label" }
  ],
  "notes": "Bought at a show, dealer said Severan.",       // optional, never logged
  "quick_evidence": {                                      // normalized output of CoinLookupService.Lookup
    "label_text": "…",
    "coin_fields": { "denomination": "denarius", "era": "roman-imperial" },
    "confidence": "medium",
    "ngc": { "cert_number": "1834646-097", "grade": "XF45",
             "lookup_url": "https://www.ngccoin.com/verify/" },
    "numista_query": "Septimius Severus denarius",
    "numista_evidence": { … }                              // models.NumistaEvidence, passthrough
  },
  "provider_override": ["numista", "nomisma"],             // empty ⇒ router decides
  "provider_catalog": [                                    // Go tells Python what is legal THIS run
    { "provider": "numista", "automatable": true,  "call_budget": 4 },
    { "provider": "nomisma", "automatable": true,  "call_budget": 3 },
    { "provider": "ngc",     "automatable": false, "reason": "terms_prohibit_automated_access",
      "link_out": "https://www.ngccoin.com/verify/" },
    { "provider": "ocre",    "automatable": false, "reason": "pending_license_validation" },
    { "provider": "rpc",     "automatable": false, "reason": "no_public_api" }
  ],
  "bounds": { "max_providers": 4, "max_concurrency": 2, "provider_timeout_s": 45,
              "total_timeout_s": 280, "recursion_limit": 12 },
  "tools_base_url": "http://api:8080",
  "internal_token": "<minted>"
}
```

Rules:

- Python **must not** contact a provider whose `automatable` is `false`; it
  emits a `provider_result` with `status: "not_automated"` (or `"unavailable"`)
  and the supplied `reason`/`link_out`. Fabricating a `no_match` for such a
  provider is a contract violation (FR-025) and is covered by a test.
- Python **must not** exceed `bounds`; the graph enforces
  `max_concurrency` with an `asyncio.Semaphore` and per-provider
  `asyncio.wait_for(provider_timeout_s)`.
- Hint images are marked by `role: "hint"` and are used only as context — they
  never enter the coin-face evidence slots of the vision prompt (FR-004).

---

## 3. Python → Go SSE (internal envelope)

Formatted with the existing `app/streaming.py::format_sse`; one JSON object per
frame. Go translates each into a persisted `DeepIdentificationEvent`
(contracts/sse-events.md) — the browser never sees this shape directly.

| `type` | Payload |
|---|---|
| `router_selected` | `{selected:[…], skipped:[{provider,reason}], rationale}` |
| `provider_started` | `{provider}` |
| `provider_result` | `ProviderEvidence` (§4) |
| `evaluation` | `{disagreements:[…], resolved:[…]}` |
| `progress` | `{phase, message}` (sanitized, emoji-free) |
| `synthesis` | `DeepSynthesis` (§5) — exactly one, terminal-success frame |
| `error` | `{code, message}` — typed codes only: `llm_unavailable`, `timeout`, `invalid_model_output`, `internal` |

Termination: exactly one of `synthesis` or `error` ends the stream; Python then
closes. Go treats stream EOF without either as `agent_unavailable`.

Cancellation propagation: Go cancels the request `context`; `httpx`/Starlette
observes the client disconnect and the graph run is abandoned. Python performs
no cleanup beyond releasing the connection (it holds no state). Go discards any
already-received partial evidence for a cancelled job (FR-018/FR-019).

---

## 4. `ProviderEvidence` (typed, never prose)

```jsonc
{
  "provider": "numista",
  "status": "contributed",     // contributed|no_match|failed|timed_out|not_automated|unavailable|skipped
  "automatable": true,
  "confidence": 0.72,
  "call_count": 3,
  "error_kind": null,          // timeout|quota|unconfigured|upstream|invalid_response
  "link_out": null,
  "attribution": "Source: Numista",
  "claims": [
    { "field": "denomination", "value": "Denarius", "confidence": 0.8,
      "citation": "https://en.numista.com/catalogue/pieces12345.html",
      "excerpt": "Silver denarius, Rome mint" }
  ]
}
```

**Citation validation** (enforced in Python before emission, re-checked in Go
before persistence): every claim must carry a `citation` whose host belongs to
the emitting provider's canonical host allowlist
(`en.numista.com`/`api.numista.com`, `nomisma.org`, `numismatics.org`,
`www.ngccoin.com`, `rpc.ashmus.ox.ac.uk`). Claims failing validation are
dropped and counted in the run's `invalid_response` telemetry — the LLM cannot
introduce arbitrary URLs (SC-006).

---

## 5. `DeepSynthesis` (typed final output)

```jsonc
{
  "narrative": "…",
  "proposed_fields": {
    "denomination": { "value": "Denarius", "confidence": 0.82,
                      "evidence_refs": [{ "provider": "numista", "claim_index": 0 }] }
  },
  "disagreements": [ { "field": "mint", "claim_refs": [ … ], "resolution": "unresolved" } ],
  "unresolved_questions": ["…"],
  "coverage": [ { "provider": "ngc", "status": "not_automated" } ],
  "partial_success": true
}
```

- `proposed_fields` keys are restricted to the coin-field allowlist supplied by
  Go; unknown keys are dropped by Go on ingest.
- Every field with provider-derived evidence must reference at least one
  validated claim (SC-006). Fields supported only by image evidence are marked
  `evidence_refs: [{ "provider": "image", … }]` and rendered as such.
- Merge determinism: evidence is merged by a pure function
  (`app/teams/deep_identification/merge.py`) that sorts claims by
  `(field, provider_rank, -confidence, citation)` before the LLM sees them, so
  the same inputs always produce the same prompt ordering.

---

## 6. Python graph topology (`src/agent/app/teams/deep_identification/`)

```text
prepare_evidence  →  router  →  provider_fanout (bounded)  →  evaluator  →  synthesizer
   (vision node)      (LLM)      (Semaphore ≤ max_conc.)      (LLM)         (LLM, strict JSON)
```

- `state.py`: `DeepIdentificationState` TypedDict — `job_id`, `images`,
  `notes`, `quick_evidence`, `catalog`, `bounds`, `selected`, `evidence`
  (list reducer), `disagreements`, `synthesis`, `errors`.
- `router.py`: single LLM call constrained to choose from
  `provider_catalog` entries; output validated against the enum; a provider
  named in `provider_override` is always included, and `automatable: false`
  entries short-circuit into a `not_automated` evidence row without a call.
- `providers/`: one node per provider; automatable nodes call the Go internal
  tool (`app/tools/provider_tools.py`), never the upstream directly.
- **Partial failure**: `asyncio.gather(..., return_exceptions=True)` (the
  existing repo pattern) converts an exception/timeout into a typed
  `failed`/`timed_out` evidence row; the graph always continues to the
  evaluator (FR-026).
- **Iteration bound**: `config={"recursion_limit": bounds.recursion_limit}`,
  no cycles in the graph; no ReAct free-tool loop is used in this pipeline.
- **Total bound**: the route wraps the run in
  `asyncio.wait_for(total_timeout_s)`; on expiry it emits a final `synthesis`
  built from evidence gathered so far with `partial_success: true`, or an
  `error` frame if nothing was gathered.

---

## 7. Go → Python provider tool endpoints (new, internal)

| Endpoint | Body | Response |
|---|---|---|
| `POST /api/internal/tools/numista_search` | `{"query": "…", "limit": 5}` | `{"status":"ok|empty|unconfigured|quota_limited|timeout|unavailable","candidates":[NumistaCandidate],"attribution":"Source: Numista"}` |
| `POST /api/internal/tools/numista_detail` | `{"id": 12345}` | `{"status":"…","candidate":NumistaCandidate,"identifier":"N#12345"}` |
| `POST /api/internal/tools/nomisma_search` | `{"query": "…", "limit": 5}` | `{"status":"ok|empty|unavailable","candidates":[NomismaCandidate],"attribution":"Data: Nomisma.org (CC BY)"}` |

- Backed by the shipped clients `services/numista_client.go` (+
  `numista_cache.go`, `numista_telemetry.go`, ADR 0007) and
  `services/nomisma_client.go` (+ `nomisma_cache.go`, ADR 0009). Statuses reuse
  the existing F341 six-status vocabulary and the F343 never-5xx rule.
- Owner scoping and per-job call budgets are enforced Go-side from the minted
  token's user binding; exceeding the budget returns `status: "quota_limited"`
  rather than an error, so the graph degrades to `no_match`/`failed` cleanly.
- No new upstream HTTP client is introduced in Python; `app/outbound.py` egress
  is not used by this pipeline.
