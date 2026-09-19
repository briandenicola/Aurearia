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


def _sequence(value: Any) -> Sequence[Any]:
    if isinstance(value, Sequence) and not isinstance(value, (str, bytes, bytearray)):
        return value
    return []


def _search_results(content: Any) -> list[dict[str, str]]:
    """Collect url/title/snippet for each provider-observed search result, in order."""
    if isinstance(content, Mapping):
        blocks: Sequence[Any] = [content]
    else:
        blocks = _sequence(content)

    results: dict[str, dict[str, str]] = {}

    def add(url: Any, title: Any = None, snippet: Any = None) -> None:
        if not isinstance(url, str) or not url.startswith(("https://", "http://")):
            return
        entry = results.setdefault(url, {"url": url, "title": "", "snippet": ""})
        if isinstance(title, str) and title.strip() and not entry["title"]:
            entry["title"] = " ".join(title.split())[:300]
        if isinstance(snippet, str) and snippet.strip() and not entry["snippet"]:
            entry["snippet"] = " ".join(snippet.split())[:500]

    for block in blocks:
        block_type = _field(block, "type")
        if block_type == "web_search_tool_result":
            for result in _sequence(_field(block, "content")):
                add(_field(result, "url"), _field(result, "title"))
        elif block_type == "text":
            for citation in _sequence(_field(block, "citations")):
                add(_field(citation, "url"), _field(citation, "title"), _field(citation, "cited_text"))
    return list(results.values())


def extract_search_result_urls(content: Any) -> list[str]:
    """Return URLs from provider web-search result blocks and text citations.

    Anthropic's server-side web_search returns result URLs in
    ``web_search_tool_result`` blocks and ``citations`` on text blocks, not in
    the text itself, so ``extract_text_content`` alone loses them.
    """
    return [result["url"] for result in _search_results(content)]


def extract_search_text(content: Any) -> str:
    """Return search prose plus every search result the provider observed.

    Each result keeps its title and any cited snippet so listings can still be
    identified when the page itself cannot be fetched.
    """
    text = extract_text_content(content)
    results = _search_results(content)
    if not results:
        return text
    lines = []
    for result in results:
        line = f"- {result['title']}\n  URL: {result['url']}" if result["title"] else f"- URL: {result['url']}"
        if result["snippet"]:
            line += f"\n  Snippet: {result['snippet']}"
        lines.append(line)
    listing = "\n".join(lines)
    return f"{text}\n\nSearch results:\n{listing}" if text else f"Search results:\n{listing}"
