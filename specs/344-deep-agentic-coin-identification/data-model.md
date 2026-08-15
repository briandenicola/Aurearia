# Phase 1 Data Model: Deep Agentic Coin Identification

**Feature**: 344-deep-agentic-coin-identification
**Storage**: SQLite via GORM, additive `AutoMigrate` only
(`src/api/database/database.go:36`), no destructive changes, no backfill.
**Ownership**: every table is owner-scoped by `user_id` (FR-006, FR-037).

New model files (Go, stdlib-only imports per Principle I):

- `src/api/models/deep_identification_job.go`
- `src/api/models/deep_identification_event.go`
- `src/api/models/deep_identification_provider_run.go`
- `src/api/models/deep_identification_artifact.go`

No existing model is modified. `models.AIJob`, `models.Coin`,
`models.CoinImage`, `models.QuickCaptureDraft`, `models.CoinIntakeDraft` are
unchanged (research.md §3).

---

## 1. Entity overview

```text
User 1───* DeepIdentificationJob 1───* DeepIdentificationEvent      (append-only, replayable)
                     │              1───* DeepIdentificationProviderRun (per provider attempt)
                     │              1───* DeepIdentificationArtifact    (coin-face refs + ephemeral hints)
                     │              0───1 Coin        (optional source coin)
                     │              0───1 DeepIdentificationJob (retry parent, self-FK)
                     └── report_json / proposal_json (terminal result, retained separately)
```

---

## 2. `DeepIdentificationJob`

| Field | Type | Notes |
|---|---|---|
| `ID` | `uint` PK | |
| `UserID` | `uint` NOT NULL | owner; every query scoped by it |
| `CoinID` | `*uint` NULL | **optional** — absent for new-intake jobs (spec Key Entities) |
| `Status` | `DeepJobStatus` (varchar 20) | see §2.1 |
| `Source` | `varchar(20)` | `intake` \| `saved_coin` |
| `InputFingerprint` | `char(64)` NOT NULL | sha256, see §2.3 |
| `Notes` | `text` | optional owner notes; **never logged** (FR-036) |
| `RequestedProviders` | `text` (JSON array) | owner override, empty ⇒ router decides |
| `SelectedProviders` | `text` (JSON array) | confirmed set after router |
| `RouterRationale` | `text` | short, sanitized |
| `RetryOfJobID` | `*uint` NULL | self-FK, retry lineage (FR-020) |
| `RetryDepth` | `int` default 0 | bounded (max 3) |
| `AttemptCount` | `int` default 0 | worker claim attempts |
| `CancelRequestedAt` | `*time.Time` | set by cancel, does not itself settle status |
| `LastSeq` | `int64` default 0 | highest appended event sequence |
| `HeartbeatAt` | `*time.Time` | worker liveness for stale recovery (FR-012) |
| `WorkerID` | `varchar(64)` | process/worker instance that claimed the job |
| `ReportJSON` | `text` | terminal narrative + citations + coverage (§6) |
| `ProposalJSON` | `text` | typed proposed fields + confidence/evidence (§7) |
| `PartialSuccess` | `bool` default false | FR-026 marker |
| `FailureCode` | `varchar(40)` | typed, client-safe (`timeout`, `agent_unavailable`, `internal`, `stale_restart`) |
| `FailureMessage` | `varchar(300)` | generic, no internals (Principle V) |
| `AppliedCoinID` | `*uint` NULL | set when proposal applied to a coin |
| `AppliedDraftID` | `*uint` NULL | set when proposal seeded a `QuickCaptureDraft` |
| `AppliedAt` | `*time.Time` | SC-009 traceability |
| `StartedAt`, `CompletedAt` | `*time.Time` | |
| `ExpiresAt` | `time.Time` | result retention horizon (created + 90 d) |
| `EventsPrunedAt` | `*time.Time` | set by janitor when events aged out (drives `stream_truncated`) |
| `CreatedAt`, `UpdatedAt` | `time.Time` | GORM |

### 2.1 Job state machine (FR-011, FR-019)

```text
        ┌───────────── cancel ─────────────┐
        v                                  │
   [queued] ──claim──> [running] ──────────┼──> [completed]   (all selected providers resolved, report stored)
        │                  │               ├──> [partial]     (≥1 provider failed/unavailable, report stored, PartialSuccess=true)
        │                  ├───────────────┼──> [failed]      (hard error / hard timeout / stale restart)
        └──────────────────┴───────────────┴──> [cancelled]   (owner cancel won the race)
```

Rules:

- Terminal states: `completed`, `partial`, `failed`, `cancelled`. All are
  one-way; no terminal → non-terminal transition exists in any repository
  method.
- Every terminal transition is a **single conditional UPDATE**
  `WHERE id = ? AND user_id = ? AND status IN ('queued','running')`;
  `RowsAffected == 0` means someone else settled it (race lost).
- The winning UPDATE and the terminal `DeepIdentificationEvent` append occur in
  **one transaction** ⇒ exactly one terminal event per job (FR-019).
- `cancel` sets `CancelRequestedAt` only; the settle-to-`cancelled` UPDATE is
  performed by whichever path observes it first (handler for `queued` jobs,
  worker for `running` jobs).
- **Stale restart recovery** (FR-012): at boot and every 60 s,
  `RecoverStaleJobs(staleAfter = 3 × heartbeatInterval)` transitions
  `running` jobs whose `HeartbeatAt` is older than the threshold (or whose
  `WorkerID` is not the live worker after boot) to `failed` with
  `FailureCode = "stale_restart"`, appending a terminal event. Nothing is left
  `running` forever.

### 2.2 Provider-run state machine (FR-025)

```text
[pending] ──dispatch──> [running] ──> [contributed]
                              ├──────> [no_match]
                              ├──────> [failed]
                              ├──────> [timed_out]
                              └──────> [skipped]
[pending] ──router/legal gate──> [not_automated] | [unavailable]
```

`not_automated` (NGC now; OCRE until gate G-OCRE) and `unavailable` (RPC;
provider misconfigured, e.g. no Numista key) are **first-class terminal
statuses** and MUST NOT be rendered or synthesized as `no_match`.

### 2.3 Idempotency / input fingerprint (FR-007)

```text
InputFingerprint = sha256(
    "v1" | user_id | coin_id_or_0 |
    sha256(obverse_bytes) | sha256(reverse_bytes) |
    sorted(sha256(hint_bytes_i)) |
    sha256(normalized_notes) |
    sorted(requested_providers)
)
```

- For saved-coin jobs where an image is reused, the *stored file path plus file
  size and mtime* hash stands in for the byte hash (avoids re-reading large
  files) — recorded on the artifact row so a changed saved image yields a
  different fingerprint (edge case: retry after inputs changed).
- Unique index `uix_deep_jobs_active_fingerprint` on
  `(user_id, input_fingerprint, active_key)` where `active_key` is a generated
  column: `1` while status ∈ {`queued`,`running`}, else `id` (SQLite supports
  this via a stored expression column; alternatively a partial unique index
  `WHERE status IN ('queued','running')` created in `AutoMigrate` follow-up).
  Duplicate start ⇒ repository returns the existing job with `reused = true`,
  mirroring `AIJobRepository.EnqueueOrFindActive`.

### 2.4 Indexes

| Index | Columns | Purpose |
|---|---|---|
| `idx_deep_jobs_user_status_created` | `(user_id, status, created_at DESC)` | list/active-limit checks |
| `idx_deep_jobs_user_coin` | `(user_id, coin_id)` | saved-coin history |
| `uix_deep_jobs_active_fingerprint` | see §2.3 | idempotency |
| `idx_deep_jobs_status_heartbeat` | `(status, heartbeat_at)` | stale recovery sweep |
| `idx_deep_jobs_expires` | `(expires_at)` | retention janitor |

---

## 3. `DeepIdentificationEvent` (append-only)

| Field | Type | Notes |
|---|---|---|
| `ID` | `uint` PK | |
| `JobID` | `uint` NOT NULL | |
| `UserID` | `uint` NOT NULL | denormalized for owner-scoped replay queries |
| `Seq` | `int64` NOT NULL | monotonic **per job**, starts at 1 (FR-016) |
| `Type` | `varchar(32)` | `job_accepted`, `status_changed`, `router_selected`, `provider_started`, `provider_result`, `evaluation`, `synthesis_started`, `progress`, `heartbeat`, `terminal` |
| `PayloadJSON` | `text` | sanitized envelope payload (§ contracts/events) |
| `CreatedAt` | `time.Time` | |

Rules:

- **Uniqueness**: `uix_deep_events_job_seq` on `(job_id, seq)` — guarantees no
  duplicate sequence, hence gap-free/duplicate-free replay (FR-016, SC-003).
- Sequence assignment is inside the same transaction as the state change:
  `UPDATE deep_identification_jobs SET last_seq = last_seq + 1 WHERE id = ?
  RETURNING last_seq` then insert. No in-memory counter is authoritative.
- **Append-only**: repository exposes `AppendEvent` / `ListEventsSince` /
  `PruneEventsBefore` only — no update/delete-by-id path.
- `heartbeat` events are **not** persisted; SSE keepalive comments (`: ping`)
  are emitted by the Go SSE handler every 15 s and carry no sequence.
- Retention: pruned 24 h after the job's terminal timestamp; pruning stamps
  `Job.EventsPrunedAt` so replay can answer `stream_truncated` truthfully
  (FR-017).
- Index `idx_deep_events_job_seq` on `(job_id, seq)` (covered by the unique
  index) and `idx_deep_events_created` on `(created_at)` for the janitor.

### 3.1 Event replay state machine

```text
client GET .../events?since=N   (or Last-Event-ID: N)
   │
   ├── job not found / not owner ──> 404
   ├── N < earliest_retained_seq  ──> emit `stream_truncated` {status, earliestSeq, lastSeq} then retained tail
   ├── N >= last_seq && terminal  ──> emit current `terminal` snapshot, close
   └── otherwise                  ──> replay (N, last_seq] from DB, then subscribe to broker for live tail
                                       └── on terminal event: flush, send `event: end`, close
```

---

## 4. `DeepIdentificationProviderRun`

| Field | Type | Notes |
|---|---|---|
| `ID` | `uint` PK | |
| `JobID`, `UserID` | `uint` | |
| `Provider` | `varchar(24)` | `nomisma` \| `numista` \| `ngc` \| `ocre` \| `rpc` |
| `Status` | `varchar(20)` | §2.2 vocabulary |
| `Automatable` | `bool` | false ⇒ `not_automated` path is expected, not an error |
| `ClaimsJSON` | `text` | typed claims + citations (see contracts) |
| `Confidence` | `float64` | 0–1, provider-reported |
| `CallCount` | `int` | quota accounting (Numista budget) |
| `LatencyMS` | `int` | |
| `ErrorKind` | `varchar(32)` | typed only (`timeout`, `quota`, `unconfigured`, `upstream`, `invalid_response`) |
| `StartedAt`, `CompletedAt` | `*time.Time` | |

Unique index `uix_deep_provider_run_job_provider` on `(job_id, provider)` —
one row per provider per job attempt (retries create a new job).

---

## 5. `DeepIdentificationArtifact`

| Field | Type | Notes |
|---|---|---|
| `ID` | `uint` PK | |
| `JobID`, `UserID` | `uint` | |
| `Role` | `varchar(12)` | `obverse` \| `reverse` \| `hint` (distinct from `models.ImageType`) |
| `Origin` | `varchar(12)` | `uploaded` \| `saved_coin_image` |
| `SourceCoinImageID` | `*uint` | set when `Origin = saved_coin_image` |
| `FilePath` | `varchar(512)` | `<uploadDir>/deep-jobs/job-<id>/<seq>-<role><ext>`; empty for saved-image reuse |
| `ContentHash` | `char(64)` | fingerprint input |
| `ByteSize` | `int64` | |
| `MimeType` | `varchar(40)` | from magic-byte detection |
| `Ephemeral` | `bool` | `true` for every `hint` artifact |
| `DeletedAt` | `*time.Time` | set when file removed at terminal cleanup |
| `CreatedAt` | `time.Time` | |

Rules (FR-030, FR-021, SC-004):

- `hint` artifacts are **always** `Ephemeral = true`, never referenced by
  `models.CoinImage`, never served by any coin-image endpoint, and never
  embedded/linked in `ReportJSON`.
- Terminal cleanup deletes hint files for **completed, failed and cancelled**
  alike and stamps `DeletedAt`; the row is kept (audit) until the job's result
  retention expires.
- Coin-face artifacts of `Origin = uploaded` are retained with the job (needed
  to render the report and to seed a Quick Capture draft on confirm) and are
  removed with the job at result-retention expiry.
- Uniqueness: at most one `obverse` and one `reverse` row per job
  (`uix_deep_artifact_job_role` on `(job_id, role)` where `role <> 'hint'`);
  at most 3 `hint` rows (enforced in service validation).

---

## 6. `ReportJSON` shape (stored, owner-facing)

```jsonc
{
  "schemaVersion": 1,
  "narrative": "…",                       // sanitized markdown-lite text, no emojis
  "coverage": [                            // FR-029 / SC-007: every selected provider appears
    { "provider": "numista", "status": "contributed", "callCount": 2, "note": "" },
    { "provider": "ngc",     "status": "not_automated",
      "note": "Manual reference only — NGC terms prohibit automated access",
      "linkOut": "https://www.ngccoin.com/verify/" }
  ],
  "disagreements": [
    { "field": "mint", "claims": [
        { "value": "Rome",   "provider": "numista", "citation": "https://…", "confidence": 0.7 },
        { "value": "Lugdunum","provider": "nomisma","citation": "http://nomisma.org/id/lugdunum","confidence": 0.6 }
      ], "resolution": "unresolved" }
  ],
  "unresolvedQuestions": ["Reverse legend partially illegible; RIC number not confirmed."],
  "attributions": [
    { "provider": "numista", "text": "Source: Numista", "identifier": "N#12345" },
    { "provider": "nomisma", "text": "Data: Nomisma.org (CC BY)" }
  ],
  "partialSuccess": true,
  "generatedAt": "2026-08-15T13:04:11Z"
}
```

## 7. `ProposalJSON` shape (editable draft, confirm-gated)

```jsonc
{
  "schemaVersion": 1,
  "fields": {
    "denomination": {
      "proposed": "Denarius",
      "confidence": 0.82,
      "evidence": [{ "provider": "numista", "citation": "https://en.numista.com/catalogue/pieces12345.html", "excerpt": "…" }],
      "ownerEdited": false,
      "ownerValue": null,
      "accepted": null            // null = undecided, true/false = per-field decision (SC-009)
    }
    // … only fields that exist on models.Coin / QuickCaptureDraft are allowed (allowlist)
  },
  "targetCoinId": 42,             // null for intake
  "sourceReportGeneratedAt": "2026-08-15T13:04:11Z"
}
```

- **Field allowlist**: only fields writable through
  `services.CoinService.UpdateCoinWithFields` / `QuickCaptureDraft` create
  payloads are permitted; anything else is dropped at ingest (no silent new
  write surface — Principle IV, F012 allowlist precedent).
- `ownerEdited`/`ownerValue` preserve the AI-vs-owner distinction (FR-032).

### 7.1 Draft / proposal application state machine (FR-031, FR-033)

```text
[proposed] ──owner edits (PATCH proposal)──> [proposed]     (job stays terminal; coin untouched)
     │
     ├── intake confirm ──> create/patch QuickCaptureDraft ──> owner Promote (existing path) ──> Coin
     │                       (job.applied_draft_id set)                                          (job.applied_coin_id set on promote)
     ├── saved-coin confirm ──> CoinService.UpdateCoinWithFields(source="deep_identification")
     │                       (job.applied_coin_id, applied_at set)
     └── source coin deleted meanwhile ──> confirm returns 409 `source_coin_missing`;
                                            report remains readable (edge case in spec)
```

Applying twice is rejected (`409 already_applied`) unless the owner explicitly
starts from the report again after the first apply — `AppliedAt` is the guard.

---

## 8. Settings (`AppSetting` keys, `services/settings_service.go`)

| Key constant | Default | Purpose |
|---|---|---|
| `SettingDeepIdentificationEnabled` | `false` | admin feature flag (FR-008); blocks **new starts only** |
| `SettingDeepIdentificationWorkerCount` | `2` | global concurrency |
| `SettingDeepIdentificationMaxActivePerUser` | `1` | per-user cap |
| `SettingDeepIdentificationQueueDepth` | `32` | backpressure → `503` |
| `SettingDeepIdentificationHardTimeoutSeconds` | `300` | FR-014 ceiling |
| `SettingDeepIdentificationEventRetentionHours` | `24` | FR-017 |
| `SettingDeepIdentificationResultRetentionDays` | `90` | FR-034 |
| `SettingDeepIdentificationMaxProviders` | `4` | FR-013 |
| `SettingDeepIdentificationNumistaCallBudget` | `4` | license/quota guard |
| `SettingDeepIdentificationOCREEnabled` | `false` | gate G-OCRE |
| `SettingDeepIdentificationRPCEnabled` | `false` | gate G-RPC (stays false until written permission) |

---

## 9. Migration & compatibility

- Append the four new models to the existing `AutoMigrate(...)` call in
  `src/api/database/database.go:36`. Additive only; no existing column is
  altered or dropped.
- Add `TestDeepIdentificationModelsAutoMigrate` to
  `src/api/database/migration_test.go` following
  `TestQuickCaptureModelsAutoMigrate`.
- Rollback = disable `SettingDeepIdentificationEnabled`; orphan tables are inert
  (no other feature reads them). No data migration is required in either
  direction.
