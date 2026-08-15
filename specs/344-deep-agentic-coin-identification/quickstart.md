# Quickstart: Deep Agentic Coin Identification (Feature 344)

Developer-facing orientation for implementing and exercising the feature. This
is a planning artifact — no production code exists yet.

---

## 1. What this feature adds (30-second version)

| | Fast Identify Coin (existing) | Deep Analysis (new) |
|---|---|---|
| Endpoint | `POST /api/coins/lookup` | `POST /api/deep-identification/jobs` |
| Shape | synchronous request/response | persisted background job + replayable SSE |
| Duration | seconds | up to 5 minutes (bounded) |
| Sources | vision model + Numista query proposal | vision + router + bounded provider fan-out + contradiction evaluator + synthesis |
| Output | prefilled draft fields | narrative report + citations + typed proposal + provider coverage |
| Writes | none until Save as Draft | none until explicit Apply/Promote |
| Status | **unchanged by this feature** | opt-in, admin-flag gated |

---

## 2. Local setup

```powershell
# from repo root
task up-all          # Go API + Vue dev server + Python agent

# individual services
task run-api         # src/api
task run-web         # src/web
task run-agent       # src/agent
```

Environment (existing variables, no new ones required beyond the `AGENT_DEEP_*`
bounds):

| Variable | Service | Purpose |
|---|---|---|
| `AGENT_SERVICE_URL` | Go | base URL of the Python agent |
| `AGENT_INTERNAL_SERVICE_TOKEN` | Go + Python | Go → Python authentication |
| `JWT_SECRET` | Go | also signs the short-lived token Python uses to call back |
| `AGENT_DEEP_MAX_CONCURRENCY` / `AGENT_DEEP_MAX_PROVIDERS` / `AGENT_DEEP_PROVIDER_TIMEOUT` / `AGENT_DEEP_TOTAL_TIMEOUT` / `AGENT_DEEP_RECURSION_LIMIT` | Python | pipeline bounds |

Admin settings (`AppSetting`, admin UI): set
`deep_identification_enabled = true` to allow new job starts. Everything else
(worker count, retention, budgets) has a safe default — see data-model.md §8.

---

## 3. End-to-end flow

```text
Vue                     Go API                                   Python agent            Providers
 │  POST jobs (multipart) │                                        │                       │
 ├───────────────────────>│ validate images/roles, store artifacts │                       │
 │  <── 202 {job}         │ create job (fingerprint idempotency)   │                       │
 │  GET .../events?since  │ enqueue → worker claims                │                       │
 ├───────────────────────>│ build quick evidence (CoinLookupService)                        │
 │  <== SSE (replay+live) │ POST /api/deep-identify/stream ───────>│ prepare → router      │
 │                        │ persist each translated event          │ fan-out (≤2 concurrent)│
 │                        │<── internal tool calls ────────────────┤ numista/nomisma tools ├──> upstream
 │                        │ (Go clients, keys, cache, budgets)     │                       │
 │                        │<== provider_result / evaluation ───────┤ evaluator → synthesis │
 │  <== terminal + end    │ settle terminal, store report+proposal │                       │
 │  PATCH proposal        │ owner edits, no coin write             │                       │
 │  POST apply            │ QuickCaptureDraft seed  OR  CoinService.UpdateCoinWithFields    │
```

---

## 4. Try it with curl

```bash
TOKEN=... # access token

# Start from new intake (uploads)
curl -X POST http://localhost:8080/api/deep-identification/jobs \
  -H "Authorization: Bearer $TOKEN" \
  -F obverse=@obv.jpg -F reverse=@rev.jpg \
  -F 'hints=@dealer-tag.jpg' \
  -F 'notes=Dealer said Severan, bought at a show' \
  -F 'providers=numista' -F 'providers=nomisma'

# Start from a saved coin (reuses its stored obverse/reverse)
curl -X POST http://localhost:8080/api/deep-identification/jobs \
  -H "Authorization: Bearer $TOKEN" -F coinId=42

# Follow the stream, resuming after event 12
curl -N -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/deep-identification/jobs/88/events?since=12"

# Cancel / retry
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/deep-identification/jobs/88/cancel
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/deep-identification/jobs/88/retry

# Review and apply (saved coin)
curl -X PATCH -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"fields":{"denomination":{"accepted":true},"mint":{"ownerValue":"Rome","accepted":true}}}' \
  http://localhost:8080/api/deep-identification/jobs/88/proposal
curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"target":"coin"}' \
  http://localhost:8080/api/deep-identification/jobs/88/apply
```

---

## 5. Acceptance walkthroughs (map to spec user stories)

1. **US1 — intake** Upload obverse + reverse, start Deep Analysis, navigate
   away, return: job still present, fast lookup unaffected. Missing a required
   role ⇒ `422 missing_reverse`, no job row created.
2. **US2 — saved coin** Start from a coin with both faces; the coin record is
   byte-identical until Apply. Another user's coin ⇒ `404`.
3. **US3 — resume/cancel/retry** Kill the browser tab mid-run and reconnect with
   `since`: missed events arrive once, in order. Cancel: job settles
   `cancelled`, no report. Retry: new job with `retryOfJobId` set; the original's
   events and report remain.
4. **US4 — report/draft** Complete a run with one provider forced to fail:
   result is `partial`, the report lists the failure status explicitly, the
   proposal is editable, and no coin changes until Apply.
5. **US5 — provider transparency** Select NGC: coverage shows
   `not_automated` with a link-out, never `no_match`. Deselect Numista: no
   Numista call is made.
6. **US6 — hint privacy** Attach a hint image; on completion, cancellation, and
   failure alike the file is gone from `<UploadDir>/deep-jobs/job-<id>/` and it
   never appears among the coin's images.

---

## 6. Test commands

```powershell
# Go
cd src/api;  go build ./...; go vet ./...; go test ./...
cd src/api;  go test -run TestArchitecture ./...

# Python agent
cd src/agent; ruff check app/ tests/; pytest tests/ -v

# Vue
cd src/web;  npm run type-check; npm run test; npm run build

# Repo-level
task test
task openapi     # required whenever a Go handler changes (CI diffs docs/openapi.json)
```

---

## 7. Provider reality check (do not regress this)

| Provider | MVP behaviour | Why |
|---|---|---|
| Nomisma | automated via existing Go client | public API, CC BY, attribution rendered |
| Numista | automated via existing Go client, ≤ 4 calls/job | official API + key; attribution "Source: Numista" and N# must be visible |
| NGC | `not_automated` — OCR cert + link to `https://www.ngccoin.com/verify/` | NGC Terms of Use §2 prohibit robot/automated access |
| OCRE | `not_automated` until gate G-OCRE | ODbL review pending; access path is the Nomisma SPARQL endpoint |
| RPC Online | `unavailable` — manual reference only | no public API, CC BY-NC-SA, server blocks non-browser clients |

Never render `not_automated` or `unavailable` as "no match", and never
fabricate a provider result (FR-025, SC-007).

---

## 8. Gotchas

- **Do not** touch `POST /api/coins/lookup` — a regression test asserts its
  shape and that no deep job is created (SC-008).
- **Do not** add a new coin-write path; Apply must call
  `QuickCaptureService.PromoteDraft` (via draft seeding) or
  `CoinService.UpdateCoinWithFields`.
- **Do not** send provider API keys or a database handle to Python.
- **Do not** log notes, hint-derived context, provider queries, or report text.
- Sequence numbers come from the job row inside the state-change transaction,
  never from an in-memory counter — that is what makes replay gap-free.
- Native `EventSource` cannot send an `Authorization` header; the frontend uses
  the `fetch` + `ReadableStream` pattern already shipped in
  `src/web/src/api/client.ts::agentChatStream`, with `?since=` instead of
  `Last-Event-ID`.
