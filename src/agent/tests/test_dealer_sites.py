"""Direct dealer-site search: parsers against saved real pages, budget rules, and wiring.

The fixtures under tests/fixtures/dealer_sites are trimmed copies of real
responses captured in September 2026 (vCoins and MA-Shops search pages, HJB's
inventory API), so a site markup change shows up as a failing parser test.
"""

import json
from decimal import Decimal
from pathlib import Path
from typing import Any
from urllib.parse import urlsplit

import pytest

from app.llm.provider import get_search_model
from app.models.requests import LLMConfig
from app.teams import coin_search
from app.teams.coin_search import run_market_search
from app.teams.specialist_contracts import _parse_currency, canonical_source_identity
from app.tools import dealer_sites
from app.tools.dealer_sites import (
    SiteListing,
    adapters_for_hosts,
    apply_budget,
    extract_search_terms,
    parse_budget,
    parse_hjb_results,
    parse_mashops_results,
    parse_price,
    parse_vcoins_results,
)
from app.tools.search import RegisteredDealerHttp

FIXTURES = Path(__file__).parent / "fixtures" / "dealer_sites"
LLM = LLMConfig(provider="anthropic", api_key="test", model="test")
ALL_ADAPTER_HOSTS = {"vcoins.com", "ma-shops.com", "hjbltd.com"}


def fixture_text(name: str) -> str:
    return (FIXTURES / name).read_text(encoding="utf-8")


def fixture_json(name: str) -> Any:
    return json.loads(fixture_text(name))


class FixtureDealerHttp:
    """Serves saved pages and records every request an adapter makes."""

    def __init__(self, hjb_payload: Any | None = None):
        self.gets: list[tuple[str, dict[str, str]]] = []
        self.posts: list[tuple[str, dict[str, Any]]] = []
        self.hjb_payload = hjb_payload if hjb_payload is not None else fixture_json("hjb_filter_results_caligula.json")

    async def get_text(self, url: str, params: dict[str, str]) -> str:
        self.gets.append((url, dict(params)))
        host = urlsplit(url).hostname
        if host == "www.vcoins.com":
            return fixture_text("vcoins_search_caligula.html")
        if host == "www.ma-shops.com":
            return fixture_text("mashops_search_caligula.html")
        raise AssertionError(f"unexpected GET {url}")

    async def post_json(self, url: str, body: dict[str, Any]) -> Any:
        self.posts.append((url, dict(body)))
        assert urlsplit(url).hostname == "www.hjbltd.com"
        return self.hjb_payload


# --- parsers --------------------------------------------------------------


def test_vcoins_parser_reads_current_results_and_skips_featured_ads():
    listings = parse_vcoins_results(fixture_text("vcoins_search_caligula.html"))

    # The fixture holds one paid "Featured" ad plus five real results.
    assert len(listings) == 5
    assert all(listing.url.startswith("https://www.vcoins.com/en/stores/") for listing in listings)
    assert all("/product/" in listing.url for listing in listings)
    assert {listing.currency for listing in listings} == {"USD", "EUR", "GBP"}
    first = listings[0]
    assert (first.title, first.dealer, first.price, first.currency) == (
        "Roman Empire, Caligula 37-41, Bronze Quadrans", "Aegean Numismatics", Decimal("395.00"), "USD"
    )
    assert all(listing.dealer == listing.dealer.strip() for listing in listings)
    assert all(listing.image_url.startswith("https://") for listing in listings)


def test_vcoins_parser_uses_the_discounted_price_not_the_struck_one():
    listings = parse_vcoins_results(fixture_text("vcoins_search_caligula.html"))
    germanicus = next(listing for listing in listings if listing.title.startswith("Germanicus"))
    assert (germanicus.price, germanicus.currency) == (Decimal("2400.00"), "EUR")  # was € 3,200.00


def test_mashops_parser_uses_discount_price_ignores_shipping_and_handles_new_badge():
    listings = parse_mashops_results(fixture_text("mashops_search_caligula.html"))

    assert len(listings) == 4
    by_shop = {listing.dealer: listing for listing in listings}
    assert by_shop["MA-Shops (noel)"].price == Decimal("568.74")  # not the struck 682.72 or 56.99 shipping
    assert by_shop["MA-Shops (khouli)"].price == Decimal("95000.00")
    assert by_shop["MA-Shops (fenzl)"].title.startswith("Sesterz 37-41 Caligula")  # row with a "New!" badge
    assert all(listing.currency == "USD" for listing in listings)
    assert all(listing.url.startswith("https://www.ma-shops.com/") for listing in listings)


def test_hjb_parser_keeps_only_items_with_stock():
    assert parse_hjb_results(fixture_json("hjb_filter_results_sold_only.json")) == []

    [listing] = parse_hjb_results(fixture_json("hjb_filter_results_caligula.json"))
    assert listing == SiteListing(
        title="Caligula. AE 21",
        url="https://www.hjbltd.com/#!/inventory/item-detail/cc/117376",
        dealer="Harlan J. Berk",
        price=Decimal("400"),
        currency="USD",
        image_url=listing.image_url,
    )


@pytest.mark.parametrize("payload", [None, [], {"status": "error", "message": "No data found"}, {"status": "ok"}])
def test_hjb_parser_treats_unexpected_payloads_as_no_results(payload):
    assert parse_hjb_results(payload) == []


@pytest.mark.parametrize(
    ("text", "expected"),
    [
        ("US$ 4,000.00", (Decimal("4000.00"), "USD")),
        ("€ 235.00", (Decimal("235.00"), "EUR")),
        ("£ 850.00", (Decimal("850.00"), "GBP")),
        ("568.74 US$", (Decimal("568.74"), "USD")),
        ("See listing", (None, None)),
    ],
)
def test_parse_price(text, expected):
    assert parse_price(text) == expected


# --- query interpretation and budget --------------------------------------


@pytest.mark.parametrize(
    ("query", "terms", "budget"),
    [
        ("Find me any Caligula coins under $500", "Caligula", (Decimal("500"), "USD")),
        ("show me a Constantine gold solidus below 3k euros", "Constantine gold solidus", (Decimal("3000"), "EUR")),
        ("I'd like Athens tetradrachms up to £1,200", "Athens tetradrachms", (Decimal("1200"), "GBP")),
        ("Roman Emperors", "Roman Emperors", (None, None)),
    ],
)
def test_query_interpretation(query, terms, budget):
    assert extract_search_terms(query) == terms
    assert parse_budget(query) == budget


def test_budget_never_converts_currencies():
    listings = [
        SiteListing("in budget", "https://a/1", "d", Decimal("450"), "USD"),
        SiteListing("over budget", "https://a/2", "d", Decimal("501"), "USD"),
        SiteListing("euro", "https://a/3", "d", Decimal("100"), "EUR"),
        SiteListing("no price", "https://a/4", "d", None, None),
    ]
    result = apply_budget(listings, Decimal("500"), "USD")
    assert [listing.title for listing in result.listings] == ["in budget"]
    assert (result.over_budget, result.other_currency, result.no_price) == (1, 1, 1)
    assert apply_budget(listings, None, None).listings == listings


def test_adapter_selection_splits_configured_hosts():
    adapters, remaining = adapters_for_hosts({"www.vcoins.com", "ma-shops.com", "biddr.com", "hjbltd.com"})
    assert [adapter.name for adapter in adapters] == ["vcoins", "ma_shops", "harlan_j_berk"]
    assert remaining == {"biddr.com"}


# --- end to end through run_market_search ---------------------------------


@pytest.mark.asyncio
async def test_market_search_queries_dealer_sites_directly_and_enforces_budget(monkeypatch):
    async def no_web_search(*_args, **_kwargs):
        raise AssertionError("dealers with a site adapter must not go through web search")

    monkeypatch.setattr(coin_search, "_search_dealer_pages", no_web_search)
    http = FixtureDealerHttp()

    result = await run_market_search(
        {"query": "Find me any Caligula coins under $500", "limit": 10},
        llm_config=LLM,
        source_hosts=ALL_ADAPTER_HOSTS,
        dealer_http=http,
    )

    assert result.outcome == "complete"
    assert {attempt.provider for attempt in result.provider_attempts} == {"vcoins", "ma_shops", "harlan_j_berk"}
    assert result.items, "expected listings within budget"
    for item in result.items:
        assert item.currency == "USD" and item.listed_price <= 500
        assert item.availability == "available"
        assert item.verification_state == "verified"
    # Sources take turns instead of the first dealer filling every slot.
    assert [item.provider for item in result.items[:3]] == ["vcoins", "ma_shops", "harlan_j_berk"]
    # Listings the site's own search matched on the title come first.
    assert result.items[0].title == "Roman Empire, Caligula 37-41, Bronze Quadrans"
    # € and £ listings can't be compared with a US$ budget, and the result says so.
    assert any("another currency" in warning for warning in result.warnings)

    vcoins_url, vcoins_params = http.gets[0]
    assert vcoins_params["searchQuery"] == "Caligula"
    assert http.gets[1][1] == {"searchstr": "Caligula"}
    hjb_url, hjb_body = http.posts[0]
    assert hjb_url == "https://www.hjbltd.com/api/cms/filter_results"
    assert hjb_body["Keyword"] == "Caligula"
    assert hjb_body["Quantity"] == 1  # in-stock only
    assert (hjb_body["PriceLow"], hjb_body["PriceHigh"]) == (0, 500)


@pytest.mark.asyncio
async def test_explicit_search_terms_and_budget_override_parsing():
    http = FixtureDealerHttp()
    result = await run_market_search(
        {"query": "something nice", "search_terms": "Caligula", "max_price": 1000, "currency": "EUR", "limit": 10},
        llm_config=LLM,
        source_hosts={"vcoins.com"},
        dealer_http=http,
    )
    assert http.gets[0][1]["searchQuery"] == "Caligula"
    assert [(item.listed_price, item.currency) for item in result.items] == [(Decimal("235.00"), "EUR")]


@pytest.mark.asyncio
async def test_hjb_sold_lots_never_reach_results():
    result = await run_market_search(
        {"query": "Caligula", "limit": 10},
        llm_config=LLM,
        source_hosts={"hjbltd.com"},
        dealer_http=FixtureDealerHttp(hjb_payload=fixture_json("hjb_filter_results_sold_only.json")),
    )
    assert result.outcome == "no_match"
    assert result.items == []


@pytest.mark.asyncio
async def test_one_dealer_failing_keeps_the_others(monkeypatch):
    class FlakyHttp(FixtureDealerHttp):
        async def post_json(self, url, body):
            import httpx

            raise httpx.TransportError("blocked")

    result = await run_market_search(
        {"query": "Caligula", "limit": 10},
        llm_config=LLM,
        source_hosts=ALL_ADAPTER_HOSTS,
        dealer_http=FlakyHttp(),
    )
    statuses = {attempt.provider: attempt.status for attempt in result.provider_attempts}
    assert statuses == {"vcoins": "success", "ma_shops": "success", "harlan_j_berk": "failure"}
    assert result.outcome == "partial"
    assert result.items


@pytest.mark.asyncio
async def test_dealers_without_an_adapter_still_use_web_search_restricted_to_their_hosts(monkeypatch):
    seen: dict[str, Any] = {}

    async def fake_collect(llm_config, query, limit, *, allowed_fetch_hosts=None, **_kwargs):
        seen["hosts"] = allowed_fetch_hosts
        return [{
            "name": "Caligula As", "sourceUrl": "https://www.forumancientcoins.com/catalog/roman/1.html",
            "estPrice": "$300", "availability": "Available",
        }]

    monkeypatch.setattr(coin_search, "_collect_market_candidates", fake_collect)
    result = await run_market_search(
        {"query": "Caligula under $500", "limit": 10},
        llm_config=LLM,
        source_hosts={"vcoins.com", "forumancientcoins.com"},
        dealer_http=FixtureDealerHttp(),
    )
    assert seen["hosts"] == {"forumancientcoins.com"}
    assert "configured_dealer_search" in {item.provider for item in result.items}
    assert all(item.listed_price <= 500 for item in result.items)


# --- supporting changes ---------------------------------------------------


def test_hash_routed_item_pages_keep_distinct_identities():
    first = canonical_source_identity("https://www.hjbltd.com/#!/inventory/item-detail/cc/117376")
    second = canonical_source_identity("https://www.hjbltd.com/#!/inventory/item-detail/cc/118011")
    assert first != second
    # Ordinary in-page anchors are still ignored.
    assert canonical_source_identity("https://a.example/p#photos") == canonical_source_identity("https://a.example/p")


def test_currency_symbols_are_recognized():
    assert (_parse_currency("€ 235.00"), _parse_currency("£ 850"), _parse_currency("US$ 5")) == ("EUR", "GBP", "USD")


def test_web_search_tool_is_restricted_to_configured_domains():
    bound = get_search_model(LLM, allowed_domains={"forumancientcoins.com", "Biddr.com"})
    [tool] = bound.kwargs["tools"]
    assert tool["allowed_domains"] == ["biddr.com", "forumancientcoins.com"]
    [unrestricted] = get_search_model(LLM).kwargs["tools"]
    assert "allowed_domains" not in unrestricted


@pytest.mark.asyncio
async def test_registered_dealer_http_refuses_hosts_outside_the_boundary():
    http = RegisteredDealerHttp({"hjbltd.com"})
    with pytest.raises(ValueError):
        await http.post_json("https://api.example.org/api/cms/filter_results", {})


def test_every_adapter_host_is_in_the_default_dealer_list():
    default_sources = {"vcoins.com", "ma-shops.com", "forumancientcoins.com", "biddr.com", "catawiki.com", "hjbltd.com"}
    assert {adapter.host for adapter in dealer_sites.DEALER_SITE_ADAPTERS} <= default_sources


@pytest.mark.asyncio
async def test_bot_challenge_is_reported_as_a_failed_source(monkeypatch):
    import httpx

    async def challenged_get(url, **_kwargs):
        return httpx.Response(202, headers={"x-amzn-waf-action": "challenge"}, request=httpx.Request("GET", url))

    monkeypatch.setattr("app.tools.search.safe_registered_get", challenged_get)
    result = await run_market_search(
        {"query": "Caligula", "limit": 5},
        llm_config=LLM,
        source_hosts={"vcoins.com"},
        dealer_http=RegisteredDealerHttp({"vcoins.com"}),
    )
    assert [(attempt.provider, attempt.status) for attempt in result.provider_attempts] == [("vcoins", "failure")]
    assert result.items == []


@pytest.mark.asyncio
async def test_market_search_returns_five_by_default_and_reports_how_many_more_matched():
    result = await run_market_search(
        {"query": "Caligula"},
        llm_config=LLM,
        source_hosts=ALL_ADAPTER_HOSTS,
        dealer_http=FixtureDealerHttp(),
    )
    # 5 vCoins + 4 MA-Shops + 1 HJB listings match; the owner sees 5 and is told about the rest.
    assert len(result.items) == 5
    assert result.truncation.truncated is True
    assert result.truncation.omitted_items == 5


def test_market_and_auction_tool_descriptions_state_the_default_limit():
    from app.tools.copilot_collection_tools import build_copilot_tool_definitions

    tools = {tool.name: tool for tool in build_copilot_tool_definitions(["market_search", "auction_search"])}
    for name in ("market_search", "auction_search"):
        assert "default of 5" in tools[name].description
        assert "up to 10 only when the owner asks for more" in tools[name].description


@pytest.mark.asyncio
async def test_legacy_chat_search_shows_at_most_ten_listings(monkeypatch):
    urls = [f"https://www.forumancientcoins.com/catalog/{index}.html" for index in range(15)]
    fetched = "".join(f"--- Source: {url} ---\nlisting {index}\n\n" for index, url in enumerate(urls))

    async def fake_search(*_args, **_kwargs):
        return "search results"

    async def fake_fetch(*_args, **_kwargs):
        return fetched

    async def fake_format(*_args, **_kwargs):
        return "", [{"name": f"Coin {index}", "sourceUrl": url} for index, url in enumerate(urls)]

    monkeypatch.setattr(coin_search, "_search_dealer_pages", fake_search)
    monkeypatch.setattr(coin_search, "_fetch_dealer_pages", fake_fetch)
    monkeypatch.setattr(coin_search, "_format_dealer_candidates", fake_format)
    team = coin_search.create_coin_search_team(LLM, "", {"forumancientcoins.com"})
    state = await team.ainvoke(
        {"user_message": "Caligula", "messages": [], "search_results": "", "fetched_listings": ""}
    )

    content = state["messages"][-1].content
    payload = json.loads(content.split("```json\n", 1)[1].split("\n```", 1)[0])
    assert len(payload) == coin_search.LEGACY_MAX_LISTINGS == 10
