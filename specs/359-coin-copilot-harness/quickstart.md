# Quickstart: Coin Copilot Read-Only Harness (#721)

## Prerequisites

- Go API, Vue app, and Python agent service running.
- Authenticated owner with several coins across categories/rulers.
- Admin AI provider configured.
- `CoinCopilotEnabled=true`.
- Selected model passes the capability check.

## Scenario 1 — Multi-tool read-only run

1. Open the existing app-wide Agent drawer.
2. Ask: "Compare my Flavian bronzes with the rest of my Roman collection and
   identify the three clearest gaps."
3. Verify a durable run is created and progress shows a plan plus multiple
   sequential tool calls.
4. Verify the final answer cites only owned collection facts.
5. Verify no write, market, auction, price-trend, similar-lot, deep-analysis,
   filesystem, shell, arbitrary HTTP, or database tool was available.

## Scenario 2 — Disconnect and replay

1. Start a run and record the latest received SSE sequence.
2. Close/reload the drawer after at least one `tool_completed` event.
3. Reconnect with `?since=<lastSeq>`.
4. Verify all later events arrive once, in order, followed by live events.
5. Restart the Go API during a test run and verify stale recovery pauses from
   the latest checkpoint or fails with `execution_lost` when no checkpoint
   exists.

## Scenario 3 — Clarification and resume

1. Ask an ambiguous question such as "Compare my bronzes with the important
   missing rulers" where inclusion of sold coins changes the result.
2. Verify `clarification_required` then `run_paused`.
3. Resume with the displayed checkpoint version, a new idempotency key, and an
   answer.
4. Verify `run_resumed` references a new execution id and the same run id.
5. Replay the same resume request and verify no second execution starts.
6. Submit a stale checkpoint version and verify `409`.

## Scenario 4 — Cancellation

1. Cancel a queued run and verify immediate `run_cancelled`.
2. Cancel a running run and verify `cancel_requested` state followed by one
   `run_cancelled` event.
3. Verify no later tool/result frame is persisted after cancellation wins.
4. Repeat cancel and verify it is idempotent.
5. Race cancel against completion and verify exactly one terminal event.

## Scenario 5 — Feature and model fallback

1. Set `CoinCopilotEnabled=false`; open/send from the same drawer.
2. Verify the existing `/api/agent/chat` supervisor flow is used and no
   Copilot rows are created.
3. Enable the flag with an Ollama model whose `/api/show` capabilities omit
   `tools`.
4. Verify capability mode remains `legacy`.
5. Configure a capable model and verify mode becomes `copilot`.

## Scenario 6 — Owner isolation and tamper resistance

1. Create a thread/run as user A.
2. As user B, attempt read, stream, cancel, and resume; verify `404` for all.
3. Tamper with execution token signature, run id, execution id, allowed tool,
   expiry, and tool-call id; verify `401`/`409` and zero tool execution.
4. Attempt to call `propose_update` or `commit_update` with an execution token;
   verify the route/tool is unavailable.

## Scenario 7 — Limits, truncation, and privacy

1. Exercise iteration, tool-call, and timeout limits, including the 120-second
   default and rejection/fallback above the 150-second maximum; verify typed
   `run_failed` events and no further calls. Verify input/output token counts
   remain recorded when reported and no estimated-cost field or dollar-cost
   enforcement is exposed. Confirm iteration, tool-call, wall-clock,
   sequential-concurrency, and payload limits remain enforced.
2. Return a tool result larger than 32 KiB; verify a bounded persisted result,
   `truncated=true`, original size, and digest.
3. Seed prompt-injection text and token-shaped values in a tool result; verify
   the data is not treated as instruction and secrets do not appear in events
   or logs.
4. Inspect stored checkpoints/events and confirm there are no reasoning,
   thought, scratchpad, raw prompt, or provider-native message fields.

## Scenario 8 — Retention

1. Age terminal events beyond 7 days and run the janitor; verify
   `stream_truncated` plus the retained terminal snapshot behavior.
2. Age checkpoints beyond 30 days; verify checkpoint/tool detail is pruned but
   the final answer remains readable on the thread.
3. Age a paused run beyond 7 days; verify it becomes failed with
   `resume_window_expired`.
4. Delete a settled thread and verify all descendant rows are removed.

## Validation commands

```powershell
Set-Location src\api
go test ./...
go vet ./...
go build ./...

Set-Location ..\agent
ruff check app tests
pytest tests -v

Set-Location ..\web
npm run test
npm run build
```

Regenerate and verify the API artifacts before Feature 359 merges:

```powershell
task openapi
git diff --exit-code -- src\api\docs\docs.go src\api\docs\swagger.json src\api\docs\swagger.yaml docs\openapi.json
Push-Location src\api
go test . -run TestRegisteredAPIRoutesAreDocumentedInOpenAPI -count=1
Pop-Location
```

The generated artifacts and `docs/api-reference.md` must describe the same
public Coin Copilot routes. The contract intentionally exposes token usage but
no estimated-cost field or dollar-cost limit.
