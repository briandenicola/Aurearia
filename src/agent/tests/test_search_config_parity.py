import json
from pathlib import Path
from unittest.mock import AsyncMock

import pytest
from langchain_core.messages import AIMessage
from pydantic import ValidationError

from app.models.requests import MAX_PROMPT_LENGTH, CopilotExecuteRequest, LLMConfig
from app.teams import coin_copilot, coin_search


def request_payload():
    return json.loads(
        (Path(__file__).parent / "fixtures" / "coin_copilot" / "valid_execute_request.json").read_text()
    )


def test_copilot_search_prompt_is_optional_and_uses_the_legacy_bound():
    payload = request_payload()
    assert CopilotExecuteRequest.model_validate(payload).coin_search_prompt == ""
    payload["coin_search_prompt"] = "x" * MAX_PROMPT_LENGTH
    assert len(CopilotExecuteRequest.model_validate(payload).coin_search_prompt) == MAX_PROMPT_LENGTH
    payload["coin_search_prompt"] += "x"
    with pytest.raises(ValidationError):
        CopilotExecuteRequest.model_validate(payload)


@pytest.mark.asyncio
@pytest.mark.parametrize("prompt", ["", "Prefer fixed-price Roman denarii below $150."])
async def test_legacy_and_copilot_send_identical_search_configuration(monkeypatch, prompt):
    search_model = AsyncMock()
    search_model.ainvoke.return_value = AIMessage(content="No matching sources.")
    formatter = AsyncMock()
    formatter.ainvoke.return_value = AIMessage(content="No matching listings.")
    monkeypatch.setattr(coin_search, "get_search_model", lambda _: search_model)
    monkeypatch.setattr(coin_search, "get_chat_model", lambda _: formatter)
    config = LLMConfig(provider="anthropic", api_key="fixture", model="fixture")
    hosts = {"ma-shops.com", "vcoins.com"}
    query = "Find a denarius under $150"
    legacy = coin_search.create_coin_search_team(config, prompt, hosts)
    await legacy.ainvoke({"user_message": query, "messages": [], "search_results": "", "fetched_listings": ""})

    payload = request_payload()
    payload["coin_search_prompt"] = prompt
    payload["dealer_search_sources"] = sorted(hosts)
    payload["allowed_tools"] = ["market_search"]
    request = CopilotExecuteRequest.model_validate(payload)
    supervisor = AsyncMock()
    supervisor.ainvoke.side_effect = [
        AIMessage(content="", tool_calls=[
            {"name": "market_search", "args": {"query": query}, "id": "call_parity", "type": "tool_call"},
        ]),
        AIMessage(content="No matching listings."),
    ]
    frames = [frame async for frame in coin_copilot.run_coin_copilot(request, model=supervisor)]

    assert frames[-1].type == "completed"
    assert search_model.ainvoke.await_count == 2
    legacy_messages = search_model.ainvoke.call_args_list[0].args[0]
    copilot_messages = search_model.ainvoke.call_args_list[1].args[0]
    assert [message.content for message in copilot_messages] == [message.content for message in legacy_messages]
    assert "vcoins.com" in copilot_messages[0].content
    assert "ma-shops.com" in copilot_messages[0].content
    if prompt:
        assert copilot_messages[0].content.startswith(prompt)
        assert all(prompt not in str(frame.model_dump()) for frame in frames)
