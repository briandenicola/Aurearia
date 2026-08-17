"""SSE envelope tests (T076) — contracts/agent-internal-contract.md §3.

Verifies the internal streaming envelope for `run_deep_identification_stream`:
terminal-frame invariants, sanitized output (no token-shaped strings leak),
and the expiry-with-partial-evidence fallback path (T069).
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
    QuickEvidence,
)
from app.models.responses import DeepSynthesis, ProviderEvidence


class FakeModel:
    async def ainvoke(self, messages, **kwargs):
        return type("Resp", (), {"content": "A generic ancient bronze coin with worn legends."})()


class _FakeStructuredModel:
    """Stands in for `get_structured_model`'s bound runnable: schema
    parsing always "fails" (`parsed=None`) with empty raw content, so the
    vision path degrades immediately to the deterministic quick-evidence
    hypothesis with no real LLM/network call. These SSE-envelope tests
    exercise the pipeline's frame contract, not the vision-hypothesis
    degrade ladder itself (covered by test_deep_identification_hypothesis.py).
    """

    async def ainvoke(self, messages, **kwargs):
        return {"raw": type("Resp", (), {"content": ""})(), "parsed": None, "parsing_error": None}


@pytest.fixture(autouse=True)
def _no_real_vision_calls(monkeypatch):
    """Prevent every test in this file from making a real provider network
    call through the vision-hypothesis structured-output path — the fake
    `LLMConfig(api_key="test-key")` these tests use is not a real
    credential, but `get_structured_model` would otherwise still attempt a
    real HTTP round trip to Anthropic.
    """
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: _FakeStructuredModel())


def _tiny_data_uri() -> str:
    return (  # noqa: E501 — a real minimal PNG data URI cannot be shortened
        "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
    )


def _request(total_timeout_s: int = 30) -> DeepIdentifyRequest:
    from app.models.requests import LLMConfig

    return DeepIdentifyRequest(
        job_id=1,
        llm=LLMConfig(provider="anthropic", model="claude-3-5-sonnet-20241022", api_key="test-key"),
        images=[
            DeepIdentifyImage(role="obverse", data_uri=_tiny_data_uri()),
            DeepIdentifyImage(role="reverse", data_uri=_tiny_data_uri()),
        ],
        notes="",
        provider_catalog=[
            DeepProviderCatalogEntry(provider="numista", automatable=True),
            DeepProviderCatalogEntry(provider="ngc", automatable=False, link_out="https://www.ngccoin.com/verify/"),
            DeepProviderCatalogEntry(provider="ocre", automatable=False, reason="pending_license_validation"),
            DeepProviderCatalogEntry(provider="rpc", automatable=False, reason="no_public_api"),
        ],
        bounds=DeepIdentifyBounds(
            max_providers=2,
            max_concurrency=2,
            provider_timeout_s=5,
            total_timeout_s=total_timeout_s,
            recursion_limit=10,
        ),
        tools_base_url="http://test-api:8080",
        internal_token="test-token-12345",
    )


async def _collect_frames(request):
    frames = []
    async for chunk in graph_module.run_deep_identification_stream(request):
        assert chunk.startswith("data: ")
        payload = json.loads(chunk[len("data: "):].strip())
        frames.append(payload)
    return frames


@pytest.mark.asyncio
async def test_happy_path_frame_sequence_and_terminal_invariant(monkeypatch):
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    async def fake_numista_run(entry, tools, quick_evidence, notes, hypothesis=None):
        return ProviderEvidence(provider="numista", status="no_match", automatable=True, call_count=1)

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", fake_numista_run)

    frames = await _collect_frames(_request())

    types = [f["type"] for f in frames]
    assert types[0] == "progress"
    assert "router_selected" in types
    assert "provider_started" in types
    assert "provider_result" in types
    assert "evaluation" in types
    assert "synthesis_started" in types
    assert types[-1] == "synthesis"

    # Terminal frame invariant: exactly one terminal frame (synthesis/error), at the end.
    terminal_frames = [f for f in frames if f["type"] in ("synthesis", "error")]
    assert len(terminal_frames) == 1
    assert terminal_frames[0] is frames[-1]

    report = frames[-1]["report"]
    DeepSynthesis.model_validate(report)  # must validate against the typed contract


@pytest.mark.asyncio
async def test_evaluator_node_receives_hypothesis_via_the_real_graph_path(monkeypatch):
    """Phase 7 wiring regression: `evaluator_node` (graph.py) must pass
    `state["hypothesis"]` into `evaluate()`. `evaluate()`/`detect_disagreements`
    have accepted an optional `hypothesis` parameter since Phase 7 landed
    (defaulting to `None`, which is why a missing call-site wire-up never
    raised) — until the graph actually threads it through, a
    provider-vs-image contradiction can never be detected in production.
    A test that called `evaluate()` directly would not catch a dropped
    argument at the call site; this drives the production entry point
    (`run_deep_identification_stream`) instead, so a hypothesis built from
    real `quick_evidence` genuinely reaches the evaluator through every
    intermediate node.
    """
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    async def contradicting_numista_run(entry, tools, quick_evidence, notes, hypothesis=None):
        from app.models.responses import ProviderClaim

        return ProviderEvidence(
            provider="numista",
            status="contributed",
            automatable=True,
            call_count=1,
            claims=[
                ProviderClaim(
                    field="denomination",
                    value="Denarius",
                    confidence=0.8,
                    citation="https://en.numista.com/catalogue/pieces1.html",
                )
            ],
        )

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", contradicting_numista_run)

    request = _request()
    # A real, degrades-to-quick-evidence hypothesis (the fixture's
    # `_FakeStructuredModel` always fails structured parsing) that
    # contradicts the provider claim above on the same field.
    request.quick_evidence = QuickEvidence(coin_fields={"denomination": "As"}, confidence="high")

    frames = await _collect_frames(request)

    evaluation_frames = [f for f in frames if f["type"] == "evaluation"]
    assert len(evaluation_frames) == 1
    assert evaluation_frames[0]["disagreement_count"] == 1, (
        "provider claim ('Denarius') contradicts the hypothesis ('As') on the same field "
        "(denomination) -- a non-zero disagreement_count proves the graph actually passed "
        "the hypothesis into evaluate(), not just that evaluate() supports one"
    )


@pytest.mark.asyncio
async def test_provider_node_receives_hypothesis_via_the_real_graph_path(monkeypatch):
    """Phase 5 wiring regression: the automated provider nodes' `run()`
    functions (`providers/numista.py`, `nomisma.py`, `ocre.py`) accepted an
    optional `hypothesis` parameter when Phase 5 (query terms + candidate
    ranking) landed, defaulting to `None` — which is why a missing call-site
    wire-up in `graph.py`'s `_run_one_provider`/`provider_fanout_node` never
    raised, but also meant no automated provider ever actually received a
    real hypothesis in production; every run silently fell back to
    quick-evidence-only query construction. A test that called `run()`
    directly would not catch a dropped argument at the fan-out call site;
    this drives the production entry point (`run_deep_identification_stream`)
    instead and asserts the exact hypothesis object the graph built is the
    one the provider node actually received.
    """
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    received: dict = {}

    async def spy_numista_run(entry, tools, quick_evidence, notes, hypothesis=None):
        received["hypothesis"] = hypothesis
        return ProviderEvidence(provider="numista", status="no_match", automatable=True, call_count=0)

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", spy_numista_run)

    request = _request()
    # Degrades to a real, non-empty `build_hypothesis_from_quick_evidence`
    # hypothesis (the fixture's `_FakeStructuredModel` always fails
    # structured parsing in this test file).
    request.quick_evidence = QuickEvidence(coin_fields={"ruler": "Trajan"}, confidence="high")

    await _collect_frames(request)

    assert received["hypothesis"] is not None
    assert received["hypothesis"].ruler is not None
    assert received["hypothesis"].ruler.value == "Trajan"


@pytest.mark.asyncio
async def test_provider_result_frame_carries_complete_claims(monkeypatch):
    """B1 regression: the production stream must emit the full, typed
    ProviderEvidence (with citation-backed claims) on `provider_result`
    (contract §3/§4). The Go runner resolves the terminal synthesis'
    evidence_refs against these streamed claims, so dropping them here
    silently empties every proposal's citation evidence in production.
    """
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    async def fake_numista_run(entry, tools, quick_evidence, notes, hypothesis=None):
        from app.models.responses import ProviderClaim

        return ProviderEvidence(
            provider="numista",
            status="contributed",
            automatable=True,
            confidence=0.7,
            call_count=1,
            attribution="Source: Numista",
            claims=[
                ProviderClaim(
                    field="denomination",
                    value="Denarius",
                    confidence=0.8,
                    citation="https://en.numista.com/catalogue/pieces12345.html",
                    excerpt="Silver denarius, Rome mint",
                )
            ],
        )

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", fake_numista_run)

    frames = await _collect_frames(_request())

    provider_results = [f for f in frames if f["type"] == "provider_result"]
    numista = next((f for f in provider_results if f.get("provider") == "numista"), None)
    assert numista is not None, "expected a numista provider_result frame"

    # Full ProviderEvidence shape must be present on the frame (not just
    # {type, provider, status}).
    assert numista["status"] == "contributed"
    assert numista["confidence"] == 0.7
    assert numista["automatable"] is True
    assert numista["call_count"] == 1

    claims = numista["claims"]
    assert len(claims) == 1
    claim = claims[0]
    # Exact field names/types the Go mirror (deepProposalClaim) unmarshals.
    assert claim["field"] == "denomination"
    assert claim["value"] == "Denarius"
    assert claim["confidence"] == 0.8
    assert claim["citation"] == "https://en.numista.com/catalogue/pieces12345.html"
    assert claim["excerpt"] == "Silver denarius, Rome mint"

    # The terminal synthesis references this claim by index, so the streamed
    # claims and the synthesis' evidence_refs are index-aligned end to end.
    synthesis = frames[-1]
    assert synthesis["type"] == "synthesis"
    ref = synthesis["report"]["proposed_fields"]["denomination"]["evidence_refs"][0]
    assert ref["provider"] == "numista"
    assert ref["claim_index"] == 0


@pytest.mark.asyncio
async def test_no_token_shaped_strings_leak_in_any_frame(monkeypatch):
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    async def leaky_numista_run(entry, tools, quick_evidence, notes, hypothesis=None):
        # Simulate a buggy upstream echoing an Authorization header inside a
        # claim value — must be redacted before it ever reaches an SSE frame.
        from app.models.responses import ProviderClaim

        return ProviderEvidence(
            provider="numista",
            status="contributed",
            automatable=True,
            confidence=0.5,
            call_count=1,
            claims=[
                ProviderClaim(
                    field="issuer",
                    value="Bearer abcdefghij.abcdefghij0123.abcdefghij0123",
                    confidence=0.5,
                    citation="https://en.numista.com/catalogue/1",
                )
            ],
        )

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", leaky_numista_run)

    request = _request()
    raw_chunks = []
    async for chunk in graph_module.run_deep_identification_stream(request):
        raw_chunks.append(chunk)
    full_text = "".join(raw_chunks)

    assert "Bearer abcdefghij" not in full_text
    assert "[REDACTED_INTERNAL_TOKEN]" in full_text


@pytest.mark.asyncio
async def test_timeout_with_partial_evidence_falls_back_to_partial_synthesis(monkeypatch):
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    async def slow_numista_run(entry, tools, quick_evidence, notes, hypothesis=None):
        await asyncio.sleep(30)
        return ProviderEvidence(provider="numista", status="no_match", automatable=True)

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", slow_numista_run)

    # Non-automatable providers (ngc/ocre/rpc) complete instantly, giving the
    # timeout branch *some* evidence to synthesize from even though numista hangs.
    request = _request(total_timeout_s=1)
    request.bounds.provider_timeout_s = 60  # provider-level timeout must not fire first

    frames = await _collect_frames(request)

    assert frames[-1]["type"] == "synthesis"
    assert frames[-1]["report"]["partial_success"] is True


@pytest.mark.asyncio
async def test_timeout_with_zero_evidence_emits_typed_error(monkeypatch):
    # The router (Phase 6, T044) is now a pure, LLM-free function — it can no
    # longer be made to hang. To exercise the zero-evidence timeout path,
    # hang the sole automatable provider's fan-out node instead, so the
    # total timeout fires before any evidence has been collected.
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    async def hanging_numista_run(entry, tools, quick_evidence, notes, hypothesis=None):
        await asyncio.sleep(30)
        return ProviderEvidence(provider="numista", status="no_match", automatable=True)

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", hanging_numista_run)

    request = _request(total_timeout_s=1)
    request.provider_catalog = [
        entry for entry in request.provider_catalog if entry.provider == "numista"
    ]
    frames = await _collect_frames(request)

    assert frames[-1]["type"] == "error"
    assert frames[-1]["code"] == "timeout"


@pytest.mark.asyncio
async def test_unexpected_failure_maps_to_typed_error_code(monkeypatch):
    """F4: an unexpected pipeline exception ends the stream with a typed
    contract §3 `error` frame carrying `code` (never `error_kind`). The
    narrow classifier maps model-output parse failures to
    `invalid_model_output`, provider/model connectivity failures to
    `llm_unavailable`, and anything else to `internal`.
    """
    from langchain_core.exceptions import OutputParserException

    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: FakeModel())

    cases = [
        (OutputParserException("bad model output"), "invalid_model_output"),
        (ConnectionError("provider unreachable"), "llm_unavailable"),
        (RuntimeError("something else broke"), "internal"),
    ]
    for exc, expected_code in cases:
        async def failing_synthesize(*args, _exc=exc, **kwargs):
            raise _exc

        monkeypatch.setattr(graph_module, "synthesize", failing_synthesize)
        frames = await _collect_frames(_request())
        assert frames[-1]["type"] == "error"
        assert frames[-1]["code"] == expected_code
        assert "error_kind" not in frames[-1]


def test_classify_pipeline_error_is_narrow():
    """The classifier only assigns specific codes to well-understood
    failures and otherwise stays `internal` (no broad guessing)."""
    from langchain_core.exceptions import OutputParserException

    assert graph_module._classify_pipeline_error(OutputParserException("x")) == "invalid_model_output"
    assert graph_module._classify_pipeline_error(ConnectionError("x")) == "llm_unavailable"
    assert graph_module._classify_pipeline_error(ValueError("x")) == "internal"
    assert graph_module._classify_pipeline_error(RuntimeError("x")) == "internal"
