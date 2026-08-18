"""Synthesis node tests (Phase 8 — 351-vision-first-deep-identification).

Covers the hypothesis-aware fallback gate (FR-020/T059), the image-only and
corroborated proposed-field paths (FR-021/FR-022/T060-T061), the provider-only
invariant for coverage/attributions (FR-025/T063), and the Maximinus
regression that this whole feature exists to fix.
"""

import asyncio

from app.models.hypothesis import CoinHypothesis, HypothesisField
from app.models.responses import ProviderClaim, ProviderEvidence
from app.teams.deep_identification.synthesis import (
    CORROBORATION_CONFIDENCE_BONUS,
    FALLBACK_NARRATIVE_NO_EVIDENCE,
    _build_proposed_fields,
    synthesize,
)


class FakeModel:
    """A minimal chat-model stand-in whose narrative echoes what it saw,
    so tests can assert the narrative call actually happened without
    depending on any real LLM.
    """

    async def ainvoke(self, messages):
        class _Response:
            content = "The images support a Maximinus I denarius in silver; no provider corroborated it."

        return _Response()


def _no_match_row(provider: str, automatable: bool = True, status: str = "no_match") -> ProviderEvidence:
    return ProviderEvidence(provider=provider, status=status, automatable=automatable, call_count=1)


def test_provider_empty_with_hypothesis_present_does_not_fall_back():
    hypothesis = CoinHypothesis(ruler=HypothesisField(value="Trajan", confidence=0.6), legible=True)
    evidence = [_no_match_row("numista"), _no_match_row("nomisma")]

    synthesis = asyncio.run(
        synthesize(
            FakeModel(),
            evidence,
            disagreements=[],
            unresolved_questions=[],
            partial_success=False,
            hypothesis=hypothesis,
        )
    )

    assert synthesis.narrative != FALLBACK_NARRATIVE_NO_EVIDENCE


def test_provider_empty_and_hypothesis_empty_falls_back():
    evidence = [_no_match_row("numista"), _no_match_row("nomisma")]

    synthesis = asyncio.run(
        synthesize(
            FakeModel(), evidence, disagreements=[], unresolved_questions=[], partial_success=False, hypothesis=None
        )
    )

    assert synthesis.narrative == FALLBACK_NARRATIVE_NO_EVIDENCE


def test_provider_empty_and_hypothesis_present_but_empty_falls_back():
    """An empty `CoinHypothesis` instance (legible=False, no fields) must be
    treated exactly like no hypothesis at all — the gate checks
    `hypothesis.is_empty()`, not merely `hypothesis is not None`.
    """
    evidence = [_no_match_row("numista")]

    synthesis = asyncio.run(
        synthesize(
            FakeModel(),
            evidence,
            disagreements=[],
            unresolved_questions=[],
            partial_success=False,
            hypothesis=CoinHypothesis(),
        )
    )

    assert synthesis.narrative == FALLBACK_NARRATIVE_NO_EVIDENCE


def test_image_only_field_carries_image_ref_at_hypothesis_confidence():
    hypothesis = CoinHypothesis(ruler=HypothesisField(value="Maximinus I", confidence=0.62), legible=True)

    proposed = _build_proposed_fields(evidence=[], disagreement_fields=set(), hypothesis=hypothesis)

    assert "ruler" in proposed
    assert proposed["ruler"].value == "Maximinus I"
    assert proposed["ruler"].confidence == 0.62
    assert [ref.provider for ref in proposed["ruler"].evidence_refs] == ["image"]
    assert proposed["ruler"].evidence_refs[0].claim_index is None


def test_corroborated_field_gets_the_flat_bonus_once_no_stacking():
    """RD-2/FR-022: `min(1.0, max(image_conf, provider_conf) + 0.10)`,
    applied once regardless of how many providers agree with the image
    hypothesis on the same field — three corroborating providers must not
    yield a higher confidence than one.
    """
    hypothesis = CoinHypothesis(denomination=HypothesisField(value="Denarius", confidence=0.6), legible=True)
    single_provider = [
        ProviderEvidence(
            provider="numista",
            status="contributed",
            automatable=True,
            claims=[ProviderClaim(field="denomination", value="Denarius", confidence=0.5, citation="https://en.numista.com/c/1")],
        )
    ]
    three_providers = [
        ProviderEvidence(
            provider="numista",
            status="contributed",
            automatable=True,
            claims=[ProviderClaim(field="denomination", value="Denarius", confidence=0.5, citation="https://en.numista.com/c/1")],
        ),
        ProviderEvidence(
            provider="nomisma",
            status="contributed",
            automatable=True,
            claims=[ProviderClaim(field="denomination", value="Denarius", confidence=0.55, citation="https://nomisma.org/id/2")],
        ),
        ProviderEvidence(
            provider="ocre",
            status="contributed",
            automatable=True,
            claims=[ProviderClaim(field="denomination", value="Denarius", confidence=0.58, citation="https://numismatics.org/ocre/id/3")],
        ),
    ]

    single_result = _build_proposed_fields(single_provider, set(), hypothesis)
    stacked_result = _build_proposed_fields(three_providers, set(), hypothesis)

    expected = min(1.0, max(0.6, 0.5) + CORROBORATION_CONFIDENCE_BONUS)
    assert single_result["denomination"].confidence == expected
    assert stacked_result["denomination"].confidence == expected
    assert "image" in [ref.provider for ref in single_result["denomination"].evidence_refs]
    assert "image" in [ref.provider for ref in stacked_result["denomination"].evidence_refs]


def test_corroboration_confidence_never_exceeds_one():
    hypothesis = CoinHypothesis(material=HypothesisField(value="Silver", confidence=0.95), legible=True)
    evidence = [
        ProviderEvidence(
            provider="numista",
            status="contributed",
            automatable=True,
            claims=[ProviderClaim(field="material", value="Silver", confidence=0.98, citation="https://en.numista.com/c/1")],
        )
    ]

    proposed = _build_proposed_fields(evidence, set(), hypothesis)

    assert proposed["material"].confidence == 1.0


def test_provider_only_field_is_unchanged_when_hypothesis_is_absent():
    evidence = [
        ProviderEvidence(
            provider="numista",
            status="contributed",
            automatable=True,
            claims=[ProviderClaim(field="mint", value="Rome", confidence=0.7, citation="https://en.numista.com/c/1")],
        )
    ]

    proposed = _build_proposed_fields(evidence, set(), hypothesis=None)

    assert proposed["mint"].value == "Rome"
    assert proposed["mint"].confidence == 0.7
    assert [ref.provider for ref in proposed["mint"].evidence_refs] == ["numista"]


def test_image_never_enters_coverage_or_attributions():
    from app.teams.deep_identification.synthesis import _build_attributions, _build_coverage

    evidence = [
        ProviderEvidence(
            provider="numista",
            status="contributed",
            automatable=True,
            attribution="Source: Numista",
            claims=[ProviderClaim(field="mint", value="Rome", confidence=0.7, citation="https://en.numista.com/c/1")],
        ),
        _no_match_row("nomisma"),
        ProviderEvidence(provider="ngc", status="not_automated", automatable=False),
    ]

    coverage = _build_coverage(evidence)
    attributions = _build_attributions(evidence)

    assert "image" not in {entry.provider for entry in coverage}
    assert "image" not in {entry.provider for entry in attributions}


def test_maximinus_regression_synthesis_proposes_identification_despite_zero_provider_contribution():
    """The exact bug this feature exists to fix: a slabbed Maximinus I
    denarius where NGC is not_automated (by design, zero claims) and every
    other provider comes back no_match/failed — Quick Identify already
    correctly read ruler/denomination/material off the coin, and that MUST
    now reach the report instead of the no-evidence fallback.
    """
    hypothesis = CoinHypothesis(
        ruler=HypothesisField(value="Maximinus I", confidence=0.8),
        denomination=HypothesisField(value="Denarius", confidence=0.8),
        material=HypothesisField(value="Silver", confidence=0.8),
        observations="NGC cert 1234567-001; graded AU 55",
        legible=True,
    )
    evidence = [
        ProviderEvidence(provider="ngc", status="not_automated", automatable=False),
        _no_match_row("numista"),
        _no_match_row("nomisma", status="failed"),
        ProviderEvidence(provider="ocre", status="not_automated", automatable=False),
        ProviderEvidence(provider="rpc", status="not_automated", automatable=False),
    ]

    synthesis = asyncio.run(
        synthesize(
            FakeModel(),
            evidence,
            disagreements=[],
            unresolved_questions=[],
            partial_success=False,
            hypothesis=hypothesis,
        )
    )

    assert synthesis.narrative != FALLBACK_NARRATIVE_NO_EVIDENCE
    assert synthesis.narrative.strip() != ""
    for field in ("ruler", "denomination", "material"):
        assert field in synthesis.proposed_fields, f"expected {field} to be proposed"
        refs = {ref.provider for ref in synthesis.proposed_fields[field].evidence_refs}
        assert refs == {"image"}
    assert "image" not in {entry.provider for entry in synthesis.coverage}
    assert "image" not in {entry.provider for entry in synthesis.attributions}
