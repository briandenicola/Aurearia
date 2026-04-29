"""Team 7: Collection Gap Analysis — AI reviews collection and suggests acquisitions.

Pipeline: Analysis Agent → Suggestion Agent

- Analysis: Reviews collection distribution and identifies gaps
- Suggestion: Researches and suggests specific acquisitions to fill gaps
"""

import logging
from typing import Annotated, TypedDict

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.provider import get_chat_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import LLMConfig, PortfolioSummary
from app.safety import with_safety

logger = logging.getLogger(__name__)

ANALYSIS_PROMPT = with_safety("""You are a numismatic collection advisor analyzing a coin collection for gaps
and completeness.

Given the collection summary, analyze:

1. **Distribution Analysis** — How coins are distributed across eras, categories, rulers, materials
2. **Identified Gaps** — What's missing for a more complete or balanced collection
3. **Strengths** — Areas where the collection is particularly strong
4. **Completeness Score** — Rate overall completeness from 1-10 with reasoning

Consider common collecting strategies:
- By ruler (completing a dynasty)
- By denomination type
- By era/period
- By mint
- By material type
- By region

Do not use emojis.""")

SUGGESTION_PROMPT = with_safety("""You are a numismatic acquisition advisor. Based on the gap analysis provided,
suggest specific coins to acquire.

For each suggestion, provide:
- **Coin Name/Type** — specific identification
- **Why** — how it fills a gap in the collection
- **Estimated Price Range** — approximate market value
- **Priority** — High, Medium, or Low
- **Where to Look** — auction houses, dealers, or marketplaces

Provide 5-8 concrete suggestions ranked by priority.
Do not use emojis. Format as clean text with markdown headers.""")


class GapAnalysisState(TypedDict):
    messages: Annotated[list, lambda a, b: a + b]
    collection_summary: str
    gap_analysis: str
    suggestions: str
    user_message: str


def create_gap_analysis_team(
    llm_config: LLMConfig,
    portfolio: PortfolioSummary | None = None,
    user_message: str = "",
):
    """Create the gap analysis team graph."""
    model = get_chat_model(llm_config)

    summary_text = _build_collection_summary(portfolio) if portfolio else "No collection data available."

    async def analysis_node(state: GapAnalysisState) -> dict:
        messages = [
            SystemMessage(content=ANALYSIS_PROMPT),
            HumanMessage(content=f"Collection Summary:\n\n{summary_text}\n\nUser request: {user_message}"),
        ]
        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)
        return {"gap_analysis": content, "messages": []}

    async def suggestion_node(state: GapAnalysisState) -> dict:
        gap = state.get("gap_analysis", "")
        if not gap:
            return {
                "suggestions": "",
                "messages": [AIMessage(content="Unable to analyze gaps without collection data.")],
            }

        messages = [
            SystemMessage(content=SUGGESTION_PROMPT),
            HumanMessage(content=f"Gap Analysis:\n\n{gap}\n\nCollection Summary:\n\n{summary_text}"),
        ]
        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)

        combined = f"{gap}\n\n---\n\n**Suggested Acquisitions**\n\n{content}"
        return {"suggestions": content, "messages": [AIMessage(content=combined)]}

    graph = StateGraph(GapAnalysisState)
    graph.add_node("analyze", analysis_node)
    graph.add_node("suggest", suggestion_node)
    graph.set_entry_point("analyze")
    graph.add_edge("analyze", "suggest")
    graph.add_edge("suggest", END)

    return graph.compile()


def _build_collection_summary(portfolio: PortfolioSummary) -> str:
    parts = [f"Total coins: {portfolio.total_coins}"]
    if portfolio.total_value:
        parts.append(f"Total value: ${portfolio.total_value:,.2f}")
    if portfolio.total_invested:
        parts.append(f"Total invested: ${portfolio.total_invested:,.2f}")

    if portfolio.categories:
        cats = ", ".join(f"{k}: {v}" for k, v in portfolio.categories.items())
        parts.append(f"By category: {cats}")
    if portfolio.eras:
        eras = ", ".join(f"{e.get('era', '?')}: {e.get('count', 0)}" for e in portfolio.eras)
        parts.append(f"By era: {eras}")
    if portfolio.materials:
        mats = ", ".join(f"{k}: {v}" for k, v in portfolio.materials.items())
        parts.append(f"By material: {mats}")
    if portfolio.rulers:
        rulers = ", ".join(f"{r.get('ruler', '?')}: {r.get('count', 0)}" for r in portfolio.rulers[:20])
        parts.append(f"Top rulers: {rulers}")

    if portfolio.top_coins:
        coin_list = []
        for c in portfolio.top_coins[:50]:
            coin_list.append(f"- {c.name} ({c.category}, {c.era}, {c.grade or 'ungraded'})")
        parts.append("Coins:\n" + "\n".join(coin_list))

    return "\n".join(parts)
