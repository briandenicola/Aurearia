"""Phase 9 — THE REGRESSION GATE (T068/T069).

The originating incident (spec.md `Context & Background`, User Story 2):
Brian photographed a slabbed NGC-graded Maximinus I (Thrax) AR denarius and
ran it through Deep Analysis. He got back a fallback sentence and an empty
draft, while Quick Lookup — and a single one-shot prompt to Anthropic —
correctly identified the coin from the images alone. Root cause: providers
were queried with a placeholder string, the vision result was computed and
discarded, and the narrative/proposal were hard-gated on provider
contribution.

T068 replays that exact fixture — two legible face images, **empty notes**,
**empty quick evidence** — end to end through the real production entry
point, `run_deep_identification_stream` (never `synthesize()`/a bare node
function directly; see `test_evaluator_node_receives_hypothesis_via_the_real_
graph_path` / `test_provider_node_receives_hypothesis_via_the_real_graph_path`
in `test_deep_identification_sse.py`, whose harness this file follows).

T069 widens the same no-placeholder invariant across a small corpus of
representative evidence shapes (empty, partial-hypothesis, notes-only),
also driven through the real entry point, so a future regression that
reintroduces *any* generic hardcoded query string — not only the literal
deleted `"unidentified ancient coin"` — is caught.
"""

import json
import re

import pytest

import app.teams.deep_identification.graph as graph_module
import app.teams.deep_identification.hypothesis as hypothesis_module
from app.models.hypothesis import CoinHypothesis, HypothesisField
from app.models.requests import (
    DeepIdentifyBounds,
    DeepIdentifyImage,
    DeepIdentifyRequest,
    DeepProviderCatalogEntry,
    LLMConfig,
    QuickEvidence,
)
from app.models.responses import DeepSynthesis
from app.teams.deep_identification.synthesis import FALLBACK_NARRATIVE_NO_EVIDENCE
from app.tools.provider_tools import ProviderToolsClient


def _tiny_data_uri() -> str:
    return (  # noqa: E501 — a real minimal PNG data URI cannot be shortened
        "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
    )


class _EchoNarrativeModel:
    """Fake chat model used for BOTH the evaluator's (unused here — zero
    disagreements) and the synthesizer's LLM call. It echoes the exact
    human-message content it is sent back as the "narrative" response,
    rather than returning a canned string. This is deliberate: the only
    way the echoed text can contain the ruler/denomination is if
    `graph.py` actually threaded `state["hypothesis"]` into
    `synthesize()` -> `_write_narrative()`'s prompt content. A canned
    response would make assertion (1) pass unconditionally and prove
    nothing about the wiring.
    """

    async def ainvoke(self, messages, **kwargs):
        human = messages[-1]
        return type("Resp", (), {"content": human.content})()


class _StructuredVisionOK:
    """Fake structured-output runnable standing in for the real vision LLM
    call (`get_structured_model`). Returns a schema-conformant
    `CoinHypothesis` on the first call — the happy path rung of the
    degrade ladder (`build_hypothesis_from_vision_traced`), exactly as
    `test_deep_identification_hypothesis.py::_StructuredOK` does.
    """

    def __init__(self, hypothesis: CoinHypothesis):
        self.hypothesis = hypothesis
        self.calls = 0

    async def ainvoke(self, messages, **kwargs):
        self.calls += 1
        return {"raw": type("Resp", (), {"content": "ignored"})(), "parsed": self.hypothesis, "parsing_error": None}


class _StructuredVisionDegrade:
    """Fake structured-output runnable that always fails schema parsing
    with empty raw content, exactly like `test_deep_identification_sse.py::
    _FakeStructuredModel` — forces the degrade ladder all the way down to
    `build_hypothesis_from_quick_evidence` (deterministic, LLM-free).
    """

    async def ainvoke(self, messages, **kwargs):
        return {"raw": type("Resp", (), {"content": ""})(), "parsed": None, "parsing_error": None}


def _catalog(*, ocre_automatable: bool = True) -> list[DeepProviderCatalogEntry]:
    return [
        DeepProviderCatalogEntry(provider="numista", automatable=True),
        DeepProviderCatalogEntry(provider="nomisma", automatable=True),
        DeepProviderCatalogEntry(provider="ocre", automatable=ocre_automatable),
        DeepProviderCatalogEntry(provider="ngc", automatable=False, link_out="https://www.ngccoin.com/verify/"),
    ]


def _request(*, notes: str = "", quick_evidence: QuickEvidence | None = None) -> DeepIdentifyRequest:
    return DeepIdentifyRequest(
        job_id=1,
        llm=LLMConfig(provider="anthropic", model="claude-3-5-sonnet-20241022", api_key="test-key"),
        images=[
            DeepIdentifyImage(role="obverse", data_uri=_tiny_data_uri()),
            DeepIdentifyImage(role="reverse", data_uri=_tiny_data_uri()),
        ],
        notes=notes,
        quick_evidence=quick_evidence,
        provider_catalog=_catalog(),
        bounds=DeepIdentifyBounds(
            max_providers=3,
            max_concurrency=3,
            provider_timeout_s=5,
            total_timeout_s=30,
            recursion_limit=10,
        ),
        tools_base_url="http://test-api:8080",
        internal_token="test-token-12345",
    )


def test_deep_request_accepts_omitted_notes():
    payload = _request().model_dump()
    payload.pop("notes")

    request = DeepIdentifyRequest.model_validate(payload)

    assert request.notes == ""


async def _collect_frames(request):
    frames = []
    async for chunk in graph_module.run_deep_identification_stream(request):
        assert chunk.startswith("data: ")
        payload = json.loads(chunk[len("data: "):].strip())
        frames.append(payload)
    return frames


def _no_match_response() -> dict:
    return {"status": "empty", "candidates": [], "attribution": "Source: stub"}


def _install_provider_spies(monkeypatch):
    """Patch the ONE real HTTP boundary each automated provider node calls
    (`ProviderToolsClient.{numista,nomisma,ocre}_search`), rather than the
    provider nodes themselves — so `_build_query`/`build_query_terms`
    genuinely run for real, and the exact query text a real production run
    would send upstream is captured and asserted on. Returns the captured
    per-provider call-argument lists.
    """
    calls: dict[str, list] = {"numista": [], "nomisma": [], "ocre": []}

    async def fake_numista_search(self, query, limit=5):
        calls["numista"].append(query)
        return _no_match_response()

    async def fake_nomisma_search(self, query, limit=5):
        calls["nomisma"].append(query)
        return _no_match_response()

    async def fake_ocre_search(self, *, ruler="", denomination="", mint="", material="", legend_tokens=None,
                                ocre_id="", limit=5):
        calls["ocre"].append((ruler, denomination, mint, material))
        return _no_match_response()

    monkeypatch.setattr(ProviderToolsClient, "numista_search", fake_numista_search)
    monkeypatch.setattr(ProviderToolsClient, "nomisma_search", fake_nomisma_search)
    monkeypatch.setattr(ProviderToolsClient, "ocre_search", fake_ocre_search)
    return calls


# The literal deleted placeholder, plus the shape of the generic phrases a
# future reintroduction would plausibly use (spec.md's "class of defect":
# searching for a generic constant instead of real evidence) — never
# derived from any specific fixture's real evidence.
_PLACEHOLDER_QUERY_PATTERN = re.compile(
    r"\bunidentified\b|\bunknown\s+(ancient\s+)?coin\b|\bgeneric\s+(ancient\s+)?coin\b|\bplaceholder\b|\bn/?a\b",
    re.IGNORECASE,
)


def _assert_not_placeholder(query: str) -> None:
    assert not _PLACEHOLDER_QUERY_PATTERN.search(query), (
        f"query {query!r} looks like a generic hardcoded placeholder rather than evidence-derived text"
    )


@pytest.mark.asyncio
async def test_prepare_evidence_keeps_face_roles_prompts_and_notes_isolated(monkeypatch):
    class CapturingFaceModel:
        def __init__(self):
            self.calls = []

        async def ainvoke(self, messages, **kwargs):
            content = messages[-1].content
            self.calls.append(content)
            prompt = content[0]["text"]
            role = "obverse" if "OBVERSE CUSTOM" in prompt else "reverse"
            return type("Resp", (), {"content": f"{role} narrative"})()

    notes = "Probus AD 276-282; Tripolis; CLEMENTIA TEMP; RIC V.2 927."
    face_model = CapturingFaceModel()
    hypothesis = CoinHypothesis(
        ruler=HypothesisField(value="Probus", confidence=0.9),
        mint=HypothesisField(value="Tripolis", confidence=0.8),
        reverseInscription=HypothesisField(value="CLEMENTIA TEMP", confidence=0.8),
        coin_type=HypothesisField(value="RIC V.2 927", confidence=0.75),
        legible=True,
    )
    monkeypatch.setattr(
        hypothesis_module,
        "get_structured_model",
        lambda config, schema: _StructuredVisionOK(hypothesis),
    )
    request = _request(notes=notes)
    request.images = [
        DeepIdentifyImage(role="obverse", data_uri="data:image/png;base64,YQ=="),
        DeepIdentifyImage(role="reverse", data_uri="data:image/png;base64,Yg=="),
    ]
    state = {
        "images": request.images,
        "notes": request.notes,
        "obverse_prompt": "OBVERSE CUSTOM",
        "reverse_prompt": "REVERSE CUSTOM",
        "quick_evidence": request.quick_evidence,
    }

    result = await graph_module.prepare_evidence_node(state, request.llm, face_model)

    assert len(face_model.calls) == 2
    calls_by_prompt = {call[0]["text"]: call for call in face_model.calls}
    obverse_prompt, obverse_call = next(
        (prompt, call) for prompt, call in calls_by_prompt.items() if "OBVERSE CUSTOM" in prompt
    )
    reverse_prompt, reverse_call = next(
        (prompt, call) for prompt, call in calls_by_prompt.items() if "REVERSE CUSTOM" in prompt
    )
    assert notes in obverse_prompt
    assert notes in reverse_prompt
    assert obverse_call[1]["image_url"]["url"].endswith("YQ==")
    assert reverse_call[1]["image_url"]["url"].endswith("Yg==")
    assert len(obverse_call) == 2
    assert len(reverse_call) == 2
    assert [face.role for face in result["face_analyses"]] == ["obverse", "reverse"]
    assert [face.narrative for face in result["face_analyses"]] == [
        "obverse narrative",
        "reverse narrative",
    ]
    assert result["hypothesis"].mint.value == "Tripolis"


# --- T068: the named Maximinus regression fixture -------------------------


@pytest.mark.asyncio
async def test_maximinus_run_produces_real_narrative_and_proposal(monkeypatch):
    """The named end-to-end fixture (spec.md US2 before/after table).

    Two legible face images, empty notes, EMPTY quick evidence (the cruel
    part — the vision hypothesis is the ONLY source of truth), all three
    automatable providers stubbed to `no_match`, NGC `not_automated`.
    """
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: _EchoNarrativeModel())

    hypothesis = CoinHypothesis(
        ruler=HypothesisField(value="Maximinus I (Thrax)", confidence=0.82),
        denomination=HypothesisField(value="Denarius", confidence=0.8),
        material=HypothesisField(value="silver", confidence=0.75),
        obverseInscription=HypothesisField(value="IMP MAXIMINVS PIVS AVG", confidence=0.7),
        era=HypothesisField(value="ancient", confidence=0.8),
        observations="Slabbed coin; worn but legible obverse portrait and legend.",
        legible=True,
    )
    monkeypatch.setattr(
        hypothesis_module, "get_structured_model", lambda config, schema: _StructuredVisionOK(hypothesis)
    )

    calls = _install_provider_spies(monkeypatch)

    request = _request(notes="", quick_evidence=None)
    frames = await _collect_frames(request)

    terminal = frames[-1]
    assert terminal["type"] == "synthesis"
    report = terminal["report"]
    DeepSynthesis.model_validate(report)  # must validate against the typed contract

    # (1) Narrative is NOT the fallback, and names both ruler and denomination.
    narrative = report["narrative"]
    assert narrative != FALLBACK_NARRATIVE_NO_EVIDENCE
    assert "Maximinus" in narrative, narrative
    assert "Denarius" in narrative, narrative

    # (2) proposed_fields has >= 4 entries.
    proposed_fields = report["proposed_fields"]
    assert len(proposed_fields) >= 4, proposed_fields

    # (3) Every one of those entries carries evidence_refs: [{"provider": "image"}].
    for field_name, entry in proposed_fields.items():
        refs = entry["evidence_refs"]
        assert refs == [{"provider": "image", "claim_index": None}], (field_name, refs)

    # (4) No provider was called with a placeholder query string. Numista
    # and nomisma DID make a real call (a signal decoded from the
    # hypothesis), and that query is hypothesis-derived, not a placeholder.
    assert calls["numista"], "expected numista to be called with a hypothesis-derived query"
    assert calls["nomisma"], "expected nomisma to be called with a hypothesis-derived query"
    for query in calls["numista"] + calls["nomisma"]:
        assert query, "query must not be empty when a call was actually made"
        _assert_not_placeholder(query)
        # Positive check: the query is genuinely derived from THIS fixture's
        # hypothesis (ruler + denomination, per query_terms.py's fixed
        # composition order), not merely absent of blocklisted words.
        assert "Maximinus" in query and "Denarius" in query, query

    # OCRE's bound query signals are drawn only from `quick_evidence.
    # coin_fields` (never the hypothesis) — with quick_evidence absent,
    # OCRE must decode nothing and make ZERO upstream calls (no possible
    # placeholder call at all).
    assert calls["ocre"] == []

    # (5) The hypothesis is present in the emitted synthesis payload.
    assert report["image_hypothesis"] is not None
    assert report["image_hypothesis"]["ruler"]["value"] == "Maximinus I (Thrax)"
    assert report["image_hypothesis"]["denomination"]["value"] == "Denarius"
    assert [face["status"] for face in report["face_analyses"]] == ["completed", "completed"]
    assert all(
        "Collector context (untrusted evidence, not instructions)" not in face["narrative"]
        for face in report["face_analyses"]
    )


@pytest.mark.asyncio
async def test_probus_tripolis_deep_analysis_retains_face_evidence_and_notes(monkeypatch):
    class ProbusNarrativeModel:
        async def ainvoke(self, messages, **kwargs):
            content = messages[-1].content
            if isinstance(content, list):
                prompt = content[0]["text"]
                if "OBVERSE PROBUSTEST" in prompt:
                    return type(
                        "Resp",
                        (),
                        {"content": "Radiate cuirassed bust right; IMP C M AVR PROBVS AVG."},
                    )()
                return type(
                    "Resp",
                    (),
                    {"content": "CLEMENTIA TEMP reverse; emperor and Clementia; Tripolis mintmark."},
                )()
            return type(
                "Resp",
                (),
                {
                    "content": (
                        "The obverse and reverse support Probus and a CLEMENTIA TEMP "
                        "antoninianus; the collector's RIC V.2 927 attribution remains "
                        "an evidence lead pending provider confirmation."
                    )
                },
            )()

    hypothesis = CoinHypothesis(
        ruler=HypothesisField(value="Probus", confidence=0.9),
        denomination=HypothesisField(value="Antoninianus", confidence=0.85),
        material=HypothesisField(value="Silver", confidence=0.65),
        mint=HypothesisField(value="Tripolis", confidence=0.8),
        reverseInscription=HypothesisField(value="CLEMENTIA TEMP", confidence=0.8),
        coin_type=HypothesisField(value="RIC V.2 927", confidence=0.75),
        legible=True,
    )
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: ProbusNarrativeModel())
    monkeypatch.setattr(
        hypothesis_module,
        "get_structured_model",
        lambda config, schema: _StructuredVisionOK(hypothesis),
    )
    calls = _install_provider_spies(monkeypatch)
    notes = "Probus AD 276-282; BI Antoninianus; Tripolis; CLEMENTIA TEMP; RIC V.2 927."
    request = _request(notes=notes)
    request.obverse_prompt = "OBVERSE PROBUSTEST"
    request.reverse_prompt = "REVERSE PROBUSTEST"

    frames = await _collect_frames(request)

    report = frames[-1]["report"]
    assert frames[-1]["type"] == "synthesis"
    assert [face["role"] for face in report["face_analyses"]] == ["obverse", "reverse"]
    assert "IMP C M AVR PROBVS AVG" in report["face_analyses"][0]["narrative"]
    assert "CLEMENTIA TEMP" in report["face_analyses"][1]["narrative"]
    assert "Tripolis" in report["face_analyses"][1]["narrative"]
    assert "Probus" in report["narrative"]
    assert "RIC V.2 927" in report["narrative"]
    for field in ("ruler", "denomination", "mint", "reverseInscription", "coin_type"):
        assert report["proposed_fields"][field]["evidence_refs"] == [
            {"provider": "image", "claim_index": None}
        ]
    assert any("Probus" in query and "Antoninianus" in query for query in calls["numista"])
    assert any("Probus" in query and "Antoninianus" in query for query in calls["nomisma"])


# --- T069: corpus-wide "never a placeholder query" assertion --------------

# Each case: (notes, quick_evidence, structured-vision fake, expected: does
# ANY signal exist anywhere for this fixture?). Spans every `build_query_
# terms` precedence tier (contract §2) plus the fully-empty case.
def _maximinus_hypothesis() -> CoinHypothesis:
    return CoinHypothesis(
        ruler=HypothesisField(value="Maximinus I (Thrax)", confidence=0.82),
        denomination=HypothesisField(value="Denarius", confidence=0.8),
        legible=True,
    )


def _ruler_only_hypothesis() -> CoinHypothesis:
    return CoinHypothesis(ruler=HypothesisField(value="Trajan Decius", confidence=0.6), legible=True)


_CORPUS = [
    pytest.param(
        "maximinus-full-hypothesis",
        "",
        None,
        _StructuredVisionOK(_maximinus_hypothesis()),
        id="maximinus_full_hypothesis",
    ),
    pytest.param(
        "ruler-only-hypothesis",
        "",
        None,
        _StructuredVisionOK(_ruler_only_hypothesis()),
        id="ruler_only_hypothesis",
    ),
    pytest.param(
        "quick-evidence-numista-query",
        "",
        QuickEvidence(numista_query="Nero as sestertius", confidence="high"),
        _StructuredVisionDegrade(),
        id="quick_evidence_numista_query_tier",
    ),
    pytest.param(
        "notes-only",
        "Found in an estate sale box, no other information available",
        None,
        _StructuredVisionDegrade(),
        id="notes_only_tier",
    ),
    pytest.param(
        "fully-empty",
        "",
        None,
        _StructuredVisionDegrade(),
        id="fully_empty_zero_signal",
    ),
]


@pytest.mark.asyncio
@pytest.mark.parametrize("case_name, notes, quick_evidence, structured_fake", _CORPUS)
async def test_no_provider_call_ever_carries_a_placeholder_query(
    monkeypatch, case_name, notes, quick_evidence, structured_fake
):
    """Corpus-wide (T069): across every fixture below — spanning each
    `build_query_terms` precedence tier and the fully-empty zero-signal
    case — zero provider calls carry a placeholder query string. This is
    driven through the real `run_deep_identification_stream` entry point
    exactly like T068, not through `build_query_terms`/provider `run()`
    directly, so a dropped call-site wire-up would also be caught.
    """
    monkeypatch.setattr(graph_module, "get_chat_model", lambda llm: _EchoNarrativeModel())
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: structured_fake)
    calls = _install_provider_spies(monkeypatch)

    request = _request(notes=notes, quick_evidence=quick_evidence)
    frames = await _collect_frames(request)

    terminal = frames[-1]
    assert terminal["type"] in ("synthesis", "error"), (case_name, terminal)

    all_queries = calls["numista"] + calls["nomisma"]

    if case_name == "fully-empty":
        # No signal anywhere (no hypothesis, no quick evidence, no notes):
        # `build_query_terms` must return "" and NEITHER provider may be
        # called at all (FR-011) — the only way a placeholder could ever
        # reach a provider is if this zero-call contract were broken.
        assert all_queries == [], all_queries
        assert calls["ocre"] == []
    else:
        # A real signal exists in this fixture; every provider call that
        # actually happened must carry real, evidence-derived text.
        assert all_queries, f"expected at least one provider call for fixture {case_name!r}"
        for query in all_queries:
            assert query, "query must not be empty when a call was actually made"
            _assert_not_placeholder(query)
