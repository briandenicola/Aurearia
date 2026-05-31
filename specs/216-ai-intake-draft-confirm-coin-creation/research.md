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
