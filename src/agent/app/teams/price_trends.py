"""Team 9: Price Trend Analysis — searches auction results and analyzes market direction.

Pipeline: Search Agent → Analysis Agent

- Search: Web search for recent auction results of similar coins
- Analysis: Analyzes price trends, calculates direction, formats results
"""

import logging
from typing import Annotated, TypedDict

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.provider import get_chat_model, get_search_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import LLMConfig
from app.safety import with_safety

logger = logging.getLogger(__name__)

SEARCH_PROMPT = with_safety("""You are a numismatic market researcher. Search for recent auction results
for the described coin type.

Search for:
- Recent auction hammer prices (last 1-2 years)
- Results from major auction houses (Heritage, CNG, Roma, Nomos, etc.)
- Results from NumisBids and other aggregators
- Different grades/conditions to show price range

Find at least 5-10 recent results if possible. For each result note:
- Auction house and date
- Grade/condition
- Hammer price (including buyer's premium if noted)
- Any notable features

Do not invent results. Only report data you actually find.""")

ANALYSIS_PROMPT = with_safety("""You are a numismatic market analyst. Given the search results for auction prices,
provide a comprehensive price trend analysis.

Structure your response:

1. **Market Overview** — Current market status for this coin type
2. **Recent Results** — Table of recent sales with date, house, grade, price
3. **Price Ranges** — By grade level (VF, EF, AU, MS, etc.)
4. **Trend Direction** — Rising, Stable, or Declining, with reasoning
5. **Market Factors** — What's driving the current trend
6. **Collector Advisory** — Is now a good time to buy, sell, or hold?

Use actual data from the search results. Do not fabricate prices.
Do not use emojis. Format as clean markdown text.""")


class PriceTrendState(TypedDict):
    messages: Annotated[list, lambda a, b: a + b]
    search_results: str
    analysis: str
    user_message: str


def create_price_trend_team(
    llm_config: LLMConfig,
    user_message: str = "",
):
    """Create the price trend analysis team graph."""
    chat_model = get_chat_model(llm_config)

    async def search_node(state: PriceTrendState) -> dict:
        search_model = get_search_model(llm_config)
        messages = [
            SystemMessage(content=SEARCH_PROMPT),
            HumanMessage(content=f"Find recent auction results for: {user_message}"),
        ]
        response = await ainvoke_with_retry(search_model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)
        return {"search_results": content, "messages": []}

    async def analysis_node(state: PriceTrendState) -> dict:
        results = state.get("search_results", "")
        if not results:
            return {
                "analysis": "",
                "messages": [AIMessage(content="Unable to find auction results for this coin type.")],
            }

        messages = [
            SystemMessage(content=ANALYSIS_PROMPT),
            HumanMessage(content=f"Coin query: {user_message}\n\nSearch results:\n\n{results}"),
        ]
        response = await ainvoke_with_retry(chat_model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)
        return {"analysis": content, "messages": [AIMessage(content=content)]}

    graph = StateGraph(PriceTrendState)
    graph.add_node("search", search_node)
    graph.add_node("analyze", analysis_node)
    graph.set_entry_point("search")
    graph.add_edge("search", "analyze")
    graph.add_edge("analyze", END)

    return graph.compile()
