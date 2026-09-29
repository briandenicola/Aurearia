"""Collection chat and Ollama search agents keep ReAct behaviour on langchain.agents.create_agent (#781)."""

import warnings

import pytest
from langchain_core.language_models.fake_chat_models import FakeMessagesListChatModel
from langchain_core.messages import AIMessage, HumanMessage, SystemMessage, ToolMessage
from langchain_core.tools import tool

from app.llm.provider import create_search_agent
from app.models.requests import AppContext, LLMConfig
from app.teams.collection_chat import COLLECTION_AGENT_PROMPT, create_collection_chat_team


class _ToolCallingFake(FakeMessagesListChatModel):
    """Scripted chat model that records every prompt it receives."""

    prompts: list = []
    bound_tools: list = []

    def bind_tools(self, tools, **kwargs):
        self.bound_tools.extend(getattr(t, "name", t) for t in tools)
        return self

    def _generate(self, messages, *args, **kwargs):
        self.prompts.append(list(messages))
        return super()._generate(messages, *args, **kwargs)


def _scripted(tool_name: str) -> _ToolCallingFake:
    return _ToolCallingFake(
        responses=[
            AIMessage(content="", tool_calls=[{"name": tool_name, "args": {"query": "moose"}, "id": "call-1"}]),
            AIMessage(content="final answer"),
        ],
        prompts=[],
        bound_tools=[],
    )


@tool
def search_my_collection(query: str) -> str:
    """Search the collection."""
    return f"found:{query}"


@tool
def searxng_search(query: str) -> str:
    """Search the web."""
    return f"web:{query}"


@pytest.fixture
def llm_config():
    return LLMConfig(provider="ollama", model="test-model", ollama_url="http://localhost:11434")


def _no_langgraph_deprecation(record):
    return [str(w.message) for w in record if "create_react_agent" in str(w.message)] == []


async def test_collection_chat_runs_tool_loop_with_system_prompt_and_context(monkeypatch, llm_config):
    model = _scripted("search_my_collection")
    monkeypatch.setattr("app.teams.collection_chat.get_chat_model", lambda _config: model)
    monkeypatch.setattr("app.teams.collection_chat.build_collection_tools", lambda _url, _token: [search_my_collection])

    with warnings.catch_warnings(record=True) as record:
        warnings.simplefilter("always")
        agent = create_collection_chat_team(
            llm_config, "http://test:8080", "token", app_context=AppContext(activeCoinId=7, route="/coins/7"),
        )
    assert _no_langgraph_deprecation(record)

    result = await agent.ainvoke({"messages": [HumanMessage("Do I have moose coins?")]})

    assert [type(m) for m in result["messages"]] == [HumanMessage, AIMessage, ToolMessage, AIMessage]
    assert result["messages"][2].content == "found:moose"
    assert result["messages"][-1].content == "final answer"
    assert set(model.bound_tools) == {"search_my_collection"}
    assert len(model.prompts) == 2
    for prompt in model.prompts:
        assert isinstance(prompt[0], SystemMessage)
        assert prompt[0].content.startswith(COLLECTION_AGENT_PROMPT)
        assert "active coin id: 7" in prompt[0].content
        assert "current route: /coins/7" in prompt[0].content
    assert not any(isinstance(m, SystemMessage) for m in result["messages"])


async def test_ollama_search_agent_runs_searxng_tool_loop_without_extra_prompt(monkeypatch, llm_config):
    model = _scripted("searxng_search")
    monkeypatch.setattr("app.llm.provider.get_chat_model", lambda _config: model)
    monkeypatch.setattr("app.tools.search.create_searxng_search", lambda _url: searxng_search)

    with warnings.catch_warnings(record=True) as record:
        warnings.simplefilter("always")
        agent = create_search_agent(llm_config)
    assert _no_langgraph_deprecation(record)

    caller_system = SystemMessage(content="dealer search instructions")
    result = await agent.ainvoke({"messages": [caller_system, HumanMessage("Find coins")]})

    assert result["messages"][-1].content == "final answer"
    assert result["messages"][-2].content == "web:moose"
    assert set(model.bound_tools) == {"searxng_search"}
    assert [(type(m), m.content) for m in model.prompts[0]] == [
        (SystemMessage, "dealer search instructions"), (HumanMessage, "Find coins"),
    ]
