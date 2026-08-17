"""Hypothesis seam tests (Phase 8 — 351-vision-first-deep-identification).

Covers the deterministic quick-evidence -> `CoinHypothesis` adapter and the
`_build_proposed_fields` consumer this seam feeds (contracts/
vision-hypothesis.md §1, spec FR-019 through FR-025).
"""

from app.models.hypothesis import CoinHypothesis, HypothesisField
from app.models.requests import QuickEvidence, QuickEvidenceNGC
from app.teams.deep_identification.hypothesis import build_hypothesis_from_quick_evidence


def test_absent_quick_evidence_yields_empty_hypothesis():
    hypothesis = build_hypothesis_from_quick_evidence(None)

    assert hypothesis.is_empty()
    assert hypothesis.legible is False
    assert hypothesis.fields() == {}


def test_coin_fields_are_mapped_onto_the_shared_vocabulary():
    quick_evidence = QuickEvidence(
        confidence="high",
        coin_fields={
            "ruler": "Maximinus I",
            "denomination": "Denarius",
            "material": "Silver",
            "era": "ancient",
            "mint": "Rome",
        },
    )

    hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)

    assert hypothesis.legible is True
    assert hypothesis.ruler == HypothesisField(value="Maximinus I", confidence=0.75)
    assert hypothesis.denomination.value == "Denarius"
    assert hypothesis.material.value == "Silver"
    assert hypothesis.era.value == "ancient"
    assert hypothesis.mint.value == "Rome"


def test_era_and_material_values_that_do_not_canonicalize_are_dropped():
    """Go casts a proposed `era`/`material` value straight into
    `models.Era`/`models.Material` with no validation of its own
    (deep_identification_proposal.go::setCoinFieldFromProposalValue), so a
    value outside those fixed enum sets must never survive this adapter —
    it would silently write an invalid enum string into the coin row.
    """
    quick_evidence = QuickEvidence(
        confidence="medium",
        coin_fields={"era": "Roman Imperial", "material": "Billon"},
    )

    hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)

    assert hypothesis.era is None
    assert hypothesis.material is None
    # Nothing else supported either, and no NGC data -> genuinely empty.
    assert hypothesis.is_empty()


def test_material_canonicalizes_case_insensitively():
    quick_evidence = QuickEvidence(confidence="low", coin_fields={"material": "silver"})

    hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)

    assert hypothesis.material.value == "Silver"


def test_ngc_cert_and_grade_become_observations_even_without_coin_fields():
    quick_evidence = QuickEvidence(ngc=QuickEvidenceNGC(cert_number="1234567-001", grade="MS 65"))

    hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)

    assert "1234567-001" in hypothesis.observations
    assert "MS 65" in hypothesis.observations
    assert hypothesis.legible is True
    # Cert/grade are never proposed as coin fields themselves (no allowlist
    # entry for them) — they only ever reach the narrative via observations.
    assert hypothesis.fields() == {}


def test_blank_and_unknown_coin_fields_are_ignored():
    quick_evidence = QuickEvidence(coin_fields={"ruler": "   ", "grade": "MS65", "name": "Denarius"})

    hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)

    assert hypothesis.is_empty()


def test_coin_hypothesis_fields_helper_excludes_observations_and_legible():
    hypothesis = CoinHypothesis(
        ruler=HypothesisField(value="Trajan", confidence=0.6),
        observations="some notes",
        legible=True,
    )

    fields = hypothesis.fields()

    assert set(fields) == {"ruler"}
    assert fields["ruler"].value == "Trajan"
