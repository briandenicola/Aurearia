"""NumisBids scraping tools — lot page parser and watchlist parser.

These tools fetch and parse HTML from numisbids.com to extract structured
coin auction data. Used by the auction search team and the Go API's
import/sync endpoints.
"""

import logging
import re

import httpx
from langchain_core.tools import tool

logger = logging.getLogger(__name__)

_USER_AGENT = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
    "AppleWebKit/537.36 (KHTML, like Gecko) "
    "Chrome/131.0.0.0 Safari/537.36"
)

_NUMISBIDS_BASE = "https://www.numisbids.com"

# Category mapping from NumisBids sidebar categories to app categories
_CATEGORY_MAP = {
    "celtic": "Other",
    "greek": "Greek",
    "oriental greek": "Greek",
    "roman provincial": "Roman",
    "roman republican": "Roman",
    "roman imperial": "Roman",
    "byzantine": "Byzantine",
    "early medieval": "Other",
    "islamic": "Other",
    "world": "Modern",
    "british": "Modern",
    "united states": "Modern",
}


def _map_category(text: str) -> str:
    """Map NumisBids category text to an app Category value."""
    lower = text.lower().strip()
    for key, value in _CATEGORY_MAP.items():
        if key in lower:
            return value
    return "Other"


def _parse_currency_value(text: str) -> tuple[float | None, str]:
    """Extract a numeric value and currency from a price string like '150 USD'."""
    match = re.search(r"([\d,]+(?:\.\d+)?)\s*(USD|EUR|GBP|CHF)", text)
    if match:
        value = float(match.group(1).replace(",", ""))
        currency = match.group(2)
        return value, currency
    return None, "USD"


@tool
async def scrape_numisbids_lot(url: str) -> dict:
    """Scrape a NumisBids lot page and extract structured coin auction data.

    Args:
        url: A NumisBids lot URL (e.g. https://www.numisbids.com/sale/10489/lot/1)

    Returns:
        Dictionary with title, description, estimate, currentBid, currency,
        imageUrl, auctionHouse, saleName, saleId, lotNumber, category, and url.
    """
    logger.debug("[numisbids] Scraping lot page: %s", url)

    try:
        async with httpx.AsyncClient(
            timeout=httpx.Timeout(15.0, connect=5.0, read=10.0),
            follow_redirects=True,
        ) as client:
            resp = await client.get(url, headers={"User-Agent": _USER_AGENT})

        if resp.status_code != 200:
            return {"error": f"HTTP {resp.status_code} fetching {url}"}

        html = resp.text
        return _parse_lot_page(html, url)

    except Exception as e:
        logger.warning("Error scraping NumisBids lot %s: %s", url, e)
        return {"error": f"Failed to fetch lot page: {e}"}


def _parse_lot_page(html: str, url: str) -> dict:
    """Parse a NumisBids lot detail page HTML into structured data."""
    result: dict = {"url": url}

    # Extract sale ID and lot number from URL
    sale_match = re.search(r"/sale/(\d+)/lot/(\d+)", url)
    if sale_match:
        result["saleId"] = sale_match.group(1)
        result["lotNumber"] = int(sale_match.group(2))

    # Auction house name: <span class="name">...</span>
    house_match = re.search(
        r'<span class="name">(.*?)</span>', html, re.DOTALL
    )
    result["auctionHouse"] = (
        _clean_html(house_match.group(1)) if house_match else ""
    )

    # Sale name: <b>...</b> after the auction house name span
    sale_name_match = re.search(
        r'<span class="name">.*?</span>\s*<br>\s*<b>(.*?)</b>', html, re.DOTALL
    )
    result["saleName"] = (
        _clean_html(sale_name_match.group(1)) if sale_name_match else ""
    )

    # Sale date: appears after the sale name as "dd Mon yyyy"
    date_match = re.search(
        r'</b>\s*(?:&nbsp;)?\s*(\d{1,2}\s+\w{3}\s+\d{4})', html
    )
    result["saleDate"] = date_match.group(1).strip() if date_match else ""

    # Lot description: <div class="description"><b>TITLE</b> rest of description</div>
    desc_match = re.search(
        r'<div class="description">(?:<div[^>]*>.*?</div>)?\s*(<b>.*?)'
        r'(?=<div class="(?:provenance|estimate|salenav)"|$)',
        html,
        re.DOTALL,
    )
    if not desc_match:
        # Fallback: try a simpler match
        desc_match = re.search(
            r'<div class="description">\s*(.*?)\s*</div>',
            html,
            re.DOTALL,
        )

    if desc_match:
        raw_desc = desc_match.group(1)
        # Title is the bold text at the start
        title_match = re.match(r"<b>(.*?)</b>(.*)", raw_desc, re.DOTALL)
        if title_match:
            result["title"] = _clean_html(title_match.group(1)).strip()
            result["description"] = _clean_html(
                title_match.group(2)
            ).strip()
        else:
            cleaned = _clean_html(raw_desc).strip()
            result["title"] = cleaned[:120]
            result["description"] = cleaned
    else:
        # Use <title> as fallback
        title_tag = re.search(r"<title>(.*?)</title>", html, re.DOTALL)
        result["title"] = _clean_html(title_tag.group(1)) if title_tag else ""
        result["description"] = ""

    # Estimate: "Estimate: 150 USD"
    est_match = re.search(r"Estimate:\s*(?:<[^>]*>)*([\d,]+(?:\.\d+)?\s*\w+)", html)
    if est_match:
        value, currency = _parse_currency_value(est_match.group(1))
        result["estimate"] = value
        result["currency"] = currency
    else:
        result["estimate"] = None
        result["currency"] = "USD"

    # Current bid: "Current bid: 200 USD" or similar
    bid_match = re.search(
        r"Current bid:.*?>([\d,]+(?:\.\d+)?\s*\w+)<", html, re.DOTALL
    )
    if bid_match:
        value, _ = _parse_currency_value(bid_match.group(1))
        result["currentBid"] = value
    else:
        result["currentBid"] = None

    # Image URL: high-res image from the lightbox link or og:image
    img_match = re.search(
        r'data-fslightbox[^>]*href="([^"]*)"', html
    )
    if not img_match:
        img_match = re.search(
            r'<meta property="og:image" content="([^"]*)"', html
        )
    if img_match:
        img_url = img_match.group(1)
        if img_url.startswith("//"):
            img_url = "https:" + img_url
        result["imageUrl"] = img_url
    else:
        result["imageUrl"] = ""

    # Category: check the active category in the sidebar
    cat_match = re.search(
        r'id="activecat"[^>]*>.*?<a[^>]*>(.*?)\s*\(', html, re.DOTALL
    )
    result["category"] = (
        _map_category(cat_match.group(1)) if cat_match else "Other"
    )

    return result


@tool
async def search_numisbids(query: str) -> list[dict]:
    """Search NumisBids across all auctions and return matching lots.

    Args:
        query: Search terms (e.g. "Roman denarius Augustus")

    Returns:
        List of lot summaries with url, title, estimate, imageUrl.
    """
    logger.debug("[numisbids] Searching: %s", query)

    search_url = f"{_NUMISBIDS_BASE}/searchall"

    try:
        async with httpx.AsyncClient(
            timeout=httpx.Timeout(15.0, connect=5.0, read=10.0),
            follow_redirects=True,
        ) as client:
            resp = await client.get(
                search_url,
                params={"searchall": query},
                headers={"User-Agent": _USER_AGENT},
            )

        if resp.status_code != 200:
            return [{"error": f"Search returned HTTP {resp.status_code}"}]

        return _parse_search_results(resp.text)

    except Exception as e:
        logger.warning("NumisBids search failed: %s", e)
        return [{"error": f"Search failed: {e}"}]


def _parse_search_results(html: str) -> list[dict]:
    """Parse NumisBids search results page into lot summaries."""
    lots: list[dict] = []

    # Search results contain lot links with thumbnails and estimates
    # Pattern: links to /sale/{saleId}/lot/{lotNum} with nearby image and text
    blocks = re.split(r'(?=<a[^>]*href="/sale/\d+/lot/\d+)', html)

    for block in blocks:
        link_match = re.search(r'href="(/sale/(\d+)/lot/(\d+))"', block)
        if not link_match:
            continue

        lot: dict = {
            "url": f"{_NUMISBIDS_BASE}{link_match.group(1)}",
            "saleId": link_match.group(2),
            "lotNumber": int(link_match.group(3)),
        }

        # Thumbnail image
        img_match = re.search(r'<img[^>]*src="([^"]*)"', block)
        if img_match:
            img_url = img_match.group(1)
            if img_url.startswith("//"):
                img_url = "https:" + img_url
            lot["imageUrl"] = img_url
        else:
            lot["imageUrl"] = ""

        # Extract text content for title
        text = _clean_html(block).strip()
        lot["title"] = text[:200] if text else ""

        # Estimate
        est_match = re.search(r"Estimate:\s*([\d,]+(?:\.\d+)?\s*\w+)", block)
        if est_match:
            value, currency = _parse_currency_value(est_match.group(1))
            lot["estimate"] = value
            lot["currency"] = currency
        else:
            lot["estimate"] = None
            lot["currency"] = "USD"

        lots.append(lot)

    logger.debug("[numisbids] Search returned %d results", len(lots))
    return lots


def _clean_html(text: str) -> str:
    """Strip HTML tags and normalize whitespace."""
    clean = re.sub(r"<[^>]+>", " ", text)
    clean = re.sub(r"&nbsp;", " ", clean)
    clean = re.sub(r"&amp;", "&", clean)
    clean = re.sub(r"&lt;", "<", clean)
    clean = re.sub(r"&gt;", ">", clean)
    clean = re.sub(r"&#\d+;", "", clean)
    clean = re.sub(r"\s+", " ", clean)
    return clean.strip()
