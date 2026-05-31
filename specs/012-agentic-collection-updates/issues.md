# Tracking Issues — Epic F012

## Issue 1 — Structured Catalog References (Feature 1)

**Title**: `feat: structured catalog references for coins`  
**Depends on**: none

### Scope

- Introduce normalized `CoinReference` model and migrations.
- Add catalog-aware validation and authority URI support.
- Extend API and UI coin flows to read/write structured references.
- Preserve legacy `referenceText` + `referenceURL` behavior.

### Acceptance

- Coins support multiple normalized references.
- Validation rejects malformed known catalog numbers.
- Existing clients using legacy reference fields continue working.

---

## Issue 2 — Agentic Coin Entry (Feature 2)

**Title**: `feat: AI intake draft + confirm coin creation`  
**Depends on**: Issue 1 (soft), may begin in parallel

### Scope

- Add `coin_intake` team in Python agent.
- Add draft/confirm API endpoints in Go and typed payloads in web.
- Build intake review UI showing confidence + source evidence.
- Commit path creates coin only after explicit user confirmation.

### Acceptance

- Draft generated from photos/OCR/lookups in structured format.
- User can edit before commit.
- Commit creates coin + journal entry with AI source tag.

---

## Issue 3 — Collection Chat Tool Layer + In-App Chat (Feature 3)

**Title**: `feat: collection chat over transport-agnostic tools`  
**Depends on**: none for read-only slice; write slice independent of Issue 4

### Scope

- Build Go collection tool layer (`get_coin`, `query_coins`, `aggregate`).
- Add collection mode to existing chat drawer and route via `collection_chat`.
- Add write operations with two-phase `propose_update` / `commit_update`.
- Enforce allowlist and disambiguation for ambiguous targets.

### Acceptance

- Chat answers collection questions from user-scoped data.
- Write actions require proposal token + explicit commit.
- Journal records each committed update with source `collection_chat`.

---

## Issue 4 — External Collection Tool Server (Feature 4)

**Title**: `feat: external tool server adapter with read/write parity`  
**Depends on**: Issue 3 (hard dependency on shared tool layer)

### Scope

- Expose same tool operations for external clients (OpenAPI-first, MCP-compatible adapter).
- Add API-key capability controls for read/write.
- Preserve proposal-token commit requirement for external writes.
- Publish setup and integration documentation.

### Acceptance

- External client read/write behavior matches in-app collection tools.
- Cross-user access attempts are denied server-side.
- External commits are journaled with source `external_tool_server`.
