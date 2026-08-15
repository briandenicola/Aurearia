"""Router tests (T073) — contracts/agent-internal-contract.md §6.

Verifies: selection is always a subset of the supplied catalog,
`provider_override` is honored without an LLM call, and an
`automatable: false` catalog entry is never present in `selected`/
`skipped` (i.e. never subject to router reasoning at all).
"""

import pytest

from app.models.requests import DeepProviderCatalogEntry
from app.teams.deep_identification.router import route


class FakeModel:
    """Fake chat model — never makes a network call."""

    def __init__(self, content: str | None = None, raise_exc: Exception | None = None):
        self.content = content
        self.raise_exc = raise_exc
        self.calls = 0

    async def ainvoke(self, messages, **kwargs):
        self.calls += 1
        if self.raise_exc:
            raise self.raise_exc
        return type("Resp", (), {"content": self.content})()


def _catalog(*entries: tuple[str, bool]) -> list[DeepProviderCatalogEntry]:
    return [DeepProviderCatalogEntry(provider=name, automatable=automatable) for name, automatable in entries]


@pytest.mark.asyncio
async def test_route_selects_only_from_catalog():
    catalog = _catalog(("numista", True), ("nomisma", True), ("ngc", False))
    model = FakeModel(content='{"selected": ["numista", "nomisma", "bogus"], "rationale": "test"}')

    decision = await route(model, catalog, provider_override=[], max_providers=5, notes="")

    assert set(decision.selected) <= {"numista", "nomisma"}
    assert "bogus" not in decision.selected
    assert model.calls == 1


@pytest.mark.asyncio
async def test_route_never_selects_non_automatable_provider():
    catalog = _catalog(("numista", True), ("ngc", False), ("ocre", False), ("rpc", False))
    model = FakeModel(content='{"selected": ["numista", "ngc", "ocre", "rpc"], "rationale": "test"}')

    decision = await route(model, catalog, provider_override=[], max_providers=5, notes="")

    assert decision.selected == ["numista"]
    assert all(skip["provider"] != "ngc" for skip in decision.skipped)
    assert all(skip["provider"] not in ("ngc", "ocre", "rpc") for skip in decision.skipped)


@pytest.mark.asyncio
async def test_provider_override_bypasses_llm():
    catalog = _catalog(("numista", True), ("nomisma", True))
    model = FakeModel(content="should never be read")

    decision = await route(model, catalog, provider_override=["nomisma"], max_providers=5, notes="")

    assert decision.selected == ["nomisma"]
    assert model.calls == 0


@pytest.mark.asyncio
async def test_provider_override_cannot_add_provider_outside_catalog():
    catalog = _catalog(("numista", True),)
    model = FakeModel(content="unused")

    decision = await route(model, catalog, provider_override=["nomisma"], max_providers=5, notes="")

    assert decision.selected == []
    assert model.calls == 0


@pytest.mark.asyncio
async def test_route_respects_max_providers_truncation():
    catalog = _catalog(("numista", True), ("nomisma", True))
    model = FakeModel(content='{"selected": ["numista", "nomisma"], "rationale": "both"}')

    decision = await route(model, catalog, provider_override=[], max_providers=1, notes="")

    assert len(decision.selected) == 1
    assert decision.selected[0] == "numista"  # PROVIDER_RANK tie-break


@pytest.mark.asyncio
async def test_route_falls_back_to_all_automatable_on_llm_failure():
    catalog = _catalog(("numista", True), ("nomisma", True))
    model = FakeModel(raise_exc=RuntimeError("llm down"))

    decision = await route(model, catalog, provider_override=[], max_providers=5, notes="")

    assert set(decision.selected) == {"numista", "nomisma"}


@pytest.mark.asyncio
async def test_route_falls_back_to_all_automatable_on_malformed_json():
    catalog = _catalog(("numista", True), ("nomisma", True))
    model = FakeModel(content="not json at all")

    decision = await route(model, catalog, provider_override=[], max_providers=5, notes="")

    assert set(decision.selected) == {"numista", "nomisma"}


@pytest.mark.asyncio
async def test_route_empty_catalog_selects_nothing_without_llm_call():
    model = FakeModel(content="unused")

    decision = await route(model, [], provider_override=[], max_providers=5, notes="")

    assert decision.selected == []
    assert decision.skipped == []
    assert model.calls == 0
