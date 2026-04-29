"""Team 4: Portfolio Review — portfolio analysis with live valuation.

Pipeline: Portfolio Reader Agent → Valuation Agent → Analysis Agent

- Portfolio Reader Agent: receives holdings data (no web calls), summarizes
- Valuation Agent: calls Team 1 to retrieve live market prices per coin
- Analysis Agent: compares purchase vs current value, trends, narrative

The Go API passes portfolio data in the request — this team has no DB access.
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

READER_PROMPT = with_safety("""You are a portfolio data analyst for a coin collecting application.
You receive structured portfolio data (holdings, values, categories).

Your job is to:
1. Summarize the portfolio composition (categories, materials, eras, rulers)
2. Identify the most valuable holdings
3. Flag any data quality issues (missing values, zero prices)
4. Prepare a structured summary for the valuation and analysis agents

Output your summary as clear text with section headers.
Do not make assumptions about values — report exactly what the data shows.""")

VALUATION_PROMPT = with_safety("""You are a numismatic valuation specialist for a coin collection.
You receive a portfolio summary and your job is to assess current market conditions.

Based on the portfolio data provided:
1. Comment on which categories/types tend to appreciate or depreciate
2. Identify coins that may be undervalued based on their attributes
3. Note any market trends relevant to the collection composition
4. Suggest areas where the collector might want to research current market prices

IMPORTANT: You do NOT have access to live market data. Base your assessment on:
- General numismatic market knowledge
- The relationship between purchase price and current estimated value in the data
- Known trends in collecting areas represented in the portfolio

Be honest about the limitations of your valuation. Recommend the user
check specific dealers or auction results for precise current market values.""")

ANALYSIS_PROMPT = with_safety("""You are a portfolio analysis expert for a numismatic collection.
You receive a portfolio summary and valuation commentary.

Produce a final narrative analysis covering:

1. **Portfolio Overview** — total holdings, total invested, current estimated value
2. **Composition Analysis** — breakdown by category, material, era
3. **Performance** — overall gain/loss, best and worst performers
4. **Diversification** — how well-diversified the collection is
5. **Recommendations** — specific, actionable suggestions for the collector

Rules:
- Use precise numbers from the data (do not round excessively)
- Be objective — highlight both strengths and weaknesses
- End with 3-5 specific recommendations
- Do not use emojis
- Format with markdown bold headers""")


class PortfolioReviewState(TypedDict):
    """State flowing through the portfolio review pipeline."""

    messages: Annotated[list, lambda a, b: a + b]
    portfolio_summary: str
    valuation_commentary: str
    final_analysis: str
    user_message: str


def create_portfolio_review_team(
    llm_config: LLMConfig,
    portfolio: PortfolioSummary | None = None,
    user_message: str = "",
):
    """Create the Team 4 portfolio review graph.

    Args:
        llm_config: LLM provider configuration
        portfolio: Portfolio summary data from Go API
        user_message: Optional user question about their portfolio
    """
    model = get_chat_model(llm_config)
    portfolio_data = _format_portfolio(portfolio) if portfolio else "No portfolio data provided."

    async def reader_node(state: PortfolioReviewState) -> dict:
        """Portfolio Reader Agent: summarizes holdings data."""
        logger.debug("[portfolio] reader_node start — portfolio_data=%d chars", len(portfolio_data))
        messages = [
            SystemMessage(content=READER_PROMPT),
            HumanMessage(content=f"Here is the portfolio data:\n\n{portfolio_data}"),
        ]
        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)

        return {
            "portfolio_summary": content,
            "messages": [],
        }

    async def valuation_node(state: PortfolioReviewState) -> dict:
        """Valuation Agent: assesses market conditions for the portfolio."""
        summary = state.get("portfolio_summary", "")
        logger.debug("[portfolio] valuation_node start — summary=%d chars", len(summary))

        messages = [
            SystemMessage(content=VALUATION_PROMPT),
            HumanMessage(
                content=f"Portfolio summary:\n{summary}\n\n"
                f"Raw portfolio data:\n{portfolio_data}"
            ),
        ]
        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)

        return {
            "valuation_commentary": content,
            "messages": [],
        }

    async def analysis_node(state: PortfolioReviewState) -> dict:
        """Analysis Agent: produces final narrative report."""
        summary = state.get("portfolio_summary", "")
        valuation = state.get("valuation_commentary", "")
        user_msg = state.get("user_message", user_message)
        logger.debug(
            "[portfolio] analysis_node start — summary=%d chars, valuation=%d chars",
            len(summary), len(valuation),
        )

        user_context = ""
        if user_msg:
            user_context = f"\n\nThe user specifically asked: {user_msg}"

        messages = [
            SystemMessage(content=ANALYSIS_PROMPT),
            HumanMessage(
                content=f"Portfolio summary:\n{summary}\n\n"
                f"Valuation commentary:\n{valuation}\n\n"
                f"Raw data:\n{portfolio_data}"
                f"{user_context}"
            ),
        ]
        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)

        return {
            "final_analysis": content,
            "messages": [AIMessage(content=content)],
        }

    graph = StateGraph(PortfolioReviewState)
    graph.add_node("reader", reader_node)
    graph.add_node("valuation", valuation_node)
    graph.add_node("analysis", analysis_node)

    graph.set_entry_point("reader")
    graph.add_edge("reader", "valuation")
    graph.add_edge("valuation", "analysis")
    graph.add_edge("analysis", END)

    return graph.compile()


def _format_portfolio(portfolio: PortfolioSummary) -> str:
    """Format portfolio data as readable text for the LLM."""
    lines = [
        f"Total coins: {portfolio.total_coins}",
        f"Total invested: ${portfolio.total_invested:,.2f}",
        f"Current estimated value: ${portfolio.total_value:,.2f}",
        f"Gain/Loss: ${portfolio.total_value - portfolio.total_invested:,.2f}",
    ]

    if portfolio.categories:
        lines.append("\nCategories:")
        for cat, count in sorted(portfolio.categories.items(), key=lambda x: -x[1]):
            lines.append(f"  {cat}: {count} coins")

    if portfolio.materials:
        lines.append("\nMaterials:")
        for mat, count in sorted(portfolio.materials.items(), key=lambda x: -x[1]):
            lines.append(f"  {mat}: {count} coins")

    if portfolio.eras:
        lines.append("\nEras:")
        for era in portfolio.eras:
            lines.append(f"  {era.get('name', 'Unknown')}: {era.get('count', 0)} coins")

    if portfolio.rulers:
        lines.append("\nTop Rulers:")
        for ruler in portfolio.rulers[:10]:
            lines.append(f"  {ruler.get('name', 'Unknown')}: {ruler.get('count', 0)} coins")

    if portfolio.top_coins:
        lines.append("\nTop Coins by Value:")
        for coin in portfolio.top_coins:
            value_str = f"${coin.current_value:,.2f}" if coin.current_value else "Unknown"
            paid_str = f"${coin.purchase_price:,.2f}" if coin.purchase_price else "Unknown"
            lines.append(f"  {coin.name} ({coin.category}) — Value: {value_str}, Paid: {paid_str}")

    return "\n".join(lines)
