"""SSE streaming utilities for LangGraph agent output.

Converts LangGraph graph events into Server-Sent Events that the Go API
proxies directly to the Vue frontend.
"""

import json
import logging
import re
from collections.abc import AsyncGenerator

from langchain_core.messages import AIMessage, AIMessageChunk

logger = logging.getLogger(__name__)

# Maps LangGraph node names to user-facing status messages.
_STATUS_MESSAGES: dict[str, str] = {
    "router": "Routing your request...",
    "coin_search": "Searching for coins...",
    "coin_shows": "Finding upcoming coin shows...",
    "portfolio": "Analyzing your portfolio...",
    "general": "Thinking...",
    "analysis": "Analyzing coin images...",
    # Sub-graph node names (may propagate through astream_events)
    "search": "Searching the web...",
    "fetch": "Fetching dealer pages...",
    "verify": "Verifying results...",
    "format": "Formatting results...",
    "summarize": "Summarizing portfolio...",
    "analyze": "Analyzing collection...",
}


async def stream_graph_events(graph, input_data: dict, config: dict | None = None) -> AsyncGenerator[str, None]:
    """Stream LangGraph events as SSE-formatted strings.

    Yields SSE data lines compatible with the Go API's existing SSE proxy format:
    - data: {"type": "text", "text": "..."} for incremental text
    - data: {"type": "status", "message": "..."} for progress indicators
    - data: {"type": "done", "message": "...", "suggestions": [...]} for completion
    - data: {"type": "error", "message": "..."} on failure
    """
    full_text = ""
    last_ai_content = ""
    last_node_message = ""
    has_streamed_text = False
    tool_use_reported = False

    try:
        async for event in graph.astream_events(input_data, config=config or {}, version="v2"):
            kind = event.get("event", "")

            if kind == "on_chain_start":
                name = event.get("name", "")
                status = _STATUS_MESSAGES.get(name)
                if status:
                    yield format_sse({"type": "status", "message": status})

            elif kind == "on_tool_start":
                tool_name = event.get("name", "")
                logger.debug("[stream] Tool started: %s", tool_name)
                if "search" in tool_name.lower():
                    yield format_sse({"type": "status", "message": "Searching the web..."})

            elif kind == "on_tool_end":
                tool_name = event.get("name", "")
                tool_output = event.get("data", {}).get("output", "")
                output_len = len(str(tool_output)) if tool_output else 0
                logger.debug("[stream] Tool ended: %s (output=%d chars)", tool_name, output_len)
                if "search" in tool_name.lower():
                    yield format_sse({"type": "status", "message": "Processing search results..."})

            elif kind == "on_chat_model_stream":
                # Skip streaming LLM output from the router node — it's an internal classification decision
                # (e.g., "portfolio", "coin_search") that should never appear in user-visible output.
                tags = event.get("tags", [])
                if "router" in tags:
                    continue

                chunk = event.get("data", {}).get("chunk")
                if isinstance(chunk, AIMessageChunk) and chunk.content:
                    # String content = normal text tokens
                    if isinstance(chunk.content, str):
                        if chunk.content:
                            if not has_streamed_text:
                                has_streamed_text = True
                            full_text += chunk.content
                            yield format_sse({"type": "text", "text": chunk.content})
                    elif isinstance(chunk.content, list):
                        # List content = Anthropic content blocks (tool use, search results, etc.)
                        for block in chunk.content:
                            if not isinstance(block, dict):
                                continue
                            block_type = block.get("type", "")

                            if block_type == "server_tool_use" and not tool_use_reported:
                                tool_name = block.get("name", "")
                                if "search" in tool_name:
                                    yield format_sse({"type": "status", "message": "Searching the web..."})
                                    tool_use_reported = True

                            elif block_type == "web_search_tool_result":
                                yield format_sse({"type": "status", "message": "Processing search results..."})
                                tool_use_reported = False

                            elif block_type == "text" and block.get("text"):
                                text = block["text"]
                                if not has_streamed_text:
                                    has_streamed_text = True
                                full_text += text
                                yield format_sse({"type": "text", "text": text})

            elif kind == "on_chat_model_end":
                # Skip router node's LLM output (internal routing decision)
                tags = event.get("tags", [])
                if "router" in tags:
                    continue

                output = event.get("data", {}).get("output")
                if isinstance(output, AIMessage) and output.content:
                    if isinstance(output.content, str):
                        content = output.content
                    elif isinstance(output.content, list):
                        # Extract text from Anthropic content blocks
                        content = "".join(
                            b.get("text", "") for b in output.content
                            if isinstance(b, dict) and b.get("type") == "text"
                        )
                    else:
                        content = ""
                    if content:
                        last_ai_content = content

            elif kind == "on_chain_end":
                # Capture AIMessages from node outputs (for paths that skip LLM calls)
                output = event.get("data", {}).get("output", {})
                if isinstance(output, dict):
                    for msg in output.get("messages", []):
                        if isinstance(msg, AIMessage) and msg.content:
                            if isinstance(msg.content, str):
                                content = msg.content
                            elif isinstance(msg.content, list):
                                content = "".join(
                                    b.get("text", "") for b in msg.content
                                    if isinstance(b, dict) and b.get("type") == "text"
                                )
                            else:
                                content = ""
                            if content:
                                last_node_message = content

    except Exception:
        logger.exception("Error during graph streaming")
        yield format_sse({"type": "error", "message": "An error occurred while processing your request."})
        return

    logger.debug(
        "Stream complete — full_text=%d chars, last_ai_content=%d chars, "
        "last_node_message=%d chars",
        len(full_text), len(last_ai_content), len(last_node_message),
    )

    # Build the authoritative final response.
    #
    # Team workflows (coin_search, coin_shows, portfolio) run as sub-graphs
    # via ainvoke(). Their internal LLM events (on_chat_model_stream/end) may
    # NOT propagate to the supervisor's astream_events. The team's final
    # AIMessage IS visible through on_chain_end events on the supervisor node,
    # captured as last_node_message.
    #
    # For direct handler nodes (general_handler), last_ai_content has the
    # correct LLM output AND last_node_message also captures it.
    #
    # Priority: last_node_message > last_ai_content > full_text
    final_text = last_node_message or last_ai_content or full_text
    suggestions = extract_suggestions(final_text)
    clean_message = remove_json_block(final_text) if suggestions else final_text

    # If removing the JSON block left nothing meaningful, provide a fallback
    if not clean_message.strip() and suggestions:
        clean_message = "Here are the results I found."

    done_event: dict = {"type": "done", "message": clean_message.strip()}
    if suggestions:
        done_event["suggestions"] = suggestions

    yield format_sse(done_event)


def format_sse(data: dict) -> str:
    """Format a dict as an SSE data line."""
    return f"data: {json.dumps(data)}\n\n"


def extract_suggestions(text: str) -> list[dict]:
    """Extract coin suggestions from a ```json block in the text."""
    json_str = _extract_json_block(text)
    if not json_str:
        return []
    try:
        data = json.loads(json_str)
        if isinstance(data, list):
            return data
    except json.JSONDecodeError:
        pass
    return []


def remove_json_block(text: str) -> str:
    """Remove the first ```json ... ``` block from text."""
    return re.sub(r"```json\s*\n.*?\n```", "", text, count=1, flags=re.DOTALL)


def _extract_json_block(text: str) -> str | None:
    """Extract the content of the first ```json ... ``` block."""
    start = text.find("```json")
    if start == -1:
        return None
    start += len("```json")
    end = text.find("```", start)
    if end == -1:
        return None
    return text[start:end].strip()
