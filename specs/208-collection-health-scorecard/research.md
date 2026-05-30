# Research — Collection Health Scorecard (v1)

## Decision 1: Use fixed weighted score with issue defaults

- **Decision**: Compute per-coin health score as a weighted composite using fixed v1 weights: metadata completeness 40%, image coverage 20%, valuation freshness 20%, AI analysis coverage 20%.
- **Rationale**: These weights are explicitly in issue #208 scope and must remain default for v1. Existing data model supports all dimensions (`models.Coin`, `models.CoinImage`, `models.CoinValueHistory`), so no speculative dimensions are needed.
- **Alternatives considered**:
  - Dynamic admin-configurable weights in v1 (rejected: expands scope and test surface).
  - Rebalanced weights from external research (rejected: conflicts with feature input defaults).

## Decision 2: Define deterministic grade thresholds

- **Decision**: Use fixed thresholds: A 90-100, B 80-89, C 70-79, D 60-69, F <60.
- **Rationale**: Deterministic thresholds simplify frontend/backend parity tests and support stable admin metrics (low-score percentage defined as `<60`).
- **Alternatives considered**:
  - Custom user thresholds (rejected: out of scope for v1).
  - 5-band equal quantiles (rejected: unstable as collection composition changes).

## Decision 3: Metadata completeness checklist keys

- **Decision**: Score metadata using required checklist keys aligned to existing coin fields: denomination, ruler, era, mint, category, material, grade, weight, diameter, rarity rating.
- **Rationale**: These fields already drive valuation readiness logic in current services and represent high-value catalog completeness.
- **Alternatives considered**:
  - Include optional narrative fields (notes/provenance text) in core metadata score (rejected: subjective and hard to validate consistently).
  - Only require 3 valuation-minimum fields (rejected: too coarse for quality scorecard).

## Decision 4: Valuation freshness buckets

- **Decision**: Use bucketed freshness scoring against latest valuation timestamp: `<=30d=100`, `31-90d=80`, `91-180d=60`, `181-365d=35`, `>365d or never valued=0`.
- **Rationale**: Buckets are easy to explain and test while still representing decay over time.
- **Alternatives considered**:
  - Continuous decay function (rejected: harder to explain/debug).
  - Binary fresh/stale valuation (rejected: insufficient signal for prioritization).

## Decision 5: Persist daily collection snapshots for 30-day trend

- **Decision**: Add a daily persisted `collection_health_snapshots` record per user and compute trend as current collection score minus score from nearest snapshot 30 days prior.
- **Rationale**: Snapshot persistence gives deterministic trend data and avoids expensive historical recomputation at request time.
- **Alternatives considered**:
  - On-demand recomputation from current coin states only (rejected: cannot reconstruct 30-day baseline).
  - Reuse value snapshots directly (rejected: financial snapshots do not encode health dimensions).

## Decision 6: Needs Attention queue ordering and quick actions

- **Decision**: Provide a dedicated queue endpoint sorted by ascending score, tie-break by oldest coin update then coin ID; each item includes action hints (`edit_metadata`, `upload_images`, `run_valuation`, `run_ai_analysis`) mapped to existing UI/API capabilities.
- **Rationale**: Keeps queue deterministic and integrates with existing workflows instead of inventing new mutation endpoints.
- **Alternatives considered**:
  - Client-side sorting from full coin list (rejected: unnecessary payload and inconsistent pagination).
  - New bespoke action APIs for each quick action (rejected: duplicates existing endpoints).

## Decision 7: Admin aggregate metrics scope

- **Decision**: Admin panel returns median per-coin score, low-score percentage (`score < 60`), and top missing checklist keys across low-score coins.
- **Rationale**: Directly matches issue scope and provides actionable quality remediation focus.
- **Alternatives considered**:
  - Mean only (rejected: less robust to outliers).
  - Aggregate across all scores without low-score focus (rejected: weak signal for cleanup prioritization).
