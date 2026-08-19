"""Generate concise wishlist featured-coin rationale text."""

import logging

from langchain_core.messages import HumanMessage, SystemMessage

from app.llm.provider import get_chat_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import WishlistFeaturedSummaryRequest
from app.safety import with_safety

logger = logging.getLogger(__name__)

SUMMARY_PROMPT = with_safety("""You are writing a short "Featured Coin" note for a user's wishlist item.

Write 1-3 sentences, <=500 characters, plain text only.
Use only the supplied fields; do not invent facts, dates, prices, mint marks, or references.
If data is sparse, explain why the coin is notable in general numismatic terms tied to provided fields.
No markdown, no emojis, no bullets, no line breaks.
""")


def _trim_summary(text: str) -> str:
    """Normalize whitespace/newlines and enforce max response length."""
    cleaned = " ".join((text or "").replace("\r", " ").replace("\n", " ").split()).strip()
    if len(cleaned) <= 500:
        return cleaned
    return cleaned[:500].rstrip(" ,;:-")


async def generate_wishlist_featured_summary(request: WishlistFeaturedSummaryRequest) -> str:
    """Produce a concise rationale for a wishlist coin."""
    model = get_chat_model(request.llm)
    coin = request.coin
    owner = request.user_display_name.strip() or "Collector"

    context = (
        f"User display name: {owner}\n"
        f"Coin name: {coin.name}\n"
        f"Era: {coin.era}\n"
        f"Category: {coin.category}\n"
        f"Denomination: {coin.denomination}\n"
        f"Ruler: {coin.ruler}\n"
        f"Mint: {coin.mint}\n"
        f"Obverse analysis: {coin.obverse_analysis}\n"
        f"Reverse analysis: {coin.reverse_analysis}\n"
        f"AI analysis: {coin.ai_analysis}\n"
    )

    messages = [
        SystemMessage(content="You write concise, factual numismatic summaries."),
        HumanMessage(content=f"{SUMMARY_PROMPT}\n\nProvided data:\n{context}"),
    ]
    response = await ainvoke_with_retry(model, messages)
    content = response.content if isinstance(response.content, str) else str(response.content)
    return _trim_summary(content)
