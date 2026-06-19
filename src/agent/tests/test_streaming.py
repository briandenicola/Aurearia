"""Tests for SSE streaming utilities."""

import json

import pytest
from langchain_core.messages import AIMessage, AIMessageChunk

from app.streaming import extract_suggestions, remove_json_block, sanitize_user_facing_text, stream_graph_events

INTERNAL_JWT = (
    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9."
    "eyJzdWIiOiIxMjMiLCJ1c2VySWQiOjQ1Nn0."
    "c2lnbmF0dXJlLXBsYWNlaG9sZGVyLTEyMzQ1"
)


class FakeGraph:
    """Minimal async graph that yields prebuilt LangGraph-style events."""

    def __init__(self, events: list[dict]):
        self.events = events

    async def astream_events(self, input_data: dict, config: dict, version: str):
        for event in self.events:
            yield event


async def collect_sse(graph: FakeGraph) -> list[dict]:
    events = []
    async for line in stream_graph_events(graph, {"messages": []}):
        assert line.startswith("data: ")
        events.append(json.loads(line.removeprefix("data: ").strip()))
    return events


def combined_text(events: list[dict]) -> str:
    return "".join(event["text"] for event in events if event["type"] == "text")


def test_sanitize_user_facing_text_redacts_jwt_bearer_token():
    text = f"Before Bearer {INTERNAL_JWT} after"

    result = sanitize_user_facing_text(text)

    assert INTERNAL_JWT not in result
    assert result == "Before [REDACTED_INTERNAL_TOKEN] after"


def test_sanitize_user_facing_text_preserves_proposal_token_text():
    text = "Use proposal_id prop-123 and token token-abc with commit_update."

    result = sanitize_user_facing_text(text)

    assert result == text


def test_extract_suggestions_valid_json():
    text = '''Here are some coins:
```json
[{"name": "Augustus Denarius", "sourceUrl": "https://example.com/1"}]
```
Hope that helps!'''
    result = extract_suggestions(text)
    assert len(result) == 1
    assert result[0]["name"] == "Augustus Denarius"


def test_extract_suggestions_empty_array():
    text = '```json\n[]\n```'
    result = extract_suggestions(text)
    assert result == []


def test_extract_suggestions_no_json():
    text = "I could not find any coins matching your request."
    result = extract_suggestions(text)
    assert result == []


def test_extract_suggestions_invalid_json():
    text = "```json\n{invalid}\n```"
    result = extract_suggestions(text)
    assert result == []


def test_remove_json_block():
    text = '''Found coins:
```json
[{"name": "Test"}]
```
All verified!'''
    result = remove_json_block(text)
    assert "```json" not in result
    assert "Found coins:" in result
    assert "All verified!" in result


def test_remove_json_block_no_block():
    text = "No coins found."
    result = remove_json_block(text)
    assert result == text


@pytest.mark.asyncio
async def test_streamed_chunk_redacts_internal_token():
    graph = FakeGraph([
        {
            "event": "on_chat_model_stream",
            "tags": [],
            "data": {"chunk": AIMessageChunk(content=f"Visible text Bearer {INTERNAL_JWT} done")},
        }
    ])

    events = await collect_sse(graph)

    assert combined_text(events) == "Visible text [REDACTED_INTERNAL_TOKEN] done"
    assert INTERNAL_JWT not in json.dumps(events)


@pytest.mark.asyncio
async def test_streamed_split_chunks_never_expose_complete_internal_token():
    first_token_part, second_token_part = INTERNAL_JWT.split(".", maxsplit=1)
    graph = FakeGraph([
        {
            "event": "on_chat_model_stream",
            "tags": [],
            "data": {"chunk": AIMessageChunk(content=f"Visible text Bearer {first_token_part}")},
        },
        {
            "event": "on_chat_model_stream",
            "tags": [],
            "data": {"chunk": AIMessageChunk(content=f".{second_token_part} done")},
        },
    ])

    events = await collect_sse(graph)

    assert combined_text(events) == "Visible text [REDACTED_INTERNAL_TOKEN] done"
    assert INTERNAL_JWT not in json.dumps(events)
    for event in events:
        if event["type"] == "text":
            assert first_token_part not in event["text"]
            assert second_token_part not in event["text"]


@pytest.mark.asyncio
async def test_anthropic_text_block_redacts_internal_token():
    graph = FakeGraph([
        {
            "event": "on_chat_model_stream",
            "tags": [],
            "data": {
                "chunk": AIMessageChunk(
                    content=[
                        {"type": "text", "text": f"Anthropic Bearer {INTERNAL_JWT} block"},
                    ]
                )
            },
        }
    ])

    events = await collect_sse(graph)

    assert combined_text(events) == "Anthropic [REDACTED_INTERNAL_TOKEN] block"
    assert INTERNAL_JWT not in json.dumps(events)


@pytest.mark.asyncio
async def test_streaming_preserves_proposal_update_tokens():
    graph = FakeGraph([
        {
            "event": "on_chat_model_stream",
            "tags": [],
            "data": {
                "chunk": AIMessageChunk(
                    content="Use proposal_id prop-123 and token token-abc with commit_update."
                )
            },
        }
    ])

    events = await collect_sse(graph)

    assert combined_text(events) == "Use proposal_id prop-123 and token token-abc with commit_update."


@pytest.mark.asyncio
async def test_final_done_message_redacts_internal_token():
    graph = FakeGraph([
        {
            "event": "on_chain_end",
            "data": {"output": {"messages": [AIMessage(content=f"Final Bearer {INTERNAL_JWT} message")]}},
        }
    ])

    events = await collect_sse(graph)

    done_event = events[-1]
    assert done_event == {"type": "done", "message": "Final [REDACTED_INTERNAL_TOKEN] message"}
    assert INTERNAL_JWT not in json.dumps(events)


@pytest.mark.asyncio
async def test_final_done_suggestions_redact_internal_token_recursively():
    suggestions = [
        {
            "name": f"Safe coin {INTERNAL_JWT}",
            "details": {
                "auth": f"Bearer {INTERNAL_JWT}",
                "proposal_id": "prop-123",
                "token": "token-abc",
            },
            "actions": ["commit_update", f"Bearer {INTERNAL_JWT}"],
        }
    ]
    graph = FakeGraph([
        {
            "event": "on_chain_end",
            "data": {
                "output": {
                    "messages": [
                        AIMessage(content=f"Found coins:\n```json\n{json.dumps(suggestions)}\n```"),
                    ]
                }
            },
        }
    ])

    events = await collect_sse(graph)

    done_event = events[-1]
    assert done_event["type"] == "done"
    assert done_event["message"] == "Found coins:"
    assert INTERNAL_JWT not in json.dumps(done_event)
    assert done_event["suggestions"][0]["name"] == "Safe coin [REDACTED_INTERNAL_TOKEN]"
    assert done_event["suggestions"][0]["details"]["auth"] == "[REDACTED_INTERNAL_TOKEN]"
    assert done_event["suggestions"][0]["details"]["proposal_id"] == "prop-123"
    assert done_event["suggestions"][0]["details"]["token"] == "token-abc"
    assert done_event["suggestions"][0]["actions"] == ["commit_update", "[REDACTED_INTERNAL_TOKEN]"]
