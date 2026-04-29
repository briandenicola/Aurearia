"""Team 6: Availability Check — verify whether wishlist coin listings are still active.

Phase 1: Fetch each URL using the verify_url tool and collect raw indicators.
Phase 2: Use LLM to reason about ambiguous results and produce structured verdicts.
"""

import asyncio
import json
import logging
from typing import Annotated, TypedDict

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.provider import get_chat_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import LLMConfig
from app.safety import with_safety
from app.models.responses import AvailabilityVerdict
from app.tools.search import verify_url

logger = logging.getLogger(__name__)

MAX_ITEMS_PER_BATCH = 10

ANALYSIS_PROMPT = with_safety("""You are an expert at determining whether online coin listings are still available for purchase.

You will receive raw URL check data for one or more coin listings. Each entry includes:
- HTTP status code
- Whether the site is a trusted dealer
- Whether sold/unavailable indicators were found
- Whether buy/bid indicators were found

For each URL, determine whether the listing is:
- "available" — the item appears to be actively for sale or at auction
- "unavailable" — the item has been sold, the auction has ended, or the page is gone
- "unknown" — you cannot confidently determine the status

Also provide:
- A brief "reason" explaining your determination
- A "confidence" level: "high", "medium", or "low"

Consider these nuances:
- A 200 status with sold indicators clearly means unavailable
- A 200 status with buy/bid indicators clearly means available
- A 404 or 410 means the listing was removed (unavailable)
- A 200 with neither indicator could be a gallery page, a redirected page, or a listing
  in a format you can't parse — mark as unknown with low confidence
- Auction sites may show "realized price" for completed auctions (unavailable)
- Some dealer sites redirect sold items to search pages (unavailable)

Output a JSON array with one object per URL:
```json
[
  {
    "url": "https://example.com/coin/123",
    "coin_name": "Name of the coin",
    "status": "available|unavailable|unknown",
    "reason": "Brief explanation",
    "confidence": "high|medium|low"
  }
]
```

Output ONLY the JSON array wrapped in ```json and ``` markers. Do not use emojis.""")


class AvailabilityCheckState(TypedDict):
    """State for the availability check pipeline."""

    messages: Annotated[list, lambda a, b: a + b]
    items: list[dict]
    raw_checks: str
    verdicts: str


def create_availability_check_team(llm_config: LLMConfig):
    """Create the availability check pipeline.

    Args:
        llm_config: LLM provider configuration
    """

    async def check_urls_node(state: AvailabilityCheckState) -> dict:
        """Phase 1: Fetch each URL using verify_url and collect raw results."""
        items = state.get("items", [])
        if not items:
            return {"raw_checks": "", "messages": []}

        # Limit batch size
        items = items[:MAX_ITEMS_PER_BATCH]
        logger.debug("[availability] check_urls_node — checking %d URLs", len(items))

        # Run verify_url in parallel
        tasks = [verify_url.ainvoke(item["url"]) for item in items]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        lines = []
        for item, result in zip(items, results):
            coin_name = item.get("coin_name", "Unknown")
            if isinstance(result, Exception):
                lines.append(
                    f"--- {coin_name} ---\n"
                    f"URL: {item['url']}\n"
                    f"Error: {result}\n"
                )
            else:
                lines.append(
                    f"--- {coin_name} ---\n"
                    f"{result}\n"
                )

        raw = "\n".join(lines)
        logger.debug("[availability] check_urls_node — raw output: %d chars", len(raw))
        return {"raw_checks": raw, "messages": []}

    async def analyze_results_node(state: AvailabilityCheckState) -> dict:
        """Phase 2: Use LLM to analyze ambiguous results and produce verdicts."""
        raw_checks = state.get("raw_checks", "")
        items = state.get("items", [])

        if not raw_checks.strip():
            # No URLs to check
            return {"verdicts": "[]", "messages": [AIMessage(content="No listings to check.")]}

        model = get_chat_model(llm_config)

        messages = [
            SystemMessage(content=ANALYSIS_PROMPT),
            HumanMessage(
                content=f"Analyze these {len(items)} listing check results:\n\n{raw_checks}"
            ),
        ]

        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)

        logger.debug("[availability] analyze_results_node — response: %d chars", len(content))
        return {"verdicts": content, "messages": [AIMessage(content=content)]}

    graph = StateGraph(AvailabilityCheckState)
    graph.add_node("check_urls", check_urls_node)
    graph.add_node("analyze_results", analyze_results_node)

    graph.set_entry_point("check_urls")
    graph.add_edge("check_urls", "analyze_results")
    graph.add_edge("analyze_results", END)

    return graph.compile()


def parse_verdicts(raw_response: str) -> list[AvailabilityVerdict]:
    """Extract AvailabilityVerdict objects from the LLM's JSON response."""
    # Find JSON block
    start = raw_response.find("```json")
    if start != -1:
        start += len("```json")
        end = raw_response.find("```", start)
        if end != -1:
            json_str = raw_response[start:end].strip()
        else:
            json_str = raw_response[start:].strip()
    else:
        # Try parsing the whole thing as JSON
        json_str = raw_response.strip()

    try:
        data = json.loads(json_str)
        if isinstance(data, list):
            return [AvailabilityVerdict(**item) for item in data]
    except (json.JSONDecodeError, TypeError, ValueError) as e:
        logger.warning("Failed to parse availability verdicts: %s", e)

    return []
