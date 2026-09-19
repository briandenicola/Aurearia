"""Team 5: Auction Search — search NumisBids for auction lots.

Phase 1: Search NumisBids for lots matching the user's query.
Phase 2: Fetch top results for full lot details.
Phase 3: Format results into structured AuctionLotSuggestion JSON.
"""

import asyncio
import logging
from collections.abc import Mapping, Sequence
from datetime import datetime
from typing import Annotated, Any, TypedDict

import httpx
from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.content import extract_text_content
from app.llm.provider import get_chat_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import LLMConfig
from app.safety import with_safety
from app.teams.specialist_contracts import (
    CancellationCheck,
    ProviderMalformedError,
    ProviderRunner,
    SpecialistQuery,
    SpecialistResult,
    raise_if_cancelled,
    run_provider_search,
)
from app.tools.numisbids import scrape_numisbids_lot, search_numisbids

logger = logging.getLogger(__name__)

FORMAT_PROMPT = with_safety("""You are a formatting specialist for a coin auction tracking application.
You receive raw auction lot data scraped from NumisBids.
Structure each lot into this exact JSON schema:

```json
[
  {
    "title": "Lot title",
    "description": "Brief description",
    "category": "Roman|Greek|Byzantine|Modern|Other",
    "auctionHouse": "Name of auction house",
    "saleName": "Sale name if available",
    "estimate": "Estimated price e.g. $150.00",
    "currentBid": "Current bid if available",
    "imageUrl": "Image URL from the lot data",
    "url": "The exact NumisBids URL — never fabricate",
    "currency": "USD|EUR|GBP|CHF"
  }
]
```

Rules:
- Use ONLY data from the lot extracts. Do NOT invent fields.
- url MUST be copied exactly from the data. NEVER fabricate URLs.
- Infer category from the lot description (Greek, Roman, Byzantine, Modern, or Other)
- If a field is unknown, use an empty string
- Do not use emojis

Output ONLY the JSON array wrapped in ```json and ``` markers.""")

NO_RESULTS_PROMPT = with_safety(
    "You are an assistant in a coin collecting application. "
    "The user searched for auction lots on NumisBids but no results were found. "
    "Generate a brief, helpful response. Suggest different search terms or "
    "browsing numisbids.com directly. Keep it concise. Do not use emojis. "
    "Do not invent auction listings."
)


class AuctionSearchState(TypedDict):
    """State for the auction search pipeline."""

    messages: Annotated[list, lambda a, b: a + b]
    search_results: str
    fetched_lots: str
    user_message: str


async def _search_auction_lots(query: str) -> list[dict[str, Any]]:
    results = await search_numisbids.ainvoke({"query": query})
    if not isinstance(results, list):
        raise ProviderMalformedError
    if any(not isinstance(result, dict) for result in results):
        raise ProviderMalformedError
    if len(results) == 1 and isinstance(results[0], dict) and "error" in results[0]:
        raise httpx.TransportError("auction provider failed")
    return results


async def _fetch_auction_lots(
    search_results: Sequence[Mapping[str, Any]],
    limit: int,
) -> list[dict[str, Any]]:
    urls = [
        str(result.get("url") or "")
        for result in search_results
        if str(result.get("url") or "").startswith("https://")
    ][:limit]
    if not urls:
        return []
    results = await asyncio.gather(
        *(scrape_numisbids_lot.ainvoke({"url": url}) for url in urls),
        return_exceptions=True,
    )
    lots: list[dict[str, Any]] = []
    failures = 0
    for result in results:
        if isinstance(result, Exception):
            logger.warning("[auction_search] lot fetch failed")
            failures += 1
            continue
        if isinstance(result, dict) and "error" not in result:
            lots.append(result)
        else:
            failures += 1
    if failures and not lots:
        raise httpx.TransportError("auction provider fetch failed")
    return lots


async def _collect_auction_candidates(
    query: str,
    limit: int,
    cancellation_check: CancellationCheck | None = None,
) -> Sequence[Mapping[str, Any]]:
    search_results = await _search_auction_lots(query)
    await raise_if_cancelled(cancellation_check)
    lots = await _fetch_auction_lots(search_results, limit)
    await raise_if_cancelled(cancellation_check)
    return lots


async def run_auction_search(
    query: SpecialistQuery | Mapping[str, Any],
    *,
    provider_runners: Sequence[ProviderRunner] | None = None,
    observed_at: datetime | None = None,
    cancellation_check: CancellationCheck | None = None,
) -> SpecialistResult:
    """Run the canonical NumisBids workflow and return a strict specialist result."""
    if provider_runners is None:
        async def canonical_provider(search_query: str, limit: int) -> Sequence[Mapping[str, Any]]:
            return await _collect_auction_candidates(
                search_query,
                limit,
                cancellation_check=cancellation_check,
            )

        provider_runners = [ProviderRunner(provider="numisbids", run=canonical_provider)]
    return await run_provider_search(
        capability="auction_search",
        query=query,
        provider_runners=provider_runners,
        observed_at=observed_at,
        cancellation_check=cancellation_check,
    )


def create_auction_search_team(llm_config: LLMConfig):
    """Create the auction search pipeline.

    Args:
        llm_config: LLM provider configuration
    """

    async def search_node(state: AuctionSearchState) -> dict:
        """Phase 1: Search NumisBids for lots matching the query."""
        user_msg = state.get("user_message", "")
        logger.debug("[auction_search] search_node start — query: %.100s", user_msg)

        try:
            results = await _search_auction_lots(user_msg)
        except (ProviderMalformedError, httpx.TransportError):
            results = []
        if not results:
            logger.debug("[auction_search] search returned no results or error")
            return {"search_results": "", "messages": []}

        # Build a text summary of search results
        lines = []
        for lot in results:
            url = lot.get("url", "")
            title = lot.get("title", "Unknown")
            estimate = lot.get("estimate")
            currency = lot.get("currency", "USD")
            est_str = f"{estimate} {currency}" if estimate else "N/A"
            lines.append(f"- {title} | Estimate: {est_str} | {url}")

        summary = "\n".join(lines)
        logger.debug(
            "[auction_search] search returned %d results", len(results),
        )
        return {"search_results": summary, "messages": []}

    async def fetch_node(state: AuctionSearchState) -> dict:
        """Phase 2: Fetch top lot pages for full details."""
        search_results = state.get("search_results", "")

        if not search_results.strip():
            return {"fetched_lots": "", "messages": []}

        search_items = []
        for line in search_results.split("\n"):
            parts = line.rsplit("| ", 1)
            if len(parts) == 2:
                url = parts[1].strip()
                if url.startswith("https://"):
                    search_items.append({"url": url})

        logger.debug("[auction_search] fetch_node — found %d URLs to fetch", len(search_items))

        if not search_items:
            return {"fetched_lots": "", "messages": []}

        results = await _fetch_auction_lots(search_items, 5)
        fetched = [
            f"--- Lot: {result.get('url', '')} ---\n{result}"
            for result in results
        ]

        logger.debug("[auction_search] fetched %d lot details", len(fetched))
        return {"fetched_lots": "\n\n".join(fetched), "messages": []}

    async def format_node(state: AuctionSearchState) -> dict:
        """Phase 3: Format fetched lot data into AuctionLotSuggestion JSON."""
        fetched = state.get("fetched_lots", "")
        user_msg = state.get("user_message", "")
        search_results = state.get("search_results", "")
        model = get_chat_model(llm_config)
        logger.debug("[auction_search] format_node — fetched_lots=%d chars", len(fetched))

        if not fetched.strip():
            # No lots found — generate a helpful response via LLM
            messages = [
                SystemMessage(content=NO_RESULTS_PROMPT),
                HumanMessage(
                    content=f"The user asked: {user_msg}\n\n"
                    f"Search results summary:\n{search_results[:1000]}\n\n"
                    "No auction lot details could be extracted. Generate a helpful response."
                ),
            ]
            response = await ainvoke_with_retry(model, messages)
            content = extract_text_content(response.content)
            return {"messages": [AIMessage(content=content)]}

        # Format real lot data via LLM
        messages = [
            SystemMessage(content=FORMAT_PROMPT),
            HumanMessage(
                content=f"User searched for: {user_msg}\n\n"
                f"Extracted lot data:\n{fetched}"
            ),
        ]
        response = await ainvoke_with_retry(model, messages)
        formatted = extract_text_content(response.content)

        summary = (
            "I found some auction lots matching your search on NumisBids. "
            "Here are the listings I found."
        )
        return {"messages": [AIMessage(content=f"{summary}\n\n{formatted}")]}

    graph = StateGraph(AuctionSearchState)
    graph.add_node("search", search_node)
    graph.add_node("fetch", fetch_node)
    graph.add_node("format", format_node)

    graph.set_entry_point("search")
    graph.add_edge("search", "fetch")
    graph.add_edge("fetch", "format")
    graph.add_edge("format", END)

    return graph.compile()
