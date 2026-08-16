# Feature Specification: Identify Coin Capture Wizard

**Feature Branch**: `348-identify-coin-wizard`
**Created**: 2026-08-16
**Status**: Implemented
**Input**: Replace the ambiguous Identify Coin capture screen with explicit
Obverse, Reverse, and Notes steps.

## User Story

As a collector identifying a coin, I can see what evidence is required, what
is optional, and what to add next without guessing how uploaded images will be
interpreted.

## Requirements

- **FR-001**: Identify Coin MUST present three explicit steps in order:
  Obverse, Reverse, and Notes.
- **FR-002**: Obverse MUST be visibly required. Reverse and Notes MUST be
  visibly optional.
- **FR-003**: After adding an obverse image, the collector MUST be able to run
  `Analyze Photos` immediately or continue to Reverse.
- **FR-004**: From Reverse, the collector MUST be able to run analysis or
  continue to Notes, whether or not a reverse image was added.
- **FR-005**: Notes MUST accept bounded text, one supporting image, or both.
- **FR-006**: Notes text MUST be sent to Quick Identify as untrusted evidence
  and MUST be retained when the Quick Capture draft is saved.
- **FR-007**: Each uploaded image MUST retain its explicit role; omitting a
  reverse MUST NOT cause a Notes image to be stored as the reverse.
- **FR-008**: Camera access MUST remain user-initiated and gallery images MUST
  retain Feature 347 normalization.
- **FR-009**: Existing Quick Identify, Numista, NGC-first, Deep Analysis, and
  draft-review behavior MUST remain compatible.
- **FR-010**: Notes text MUST be limited to 2,000 characters at both UI and API
  boundaries.

## Success Criteria

- **SC-001**: A collector can identify from only an obverse in one step.
- **SC-002**: A collector can navigate Obverse to Reverse to Notes with clear
  current, complete, required, and optional states at mobile width.
- **SC-003**: A Notes image with no reverse reaches analysis as supporting
  evidence and is persisted as a detail image.
- **SC-004**: Notes text reaches the AI request and the saved draft.
- **SC-005**: Exact wizard navigation, role mapping, notes propagation, and
  validation behavior are covered by automated tests.
