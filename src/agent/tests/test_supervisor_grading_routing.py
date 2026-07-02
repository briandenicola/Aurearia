import asyncio

from langchain_core.messages import AIMessage

from app.models.requests import LLMConfig
from app.supervisor import ROUTER_PROMPT, create_router


def test_grading_is_not_advertised_as_chat_route():
    assert '"grading"' not in ROUTER_PROMPT


def test_stale_grading_router_response_falls_back_to_general(monkeypatch):
    async def fake_ainvoke_with_retry(_model, _messages):
        return AIMessage(content="grading")

    monkeypatch.setattr("app.supervisor.get_chat_model", lambda _llm_config: object())
    monkeypatch.setattr("app.supervisor.ainvoke_with_retry", fake_ainvoke_with_retry)

    router = create_router(LLMConfig(provider="anthropic", api_key="k", model="claude-opus-4-8"))
    command = asyncio.run(router({"messages": []}))

    assert command.goto == "general"
