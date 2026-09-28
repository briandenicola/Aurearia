"""Direct dealer-site search adapters.

Web search finds dealer pages through a search engine's index, which is full of
old listing pages for coins that sold long ago. Querying a dealer's own search
returns only what the dealer currently has for sale, with structured price and
currency, so dealers that expose a usable search get an adapter here instead.

Each adapter knows one site: how to build its search request and how to parse
the response into ``SiteListing`` values. Network access goes through a
``DealerHttp`` bound to the administrator-configured hosts, so adapters can't
reach anything outside that boundary and tests can substitute saved pages.
"""

from __future__ import annotations

import html as html_lib
import logging
import re
from collections.abc import Awaitable, Callable, Mapping, Sequence
from dataclasses import dataclass
from decimal import Decimal, InvalidOperation
from typing import Any, Protocol
from urllib.parse import urljoin, urlsplit

logger = logging.getLogger(__name__)

# Symbols seen on dealer pages, longest first so "US$" wins over "$".
_CURRENCY_SYMBOLS: tuple[tuple[str, str], ...] = (
    ("US$", "USD"),
    ("USD", "USD"),
    ("EUR", "EUR"),
    ("GBP", "GBP"),
    ("CHF", "CHF"),
    ("€", "EUR"),
    ("£", "GBP"),
    ("$", "USD"),
)


@dataclass(frozen=True)
class SiteListing:
    """One listing a dealer's own search reports as currently for sale."""

    title: str
    url: str
    dealer: str
    price: Decimal | None
    currency: str | None
    image_url: str = ""

    def as_candidate(self) -> dict[str, Any]:
        """Shape the listing like the formatter output the specialist adapter expects."""
        return {
            "name": self.title,
            "sourceUrl": self.url,
            "sourceName": self.dealer,
            "listed_price": self.price,
            "currency": self.currency,
            "availability": "available",
            "imageUrl": self.image_url,
            "verificationState": "verified",
            "confidence": "high",
        }


class DealerHttp(Protocol):
    """Host-bound transport; implementations reject URLs outside the configured hosts."""

    async def get_text(self, url: str, params: Mapping[str, str]) -> str: ...

    async def post_json(self, url: str, body: Mapping[str, Any]) -> Any: ...


@dataclass(frozen=True)
class DealerSiteAdapter:
    """A dealer whose own search can be queried directly."""

    name: str
    host: str
    search: Callable[[DealerHttp, str, Decimal | None, str | None, int], Awaitable[list[SiteListing]]]


def parse_price(text: str) -> tuple[Decimal | None, str | None]:
    """Parse "US$ 4,000.00", "€ 235.00" or "568.74 US$" into amount and ISO currency."""
    cleaned = html_lib.unescape(text or "").replace("\xa0", " ").strip()
    upper = cleaned.upper()
    currency = next((code for symbol, code in _CURRENCY_SYMBOLS if symbol in upper), None)
    match = re.search(r"\d[\d,]*(?:\.\d+)?", cleaned)
    if not match:
        return None, currency
    try:
        return Decimal(match.group(0).replace(",", "")), currency
    except InvalidOperation:
        return None, currency


def _text(fragment: str) -> str:
    return html_lib.unescape(re.sub(r"\s+", " ", re.sub(r"<[^>]+>", " ", fragment))).strip()


# --- vCoins ---------------------------------------------------------------

VCOINS_SEARCH_URL = "https://www.vcoins.com/en/Search.aspx"


def parse_vcoins_results(page: str, base_url: str = VCOINS_SEARCH_URL) -> list[SiteListing]:
    """Parse a vCoins search results page.

    Each result is a ``div.item``. The first few are paid "Featured" ads that
    may not match the query, so they're skipped. A sale shows a struck-through
    old price next to ``span.discounted-price``; the discounted price is used.
    """
    listings: list[SiteListing] = []
    for block in re.split(r'<div class="item">', page)[1:]:
        if "featured-listing" in block:
            continue
        title = re.search(r'_lnkTitle"[^>]*href="([^"]+)"[^>]*>(.*?)</a>', block, re.S)
        if not title:
            continue
        store = re.search(r'_lnkStore" title="([^"]*)"', block)
        price_html = re.search(r'<div class="prices">\s*<h3>(.*?)</h3>', block, re.S)
        price_text = ""
        if price_html:
            discounted = re.search(r'class="discounted-price">(.*?)</span>', price_html.group(1), re.S)
            price_text = discounted.group(1) if discounted else re.sub(
                r'<span style="text-decoration: line-through;">.*?</span>', "", price_html.group(1), flags=re.S
            )
        price, currency = parse_price(_text(price_text))
        image = re.search(r'_imgImage"[^>]*src="([^"]+)"', block)
        listings.append(SiteListing(
            title=_text(title.group(2)),
            url=urljoin(base_url, html_lib.unescape(title.group(1))),
            dealer=html_lib.unescape(store.group(1)).strip() if store else "vCoins dealer",
            price=price,
            currency=currency,
            image_url=html_lib.unescape(image.group(1)) if image else "",
        ))
    return listings


async def search_vcoins(
    http: DealerHttp, terms: str, max_price: Decimal | None, currency: str | None, limit: int
) -> list[SiteListing]:
    # vCoins prices stay in each dealer's own currency, so budget filtering
    # happens after parsing rather than through its price-range parameters.
    params = {
        "search": "true",
        "searchQuery": terms,
        "searchCategoryAncient": "True",
        "searchCategoryUs": "True",
        "searchCategoryWorld": "True",
        "searchCategoryMints": "True",
        "searchTitleAndDescription": "True",
        "searchUseThesaurus": "True",
        "searchDisplay": "1",
        "searchIdStore": "0",
        "searchMaxRecords": "100",
    }
    return parse_vcoins_results(await http.get_text(VCOINS_SEARCH_URL, params))


# --- MA-Shops -------------------------------------------------------------

MASHOPS_SEARCH_URL = "https://www.ma-shops.com/shops/search.php"


def parse_mashops_results(page: str, base_url: str = MASHOPS_SEARCH_URL) -> list[SiteListing]:
    """Parse an MA-Shops search results table.

    Each result is a table row with ``spx-title`` and ``spx-price`` cells. A
    sale shows ``price oldPrice`` (struck) and ``price discount``; the shipping
    note inside the cell carries its own price, which must be ignored.
    """
    listings: list[SiteListing] = []
    for row in re.findall(r"<TR><TD class='spxThumbTd'.*?</TR>", page, re.S | re.I):
        # Take the title cell first, then its first link: a "New!" badge span can
        # precede the link. Two linear searches avoid a nested, backtracking pattern.
        title_cell = re.search(r'spx-title">(.*?)</TD>', row, re.S | re.I)
        title = re.search(r'<A HREF="([^"]+)">(.*?)</A>', title_cell.group(1), re.S | re.I) if title_cell else None
        if not title:
            continue
        cell = re.search(r'spx-price[^>]*>(.*?)</TD>', row, re.S | re.I)
        price_text = ""
        if cell:
            without_shipping = re.sub(
                r"<span class='vatShippingInfo'>.*?</span>\s*</span>", "", cell.group(1), flags=re.S
            )
            without_old = re.sub(r"<del[^>]*>.*?</del>", "", without_shipping, flags=re.S | re.I)
            price = re.search(r"class='price[^']*'>([^<]*)<", without_old)
            price_text = price.group(1) if price else ""
        amount, currency = parse_price(price_text)
        url = urljoin(base_url, html_lib.unescape(title.group(1)))
        shop = (urlsplit(url).path.strip("/").split("/") or [""])[0]
        image = re.search(r"spx-thumb' SRC=\"([^\"]+)\"", row, re.I)
        listings.append(SiteListing(
            title=_text(title.group(2)),
            url=url,
            dealer=f"MA-Shops ({shop})" if shop else "MA-Shops",
            price=amount,
            currency=currency,
            image_url=html_lib.unescape(image.group(1)) if image else "",
        ))
    return listings


async def search_mashops(
    http: DealerHttp, terms: str, max_price: Decimal | None, currency: str | None, limit: int
) -> list[SiteListing]:
    return parse_mashops_results(await http.get_text(MASHOPS_SEARCH_URL, {"searchstr": terms}))


# --- Harlan J. Berk -------------------------------------------------------

HJB_SEARCH_URL = "https://www.hjbltd.com/api/cms/filter_results"
HJB_ANCIENT_COINS_GROUP = "cc"


def parse_hjb_results(payload: Any) -> list[SiteListing]:
    """Parse HJB's inventory JSON, keeping only items with stock on hand.

    ``Quantity`` is the in-stock count; closed Buy-or-Bid lots stay in the
    unfiltered feed with ``Quantity`` 0, so they're dropped here even though
    the request already asks for in-stock items only.
    """
    if not isinstance(payload, Mapping) or payload.get("status") != "ok":
        return []
    data = payload.get("data")
    items = data.get("items") if isinstance(data, Mapping) else None
    if not isinstance(items, Sequence) or isinstance(items, str | bytes):
        return []
    listings: list[SiteListing] = []
    for item in items:
        if not isinstance(item, Mapping):
            continue
        try:
            quantity = int(item.get("Quantity") or 0)
        except (TypeError, ValueError):
            quantity = 0
        number = str(item.get("InventoryNumber") or "").strip()
        title = str(item.get("Title") or "").strip()
        if quantity < 1 or not number.isdigit() or not title:
            continue
        group = str(item.get("InventoryGroup") or HJB_ANCIENT_COINS_GROUP).strip() or HJB_ANCIENT_COINS_GROUP
        raw_price = item.get("Price")
        try:
            price = Decimal(str(raw_price)) if raw_price not in (None, "") else None
        except InvalidOperation:
            price = None
        image = item.get("Image") if isinstance(item.get("Image"), str) else ""
        listings.append(SiteListing(
            title=html_lib.unescape(title),
            # HJB is a hash-routed app; item pages only exist under #!/.
            url=f"https://www.hjbltd.com/#!/inventory/item-detail/{group}/{number}",
            dealer="Harlan J. Berk",
            price=price,
            currency="USD",
            image_url=image,
        ))
    return listings


async def search_hjb(
    http: DealerHttp, terms: str, max_price: Decimal | None, currency: str | None, limit: int
) -> list[SiteListing]:
    body: dict[str, Any] = {
        "InventoryGroup": HJB_ANCIENT_COINS_GROUP,
        "Keyword": terms,
        "Offset": 1,
        "FetchNext": 60,
        "Quantity": 1,  # in-stock only
    }
    if max_price is not None and currency in (None, "USD"):
        # HJB ignores the whole request ("No data found") unless both bounds are sent.
        body["PriceLow"] = 0
        body["PriceHigh"] = int(max_price)
    return parse_hjb_results(await http.post_json(HJB_SEARCH_URL, body))


DEALER_SITE_ADAPTERS: tuple[DealerSiteAdapter, ...] = (
    DealerSiteAdapter(name="vcoins", host="vcoins.com", search=search_vcoins),
    DealerSiteAdapter(name="ma_shops", host="ma-shops.com", search=search_mashops),
    DealerSiteAdapter(name="harlan_j_berk", host="hjbltd.com", search=search_hjb),
)


def _normalize_host(host: str) -> str:
    return host.strip().lower().rstrip(".").removeprefix("www.")


def adapters_for_hosts(
    hosts: set[str], available: Sequence[DealerSiteAdapter] = DEALER_SITE_ADAPTERS
) -> tuple[list[DealerSiteAdapter], set[str]]:
    """Split configured hosts into direct-search adapters and hosts left for web search."""
    normalized = {_normalize_host(host): host for host in hosts if host.strip()}
    adapters = [adapter for adapter in available if adapter.host in normalized]
    covered = {adapter.host for adapter in adapters}
    remaining = {original for key, original in normalized.items() if key not in covered}
    return adapters, remaining


# --- Query interpretation -------------------------------------------------

_BUDGET_PATTERN = re.compile(
    r"\b(?:under|below|less than|up to|at most|no more than|max(?:imum)?|cheaper than|<=?)\s*"
    r"(?P<pre>US\$|\$|€|£|USD|EUR|GBP)?\s*(?P<amount>\d[\d,]*(?:\.\d+)?)\s*(?P<k>k\b)?\s*"
    r"(?P<post>dollars?|usd|euros?|eur|pounds?|gbp)?",
    re.IGNORECASE,
)
_BUDGET_CURRENCIES = {
    "us$": "USD", "$": "USD", "usd": "USD", "dollar": "USD", "dollars": "USD",
    "€": "EUR", "eur": "EUR", "euro": "EUR", "euros": "EUR",
    "£": "GBP", "gbp": "GBP", "pound": "GBP", "pounds": "GBP",
}
_FILLER_WORDS = {
    "a", "an", "and", "any", "available", "buy", "can", "coin", "coins", "could", "find", "for", "get",
    "i", "id", "im", "in", "is", "like", "listing", "listings", "looking", "me", "my", "of", "on", "online",
    "please", "purchase", "sale", "search", "show", "some", "the", "to", "want", "with", "would", "you",
}


def parse_budget(text: str) -> tuple[Decimal | None, str | None]:
    """Find a price ceiling such as "under $500" or "below 300 euros"."""
    match = _BUDGET_PATTERN.search(text or "")
    if not match:
        return None, None
    try:
        amount = Decimal(match.group("amount").replace(",", ""))
    except InvalidOperation:
        return None, None
    if match.group("k"):
        amount *= 1000
    token = (match.group("pre") or match.group("post") or "$").lower()
    return amount, _BUDGET_CURRENCIES.get(token, "USD")


def extract_search_terms(text: str) -> str:
    """Reduce a request like "find me any Caligula coins under $500" to "Caligula".

    Dealer search engines match keywords, so conversational words and the budget
    phrase only narrow or break the match.
    """
    without_budget = _BUDGET_PATTERN.sub(" ", text or "")
    words = re.findall(r"[\w'-]+", without_budget, re.UNICODE)
    kept = [word for word in words if word.lower().replace("'", "") not in _FILLER_WORDS]
    return " ".join(kept).strip() or (text or "").strip()


@dataclass
class BudgetFilterResult:
    listings: list[SiteListing]
    other_currency: int = 0
    no_price: int = 0
    over_budget: int = 0


def apply_budget(
    listings: Sequence[SiteListing], max_price: Decimal | None, currency: str | None
) -> BudgetFilterResult:
    """Keep listings at or under the budget, never converting currencies.

    A listing priced in a different currency, or with no readable price, can't
    be shown to be within budget, so it's left out and counted for a warning.
    """
    if max_price is None:
        return BudgetFilterResult(list(listings))
    budget_currency = currency or "USD"
    result = BudgetFilterResult([])
    for listing in listings:
        if listing.price is None or listing.currency is None:
            result.no_price += 1
        elif listing.currency != budget_currency:
            result.other_currency += 1
        elif listing.price > max_price:
            result.over_budget += 1
        else:
            result.listings.append(listing)
    return result
