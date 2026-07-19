"""Team 10: Similar Lot Finder — searches for auction lots similar to user's coins.

Pipeline: Search Agent → Relevance Scorer

- Search: Web search for similar lots across active auctions
- Scorer: Ranks results by relevance and formats for display
"""

import logging
from typing import Annotated, TypedDict

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.provider import get_chat_model, get_search_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import CoinData, LLMConfig
from app.safety import with_safety
from app.teams.coin_description import build_coin_description

logger = logging.getLogger(__name__)

SEARCH_PROMPT = with_safety("""You are a numismatic auction researcher. Search for active auction lots
that are similar to the described coin.

Search across:
- NumisBids.com active lots
- Heritage Auctions current listings
- CNG, Roma, Nomos upcoming sales
- Any other active numismatic auction platforms

For each lot found, note:
- Lot title and description
- Auction house and sale name
- Estimate or current bid
- Lot URL if available
- Key similarities to the user's coin

Find 5-10 similar lots. Focus on matching: ruler/type, denomination, era, and category.""")

SCORING_PROMPT = with_safety("""You are a numismatic similarity assessor. Given search results for auction lots
similar to a specific coin, rank and format them.

Structure your response:

For each lot (ranked by relevance):
1. **Lot Title** — Brief description
2. **Auction House** — Name and sale
3. **Estimate/Bid** — Current pricing
4. **Similarity** — Why this lot is similar (specific attributes matching)
5. **URL** — Link if available

Start with a brief note about the search scope, then list results.
Only include lots that genuinely match — do not pad with irrelevant results.
Do not use emojis. Format as clean markdown text.""")


class SimilarLotState(TypedDict):
    messages: Annotated[list, lambda a, b: a + b]
    search_results: str
    scored_results: str
    user_message: str


def create_similar_lot_team(
    llm_config: LLMConfig,
    coin: CoinData | None = None,
    user_message: str = "",
):
    """Create the similar lot finder team graph."""
    chat_model = get_chat_model(llm_config)

    coin_desc = build_coin_description(coin) if coin else user_message

    async def search_node(state: SimilarLotState) -> dict:
        search_model = get_search_model(llm_config)
        messages = [
            SystemMessage(content=SEARCH_PROMPT),
            HumanMessage(content=f"Find active auction lots similar to:\n\n{coin_desc}"),
        ]
        response = await ainvoke_with_retry(search_model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)
        return {"search_results": content, "messages": []}

    async def score_node(state: SimilarLotState) -> dict:
        results = state.get("search_results", "")
        if not results:
            return {
                "scored_results": "",
                "messages": [AIMessage(content="No similar lots found in active auctions.")],
            }

        messages = [
            SystemMessage(content=SCORING_PROMPT),
            HumanMessage(content=f"Reference coin:\n{coin_desc}\n\nSearch results:\n\n{results}"),
        ]
        response = await ainvoke_with_retry(chat_model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)
        return {"scored_results": content, "messages": [AIMessage(content=content)]}

    graph = StateGraph(SimilarLotState)
    graph.add_node("search", search_node)
    graph.add_node("score", score_node)
    graph.set_entry_point("search")
    graph.add_edge("search", "score")
    graph.add_edge("score", END)

    return graph.compile()
