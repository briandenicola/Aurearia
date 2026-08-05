"""Tests for LLM provider factory behavior."""

import sys
import types

from app.llm.provider import ANTHROPIC_CACHE_CONTROL, WEB_SEARCH_TOOL, get_chat_model, get_search_model
from app.models.requests import LLMConfig


class _FakeAnthropicModel:
    def __init__(self, **kwargs):
        self.kwargs = kwargs
        self.bound_tools = None

    def bind_tools(self, tools):
        self.bound_tools = tools
        return self


def test_get_chat_model_anthropic_enables_prompt_caching(monkeypatch):
    fake_module = types.SimpleNamespace(ChatAnthropic=_FakeAnthropicModel)
    monkeypatch.setitem(sys.modules, "langchain_anthropic", fake_module)

    model = get_chat_model(LLMConfig(provider="anthropic", api_key="k", model="claude-sonnet-5"))

    assert isinstance(model, _FakeAnthropicModel)
    assert model.kwargs["model"] == "claude-sonnet-5"
    assert model.kwargs["api_key"] == "k"
    assert model.kwargs["model_kwargs"]["cache_control"] == ANTHROPIC_CACHE_CONTROL


def test_get_search_model_binds_web_search_tool_for_anthropic(monkeypatch):
    fake_module = types.SimpleNamespace(ChatAnthropic=_FakeAnthropicModel)
    monkeypatch.setitem(sys.modules, "langchain_anthropic", fake_module)

    model = get_search_model(LLMConfig(provider="anthropic", api_key="k", model="claude-sonnet-5"))

    assert isinstance(model, _FakeAnthropicModel)
    assert model.bound_tools == [WEB_SEARCH_TOOL]
