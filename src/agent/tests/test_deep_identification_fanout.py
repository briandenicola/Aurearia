"""Provider fan-out tests (T074) — contracts/agent-internal-contract.md §6.

Verifies bounded concurrency (`max_concurrency`), per-provider timeout ->
`timed_out`, and that one failing provider does not prevent the others
(or the overall pipeline) from reaching a result (partial success).
"""

import asyncio

import pytest

from app.models.requests import DeepIdentifyBounds, DeepProviderCatalogEntry
from app.models.responses import ProviderEvidence
from app.teams.deep_identification.graph import provider_fanout_node


def _bounds(max_concurrency=2, provider_timeout_s=1, max_providers=5) -> DeepIdentifyBounds:
    return DeepIdentifyBounds(
        max_providers=max_providers,
        max_concurrency=max_concurrency,
        provider_timeout_s=provider_timeout_s,
        total_timeout_s=60,
        recursion_limit=10,
    )


@pytest.mark.asyncio
async def test_fanout_includes_non_automatable_providers_even_if_not_selected(monkeypatch):
    catalog = [
        DeepProviderCatalogEntry(provider="ngc", automatable=False, link_out="https://www.ngccoin.com/verify/"),
        DeepProviderCatalogEntry(provider="ocre", automatable=False, reason="pending_license_validation"),
        DeepProviderCatalogEntry(provider="rpc", automatable=False, reason="no_public_api"),
    ]
    state = {
        "catalog": catalog,
        "bounds": _bounds(),
        "quick_evidence": None,
        "notes": "",
        "selected": [],  # router selected nothing automatable
        "skipped": [],
    }

    result = await provider_fanout_node(state, tools=None)

    providers_seen = {row.provider for row in result["evidence"]}
    assert providers_seen == {"ngc", "ocre", "rpc"}


@pytest.mark.asyncio
async def test_fanout_respects_max_concurrency(monkeypatch):
    concurrent = 0
    max_seen = 0
    lock = asyncio.Lock()

    async def fake_run(entry, tools, quick_evidence, notes):
        nonlocal concurrent, max_seen
        async with lock:
            concurrent += 1
            max_seen = max(max_seen, concurrent)
        await asyncio.sleep(0.05)
        async with lock:
            concurrent -= 1
        return ProviderEvidence(provider=entry.provider, status="no_match", automatable=True, call_count=1)

    import app.teams.deep_identification.graph as graph_module

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", fake_run)
    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "nomisma", fake_run)

    catalog = [
        DeepProviderCatalogEntry(provider=name, automatable=True) for name in ("numista", "nomisma")
    ]
    state = {
        "catalog": catalog,
        "bounds": _bounds(max_concurrency=1),
        "quick_evidence": None,
        "notes": "",
        "selected": ["numista", "nomisma"],
        "skipped": [],
    }

    await provider_fanout_node(state, tools=object())

    assert max_seen == 1


@pytest.mark.asyncio
async def test_fanout_provider_timeout_yields_timed_out(monkeypatch):
    async def slow_run(entry, tools, quick_evidence, notes):
        await asyncio.sleep(10)
        return ProviderEvidence(provider=entry.provider, status="no_match", automatable=True)

    import app.teams.deep_identification.graph as graph_module

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", slow_run)

    catalog = [DeepProviderCatalogEntry(provider="numista", automatable=True)]
    state = {
        "catalog": catalog,
        "bounds": _bounds(provider_timeout_s=1),
        "quick_evidence": None,
        "notes": "",
        "selected": ["numista"],
        "skipped": [],
    }

    result = await provider_fanout_node(state, tools=object())

    assert result["evidence"][0].status == "timed_out"
    assert result["evidence"][0].error_kind == "timeout"


@pytest.mark.asyncio
async def test_fanout_one_failing_provider_does_not_block_others(monkeypatch):
    async def failing_run(entry, tools, quick_evidence, notes):
        raise RuntimeError("boom")

    async def ok_run(entry, tools, quick_evidence, notes):
        return ProviderEvidence(provider=entry.provider, status="contributed", automatable=True, call_count=1)

    import app.teams.deep_identification.graph as graph_module

    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "numista", failing_run)
    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "nomisma", ok_run)

    catalog = [
        DeepProviderCatalogEntry(provider="numista", automatable=True),
        DeepProviderCatalogEntry(provider="nomisma", automatable=True),
    ]
    state = {
        "catalog": catalog,
        "bounds": _bounds(),
        "quick_evidence": None,
        "notes": "",
        "selected": ["numista", "nomisma"],
        "skipped": [],
    }

    result = await provider_fanout_node(state, tools=object())

    by_provider = {row.provider: row for row in result["evidence"]}
    assert by_provider["numista"].status == "failed"
    assert by_provider["nomisma"].status == "contributed"


@pytest.mark.asyncio
async def test_fanout_runs_ocre_through_automated_timeout_wrapped_path(monkeypatch):
    """Feature 345: a selected OCRE now runs via the automated fan-out path
    (async, timeout/semaphore-wrapped) — invoked with the 4-arg node
    signature — not the trivial single-arg dict."""
    seen_args = {}

    async def spy_ocre(entry, tools, quick_evidence, notes):
        seen_args["argc"] = 4
        seen_args["provider"] = entry.provider
        return ProviderEvidence(provider="ocre", status="contributed", automatable=True, call_count=1)

    import app.teams.deep_identification.graph as graph_module

    assert "ocre" in graph_module._AUTOMATED_PROVIDER_NODES
    assert "ocre" not in graph_module._TRIVIAL_PROVIDER_NODES
    monkeypatch.setitem(graph_module._AUTOMATED_PROVIDER_NODES, "ocre", spy_ocre)

    catalog = [DeepProviderCatalogEntry(provider="ocre", automatable=True)]
    state = {
        "catalog": catalog,
        "bounds": _bounds(provider_timeout_s=5),
        "quick_evidence": None,
        "notes": "",
        "selected": ["ocre"],
        "skipped": [],
    }

    result = await provider_fanout_node(state, tools=object())

    assert seen_args == {"argc": 4, "provider": "ocre"}
    assert result["evidence"][0].status == "contributed"


@pytest.mark.asyncio
async def test_fanout_rpc_trivial_path_unaffected_by_dict_split():
    """RPC remains the sole trivial (single-arg, synchronous) node after the
    OCRE move; it still emits its typed unavailable row with zero calls."""
    import app.teams.deep_identification.graph as graph_module

    assert set(graph_module._TRIVIAL_PROVIDER_NODES) == {"rpc"}

    catalog = [DeepProviderCatalogEntry(provider="rpc", automatable=False, reason="no_public_api")]
    state = {
        "catalog": catalog,
        "bounds": _bounds(),
        "quick_evidence": None,
        "notes": "",
        "selected": [],
        "skipped": [],
    }

    result = await provider_fanout_node(state, tools=None)

    row = result["evidence"][0]
    assert row.provider == "rpc"
    assert row.status == "unavailable"
    assert row.call_count == 0
