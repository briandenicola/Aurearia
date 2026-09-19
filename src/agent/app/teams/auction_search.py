"""Team 5: Auction Search across administrator-configured auction sources.

NumisBids is searched and scraped directly (no model in the loop); any other
configured auction host goes through the web-search pipeline below.

Phase 1: Search configured auction sites for lots matching the user's query.
Phase 2: Fetch top results for full lot details.
Phase 3: Format results into structured AuctionLotSuggestion JSON.
"""

import asyncio
import json
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
from app.teams.coin_search import (
    _extract_json_array_strict,
    _fetch_dealer_pages,
    _search_dealer_pages,
)
from app.teams.specialist_contracts import (
    CancellationCheck,
    ProviderMalformedError,
    ProviderRunner,
    ProviderUnavailableError,
    SpecialistQuery,
    SpecialistResult,
    raise_if_cancelled,
    run_provider_search,
)
from app.tools.numisbids import scrape_numisbids_lot, search_numisbids

logger = logging.getLogger(__name__)

FORMAT_PROMPT = with_safety("""You are a formatting specialist for a coin auction tracking application.
You receive raw auction lot data fetched from administrator-configured auction sites.
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
    "url": "The exact source URL — never fabricate",
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
    "The user searched configured auction sources but no results were found. "
    "Generate a brief, helpful response. Suggest different search terms or "
    "browsing a configured auction source directly. Keep it concise. Do not use emojis. "
    "Do not invent auction listings."
)


class AuctionSearchState(TypedDict):
    """State for the auction search pipeline."""

    messages: Annotated[list, lambda a, b: a + b]
    search_results: str
    fetched_lots: str
    user_message: str


def _source_prompt(source_hosts: set[str]) -> str:
    sources = ", ".join(sorted(source_hosts))
    return with_safety(
        "Search only these administrator-configured auction hosts: "
        f"{sources}. Find current or upcoming coin auction lots matching the request. "
        "Ignore every other host and copy result URLs exactly."
    )


async def _format_auction_candidates(
    llm_config: LLMConfig,
    query: str,
    fetched_lots: str,
) -> list[dict[str, Any]]:
    model = get_chat_model(llm_config)
    response = await ainvoke_with_retry(
        model,
        [
            SystemMessage(content=FORMAT_PROMPT),
            HumanMessage(content=f"User searched for: {query}\n\nExtracted lot data:\n{fetched_lots}"),
        ],
    )
    candidates = _extract_json_array_strict(extract_text_content(response.content))
    # Lots may come from a sale/results page rather than a fetched lot page;
    # keep any lot whose URL appears in the fetched data.
    return [
        candidate
        for candidate in candidates
        if (url := str(candidate.get("url") or "").strip()) and url in fetched_lots
    ]


async def _collect_auction_candidates(
    llm_config: LLMConfig,
    query: str,
    limit: int,
    source_hosts: set[str],
    cancellation_check: CancellationCheck | None = None,
) -> Sequence[Mapping[str, Any]]:
    search_results = await _search_dealer_pages(llm_config, query, _source_prompt(source_hosts))
    await raise_if_cancelled(cancellation_check)
    fetched = await _fetch_dealer_pages(
        search_results,
        source_hosts,
        specialist_boundary=True,
    )
    await raise_if_cancelled(cancellation_check)
    if not fetched:
        return []
    lots = await _format_auction_candidates(llm_config, query, fetched)
    return lots[:limit]


def _is_numisbids_host(host: str) -> bool:
    return host == "numisbids.com" or host.endswith(".numisbids.com")


async def _collect_numisbids_lots(
    query: str,
    limit: int,
    cancellation_check: CancellationCheck | None = None,
) -> list[dict[str, Any]]:
    """Search NumisBids and scrape the top lot pages without model involvement."""
    results = await search_numisbids.ainvoke({"query": query})
    if not isinstance(results, list):
        raise ProviderMalformedError
    if any(isinstance(result, dict) and "error" in result for result in results):
        raise httpx.TransportError("NumisBids search failed")
    summaries = [result for result in results if isinstance(result, dict) and result.get("url")][:limit]
    await raise_if_cancelled(cancellation_check)
    pages = await asyncio.gather(
        *(scrape_numisbids_lot.ainvoke({"url": summary["url"]}) for summary in summaries),
        return_exceptions=True,
    )
    await raise_if_cancelled(cancellation_check)
    lots: list[dict[str, Any]] = []
    for summary, page in zip(summaries, pages, strict=True):
        if isinstance(page, dict) and "error" not in page and page.get("title"):
            lots.append({**page, "url": summary["url"]})
        else:
            # Lot page unavailable: fall back to the search-result summary.
            lots.append(summary)
    return lots


async def run_auction_search(
    query: SpecialistQuery | Mapping[str, Any],
    *,
    provider_runners: Sequence[ProviderRunner] | None = None,
    observed_at: datetime | None = None,
    cancellation_check: CancellationCheck | None = None,
    llm_config: LLMConfig | None = None,
    source_hosts: set[str] | None = None,
) -> SpecialistResult:
    """Search configured auction sources and return a strict specialist result."""
    if provider_runners is None:
        if llm_config is None or not source_hosts:
            raise ProviderUnavailableError
        web_search_hosts = {host for host in source_hosts if not _is_numisbids_host(host)}

        async def numisbids_provider(search_query: str, limit: int) -> Sequence[Mapping[str, Any]]:
            return await _collect_numisbids_lots(search_query, limit, cancellation_check)

        async def canonical_provider(search_query: str, limit: int) -> Sequence[Mapping[str, Any]]:
            return await _collect_auction_candidates(
                llm_config,
                search_query,
                limit,
                web_search_hosts,
                cancellation_check=cancellation_check,
            )

        provider_runners = []
        if len(web_search_hosts) < len(source_hosts):
            provider_runners.append(ProviderRunner(provider="numisbids", run=numisbids_provider))
        if web_search_hosts:
            provider_runners.append(
                ProviderRunner(
                    provider="configured_auction_search",
                    run=canonical_provider,
                    allowed_hosts=frozenset(web_search_hosts),
                )
            )
    return await run_provider_search(
        capability="auction_search",
        query=query,
        provider_runners=provider_runners,
        observed_at=observed_at,
        cancellation_check=cancellation_check,
    )


def create_auction_search_team(llm_config: LLMConfig, source_hosts: set[str]):
    """Create the auction search pipeline.

    Args:
        llm_config: LLM provider configuration
    """

    async def search_node(state: AuctionSearchState) -> dict:
        """Phase 1: Search NumisBids for lots matching the query."""
        user_msg = state.get("user_message", "")
        logger.debug("[auction_search] search_node start — query: %.100s", user_msg)

        try:
            results = await _search_dealer_pages(llm_config, user_msg, _source_prompt(source_hosts))
        except (ProviderMalformedError, httpx.TransportError, ValueError):
            results = ""
        if not results.strip():
            logger.debug("[auction_search] search returned no results or error")
            return {"search_results": "", "messages": []}

        return {"search_results": results, "messages": []}

    async def fetch_node(state: AuctionSearchState) -> dict:
        """Phase 2: Fetch top lot pages for full details."""
        search_results = state.get("search_results", "")

        if not search_results.strip():
            return {"fetched_lots": "", "messages": []}

        fetched = await _fetch_dealer_pages(
            search_results,
            source_hosts,
            specialist_boundary=True,
        )
        return {"fetched_lots": fetched, "messages": []}

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

        candidates = await _format_auction_candidates(llm_config, user_msg, fetched)
        formatted = f"```json\n{json.dumps(candidates, ensure_ascii=False, indent=2)}\n```"

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
