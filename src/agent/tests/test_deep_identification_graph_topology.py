"""Graph topology test (T067) — verifies node/edge shape without ever
invoking a real LLM or provider call.
"""

import asyncio

from app.models.responses import ProviderClaim, ProviderEvidence
from app.teams.deep_identification.graph import build_graph
from app.teams.deep_identification.providers.ocre import OCRE_ATTRIBUTION
from app.teams.deep_identification.synthesis import _build_attributions, synthesize


def test_build_graph_has_expected_node_topology():
    graph = build_graph(model=None, tools=None)
    nodes = set(graph.get_graph().nodes.keys())

    assert nodes == {
        "__start__",
        "prepare_evidence",
        "router",
        "provider_fanout",
        "evaluator",
        "synthesizer",
        "__end__",
    }

    edges = {(edge.source, edge.target) for edge in graph.get_graph().edges}
    assert ("__start__", "prepare_evidence") in edges
    assert ("prepare_evidence", "router") in edges
    assert ("router", "provider_fanout") in edges
    assert ("provider_fanout", "evaluator") in edges
    assert ("evaluator", "synthesizer") in edges
    assert ("synthesizer", "__end__") in edges


def test_build_graph_binds_recursion_limit_into_invocation_config():
    """F3 (contract §6): `bounds.recursion_limit` is bound into the compiled
    graph's invocation config so graph-based invocation is capped at the
    iteration bound."""
    graph = build_graph(model=None, tools=None, recursion_limit=12)
    assert graph.config.get("recursion_limit") == 12
    # Topology is preserved through the config binding.
    assert "synthesizer" in set(graph.get_graph().nodes.keys())


def test_build_graph_without_recursion_limit_is_unbound():
    graph = build_graph(model=None, tools=None)
    assert (getattr(graph, "config", None) or {}).get("recursion_limit") is None


# --- Feature 345 (T026): deterministic provider attribution synthesis ---

_OCRE_CITATION = "https://numismatics.org/ocre/id/ric.1(2).aug.1"
_NOMISMA_CITATION = "https://nomisma.org/id/augustus"


def _ocre_contributed():
    return ProviderEvidence(
        provider="ocre", status="contributed", automatable=True, confidence=0.8, call_count=1,
        attribution=OCRE_ATTRIBUTION,
        claims=[ProviderClaim(field="coin_type", value="RIC I Augustus 1", confidence=0.8, citation=_OCRE_CITATION)],
    )


def _nomisma_contributed():
    return ProviderEvidence(
        provider="nomisma", status="contributed", automatable=True, confidence=0.7, call_count=1,
        attribution="Data: Nomisma.org (CC BY)",
        claims=[ProviderClaim(field="mint_authority", value="Augustus", confidence=0.7, citation=_NOMISMA_CITATION)],
    )


def test_attribution_present_only_when_provider_contributed_a_claim():
    attributions = _build_attributions([_ocre_contributed()])
    assert len(attributions) == 1
    assert attributions[0].provider == "ocre"
    assert attributions[0].text == OCRE_ATTRIBUTION
    assert attributions[0].identifier == _OCRE_CITATION


def test_ocre_attribution_text_is_exact_odbl_ans_string():
    attributions = _build_attributions([_ocre_contributed()])
    assert attributions[0].text == (
        "Coin type data: Online Coins of the Roman Empire (OCRE), "
        "American Numismatic Society \u2014 ODbL 1.0."
    )


def test_multiple_providers_produce_distinct_unmerged_attributions():
    attributions = _build_attributions([_ocre_contributed(), _nomisma_contributed()])
    providers = [a.provider for a in attributions]
    texts = {a.text for a in attributions}
    assert providers == ["ocre", "nomisma"]
    assert len(texts) == 2  # distinct, never merged


def test_no_attribution_for_no_match_failed_or_not_automated():
    rows = [
        ProviderEvidence(provider="ocre", status="no_match", automatable=True, attribution=OCRE_ATTRIBUTION),
        ProviderEvidence(provider="numista", status="failed", automatable=True, error_kind="upstream"),
        ProviderEvidence(provider="rpc", status="not_automated", automatable=False,
                         attribution="RPC — not yet automated"),
    ]
    assert _build_attributions(rows) == []


def test_synthesize_wires_attributions_into_report():
    report = asyncio.run(
        synthesize(
            model=None,
            evidence=[_ocre_contributed()],
            disagreements=[],
            unresolved_questions=[],
            partial_success=False,
        )
    )
    assert len(report.attributions) == 1
    assert report.attributions[0].provider == "ocre"
    assert report.attributions[0].identifier == _OCRE_CITATION
