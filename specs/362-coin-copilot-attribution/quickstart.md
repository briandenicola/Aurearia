# Quickstart: Verify Coin Copilot Attribution Integration

## Prerequisites

- Work on the existing `beta` branch.
- Configure an AI provider/model that supports Coin Copilot tool calling.
- Enable `CoinCopilotEnabled` and `DeepIdentificationEnabled`.
- Keep OCRE disabled unless its separately gated behavior is under test.
- Prepare owner A and owner B; create collection, wishlist, and active draft
  fixtures with distinct obverse/reverse images.

## Core acceptance walkthrough

1. Open a saved owned coin and ask Coin Copilot to attribute it.
2. Confirm the drawer identifies the exact coin and returns a Deep job with an
   `Open Deep Analysis` link.
3. Repeat the request before completion. Confirm the same active job id is
   returned and provider execution is not duplicated.
4. Disconnect/reconnect the Copilot stream. Confirm the checkpoint restores the
   handoff without another tool call.
5. Open `/deep-analysis/{jobId}`. Confirm the existing progress/review page is
   used.
6. For a completed result, verify the chat summary preserves confidence,
   citations, conflicts, provider coverage, no-match/partial limitations, and
   attribution from the persisted report.
7. Accept exactly one destination-valid field in the existing proposal editor
   and confirm apply. Verify every other manual field and all prior references
   remain unchanged.
8. Repeat with a wishlist coin and an active Quick Capture draft.

## Safety cases

- Ambiguous label or prompt/context disagreement asks for clarification and
  creates no job.
- Foreign and unknown coin, draft, and job ids have indistinguishable
  `not_found` behavior.
- Missing obverse, reverse, duplicate face content, invalid MIME, discarded or
  promoted draft creates no job and names the corrective step.
- Equivalent terminal result reopens without spending work; a fresh run occurs
  only after explicit rerun intent.
- Failed, cancelled, stale, changed-input, expired, and missing-result fixtures
  are never described as current success.
- Cancellation winning the admission race creates no Deep job; late Python
  output cannot settle the cancelled Copilot run.
- Disabled Deep Analysis returns a bounded unavailable result. Disabled or
  unsupported Coin Copilot keeps the pre-accept legacy chat fallback.
- Fast Identify works unchanged before and after all cases.

## Contract/tamper tests

Use canonical valid and invalid fixtures on both Go and Python sides:

- unknown fields/enums;
- wrong target/job pairing;
- out-of-range/NaN confidence;
- unsafe citation/review URLs and embedded credentials;
- oversized result/checkpoint/event;
- duplicate tool call and replay with changed target;
- prompt injection and token-shaped result text;
- fabricated accept/apply/provider override properties.

## Expected automated suites

```powershell
# Go
Set-Location src/api
go test ./services ./repository ./handlers ./integration -run 'CoinCopilot|DeepIdentification|AttributionHandoff' -count=1
go vet ./...
go test ./...

# Python
Set-Location ..\agent
ruff check app/ tests/
pytest tests/test_coin_copilot_contract.py `
  tests/test_coin_copilot_harness.py `
  tests/test_coin_copilot_security.py -v
pytest tests/ -v

# Vue
Set-Location ..\web
npm run test -- --run
npm run build
```

Also run route/OpenAPI drift tests after regenerating the documented API
surface. No deployment, migration rollback, or production provider call is part
of this planning quickstart.

## Manual mobile/PWA check

At a narrow viewport:

1. Open the Agent drawer, request attribution, background the PWA, and return.
2. Verify Copilot replay resumes at the last event sequence.
3. Tap the 44 px minimum `Open Deep Analysis` control.
4. Background and restore the Deep Analysis page; verify its independent
   backoff reconnect resumes from its last Deep event sequence.
5. Confirm report, conflicts, provider attribution, and proposal controls do not
   overflow horizontally and no duplicate editor appears in chat.
