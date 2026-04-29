"""Team 3: Coin Analysis — vision-based coin image analysis.

Pipeline: Analysis Agent → Formatter Agent

- Analysis Agent: uses vision model to analyze coin images (obverse/reverse)
- Formatter Agent: structures analysis into consistent narrative format

This team does NOT perform web searches — it relies entirely on the
vision model's training data and the provided images.
"""

import base64
import logging
from typing import Annotated, TypedDict

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.provider import get_chat_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import CoinData, LLMConfig
from app.safety import with_safety

logger = logging.getLogger(__name__)

DEFAULT_ANALYSIS_PROMPT = with_safety("""You are a numismatic expert analyzing a coin image.
Provide a detailed analysis covering:

1. **Identification** — ruler, denomination, mint, approximate date
2. **Design Description** — portrait or design elements, iconography
3. **Inscriptions** — all visible text, legends, mint marks
4. **Condition Assessment** — wear patterns, surface quality, grade estimate
5. **Die Analysis** — die orientation, any die breaks, centering
6. **Authenticity Indicators** — surface texture, style consistency, weight/flan observations

Use precise numismatic terminology. Do not invent details you cannot observe in the image.""")

FORMAT_PROMPT = with_safety("""You are a formatting specialist for a coin collecting application.
You receive a raw coin analysis from a vision model. Your job is to clean up
the formatting and ensure consistency.

Rules:
- Keep ALL factual content from the analysis intact
- Use clear section headers with markdown bold (**Header**)
- Remove any hedging about image quality unless critical to the assessment
- End with a brief 1-2 sentence summary
- Do not add information not present in the original analysis
- Do not use emojis

Output the formatted analysis as clean text (not JSON).""")


class CoinAnalysisState(TypedDict):
    """State flowing through the coin analysis pipeline."""

    messages: Annotated[list, lambda a, b: a + b]
    raw_analysis: str
    formatted_analysis: str


def create_coin_analysis_team(
    llm_config: LLMConfig,
    coin: CoinData | None = None,
    images: list[str] | None = None,
    side: str = "",
    custom_prompt: str = "",
):
    """Create the Team 3 coin analysis graph.

    Args:
        llm_config: LLM provider configuration
        coin: Optional coin data for context
        images: Base64-encoded image data
        side: "obverse", "reverse", or "" for general analysis
        custom_prompt: Custom analysis prompt from admin settings
    """
    model = get_chat_model(llm_config)

    # Build coin context string
    coin_context = _build_coin_context(coin) if coin else ""

    # Build image content blocks for the vision model
    image_contents = _build_image_contents(images or [])

    # Select the analysis prompt
    analysis_prompt = custom_prompt or DEFAULT_ANALYSIS_PROMPT
    if side:
        analysis_prompt += f"\n\nFocus your analysis on the {side} of the coin."

    async def analysis_node(state: CoinAnalysisState) -> dict:
        """Analysis Agent: vision model analyzes coin images."""
        # Use closure values — state fields may be empty/unset
        img_contents = image_contents
        ctx = coin_context
        logger.debug("[coin_analysis] analysis_node start — %d images, side=%s", len(img_contents), side or "general")

        if not img_contents:
            return {
                "raw_analysis": "",
                "messages": [AIMessage(content="No images were provided for analysis. "
                                               "Please upload one or more coin images to get started.")],
            }

        # Build the message with text + images
        human_content: list[dict] = [
            {"type": "text", "text": f"{analysis_prompt}\n\n{ctx}" if ctx else analysis_prompt},
        ]
        human_content.extend(img_contents)

        messages = [
            SystemMessage(content="You are an expert numismatist analyzing coin images."),
            HumanMessage(content=human_content),
        ]
        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)

        return {
            "raw_analysis": content,
            "messages": [],
        }

    async def format_node(state: CoinAnalysisState) -> dict:
        """Formatter Agent: cleans up and structures the analysis."""
        raw = state.get("raw_analysis", "")

        if not raw:
            return {
                "formatted_analysis": "",
                "messages": [AIMessage(content="Unable to complete the analysis. "
                                               "Please try again with clearer, well-lit images of the coin.")],
            }

        messages = [
            SystemMessage(content=FORMAT_PROMPT),
            HumanMessage(content=f"Raw analysis to format:\n\n{raw}"),
        ]
        response = await ainvoke_with_retry(model, messages)
        formatted = response.content if isinstance(response.content, str) else str(response.content)

        return {
            "formatted_analysis": formatted,
            "messages": [AIMessage(content=formatted)],
        }

    graph = StateGraph(CoinAnalysisState)
    graph.add_node("analyze", analysis_node)
    graph.add_node("format", format_node)

    graph.set_entry_point("analyze")
    graph.add_edge("analyze", "format")
    graph.add_edge("format", END)

    return graph.compile()


def _build_coin_context(coin: CoinData) -> str:
    """Build a textual coin context string from CoinData."""
    parts = []
    if coin.name:
        parts.append(f"Name: {coin.name}")
    if coin.category:
        parts.append(f"Category: {coin.category}")
    if coin.denomination:
        parts.append(f"Denomination: {coin.denomination}")
    if coin.ruler:
        parts.append(f"Ruler: {coin.ruler}")
    if coin.era:
        parts.append(f"Era: {coin.era}")
    if coin.material:
        parts.append(f"Material: {coin.material}")
    if coin.grade:
        parts.append(f"Grade: {coin.grade}")

    if not parts:
        return ""
    return "Known coin details:\n" + "\n".join(parts)


def _build_image_contents(images: list[str]) -> list[dict]:
    """Build LangChain image content blocks from base64 strings.

    Supports both Anthropic (image_url with base64 data URI) and
    Ollama (same format, handled by langchain-ollama).
    """
    contents = []
    for img_b64 in images:
        if not img_b64:
            continue
        # Detect MIME type from base64 header or default to JPEG
        mime = "image/jpeg"
        if img_b64.startswith("data:"):
            # Already a data URI
            data_uri = img_b64
        else:
            # Sniff MIME type from first few bytes (avoid decoding entire image)
            try:
                preview = base64.b64decode(img_b64[:200], validate=False)
                if preview[:4] == b"\x89PNG":
                    mime = "image/png"
                elif preview[:2] == b"\xff\xd8":
                    mime = "image/jpeg"
                elif preview[:4] == b"RIFF":
                    mime = "image/webp"
            except Exception:
                logger.warning("Skipping invalid base64 image data")
                continue
            data_uri = f"data:{mime};base64,{img_b64}"

        contents.append({
            "type": "image_url",
            "image_url": {"url": data_uri},
        })
    return contents
