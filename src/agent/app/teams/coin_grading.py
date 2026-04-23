"""Team 6: Coin Grading Assistant — AI-powered grade estimation from photos.

Pipeline: Grade Analysis Agent → Format Agent

- Grade Analysis: Vision model examines wear, surfaces, and strike quality
- Format: Structures the assessment into grade estimate with reasoning
"""

import logging
from typing import Annotated, TypedDict

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.provider import get_chat_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import CoinData, LLMConfig

logger = logging.getLogger(__name__)

GRADING_PROMPT = """You are a professional numismatic grading expert. Analyze the coin image(s)
and provide a grade estimate using the Sheldon scale.

Your assessment should cover:

1. **Grade Estimate** — Provide a grade (e.g., G-4, VG-8, F-12, VF-20, VF-25, VF-30, EF-40, EF-45,
   AU-50, AU-53, AU-55, AU-58, MS-60 through MS-70) with the numeric designation
2. **Confidence** — Your confidence level: High, Medium, or Low
3. **Wear Analysis** — Describe wear on high points, fields, and devices
4. **Strike Quality** — Sharpness of details, centering, any weakness
5. **Surface Analysis** — Scratches, corrosion, cleaning, environmental damage
6. **Eye Appeal** — Overall visual impression, luster, toning
7. **Comparison Notes** — How this coin compares to typical examples at this grade level

Use precise numismatic grading terminology. Be honest about limitations from photo quality.
Do not use emojis."""

FORMAT_PROMPT = """You are a formatting specialist for a coin grading application.
Structure the raw grading assessment into a clear, professional report.

Rules:
- Start with a prominent grade line: "**Estimated Grade: [GRADE]** (Confidence: [LEVEL])"
- Use clear section headers with markdown bold
- Keep all factual content intact
- End with a brief summary of key factors that determined the grade
- Do not add information not in the original assessment
- Do not use emojis

Output clean formatted text (not JSON)."""


class GradingState(TypedDict):
    messages: Annotated[list, lambda a, b: a + b]
    raw_assessment: str
    formatted_assessment: str


def create_coin_grading_team(
    llm_config: LLMConfig,
    coin: CoinData | None = None,
    images: list[str] | None = None,
):
    """Create the coin grading team graph."""
    model = get_chat_model(llm_config)

    from app.teams.coin_analysis import _build_coin_context, _build_image_contents

    coin_context = _build_coin_context(coin) if coin else ""
    image_contents = _build_image_contents(images or [])

    async def grade_node(state: GradingState) -> dict:
        if not image_contents:
            return {
                "raw_assessment": "",
                "messages": [AIMessage(content="No images provided. Please upload coin photos for grading.")],
            }

        human_content: list[dict] = [
            {"type": "text", "text": f"{GRADING_PROMPT}\n\n{coin_context}" if coin_context else GRADING_PROMPT},
        ]
        human_content.extend(image_contents)

        messages = [
            SystemMessage(content="You are a professional coin grading expert."),
            HumanMessage(content=human_content),
        ]
        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)
        return {"raw_assessment": content, "messages": []}

    async def format_node(state: GradingState) -> dict:
        raw = state.get("raw_assessment", "")
        if not raw:
            return {
                "formatted_assessment": "",
                "messages": [AIMessage(content="Unable to complete grading. Please try with clearer photos.")],
            }

        messages = [
            SystemMessage(content=FORMAT_PROMPT),
            HumanMessage(content=f"Raw grading assessment:\n\n{raw}"),
        ]
        response = await ainvoke_with_retry(model, messages)
        formatted = response.content if isinstance(response.content, str) else str(response.content)
        return {"formatted_assessment": formatted, "messages": [AIMessage(content=formatted)]}

    graph = StateGraph(GradingState)
    graph.add_node("grade", grade_node)
    graph.add_node("format", format_node)
    graph.set_entry_point("grade")
    graph.add_edge("grade", "format")
    graph.add_edge("format", END)

    return graph.compile()
