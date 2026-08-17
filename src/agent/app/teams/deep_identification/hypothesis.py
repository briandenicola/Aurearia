"""Deterministic quick-evidence -> `CoinHypothesis` adapter (Phase 8 seam).

This is the CURRENT source behind the hypothesis seam (contracts/
vision-hypothesis.md §1): it composes a `CoinHypothesis` purely from
`quick_evidence` (Go's Quick Identify output, already attached to every
deep-identification request) with **no LLM call of any kind** — the vision
LLM call itself is Phase 3/4 scope. Every downstream consumer (synthesis
today; router/query-construction/evaluator in later phases) reads the
`hypothesis` state key, never this module directly, so replacing this
function's body with the real vision-derived hypothesis later is a
drop-in swap.
"""

from app.models.hypothesis import CoinHypothesis, HypothesisField
from app.models.requests import QuickEvidence

# Coin-field vocabulary keys read straight off `quick_evidence.coin_fields`
# (contracts/vision-hypothesis.md §1). These are exactly the camelCase keys
# Go's CoinLookupService.copyCoinFieldsFromMap/mergeLinePatternFields already
# normalize to (src/api/services/coin_lookup_service.go), so no aliasing is
# needed on this side.
_HYPOTHESIS_FIELDS = (
    "ruler",
    "denomination",
    "material",
    "mint",
    "dateRange",
    "era",
    "obverseInscription",
    "reverseInscription",
    "obverseDescription",
    "reverseDescription",
    "weightGrams",
    "diameterMm",
)

# Go's models.Era / models.Material enums (src/api/models/coin.go). Go casts
# a proposed field value straight into these typed columns with no
# validation of its own (deep_identification_proposal.go::
# setCoinFieldFromProposalValue), so a value outside these fixed sets must
# never be forwarded as a proposed field here — it would silently write an
# invalid enum string into the coin row.
_VALID_ERAS = {"ancient", "medieval", "modern"}
_VALID_MATERIALS = {"gold", "silver", "bronze", "copper", "electrum", "other"}

# quick_evidence.confidence is a coarse low/medium/high tier (Go's
# CoinLookupService.determineConfidence), not a per-field probability. In the
# absence of a real per-field confidence signal, every field derived from
# this adapter shares the run's overall tier as a deterministic stand-in.
_CONFIDENCE_BY_TIER = {"high": 0.75, "medium": 0.55, "low": 0.35}
_DEFAULT_CONFIDENCE = 0.5


def _tier_confidence(quick_evidence: QuickEvidence | None) -> float:
    tier = (quick_evidence.confidence or "").strip().lower() if quick_evidence else ""
    return _CONFIDENCE_BY_TIER.get(tier, _DEFAULT_CONFIDENCE)


def _canonical_era(value: str) -> str | None:
    normalized = value.strip().lower()
    return normalized if normalized in _VALID_ERAS else None


def _canonical_material(value: str) -> str | None:
    normalized = value.strip().lower()
    if normalized not in _VALID_MATERIALS:
        return None
    return normalized[0].upper() + normalized[1:]


def _ngc_observations(quick_evidence: QuickEvidence) -> str:
    ngc = quick_evidence.ngc
    if ngc is None:
        return ""
    parts: list[str] = []
    if ngc.cert_number:
        parts.append(f"NGC cert {ngc.cert_number}")
    if ngc.grade:
        parts.append(f"graded {ngc.grade}")
    return "; ".join(parts)


def build_hypothesis_from_quick_evidence(quick_evidence: QuickEvidence | None) -> CoinHypothesis:
    """Deterministic, LLM-free `CoinHypothesis` built from `quick_evidence`.

    Never guesses: a field is included only when `coin_fields` carries a
    non-empty value for it, and `era`/`material` are further dropped unless
    they canonicalize onto Go's fixed enum sets. Validation failure or an
    absent `quick_evidence` yields the typed empty hypothesis
    (`legible=False`), matching the contract's failure-mode guarantee — the
    pipeline never fails for lack of a hypothesis (spec FR-006).
    """
    if quick_evidence is None:
        return CoinHypothesis(legible=False)

    fields = quick_evidence.coin_fields or {}
    confidence = _tier_confidence(quick_evidence)

    values: dict[str, HypothesisField] = {}
    for key in _HYPOTHESIS_FIELDS:
        raw = fields.get(key)
        if not isinstance(raw, str) or not raw.strip():
            continue
        value = raw.strip()
        if key == "era":
            canonical = _canonical_era(value)
            if canonical is None:
                continue
            value = canonical
        elif key == "material":
            canonical = _canonical_material(value)
            if canonical is None:
                continue
            value = canonical
        values[key] = HypothesisField(value=value, confidence=confidence)

    observations = _ngc_observations(quick_evidence)[:500]

    return CoinHypothesis(
        **values,
        observations=observations,
        legible=bool(values) or bool(observations),
    )
