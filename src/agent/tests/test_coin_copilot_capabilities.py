"""Provider tool-capability checks fail closed."""

from unittest.mock import Mock

import httpx
import pytest
from fastapi.testclient import TestClient

from app.llm.capabilities import CopilotCapabilityError, bind_coin_copilot_model
from app.main import app
from app.models.requests import COPILOT_ALLOWED_TOOLS, LLMConfig
from app.tools.copilot_collection_tools import build_copilot_tool_definitions

client = TestClient(app)
AUTH_HEADERS = {"X-Internal-Service-Token": "test-agent-service-token"}


class _Model:
    def __init__(self):
        self.calls = []

    def bind_tools(self, tools, **kwargs):
        self.calls.append((tools, kwargs))
        return "bound"


@pytest.mark.asyncio
async def test_anthropic_binds_fixed_tools_strictly(monkeypatch):
    model = _Model()
    monkeypatch.setattr("app.llm.capabilities.get_chat_model", lambda _config: model)
    tools = build_copilot_tool_definitions()

    bound = await bind_coin_copilot_model(
        LLMConfig(provider="anthropic", api_key="key", model="claude-sonnet-5"),
        tools,
    )

    assert bound == "bound"
    bound_tools, options = model.calls[0]
    assert options == {"tool_choice": "auto", "strict": True}
    assert {tool["name"] for tool in bound_tools} == set(COPILOT_ALLOWED_TOOLS)

    unsupported = {
        "exclusiveMaximum",
        "exclusiveMinimum",
        "format",
        "maxItems",
        "maxLength",
        "maxProperties",
        "maximum",
        "minItems",
        "minLength",
        "minProperties",
        "minimum",
        "multipleOf",
        "pattern",
        "uniqueItems",
    }

    def schema_keys(value):
        if isinstance(value, dict):
            return set(value).union(*(schema_keys(item) for item in value.values()))
        if isinstance(value, list):
            return set().union(*(schema_keys(item) for item in value))
        return set()

    assert all(not (schema_keys(tool["input_schema"]) & unsupported) for tool in bound_tools)
    get_coin = next(tool for tool in bound_tools if tool["name"] == "get_coin")
    assert get_coin["input_schema"]["properties"]["coin_id"]["type"] == "integer"
    assert get_coin["input_schema"]["required"] == ["coin_id"]


@pytest.mark.asyncio
async def test_anthropic_binding_failure_is_unsupported(monkeypatch):
    model = _Model()
    model.bind_tools = Mock(side_effect=RuntimeError("unsupported"))
    monkeypatch.setattr("app.llm.capabilities.get_chat_model", lambda _config: model)
    with pytest.raises(CopilotCapabilityError):
        await bind_coin_copilot_model(
            LLMConfig(provider="anthropic", api_key="key", model="claude-sonnet-5"),
            [],
        )


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "payload",
    [
        {"capabilities": ["completion"]},
        {"capabilities": "tools"},
        {},
    ],
)
async def test_ollama_requires_explicit_tools_capability(monkeypatch, payload):
    model = _Model()
    monkeypatch.setattr("app.llm.capabilities.get_chat_model", lambda _config: model)
    transport = httpx.MockTransport(lambda _request: httpx.Response(200, json=payload))
    async with httpx.AsyncClient(transport=transport) as client:
        with pytest.raises(CopilotCapabilityError):
            await bind_coin_copilot_model(
                LLMConfig(
                    provider="ollama",
                    model="qwen",
                    ollama_url="http://test-api:8080",
                ),
                [],
                client=client,
            )


@pytest.mark.asyncio
async def test_ollama_accepts_explicit_tools_capability(monkeypatch):
    model = _Model()
    monkeypatch.setattr("app.llm.capabilities.get_chat_model", lambda _config: model)
    transport = httpx.MockTransport(
        lambda _request: httpx.Response(200, json={"capabilities": ["completion", "tools"]})
    )
    async with httpx.AsyncClient(transport=transport) as client:
        bound = await bind_coin_copilot_model(
            LLMConfig(
                provider="ollama",
                model="qwen",
                ollama_url="http://test-api:8080",
            ),
            [],
            client=client,
        )
    assert bound == "bound"
    assert model.calls == [([], {"tool_choice": "auto"})]


@pytest.mark.asyncio
@pytest.mark.parametrize("failure", ["timeout", "malformed"])
async def test_ollama_capability_transport_failures_are_unsupported(monkeypatch, failure):
    model = _Model()
    monkeypatch.setattr("app.llm.capabilities.get_chat_model", lambda _config: model)

    def handler(request):
        if failure == "timeout":
            raise httpx.ReadTimeout("timed out", request=request)
        return httpx.Response(200, content=b"{")

    transport = httpx.MockTransport(handler)
    async with httpx.AsyncClient(transport=transport) as client:
        with pytest.raises(CopilotCapabilityError):
            await bind_coin_copilot_model(
                LLMConfig(
                    provider="ollama",
                    model="qwen",
                    ollama_url="http://test-api:8080",
                ),
                [],
                client=client,
            )


def test_capability_preflight_binds_all_fixed_authorized_tools(monkeypatch):
    observed = {}

    async def bind_model(config, tools):
        observed["provider"] = config.provider
        observed["tool_names"] = {tool.name for tool in tools}
        return object()

    monkeypatch.setattr("app.routes.bind_coin_copilot_model", bind_model)
    response = client.post(
        "/api/copilot/capability",
        headers=AUTH_HEADERS,
        json={"llm": {"provider": "anthropic", "api_key": "secret", "model": "claude"}},
    )

    assert response.status_code == 200
    assert response.json() == {"supported": True}
    assert observed == {
        "provider": "anthropic",
        "tool_names": set(COPILOT_ALLOWED_TOOLS),
    }


def test_capability_preflight_reports_binding_failure_without_error_details(monkeypatch):
    async def reject_binding(_config, _tools):
        raise CopilotCapabilityError("provider detail must not cross the boundary")

    monkeypatch.setattr("app.routes.bind_coin_copilot_model", reject_binding)
    response = client.post(
        "/api/copilot/capability",
        headers=AUTH_HEADERS,
        json={"llm": {"provider": "anthropic", "api_key": "secret", "model": "claude"}},
    )

    assert response.status_code == 200
    assert response.json() == {"supported": False}
    assert "provider detail" not in response.text
