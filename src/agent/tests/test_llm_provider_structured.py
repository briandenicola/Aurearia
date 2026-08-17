"""Tests for `get_structured_model` (T019-T022, Phase 3 — NE-2).

Covers the provider-agnostic structured-output factory itself: correct
method selection per provider, statelessness, and the schema-conformant vs.
malformed-JSON shapes that `hypothesis.py`'s degrade ladder is built to
handle (the ladder's own behavior is tested end-to-end in
`test_deep_identification_hypothesis.py`).
"""

import asyncio
import sys
import types

from pydantic import BaseModel

from app.llm.provider import get_structured_model
from app.models.requests import LLMConfig


class DummySchema(BaseModel):
    value: str


class _FakeStructuredRunnable:
    """What `.with_structured_output(..., include_raw=True)` returns."""

    def __init__(self, conformant: bool):
        self.conformant = conformant

    async def ainvoke(self, messages, **kwargs):
        if self.conformant:
            return {
                "raw": type("Resp", (), {"content": '{"value": "ok"}'})(),
                "parsed": DummySchema(value="ok"),
                "parsing_error": None,
            }
        return {
            "raw": type("Resp", (), {"content": "not json at all"})(),
            "parsed": None,
            "parsing_error": ValueError("could not parse"),
        }


class _FakeAnthropicModel:
    def __init__(self, **kwargs):
        self.kwargs = kwargs
        self.structured_calls: list[dict] = []

    def with_structured_output(self, schema, **kwargs):
        self.structured_calls.append({"schema": schema, **kwargs})
        return _FakeStructuredRunnable(conformant=True)


class _FakeOllamaModel:
    def __init__(self, **kwargs):
        self.kwargs = kwargs
        self.structured_calls: list[dict] = []

    def with_structured_output(self, schema, **kwargs):
        self.structured_calls.append({"schema": schema, **kwargs})
        return _FakeStructuredRunnable(conformant=False)


def test_anthropic_uses_tool_based_function_calling_method(monkeypatch):
    fake_module = types.SimpleNamespace(ChatAnthropic=_FakeAnthropicModel)
    monkeypatch.setitem(sys.modules, "langchain_anthropic", fake_module)

    config = LLMConfig(provider="anthropic", api_key="k", model="claude-sonnet-5")
    runnable = get_structured_model(config, DummySchema)

    result = asyncio.run(runnable.ainvoke([]))
    assert result["parsed"] == DummySchema(value="ok")


def test_anthropic_structured_binding_uses_function_calling_and_include_raw(monkeypatch):
    fake_module = types.SimpleNamespace(ChatAnthropic=_FakeAnthropicModel)
    monkeypatch.setitem(sys.modules, "langchain_anthropic", fake_module)
    captured: dict = {}

    class _Capturing(_FakeAnthropicModel):
        def with_structured_output(self, schema, **kwargs):
            captured.update({"schema": schema, **kwargs})
            return _FakeStructuredRunnable(conformant=True)

    fake_module.ChatAnthropic = _Capturing

    get_structured_model(LLMConfig(provider="anthropic", api_key="k"), DummySchema)

    assert captured["schema"] is DummySchema
    assert captured["method"] == "function_calling"
    assert captured["include_raw"] is True


def test_ollama_structured_binding_uses_json_schema_method(monkeypatch):
    fake_module = types.SimpleNamespace(ChatOllama=_FakeOllamaModel)
    monkeypatch.setitem(sys.modules, "langchain_ollama", fake_module)
    captured: dict = {}

    class _Capturing(_FakeOllamaModel):
        def with_structured_output(self, schema, **kwargs):
            captured.update({"schema": schema, **kwargs})
            return _FakeStructuredRunnable(conformant=False)

    fake_module.ChatOllama = _Capturing

    get_structured_model(LLMConfig(provider="ollama", model="llama3.1"), DummySchema)

    assert captured["schema"] is DummySchema
    assert captured["method"] == "json_schema"
    assert captured["include_raw"] is True


def test_ollama_malformed_json_surfaces_as_parsing_error_not_an_exception():
    """Ollama varies sharply by model and MUST be assumed to sometimes
    return malformed JSON — `with_structured_output(..., include_raw=True)`
    must surface that as a `parsing_error` in the result dict rather than
    raising, so a caller's degrade ladder can inspect the raw content
    without a second LLM call.
    """
    runnable = _FakeStructuredRunnable(conformant=False)

    result = asyncio.run(runnable.ainvoke([]))

    assert result["parsed"] is None
    assert result["parsing_error"] is not None
    assert result["raw"].content == "not json at all"


def test_get_structured_model_is_stateless(monkeypatch):
    """No module-level cache of models, config, or credentials — every call
    builds a fresh model/binding from the config passed in, exactly like
    `get_chat_model`/`get_search_model` (spec FR-032)."""
    fake_module = types.SimpleNamespace(ChatAnthropic=_FakeAnthropicModel)
    monkeypatch.setitem(sys.modules, "langchain_anthropic", fake_module)

    config_a = LLMConfig(provider="anthropic", api_key="key-a", model="claude-sonnet-5")
    config_b = LLMConfig(provider="anthropic", api_key="key-b", model="claude-sonnet-5")

    get_structured_model(config_a, DummySchema)
    get_structured_model(config_b, DummySchema)

    import app.llm.provider as provider_module

    module_level_state = {
        name: value
        for name, value in vars(provider_module).items()
        if not name.startswith("__")
        and not callable(value)
        and not isinstance(value, type)
        and not isinstance(value, types.ModuleType)
    }
    # Only the two documented constants (WEB_SEARCH_TOOL, ANTHROPIC_CACHE_CONTROL)
    # and the module logger are expected module-level state — nothing that
    # could carry a credential or cached model instance across calls.
    assert set(module_level_state) <= {"WEB_SEARCH_TOOL", "ANTHROPIC_CACHE_CONTROL", "logger"}


def test_unknown_provider_raises_before_binding_schema():
    import pytest

    with pytest.raises(ValueError, match="Unknown LLM provider"):
        get_structured_model(LLMConfig(provider="bogus"), DummySchema)
