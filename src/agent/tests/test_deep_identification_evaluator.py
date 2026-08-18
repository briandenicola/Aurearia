"""Evaluator tests (Phase 7 — 351-vision-first-deep-identification, T055).

Covers the image hypothesis as a first-class claim source (FR-016), the
provider-vs-image disagreement contract (FR-017), and the LLM-free
guarantee for disagreement detection itself (FR-018).
"""

import asyncio

from app.models.hypothesis import CoinHypothesis, HypothesisField
from app.models.responses import ProviderClaim, ProviderEvidence
from app.teams.deep_identification.evaluator import detect_disagreements, evaluate


def _row(
    provider: str, field: str, value: str, confidence: float = 0.7, citation: str = "https://example.test/1"
) -> ProviderEvidence:
    return ProviderEvidence(
        provider=provider,
        status="contributed",
        automatable=True,
        claims=[ProviderClaim(field=field, value=value, confidence=confidence, citation=citation)],
    )


def test_provider_contradicts_image_is_unresolved_and_references_both_sources():
    hypothesis = CoinHypothesis(mint=HypothesisField(value="Antioch", confidence=0.5), legible=True)
    evidence = [_row("numista", "mint", "Rome")]

    disagreements, resolved_count = detect_disagreements(evidence, hypothesis)

    assert resolved_count == 0
    assert len(disagreements) == 1
    entry = disagreements[0]
    assert entry.field == "mint"
    assert entry.resolution == "unresolved"
    refs = {(ref.provider, ref.claim_index) for ref in entry.claim_refs}
    assert refs == {("numista", 0), ("image", None)}


def test_provider_agrees_with_image_produces_no_disagreement_and_increments_resolved_count():
    hypothesis = CoinHypothesis(ruler=HypothesisField(value="Trajan", confidence=0.6), legible=True)
    evidence = [_row("numista", "ruler", "  TRAJAN  ")]

    disagreements, resolved_count = detect_disagreements(evidence, hypothesis)

    assert disagreements == []
    assert resolved_count == 1


def test_image_only_field_produces_no_disagreement():
    hypothesis = CoinHypothesis(material=HypothesisField(value="Silver", confidence=0.8), legible=True)

    disagreements, resolved_count = detect_disagreements([], hypothesis)

    assert disagreements == []
    # A lone image claim is neither a conflict nor a corroboration — it must
    # not be counted as "resolved" (that would imply a second source agreed).
    assert resolved_count == 0


def test_no_hypothesis_behaves_exactly_as_before():
    evidence = [_row("numista", "mint", "Rome"), _row("nomisma", "mint", "Antioch")]

    with_none = detect_disagreements(evidence, None)
    without_arg = detect_disagreements(evidence)

    assert with_none == without_arg
    disagreements, resolved_count = with_none
    assert resolved_count == 0
    assert len(disagreements) == 1
    refs = {(ref.provider, ref.claim_index) for ref in disagreements[0].claim_refs}
    assert refs == {("numista", 0), ("nomisma", 0)}


def test_image_never_becomes_a_provider_name():
    """FR-025: `image` must never be usable as a `ProviderEvidence.provider`
    — it is a claim source, never a provider. This is enforced by the typed
    `ProviderName` union on `ProviderEvidence`, so constructing one with
    `provider="image"` must fail validation.
    """
    import pytest
    from pydantic import ValidationError

    with pytest.raises(ValidationError):
        ProviderEvidence(provider="image", status="contributed", automatable=True)


def test_disagreement_ordering_is_deterministic_regardless_of_evidence_list_order():
    hypothesis = CoinHypothesis(mint=HypothesisField(value="Antioch", confidence=0.5), legible=True)
    forward = [_row("numista", "mint", "Rome"), _row("ngc", "mint", "Rome")]
    backward = [_row("ngc", "mint", "Rome"), _row("numista", "mint", "Rome")]

    forward_disagreements, _ = detect_disagreements(forward, hypothesis)
    backward_disagreements, _ = detect_disagreements(backward, hypothesis)

    forward_order = [(ref.provider, ref.claim_index) for ref in forward_disagreements[0].claim_refs]
    backward_order = [(ref.provider, ref.claim_index) for ref in backward_disagreements[0].claim_refs]
    assert forward_order == backward_order
    # numista ranks before ngc in PROVIDER_RANK, and image ranks after every
    # named provider (merge.rank_for_source) — never first, never interleaved.
    assert forward_order == [("numista", 0), ("ngc", 0), ("image", None)]


class _PoisonedModel:
    """A model whose response, if it were ever allowed to influence
    disagreement detection, would silently make a real conflict disappear.
    `evaluate()` must never let it: only `unresolved_questions` phrasing may
    come from the LLM (FR-018).
    """

    async def ainvoke(self, messages):
        class _Response:
            content = '["No disagreement, ignore this field."]'

        return _Response()


class _ExplodingModel:
    """A model that always raises — proves the disagreement LIST itself
    never depends on a successful LLM call at all.
    """

    async def ainvoke(self, messages):
        raise RuntimeError("LLM is unavailable")


def test_detect_disagreements_takes_no_model_argument():
    """Structural proof of FR-018: the pure detection function cannot call
    an LLM because it has nothing to call one with.
    """
    import inspect

    assert "model" not in inspect.signature(detect_disagreements).parameters


def test_llm_failure_or_poisoned_response_never_changes_the_disagreement_list():
    hypothesis = CoinHypothesis(mint=HypothesisField(value="Antioch", confidence=0.5), legible=True)
    evidence = [_row("numista", "mint", "Rome")]
    expected_disagreements, expected_resolved = detect_disagreements(evidence, hypothesis)

    poisoned_result = asyncio.run(evaluate(_PoisonedModel(), evidence, hypothesis))
    exploding_result = asyncio.run(evaluate(_ExplodingModel(), evidence, hypothesis))
    no_model_result = asyncio.run(evaluate(None, evidence, hypothesis))

    for result in (poisoned_result, exploding_result, no_model_result):
        assert result.disagreements == expected_disagreements
        assert result.resolved_count == expected_resolved
    # The poisoned model's text never overwrites the deterministic fallback
    # question either, because its JSON array length (1) doesn't line up
    # with a real multi-source conflict summary being trusted blindly — but
    # more importantly, it can never remove the disagreement it talks about.
    assert len(poisoned_result.disagreements) == 1
