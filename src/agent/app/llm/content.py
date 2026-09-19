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


def _field(block: Any, name: str) -> Any:
    if isinstance(block, Mapping):
        return block.get(name)
    return getattr(block, name, None)


def extract_search_result_urls(content: Any) -> list[str]:
    """Return URLs from provider web-search result blocks and text citations.

    Anthropic's server-side web_search returns result URLs in
    ``web_search_tool_result`` blocks and ``citations`` on text blocks, not in
    the text itself, so ``extract_text_content`` alone loses them.
    """
    if isinstance(content, Mapping):
        blocks: Sequence[Any] = [content]
    elif isinstance(content, Sequence) and not isinstance(content, (str, bytes, bytearray)):
        blocks = content
    else:
        return []

    urls: list[str] = []

    def add(url: Any) -> None:
        if isinstance(url, str) and url.startswith(("https://", "http://")) and url not in urls:
            urls.append(url)

    for block in blocks:
        block_type = _field(block, "type")
        if block_type == "web_search_tool_result":
            results = _field(block, "content")
            if isinstance(results, Sequence) and not isinstance(results, (str, bytes, bytearray)):
                for result in results:
                    add(_field(result, "url"))
        elif block_type == "text":
            citations = _field(block, "citations")
            if isinstance(citations, Sequence) and not isinstance(citations, (str, bytes, bytearray)):
                for citation in citations:
                    add(_field(citation, "url"))
    return urls


def extract_search_text(content: Any) -> str:
    """Return search prose plus every search-result URL the provider observed."""
    text = extract_text_content(content)
    urls = extract_search_result_urls(content)
    if not urls:
        return text
    listing = "\n".join(urls)
    return f"{text}\n\nSearch result URLs:\n{listing}" if text else f"Search result URLs:\n{listing}"
