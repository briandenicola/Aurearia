# Implementation Plan: Measured Numista Text-Query Tuning

**Branch**: `342-numista-text-query-tuning` | **Date**: 2026-08-11 | **Spec**: [spec.md](./spec.md)

## Summary

Move generated Numista query ownership from divergent Go/TypeScript helpers to
one pure, versioned Go builder. Add a local authenticated proposal endpoint for
direct and draft workflows, reuse the builder in non-NGC photo analysis, add
explicit query-source metadata to the existing lookup contract, and allow the
lookup service one deterministic relaxed retry only after an empty verified
generated search. No image search or persistence change is included.

## Governance Decision

PR #602 is already merged. Feature 341 is therefore landed and its artifacts
are historical authority for shipped behavior, not editable planning space.
Feature 342 is the smallest constitution-compliant follow-up because it changes
cross-path contract behavior and requires coordinated Go/Vue/tests/OpenAPI;
a backlog card would be too weak, and retroactive amendment would violate §0,
§18.2, and Principle VIII.

## Technical Context

**Language/Version**: Go 1.26.5; Vue 3 strict TypeScript  
**Storage**: None  
**Testing**: Go unit/integration tests, `httptest`, Vitest, strict Vue build,
OpenAPI route-drift tests, sanitized fixture replay  
**Performance**: Proposal generation is local; one extra application provider
search only for empty verified generated queries  
**Constraints**: Shared quota, exact manual query preservation, no raw query
telemetry, no image search, NGC-first/no-eager behavior

## Constitution Check

| Authority | Result |
|---|---|
| Principle I | PASS: handler binds/maps; services own generation and fallback; no DB work. |
| Principle II | PASS: Vue calls Go; no new Python/LLM or direct provider call. |
| Principle III | PASS: additive typed DTOs and OpenAPI; no frontend query assembly. |
| Principle IV | PASS: one builder, one proposal route, one bounded fallback. |
| Principle V | PASS: authenticated route, bounded fields, no raw text telemetry. |
| Principle VI | PASS: existing editable panel and accessibility behavior retained. |
| Principles VII/IX, §17/§21 | PASS at plan: exact path, contract, fixture, privacy, and regression tasks are specified. |
| Principle VIII | PASS: this spec and decision inbox record the cross-path ownership choice. |

No ADR is required: this is a proportional refinement within ADR 0007's
existing shared Numista service boundary, not a new service, storage model,
security posture, or third-party integration.

## Affected Workflow Trace

### Saved coin / direct

```text
CoinReferencesSection
  -> CoinNumistaPanel (build evidence only)
  -> POST /api/numista/query-proposal
  -> NumistaLookupPanel (display/edit + source marker)
  -> POST /api/numista/lookup
  -> optional POST /api/numista/enrich
  -> existing explicit CoinReference persistence
```

Remove direct query assembly from `src/web/src/utils/numistaLookup.ts`.

### Non-NGC photo

```text
CoinLookupPage -> POST /api/coins/lookup
  -> CoinLookupService extracts bounded evidence
  -> shared NumistaQueryBuilder creates proposal (no provider request)
  -> NumistaLookupPanel -> existing lookup/enrich calls
  -> existing draft save/selection behavior
```

Replace `buildPhotoNumistaQuery`; retain evidence extraction.

### Quick Capture draft

```text
QuickCaptureDraftPage (build evidence only)
  -> POST /api/numista/query-proposal
  -> NumistaLookupPanel -> existing lookup/enrich calls
  -> existing selected-reference update/promotion
```

### NGC path

A usable NGC result continues to suppress automatic proposal generation and
all Numista provider access. `Also search Numista` remains explicit. If the
panel has no server proposal, collector input is `manual`; it receives no
automatic relaxation.

## Canonical Ownership

Add `src/api/services/numista_query.go` with a pure
`NumistaQueryBuilder`/`NumistaQueryPlan`:

- version: `numista-query-v2`;
- primary: subject + reverse legend/type + normalized mint;
- relaxed: subject + normalized mint;
- exact, versioned mint alias table;
- no provider, cache, telemetry, Gin, repository, or mutable global access.

`NumistaLookupService` receives the builder through constructor injection and
uses it to verify generated requests and obtain the relaxed query.
`CoinLookupService` receives/reuses the same builder for photo proposals.
`NumistaHandler` exposes proposal generation through a thin authenticated
route. Vue owns only evidence collection and edit-state semantics.

## Contract Decisions

### Proposal

`POST /api/numista/query-proposal` accepts existing `path` and
`NumistaEvidence`, and returns:

```json
{
  "query": "Constantine I VOT XX Nicomedia",
  "querySource": "generated",
  "generationVersion": "numista-query-v2"
}
```

The route performs no external call and records no provider telemetry.

### Marker semantics

- `generated`: untouched proposal; server must rebuild an exact match using
  the supplied evidence/version before enabling fallback.
- `user-edited`: a proposal initialized the control and a collector input
  event changed it. Sticky for that mounted panel session.
- `manual`: no proposal initialized the control.
- Programmatic response updates and parent evidence refreshes do not mark a
  query edited.
- A falsely labeled or stale `generated` query is safely downgraded to
  `user-edited`; it is still searched exactly once.

### Fallback

The existing `POST /api/numista/lookup` remains the only search endpoint.
Lookup performs:

1. exact primary search;
2. only after `empty` + verified `generated`, one distinct relaxed search;
3. return the final outcome.

No fallback follows success, unconfigured, quota-limited, timeout,
unavailable, cancellation, validation failure, manual, or user-edited input.
Existing low-level transient transport retry remains unchanged.

### Effective-query reporting

Additive response fields:

- `querySource`;
- `searchAttempt`: `primary` or `relaxed`;
- `searchAttemptCount`: `1` or `2`.

`effectiveQuery` is the exact query used for the returned/final result.
The editable input remains the collector's submitted primary text; the panel
shows a separate message when the effective query is relaxed.

## Cache and Telemetry

- Cache keys remain derived from the actual query and result limit. Primary
  and relaxed queries are separate reusable entries.
- A cached primary empty can trigger the one relaxed attempt.
- Final cache metadata describes the effective attempt; attempt/source enums
  provide attribution without echoing text.
- Add safe telemetry dimensions `querySource` and `searchAttempt`.
- Record primary and relaxed operations separately. Health aggregates add
  generated/edited/manual and relaxed-attempt counts only.
- Correlation remains digest-only. Raw proposal/query/evidence/alias/slab/image
  content is prohibited.

## Mint Alias Strategy

- Use a small immutable map, not a parser.
- Normalize candidate alias tokens with NFKC, case folding, and removal of
  spaces, dots, and hyphens.
- Seed exact entries: `SMN` and `SMNT` -> `Nicomedia`.
- Additional observed variants require a sanitized fixture, reviewer approval,
  and an explicit exact entry. Keep the table bounded (maximum 32 aliases in
  this feature).
- Never match prefixes/substrings, mine visible/slab text, or infer mint from
  arbitrary reverse prose.

## Evidence and Measurement

Create a sanitized fixture matrix with at least 12 cases:

- direct, photo, and draft evidence;
- `SMN`, `SMNT`, ordinary mint names, and unknown alias-like text;
- reverse legend, reverse type fallback, and missing components;
- excluded date/material/reference/prose/slab fields;
- manual and user-edited exact-query controls.

Record the landed verbose query, v2 primary, optional relaxed query, expected
candidate ID, and frozen provider response IDs. A separate sanitized
live-evidence note records observation date and candidate ranks, but no images,
credentials, owner data, raw slab text, or full prose. CI replays fixtures and
does not call Numista.

Future query tuning must remain proportional: make the smallest measured
transformation supported by sanitized fixtures, retain omitted evidence for
scoring, and do not generalize aliases, retries, scoring, or provider
infrastructure without new evidence and an independently reviewed need.

## Implementation Ownership and Dependencies

1. **Cassius / backend**: models, query builder, proposal route, lookup
   fallback, cache/telemetry attribution, DI, Swagger.
2. **Aurelia / frontend**: evidence-only adapters, proposal loading, sticky
   marker, effective-query disclosure; preserve panel selection/accessibility.
3. **Brutus / QA**: fixture baseline, manual/edit exactness, request ceilings,
   cache/telemetry privacy, all direct/photo/draft/NGC regressions.
4. **Maximus / review**: verify no image-search scope, exact alias bounds,
   measured improvement, and Principle IV proportionality.

Backend contracts and builder land before frontend integration. Fixture
baseline must be frozen before implementation results are accepted.
