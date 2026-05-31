# Quickstart: Collection Chat Over Transport-Agnostic Tools (#217)

## Prerequisites

- Go API, Vue web app, and Python agent service running (`task up-all` from repo root).
- Authenticated user with at least 5 coins spanning multiple eras/materials.
- Existing chat drawer accessible in web UI.

## Scenario 1: Read-only collection Q&A in chat

1. Open chat drawer and switch to **Collection** mode.
2. Ask:
   - "How many Roman silver coins do I own?"
   - "Show my top 3 highest-value coins."
3. Verify responses are grounded in owned collection data and include no cross-user records.
4. Ask a no-result query (e.g., "How many electrum coins do I own?") and verify deterministic no-results response.

## Scenario 2: Propose update (no write yet)

1. Ask: "Update my Hadrian denarius notes to mention provenance from recent show."
2. If multiple matches are found, pick one from disambiguation choices.
3. Verify assistant returns proposal preview with:
   - target coin,
   - changed fields,
   - proposal token reference/expiry.
4. Verify coin remains unchanged before commit.

## Scenario 3: Explicit commit persists exactly once

1. Submit explicit confirm action using proposal token.
2. Verify API returns committed status and changed fields.
3. Refresh coin detail and confirm updates persisted.
4. Verify coin journal contains entry tagged with `collection_chat`.
5. Retry same commit and verify it is rejected with no additional write.

## Scenario 4: Cancel proposal

1. Create a proposal, then cancel it.
2. Verify status transitions to `cancelled`.
3. Verify commit after cancellation is rejected.

## Negative Scenarios

1. Commit with wrong token -> deterministic rejection, no write.
2. Commit after proposal expiry -> deterministic rejection, no write.
3. Attempt to update non-allowlisted field (e.g., `category`) -> proposal rejected.
4. Attempt to target another user's coin id in prompt/tool args -> access denied.

## Compatibility Check

1. Switch chat back to default search mode.
2. Run a normal coin marketplace search request.
3. Verify existing search chat behavior and suggestion rendering remain unchanged.
