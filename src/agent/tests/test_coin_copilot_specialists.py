"""Contract-first tests for Coin Copilot market and auction specialist runners."""

import asyncio
from datetime import UTC, datetime

import httpx
import pytest

from app.models.requests import LLMConfig
from app.teams.auction_search import run_auction_search
from app.teams.coin_search import _collect_market_candidates, _fetch_dealer_pages, run_market_search
from app.teams.specialist_contracts import (
    ProviderMalformedError,
    ProviderRunner,
    ProviderUnavailableError,
)
from app.tools.search import validate_dealer_url

OBSERVED_AT = datetime(2026, 9, 18, 12, 0, tzinfo=UTC)


def _dealer_candidate(**overrides):
    candidate = {
        "sourceUrl": "https://www.cngcoins.com/Coin.aspx?CoinID=400001",
        "name": "Domitian denarius with Minerva reverse",
        "sourceName": "Classical Numismatic Group",
        "estPrice": "USD 275",
        "availability": "Available",
        "ruler": "Domitian",
        "denomination": "Denarius",
        "era": "Roman Imperial",
        "material": "Silver",
    }
    candidate.update(overrides)
    return candidate


def _auction_candidate(**overrides):
    candidate = {
        "url": "https://www.numisbids.com/sale/10489/lot/1",
        "title": "Domitian AR denarius, Minerva reverse",
        "description": "Silver denarius of Domitian.",
        "auctionHouse": "Example Numismatic Auction",
        "saleName": "Ancient Coins 42",
        "lotNumber": 1,
        "saleDate": "1 Oct 2026",
        "estimate": 250,
        "currentBid": 175,
        "currency": "USD",
        "lotStatus": "upcoming",
    }
    candidate.update(overrides)
    return candidate


def _provider(provider, result=None, error=None):
    async def run(_query, _limit):
        if error is not None:
            raise error
        return result

    return ProviderRunner(provider=provider, run=run)


@pytest.mark.asyncio
async def test_legacy_dealer_fetch_keeps_preexisting_broad_fetch_path(monkeypatch):
    calls = []

    class LegacyFetch:
        async def ainvoke(self, args):
            calls.append(("legacy", args["url"]))
            return "legacy listing"

    async def specialist_fetch(_url):
        raise AssertionError("legacy graph must not use the specialist fetch boundary")

    monkeypatch.setattr("app.teams.coin_search.fetch_dealer_page", LegacyFetch())
    monkeypatch.setattr("app.teams.coin_search.fetch_registered_dealer_page", specialist_fetch)

    result = await _fetch_dealer_pages(
        "https://coinshows.com/listing",
        {"coinshows.com"},
        specialist_boundary=False,
    )

    assert calls == [("legacy", "https://coinshows.com/listing")]
    assert "legacy listing" in result


@pytest.mark.asyncio
async def test_specialist_dealer_fetch_uses_registered_boundary(monkeypatch):
    calls = []

    class LegacyFetch:
        async def ainvoke(self, _args):
            raise AssertionError("specialist runner must not use the legacy fetch tool")

    async def specialist_fetch(url):
        calls.append(("specialist", url))
        validate_dealer_url(url)
        return "specialist listing"

    async def search_pages(*_args):
        return "https://coinshows.com/listing"

    monkeypatch.setattr("app.teams.coin_search.fetch_dealer_page", LegacyFetch())
    monkeypatch.setattr("app.teams.coin_search.fetch_registered_dealer_page", specialist_fetch)
    monkeypatch.setattr("app.teams.coin_search._search_dealer_pages", search_pages)

    with pytest.raises(ValueError, match="registered source"):
        await _collect_market_candidates(
            LLMConfig(
                provider="anthropic",
                api_key="test",
                model="test",
            ),
            "Domitian denarius",
            5,
        )

    assert calls == [("specialist", "https://coinshows.com/listing")]


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("error", "status", "warning_code"),
    [
        (TimeoutError(), "timeout", "provider_timeout"),
        (httpx.TransportError("socket failed"), "failure", "provider_failure"),
        (ProviderUnavailableError(), "unavailable", "provider_unavailable"),
        (ProviderMalformedError(), "malformed", "provider_malformed"),
    ],
)
async def test_market_search_maps_degraded_provider_outcomes(error, status, warning_code):
    result = await run_market_search(
        {"query": "Domitian denarius", "limit": 5},
        provider_runners=[_provider("cng_dealer_search", error=error)],
        observed_at=OBSERVED_AT,
    )

    assert result.outcome == "unavailable"
    assert result.items == []
    assert result.provider_attempts[0].status == status
    assert result.provider_attempts[0].warning_code == warning_code
    if str(error):
        assert str(error) not in " ".join(result.warnings)


@pytest.mark.asyncio
async def test_market_search_returns_source_backed_success():
    result = await run_market_search(
        {"query": "Domitian denarius", "limit": 5},
        provider_runners=[_provider("cng_dealer_search", [_dealer_candidate()])],
        observed_at=OBSERVED_AT,
    )

    assert result.outcome == "complete"
    assert len(result.items) == 1
    item = result.items[0]
    assert item.kind == "dealer_listing"
    assert item.source_url == _dealer_candidate()["sourceUrl"]
    assert item.listed_price == 275
    assert item.currency == "USD"
    assert item.availability == "available"
    assert {entry.field for entry in item.provenance} >= {
        "title",
        "dealer_name",
        "listed_price",
        "currency",
        "availability",
    }


@pytest.mark.asyncio
async def test_market_search_returns_no_match_for_successful_zero_results():
    result = await run_market_search(
        {"query": "Domitian denarius"},
        provider_runners=[_provider("cng_dealer_search", [])],
        observed_at=OBSERVED_AT,
    )

    assert result.outcome == "no_match"
    assert result.items == []
    assert result.provider_attempts[0].status == "no_match"


@pytest.mark.asyncio
async def test_market_search_preserves_valid_results_when_another_provider_times_out():
    result = await run_market_search(
        {"query": "Domitian denarius"},
        provider_runners=[
            _provider("cng_dealer_search", [_dealer_candidate()]),
            _provider("market_search_secondary", error=TimeoutError()),
        ],
        observed_at=OBSERVED_AT,
    )

    assert result.outcome == "partial"
    assert len(result.items) == 1
    assert [attempt.status for attempt in result.provider_attempts] == ["success", "timeout"]


@pytest.mark.asyncio
async def test_market_search_omits_malformed_candidates_without_raw_data():
    result = await run_market_search(
        {"query": "Domitian denarius"},
        provider_runners=[_provider("cng_dealer_search", [{"name": "Missing URL", "password": "secret"}])],
        observed_at=OBSERVED_AT,
    )

    assert result.outcome == "unavailable"
    assert result.items == []
    assert result.provider_attempts[0].status == "malformed"
    encoded = result.model_dump_json()
    assert "secret" not in encoded
    assert "Missing URL" not in encoded


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("error", "status", "warning_code"),
    [
        (asyncio.TimeoutError(), "timeout", "provider_timeout"),
        (httpx.TransportError("connection reset"), "failure", "provider_failure"),
        (ProviderUnavailableError(), "unavailable", "provider_unavailable"),
        (ProviderMalformedError(), "malformed", "provider_malformed"),
    ],
)
async def test_auction_search_maps_degraded_provider_outcomes(error, status, warning_code):
    result = await run_auction_search(
        {"query": "Domitian denarius", "limit": 5},
        provider_runners=[_provider("numisbids", error=error)],
        observed_at=OBSERVED_AT,
    )

    assert result.outcome == "unavailable"
    assert result.items == []
    assert result.provider_attempts[0].status == status
    assert result.provider_attempts[0].warning_code == warning_code
    if str(error):
        assert str(error) not in " ".join(result.warnings)


@pytest.mark.asyncio
async def test_auction_search_returns_source_backed_success():
    result = await run_auction_search(
        {"query": "Domitian denarius", "limit": 5},
        provider_runners=[_provider("numisbids", [_auction_candidate()])],
        observed_at=OBSERVED_AT,
    )

    assert result.outcome == "complete"
    assert len(result.items) == 1
    item = result.items[0]
    assert item.kind == "auction_lot"
    assert item.auction_house == "Example Numismatic Auction"
    assert item.lot_number == "1"
    assert item.sale_date.isoformat() == "2026-10-01"
    assert item.estimate == 250
    assert item.current_bid == 175


@pytest.mark.asyncio
async def test_auction_search_returns_no_match_for_successful_zero_results():
    result = await run_auction_search(
        {"query": "Domitian denarius"},
        provider_runners=[_provider("numisbids", [])],
        observed_at=OBSERVED_AT,
    )

    assert result.outcome == "no_match"
    assert result.items == []
    assert result.provider_attempts[0].status == "no_match"


@pytest.mark.asyncio
async def test_auction_search_preserves_valid_results_when_another_provider_fails():
    result = await run_auction_search(
        {"query": "Domitian denarius"},
        provider_runners=[
            _provider("numisbids", [_auction_candidate()]),
            _provider("auction_search_secondary", error=httpx.TransportError("failed")),
        ],
        observed_at=OBSERVED_AT,
    )

    assert result.outcome == "partial"
    assert len(result.items) == 1
    assert [attempt.status for attempt in result.provider_attempts] == ["success", "failure"]


@pytest.mark.asyncio
async def test_auction_search_treats_invalid_candidate_shape_as_malformed():
    result = await run_auction_search(
        {"query": "Domitian denarius"},
        provider_runners=[_provider("numisbids", [{"url": "https://www.numisbids.com/sale/1/lot/1"}])],
        observed_at=OBSERVED_AT,
    )

    assert result.outcome == "unavailable"
    assert result.items == []
    assert result.provider_attempts[0].status == "malformed"
