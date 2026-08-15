# Contract: Deep Identification SSE Event Envelope (Vue ← Go)

**Endpoint**: `GET /api/deep-identification/jobs/{id}/events`
**Media type**: `text/event-stream`
**Owner**: Go API (`src/api/handlers/deep_identification.go`). Events are
persisted first (`deep_identification_events`) and served from storage, so the
stream is replayable across client disconnects and API restarts (FR-009,
FR-015, FR-016).

---

## 1. Frame format

```text
id: 17
event: provider_result
data: {"seq":17,"jobId":88,"type":"provider_result","ts":"2026-08-15T13:02:59Z","payload":{ … }}

: ping
```

- `id` — the per-job monotonic sequence (`deep_identification_events.seq`,
  unique per `(job_id, seq)`). Browsers echo it as `Last-Event-ID` on
  auto-reconnect.
- `event` — envelope type (§2). Clients MUST ignore unknown types (forward
  compatibility).
- `data` — a single-line JSON **application-owned envelope**. It is *not* a
  passthrough of LangGraph events; the Go worker translates the internal agent
  stream into this envelope before persisting.
- `: ping` — keepalive comment every 15 s. No `id`, no sequence consumed.

### Envelope

| Field | Type | Notes |
|---|---|---|
| `seq` | int64 | equals the `id:` line |
| `jobId` | int | |
| `type` | string | §2 |
| `ts` | RFC3339 | server time |
| `payload` | object | type-specific, always sanitized (§4) |

---

## 2. Event types

| `type` | Payload | Emitted when |
|---|---|---|
| `job_accepted` | `{status, source, coinId, requestedProviders}` | job row created |
| `status_changed` | `{status, previousStatus}` | queued→running, or terminal settle |
| `router_selected` | `{selectedProviders[], rationale, skipped:[{provider,reason}]}` | router node finished |
| `provider_started` | `{provider}` | provider worker dispatched |
| `provider_result` | `{provider, status, confidence, claimCount, errorKind?, linkOut?}` | provider worker settled (any status, incl. `not_automated`) |
| `evaluation` | `{disagreementCount, resolvedCount}` | contradiction/provenance node finished |
| `synthesis_started` | `{}` | synthesis node began |
| `progress` | `{phase, message}` | coarse progress; `message` is short, sanitized, emoji-free |
| `terminal` | `{status, partialSuccess, failureCode?, hasReport, hasProposal}` | job reached exactly one terminal state |
| `stream_truncated` | `{status, earliestSeq, lastSeq}` | requested `since` predates retention (control frame; consumes no sequence — sent with `id:` omitted) |

After the `terminal` frame the server emits:

```text
event: end
data: {"jobId":88,"status":"partial"}
```

and closes. Clients MUST NOT auto-reconnect after `end`; they fetch
`GET /api/deep-identification/jobs/{id}` for the report/proposal.

---

## 3. Resume semantics

| Client state | Request | Server behaviour |
|---|---|---|
| First connect | no `since`, no `Last-Event-ID` | replay all retained events from seq 1, then follow live |
| Reconnect after drop | `?since=N` (fetch reader) or `Last-Event-ID: N` (EventSource) | replay `(N, lastSeq]`, then follow live. `since` wins if both present |
| Reconnect after long outage | `since` < earliest retained | `stream_truncated` first, then retained tail, then live |
| Job already terminal | any | replay remaining events, then `terminal` (from storage) + `end` |
| Events pruned, job terminal | any | `stream_truncated` + `terminal` snapshot + `end` (result still readable via GET) |
| Job/result past retention | any | HTTP `410` before the stream opens |

Guarantees (SC-003): within retention, a reconnecting client receives every
missed event **exactly once, in order, with no gaps** — enforced by the unique
`(job_id, seq)` index and by assigning `seq` in the same transaction as the
state change, never from an in-memory counter.

## 4. Authorization, limits, sanitization

- Standard bearer auth; the browser reader uses the `fetch` +
  `ReadableStream` + `Authorization` header pattern already shipped in
  `src/web/src/api/client.ts::agentChatStream` (native `EventSource` cannot set
  headers — hence the `?since=` alternative to `Last-Event-ID`).
- A job is streamable only by its owner; non-owner and unknown ids both return
  `404` (no existence leak).
- Max 3 concurrent streams per job per owner; excess connections get `429`.
- Payloads are sanitized before persistence with the existing
  `sanitize_user_facing_text` on the Python side and a Go-side guard that
  strips any `Bearer`/`rt_`/token-shaped substring. Owner notes, hint-derived
  context, and provider query strings are **never** placed in `progress`
  messages or logs (FR-036).
- No hint-image URL or binary ever appears in any event (FR-030).

## 5. Backward compatibility

New endpoint; no existing stream changes. `POST /api/agent/chat` and its
`{type: text|status|done|error}` shape (`app/streaming.py::format_sse`) are
untouched. The deep envelope deliberately uses a different, versioned shape
because it is persisted and replayed rather than transient.
