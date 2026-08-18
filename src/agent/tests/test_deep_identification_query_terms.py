"""Provider query-term builder tests (T034/T035/T041/T042/T043).

Covers the shared, deterministic `query_terms.build_query_terms` used by
every automatable provider node (`numista.py`, `nomisma.py`) and, in this
test module, its integration into those two nodes: the precedence table
(contracts/vision-hypothesis.md §2), the zero-placeholder guarantee
(FR-011), and the fixed hypothesis-derived composition order that
deliberately excludes reverse legend/type (RD-4/FR-010).
"""

import asyncio

from app.models.hypothesis import CoinHypothesis, HypothesisField
from app.models.requests import DeepProviderCatalogEntry, QuickEvidence
from app.teams.deep_identification.providers import nomisma, numista
from app.teams.deep_identification.query_terms import build_query_terms


def _field(value: str) -> HypothesisField:
    return HypothesisField(value=value, confidence=0.8)


def _entry(provider: str) -> DeepProviderCatalogEntry:
    return DeepProviderCatalogEntry(provider=provider, automatable=True, call_budget=3)


class FakeTools:
    """Records every query string sent; never actually called when the
    precedence tiers all yield nothing (the zero-call assertion below).
    """

    def __init__(self):
        self.numista_calls: list[str] = []
        self.nomisma_calls: list[str] = []

    async def numista_search(self, query: str, limit: int = 5):
        self.numista_calls.append(query)
        return {"status": "empty", "candidates": []}

    async def nomisma_search(self, query: str, limit: int = 5):
        self.nomisma_calls.append(query)
        return {"status": "empty", "candidates": []}


# --- T042: precedence table, all four tiers + quick-evidence-wins case ---


def test_tier1_numista_query_wins_over_everything():
    quick_evidence = QuickEvidence(numista_query="Maximinus denarius", label_text="ignored label")
    hypothesis = CoinHypothesis(ruler=_field("Ignored Ruler"))

    query = build_query_terms(quick_evidence, hypothesis, notes="ignored notes")

    assert query == "Maximinus denarius"


def test_tier2_label_text_wins_over_hypothesis_and_notes():
    quick_evidence = QuickEvidence(label_text="MAXIMINVS PIVS AVG denarius")
    hypothesis = CoinHypothesis(ruler=_field("Ignored Ruler"))

    query = build_query_terms(quick_evidence, hypothesis, notes="ignored notes")

    assert query == "MAXIMINVS PIVS AVG denarius"


def test_tier3_hypothesis_terms_used_when_quick_evidence_absent():
    hypothesis = CoinHypothesis(ruler=_field("Maximinus I"), denomination=_field("Denarius"))

    query = build_query_terms(None, hypothesis, notes="ignored notes")

    assert query == "Maximinus I Denarius"


def test_tier4_notes_used_when_nothing_else_yields_terms():
    query = build_query_terms(None, None, notes="a worn silver coin, portrait right")

    assert query == "a worn silver coin, portrait right"


def test_empty_quick_evidence_object_falls_through_to_hypothesis():
    """An empty (but present) QuickEvidence must not itself count as a
    non-empty tier 1/2 value — precedence falls through to tier 3.
    """
    quick_evidence = QuickEvidence()
    hypothesis = CoinHypothesis(ruler=_field("Maximinus I"))

    query = build_query_terms(quick_evidence, hypothesis, notes="")

    assert query == "Maximinus I"


def test_all_tiers_empty_yields_empty_string():
    assert build_query_terms(None, None, notes="") == ""
    assert build_query_terms(QuickEvidence(), CoinHypothesis(), notes="") == ""


# --- T035: fixed hypothesis composition order + reverse exclusion ---


def test_ruler_and_denomination_take_precedence_over_ruler_alone():
    hypothesis = CoinHypothesis(ruler=_field("Maximinus I"), denomination=_field("Denarius"), material=_field("Silver"))

    assert build_query_terms(None, hypothesis, notes="") == "Maximinus I Denarius"


def test_ruler_alone_takes_precedence_over_denomination_and_material():
    hypothesis = CoinHypothesis(ruler=_field("Maximinus I"), material=_field("Silver"))

    assert build_query_terms(None, hypothesis, notes="") == "Maximinus I"


def test_denomination_and_material_take_precedence_over_obverse_inscription():
    hypothesis = CoinHypothesis(
        denomination=_field("Denarius"), material=_field("Silver"), obverseInscription=_field("IMP MAXIMINVS")
    )

    assert build_query_terms(None, hypothesis, notes="") == "Denarius Silver"


def test_obverse_inscription_is_the_last_hypothesis_tier():
    hypothesis = CoinHypothesis(obverseInscription=_field("IMP MAXIMINVS PIVS AVG"))

    assert build_query_terms(None, hypothesis, notes="") == "IMP MAXIMINVS PIVS AVG"


def test_reverse_fields_never_appear_in_any_generated_query():
    """No generated query string may ever contain a reverse-legend or
    reverse-type term (T035) — reverse content is ranking-only (RD-4).
    """
    hypothesis = CoinHypothesis(
        ruler=_field("Maximinus I"),
        denomination=_field("Denarius"),
        material=_field("Silver"),
        obverseInscription=_field("IMP MAXIMINVS PIVS AVG"),
        reverseInscription=_field("PAX AVGVSTI SECRET REVERSE TOKEN"),
        coin_type=_field("SECRET COIN TYPE TOKEN"),
        reverseDescription=_field("SECRET REVERSE DESCRIPTION TOKEN"),
    )

    for partial_hypothesis in (
        hypothesis,
        CoinHypothesis(ruler=hypothesis.ruler, reverseInscription=hypothesis.reverseInscription,
                       coin_type=hypothesis.coin_type, reverseDescription=hypothesis.reverseDescription),
        CoinHypothesis(denomination=hypothesis.denomination, material=hypothesis.material,
                       reverseInscription=hypothesis.reverseInscription, coin_type=hypothesis.coin_type),
        CoinHypothesis(obverseInscription=hypothesis.obverseInscription,
                       reverseInscription=hypothesis.reverseInscription, coin_type=hypothesis.coin_type),
    ):
        query = build_query_terms(None, partial_hypothesis, notes="")
        assert "SECRET" not in query


# --- T041: zero-placeholder test across nodes, including empty-everything ---


def test_numista_node_never_issues_a_placeholder_query():
    tools = FakeTools()
    result = asyncio.run(numista.run(_entry("numista"), tools, None, notes=""))

    assert tools.numista_calls == []
    assert result.status == "no_match"
    assert result.error_kind == "insufficient_query_evidence"
    assert result.call_count == 0


def test_nomisma_node_never_issues_a_placeholder_query():
    tools = FakeTools()
    result = asyncio.run(nomisma.run(_entry("nomisma"), tools, None, notes=""))

    assert tools.nomisma_calls == []
    assert result.status == "no_match"
    assert result.error_kind == "insufficient_query_evidence"
    assert result.call_count == 0


def test_numista_query_never_equals_the_deleted_placeholder_string():
    """Even with SOME evidence, the emitted query is never the historical
    placeholder constant, and the placeholder is gone from the module.
    """
    assert not hasattr(numista, "_DEFAULT_QUERY")
    assert not hasattr(nomisma, "_DEFAULT_QUERY")

    tools = FakeTools()
    quick_evidence = QuickEvidence(label_text="MAXIMINVS PIVS AVG")
    asyncio.run(numista.run(_entry("numista"), tools, quick_evidence, notes=""))
    asyncio.run(nomisma.run(_entry("nomisma"), tools, quick_evidence, notes=""))

    for query in tools.numista_calls + tools.nomisma_calls:
        assert query != "unidentified ancient coin"


# --- T043: no-terms path performs zero ProviderToolsClient calls ---


def test_no_terms_path_calls_zero_upstream_and_consumes_zero_budget():
    tools = FakeTools()

    numista_result = asyncio.run(numista.run(_entry("numista"), tools, QuickEvidence(), notes=""))
    nomisma_result = asyncio.run(nomisma.run(_entry("nomisma"), tools, QuickEvidence(), notes=""))

    assert tools.numista_calls == []
    assert tools.nomisma_calls == []
    assert numista_result.call_count == 0
    assert nomisma_result.call_count == 0
