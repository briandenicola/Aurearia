"""`CoinHypothesis` sources (contracts/vision-hypothesis.md §1).

Two sources exist behind the same seam, forming a fail-soft degrade ladder
(spec FR-006; NOTE deviation from tasks.md T020/T027, recorded in
`.squad/decisions/inbox/cassius-vision-hypothesis.md`):

    structured vision call -> (retry once) -> prose extraction
        -> deterministic quick-evidence hypothesis -> typed-empty

`build_hypothesis_from_vision` is the primary source: it runs the same
single per-job vision LLM call `prepare_evidence_node` already makes
(no second vision call is ever introduced), binds it to the `CoinHypothesis`
schema via `get_structured_model`, and normalizes the result onto the
coin-field allowlist. `build_hypothesis_from_quick_evidence` is the
deterministic, LLM-free fallback used when the vision call fails, degrades
to nothing, or is unavailable — it is strictly better than a typed-empty
hypothesis and is what makes an unreadable-image coin (e.g. Brian's
Maximinus run) still produce a usable hypothesis. Every downstream consumer
(synthesis, router, query-construction, evaluator) reads the `hypothesis`
state key, never this module's functions directly.
"""

import json
import logging
import re

from app.llm.content import extract_text_content
from app.llm.provider import get_structured_model
from app.models.hypothesis import CoinHypothesis, HypothesisField
from app.models.requests import LLMConfig, QuickEvidence
from app.safety import with_safety

logger = logging.getLogger(__name__)

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


# --- Vision path (Phase 3/4) ---------------------------------------------

# Full coin-field allowlist a hypothesis may populate (contracts/
# vision-hypothesis.md §1; Go's `deepProposalCoinFieldAllowlist`,
# `src/api/services/deep_identification_proposal.go:36-56`). Wider than
# `_HYPOTHESIS_FIELDS` above (which only covers what `quick_evidence.
# coin_fields` can carry) because the vision call can additionally support
# `notes` and `coin_type`.
_ALLOWLIST_FIELDS = frozenset(CoinHypothesis.model_fields) - {"observations", "legible"}

# Some models (particularly Ollama's JSON mode) emit snake_case keys even
# when bound to a camelCase schema. Normalize before the allowlist check
# rather than dropping a perfectly legible field for a casing mismatch.
_KEY_ALIASES = {
    "date_range": "dateRange",
    "obverse_inscription": "obverseInscription",
    "reverse_inscription": "reverseInscription",
    "obverse_description": "obverseDescription",
    "reverse_description": "reverseDescription",
    "weight_grams": "weightGrams",
    "diameter_mm": "diameterMm",
}

# Confidence assigned to a field recovered only through prose extraction —
# deliberately below any confidence a conformant structured or deterministic
# source would assign, since this rung has no real per-field signal, only a
# best-effort regex/JSON scrape of unstructured text.
_PROSE_FALLBACK_CONFIDENCE = 0.4

VISION_HYPOTHESIS_PROMPT = with_safety("""You are a numismatic expert examining a coin image pair (obverse and
reverse). Produce a strict-JSON hypothesis of what the images alone
support, using ONLY these fields when you have real support for them:
ruler, denomination, material, mint, dateRange, era, obverseInscription,
reverseInscription, obverseDescription, reverseDescription, diameterMm,
weightGrams, notes, coin_type.

Rules:
- Each field you include MUST be an object: {"value": <string>, "confidence": <float 0-1>}.
- OMIT any field the images do not legibly support. Never guess a value at
  low confidence — an absent field is correct; a fabricated one is not.
- `era`, when included, MUST be exactly one of: ancient, medieval, modern.
- `material`, when included, MUST be exactly one of: gold, silver, bronze,
  copper, electrum, other.
- `observations` is a short (<=500 character) plain-prose summary for a
  human reader, never itself a proposed field value.
- `legible` is a boolean: true if the images support at least one field or
  a meaningful observation, false otherwise.
- No markdown, no emojis, no invented facts beyond what is visible.""")


def _canonicalize_hypothesis_field(key: str, value: str) -> str | None:
    """Shared era/material canonicalization used by every hypothesis
    source (deterministic adapter, structured vision parse, and prose
    fallback) — see `_canonical_era`/`_canonical_material` above. Returns
    `None` when the field must be dropped (garbage era/material value);
    returns the (possibly rewritten) value otherwise.
    """
    if key == "era":
        return _canonical_era(value)
    if key == "material":
        return _canonical_material(value)
    return value


def _normalize_vision_hypothesis(raw: CoinHypothesis) -> CoinHypothesis:
    """Post-validation normalization of a schema-conformant structured
    vision parse (spec FR-003/FR-005): re-applies the same era/material
    canonicalization the deterministic adapter uses, dropping any field
    that fails it, so a bound-but-garbage enum value never survives to
    become a proposed field on the Go side.
    """
    values: dict[str, HypothesisField] = {}
    for key, field in raw.fields().items():
        value = field.value.strip()
        if not value:
            continue
        canonical = _canonicalize_hypothesis_field(key, value)
        if canonical is None:
            continue
        values[key] = HypothesisField(value=canonical, confidence=field.confidence)

    observations = (raw.observations or "").strip()[:500]
    return CoinHypothesis(**values, observations=observations, legible=bool(values) or bool(observations))


def _coerce_prose_field_value(raw_value: object) -> tuple[str | None, float]:
    """Extract a `(value, confidence)` pair from one JSON value scraped out
    of unstructured prose. Accepts either the `{"value", "confidence"}`
    shape the schema asks for, or a bare string a looser model might emit
    instead. Returns `(None, 0.0)` when nothing usable is present.
    """
    if isinstance(raw_value, dict):
        value = raw_value.get("value")
        if not isinstance(value, str) or not value.strip():
            return None, 0.0
        confidence = raw_value.get("confidence")
        if isinstance(confidence, (int, float)) and not isinstance(confidence, bool):
            confidence = max(0.0, min(1.0, float(confidence)))
        else:
            confidence = _PROSE_FALLBACK_CONFIDENCE
        return value.strip()[:1000], confidence
    if isinstance(raw_value, str) and raw_value.strip():
        return raw_value.strip()[:1000], _PROSE_FALLBACK_CONFIDENCE
    return None, 0.0


def _parse_prose_hypothesis(text: str) -> CoinHypothesis | None:
    """Best-effort recovery of a `CoinHypothesis` from a non-conformant
    prose/JSON-ish response — the ladder's rung below structured output and
    above the deterministic quick-evidence adapter. Returns `None` on any
    failure so the caller falls through to the next rung; never raises.
    """
    if not text:
        return None

    match = re.search(r"\{.*\}", text, re.DOTALL)
    if not match:
        return None
    try:
        data = json.loads(match.group(0))
    except (json.JSONDecodeError, ValueError):
        return None
    if not isinstance(data, dict):
        return None

    values: dict[str, HypothesisField] = {}
    for raw_key, raw_value in data.items():
        if raw_key in ("observations", "legible"):
            continue
        key = _KEY_ALIASES.get(raw_key, raw_key)
        if key not in _ALLOWLIST_FIELDS:
            continue
        value, confidence = _coerce_prose_field_value(raw_value)
        if value is None:
            continue
        canonical = _canonicalize_hypothesis_field(key, value)
        if canonical is None:
            continue
        try:
            values[key] = HypothesisField(value=canonical, confidence=confidence)
        except Exception:
            continue

    observations_raw = data.get("observations")
    observations = observations_raw.strip()[:500] if isinstance(observations_raw, str) else ""

    if not values and not observations:
        return None

    try:
        return CoinHypothesis(**values, observations=observations, legible=bool(values) or bool(observations))
    except Exception:
        logger.exception("[deep_identification.hypothesis] prose-derived hypothesis failed model validation")
        return None


async def build_hypothesis_from_vision(
    llm_config: LLMConfig,
    image_contents: list[dict],
    quick_evidence: QuickEvidence | None,
) -> CoinHypothesis:
    """Structured vision-derived `CoinHypothesis`, produced by the SAME
    single per-job vision LLM call `prepare_evidence_node` already makes on
    every job — this function gives that existing call a typed schema
    instead of the old free-prose output; it never adds a call on the
    happy path.

    Degrade ladder (spec FR-006; documented deviation from tasks.md
    T020/T027 recorded in `.squad/decisions/inbox/cassius-vision-hypothesis.md`):

        structured call -> retry once (schema failure only)
            -> prose extraction -> deterministic quick-evidence hypothesis

    Never raises: every failure mode (LLM exception, timeout, empty
    content, schema-validation failure) degrades to a later rung, and the
    final rung (`build_hypothesis_from_quick_evidence`) is itself
    exception-free and always returns a valid `CoinHypothesis`.
    """
    fallback = build_hypothesis_from_quick_evidence(quick_evidence)
    if not image_contents:
        return fallback

    try:
        structured_model = get_structured_model(llm_config, CoinHypothesis)
    except Exception:
        logger.exception("[deep_identification.hypothesis] could not bind structured vision model")
        return fallback

    from langchain_core.messages import HumanMessage, SystemMessage

    from app.llm.retry import ainvoke_with_retry

    human_content: list[dict] = [{"type": "text", "text": VISION_HYPOTHESIS_PROMPT}, *image_contents]
    messages = [
        SystemMessage(content="You are an expert numismatist."),
        HumanMessage(content=human_content),
    ]

    last_raw_text = ""
    # "Retry once" (tasks.md T020) fires only when the LLM already
    # responded but failed schema validation — the happy path below makes
    # exactly one call, so cost/latency is unchanged for the overwhelming
    # majority of runs. This reuses the existing transient-failure retry
    # transport (app/llm/retry.py) rather than a bespoke mechanism.
    for _attempt in range(2):
        try:
            result = await ainvoke_with_retry(structured_model, messages)
        except Exception:
            logger.exception("[deep_identification.hypothesis] structured vision call failed")
            break

        parsed = result.get("parsed") if isinstance(result, dict) else None
        if isinstance(parsed, CoinHypothesis):
            normalized = _normalize_vision_hypothesis(parsed)
            if not normalized.is_empty():
                return normalized
            # Schema-conformant but nothing survived normalization (e.g.
            # era/material both invalid, nothing else legible) — treat as a
            # failed attempt and keep moving down the ladder.
            continue

        raw = result.get("raw") if isinstance(result, dict) else None
        raw_content = getattr(raw, "content", "") if raw is not None else ""
        text = extract_text_content(raw_content)
        if text:
            last_raw_text = text

    prose = _parse_prose_hypothesis(last_raw_text)
    if prose is not None and not prose.is_empty():
        return prose

    return fallback
