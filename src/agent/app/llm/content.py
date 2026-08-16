"""Safe extraction of user-facing text from provider response content."""

from collections.abc import Mapping, Sequence
from typing import Any


def extract_text_content(content: Any) -> str:
    """Return only explicit text blocks, never transport metadata."""
    if isinstance(content, str):
        return content.strip()

    blocks: Sequence[Any]
    if isinstance(content, Mapping):
        blocks = [content]
    elif isinstance(content, Sequence) and not isinstance(content, (bytes, bytearray)):
        blocks = content
    else:
        return ""

    text_parts: list[str] = []
    for block in blocks:
        if isinstance(block, str):
            text = block
        elif isinstance(block, Mapping):
            if block.get("type") != "text":
                continue
            text = block.get("text")
        else:
            if getattr(block, "type", None) != "text":
                continue
            text = getattr(block, "text", None)
        if isinstance(text, str) and text.strip():
            text_parts.append(text.strip())
    return "\n\n".join(text_parts)
