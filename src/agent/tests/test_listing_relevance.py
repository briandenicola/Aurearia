"""Title relevance check for dealer search (#774): drops listings that only mention the search words."""

import asyncio
import time
from decimal import Decimal

import pytest

from app.models.requests import LLMConfig
from app.teams import listing_relevance
from app.teams.coin_search import run_market_search
from app.teams.listing_relevance import TitleLabels, classify_titles, order_by_relevance
from app.teams.specialist_contracts import ProviderRunner
from app.tools.dealer_sites import SiteListing, parse_vcoins_results
from tests.test_dealer_sites import fixture_text

LLM = LLMConfig(provider="anthropic", api_key="test", model="test")
ABBASID = SiteListing(
    title="Abbasid Caliphate, al-Saffah, the CALIGULA OF THE ISLAMIC WORLD, AR Dirham",
    url="https://www.vcoins.com/en/stores/example/1/product/abbasid_dirham/1/Default.aspx",
    dealer="Example Numismatics",
    price=Decimal("120.00"),
    currency="USD",
)
GERMANICUS = "Germanicus Æ Dupondius. Struck under Caligula. Rome, AD 37-41."


def _label(title: str) -> str:
    lowered = title.lower()
    if "islamic" in lowered:
        return "unrelated"
    if "struck under" in lowered or lowered.startswith("germanicus"):
        return "related"
    return "match"


class FakeStructuredModel:
    """Labels titles like a well-behaved model and records what it was sent."""

    def __init__(self, behaviour: str = "ok"):
        self.behaviour = behaviour
        self.calls: list[list] = []

    async def ainvoke(self, messages):
        self.calls.append(messages)
        if self.behaviour == "raise":
            raise RuntimeError("provider 529")
        if self.behaviour == "slow":
            await asyncio.sleep(5)
        import json

        titles = json.loads(messages[1].content)["titles"]
        labels = [{"index": t["index"], "label": _label(t["title"])} for t in titles]
        if self.behaviour == "incomplete":
            labels = labels[:-1]
        if self.behaviour == "all_unrelated":
            labels = [{"index": t["index"], "label": "unrelated"} for t in titles]
        return {"raw": None, "parsed": TitleLabels.model_validate({"labels": labels}), "parsing_error": None}


@pytest.fixture
def fake_model(monkeypatch):
    model = FakeStructuredModel()
    monkeypatch.setattr(listing_relevance, "classify_titles", classify_titles)
    monkeypatch.setattr(listing_relevance, "get_structured_model", lambda _config, _schema: model)
    return model


def _vcoins_runner() -> ProviderRunner:
    listings = [*parse_vcoins_results(fixture_text("vcoins_search_caligula.html")), ABBASID]

    async def run(_query, _limit):
        return [listing.as_candidate() for listing in listings]

    return ProviderRunner(provider="vcoins", run=run, allowed_hosts=frozenset({"vcoins.com"}))


async def _search(limit: int = 10):
    return await run_market_search(
        {"query": "Caligula", "search_terms": "Caligula", "limit": limit},
        llm_config=LLM,
        provider_runners=[_vcoins_runner()],
    )


async def test_unrelated_titles_are_dropped_and_related_follow_matches(fake_model):
    result = await _search()
    titles = [item.title for item in result.items]

    assert ABBASID.title not in titles
    assert GERMANICUS in titles
    labels = [_label(title) for title in titles]
    assert labels == sorted(labels, key=["match", "related"].index), "related listings must follow every match"
    assert any("only mentioned the search words" in warning for warning in result.warnings)

    system, human = fake_model.calls[0]
    assert "untrusted" in system.content
    assert ABBASID.title in human.content


async def test_dropped_listings_are_replaced_up_to_the_limit(fake_model):
    unfiltered = await run_market_search(
        {"query": "Caligula", "search_terms": "Caligula", "limit": 10}, provider_runners=[_vcoins_runner()]
    )
    assert ABBASID.title in [item.title for item in unfiltered.items]

    result = await _search(limit=3)
    assert len(result.items) == 3
    assert all(_label(item.title) == "match" for item in result.items)
    assert result.truncation.truncated


@pytest.mark.parametrize("behaviour", ["raise", "incomplete", "all_unrelated"])
async def test_relevance_failure_never_removes_results(monkeypatch, behaviour):
    baseline = [item.title for item in (await _search()).items]  # conftest stub: check unavailable
    assert ABBASID.title in baseline

    monkeypatch.setattr(listing_relevance, "classify_titles", classify_titles)
    monkeypatch.setattr(listing_relevance, "get_structured_model", lambda *_: FakeStructuredModel(behaviour))
    result = await _search()

    assert [item.title for item in result.items] == baseline
    assert not any("only mentioned the search words" in warning for warning in result.warnings)


async def test_relevance_latency_is_bounded_by_the_timeout(monkeypatch):
    monkeypatch.setattr(listing_relevance, "get_structured_model", lambda *_: FakeStructuredModel("slow"))
    started = time.monotonic()
    labels = await classify_titles(LLM, "Caligula", ["Caligula AE As"], timeout=0.05)
    assert labels is None
    assert time.monotonic() - started < 1


async def test_relevance_check_is_logged_with_elapsed_time(monkeypatch, caplog):
    monkeypatch.setattr(listing_relevance, "get_structured_model", lambda *_: FakeStructuredModel())
    with caplog.at_level("INFO", logger="app.teams.listing_relevance"):
        assert await classify_titles(LLM, "Caligula", ["Caligula AE As", ABBASID.title]) == ["match", "unrelated"]
    assert "titles=2 outcome=ok elapsed_ms=" in caplog.text


async def test_no_llm_config_means_no_relevance_call(fake_model):
    result = await run_market_search(
        {"query": "Caligula", "search_terms": "Caligula", "limit": 10}, provider_runners=[_vcoins_runner()]
    )
    assert fake_model.calls == []
    assert ABBASID.title in [item.title for item in result.items]


def test_order_by_relevance_keeps_order_within_groups():
    assert order_by_relevance(["a", "b", "c", "d"], ["related", "match", "unrelated", "match"]) == (
        ["b", "d", "a"],
        1,
    )


async def test_incomplete_answer_is_rejected_not_padded(monkeypatch):
    monkeypatch.setattr(listing_relevance, "get_structured_model", lambda *_: FakeStructuredModel("incomplete"))
    assert await classify_titles(LLM, "Caligula", ["Caligula AE As", "Caligula denarius"]) is None


async def test_dict_parsed_answer_is_accepted(monkeypatch):
    class DictModel:
        async def ainvoke(self, _messages):
            return {"parsed": {"labels": [{"index": 0, "label": "related"}]}}

    monkeypatch.setattr(listing_relevance, "get_structured_model", lambda *_: DictModel())
    assert await classify_titles(LLM, "Caligula", [GERMANICUS]) == ["related"]
