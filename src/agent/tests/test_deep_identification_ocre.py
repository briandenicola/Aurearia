"""OCRE provider node tests (Feature 345 / T021).

Covers the Python-side behavioral contract of the automated OCRE node:
flag-off zero-call short-circuit, no-signal zero-call short-circuit, typed
partial-failure mapping (never raises), citation host validation, ambiguity
preservation, and the invariant that free-text legend content is only ever
passed as scoring-only `legend_tokens` — never as a query slug.
"""

import asyncio

import pytest

from app.models.requests import DeepIdentifyBounds, DeepProviderCatalogEntry, QuickEvidence
from app.teams.deep_identification.graph import _run_one_provider, provider_fanout_node
from app.teams.deep_identification.providers import ocre
from app.tools.provider_tools import ProviderToolError

VALID_CITATION = "https://numismatics.org/ocre/id/ric.1(2).aug.1"


class FakeOCRETools:
    """Records every ocre_search call and returns a scripted response."""

    def __init__(self, response=None, error=None):
        self._response = response or {}
        self._error = error
        self.calls: list[dict] = []

    async def ocre_search(self, **kwargs):
        self.calls.append(kwargs)
        if self._error is not None:
            raise self._error
        return self._response


def _entry(automatable=True, budget=3):
    return DeepProviderCatalogEntry(provider="ocre", automatable=automatable, call_budget=budget)


def _evidence(**coin_fields):
    label = coin_fields.pop("label_text", "")
    return QuickEvidence(label_text=label, coin_fields=coin_fields)


def _run(entry, tools, quick_evidence, notes=""):
    return asyncio.run(ocre.run(entry, tools, quick_evidence, notes))


def test_flag_off_short_circuits_with_zero_calls():
    tools = FakeOCRETools()
    result = _run(_entry(automatable=False), tools, _evidence(ruler="augustus"))
    assert result.status == "not_automated"
    assert result.automatable is False
    assert result.call_count == 0
    assert result.claims == []
    assert tools.calls == []  # provably no upstream call


def test_no_signal_short_circuits_with_zero_calls():
    tools = FakeOCRETools()
    # Only a legend (scoring-only) present — nothing type-bearing to bind.
    result = _run(_entry(), tools, _evidence(label_text="CAESAR AVGVSTVS"))
    assert result.status == "no_match"
    assert result.call_count == 0
    assert tools.calls == []


def test_none_quick_evidence_short_circuits_with_zero_calls():
    tools = FakeOCRETools()
    result = _run(_entry(), tools, None)
    assert result.status == "no_match"
    assert tools.calls == []


def test_ok_candidates_map_to_coin_type_claims():
    tools = FakeOCRETools(
        response={
            "status": "ok",
            "attribution": ocre.OCRE_ATTRIBUTION,
            "candidates": [
                {"type_uri": VALID_CITATION, "label": "RIC I Augustus 1", "confidence": 0.82,
                 "explanation": "matched authority, mint"},
                {"type_uri": "https://numismatics.org/ocre/id/ric.1(2).aug.2", "label": "RIC I Augustus 2",
                 "confidence": 0.55, "explanation": "matched authority"},
            ],
        }
    )
    result = _run(_entry(), tools, _evidence(ruler="augustus", mint="lugdunum"))
    assert result.status == "contributed"
    assert result.call_count == 1
    assert result.attribution == ocre.OCRE_ATTRIBUTION
    assert len(result.claims) == 2  # ambiguity preserved, not collapsed
    assert all(c.field == "coin_type" for c in result.claims)
    assert result.claims[0].citation == VALID_CITATION
    assert result.confidence == pytest.approx(0.82)


def test_off_host_citation_is_dropped_as_invalid_response():
    tools = FakeOCRETools(
        response={
            "status": "ok",
            "candidates": [
                {"type_uri": "https://evil.example.com/ocre/id/ric.1", "label": "Spoofed", "confidence": 0.9},
            ],
        }
    )
    result = _run(_entry(), tools, _evidence(ruler="augustus"))
    # A present-but-dropped citation is malformed upstream data, never a claim.
    assert result.status == "failed"
    assert result.error_kind == "invalid_response"
    assert result.claims == []


def test_empty_status_maps_to_no_match():
    tools = FakeOCRETools(response={"status": "empty", "candidates": []})
    result = _run(_entry(), tools, _evidence(ruler="augustus"))
    assert result.status == "no_match"
    assert result.call_count == 1


@pytest.mark.parametrize(
    "status,expected_status,expected_kind",
    [
        ("unavailable", "failed", "upstream"),
        ("cancelled", "failed", "upstream"),
        ("timeout", "timed_out", "timeout"),
        ("invalid_response", "failed", "invalid_response"),
        ("quota_limited", "no_match", "quota"),
    ],
)
def test_typed_partial_failures_never_raise(status, expected_status, expected_kind):
    tools = FakeOCRETools(response={"status": status, "candidates": []})
    result = _run(_entry(), tools, _evidence(ruler="augustus"))
    assert result.status == expected_status
    assert result.error_kind == expected_kind


def test_transport_error_maps_to_failed_never_raises():
    tools = FakeOCRETools(error=ProviderToolError("boom"))
    result = _run(_entry(), tools, _evidence(ruler="augustus"))
    assert result.status == "failed"
    assert result.error_kind == "upstream"
    assert result.call_count == 1


def test_missing_tools_client_is_unconfigured_not_crash():
    result = _run(_entry(), None, _evidence(ruler="augustus"))
    assert result.status == "failed"
    assert result.error_kind == "unconfigured"
    assert result.call_count == 0


def test_known_ocre_id_confirms_type():
    """Template-K path: a supplied OCRE id resolves/confirms a single type."""
    tools = FakeOCRETools(
        response={
            "status": "ok",
            "attribution": ocre.OCRE_ATTRIBUTION,
            "candidates": [
                {"type_uri": VALID_CITATION, "label": "RIC I Augustus 1", "confidence": 0.95,
                 "explanation": "confirmed by OCRE id"},
            ],
        }
    )
    result = _run(_entry(), tools, _evidence(ocre_id="ric.1(2).aug.1"))
    assert len(tools.calls) == 1
    assert tools.calls[0]["ocre_id"] == "ric.1(2).aug.1"
    assert result.status == "contributed"
    assert len(result.claims) == 1
    assert result.claims[0].citation == VALID_CITATION


def test_unknown_ocre_id_reported_unresolved_not_fabricated():
    """An OCRE id that resolves to nothing is no_match, never a made-up type."""
    tools = FakeOCRETools(response={"status": "empty", "candidates": []})
    result = _run(_entry(), tools, _evidence(ocre_id="ric.9.nonexistent.999"))
    assert len(tools.calls) == 1
    assert result.status == "no_match"
    assert result.claims == []


def test_legend_injection_never_enters_slug_slots():
    """Free-text / injection content is only ever passed as legend_tokens.

    The node must not smuggle legend/label text into ruler/denomination/mint/
    material/ocre_id — those slots carry only the discrete coin_fields values,
    which Go re-validates into slugs. This proves the emitted query's bound
    slots are invariant under adversarial label text.
    """
    tools = FakeOCRETools(response={"status": "empty", "candidates": []})
    injection = "augustus> } UNION { ?type ?p ?o } #"
    _run(
        _entry(),
        tools,
        _evidence(ruler="augustus", label_text=injection),
    )
    assert len(tools.calls) == 1
    call = tools.calls[0]
    assert call["ruler"] == "augustus"
    for slot in ("ruler", "denomination", "mint", "material", "ocre_id"):
        assert "UNION" not in call[slot]
        assert "}" not in call[slot]
    # The injection text is reduced to alnum scoring tokens only.
    for token in call["legend_tokens"]:
        assert token.isalnum()


# --- T035 (US3): OCRE failure never breaks partial synthesis ---


def _bounds(provider_timeout_s=5):
    return DeepIdentifyBounds(
        max_providers=5, max_concurrency=3, provider_timeout_s=provider_timeout_s,
        total_timeout_s=60, recursion_limit=10,
    )


@pytest.mark.asyncio
async def test_ocre_node_exception_is_absorbed_by_run_one_provider(monkeypatch):
    """A node that raises must surface as a typed failed row, never propagate."""
    async def boom(entry, tools, quick_evidence, notes):
        raise RuntimeError("kaboom")

    import app.teams.deep_identification.graph as graph_module

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "ocre", boom)

    catalog_by_name = {"ocre": _entry(automatable=True)}
    row = await _run_one_provider(
        "ocre", catalog_by_name, tools=object(), quick_evidence=None, notes="",
        bounds=_bounds(), semaphore=asyncio.Semaphore(1),
    )
    assert row.status == "failed"
    assert row.error_kind == "upstream"


@pytest.mark.asyncio
async def test_ocre_failure_alongside_contributor_still_reaches_synthesis(monkeypatch):
    """OCRE timing out / failing leaves every other provider's evidence intact."""
    from app.models.responses import ProviderEvidence

    async def ocre_timeout(entry, tools, quick_evidence, notes):
        return ProviderEvidence(provider="ocre", status="timed_out", automatable=True,
                                error_kind="timeout", call_count=1)

    async def numista_ok(entry, tools, quick_evidence, notes):
        return ProviderEvidence(provider="numista", status="contributed", automatable=True, call_count=1)

    import app.teams.deep_identification.graph as graph_module

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "ocre", ocre_timeout)
    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", numista_ok)

    catalog = [
        DeepProviderCatalogEntry(provider="ocre", automatable=True),
        DeepProviderCatalogEntry(provider="numista", automatable=True),
    ]
    state = {
        "catalog": catalog, "bounds": _bounds(), "quick_evidence": None, "notes": "",
        "selected": ["ocre", "numista"], "skipped": [],
    }

    result = await provider_fanout_node(state, tools=object())
    by_provider = {row.provider: row.status for row in result["evidence"]}
    assert by_provider["ocre"] == "timed_out"
    assert by_provider["numista"] == "contributed"
