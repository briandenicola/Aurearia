# Feature Specification: Structured Numismatic Catalog References

**Feature Branch**: `214-structured-numismatic-catalog-references`  
**Created**: 2026-05-30  
**Status**: Draft  
**Input**: GitHub issue #214

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Persist and validate structured references (Priority: P1)

As a collector, I want each coin to store structured references (catalog, volume, number, certainty, uri) so attribution is accurate and machine-usable.

**Why this priority**: This is the data foundation for UI, AI, and export work.

**Independent Test**: Create and update coins through API using mixed catalogs; verify validation, per-catalog volume rules, and persistence of multiple references.

**Acceptance Scenarios**:

1. **Given** a coin payload with references, **When** it is saved, **Then** references persist in a one-to-many table.
2. **Given** a catalog requiring volume (e.g., RIC), **When** volume is omitted, **Then** validation rejects request.
3. **Given** a valid catalog/volume/number combination, **When** saved, **Then** number preserves qualifiers as string (`256a`, `cf. 88`).

---

### User Story 2 - Manage and browse references in UI (Priority: P1)

As a collector, I want to add/edit/remove references in Coin Detail and filter by era so attribution is manageable and discoverable in app.

**Why this priority**: Data model value is limited without user-facing workflows.

**Independent Test**: Use Coin Detail page and collection filters to manage references and era display end-to-end without direct API calls.

**Acceptance Scenarios**:

1. **Given** a coin detail page, **When** user adds/edits/removes references, **Then** UI reflects saved values and validation errors.
2. **Given** a reference with URI, **When** rendered, **Then** URI appears as clickable link.
3. **Given** collection browsing, **When** user filters by era, **Then** list updates to matching coins only.

---

### User Story 3 - AI discovery + export interoperability (Priority: P2)

As a collector, I want AI attribution suggestions and exports to include structured references and era for portability.

**Why this priority**: This extends structured references into automation and interoperability surfaces.

**Independent Test**: Run AI discovery and CSV/JSON export on attributed coins; verify structured reference payload and era fields appear consistently.

**Acceptance Scenarios**:

1. **Given** AI discovery result, **When** candidate attribution is generated, **Then** it returns structured `CoinReference` with certainty.
2. **Given** catalogs with known online authorities (OCRE/RPC), **When** URI lookup succeeds, **Then** uri is populated; otherwise it remains optional.
3. **Given** CSV/JSON export, **When** coin has references and era, **Then** both are included in export output.

### Edge Cases

- Same catalog/volume/number duplicate entries on a coin must dedupe or reject deterministically.
- Catalog enum can expand without code rewrite of validation rules.
- Volume requirement differs by catalog and cannot be globally hardcoded.
- Missing online authority must not block save.
- Legacy `referenceText`/`referenceURL` flows remain backward compatible.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST add `CoinReference` model with fields `catalog`, `volume`, `number`, `certainty`, `uri`.
- **FR-002**: System MUST add `era` field on `Coin` with values `ancient|medieval|modern`.
- **FR-003**: System MUST support zero-to-many references per coin.
- **FR-004**: System MUST introduce a catalog registry table/seed source that defines era and volume-required rules per catalog.
- **FR-005**: System MUST validate references using catalog registry rules (not hardcoded per endpoint).
- **FR-006**: System MUST treat `number` as string with qualifier support.
- **FR-007**: System MUST expose API read/write operations for references in authenticated coin workflows.
- **FR-008**: System MUST provide UI controls to add/edit/remove references and render URI links.
- **FR-009**: System MUST provide UI filtering by `era`.
- **FR-010**: System MUST include references + era in CSV/JSON export.
- **FR-011**: System MUST update AI discovery flow to output candidate `CoinReference` with certainty and optional authority URI.
- **FR-012**: System MUST preserve backward compatibility for existing free-text reference fields.

### Key Entities *(include if feature involves data)*

- **Coin**: existing entity with new `era` classification.
- **CoinReference**: new one-to-many normalized attribution records.
- **CatalogRegistry**: per-catalog metadata and validation rules.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of reference writes pass through registry-driven validation.
- **SC-002**: Coin detail supports complete CRUD of structured references with no page reload.
- **SC-003**: Export payloads include era and structured references for all attributed coins.
- **SC-004**: AI discovery emits structured references (with certainty) for attribution-capable results.
