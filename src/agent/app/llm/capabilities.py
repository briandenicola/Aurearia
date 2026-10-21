"""Fail-closed model capability checks for Coin Copilot."""

import logging
from copy import deepcopy
from typing import Any

import httpx
from langchain_anthropic.chat_models import convert_to_anthropic_tool

from app.llm.provider import get_chat_model
from app.models.requests import LLMConfig
from app.outbound import validate_outbound_url

logger = logging.getLogger(__name__)


class CopilotCapabilityError(RuntimeError):
    """The configured model cannot safely run the tool-calling harness."""


_ANTHROPIC_UNSUPPORTED_SCHEMA_KEYWORDS = frozenset(
    {
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
)


# Models that accept the dynamic-filtering web tool variants; older models
# keep the basic variants.
_DYNAMIC_WEB_TOOL_MODEL_PREFIXES = (
    "claude-opus-5",
    "claude-opus-4-8",
    "claude-opus-4-7",
    "claude-opus-4-6",
    "claude-sonnet-5",
    "claude-sonnet-4-6",
)
WEB_SEARCH_MAX_USES = 5
WEB_FETCH_MAX_USES = 6
WEB_FETCH_MAX_CONTENT_TOKENS = 20000


def anthropic_web_research_tools(model: str) -> list[dict[str, Any]]:
    """Anthropic server-side web search and fetch, executed by Anthropic.

    Web fetch only retrieves URLs already present in the conversation (the
    owner's message or earlier search/fetch results), so it cannot be steered
    to an address the model invents.
    """
    dynamic = model.startswith(_DYNAMIC_WEB_TOOL_MODEL_PREFIXES)
    return [
        {
            "type": "web_search_20260209" if dynamic else "web_search_20250305",
            "name": "web_search",
            "max_uses": WEB_SEARCH_MAX_USES,
        },
        {
            "type": "web_fetch_20260209" if dynamic else "web_fetch_20250910",
            "name": "web_fetch",
            "max_uses": WEB_FETCH_MAX_USES,
            "max_content_tokens": WEB_FETCH_MAX_CONTENT_TOKENS,
            "citations": {"enabled": True},
        },
    ]


def _anthropic_compatible_tools(tools: list) -> list[dict[str, Any]]:
    """Remove unsupported JSON Schema validators from Anthropic tool inputs."""

    def sanitize(value: Any) -> Any:
        if isinstance(value, list):
            return [sanitize(item) for item in value]
        if not isinstance(value, dict):
            return value
        return {
            key: sanitize(item)
            for key, item in value.items()
            if key not in _ANTHROPIC_UNSUPPORTED_SCHEMA_KEYWORDS
        }

    compatible = []
    for tool in tools:
        definition = deepcopy(convert_to_anthropic_tool(tool, strict=True))
        definition["input_schema"] = sanitize(definition["input_schema"])
        compatible.append(definition)
    return compatible


async def bind_coin_copilot_model(
    config: LLMConfig,
    tools: list,
    *,
    client: httpx.AsyncClient | None = None,
):
    """Verify tool support and bind the fixed Coin Copilot tool set."""
    if config.provider == "ollama":
        if not config.model or not config.ollama_url:
            raise CopilotCapabilityError("Ollama model configuration is incomplete")
        base_url = validate_outbound_url(config.ollama_url, "ollama_url")
        owns_client = client is None
        http_client = client or httpx.AsyncClient(
            timeout=httpx.Timeout(connect=3.0, read=3.0, write=3.0, pool=3.0)
        )
        try:
            response = await http_client.post(
                f"{base_url}/api/show",
                json={"name": config.model},
            )
            response.raise_for_status()
            payload = response.json()
        except (httpx.HTTPError, ValueError, TypeError) as exc:
            raise CopilotCapabilityError("Ollama tool capability could not be verified") from exc
        finally:
            if owns_client:
                await http_client.aclose()
        capabilities = payload.get("capabilities") if isinstance(payload, dict) else None
        if not isinstance(capabilities, list) or "tools" not in capabilities:
            raise CopilotCapabilityError("Ollama model does not explicitly advertise tools")
    elif config.provider == "anthropic":
        if not config.api_key or not config.model:
            raise CopilotCapabilityError("Anthropic model configuration is incomplete")
    else:
        raise CopilotCapabilityError("Unsupported Coin Copilot provider")

    try:
        model = get_chat_model(config)
        provider_tools = tools
        binding_options = {"tool_choice": "auto"}
        if config.provider == "anthropic":
            provider_tools = [
                *_anthropic_compatible_tools(tools),
                *anthropic_web_research_tools(config.model),
            ]
            binding_options["strict"] = True
        return model.bind_tools(provider_tools, **binding_options)
    except Exception as exc:
        logger.warning(
            "Coin Copilot model binding failed provider=%s model=%s",
            config.provider,
            config.model,
        )
        raise CopilotCapabilityError("Model tool binding failed") from exc
