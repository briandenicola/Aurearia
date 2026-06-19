"""Web search and dealer page tools.

- Anthropic/Claude: uses Claude's built-in web_search (handled by LangChain)
- Ollama: uses SearXNG via HTTP
- fetch_dealer_page: fetches a dealer URL and extracts coin listing data
"""

import logging
import re
from html.parser import HTMLParser

import httpx
from langchain_core.tools import tool

from app.config import settings
from app.outbound import safe_get, validate_outbound_url

logger = logging.getLogger(__name__)

# Trusted coin dealer domains for search filtering
TRUSTED_DOMAINS = [
    "vcoins.com",
    "forumancientcoins.com",
    "hjbltd.com",
    "biddr.com",
    "catawiki.com",
    "ma-shops.com",
    "coinshows.com",
    "coinshows-usa.com",
    "money.org",
    "pngdealers.org",
    "nyinc.info",
]


def _domain_matches(domain: str, expected_domain: str) -> bool:
    """Return True for exact host matches or subdomains of expected_domain."""
    clean_domain = domain.strip(".").lower()
    clean_expected = expected_domain.strip(".").lower()
    return clean_domain == clean_expected or clean_domain.endswith(f".{clean_expected}")


def create_searxng_search(searxng_url: str = ""):
    """Create a SearXNG search tool with a specific URL."""
    url = validate_outbound_url(searxng_url or settings.searxng_url, "searxng_url")

    @tool
    async def searxng_search(query: str) -> str:
        """Search the web for current information.

        Use this tool when you need to find real-time data such as items for
        sale, upcoming events, prices, or any other information that requires
        a live web search.  Pass a descriptive search query and receive
        titles, URLs, and text snippets from multiple search engines.
        """
        if not url:
            return "Search error: SearXNG is not configured."
        logger.debug("[searxng] Searching: %.120s (url=%s)", query, url)
        try:
            resp = await safe_get(
                f"{url}/search",
                field_name="searxng_url",
                params={
                    "q": query,
                    "format": "json",
                    "engines": "google,bing,duckduckgo",
                    "categories": "general",
                },
                timeout=settings.verification_timeout,
            )
            resp.raise_for_status()
            data = resp.json()
        except httpx.HTTPError as e:
            return f"Search error: {e}. SearXNG may be unavailable."
        except Exception as e:
            return f"Search failed: {e}"

        results = data.get("results", [])[:settings.max_search_results]
        if not results:
            logger.debug("[searxng] No results for query: %.80s", query)
            return "No results found."

        logger.debug("[searxng] Got %d results for query: %.80s", len(results), query)

        formatted = []
        for r in results:
            formatted.append(f"Title: {r.get('title', '')}\nURL: {r.get('url', '')}\nSnippet: {r.get('content', '')}\n")
        return "\n---\n".join(formatted)

    return searxng_search


# Standard browser user-agent— many dealer sites block bot-like strings
_USER_AGENT = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
    "AppleWebKit/537.36 (KHTML, like Gecko) "
    "Chrome/131.0.0.0 Safari/537.36"
)


@tool
async def verify_url(url: str) -> str:
    """Fetch a URL and return its status and key content indicators.

    Used by verification agents to confirm a listing is live and available.
    """
    from urllib.parse import urlparse

    parsed = urlparse(url)
    domain = (parsed.hostname or "").lower().lstrip("www.")
    is_trusted = any(_domain_matches(domain, d) for d in TRUSTED_DOMAINS)

    # Detect search/category page URLs (not individual listings)
    path_lower = parsed.path.lower() + "?" + (parsed.query or "").lower()
    search_indicators = [
        "search.aspx", "/search?", "/search/", "/browse", "/category/",
        "/results", "viewmode=", "searchterm=", "/en/search",
    ]
    is_search_page = any(ind in path_lower for ind in search_indicators)

    try:
        resp = await safe_get(
            url,
            field_name="url",
            headers={"User-Agent": _USER_AGENT},
            timeout=settings.verification_timeout,
        )

        status = resp.status_code
        text = resp.text[:5000].lower()

        sold_indicators = ["sold", "auction ended", "realized price", "no longer available", "out of stock"]
        is_sold = any(indicator in text for indicator in sold_indicators)

        buy_indicators = ["add to cart", "buy now", "add to basket", "purchase", "bid now", "place bid"]
        has_buy = any(indicator in text for indicator in buy_indicators)

        # Also detect search pages from page content
        if not is_search_page:
            search_content_hints = ["showing results for", "items found", "search results", "refine your search"]
            is_search_page = any(hint in text for hint in search_content_hints)

        return (
            f"Status: {status}\n"
            f"Trusted Dealer Site: {is_trusted}\n"
            f"Search/Category Page (NOT individual listing): {is_search_page}\n"
            f"Sold/Unavailable: {is_sold}\n"
            f"Has Buy/Bid Option: {has_buy}\n"
            f"URL: {url}"
        )
    except Exception as e:
        return (
            f"Error fetching URL: {e}\n"
            f"Trusted Dealer Site: {is_trusted}\n"
            f"Search/Category Page: {is_search_page}\n"
            f"URL: {url}"
        )


@tool
async def fetch_dealer_page(url: str) -> str:
    """Fetch a coin dealer page and extract listing information.

    Use this tool to visit a dealer search results page or individual listing
    and extract the actual coin titles, prices, and URLs found on the page.
    This is the ONLY reliable way to get real listing data from dealer sites.

    Args:
        url: The dealer page URL to fetch (search results page or listing page)

    Returns:
        Extracted listing data with titles, prices, and URLs found on the page.
    """
    try:
        resp = await safe_get(
            url,
            field_name="url",
            headers={"User-Agent": _USER_AGENT},
            timeout=httpx.Timeout(15.0, connect=5.0, read=10.0),
        )

        if resp.status_code != 200:
            return f"Error: HTTP {resp.status_code} fetching {url}"

        html = resp.text
        from urllib.parse import urlparse

        domain = (urlparse(url).hostname or "").lower().lstrip("www.")

        # Extract listings based on the dealer site
        if _domain_matches(domain, "vcoins.com"):
            return _parse_vcoins(html, url)
        elif _domain_matches(domain, "ma-shops.com"):
            return _parse_mashops(html, url)
        else:
            return _parse_generic(html, url)

    except Exception as e:
        logger.warning("Error fetching dealer page %s: %s", url, e)
        return f"Error fetching page: {e}"


def _parse_vcoins(html: str, base_url: str) -> str:
    """Parse VCoins search results or listing page."""
    listings = []

    # VCoins listing pages have product links with prices
    # Look for individual item patterns in HTML
    # VCoins uses ViewItem.aspx?UniqueID= for individual listings
    item_pattern = re.compile(
        r'<a[^>]*href="([^"]*ViewItem[^"]*)"[^>]*>(.*?)</a>',
        re.IGNORECASE | re.DOTALL,
    )
    for match in item_pattern.finditer(html):
        link = match.group(1)
        text = re.sub(r"<[^>]+>", "", match.group(2)).strip()
        if text and len(text) > 10:
            # Make absolute URL
            if link.startswith("/"):
                link = "https://www.vcoins.com" + link
            listings.append({"title": text[:200], "url": link})

    # Extract prices — VCoins shows prices near listings
    price_pattern = re.compile(r'(?:US\s*)?\$[\d,]+(?:\.\d{2})?')
    prices = price_pattern.findall(html)

    # Match prices to listings
    for i, listing in enumerate(listings):
        if i < len(prices):
            listing["price"] = prices[i]
        else:
            listing["price"] = "See listing"

    if not listings:
        # Fallback: extract any useful text
        return _parse_generic(html, base_url)

    result = f"Found {len(listings)} listings on VCoins:\n\n"
    for i, item in enumerate(listings[:10], 1):
        result += f"{i}. {item['title']}\n"
        result += f"   Price: {item.get('price', 'See listing')}\n"
        result += f"   URL: {item['url']}\n\n"
    return result


def _parse_mashops(html: str, base_url: str) -> str:
    """Parse MA-Shops search results or listing page."""
    listings = []

    # MA-Shops uses product links with item descriptions
    item_pattern = re.compile(
        r'<a[^>]*href="(https?://www\.ma-shops\.com/[^"]*item\d+[^"]*)"[^>]*>'
        r'(.*?)</a>',
        re.IGNORECASE | re.DOTALL,
    )
    for match in item_pattern.finditer(html):
        link = match.group(1)
        text = re.sub(r"<[^>]+>", "", match.group(2)).strip()
        if text and len(text) > 10:
            listings.append({"title": text[:200], "url": link})

    # Extract prices
    price_pattern = re.compile(r'(?:EUR|USD|US\$|\$)\s*[\d,]+(?:\.\d{2})?')
    prices = price_pattern.findall(html)
    for i, listing in enumerate(listings):
        if i < len(prices):
            listing["price"] = prices[i]
        else:
            listing["price"] = "See listing"

    if not listings:
        return _parse_generic(html, base_url)

    result = f"Found {len(listings)} listings on MA-Shops:\n\n"
    for i, item in enumerate(listings[:10], 1):
        result += f"{i}. {item['title']}\n"
        result += f"   Price: {item.get('price', 'See listing')}\n"
        result += f"   URL: {item['url']}\n\n"
    return result


def _parse_generic(html: str, base_url: str) -> str:
    """Generic HTML parser — extract links, prices, and text from any dealer page."""
    from urllib.parse import urljoin

    class DealerPageParser(HTMLParser):
        def __init__(self) -> None:
            super().__init__()
            self.in_script_or_style = 0
            self.in_title = False
            self.title_parts: list[str] = []
            self.text_parts: list[str] = []
            self.current_href = ""
            self.current_link_parts: list[str] = []
            self.links: list[dict[str, str]] = []

        def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
            lower_tag = tag.lower()
            if lower_tag in {"script", "style"}:
                self.in_script_or_style += 1
                return
            if self.in_script_or_style > 0:
                return
            if lower_tag == "title":
                self.in_title = True
            elif lower_tag == "a":
                self.current_href = ""
                self.current_link_parts = []
                for key, value in attrs:
                    if key.lower() == "href" and value:
                        self.current_href = value
                        break

        def handle_endtag(self, tag: str) -> None:
            lower_tag = tag.lower()
            if lower_tag in {"script", "style"}:
                if self.in_script_or_style > 0:
                    self.in_script_or_style -= 1
                return
            if self.in_script_or_style > 0:
                return
            if lower_tag == "title":
                self.in_title = False
            elif lower_tag == "a" and self.current_href:
                text = re.sub(r"\s+", " ", "".join(self.current_link_parts)).strip()
                if text and len(text) > 15 and not self.current_href.startswith(("#", "javascript")):
                    abs_url = urljoin(base_url, self.current_href)
                    self.links.append({"title": text[:200], "url": abs_url})
                self.current_href = ""
                self.current_link_parts = []

        def handle_data(self, data: str) -> None:
            if self.in_script_or_style > 0:
                return
            if self.in_title:
                self.title_parts.append(data)
            self.text_parts.append(data)
            if self.current_href:
                self.current_link_parts.append(data)

    parser = DealerPageParser()
    parser.feed(html)
    links = parser.links

    # Find prices
    price_pattern = re.compile(r'(?:US\s*)?\$[\d,]+(?:\.\d{2})?|EUR\s*[\d,]+(?:\.\d{2})?|GBP\s*[\d,]+(?:\.\d{2})?')
    prices = price_pattern.findall(html)

    page_title = re.sub(r"\s+", " ", "".join(parser.title_parts)).strip() or "Unknown"

    # Get text-only version for context (first 2000 chars)
    text_only = re.sub(r"\s+", " ", "".join(parser.text_parts)).strip()[:2000]

    result = f"Page title: {page_title}\n"
    result += f"Base URL: {base_url}\n\n"

    if links:
        result += f"Found {len(links)} links on page. Most relevant:\n\n"
        for i, link in enumerate(links[:15], 1):
            result += f"{i}. {link['title']}\n   URL: {link['url']}\n"
        result += "\n"

    if prices:
        result += f"Prices found on page: {', '.join(prices[:20])}\n\n"

    result += f"Page content summary:\n{text_only[:1000]}"

    return result
