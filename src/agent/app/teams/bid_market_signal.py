"""Team 9b: Bid Market Signal — structured price-trend summary for a specific
tracked auction lot.

Reuses Team 9's search step (see price_trends.py) rather than duplicating
web-search logic; produces a small structured JSON signal instead of markdown,
since this is consumed programmatically by the Go bid recommendation flow
(services/auction_lot_service.go's MarketSignal method), not rendered as chat
prose the way Team 9's own output is.
"""

import json
import logging
from typing import TypedDict

from langchain_core.messages import HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph
from pydantic import ValidationError

from app.llm.provider import create_search_agent, get_chat_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import CoinData, LLMConfig
from app.models.responses import MarketSignalResponse
from app.safety import with_safety
from app.teams.coin_description import build_coin_description
from app.teams.json_extraction import extract_json_payload
from app.teams.price_trends import SEARCH_PROMPT, search_auction_results

logger = logging.getLogger(__name__)

MARKET_SIGNAL_PROMPT = with_safety("""You are a numismatic market analyst. Given web search results
for recent auction prices of a described coin type, produce a compact structured signal.

Respond with ONLY a fenced JSON object, no other text before or after it, matching this shape:
```json
{
  "trend_direction": "rising|stable|declining|unknown",
  "price_low": 0.0,
  "price_high": 0.0,
  "currency": "USD",
  "sample_size": 0,
  "rationale": "one or two sentence explanation grounded in the search results",
  "sources": ["https://...", "..."]
}
```

Rules:
- Use ONLY data actually present in the search results. Do not fabricate prices or sources.
- price_low/price_high should span the range of recent comparable hammer prices found.
- sample_size is the number of comparable results you actually used.
- If the search results contain no usable auction data, set trend_direction to "unknown",
  leave price_low and price_high as null, sample_size to 0, and explain why in rationale.
- Do not use emojis.""")


class BidMarketSignalState(TypedDict):
    coin_desc: str
    search_results: str
    signal_raw: str


class MarketSignalParseError(ValueError):
    """Raised when the LLM's market-signal output is not valid for MarketSignalResponse."""


def create_bid_market_signal_team(llm_config: LLMConfig, coin: CoinData):
    """Create the bid market-signal team graph for a specific tracked lot."""
    coin_desc = build_coin_description(coin)

    async def search_node(state: BidMarketSignalState) -> dict:
        if llm_config.provider == "ollama" and not llm_config.searxng_url:
            # No grounded search available for Ollama without SearXNG configured — short-circuit
            # rather than let the model hallucinate "results" (get_search_model does not
            # actually search for non-Anthropic providers).
            return {"search_results": ""}

        if llm_config.provider == "ollama":
            search_agent = create_search_agent(llm_config)
            messages = [
                SystemMessage(content=SEARCH_PROMPT),
                HumanMessage(content=f"Find recent auction results for: {coin_desc}"),
            ]
            result = await search_agent.ainvoke({"messages": messages})
            last_msg = result["messages"][-1]
            content = last_msg.content if isinstance(last_msg.content, str) else str(last_msg.content)
        else:
            content = await search_auction_results(llm_config, coin_desc)

        return {"search_results": content}

    async def extract_node(state: BidMarketSignalState) -> dict:
        results = state.get("search_results", "")
        if not results:
            return {"signal_raw": ""}

        model = get_chat_model(llm_config)
        messages = [
            SystemMessage(content=MARKET_SIGNAL_PROMPT),
            HumanMessage(content=f"Coin: {coin_desc}\n\nSearch results:\n\n{results}"),
        ]
        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)
        return {"signal_raw": content}

    graph = StateGraph(BidMarketSignalState)
    graph.add_node("search", search_node)
    graph.add_node("extract", extract_node)
    graph.set_entry_point("search")
    graph.add_edge("search", "extract")
    graph.add_edge("extract", END)

    return graph.compile()


def parse_market_signal(raw: str) -> MarketSignalResponse:
    """Parse the LLM's fenced JSON output into a MarketSignalResponse.

    Empty/blank input means "no usable search results" — a normal, degraded
    (but not erroring) outcome, distinct from a parse failure on real output.
    """
    if not raw.strip():
        return MarketSignalResponse(
            degraded=True,
            rationale="No usable market data found for this coin.",
        )

    json_str = extract_json_payload(raw)
    try:
        data = json.loads(json_str)
    except json.JSONDecodeError as e:
        raise MarketSignalParseError("invalid JSON in market signal output") from e

    try:
        return MarketSignalResponse.model_validate(data)
    except ValidationError as e:
        raise MarketSignalParseError("market signal JSON failed schema validation") from e
