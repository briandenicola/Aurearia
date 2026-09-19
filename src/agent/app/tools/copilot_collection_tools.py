"""Least-privilege Coin Copilot adapters for Go-owned collection reads."""

import hashlib
import json
import logging
import re
from collections.abc import Awaitable, Callable
from typing import Annotated, Any, Literal

import httpx
from langchain_core.tools import StructuredTool
from pydantic import BaseModel, ConfigDict, Field, StringConstraints, ValidationError

from app.models.requests import (
    COPILOT_ALLOWED_TOOLS,
    DeepAnalysisHandoffArguments,
    DeepAnalysisHandoffRequest,
    validate_deep_analysis_handoff_request_envelope,
)
from app.models.responses import (
    DeepAnalysisHandoffResult,
    validate_deep_analysis_handoff_persisted_result_envelope,
)
from app.outbound import validate_outbound_url
from app.teams.specialist_contracts import SpecialistQuery, SpecialistResult

logger = logging.getLogger(__name__)

CALLBACK_TOOLS = frozenset(
    {
        "search_my_collection",
        "get_coin",
        "collection_summary",
        "top_coins_by_value",
        "deep_analysis_handoff",
    }
)
VIRTUAL_TOOLS = frozenset({"portfolio_review", "gap_analysis"})
SPECIALIST_TOOLS = frozenset(
    {
        "market_search",
        "auction_search",
        "price_trends",
        "similar_lots",
    }
)
LOCAL_TOOLS = VIRTUAL_TOOLS | SPECIALIST_TOOLS
_PROMPT_INJECTION_RE = re.compile(
    r"(?i)\b(?:ignore|disregard|override|forget)\b.{0,40}\b(?:instructions?|prompt|rules?)\b"
)
_TOKEN_RE = re.compile(
    r"(?i)(?:bearer\s+[A-Za-z0-9._~+/\-=]{12,}|"
    r"(?:api[_-]?key|token|secret|password)\s*[:=]\s*[\"']?[A-Za-z0-9._~+/\-=]{8,})"
)


class CopilotToolError(RuntimeError):
    """Typed, safe tool execution failure."""

    def __init__(self, code: str, message: str):
        super().__init__(message)
        self.code = code
        self.safe_message = message


class StrictToolModel(BaseModel):
    model_config = ConfigDict(extra="forbid")


class SearchMyCollectionArgs(StrictToolModel):
    query: Annotated[str, StringConstraints(min_length=1, max_length=4000)]
    limit: int = Field(default=5, ge=1, le=20)


class GetCoinArgs(StrictToolModel):
    coin_id: int = Field(ge=1)


class CollectionSummaryArgs(StrictToolModel):
    pass


class TopCoinsByValueArgs(StrictToolModel):
    limit: int = Field(default=3, ge=1, le=10)


class PortfolioReviewArgs(StrictToolModel):
    pass


class GapAnalysisArgs(StrictToolModel):
    pass


class CopilotCoin(StrictToolModel):
    id: int = Field(ge=1)
    name: Annotated[str, StringConstraints(min_length=1, max_length=300)]
    category: Annotated[str, StringConstraints(max_length=300)] = ""
    denomination: Annotated[str, StringConstraints(max_length=300)] = ""
    era: Annotated[str, StringConstraints(max_length=300)] = ""
    ruler: Annotated[str, StringConstraints(max_length=300)] = ""
    mint: Annotated[str, StringConstraints(max_length=300)] = ""
    material: Annotated[str, StringConstraints(max_length=300)] = ""
    grade: Annotated[str, StringConstraints(max_length=64)] = ""
    weightGrams: float | None = None
    diameterMm: float | None = None
    purchasePrice: float | None = None
    currentValue: float | None = None
    missingFields: list[Annotated[str, StringConstraints(max_length=100)]] = Field(
        default_factory=list,
        max_length=50,
    )


class SearchResult(StrictToolModel):
    coins: list[CopilotCoin] = Field(default_factory=list, max_length=20)


class GetCoinResult(StrictToolModel):
    coin: CopilotCoin


class CollectionSummary(StrictToolModel):
    totalCoins: int = Field(ge=0)
    totalWishlist: int = Field(ge=0)
    totalSold: int = Field(ge=0)
    totalCurrentUsd: float
    totalPurchaseUsd: float
    missingFields: dict[str, int] = Field(default_factory=dict, max_length=100)


class CollectionSummaryResult(StrictToolModel):
    summary: CollectionSummary


class TopCoinsResult(StrictToolModel):
    coins: list[CopilotCoin] = Field(default_factory=list, max_length=10)


class AnalysisResult(StrictToolModel):
    analysis: Annotated[str, StringConstraints(min_length=1, max_length=12000)]
    mode: Literal["collection_only"] = "collection_only"


ARG_MODELS: dict[str, type[BaseModel]] = {
    "search_my_collection": SearchMyCollectionArgs,
    "get_coin": GetCoinArgs,
    "collection_summary": CollectionSummaryArgs,
    "top_coins_by_value": TopCoinsByValueArgs,
    "portfolio_review": PortfolioReviewArgs,
    "gap_analysis": GapAnalysisArgs,
    "market_search": SpecialistQuery,
    "auction_search": SpecialistQuery,
    "price_trends": SpecialistQuery,
    "similar_lots": SpecialistQuery,
    "deep_analysis_handoff": DeepAnalysisHandoffArguments,
}
RESULT_MODELS: dict[str, type[BaseModel]] = {
    "search_my_collection": SearchResult,
    "get_coin": GetCoinResult,
    "collection_summary": CollectionSummaryResult,
    "top_coins_by_value": TopCoinsResult,
    "portfolio_review": AnalysisResult,
    "gap_analysis": AnalysisResult,
    "market_search": SpecialistResult,
    "auction_search": SpecialistResult,
    "price_trends": SpecialistResult,
    "similar_lots": SpecialistResult,
    "deep_analysis_handoff": DeepAnalysisHandoffResult,
}

LocalRunner = Callable[[dict[str, Any]], Awaitable[BaseModel | dict[str, Any] | str]]


def _neutralize_text(value: str) -> str:
    value = _TOKEN_RE.sub("[REDACTED]", value)
    return _PROMPT_INJECTION_RE.sub("[UNTRUSTED INSTRUCTION REMOVED]", value)


def sanitize_untrusted_tool_data(value: Any) -> Any:
    """Recursively redact secrets and neutralize instruction-shaped tool data."""
    if isinstance(value, str):
        return _neutralize_text(value)
    if isinstance(value, list):
        return [sanitize_untrusted_tool_data(item) for item in value]
    if isinstance(value, dict):
        sanitized: dict[str, Any] = {}
        for key, item in value.items():
            lowered = key.lower()
            if any(marker in lowered for marker in ("token", "secret", "password", "api_key", "apikey")):
                sanitized[key] = "[REDACTED]"
            else:
                sanitized[key] = sanitize_untrusted_tool_data(item)
        return sanitized
    return value


def bound_tool_result(value: dict[str, Any], max_bytes: int) -> tuple[dict[str, Any], int, bool, str]:
    """Return a canonical, sanitized result within the persisted byte boundary."""
    sanitized = sanitize_untrusted_tool_data(value)
    encoded = json.dumps(
        sanitized,
        ensure_ascii=False,
        separators=(",", ":"),
        sort_keys=True,
    ).encode()
    digest = hashlib.sha256(encoded).hexdigest()
    if len(encoded) <= max_bytes:
        return sanitized, len(encoded), False, digest
    bounded = {
        "truncated": True,
        "original_bytes": len(encoded),
        "digest": digest,
        "summary": "Tool result exceeded the persisted-result limit.",
    }
    if len(
        json.dumps(
            bounded,
            ensure_ascii=False,
            separators=(",", ":"),
            sort_keys=True,
        ).encode()
    ) > max_bytes:
        raise CopilotToolError("invalid_tool_call", "Tool result limit is too small.")
    return bounded, len(encoded), True, digest


class CopilotCollectionToolClient:
    """Execute only the run-token-authorized Coin Copilot capability set."""

    def __init__(
        self,
        *,
        tools_base_url: str,
        execution_token: str,
        allowed_tools: list[str],
        max_result_bytes: int,
        checkpoint_version: int = 0,
        local_runners: dict[str, LocalRunner] | None = None,
        analysis_runners: dict[str, LocalRunner] | None = None,
        completed_call_ids: set[str] | None = None,
        completed_results: dict[str, dict[str, Any]] | None = None,
        client: httpx.AsyncClient | None = None,
    ):
        allowed = frozenset(allowed_tools)
        if not allowed or not allowed.issubset(COPILOT_ALLOWED_TOOLS):
            raise ValueError("invalid Coin Copilot tool allowlist")
        self.base_url = validate_outbound_url(tools_base_url, "tools_base_url")
        self.execution_token = execution_token
        self.allowed_tools = allowed
        self.max_result_bytes = max_result_bytes
        if checkpoint_version < 0:
            raise ValueError("checkpoint_version must be non-negative")
        self.checkpoint_version = checkpoint_version
        self.local_runners = {**(analysis_runners or {}), **(local_runners or {})}
        if not set(self.local_runners).issubset(LOCAL_TOOLS):
            raise ValueError("local runners contain an unsupported capability")
        self._client = client
        self._completed_call_ids = set(completed_call_ids or ())
        self._results = self._validate_completed_results(completed_results or {})

    def _validate_completed_results(
        self,
        completed_results: dict[str, dict[str, Any]],
    ) -> dict[str, dict[str, Any]]:
        validated_results: dict[str, dict[str, Any]] = {}
        for tool_name, result in completed_results.items():
            if tool_name not in self.allowed_tools or tool_name not in RESULT_MODELS:
                raise ValueError("completed result is not in the execution tool allowlist")
            try:
                validated_model = RESULT_MODELS[tool_name].model_validate(result)
                if tool_name in SPECIALIST_TOOLS and validated_model.capability != tool_name:
                    raise ValueError("completed specialist result does not match its capability")
                validated = validated_model.model_dump(mode="json")
            except ValidationError as exc:
                raise ValueError("completed tool result is invalid") from exc
            bounded, _, truncated, _ = bound_tool_result(validated, self.max_result_bytes)
            if truncated:
                raise ValueError("completed tool result exceeds the execution result limit")
            validated_results[tool_name] = bounded
        return validated_results

    async def execute(
        self,
        tool_name: str,
        tool_call_id: str,
        raw_args: dict[str, Any],
    ) -> tuple[dict[str, Any], int, bool, str]:
        if tool_name not in self.allowed_tools or tool_name not in ARG_MODELS:
            raise CopilotToolError("invalid_tool_call", "The requested tool is not allowed.")
        if not tool_call_id or tool_call_id in self._completed_call_ids:
            raise CopilotToolError("invalid_tool_call", "The tool call id is invalid or duplicated.")
        try:
            args = ARG_MODELS[tool_name].model_validate(raw_args)
        except ValidationError as exc:
            raise CopilotToolError("invalid_tool_call", "The tool arguments are invalid.") from exc

        if tool_name in CALLBACK_TOOLS:
            callback_args = args.model_dump(exclude_none=True)
            if tool_name == "deep_analysis_handoff":
                callback_args["expected_checkpoint_version"] = self.checkpoint_version
                if args.operation != "status":
                    callback_args["handoff_idempotency_key"] = hashlib.sha256(
                        tool_call_id.encode("utf-8")
                    ).hexdigest()
            result = await self._execute_callback(tool_name, tool_call_id, callback_args)
        else:
            result = await self._execute_local(
                tool_name,
                tool_call_id,
                args.model_dump(exclude_none=True),
            )
        try:
            validated_model = RESULT_MODELS[tool_name].model_validate(result)
        except ValidationError as exc:
            raise CopilotToolError("invalid_tool_call", "The tool returned an invalid result.") from exc
        if tool_name in SPECIALIST_TOOLS and validated_model.capability != tool_name:
            raise CopilotToolError("invalid_tool_call", "The tool returned an invalid result.")
        validated = validated_model.model_dump(mode="json")

        bounded, original_bytes, truncated, digest = bound_tool_result(
            validated,
            self.max_result_bytes,
        )
        self._completed_call_ids.add(tool_call_id)
        self._results[tool_name] = bounded
        return bounded, original_bytes, truncated, digest

    async def _execute_callback(
        self,
        tool_name: str,
        tool_call_id: str,
        args: dict[str, Any],
    ) -> dict[str, Any]:
        client = self._client
        owns_client = client is None
        if client is None:
            client = httpx.AsyncClient(timeout=httpx.Timeout(connect=5.0, read=20.0, write=5.0, pool=5.0))
        body = {"tool_call_id": tool_call_id, **args}
        if tool_name == "deep_analysis_handoff":
            request = DeepAnalysisHandoffRequest.model_validate(body)
            encoded = json.dumps(
                request.model_dump(mode="json", exclude_none=True),
                ensure_ascii=False,
                allow_nan=False,
                separators=(",", ":"),
                sort_keys=True,
            ).encode("utf-8")
            validate_deep_analysis_handoff_request_envelope(encoded)
        try:
            response = await client.post(
                f"{self.base_url}/api/internal/copilot/tools/{tool_name}",
                json=body,
                headers={"Authorization": f"Bearer {self.execution_token}"},
            )
            response.raise_for_status()
            if tool_name == "deep_analysis_handoff":
                validate_deep_analysis_handoff_persisted_result_envelope(response.content)
            payload = response.json()
            if not isinstance(payload, dict):
                raise ValueError("tool response must be an object")
            return payload
        except httpx.TimeoutException as exc:
            raise CopilotToolError("agent_unavailable", "The collection tool timed out.") from exc
        except (httpx.HTTPError, ValueError) as exc:
            logger.warning("Coin Copilot callback failed tool=%s", tool_name)
            raise CopilotToolError("invalid_tool_call", "The collection tool failed.") from exc
        finally:
            if owns_client:
                await client.aclose()

    async def _execute_local(
        self,
        tool_name: str,
        tool_call_id: str,
        args: dict[str, Any],
    ) -> BaseModel | dict[str, Any]:
        if tool_name not in LOCAL_TOOLS:
            raise CopilotToolError("invalid_tool_call", "The requested tool is not locally executable.")
        runner = self.local_runners.get(tool_name)
        if runner is None:
            raise CopilotToolError("internal", f"{tool_name} is unavailable.")
        if tool_name in VIRTUAL_TOOLS:
            summary_result = self._results.get("collection_summary")
            if not summary_result or "summary" not in summary_result:
                summary_result = await self._execute_callback(
                    "collection_summary",
                    hashlib.sha256(
                        f"{tool_call_id}:collection_summary".encode("utf-8")
                    ).hexdigest(),
                    {},
                )
                try:
                    validated_summary = CollectionSummaryResult.model_validate(summary_result)
                except ValidationError as exc:
                    raise CopilotToolError(
                        "invalid_tool_call",
                        "The collection summary returned an invalid result.",
                    ) from exc
                summary_result = validated_summary.model_dump(mode="json")
                self._results["collection_summary"] = summary_result
            result = await runner(summary_result["summary"])
            if not isinstance(result, str):
                raise CopilotToolError("invalid_tool_call", f"{tool_name} returned an invalid result.")
            return {"analysis": result, "mode": "collection_only"}
        result = await runner(args)
        if isinstance(result, str):
            raise CopilotToolError("invalid_tool_call", f"{tool_name} returned an invalid result.")
        return result


def build_copilot_tool_definitions(
    allowed_tools: list[str] | None = None,
) -> list[StructuredTool]:
    """Build fixed schemas for model binding; the harness executes calls itself."""

    async def _unreachable(**_kwargs):
        raise RuntimeError("Coin Copilot tools are executed by the bounded harness")

    descriptions = {
        "search_my_collection": "Search only the owner's collection.",
        "get_coin": "Read one owner-scoped coin by id.",
        "collection_summary": "Read owner-scoped aggregate collection statistics.",
        "top_coins_by_value": "Read the owner's highest-valued coins.",
        "portfolio_review": "Analyze validated collection summary data only.",
        "gap_analysis": "Identify structural collection gaps without market or acquisition advice.",
        "market_search": "Search the administrator-configured dealer sources for current listings.",
        "auction_search": "Search the administrator-configured auction sources for relevant lots.",
        "price_trends": "Analyze source-backed completed-sale observations.",
        "similar_lots": "Find and rank source-backed similar auction lots.",
        "deep_analysis_handoff": "Request, read, or explicitly rerun the existing owner-scoped Deep Analysis workflow.",
    }
    selected = allowed_tools or list(COPILOT_ALLOWED_TOOLS)
    if not set(selected).issubset(COPILOT_ALLOWED_TOOLS):
        raise ValueError("invalid Coin Copilot tool allowlist")
    return [
        StructuredTool.from_function(
            coroutine=_unreachable,
            name=name,
            description=descriptions[name],
            args_schema=ARG_MODELS[name],
        )
        for name in selected
    ]
