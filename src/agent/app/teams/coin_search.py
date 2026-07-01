"""Team 1: Coin Search — two-phase search with page fetching.

Phase 1: Search the web for dealer pages (Anthropic uses built-in web_search;
         Ollama uses a ReAct agent with SearXNG tool — model decides when to search).
Phase 2: We fetch dealer pages from the URLs found and extract real listings.
Phase 3: Format the extracted listings into the CoinSuggestion JSON schema.
"""

import json
import logging
import re
from datetime import UTC, datetime
from typing import Annotated, TypedDict
from urllib.parse import urlparse

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.provider import create_search_agent, get_chat_model, get_search_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import AlertDiscoveryRequest, LLMConfig
from app.models.responses import AlertDiscoveryCandidate, AlertDiscoveryProvenance, AlertDiscoveryResponse
from app.safety import with_safety
from app.tools.numismatic_authority import normalize_candidate_references
from app.tools.search import fetch_dealer_page

logger = logging.getLogger(__name__)

TRUSTED_ALERT_FETCH_HOSTS = {
    "vcoins.com",
    "ma-shops.com",
    "forumancientcoins.com",
    "biddr.com",
    "catawiki.com",
    "hjbltd.com",
}

SEARCH_PROMPT = with_safety("""You are a numismatic search specialist. Search the web to find coins
currently for sale that match the user's request.

Search on these dealer sites:
- vcoins.com, ma-shops.com, forumancientcoins.com, biddr.com, catawiki.com, hjbltd.com

Use targeted site-specific queries like:
- "Domitian denarius for sale site:vcoins.com"
- "Greek tetradrachm Athens site:ma-shops.com"

Include the user's budget/price range in your searches if mentioned.
Run at least 3-5 searches across different dealer sites.

For each result you find, report the URL exactly as it appeared in the search results.
Do NOT invent or modify URLs. Do not use emojis.""")

FORMAT_PROMPT = with_safety("""You are a formatting specialist for a coin collecting application.
You receive raw coin listing data extracted from dealer websites.
Structure each listing into this exact JSON schema:

```json
[
  {
    "name": "Coin title from the listing",
    "description": "Brief description from the listing",
    "category": "Roman|Greek|Byzantine|Modern|Other",
    "era": "Time period",
    "ruler": "Ruler name",
    "material": "Gold|Silver|Bronze|Copper|Other",
    "denomination": "e.g. Denarius, Tetradrachm",
    "estPrice": "Listed price e.g. $150.00",
    "availability": "Available|Sold|Unknown based only on listing text",
    "imageUrl": "",
    "sourceUrl": "The exact URL from the listing data — never fabricate",
    "sourceName": "Dealer or site name",
    "candidateReferences": [
      {
        "catalog": "RIC",
        "volume": "VII",
        "number": "162",
        "uri": ""
      }
    ]
  }
]
```

Rules:
- Use ONLY data from the listing extracts. Do NOT invent fields.
- sourceUrl MUST be copied exactly from the data. NEVER fabricate URLs.
- Set imageUrl to "" (the frontend handles images)
- Infer category, era, ruler, material, denomination from the listing text
- Exclude listings that are clearly sold, sold out, unavailable, or no longer available
- Set availability from explicit listing text only; use "Unknown" when the listing does not say
- Add candidateReferences only when a catalog reference appears in the listing text
- candidateReferences items require catalog and number; volume and uri are optional
- If you cannot determine a field, use an empty string
- If no candidate references are present, return "candidateReferences": []
- Do not use emojis

Output ONLY the JSON array wrapped in ```json and ``` markers.""")

NO_RESULTS_PROMPT = (
    "You are an assistant in a coin collecting application. "
    "The user searched for coins to buy but no listings were found. "
    "Generate a brief, helpful response. Suggest broadening search criteria, "
    "checking back later, or trying specific dealer sites like vcoins.com "
    "or ma-shops.com. Keep it concise. Do not use emojis. "
    "Do not invent coin listings."
)


class CoinSearchState(TypedDict):
    """State for the coin search pipeline."""

    messages: Annotated[list, lambda a, b: a + b]
    search_results: str
    fetched_listings: str
    user_message: str


def create_coin_search_team(
    llm_config: LLMConfig,
    search_prompt: str = "",
    allowed_fetch_hosts: set[str] | None = None,
):
    """Create the coin search pipeline.

    Args:
        llm_config: LLM provider configuration
        search_prompt: Additional context from admin settings (prepended)
        allowed_fetch_hosts: Optional host allowlist for fetched listing pages
    """
    if search_prompt:
        combined_search = f"{search_prompt}\n\n{SEARCH_PROMPT}"
    else:
        combined_search = SEARCH_PROMPT

    use_react_agent = llm_config.provider == "ollama"
    if use_react_agent:
        search_agent = create_search_agent(llm_config)

    async def search_node(state: CoinSearchState) -> dict:
        """Phase 1: Search the web for dealer pages."""
        user_msg = state.get("user_message", "")
        logger.debug("[coin_search] search_node start — query: %.100s", user_msg)

        messages = [
            SystemMessage(content=combined_search),
            HumanMessage(
                content=f"Find coins for sale matching: {user_msg}\n\n"
                "Search multiple dealer sites and report all URLs you find."
            ),
        ]

        if use_react_agent:
            # Ollama: ReAct agent calls SearXNG tool autonomously
            result = await search_agent.ainvoke({"messages": messages})
            last_msg = result["messages"][-1]
            content = last_msg.content if isinstance(last_msg.content, str) else str(last_msg.content)
            logger.debug(
                "[coin_search] ReAct agent returned %d messages, content=%d chars",
                len(result["messages"]), len(content),
            )
        else:
            # Anthropic: built-in web_search handled server-side
            model = get_search_model(llm_config)
            response = await ainvoke_with_retry(model, messages)
            content = response.content if isinstance(response.content, str) else str(response.content)
            logger.debug("[coin_search] Anthropic search response=%d chars", len(content))

        return {"search_results": content, "messages": []}

    async def fetch_node(state: CoinSearchState) -> dict:
        """Phase 2: Fetch dealer pages and extract real listings."""
        import asyncio

        search_results = state.get("search_results", "")
        urls = _extract_urls(search_results)
        urls = _filter_allowed_fetch_urls(urls, allowed_fetch_hosts)
        logger.debug("[coin_search] fetch_node — found %d URLs to fetch", len(urls))

        if not urls:
            return {"fetched_listings": "", "messages": []}

        # Fetch up to 5 URLs in parallel
        tasks = [fetch_dealer_page.ainvoke({"url": u}) for u in urls[:5]]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        fetched = []
        for url, result in zip(urls[:5], results):
            if isinstance(result, Exception):
                logger.warning("Failed to fetch %s: %s", url, result)
                continue
            text = str(result)
            if not text.startswith("Error"):
                fetched.append(f"--- Source: {url} ---\n{text}")

        return {"fetched_listings": "\n\n".join(fetched), "messages": []}

    async def format_node(state: CoinSearchState) -> dict:
        """Phase 3: Format extracted listings into CoinSuggestion JSON."""
        fetched = state.get("fetched_listings", "")
        user_msg = state.get("user_message", "")
        search_results = state.get("search_results", "")
        model = get_chat_model(llm_config)
        logger.debug("[coin_search] format_node — fetched_listings=%d chars", len(fetched))

        if not fetched.strip():
            # No listings found — generate a helpful response via LLM (streams)
            messages = [
                SystemMessage(content=NO_RESULTS_PROMPT),
                HumanMessage(
                    content=f"The user asked: {user_msg}\n\n"
                    f"Search results summary:\n{search_results[:1000]}\n\n"
                    "No coin listings could be extracted. Generate a helpful response."
                ),
            ]
            response = await ainvoke_with_retry(model, messages)
            content = response.content if isinstance(response.content, str) else str(response.content)
            return {"messages": [AIMessage(content=content)]}

        # Format real listings via LLM (this call streams to user)
        messages = [
            SystemMessage(content=FORMAT_PROMPT),
            HumanMessage(
                content=f"User searched for: {user_msg}\n\n"
                f"Extracted listing data:\n{fetched}"
            ),
        ]
        response = await ainvoke_with_retry(model, messages)
        formatted = response.content if isinstance(response.content, str) else str(response.content)
        formatted = _enrich_references_with_authority_links(formatted)

        summary = (
            "I found some coins matching your search. "
            "Here are the listings I extracted from dealer sites."
        )
        return {"messages": [AIMessage(content=f"{summary}\n\n{formatted}")]}

    graph = StateGraph(CoinSearchState)
    graph.add_node("search", search_node)
    graph.add_node("fetch", fetch_node)
    graph.add_node("format", format_node)

    graph.set_entry_point("search")
    graph.add_edge("search", "fetch")
    graph.add_edge("fetch", "format")
    graph.add_edge("format", END)

    return graph.compile()


def _extract_urls(text: str) -> list[str]:
    """Extract dealer URLs from search results text."""
    urls = re.findall(r'https?://[^\s"\'<>)\],]+', text)
    # Deduplicate while preserving order
    seen = set()
    unique = []
    for url in urls:
        if url not in seen:
            seen.add(url)
            unique.append(url)
    return unique


def _enrich_references_with_authority_links(text: str) -> str:
    """Normalize candidate references and fill authority URIs when possible."""
    match = re.search(r"```json\s*\n(.*?)\n```", text, flags=re.DOTALL)
    if not match:
        return text

    json_block = match.group(1).strip()
    try:
        payload = json.loads(json_block)
    except json.JSONDecodeError:
        return text

    if not isinstance(payload, list):
        return text

    for item in payload:
        if not isinstance(item, dict):
            continue
        raw_refs = item.get("candidateReferences")
        item["candidateReferences"] = normalize_candidate_references(raw_refs if isinstance(raw_refs, list) else [])

    enriched = json.dumps(payload, ensure_ascii=False, indent=2)
    return f"{text[:match.start()]}```json\n{enriched}\n```{text[match.end():]}"


async def discover_alert_candidates(request: AlertDiscoveryRequest) -> AlertDiscoveryResponse:
    """Run stateless wishlist alert discovery using the existing coin search pipeline."""
    query = _alert_criteria_query(request.alert.criteria_snapshot)
    allowed_fetch_hosts = _trusted_alert_fetch_hosts(request.alert.criteria_snapshot.source_filters)
    graph = create_coin_search_team(request.llm, allowed_fetch_hosts=allowed_fetch_hosts)
    try:
        result = await graph.ainvoke({
            "messages": [],
            "search_results": "",
            "fetched_listings": "",
            "user_message": query,
        })
    except Exception:
        logger.exception("Wishlist alert discovery failed")
        return AlertDiscoveryResponse(
            candidates=[],
            warnings=["Discovery could not complete. Please try again later."],
            partial=True,
        )

    suggestions = _extract_json_array(_last_message_content(result))
    if not suggestions:
        return AlertDiscoveryResponse(candidates=[], warnings=[], partial=False)

    candidates: list[AlertDiscoveryCandidate] = []
    warnings: list[str] = []
    for item in suggestions[: request.alert.max_candidates]:
        candidate = _candidate_from_suggestion(item)
        if candidate is None:
            warnings.append("One result was omitted because required source-backed fields were missing.")
            continue
        candidates.append(candidate)
    if len(suggestions) > request.alert.max_candidates:
        warnings.append("Some candidates were omitted because the result cap was reached.")
    return AlertDiscoveryResponse(candidates=candidates, warnings=warnings, partial=bool(warnings))


def _alert_criteria_query(criteria) -> str:
    parts = [
        criteria.name,
        criteria.ruler_or_issuer,
        criteria.coin_type,
        criteria.mint,
        criteria.material,
        criteria.grade_or_condition,
        criteria.dealer_preference,
        criteria.keywords,
    ]
    if criteria.date_from is not None or criteria.date_to is not None:
        parts.append(f"date range {criteria.date_from or ''} to {criteria.date_to or ''}")
    if criteria.price_min is not None or criteria.price_max is not None:
        parts.append(f"price {criteria.price_min or 0} to {criteria.price_max or ''} {criteria.currency}")
    for source in criteria.source_filters:
        parts.append(f"site:{source}")
    return " ".join(part for part in parts if part).strip() or criteria.name


def _source_filter_hosts(source_filters: list[str]) -> set[str]:
    hosts: set[str] = set()
    for source in source_filters:
        parsed = urlparse(source if "://" in source else f"https://{source}")
        if parsed.hostname:
            hosts.add(parsed.hostname.lower().removeprefix("www."))
    return hosts


def _trusted_alert_fetch_hosts(source_filters: list[str]) -> set[str]:
    requested = _source_filter_hosts(source_filters)
    if not requested:
        return set(TRUSTED_ALERT_FETCH_HOSTS)
    return {
        trusted
        for trusted in TRUSTED_ALERT_FETCH_HOSTS
        if any(requested_host == trusted or requested_host.endswith(f".{trusted}") for requested_host in requested)
    }


def _url_matches_allowed_hosts(url: str, allowed_hosts: set[str]) -> bool:
    parsed = urlparse(url)
    if not parsed.hostname:
        return False
    host = parsed.hostname.lower().removeprefix("www.")
    return any(host == allowed or host.endswith(f".{allowed}") for allowed in allowed_hosts)


def _filter_allowed_fetch_urls(urls: list[str], allowed_fetch_hosts: set[str] | None) -> list[str]:
    if allowed_fetch_hosts is None:
        return urls
    allowed_hosts = {host.lower() for host in allowed_fetch_hosts if host}
    return [url for url in urls if _url_matches_allowed_hosts(url, allowed_hosts)]


def _last_message_content(result: dict) -> str:
    messages = result.get("messages", [])
    if not messages:
        return ""
    content = getattr(messages[-1], "content", "")
    return content if isinstance(content, str) else str(content)


def _extract_json_array(text: str) -> list[dict]:
    match = re.search(r"```json\s*\n(.*?)\n```", text, flags=re.DOTALL)
    payload = match.group(1).strip() if match else text.strip()
    try:
        parsed = json.loads(payload)
    except json.JSONDecodeError:
        return []
    if not isinstance(parsed, list):
        return []
    return [item for item in parsed if isinstance(item, dict)]


def _candidate_from_suggestion(item: dict) -> AlertDiscoveryCandidate | None:
    source_url = str(item.get("sourceUrl") or item.get("source_url") or "").strip()
    title = str(item.get("name") or item.get("title") or "").strip()
    if not source_url or not title:
        return None
    source_name = str(item.get("sourceName") or item.get("source_name") or "").strip()
    price_text = str(item.get("estPrice") or item.get("price") or "").strip()
    availability = str(item.get("availability") or item.get("status") or "").strip()
    observed_price, observed_currency = _parse_price(price_text)
    now = datetime.now(UTC).isoformat().replace("+00:00", "Z")
    provenance = [
        AlertDiscoveryProvenance(
            field="source_url",
            value=source_url,
            source_url=source_url,
            observed_at=now,
            confidence="high",
            verification_state="verified",
            notes="URL came from fetched search/listing data.",
        ),
        AlertDiscoveryProvenance(
            field="title",
            value=title,
            source_url=source_url,
            observed_at=now,
            confidence="high",
            verification_state="verified",
            notes="Title came from fetched search/listing data.",
        ),
    ]
    if price_text:
        provenance.append(
            AlertDiscoveryProvenance(
                field="observed_price",
                value=price_text,
                source_url=source_url,
                observed_at=now,
                confidence="medium" if observed_price is not None else "low",
                verification_state="verified" if observed_price is not None else "partial",
                notes="Price text came from fetched listing data.",
            )
        )
    if availability:
        provenance.append(
            AlertDiscoveryProvenance(
                field="availability",
                value=availability,
                source_url=source_url,
                observed_at=now,
                confidence="high",
                verification_state="verified",
                notes="Availability text came from fetched listing data.",
            )
        )
    return AlertDiscoveryCandidate(
        source_url=source_url,
        source_name=source_name,
        title=title,
        observed_price=observed_price,
        observed_currency=observed_currency,
        reason_for_match="Candidate title and source listing matched the saved alert criteria.",
        last_seen_at=now,
        provenance_status="verified",
        fields={
            "ruler": str(item.get("ruler") or ""),
            "denomination": str(item.get("denomination") or ""),
            "material": str(item.get("material") or ""),
            "grade_or_condition": str(item.get("grade") or ""),
            "availability": availability,
        },
        provenance=provenance,
    )


def _parse_price(price_text: str) -> tuple[float | None, str]:
    match = re.search(r"(US\$|\$|USD|EUR|GBP)\s*([\d,]+(?:\.\d{1,2})?)", price_text, re.IGNORECASE)
    if not match:
        return None, ""
    currency_token = match.group(1).upper()
    currency = "USD" if currency_token in {"$", "US$", "USD"} else currency_token
    try:
        return float(match.group(2).replace(",", "")), currency
    except ValueError:
        return None, currency
