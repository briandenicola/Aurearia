# Session Log: Issue #218 — External Tool Server Adapter (Full Implementation)

**Date:** 2026-06-01  
**Session:** Distributed sprint (5 agents, multi-phase)  
**Status:** Complete — Ready for merge  
**GitHub Issue:** #218  

---

## What Shipped

### Backend (Go API)

**Phase 2 — Foundational Infrastructure (Cassius, T003–T011)**
- Capability model: String-based (`"read"` or `"read,write"`) on `ApiKey`, with `HasRead()` / `HasWrite()` helpers
- Admin kill-switch: `SettingExternalToolServerEnabled` (default false) with 503 gate middleware
- Capability enforcement middleware: `RequireCapability("read"/"write")` per-route
- Per-key external rate limiter: 50 req/min per API key (stricter than 100/min in-app)
- Public route group: `/api/v1/tools` with middleware chain (gate → auth → rate limit)

**Phase 3 — Handlers & Routes (Cassius, T012–T021)**
- Six tool endpoints: 4 read (SearchMyCollection, GetCoin, CollectionSummary, TopCoinsByValue), 2 write (ProposeUpdate, CommitUpdate)
- Journal-source threading: `CommitUpdate()` (internal, source `collection_chat`) vs `CommitUpdateExternal()` (external, source `external_tool_server` + API key metadata)
- Served OpenAPI spec: `/api/v1/tools/openapi.json` (unauthenticated, YAML → JSON via `go:embed`)
- API key scope parameter: Optional `scope` field in handler, defaults to `"read"`

**Validation (Brutus, T027–T031)**
- ✅ 10 capability middleware tests (100% pass)
- ✅ Go build/vet/test (all green, 23 middleware tests total)
- ✅ Quickstart traceability (scenarios A–C, negative N1–N6, success criteria SC-001–SC-007 satisfied)
- ✅ Manual runtime verification steps documented for Brian

**BLOCK → Fix → Re-Review Cycle (Maximus → Brutus → Maximus, §18.2 Strict Lockout)**
- Initial review: 7 type assertion panic risks identified in CommitUpdate handler (BLOCK issued)
- Fix by Brutus: Comma-ok defensive guards applied to all six handlers (userID + API key metadata checks)
- Re-review: BLOCK cleared, all tests pass, zero regressions, APPROVED for merge

### Frontend (Vue/TypeScript)

**API Key UI (Aurelia, T022–T023)**
- Added `capabilities: string` field to `ApiKey` interface
- Chip-based scope selector in `SettingsDataSection.vue` (Read | Read/Write)
- Inline capability badges (blue for read, gold for read+write) in key list

**In-App Documentation (Aurelia, T024)**
- Three-section help accordion "Connecting AI Tools" (Admins, Users, Developers)
- Six-tool capability table, client setup guides, security overview

**Validation (Brutus, T030)**
- ✅ `npm run build` (0 errors, type-check clean)
- ✅ `npm run lint` (0 errors from #218 changes)

### Documentation

**End-User & API Reference (Scribe, T024–T026)**
- Comprehensive guide: `docs/external-tool-server.md` (setup, security model, tools reference, client guides, MCP, troubleshooting)
- Updated `docs/features.md` with External Tool Server section
- Updated `docs/api-reference.md` with API Keys scope + `/api/v1/tools/*` surface documentation
- Updated `docs/threat-model.md` with B-10 finding (external write surface, High, Mitigated)

**Documentation Reorganization (Scribe, T028)**
- Restructured `docs/external-tool-server.md` into three audience sections (Admins, Users, Developers)
- Updated `docs/features.md` pointer to role-based guide organization

---

## Architecture & Design

### Capability Scope Model

**Data Model:**
```
ApiKey {
  Capabilities: string  // "read" or "read,write"
  HasRead() bool        // true if capabilities contains "read" OR "write"
  HasWrite() bool       // true if capabilities contains "write"
}
```

**Enforcement:**
- String-based normalized representation avoids join tables (v1 efficiency)
- Helpers encapsulate "write implies read" logic
- Validation: `repository.ValidateCapabilities()` rejects any value other than `"read"` or `"read,write"`
- Default: `"read"` (least privilege per FR-003)

### Middleware Stack

```
ExternalToolServerEnabled()  // Kill-switch: 503 if disabled
  ↓
AuthRequired()               // API-key auth, set userId/userRole/apiKeyCapabilities/apiKeyId/apiKeyName
  ↓
ExternalAPIKeyRateLimit()    // Per-key limiter: 50 req/min
  ↓
RequireCapability("read"|"write")  // Route-specific capability check
  ↓
Handler
```

### Journal-Source Threading

**Pattern:**
```go
// Internal (in-app collection chat)
CommitUpdate(userID, proposalID, token, confirm)
  → journalSource = "collection_chat"
  → journalMetadata = nil

// External (external tool server)
CommitUpdateExternal(userID, proposalID, token, confirm, apiKeyID, apiKeyName, apiKeyCapabilities)
  → journalSource = "external_tool_server"
  → journalMetadata = {apiKeyID, apiKeyName, apiKeyCapabilities}
```

**Journal Entry:**  
Internal: `"Updated fields via collection chat"`  
External: `"Updated fields via external_tool_server [API key #123 'my-read-key']"`

### Two-Phase Write Protection

1. **Propose:** `POST /api/v1/tools/propose_update` → preview + token (no write)
2. **Confirm:** `POST /api/v1/tools/commit_update` + token + `confirm=true` → write + journal

Token expires after ~5 minutes; replay detected via proposal state machine (→ 409 conflict).

### OpenAPI-First Client Integration

**Surface:**
- Unauthenticated `GET /api/v1/tools/openapi.json` (returns YAML parsed to JSON)
- Spec describes all 6 tools with request/response schemas
- Clients: OpenWebUI, LibreChat, n8n (import directly), MCP (wrap with `mcpo`)

---

## The BLOCK → Fix Cycle

### Initial Review (Maximus, 2026-06-01)

**BLOCK Finding:** Unchecked type assertions in CommitUpdate handler
- `apiKeyID.(uint)` — panic if nil or wrong type
- `apiKeyName.(string)` — panic if nil or wrong type  
- `apiKeyCap.(string)` — panic if nil or wrong type
- Similar risks in all six handlers on `userID.(uint)`

**Risk:** Server crash (availability failure) if middleware chain bypassed or implementation changes.

### Fix by Brutus (Strict Lockout, 2026-06-01)

**Strategy:** Comma-ok defensive guards on all handlers
```go
apiKeyID, exists := c.Get("apiKeyId")
if !exists {
    c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
    return
}
apiKeyIDUint, ok := apiKeyID.(uint)
if !ok {
    c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient capability"})
    return
}
```

**Applied to:**
1. SearchMyCollection (userID check → 401)
2. GetCoin (userID check → 401)
3. CollectionSummary (userID check → 401)
4. TopCoinsByValue (userID check → 401)
5. ProposeUpdate (userID check → 401)
6. CommitUpdate (userID + 3x API key metadata checks → 401/403)

**Error Mapping:**
- userId missing/wrong type → 401 Unauthorized
- API key metadata missing/wrong type → 403 Forbidden (fail-closed)

### Re-Review by Maximus (2026-06-01)

✅ **APPROVED** — BLOCK CLEARED

- All handlers now defend against missing/wrong-type context values
- Server returns controlled HTTP errors instead of panicking
- Zero behavioral change for valid requests (success path identical)
- All tests pass (architecture + capability middleware + unit tests)
- No regressions in service/repo layers (fix scoped to handlers only)
- Previously-approved items (security, isolation, journaling, architecture) all still valid

---

## Validation Summary

| Component | Validation | Status |
|---|---|---|
| Go Backend | `go build ./...` | ✅ Pass |
| Go Backend | `go vet ./...` | ✅ Pass |
| Go Backend | `go test ./...` (23 middleware tests) | ✅ Pass |
| Vue Frontend | `npm run build` | ✅ Pass (type-check clean) |
| Vue Frontend | `npm run lint` | ✅ Pass (0 errors from #218) |
| Quickstart | Scenarios A–C + N1–N6 | ✅ Code trace satisfied |
| Success Criteria | SC-001–SC-007 | ✅ All met |
| Documentation | Threat model, API reference, guide | ✅ Complete & verified |

---

## Key Design Decisions

### 1. Capability Scope (Cassius)
- **Chosen:** String-based (`"read"`, `"read,write"`) on ApiKey field
- **Rationale:** Avoids join tables (v1 efficiency), aligns with existing AppSetting pattern
- **Recorded in:** `.squad/decisions/inbox/cassius-218-foundational.md`

### 2. Journal-Source Attribution (Cassius)
- **Chosen:** Separate service methods (`CommitUpdate` vs `CommitUpdateExternal`) with shared `commitProposalWithSource(source, metadata)` implementation
- **Rationale:** DRY (avoids duplication), clean separation of concerns (internal vs external journal source)
- **Recorded in:** `.squad/decisions/inbox/cassius-218-handlers.md`

### 3. OpenAPI Embedding (Cassius)
- **Chosen:** YAML spec embedded via `go:embed`, served as JSON via `yaml.v3` parser at runtime
- **Rationale:** Binary-bundled spec cannot drift from repo source; YAML → JSON parsing at runtime negligible overhead
- **Alternative considered:** Pre-generate JSON (adds build step; avoided)
- **Recorded in:** `.squad/decisions/inbox/cassius-218-handlers.md`

### 4. Defensive Error Handling (Brutus)
- **Chosen:** Comma-ok guards + fail-closed 401/403 responses for missing/wrong-type context values
- **Rationale:** Defense in depth (Principle XI); treats unexpected state as auth failure, not server error
- **Recorded in:** `.squad/decisions/inbox/brutus-218-block-fix.md`

### 5. Frontend Scope UI (Aurelia)
- **Chosen:** Chip-based toggle (Read | Read/Write) in API key creation
- **Alternatives considered:** Radio buttons (rejected—chips more compact), dropdown (rejected—overkill for binary)
- **Recorded in:** `.squad/decisions/inbox/aurelia-218-keyscope-ui.md`

### 6. Documentation Structure (Scribe)
- **Chosen:** Three-audience organization (Administrators, Users, Developers) for discoverability
- **Rationale:** Reduces cognitive load; users find role-specific guidance without wading through irrelevant sections
- **Recorded in:** `.squad/decisions/inbox/scribe-218-persona-docs.md`

---

## Unresolved Items

**None blocking.** Manual runtime verification steps documented in `brutus-218-validation.md` for Brian (admin toggle, key creation, scenario tests, client discovery).

---

## Principle Compliance

- ✅ **Principle I (Layered Architecture):** Handlers → services → repository, no violations
- ✅ **Principle XI (Security Hardening):** Defensive coding, fail-closed, generic error messages, least-privilege defaults
- ✅ **Principle XII (Auth & Token Policy):** API key auth, kill switch, capability scoping, no JWT on external surface
- ✅ **Principle XIII (PWA/Mobile):** Design tokens, no emojis, responsive chips/badges
- ✅ **§17 Quality Gate:** Build + vet + test all pass
- ✅ **§18.2 Strict Lockout:** BLOCK issued, fix by independent team member (Brutus), lockout cleared
- ✅ **§21 Definition of Done:** All 31 tasks complete, code landed, documentation published

---

## What's Next

1. **Manual verification (Brian):**
   - Admin toggle enable/disable
   - API key creation with read/write scopes
   - Scenario A–C runtime tests (read operations, two-phase write, OpenAPI discovery)
   - Client import tests (OpenWebUI, LibreChat, n8n, MCP)

2. **Merge to main** after manual verification complete

3. **Future enhancements (v2 backlog):**
   - Per-key read/write rate limit distinction (currently unified 50 req/min)
   - Additional allowlisted fields based on user feedback
   - Optional MCP server for tighter client coupling

---

## Handoff

All code complete, tested, documented, and approved. Issue #218 ready for Brian's manual verification and merge to main.

---

**Agents Involved:** Cassius (backend), Aurelia (frontend), Brutus (testing), Maximus (review), Scribe (docs/orchestration)  
**Coordinator:** Team lead (merged specs/218-*/tasks.md, committed 6b01bc8)
