"""Team 8: Coin Photography Guide — AI evaluates coin photos and suggests improvements.

Pipeline: Evaluation Agent → Tips Agent

- Evaluation: Vision model scores photo quality across multiple criteria
- Tips: Generates specific, actionable improvement suggestions
"""

import logging
from typing import Annotated, TypedDict

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.provider import get_chat_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import LLMConfig

logger = logging.getLogger(__name__)

EVALUATION_PROMPT = """You are an expert numismatic photographer evaluating coin photographs.

Assess the image(s) on these criteria (rate each 1-10):

1. **Lighting** — Even illumination, no harsh shadows or hot spots
2. **Focus/Sharpness** — Crisp details, especially on devices and legends
3. **Background** — Clean, neutral, non-distracting
4. **Angle/Perspective** — Proper flat-on view, minimal distortion
5. **Color Accuracy** — True metal colors, no color cast
6. **Scale/Framing** — Appropriate size in frame, good use of space
7. **Surface Detail Capture** — Luster, toning, and surface features visible

Provide an overall score (1-10) and note what's done well and what needs improvement.
Be specific about issues — e.g., "shadow on the left side obscures the legend" not just "lighting could be better".
Do not use emojis."""

TIPS_PROMPT = """You are a coin photography advisor. Based on the evaluation, provide specific,
actionable tips to improve the photos.

Structure your response as:
1. **Overall Assessment** — Brief summary with score
2. **What's Working** — Positive aspects to maintain
3. **Improvement Checklist** — Specific actionable items, ordered by impact
4. **Equipment Tips** — Any gear suggestions (lighting, background, etc.)
5. **Technique Tips** — Camera settings, positioning, workflow

Keep tips practical and achievable with consumer equipment (phone cameras are fine).
Do not use emojis. Format as clean markdown text."""


class PhotoGuideState(TypedDict):
    messages: Annotated[list, lambda a, b: a + b]
    raw_evaluation: str
    formatted_guide: str


def create_photo_guide_team(
    llm_config: LLMConfig,
    images: list[str] | None = None,
):
    """Create the photography guide team graph."""
    model = get_chat_model(llm_config)

    from app.teams.coin_analysis import _build_image_contents

    image_contents = _build_image_contents(images or [])

    async def evaluate_node(state: PhotoGuideState) -> dict:
        if not image_contents:
            return {
                "raw_evaluation": "",
                "messages": [AIMessage(content="No images provided. Upload coin photos to get photography tips.")],
            }

        human_content: list[dict] = [
            {"type": "text", "text": EVALUATION_PROMPT},
        ]
        human_content.extend(image_contents)

        messages = [
            SystemMessage(content="You are a professional numismatic photographer and photo evaluator."),
            HumanMessage(content=human_content),
        ]
        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)
        return {"raw_evaluation": content, "messages": []}

    async def tips_node(state: PhotoGuideState) -> dict:
        raw = state.get("raw_evaluation", "")
        if not raw:
            return {
                "formatted_guide": "",
                "messages": [AIMessage(content="Unable to evaluate photos. Please try with different images.")],
            }

        messages = [
            SystemMessage(content=TIPS_PROMPT),
            HumanMessage(content=f"Photo evaluation:\n\n{raw}"),
        ]
        response = await ainvoke_with_retry(model, messages)
        formatted = response.content if isinstance(response.content, str) else str(response.content)
        return {"formatted_guide": formatted, "messages": [AIMessage(content=formatted)]}

    graph = StateGraph(PhotoGuideState)
    graph.add_node("evaluate", evaluate_node)
    graph.add_node("tips", tips_node)
    graph.set_entry_point("evaluate")
    graph.add_edge("evaluate", "tips")
    graph.add_edge("tips", END)

    return graph.compile()
