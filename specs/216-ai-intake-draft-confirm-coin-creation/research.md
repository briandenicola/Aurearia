# Research: AI Intake Draft + Confirm Coin Creation (#216)

## Decision 1: Persist drafts server-side with explicit lifecycle states

- **Decision**: Introduce `CoinIntakeDraft` persisted in Go API storage with status states `drafted|confirmed|discarded|expired`.
- **Rationale**: Server-side draft persistence supports explicit confirm gating, ownership checks, expiration, and audit-safe commit semantics.
- **Alternatives considered**:
  - Client-only draft state (rejected: no server-side ownership/expiry guarantees).
  - Immediate write on draft generation (rejected: violates confirm-before-write requirement).

## Decision 2: Return typed confidence + evidence metadata with each draft

- **Decision**: Draft response includes `confidenceSummary` and `evidence[]` with per-field uncertainty indicators.
- **Rationale**: Review UX requires interpretable confidence and provenance, not free-form assistant text.
- **Alternatives considered**:
  - Single free-text confidence paragraph (rejected: not machine-renderable for field-level cues).
  - No evidence payload (rejected: user cannot verify AI claims before commit).

## Decision 3: Commit endpoint applies user overrides transactionally

- **Decision**: `POST /api/coins/intake/commit` receives `draftId` + overrides; service validates ownership/state and creates coin in one transaction.
- **Rationale**: Transactional commit prevents partial writes and enforces one-time confirmation semantics.
- **Alternatives considered**:
  - Multi-step non-transactional commit (rejected: risk of coin created without draft transition/journal).
  - Agent-side commit directly to DB (rejected: violates Go API as sole writer boundary).

## Decision 4: Partial drafts are valid outputs when extraction is incomplete

- **Decision**: Intake pipeline may return partial draft plus `unresolvedFields` instead of failing entire flow.
- **Rationale**: OCR/input quality varies; partial drafts preserve user productivity while surfacing uncertainty explicitly.
- **Alternatives considered**:
  - Hard fail on missing high-confidence fields (rejected: poor UX for common low-quality images).
  - Silent fallback to blank manual form (rejected: loses AI-generated signal and context).

## Decision 5: Audit trail uses existing coin journal with `coin_intake` source

- **Decision**: Successful commits create a `CoinJournal` entry tagged with source `coin_intake`, draft id, and changed fields summary.
- **Rationale**: Reuses established auditing model and keeps provenance visible in existing activity surfaces.
- **Alternatives considered**:
  - Separate intake audit table only (rejected: duplicates audit surfaces and fragments history).
  - No journal entry for AI-created coins (rejected: fails acceptance criterion and traceability).

## Decision 6: Keep manual entry as an explicit first-class bypass

- **Decision**: Add Coin keeps a direct manual path that bypasses intake draft/commit; AI intake remains optional.
- **Rationale**: Desktop-heavy workflows still rely on fast manual entry and should not be forced through AI mediation.
- **Alternatives considered**:
  - Force intake-first for all users (rejected: unnecessary friction for experienced users).
  - Hide manual mode behind advanced settings (rejected: discoverability and speed regressions).

## Decision 7: Replace free-form prompt input with optional coin-card upload

- **Decision**: Intake draft input uses coin images plus optional coin-card image rather than an optional free-form prompt field.
- **Rationale**: Coin cards contain structured attribution clues that improve extraction quality and provenance.
- **Alternatives considered**:
  - Keep prompt-only third input (rejected: lower-quality and less verifiable than card evidence).
  - Require coin-card upload (rejected: not always available; must remain optional).

## Decision 8: PWA defaults to camera-first agentic entry with manual fallback link

- **Decision**: In PWA mode, Add Coin opens directly into the agentic intake camera view; upload remains available and a visible `Use Manual Mode instead` link is shown under the camera surface.
- **Rationale**: Mobile capture is the primary PWA workflow and benefits from immediate camera readiness while still preserving user control to bypass AI.
- **Alternatives considered**:
  - Keep neutral mode picker in PWA (rejected: extra tap/friction for the dominant mobile path).
  - Force camera with no manual link (rejected: violates optional AI and accessibility expectations).
