"""Graph topology test (T067/T100) — verifies node/stage sequence without
ever invoking a real LLM or provider call.

T100: the previous version of this test asserted on `build_graph()`, a
compiled `StateGraph` that is test-only — production always runs the
hand-written async generator `run_deep_identification_stream`, which never
invokes the compiled graph at all. That meant this test was passing while
proving nothing about the code path users actually execute (the same class
of defect that caused the original outage: something fully tested that
production never ran). `build_graph()` has been deleted; the test below
drives `run_deep_identification_stream` itself (with the LLM/provider calls
faked out, same pattern as `test_deep_identification_sse.py`) and asserts
its emitted progress/stage frames occur in the exact
prepare_evidence -> router -> provider_fanout -> evaluator -> synthesizer
order (contract §3/§6), so it now verifies the real production topology.
"""

import asyncio
import json

import pytest

import app.teams.deep_identification.graph as graph_module
import app.teams.deep_identification.hypothesis as hypothesis_module
from app.models.requests import (
    DeepIdentifyBounds,
    DeepIdentifyImage,
    DeepIdentifyRequest,
    DeepProviderCatalogEntry,
    LLMConfig,
)
from app.models.responses import ProviderClaim, ProviderEvidence
from app.teams.deep_identification.providers.ocre import OCRE_ATTRIBUTION
from app.teams.deep_identification.synthesis import _build_attributions, synthesize


class _FakeChatModel:
    async def ainvoke(self, messages, **kwargs):
        return type("Resp", (), {"content": "A generic ancient bronze coin with worn legends."})()


class _FakeStructuredModel:
    """Same stand-in as `test_deep_identification_sse.py`'s: schema parsing
    always "fails" so the vision path degrades immediately to the
    deterministic quick-evidence hypothesis with no real network call."""

    async def ainvoke(self, messages, **kwargs):
        return {"raw": type("Resp", (), {"content": ""})(), "parsed": None, "parsing_error": None}


def _tiny_data_uri() -> str:
    return (  # noqa: E501 — a real minimal PNG data URI cannot be shortened
        "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
    )


def _topology_request() -> DeepIdentifyRequest:
    return DeepIdentifyRequest(
        job_id=1,
        llm=LLMConfig(provider="anthropic", model="claude-3-5-sonnet-20241022", api_key="test-key"),
        images=[
            DeepIdentifyImage(role="obverse", data_uri=_tiny_data_uri()),
            DeepIdentifyImage(role="reverse", data_uri=_tiny_data_uri()),
        ],
        notes="",
        provider_catalog=[DeepProviderCatalogEntry(provider="numista", automatable=True)],
        bounds=DeepIdentifyBounds(
            max_providers=1,
            max_concurrency=1,
            provider_timeout_s=5,
            total_timeout_s=30,
            recursion_limit=10,
        ),
        tools_base_url="http://test-api:8080",
        internal_token="test-token-12345",
    )


@pytest.mark.asyncio
async def test_production_stream_visits_the_five_nodes_in_order(monkeypatch):
    """Proves `run_deep_identification_stream` — the actual production
    driver — visits prepare_evidence, router, provider_fanout, evaluator,
    and synthesizer, strictly in that order, via its emitted frames."""
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: _FakeStructuredModel())
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: _FakeChatModel())

    async def fake_numista_run(entry, tools, quick_evidence, notes, hypothesis=None):
        return ProviderEvidence(provider="numista", status="no_match", automatable=True, call_count=1)

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", fake_numista_run)

    frames = []
    async for chunk in graph_module.run_deep_identification_stream(_topology_request()):
        assert chunk.startswith("data: ")
        frames.append(json.loads(chunk[len("data: "):].strip()))

    markers = [frame.get("stage") or frame["type"] for frame in frames]

    def index_of(marker: str) -> int:
        assert marker in markers, f"expected {marker!r} in {markers}"
        return markers.index(marker)

    prepare_evidence_order = (index_of("image_evidence_ready"), index_of("vision_completed"))
    router_order = (index_of("router_selected"),)
    provider_fanout_order = (
        index_of("provider_fanout_started"),
        index_of("provider_started"),
        index_of("provider_result"),
    )
    evaluator_order = (index_of("evaluation_started"), index_of("evaluation"))
    synthesizer_order = (index_of("synthesis_started"), index_of("synthesis"))

    node_boundaries = [
        max(prepare_evidence_order),
        max(router_order),
        max(provider_fanout_order),
        max(evaluator_order),
        max(synthesizer_order),
    ]
    assert node_boundaries == sorted(node_boundaries), (
        f"nodes did not run in prepare_evidence -> router -> provider_fanout -> "
        f"evaluator -> synthesizer order: {markers}"
    )
    assert frames[-1]["type"] == "synthesis"


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
